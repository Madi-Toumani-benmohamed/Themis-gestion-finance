package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"themis/internal/llm"
)

// ============================================================================
// Types
// ============================================================================

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatIn struct {
	ConversationID *int64    `json:"conversation_id"`
	Messages       []chatMsg `json:"messages"`
}

type llmDecision struct {
	Answer    string          `json:"answer"`
	ToolCalls []toolCallInner `json:"tool_calls"`
	Status    string          `json:"status"`
	Evidence  []any           `json:"evidence"`
}

type toolCallInner struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// toolResult est le retour d'un outil analytique.
type toolResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// chatResult est le résultat final de l'orchestration.
type chatResult struct {
	Answer      string
	Evidence    []any
	DataQuality map[string]any
	Status      string
}

// ============================================================================
// Outils analytiques (lecture seule)
// ============================================================================

func toolSearchTransactions(db *sql.DB, owner int64, args map[string]any) toolResult {
	var startDate, endDate string
	if v, ok := args["start_date"].(string); ok {
		startDate = v
	}
	if v, ok := args["end_date"].(string); ok {
		endDate = v
	}
	if startDate == "" || endDate == "" {
		return toolResult{Error: "start_date et end_date requis (format YYYY-MM-DD)"}
	}
	var limit int = 20
	if v, ok := args["limit"].(float64); ok && v > 0 && v <= 50 {
		limit = int(v)
	}
	var merchant string
	if v, ok := args["merchant"].(string); ok {
		merchant = v
	}
	q := `SELECT t.tx_date::text, t.label, t.amount, COALESCE(d.name,'Sans domaine')
		  FROM transactions t LEFT JOIN domains d ON d.id = t.domain_id
		  WHERE t.owner_id = $1 AND t.tx_date >= $2::date AND t.tx_date <= $3::date` + notTransfer
	args2 := []any{owner, startDate, endDate}
	argIdx := 4
	if merchant != "" {
		args2 = append(args2, "%"+strings.ToLower(merchant)+"%")
		q += fmt.Sprintf(" AND (ILIKE t.label $%d OR ILIKE t.display_label $%d)", argIdx, argIdx)
		argIdx++
	}
	q += fmt.Sprintf(" ORDER BY t.tx_date DESC LIMIT $%d", argIdx)
	args2 = append(args2, limit)

	rows, err := db.Query(q, args2...)
	if err != nil {
		return toolResult{Error: err.Error()}
	}
	defer rows.Close()
	type tx struct {
		Date  string  `json:"date"`
		Label string  `json:"label"`
		Amt   float64 `json:"amount"`
		Cat   string  `json:"category"`
	}
	out := []tx{}
	for rows.Next() {
		var x tx
		if err := rows.Scan(&x.Date, &x.Label, &x.Amt, &x.Cat); err != nil {
			continue
		}
		out = append(out, x)
	}
	b, _ := json.Marshal(out)
	return toolResult{Content: string(b)}
}

