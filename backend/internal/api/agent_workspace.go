package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"themis/internal/categorize"
	"themis/internal/classification"
	"themis/internal/intelligence"
)

type agentMission struct {
	ID             int64      `json:"id"`
	ConversationID *int64     `json:"conversation_id"`
	Title          string     `json:"title"`
	Objective      string     `json:"objective"`
	Status         string     `json:"status"`
	LastReview     *time.Time `json:"last_review_at"`
	Updated        time.Time  `json:"updated_at"`
	NewStatements  int        `json:"new_statements"`
}
type agentMemory struct {
	ID      int64  `json:"id"`
	Content string `json:"content"`
}
type agentActionInput struct {
	Kind           string          `json:"kind"`
	TransactionIDs []int64         `json:"transaction_ids,omitempty"`
	DomainID       int64           `json:"domain_id,omitempty"`
	Tag            string          `json:"tag,omitempty"`
	Pattern        string          `json:"pattern,omitempty"`
	Content        string          `json:"content,omitempty"`
	Month          string          `json:"month,omitempty"`
	Simulation     simulationInput `json:"simulation,omitempty"`
}
type agentAction struct {
	ID      int64            `json:"id"`
	Kind    string           `json:"kind"`
	Status  string           `json:"status"`
	Input   agentActionInput `json:"input"`
	Preview json.RawMessage  `json:"preview"`
	Result  json.RawMessage  `json:"result"`
	Created time.Time        `json:"created_at"`
}
type agentSnapshot struct {
	Revision int64    `json:"classification_revision"`
	Source   string   `json:"classification_source"`
	ID       int64    `json:"id"`
	Date     string   `json:"date"`
	Label    string   `json:"label"`
	Amount   string   `json:"amount"`
	Domain   *int64   `json:"domain_id"`
	Tags     []string `json:"tags"`
}
type agentStep struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration int64  `json:"duration_ms"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
}

func agentDecode(w http.ResponseWriter, r *http.Request, out any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		writeErr(w, 400, fmt.Errorf("demande assistant invalide"))
		return false
	}
	return true
}
func agentID(r *http.Request) (int64, error) {
	id, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if e != nil || id <= 0 {
		return 0, fmt.Errorf("identifiant invalide")
	}
	return id, nil
}
func agentFailure(w http.ResponseWriter, e error) {
	var postgres *pq.Error
	if errors.As(e, &postgres) {
		writeErr(w, 500, fmt.Errorf("données assistant indisponibles"))
		return
	}
	if errors.Is(e, sql.ErrNoRows) {
		writeErr(w, 404, fmt.Errorf("élément introuvable"))
		return
	}
	writeErr(w, 400, e)
}
func agentJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func agentHash(v any) string { h := sha256.Sum256(agentJSON(v)); return hex.EncodeToString(h[:]) }
func agentExcerpt(s string, n int) string {
	v := []rune(s)
	if len(v) > n {
		return string(v[:n])
	}
	return s
}

func (s *Server) agentMissions(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == "POST" {
		var in struct {
			Objective      string `json:"objective"`
			Title          string `json:"title"`
			ConversationID *int64 `json:"conversation_id"`
		}
		if !agentDecode(w, r, &in) {
			return
		}
		in.Objective = strings.TrimSpace(in.Objective)
		in.Title = strings.TrimSpace(in.Title)
		if in.Title == "" {
			in.Title = agentExcerpt(in.Objective, 100)
		}
		if len([]rune(in.Objective)) < 1 || len([]rune(in.Objective)) > 2000 || len([]rune(in.Title)) > 160 {
			writeErr(w, 400, fmt.Errorf("objectif requis, 2000 caractères maximum"))
			return
		}
		tx, err := s.DB.BeginTx(r.Context(), nil)
		if err != nil {
			writeErr(w, 500, fmt.Errorf("mission indisponible"))
			return
		}
		defer tx.Rollback()
		var locked int64
		if err = tx.QueryRowContext(r.Context(), `SELECT id FROM users WHERE id=$1 FOR UPDATE`, owner).Scan(&locked); err != nil {
			writeErr(w, 500, fmt.Errorf("mission indisponible"))
			return
		}
		var active int
		if err = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM agent_missions WHERE owner_id=$1 AND status<>'completed'`, owner).Scan(&active); err != nil || active >= 50 {
			writeErr(w, 409, fmt.Errorf("50 missions ouvertes maximum ; terminer une mission avant d’en créer une"))
			return
		}
		var conversation int64
		if in.ConversationID != nil {
			err = tx.QueryRowContext(r.Context(), `SELECT id FROM chat_conversations WHERE id=$1 AND owner_id=$2`, *in.ConversationID, owner).Scan(&conversation)
		} else {
			err = tx.QueryRowContext(r.Context(), `INSERT INTO chat_conversations(owner_id,title) VALUES($1,$2) RETURNING id`, owner, in.Title).Scan(&conversation)
		}
		if err != nil {
			agentFailure(w, err)
			return
		}
		var id int64
		err = tx.QueryRowContext(r.Context(), `INSERT INTO agent_missions(owner_id,conversation_id,title,objective) VALUES($1,$2,$3,$4) RETURNING id`, owner, conversation, in.Title, in.Objective).Scan(&id)
		if err == nil {
			err = tx.Commit()
		}
		if err != nil {
			writeErr(w, 500, fmt.Errorf("mission non enregistrée"))
			return
		}
		writeJSON(w, 201, map[string]any{"id": id, "conversation_id": conversation})
		return
	}
	rows, e := s.DB.QueryContext(r.Context(), `SELECT m.id,m.conversation_id,m.title,m.objective,m.status,m.last_review_at,m.updated_at,(SELECT count(*) FROM statements st WHERE st.owner_id=m.owner_id AND st.uploaded_at>COALESCE(m.last_review_at,m.created_at)) FROM agent_missions m WHERE owner_id=$1 ORDER BY (status='completed'),updated_at DESC,id DESC LIMIT 50`, owner)
	if e != nil {
		writeErr(w, 500, fmt.Errorf("missions indisponibles"))
		return
	}
	defer rows.Close()
	out := []agentMission{}
	for rows.Next() {
		var m agentMission
		if rows.Scan(&m.ID, &m.ConversationID, &m.Title, &m.Objective, &m.Status, &m.LastReview, &m.Updated, &m.NewStatements) != nil {
			writeErr(w, 500, fmt.Errorf("missions illisibles"))
			return
		}
		out = append(out, m)
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("missions indisponibles"))
		return
	}
	writeJSON(w, 200, map[string]any{"missions": out})
}
func (s *Server) agentMissionUpdate(w http.ResponseWriter, r *http.Request) {
	id, e := agentID(r)
	if e != nil {
		agentFailure(w, e)
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !agentDecode(w, r, &in) {
		return
	}
	if in.Status != "active" && in.Status != "paused" && in.Status != "completed" {
		writeErr(w, 400, fmt.Errorf("statut de mission invalide"))
		return
	}
	result, e := s.DB.ExecContext(r.Context(), `UPDATE agent_missions SET status=$1,updated_at=now() WHERE id=$2 AND owner_id=$3`, in.Status, id, s.ownerID(r))
	if e != nil {
		writeErr(w, 500, fmt.Errorf("mission non enregistrée"))
		return
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		agentFailure(w, sql.ErrNoRows)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) agentMemories(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == "DELETE" {
		id, e := agentID(r)
		if e != nil {
			agentFailure(w, e)
			return
		}
		res, e := s.DB.ExecContext(r.Context(), `DELETE FROM agent_memories WHERE id=$1 AND owner_id=$2`, id, owner)
		if e != nil {
			writeErr(w, 500, fmt.Errorf("mémoire non supprimée"))
			return
		}
		n, _ := res.RowsAffected()
		if n != 1 {
			agentFailure(w, sql.ErrNoRows)
			return
		}
		w.WriteHeader(204)
		return
	}
	items, e := s.loadAgentMemories(r.Context(), owner)
	if e != nil {
		writeErr(w, 500, fmt.Errorf("préférences indisponibles"))
		return
	}
	writeJSON(w, 200, map[string]any{"memories": items})
}
func (s *Server) loadAgentMemories(ctx context.Context, owner int64) ([]agentMemory, error) {
	items := []agentMemory{}
	rows, e := s.DB.QueryContext(ctx, `SELECT id,content FROM agent_memories WHERE owner_id=$1 ORDER BY id DESC LIMIT 50`, owner)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var x agentMemory
		if e = rows.Scan(&x.ID, &x.Content); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (s *Server) agentBriefing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	zone, err := agentLocation(r.URL.Query().Get("timezone"))
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	month := r.URL.Query().Get("month")
	if month != "" && !validCalendarMonth(month) {
		writeErr(w, 400, fmt.Errorf("mois du briefing invalide"))
		return
	}
	period, err := resolveAgentPeriod("", time.Now().In(zone), agentPeriod{}, month)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	owner := s.ownerID(r)
	observations, err := agentReadPeriodBriefing(r.Context(), s.DB, owner, period, zone.String())
	if err != nil {
		writeErr(w, 500, fmt.Errorf("observations de période indisponibles"))
		return
	}
	items, e := intelligence.Feed(r.Context(), s.DB, owner, 3)
	if e != nil {
		writeErr(w, 500, fmt.Errorf("initiatives indisponibles"))
		return
	}
	var latest sql.NullString
	var imported sql.NullTime
	e = s.DB.QueryRowContext(r.Context(), `SELECT (SELECT max(tx_date)::text FROM transactions WHERE owner_id=$1),(SELECT max(uploaded_at) FROM statements WHERE owner_id=$1)`, owner).Scan(&latest, &imported)
	if e != nil {
		writeErr(w, 500, fmt.Errorf("fraîcheur des sources indisponible"))
		return
	}
	writeJSON(w, 200, map[string]any{"discoveries": items, "last_transaction_date": agentNullableString(latest), "last_import_at": agentNullableTime(imported), "checked_at": time.Now().UTC(), "coverage": "unknown", "period_summary": observations})
}
func (s *Server) agentConversations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	owner := s.ownerID(r)
	if r.PathValue("id") == "" {
		rows, e := s.DB.QueryContext(r.Context(), `SELECT id,title,updated_at FROM chat_conversations WHERE owner_id=$1 ORDER BY updated_at DESC,id DESC LIMIT 30`, owner)
		if e != nil {
			writeErr(w, 500, fmt.Errorf("conversations indisponibles"))
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id int64
			var title string
			var updated time.Time
			if rows.Scan(&id, &title, &updated) != nil {
				writeErr(w, 500, fmt.Errorf("historique illisible"))
				return
			}
			out = append(out, map[string]any{"id": id, "title": title, "updated_at": updated})
		}
		if rows.Err() != nil {
			writeErr(w, 500, fmt.Errorf("historique indisponible"))
			return
		}
		writeJSON(w, 200, map[string]any{"conversations": out})
		return
	}
	id, e := agentID(r)
	if e != nil {
		agentFailure(w, e)
		return
	}
	var context json.RawMessage
	e = s.DB.QueryRowContext(r.Context(), `SELECT agent_context FROM chat_conversations WHERE id=$1 AND owner_id=$2`, id, owner).Scan(&context)
	if e != nil {
		agentFailure(w, e)
		return
	}
	rows, e := s.DB.QueryContext(r.Context(), `SELECT id,role,content,metadata FROM (SELECT id,role,content,metadata FROM chat_messages WHERE conversation_id=$1 AND role IN ('user','assistant') ORDER BY id DESC LIMIT 100) recent ORDER BY id`, id)
	if e != nil {
		writeErr(w, 500, fmt.Errorf("messages indisponibles"))
		return
	}
	defer rows.Close()
	messages := []map[string]any{}
	for rows.Next() {
		var mid int64
		var role, content string
		var meta map[string]any
		var raw []byte
		if rows.Scan(&mid, &role, &content, &raw) != nil || json.Unmarshal(raw, &meta) != nil {
			writeErr(w, 500, fmt.Errorf("messages illisibles"))
			return
		}
		meta["id"] = mid
		meta["role"] = role
		meta["content"] = content
		messages = append(messages, meta)
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("messages indisponibles"))
		return
	}
	writeJSON(w, 200, map[string]any{"conversation_id": id, "context": context, "messages": messages})
}

type agentReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func agentSnapshots(ctx context.Context, reader agentReader, owner int64, ids []int64, lock bool) ([]agentSnapshot, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return nil, fmt.Errorf("sélection de 1 à 100 opérations requise")
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return nil, fmt.Errorf("sélection invalide")
		}
		seen[id] = true
	}
	suffix := ""
	if lock {
		suffix = " FOR UPDATE OF t"
	}
	const q = `SELECT t.id,t.tx_date::text,t.label,t.amount::text,t.domain_id,t.classification_source,COALESCE((SELECT revision FROM classification_state cs WHERE cs.owner_id=t.owner_id AND cs.transaction_id=t.id),0),COALESCE((SELECT array_agg(g.name ORDER BY g.name) FROM transaction_tags tt JOIN tags g ON g.id=tt.tag_id AND g.owner_id=t.owner_id WHERE tt.transaction_id=t.id),'{}') FROM transactions t WHERE t.owner_id=$1 AND t.id=ANY($2) ORDER BY t.id`
	rows, e := reader.QueryContext(ctx, q+suffix, owner, pq.Array(ids))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []agentSnapshot{}
	for rows.Next() {
		var x agentSnapshot
		if e = rows.Scan(&x.ID, &x.Date, &x.Label, &x.Amount, &x.Domain, &x.Source, &x.Revision, pq.Array(&x.Tags)); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if len(items) != len(ids) {
		return nil, fmt.Errorf("sélection indisponible pour ce compte")
	}
	return items, nil
}

