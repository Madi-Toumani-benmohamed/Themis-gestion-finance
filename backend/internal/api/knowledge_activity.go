package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"themis/internal/memory"
)

type knowledgeActivity struct {
	ID             int64           `json:"id"`
	Kind           string          `json:"kind"`
	Source         string          `json:"source"`
	Status         string          `json:"status"`
	StatementID    *int64          `json:"statement_id"`
	ConversationID *int64          `json:"conversation_id"`
	StatementName  *string         `json:"statement_name"`
	FactCount      *int            `json:"fact_count"`
	ProcessedCount *int            `json:"processed_count"`
	Entities       *int            `json:"entities_processed"`
	Edges          *int            `json:"edges_processed"`
	ResultCount    *int            `json:"result_count"`
	Included       bool            `json:"context_included"`
	Facts          []memory.Result `json:"facts"`
	Code           string          `json:"error_code"`
	Duration       *int64          `json:"duration_ms"`
	Started        time.Time       `json:"started_at"`
	Finished       *time.Time      `json:"finished_at"`
	Stale          bool            `json:"possibly_interrupted"`
}
type knowledgeUpdate struct {
	Status                                             string
	FactCount, Processed, Entities, Edges, ResultCount *int
	Included                                           bool
	Facts                                              []memory.Result
	Code                                               string
}