func toolFinancialSummary(db *sql.DB, owner int64, args map[string]any) toolResult {
	var startDate, endDate string
	if v, ok := args["start_date"].(string); ok {
		startDate = v
	}
	if v, ok := args["end_date"].(string); ok {
		endDate = v
	}
	if startDate == "" || endDate == "" {
		return toolResult{Error: "start_date et end_date requis"}
	}
	groupBy := "none"
	if v, ok := args["group_by"].(string); ok {
		if v == "category" || v == "merchant" {
			groupBy = v
		}
	}
	if groupBy == "none" {
		var income, expenses float64
		err := db.QueryRow(`
			SELECT COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0),
			       COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0)
			FROM transactions
			WHERE owner_id = $1 AND tx_date >= $2::date AND tx_date <= $3::date`+notTransfer,
			owner, startDate, endDate).Scan(&income, &expenses)
		if err != nil {
			return toolResult{Error: err.Error()}
		}
		b, _ := json.Marshal(map[string]any{"income": income, "expenses": expenses, "net": income - expenses})
		return toolResult{Content: string(b)}
	}
	col := "COALESCE(d.name, 'Sans domaine')"
	if groupBy == "merchant" {
		col = "COALESCE(NULLIF(t.merchant,''), t.display_label, t.label)"
	}
	rows, err := db.Query(fmt.Sprintf(`
		SELECT %s,
		       COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0),
		       COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0)
		FROM transactions t LEFT JOIN domains d ON d.id = t.domain_id
		WHERE t.owner_id = $1 AND t.tx_date >= $2::date AND t.tx_date <= $3::date`+notTransfer+`
		GROUP BY 1 ORDER BY 3 DESC`, col), owner, startDate, endDate)
	if err != nil {
		return toolResult{Error: err.Error()}
	}
	defer rows.Close()
	type row struct {
		Key      string  `json:"key"`
		Income   float64 `json:"income"`
		Expenses float64 `json:"expenses"`
	}
	out := []row{}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.Key, &r.Income, &r.Expenses); err != nil {
			continue
		}
		out = append(out, r)
	}
	b, _ := json.Marshal(out)
	return toolResult{Content: string(b)}
}

func toolComparePeriods(db *sql.DB, owner int64, args map[string]any) toolResult {
	getPeriod := func(m map[string]any, key string) (string, string) {
		period, _ := m[key].(map[string]any)
		if period == nil {
			return "", ""
		}
		sa, _ := period["start_date"].(string)
		ea, _ := period["end_date"].(string)
		return sa, ea
	}
	startA, endA := getPeriod(args, "period_a")
	startB, endB := getPeriod(args, "period_b")
	if startA == "" || endA == "" || startB == "" || endB == "" {
		return toolResult{Error: "period_a et period_b requis"}
	}
	metric, _ := args["metric"].(string)
	if metric != "income" && metric != "net" {
		metric = "expenses"
	}
	groupBy, _ := args["group_by"].(string)

	type agg struct {
		Key  string  `json:"key"`
		ValA float64 `json:"value_a"`
		ValB float64 `json:"value_b"`
	}
	var results []agg

	if groupBy == "category" {
		expOp := "SUM(amount) FILTER (WHERE amount < 0)"
		inOp := "SUM(amount) FILTER (WHERE amount > 0)"
		if metric == "income" {
			expOp, inOp = inOp, expOp
		}
		rows, err := db.Query(fmt.Sprintf(`
			WITH cur AS (
				SELECT COALESCE(d.name,'Sans domaine') AS k, %s AS v
				FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id
				WHERE t.owner_id=$1 AND t.tx_date>=$2::date AND t.tx_date<=$3::date`+notTransfer+`
				GROUP BY k
			), hist AS (
				SELECT COALESCE(d.name,'Sans domaine') AS k, %s AS v
				FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id
				WHERE t.owner_id=$1 AND t.tx_date>=$4::date AND t.tx_date<=$5::date`+notTransfer+`
				GROUP BY k
			)
			SELECT COALESCE(cur.k,hist.k), COALESCE(cur.v,0), COALESCE(hist.v,0)
			FROM cur FULL OUTER JOIN hist ON cur.k=hist.k
			ORDER BY abs(cur.v-hist.v) DESC LIMIT 10`, expOp, expOp),
			owner, startA, endA, startB, endB)
		if err != nil {
			return toolResult{Error: err.Error()}
		}
		defer rows.Close()
		for rows.Next() {
			var a agg
			if err := rows.Scan(&a.Key, &a.ValA, &a.ValB); err != nil {
				continue
			}
			results = append(results, a)
		}
	} else {
		var valA, valB float64
		if metric == "expenses" {
			db.QueryRow(`SELECT COALESCE(-SUM(amount) FILTER (WHERE amount < 0),0) FROM transactions WHERE owner_id=$1 AND tx_date>=$2::date AND tx_date<=$3::date`+notTransfer, owner, startA, endA).Scan(&valA)
			db.QueryRow(`SELECT COALESCE(-SUM(amount) FILTER (WHERE amount < 0),0) FROM transactions WHERE owner_id=$1 AND tx_date>=$2::date AND tx_date<=$3::date`+notTransfer, owner, startB, endB).Scan(&valB)
		} else if metric == "income" {
			db.QueryRow(`SELECT COALESCE(SUM(amount) FILTER (WHERE amount > 0),0) FROM transactions WHERE owner_id=$1 AND tx_date>=$2::date AND tx_date<=$3::date`+notTransfer, owner, startA, endA).Scan(&valA)
			db.QueryRow(`SELECT COALESCE(SUM(amount) FILTER (WHERE amount > 0),0) FROM transactions WHERE owner_id=$1 AND tx_date>=$2::date AND tx_date<=$3::date`+notTransfer, owner, startB, endB).Scan(&valB)
		} else {
			db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE owner_id=$1 AND tx_date>=$2::date AND tx_date<=$3::date`+notTransfer, owner, startA, endA).Scan(&valA)
			db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM transactions WHERE owner_id=$1 AND tx_date>=$2::date AND tx_date<=$3::date`+notTransfer, owner, startB, endB).Scan(&valB)
		}
		results = []agg{{Key: "total", ValA: valA, ValB: valB}}
	}

	type compResult struct {
		Key            string   `json:"key"`
		ValueA         float64  `json:"value_a"`
		ValueB         float64  `json:"value_b"`
		AbsoluteChange float64  `json:"absolute_change"`
		RelChangePct   *float64 `json:"relative_change_percent,omitempty"`
	}
	out := []compResult{}
	for _, r := range results {
		change := r.ValA - r.ValB
		pct := 0.0
		if r.ValB != 0 {
			pct = change / r.ValB * 100
		}
		out = append(out, compResult{Key: r.Key, ValueA: r.ValA, ValueB: r.ValB, AbsoluteChange: change, RelChangePct: &pct})
	}
	b, _ := json.Marshal(out)
	return toolResult{Content: string(b)}
}