type agentDomainSnapshot struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Transfer       bool   `json:"is_transfer"`
	Parent         *int64 `json:"parent_id"`
	ParentTransfer bool   `json:"parent_is_transfer"`
}

func agentDomainInfo(ctx context.Context, reader agentReader, owner, id int64, lock bool) (agentDomainSnapshot, error) {
	var d agentDomainSnapshot
	suffix := ""
	if lock {
		suffix = " FOR SHARE OF d"
	}
	rows, e := reader.QueryContext(ctx, `SELECT d.id,d.name,d.is_transfer,d.parent_id,COALESCE(p.is_transfer,false) FROM domains d LEFT JOIN domains p ON p.id=d.parent_id AND p.owner_id=d.owner_id WHERE d.id=$1 AND d.owner_id=$2`+suffix, id, owner)
	if e != nil {
		return d, e
	}
	if !rows.Next() {
		rows.Close()
		return d, fmt.Errorf("catégorie indisponible")
	}
	e = rows.Scan(&d.ID, &d.Name, &d.Transfer, &d.Parent, &d.ParentTransfer)
	rows.Close()
	if e != nil {
		return d, e
	}
	if lock && d.Parent != nil {
		rows, e = reader.QueryContext(ctx, `SELECT is_transfer FROM domains WHERE id=$1 AND owner_id=$2 FOR SHARE`, *d.Parent, owner)
		if e != nil {
			return d, e
		}
		if rows.Next() {
			e = rows.Scan(&d.ParentTransfer)
		}
		rows.Close()
		if e != nil {
			return d, e
		}
	}
	return d, nil
}
func (s *Server) agentPreview(ctx context.Context, owner int64, in agentActionInput) (any, string, error) {
	switch in.Kind {
	case "categorize", "tag":
		snapshots, e := agentSnapshots(ctx, s.DB, owner, in.TransactionIDs, false)
		if e != nil {
			return nil, "", e
		}
		preview := map[string]any{"transactions": snapshots, "count": len(snapshots), "effect": "Seules les opérations listées seront modifiées après validation."}
		if in.Kind == "categorize" {
			domain, err := agentDomainInfo(ctx, s.DB, owner, in.DomainID, false)
			e = err
			if e != nil {
				return nil, "", fmt.Errorf("catégorie indisponible")
			}
			preview["category"] = domain.Name
			return preview, agentHash(struct {
				Rows   []agentSnapshot
				Domain agentDomainSnapshot
			}{snapshots, domain}), nil
		} else {
			tags, e := validateTransactionTags([]string{in.Tag})
			if e != nil {
				return nil, "", e
			}
			preview["tag"] = tags[0]
			for _, x := range snapshots {
				if _, e = validateTransactionTags(append(x.Tags, tags[0])); e != nil {
					return nil, "", e
				}
			}
		}
		return preview, agentHash(snapshots), nil
	case "rule":
		if len(in.Pattern) > 200 || strings.TrimSpace(in.Pattern) == "" {
			return nil, "", fmt.Errorf("motif requis, 200 caractères maximum")
		}
		if e := s.validateRule(ctx, owner, in.DomainID, in.Pattern); e != nil {
			return nil, "", e
		}
		match, e := categorize.CompilePattern(in.Pattern)
		if e != nil {
			return nil, "", e
		}
		var exists bool
		e = s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM rules WHERE owner_id=$1 AND lower(pattern)=lower($2))`, owner, in.Pattern).Scan(&exists)
		if e != nil {
			return nil, "", e
		}
		if exists {
			return nil, "", fmt.Errorf("ce motif existe déjà ; le modifier dans les règles pour éviter un remplacement implicite")
		}
		rows, e := s.DB.QueryContext(ctx, `SELECT id,label FROM transactions WHERE owner_id=$1 ORDER BY id LIMIT 10001`, owner)
		if e != nil {
			return nil, "", e
		}
		defer rows.Close()
		samples := []map[string]any{}
		count, scanned := 0, 0
		for rows.Next() {
			var id int64
			var label string
			if e = rows.Scan(&id, &label); e != nil {
				return nil, "", e
			}
			scanned++
			if match(label) {
				count++
				if len(samples) < 10 {
					samples = append(samples, map[string]any{"id": id, "label": label})
				}
			}
		}
		if rows.Err() != nil {
			return nil, "", rows.Err()
		}
		if scanned > 10000 {
			return nil, "", fmt.Errorf("aperçu trop large ; utiliser un motif et une sélection plus ciblés")
		}
		domain, err := agentDomainInfo(ctx, s.DB, owner, in.DomainID, false)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"count": count, "category": domain.Name, "samples": samples, "pattern": in.Pattern, "effect": "Créer une règle pour les prochains imports. Aucune opération existante ne sera modifiée."}, agentHash(domain), nil
	case "memory":
		if strings.TrimSpace(in.Content) == "" || len([]rune(in.Content)) > 1000 {
			return nil, "", fmt.Errorf("préférence requise, 1000 caractères maximum")
		}
		return map[string]any{"content": in.Content, "effect": "Cette préférence déclarée sera consultable et supprimable ; elle n’établit aucun montant financier."}, "", nil
	case "simulation":
		if !validCalendarMonth(in.Month) {
			return nil, "", fmt.Errorf("mois de référence requis")
		}
		report, e := s.loadAnalysisReport(ctx, owner, in.Month, time.Now())
		if e != nil {
			return nil, "", fmt.Errorf("référence de simulation indisponible")
		}
		if !report.Available {
			return nil, "", fmt.Errorf("aucune opération dans cette référence ; choisir un mois documenté")
		}
		base := []simulationBase{}
		for _, c := range report.Categories {
			v, e := parseAnalysisAmount(string(c.Spent))
			if e != nil {
				return nil, "", e
			}
			base = append(base, simulationBase{Name: c.Name, Amount: v})
		}
		income, e := parseAnalysisAmount(string(report.Income))
		if e != nil {
			return nil, "", e
		}
		result, e := calculateSimulation(in.Simulation, income, base)
		if e != nil {
			return nil, "", e
		}
		return map[string]any{"scenario": result, "month": in.Month, "coverage": report.Coverage, "effect": "Scénario hypothétique sur les opérations observées ; aucune écriture bancaire."}, agentHash(report), nil
	}
	return nil, "", fmt.Errorf("action non disponible")
}
func (s *Server) prepareAgentAction(ctx context.Context, owner, conversation, mission int64, in agentActionInput) (agentAction, error) {
	var action agentAction
	if conversation <= 0 {
		return action, fmt.Errorf("conversation requise")
	}
	preview, hash, e := s.agentPreview(ctx, owner, in)
	if e != nil {
		return action, e
	}
	payload := agentJSON(in)
	data := agentJSON(preview)
	if len(payload) > 32768 || len(data) > 131072 {
		return action, fmt.Errorf("aperçu trop volumineux")
	}
	e = s.DB.QueryRowContext(ctx, `INSERT INTO agent_actions(owner_id,conversation_id,mission_id,kind,payload,preview,fingerprint) SELECT $1,$2,$3::bigint,$4,$5,$6,$7 WHERE EXISTS(SELECT 1 FROM chat_conversations WHERE id=$2 AND owner_id=$1) AND ($3::bigint IS NULL OR EXISTS(SELECT 1 FROM agent_missions WHERE id=$3 AND owner_id=$1 AND conversation_id=$2)) RETURNING id,created_at`, owner, conversation, optionalKnowledgeID(mission), in.Kind, string(payload), string(data), hash).Scan(&action.ID, &action.Created)
	if e != nil {
		return action, e
	}
	action.Kind = in.Kind
	action.Status = "prepared"
	action.Input = in
	action.Preview = data
	return action, nil
}
func (s *Server) agentActions(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == "POST" {
		var in struct {
			ConversationID int64            `json:"conversation_id"`
			MissionID      int64            `json:"mission_id"`
			Action         agentActionInput `json:"action"`
		}
		if !agentDecode(w, r, &in) {
			return
		}
		action, e := s.prepareAgentAction(r.Context(), owner, in.ConversationID, in.MissionID, in.Action)
		if e != nil {
			agentFailure(w, e)
			return
		}
		writeJSON(w, 201, action)
		return
	}
	conversation, e := strconv.ParseInt(r.URL.Query().Get("conversation_id"), 10, 64)
	if e != nil || conversation <= 0 {
		writeErr(w, 400, fmt.Errorf("conversation requise"))
		return
	}
	rows, e := s.DB.QueryContext(r.Context(), `SELECT id,kind,status,payload,preview,COALESCE(result,'null'::jsonb),created_at FROM agent_actions WHERE owner_id=$1 AND conversation_id=$2 ORDER BY id DESC LIMIT 30`, owner, conversation)
	if e != nil {
		writeErr(w, 500, fmt.Errorf("actions indisponibles"))
		return
	}
	defer rows.Close()
	out := []agentAction{}
	for rows.Next() {
		var a agentAction
		var in []byte
		if rows.Scan(&a.ID, &a.Kind, &a.Status, &in, &a.Preview, &a.Result, &a.Created) != nil || json.Unmarshal(in, &a.Input) != nil {
			writeErr(w, 500, fmt.Errorf("actions illisibles"))
			return
		}
		out = append(out, a)
	}
	if rows.Err() != nil {
		writeErr(w, 500, fmt.Errorf("actions indisponibles"))
		return
	}
	writeJSON(w, 200, map[string]any{"actions": out})
}
func (s *Server) agentActionDecision(w http.ResponseWriter, r *http.Request) {
	id, e := agentID(r)
	if e != nil {
		agentFailure(w, e)
		return
	}
	var in struct {
		Decision string `json:"decision"`
	}
	if !agentDecode(w, r, &in) {
		return
	}
	if in.Decision != "apply" && in.Decision != "dismiss" {
		writeErr(w, 400, fmt.Errorf("décision invalide"))
		return
	}
	owner := s.ownerID(r)
	tx, e := s.DB.BeginTx(r.Context(), nil)
	if e != nil {
		writeErr(w, 500, fmt.Errorf("action indisponible"))
		return
	}
	defer tx.Rollback()
	var kind, status, hash string
	var payload, oldPreview []byte
	var result json.RawMessage
	e = tx.QueryRowContext(r.Context(), `SELECT kind,status,payload,preview,fingerprint,COALESCE(result,'null'::jsonb) FROM agent_actions WHERE id=$1 AND owner_id=$2 FOR UPDATE`, id, owner).Scan(&kind, &status, &payload, &oldPreview, &hash, &result)
	if e != nil {
		agentFailure(w, e)
		return
	}
	if status != "prepared" {
		writeJSON(w, 200, map[string]any{"id": id, "status": status, "result": result})
		return
	}
	var action agentActionInput
	if json.Unmarshal(payload, &action) != nil {
		writeErr(w, 500, fmt.Errorf("action illisible"))
		return
	}
	if in.Decision == "dismiss" {
		_, e = tx.ExecContext(r.Context(), `UPDATE agent_actions SET status='dismissed',updated_at=now() WHERE id=$1 AND owner_id=$2`, id, owner)
		if e == nil {
			e = tx.Commit()
		}
		if e != nil {
			writeErr(w, 500, fmt.Errorf("décision non enregistrée"))
			return
		}
		writeJSON(w, 200, map[string]any{"id": id, "status": "dismissed"})
		return
	}
	staleAction := func(message string) {
		_, err := tx.ExecContext(r.Context(), `UPDATE agent_actions SET status='stale',updated_at=now() WHERE id=$1 AND owner_id=$2`, id, owner)
		if err == nil {
			err = tx.Commit()
		}
		if err != nil {
			writeErr(w, 500, fmt.Errorf("vérification non enregistrée"))
			return
		}
		writeErr(w, 409, fmt.Errorf("%s", message))
	}
	output := map[string]any{}
	if kind == "categorize" || kind == "tag" {
		snapshots, err := agentSnapshots(r.Context(), tx, owner, action.TransactionIDs, true)
		currentHash := agentHash(snapshots)
		if err == nil && kind == "categorize" {
			domain, domainErr := agentDomainInfo(r.Context(), tx, owner, action.DomainID, true)
			if domainErr != nil {
				err = domainErr
			} else {
				currentHash = agentHash(struct {
					Rows   []agentSnapshot
					Domain agentDomainSnapshot
				}{snapshots, domain})
			}
		}
		if err != nil || currentHash != hash {
			staleAction("les opérations ont changé ; préparer un nouvel aperçu")
			return
		}

		if kind == "categorize" {
			var valid bool
			e = tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM domains WHERE id=$1 AND owner_id=$2)`, action.DomainID, owner).Scan(&valid)
			if e != nil || !valid {
				writeErr(w, 409, fmt.Errorf("catégorie indisponible ; préparer un nouvel aperçu"))
				return
			}
		}
		for _, x := range snapshots {
			if _, e = classification.Begin(r.Context(), tx, owner, x.ID, "agent", fmt.Sprintf("agent:%d:%d", id, x.ID), action, &x.Revision, ""); e != nil {
				classificationFailure(w, e)
				return
			}

			if kind == "categorize" {
				_, e = tx.ExecContext(r.Context(), `UPDATE transactions SET domain_id=$1,classification_source='manual' WHERE id=$2 AND owner_id=$3`, action.DomainID, x.ID, owner)
				if e == nil {
					_, e = tx.ExecContext(r.Context(), `DELETE FROM transaction_suggestions WHERE transaction_id=$1 AND owner_id=$2`, x.ID, owner)
				}
			} else {
				tags, err := validateTransactionTags(append(x.Tags, action.Tag))
				if err != nil {
					writeErr(w, 400, err)
					return
				}
				e = replaceTransactionTags(r.Context(), tx, x.ID, owner, tags)
			}
			if e == nil {
				e = classification.Finish(r.Context(), tx, owner, x.ID)
			}
			if e != nil {
				writeErr(w, 500, fmt.Errorf("action non appliquée, aucune modification conservée"))
				return
			}
		}
		output["count"] = len(snapshots)
	} else if kind == "rule" {
		domain, err := agentDomainInfo(r.Context(), tx, owner, action.DomainID, true)
		if err != nil || agentHash(domain) != hash {
			staleAction("la catégorie a changé ; préparer un nouvel aperçu")
			return
		}
		var rule int64
		e = tx.QueryRowContext(r.Context(), `INSERT INTO rules(owner_id,pattern,domain_id,tags) SELECT $1,$2,$3,'' WHERE EXISTS(SELECT 1 FROM domains WHERE id=$3 AND owner_id=$1) ON CONFLICT(owner_id,lower(pattern)) DO NOTHING RETURNING id`, owner, action.Pattern, action.DomainID).Scan(&rule)
		if e != nil {
			writeErr(w, 409, fmt.Errorf("règle déjà présente ou catégorie indisponible ; préparer un nouvel aperçu"))
			return
		}
		output["rule_id"] = rule
	} else if kind == "memory" {
		var lockedOwner int64
		if e = tx.QueryRowContext(r.Context(), `SELECT id FROM users WHERE id=$1 FOR UPDATE`, owner).Scan(&lockedOwner); e != nil {
			writeErr(w, 500, fmt.Errorf("préférence non enregistrée"))
			return
		}
		var count int
		if e = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM agent_memories WHERE owner_id=$1`, owner).Scan(&count); e != nil || count >= 50 {
			writeErr(w, 409, fmt.Errorf("50 préférences maximum ; retirer une ancienne préférence"))
			return
		}
		var memory int64
		e = tx.QueryRowContext(r.Context(), `INSERT INTO agent_memories(owner_id,content) VALUES($1,$2) RETURNING id`, owner, action.Content).Scan(&memory)
		if e != nil {
			writeErr(w, 500, fmt.Errorf("préférence non enregistrée"))
			return
		}
		output["memory_id"] = memory
	} else if kind == "simulation" {
		preview, newHash, err := s.agentPreview(r.Context(), owner, action)
		if err != nil || newHash != hash {
			staleAction("la référence de simulation a changé ; préparer un nouvel aperçu")
			return
		}
		output["simulation"] = preview
	} else {
		writeErr(w, 400, fmt.Errorf("action indisponible"))
		return
	}
	_, e = tx.ExecContext(r.Context(), `UPDATE agent_actions SET status='applied',result=$1,updated_at=now() WHERE id=$2 AND owner_id=$3`, string(agentJSON(output)), id, owner)
	if e == nil {
		e = tx.Commit()
	}
	if e != nil {
		writeErr(w, 500, fmt.Errorf("action non appliquée, aucune modification conservée"))
		return
	}
	if kind == "categorize" || kind == "tag" {
		s.ScheduleIntelligenceRecompute(owner, "agent_action")
	}
	writeJSON(w, 200, map[string]any{"id": id, "status": "applied", "result": output})
}