func optionalKnowledgeID(id int64) any {
	if id > 0 {
		return id
	}
	return nil
}
func (s *Server) beginKnowledge(owner int64, kind, source string, statement, conversation int64) (int64, error) {
	if s.DB == nil {
		return 0, errors.New("journal mémoire indisponible")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var id int64
	err := s.DB.QueryRowContext(ctx, `INSERT INTO knowledge_activity(owner_id,kind,source,statement_id,conversation_id,status)
 SELECT $1::bigint,$2::text,$3::text,$4::bigint,$5::bigint,'running' WHERE $1::bigint>0
 AND ($4::bigint IS NULL OR EXISTS(SELECT 1 FROM statements WHERE id=$4 AND owner_id=$1))
 AND ($5::bigint IS NULL OR EXISTS(SELECT 1 FROM chat_conversations WHERE id=$5 AND owner_id=$1)) RETURNING id`, owner, kind, source, optionalKnowledgeID(statement), optionalKnowledgeID(conversation)).Scan(&id)
	return id, err
}
func (s *Server) finishKnowledge(owner, id int64, start time.Time, u knowledgeUpdate) bool {
	if id <= 0 || s.DB == nil {
		return false
	}
	for i := range u.Facts {
		text := []rune(u.Facts[i].Fact)
		if len(text) > 2000 {
			u.Facts[i].Fact = string(text[:2000])
			u.Facts[i].Truncated = true
		}
	}
	if u.Facts == nil {
		u.Facts = []memory.Result{}
	}
	recorded := true
	data, e := json.Marshal(u.Facts)
	if e != nil || len(data) > 64000 {
		recorded = false
		u.Status = "error"
		u.Code = "protocol"
		data = []byte("[]")
		u.Included = false
		u.ResultCount = nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, e := s.DB.ExecContext(ctx, `UPDATE knowledge_activity SET status=$3,fact_count=$4,processed_count=$5,entities_processed=$6,edges_processed=$7,result_count=$8,context_included=$9,facts=$10,error_code=$11,duration_ms=$12,finished_at=now() WHERE id=$1 AND owner_id=$2`, id, owner, u.Status, u.FactCount, u.Processed, u.Entities, u.Edges, u.ResultCount, u.Included, string(data), u.Code, time.Since(start).Milliseconds())
	if e != nil {
		log.Print("journal mémoire indisponible")
		return false
	}
	n, e := result.RowsAffected()
	return recorded && e == nil && n == 1
}
func knowledgeFailure(err error, learning bool) string {
	switch memory.FailureCode(err) {
	case "disabled":
		return "disabled"
	case "not_configured":
		return "not_configured"
	case "cancelled":
		if learning {
			return "uncertain"
		}
		return "cancelled"
	case "timeout":
		if learning {
			return "uncertain"
		}
	}
	return "error"
}
func (s *Server) learnKnowledge(ctx context.Context, owner, statement int64, source string, facts []string) (knowledgeUpdate, error) {
	start := time.Now()
	id, e := s.beginKnowledge(owner, "learn", source, statement, 0)
	if errors.Is(e, sql.ErrNoRows) {
		return knowledgeUpdate{Status: "error", Code: "scope"}, &memory.Failure{Code: "scope"}
	}
	if e != nil {
		log.Print("journal mémoire indisponible")
	}
	total := len(facts)
	u := knowledgeUpdate{Status: "success", FactCount: &total}
	done, entities, edges := 0, 0, 0
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	client := s.getMemory()
	var failure error
	if client == nil {
		failure = &memory.Failure{Code: "disabled"}
	} else {
		sourceRef := "manual"
		if statement > 0 {
			sourceRef = fmt.Sprintf("statement:%d", statement)
		}
		for from := 0; from < len(facts); from += 100 {
			to := from + 100
			if to > len(facts) {
				to = len(facts)
			}
			n, m, err := client.LearnSource(ctx, memory.Group(owner), "transaction_import", sourceRef, facts[from:to])
			if err != nil {
				failure = err
				break
			}
			done += to - from
			entities += n
			edges += m
		}
	}
	if done > 0 || failure == nil {
		u.Processed = &done
		u.Entities = &entities
		u.Edges = &edges
	}
	if failure != nil {
		u.Status = knowledgeFailure(failure, true)
		u.Code = memory.FailureCode(failure)
		if done > 0 && u.Status == "error" {
			u.Status = "partial"
		}
	}
	s.finishKnowledge(owner, id, start, u)
	return u, failure
}
func (s *Server) skipKnowledgeImport(owner, statement int64) {
	start := time.Now()
	id, e := s.beginKnowledge(owner, "learn", "statement", statement, 0)
	if e != nil {
		log.Print("journal mémoire indisponible")
		return
	}
	s.finishKnowledge(owner, id, start, knowledgeUpdate{Status: "skipped", Code: "statement_unvalidated"})
}

type knowledgeTrace struct {
	Status         string          `json:"status"`
	ActivityID     int64           `json:"activity_id"`
	AuditAvailable bool            `json:"audit_available"`
	Facts          []memory.Result `json:"facts"`
	Included       bool            `json:"context_included"`
	Code           string          `json:"error_code,omitempty"`
	CheckedAt      time.Time       `json:"checked_at"`
	Duration       int64           `json:"duration_ms"`
}

func (s *Server) retrieveKnowledge(ctx context.Context, owner, conversation int64, query, kind string) *knowledgeTrace {
	start := time.Now()
	trace := &knowledgeTrace{Status: "disabled", Facts: []memory.Result{}, CheckedAt: start}
	source := "diagnostic"
	if kind == "chat" {
		source = "assistant"
	}
	id, e := s.beginKnowledge(owner, kind, source, 0, conversation)
	trace.ActivityID = id
	trace.AuditAvailable = e == nil
	var items []memory.Result
	var err error
	client := s.getMemory()
	if client == nil {
		err = &memory.Failure{Code: "disabled"}
	} else {
		searchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		items, err = client.Query(searchCtx, memory.Group(owner), query, 5)
	}
	update := knowledgeUpdate{}
	if err != nil {
		trace.Status = knowledgeFailure(err, false)
		trace.Code = memory.FailureCode(err)
		update.Code = trace.Code
	} else {
		trace.Facts = items
		trace.Status = "success"
		if len(items) == 0 {
			trace.Status = "empty"
		}
		count := len(items)
		update.ResultCount = &count
		update.Facts = items
	}
	update.Status = trace.Status
	trace.Duration = time.Since(start).Milliseconds()
	trace.AuditAvailable = s.finishKnowledge(owner, id, start, update) && trace.AuditAvailable
	return trace
}
func (s *Server) knowledgeContextIncluded(owner int64, trace *knowledgeTrace) {
	trace.Included = len(trace.Facts) > 0
	if !trace.Included || trace.ActivityID <= 0 || s.DB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, e := s.DB.ExecContext(ctx, `UPDATE knowledge_activity SET context_included=true WHERE id=$1 AND owner_id=$2 AND kind='chat' AND status='success'`, trace.ActivityID, owner)
	if e != nil {
		trace.AuditAvailable = false
		return
	}
	n, e := result.RowsAffected()
	trace.AuditAvailable = trace.AuditAvailable && e == nil && n == 1
}
func (s *Server) knowledgeOverview(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	w.Header().Set("Cache-Control", "no-store")
	before := int64(0)
	if raw := r.URL.Query().Get("before_id"); raw != "" {
		var err error
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 0 || before > 9007199254740991 {
			writeErr(w, 400, errors.New("pagination invalide"))
			return
		}
	}

	rows, e := s.DB.QueryContext(r.Context(), `SELECT a.id,a.kind,a.source,a.status,a.statement_id,a.conversation_id,s.filename,a.fact_count,a.processed_count,a.entities_processed,a.edges_processed,a.result_count,a.context_included,a.facts,a.error_code,a.duration_ms,a.started_at,a.finished_at FROM knowledge_activity a LEFT JOIN statements s ON s.id=a.statement_id AND s.owner_id=a.owner_id WHERE a.owner_id=$1 AND ($2::bigint=0 OR a.id<$2) ORDER BY a.id DESC LIMIT 51`, owner, before)
	if e != nil {
		writeErr(w, 500, errors.New("journal mémoire indisponible"))
		return
	}
	events := []knowledgeActivity{}
	for rows.Next() {
		var a knowledgeActivity
		var data []byte
		if e = rows.Scan(&a.ID, &a.Kind, &a.Source, &a.Status, &a.StatementID, &a.ConversationID, &a.StatementName, &a.FactCount, &a.ProcessedCount, &a.Entities, &a.Edges, &a.ResultCount, &a.Included, &data, &a.Code, &a.Duration, &a.Started, &a.Finished); e != nil {
			break
		}
		if json.Unmarshal(data, &a.Facts) != nil {
			e = errors.New("invalid history")
			break
		}
		a.Stale = a.Status == "running" && time.Since(a.Started) > 3*time.Minute
		events = append(events, a)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		writeErr(w, 500, errors.New("journal mémoire illisible"))
		return
	}
	more := len(events) > 50
	if more {
		events = events[:50]
	}
	var lastLearning *knowledgeActivity
	last := knowledgeActivity{Kind: "learn", Facts: []memory.Result{}}
	e = s.DB.QueryRowContext(r.Context(), `SELECT a.id,a.source,a.status,a.statement_id,s.filename,a.fact_count,a.processed_count,a.entities_processed,a.edges_processed,a.error_code,a.duration_ms,a.started_at,a.finished_at FROM knowledge_activity a LEFT JOIN statements s ON s.id=a.statement_id AND s.owner_id=a.owner_id WHERE a.owner_id=$1 AND a.kind='learn' ORDER BY a.id DESC LIMIT 1`, owner).Scan(&last.ID, &last.Source, &last.Status, &last.StatementID, &last.StatementName, &last.FactCount, &last.ProcessedCount, &last.Entities, &last.Edges, &last.Code, &last.Duration, &last.Started, &last.Finished)
	if e == nil {
		last.Stale = last.Status == "running" && time.Since(last.Started) > 3*time.Minute
		lastLearning = &last
	} else if !errors.Is(e, sql.ErrNoRows) {
		writeErr(w, 500, errors.New("dernier apprentissage indisponible"))
		return
	}
	status, detail := "disabled", "Mémoire désactivée. Le journal antérieur reste consultable."
	var graph *memory.Graph
	var code string
	client := s.getMemory()
	if client != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		graph, e = client.Inspect(ctx, memory.Group(owner))
		if e == nil {
			status = "available"
			detail = "Lecture effective de FalkorDB réussie."
		} else {
			code = memory.FailureCode(e)
			status = knowledgeFailure(e, false)
			detail = e.Error()
		}
	}
	writeJSON(w, 200, map[string]any{"service": map[string]any{"status": status, "detail": detail, "error_code": code, "checked_at": time.Now().UTC()}, "graph": graph, "activity": events, "has_more": more, "last_learning": lastLearning, "scope": "current_user"})
}
func (s *Server) knowledgeSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var input struct {
		Query string `json:"query"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil || d.Decode(new(any)) != io.EOF || strings.TrimSpace(input.Query) == "" || len([]rune(input.Query)) > 500 {
		writeErr(w, 400, errors.New("recherche requise, 500 caractères maximum"))
		return
	}
	trace := s.retrieveKnowledge(r.Context(), s.ownerID(r), 0, strings.TrimSpace(input.Query), "search")
	writeJSON(w, 200, trace)
}

func (s *Server) knowledgeLearn(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Facts []string `json:"facts"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil || d.Decode(new(any)) != io.EOF || len(input.Facts) == 0 || len(input.Facts) > 1000 {
		writeErr(w, 400, errors.New("faits requis, taille limitée à 64 ko"))
		return
	}
	for _, fact := range input.Facts {
		if len([]rune(fact)) == 0 || len([]rune(fact)) > 4096 {
			writeErr(w, 400, errors.New("fait invalide ou trop long"))
			return
		}
	}
	out, e := s.learnKnowledge(r.Context(), s.ownerID(r), 0, "manual", input.Facts)
	if e != nil {
		writeErr(w, 502, e)
		return
	}
	writeJSON(w, 200, map[string]any{"group_id": memory.Group(s.ownerID(r)), "entities_added": out.Entities, "edges_added": out.Edges, "status": out.Status})
}