func toolCategoryBreakdown(db *sql.DB, owner int64, args map[string]any) toolResult {
	var startDate, endDate string
	if v, ok := args["start_date"].(string); ok {
		startDate = v
	}
	if v, ok := args["end_date"].(string); ok {
		endDate = v
	}
	if startDate == "" || endDate == "" {
		return toolResult{Error: "start_date et end_date requis"}
	}
	direction, _ := args["direction"].(string)
	if direction != "income" {
		direction = "expense"
	}
	var limit int = 10
	if v, ok := args["limit"].(float64); ok && v > 0 && v <= 20 {
		limit = int(v)
	}
	op := "-SUM(amount) FILTER (WHERE amount < 0)"
	if direction == "income" {
		op = "SUM(amount) FILTER (WHERE amount > 0)"
	}
	rows, err := db.Query(fmt.Sprintf(`
		SELECT COALESCE(d.name,'Sans domaine') AS cat, %s AS total
		FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id
		WHERE t.owner_id=$1 AND t.tx_date>=$2::date AND t.tx_date<=$3::date`+notTransfer+`
		GROUP BY 1 ORDER BY total DESC LIMIT $4`, op), owner, startDate, endDate, limit)
	if err != nil {
		return toolResult{Error: err.Error()}
	}
	defer rows.Close()
	type cat struct {
		Cat   string  `json:"category"`
		Total float64 `json:"total"`
		Pct   float64 `json:"pct"`
	}
	out := []cat{}
	var totalAll float64
	for rows.Next() {
		var c cat
		if err := rows.Scan(&c.Cat, &c.Total); err != nil {
			continue
		}
		out = append(out, c)
		totalAll += c.Total
	}
	for i := range out {
		if totalAll > 0 {
			out[i].Pct = out[i].Total / totalAll * 100
		}
	}
	b, _ := json.Marshal(out)
	return toolResult{Content: string(b)}
}

func toolDataQuality(db *sql.DB, owner int64) toolResult {
	var totalTx, categorized, uncategorized int
	var firstDate, lastDate sql.NullString
	db.QueryRow(`SELECT count(*), count(domain_id), count(*)-count(domain_id),
	                   min(tx_date)::text, max(tx_date)::text
	                FROM transactions WHERE owner_id=$1`, owner).Scan(&totalTx, &categorized, &uncategorized, &firstDate, &lastDate)
	var stmtCount int
	db.QueryRow(`SELECT count(*) FROM statements WHERE owner_id=$1`, owner).Scan(&stmtCount)
	var monthsCovered int
	db.QueryRow(`SELECT count(DISTINCT to_char(tx_date,'YYYY-MM')) FROM transactions WHERE owner_id=$1`, owner).Scan(&monthsCovered)
	out := map[string]any{
		"total_transactions": totalTx, "categorized": categorized, "uncategorized": uncategorized,
		"statements": stmtCount, "months_covered": monthsCovered,
		"first_tx": firstDate.String, "last_tx": lastDate.String,
		"coverage": map[string]any{"complete": false, "status": "unknown", "reason": "Le nombre de transactions ne prouve pas la couverture documentaire", "sufficient": monthsCovered >= 2 && uncategorized < int(float64(totalTx)*0.3)},
	}
	b, _ := json.Marshal(out)
	return toolResult{Content: string(b)}
}

// ============================================================================
// Parsing de la réponse LLM
// ============================================================================

func parseLLMDecision(resp string) *llmDecision {
	resp = strings.TrimSpace(resp)
	start := strings.Index(resp, "{")
	end := strings.LastIndex(resp, "}")
	if start < 0 || end <= start {
		return nil
	}
	jsonStr := resp[start : end+1]
	var dec llmDecision
	if err := json.Unmarshal([]byte(jsonStr), &dec); err != nil {
		return nil
	}
	return &dec
}

// ============================================================================
// Système de messages
// ============================================================================

func formatMessages(msgs []chatMsg) string {
	var b strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case "user":
			b.WriteString(fmt.Sprintf("Utilisateur : %s\n", m.Content))
		case "assistant":
			b.WriteString(fmt.Sprintf("Assistant : %s\n", m.Content))
		case "tool":
			b.WriteString(fmt.Sprintf("Résultat outil : %s\n", m.Content))
		}
	}
	return b.String()
}

func buildContextMessages(history []chatMsg, lastUser string) []chatMsg {
	if len(history) > 8 {
		history = history[len(history)-8:]
	}
	return append(history, chatMsg{Role: "user", Content: lastUser})
}

// ============================================================================
// Handler principal
// ============================================================================

func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	if s.getLLM() == nil {
		writeErr(w, http.StatusNotImplemented, fmt.Errorf("assistant indisponible : aucune IA configurée"))
		return
	}

	var req chatIn
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	if len(req.Messages) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("au moins un message requis"))
		return
	}

	owner := s.ownerID(r)

	// Résoudre ou créer la conversation
	var convID int64
	if req.ConversationID != nil && *req.ConversationID > 0 {
		var checkOwner int64
		err := s.DB.QueryRow(`SELECT owner_id FROM chat_conversations WHERE id = $1`, *req.ConversationID).Scan(&checkOwner)
		if err == nil && checkOwner == owner {
			convID = *req.ConversationID
		}
	}
	if convID == 0 {
		lastMsg := req.Messages[len(req.Messages)-1].Content
		err := s.DB.QueryRow(
			`INSERT INTO chat_conversations (owner_id, title) VALUES ($1, $2) RETURNING id`,
			owner, truncate(lastMsg, 60),
		).Scan(&convID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}

	// Persister les messages utilisateur
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			_, _ = s.DB.Exec(`INSERT INTO chat_messages (conversation_id, role, content) VALUES ($1, $2, $3)`, convID, msg.Role, msg.Content)
		}
	}

	// Charger l'historique
	rows, err := s.DB.Query(`SELECT role, content FROM chat_messages WHERE conversation_id = $1 ORDER BY id ASC`, convID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	var history []chatMsg
	for rows.Next() {
		var m chatMsg
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			continue
		}
		history = append(history, m)
	}
	rows.Close()

	// Orchestrer
	ctx, cancel := context.WithTimeout(r.Context(), llm.ChatTimeout())
	defer cancel()

	result, err := s.orchestrate(ctx, s.DB, owner, history, req.Messages[len(req.Messages)-1].Content)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}

	// Persister la réponse
	_, _ = s.DB.Exec(`INSERT INTO chat_messages (conversation_id, role, content) VALUES ($1, $2, $3)`, convID, "assistant", result.Answer)
	_, _ = s.DB.Exec(`UPDATE chat_conversations SET updated_at = now() WHERE id = $1`, convID)

	writeJSON(w, http.StatusOK, map[string]any{
		"conversation_id": convID,
		"answer":          result.Answer,
		"evidence":        result.Evidence,
		"data_quality":    result.DataQuality,
		"status":          result.Status,
	})
}

func (s *Server) orchestrate(ctx context.Context, db *sql.DB, owner int64, history []chatMsg, lastUser string) (*chatResult, error) {
	client := s.getLLM()
	if client == nil {
		return nil, fmt.Errorf("IA non disponible")
	}

	systemPrompt := buildSystemPrompt()

	// Enrichir le prompt système avec les connaissances du graphe si dispo.
	memCtx := ""
	if mem := s.getMemory(); mem != nil && mem.IsReady(ctx) {
		memCtx = mem.SearchFacts(ctx, fmt.Sprint(owner), lastUser, 5)
	}
	if memCtx != "" {
		systemPrompt += "\nMémoire indicative, potentiellement périmée : ne constitue jamais une preuve de montant ou de complétude. Vérifier tout chiffre via les outils transactionnels, citer la période et la source ; en cas de contradiction les transactions PostgreSQL font référence.\n" + memCtx
	}

	contextMsgs := buildContextMessages(history, lastUser)
	toolsDef, _ := json.Marshal([]map[string]string{
		{"name": "search_transactions", "description": "Rechercher des transactions par période et critère"},
		{"name": "get_financial_summary", "description": "Calculer revenus/dépenses nets sur une période"},
		{"name": "compare_periods", "description": "Comparer deux périodes entre elles"},
		{"name": "get_category_breakdown", "description": "Ventilation des dépenses par catégorie"},
		{"name": "get_data_quality", "description": "Vérifier la qualité et la couverture des données"},
	})

	var finalAnswer string
	var evidence []any
	status := "ok"
	context := contextMsgs

	for round := 0; round < 4; round++ {
		msgText := formatMessages(context)
		resp, err := client.Complete(ctx, systemPrompt+string(toolsDef), msgText, 2000, true)
		if err != nil {
			return &chatResult{Answer: "Erreur de connexion au modèle IA. Vérifiez la configuration Ollama.", Status: "error"}, nil
		}

		decision := parseLLMDecision(resp)
		if decision == nil {
			finalAnswer = strings.TrimSpace(resp)
			break
		}

		if decision.Answer != "" {
			finalAnswer = strings.TrimSpace(decision.Answer)
			evidence = decision.Evidence
			if decision.Status != "" {
				status = decision.Status
			}
			break
		}

		if len(decision.ToolCalls) == 0 {
			finalAnswer = "Je n'ai pas trouvé assez d'informations. Pouvez-vous préciser votre question ?"
			status = "needs_clarification"
			break
		}

		// Exécuter les outils
		for _, tc := range decision.ToolCalls {
			var args map[string]any
			_ = json.Unmarshal(tc.Arguments, &args)

			var tr toolResult
			switch tc.Name {
			case "search_transactions":
				tr = toolSearchTransactions(db, owner, args)
			case "get_financial_summary":
				tr = toolFinancialSummary(db, owner, args)
			case "compare_periods":
				tr = toolComparePeriods(db, owner, args)
			case "get_category_breakdown":
				tr = toolCategoryBreakdown(db, owner, args)
			case "get_data_quality":
				tr = toolDataQuality(db, owner)
			default:
				tr = toolResult{Error: "outil inconnu: " + tc.Name}
			}

			context = append(context, chatMsg{Role: "assistant", Content: fmt.Sprintf("[Outil %s appelé]", tc.Name)})
			if tr.Error != "" {
				context = append(context, chatMsg{Role: "tool", Content: fmt.Sprintf("Erreur outil %s: %s", tc.Name, tr.Error)})
			} else {
				context = append(context, chatMsg{Role: "tool", Content: tr.Content})
			}
		}
	}

	if finalAnswer == "" {
		finalAnswer = "Désolé, je n'ai pas pu traiter votre question. Veuillez réessayer."
	}

	return &chatResult{
		Answer:      finalAnswer,
		Evidence:    evidence,
		DataQuality: map[string]any{"status": "complete"},
		Status:      status,
	}, nil
}

func buildSystemPrompt() string {
	return `Tu es l'assistant financier de thémis. Tu consultes les DONNÉES RÉELLES de l'utilisateur via des outils.

RÈGLES ABSOLUES :
1. Tu NE CALCULES JAMAIS de montants toi-même. Utilise TOUJOURS les outils.
2. Les chiffres dans ta réponse DOIVENT correspondre EXACTEMENT aux résultats des outils. Ne modifie aucun montant.
3. Réponds en français, en 3-5 phrases maximum.
4. Ne donne JAMAIS de conseil financier engageant. Présente les faits.
5. Si les outils retournent une erreur, dis-le clairement.
6. Une absence de transaction ne prouve ni un zéro réel ni une période complète. Le salaire peut arriver en fin de mois ; un déficit provisoire ne justifie aucune qualification négative.
7. Sépare faits transactionnels, hypothèses et projections. Cite l'outil et les dates de référence des chiffres. La mémoire Knowledge Graph ne remplace jamais les outils transactionnels.
8. Ne conclus pas à une évolution entre périodes dont la couverture documentaire n'est pas établie comparable.

FORMAT QUAND TU AS LES INFOS :
{"answer": "Réponse avec les chiffres EXACTS des outils.", "status": "ok"}

FORMAT QUAND TU BESOINS D'UN OUTIL :
{"tool_calls": [{"id":"call_1","name":"nom_outil","arguments":{"param":"valeur"}}]}

FORMAT SI TU AS BESOIN D'UNE PRÉCISION :
{"answer": "Question à préciser...", "status": "needs_clarification"}

PÉRIODES COURANTES (traduis en dates YYYY-MM-DD) :
- « ce mois » = du 1er du mois courant au jour actuel
- « mois dernier » = mois complet précédent
- « ces 3 derniers mois » = 3 mois glissants depuis aujourd'hui
- « cette année » = janvier au jour actuel
- « l'année dernière » = année civile précédente

OUTILS DISPONIBLES :
- search_transactions : cherche des transactions par période/marchand/montant
- get_financial_summary : revenus, dépenses, net sur une période
- compare_periods : compare deux périodes entre elles
- get_category_breakdown : ventilation par catégorie
- get_data_quality : état de la couverture des données

IMPORTANT : copie les chiffres EXACTS des résultats d'outils dans ta réponse. Ne devine jamais un montant.`
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
