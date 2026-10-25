package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"themis/internal/ai"
	"themis/internal/categorize"
	"themis/internal/config"
	"themis/internal/memory"
	"themis/internal/models"
	"themis/internal/money"
	"themis/internal/normalize"
	"themis/internal/parser"
)

type Server struct {
	DB             *sql.DB
	Cfg            config.Config
	ai             aiState        // config + client IA actifs (modifiables à chaud via l'admin)
	Memory         *memory.Client // knowledge graph client (Graphiti)
	importPreviews importPreviewStore
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/statements", s.uploadStatement)
	mux.HandleFunc("POST /api/statements/previews", s.uploadStatement)
	mux.HandleFunc("POST /api/statements/previews/{token}/confirm", s.confirmImportPreview)
	mux.HandleFunc("DELETE /api/statements/previews/{token}", s.cancelImportPreview)
	mux.HandleFunc("GET /api/statements", s.listStatements)
	mux.HandleFunc("DELETE /api/statements/{id}", s.deleteStatement)
	mux.HandleFunc("GET /api/transactions", s.listTransactions)
	mux.HandleFunc("PATCH /api/transactions/{id}", s.updateTransaction)
	mux.HandleFunc("GET /api/categories", s.listCategories)
	mux.HandleFunc("GET /api/rules", s.listRules)
	mux.HandleFunc("POST /api/rules", s.createRule)
	mux.HandleFunc("POST /api/rules/apply", s.applyRules)
	mux.HandleFunc("DELETE /api/rules/{id}", s.deleteRule)
	mux.HandleFunc("PATCH /api/rules/{id}", s.updateRule)
	mux.HandleFunc("GET /api/summary", s.summary)
	mux.HandleFunc("GET /api/pulse", s.pulse)
	mux.HandleFunc("GET /api/anomalies", s.anomalies)
	mux.HandleFunc("GET /api/journal", s.journalFeed)
	mux.HandleFunc("GET /api/remaining", s.remaining)
	mux.HandleFunc("GET /api/upload/progress", s.uploadProgress)
	mux.HandleFunc("GET /api/coverage", s.coverage)
	mux.HandleFunc("GET /api/reliability", s.reliability)
	mux.HandleFunc("GET /api/intel/merchants", s.merchantsView)
	mux.HandleFunc("GET /api/intel/nature", s.natureBreakdown)
	mux.HandleFunc("GET /api/intel/comparison", s.categoryComparison)
	mux.HandleFunc("GET /api/intel/discoveries", s.discoveries)
	mux.HandleFunc("GET /api/intel/narrative", s.intelNarrative)
	mux.HandleFunc("GET /api/hero", s.hero)
	mux.HandleFunc("GET /api/coherence", s.coherence)
	mux.HandleFunc("GET /api/period", s.periodHandler)
	mux.HandleFunc("GET /api/period/compare", s.periodCompareHandler)
	mux.HandleFunc("GET /api/merchants/unrecognized", s.unrecognizedMerchants)
	mux.HandleFunc("GET /api/merchants/aliases", s.listAliases)
	mux.HandleFunc("POST /api/merchants/aliases", s.setAlias)
	mux.HandleFunc("DELETE /api/merchants/aliases/{id}", s.deleteAlias)
	mux.HandleFunc("POST /api/merchants/suggest", s.suggestMerchantNames)
	mux.HandleFunc("GET /api/analytics/tags-in-category", s.tagsInCategory)
	mux.HandleFunc("GET /api/analytics/tag-series", s.tagSeries)
	mux.HandleFunc("GET /api/budgets", s.listBudgets)
	mux.HandleFunc("GET /api/budgets/report", s.budgetReportHandler)
	mux.HandleFunc("POST /api/budgets", s.upsertBudget)
	mux.HandleFunc("DELETE /api/budgets/{id}", s.deleteBudget)
	mux.HandleFunc("GET /api/recurring", s.listRecurring)
	mux.HandleFunc("GET /api/recurring/review", s.listRecurringReview)
	mux.HandleFunc("PUT /api/recurring/review/{key}", s.updateRecurringReview)
	mux.HandleFunc("POST /api/enrich/ai", s.aiEnrich)
	mux.HandleFunc("GET /api/admin/config", s.aiAdminOnly(s.adminGetConfig))
	mux.HandleFunc("PUT /api/admin/config", s.aiAdminOnly(s.adminPutConfig))
	mux.HandleFunc("POST /api/admin/ai/test", s.aiAdminOnly(s.adminTestAI))
	mux.HandleFunc("GET /api/admin/ollama/models", s.aiAdminOnly(s.adminOllamaModels))
	mux.HandleFunc("POST /api/admin/ai/memory/test", s.aiAdminOnly(s.adminTestMemory))
	mux.HandleFunc("GET /api/admin/stats", s.adminStats)
	mux.HandleFunc("POST /api/admin/reapply-rules", s.adminReapplyRules)
	mux.HandleFunc("POST /api/admin/purge", s.adminPurge)
	mux.HandleFunc("GET /api/settings/{key}", s.getSetting)
	mux.HandleFunc("PUT /api/settings/{key}", s.putSetting)
	mux.HandleFunc("GET /api/transactions/export", s.exportTransactions)
	mux.HandleFunc("GET /api/analyses/{month}", s.getAnalysis)
	mux.HandleFunc("POST /api/analyses/{month}", s.createAnalysis)
	mux.HandleFunc("POST /api/chat", s.chatHandler)

	// Knowledge Graph
	mux.HandleFunc("GET /api/knowledge/health", s.kgHealth)
	mux.HandleFunc("POST /api/knowledge/learn", s.kgLearn)

	// Authentification
	mux.HandleFunc("GET /api/auth/status", s.authStatus)
	mux.HandleFunc("POST /api/auth/setup", s.authSetup)
	mux.HandleFunc("POST /api/auth/login", s.authLogin)
	mux.HandleFunc("POST /api/auth/logout", s.authLogout)
	mux.HandleFunc("GET /api/auth/me", s.authMe)
	mux.HandleFunc("POST /api/auth/password", s.authChangePassword)
	mux.HandleFunc("GET /api/export", s.exportAll)
	mux.HandleFunc("GET /api/admin/users", s.adminListUsers)
	mux.HandleFunc("POST /api/admin/users", s.adminCreateUser)
	mux.HandleFunc("DELETE /api/admin/users/{id}", s.adminDeleteUser)

	// Multi-comptes
	mux.HandleFunc("GET /api/accounts", s.listAccounts)
	mux.HandleFunc("POST /api/accounts", s.createAccount)
	mux.HandleFunc("DELETE /api/accounts/{id}", s.deleteAccount)
	mux.HandleFunc("POST /api/accounts/{id}/snapshots", s.addSnapshot)

	// Crédits, taxes, dettes
	mux.HandleFunc("GET /api/credits", s.engagementAuthenticated(s.listCredits))
	mux.HandleFunc("POST /api/credits", s.engagementAuthenticated(s.createCredit))
	mux.HandleFunc("PATCH /api/credits/{id}", s.engagementAuthenticated(s.updateCredit))
	mux.HandleFunc("DELETE /api/credits/{id}", s.engagementAuthenticated(s.deleteCredit))
	mux.HandleFunc("GET /api/taxes", s.engagementAuthenticated(s.listTaxes))
	mux.HandleFunc("POST /api/taxes", s.engagementAuthenticated(s.createTax))
	mux.HandleFunc("PATCH /api/taxes/{id}", s.engagementAuthenticated(s.patchTax))
	mux.HandleFunc("DELETE /api/taxes/{id}", s.engagementAuthenticated(s.deleteTax))
	mux.HandleFunc("GET /api/obligations", s.engagementAuthenticated(s.obligations))
	mux.HandleFunc("POST /api/credits/{id}/payments", s.engagementAuthenticated(s.recordCreditPayment))
	mux.HandleFunc("DELETE /api/credits/{id}/payments/{payment_id}", s.engagementAuthenticated(s.voidCreditPayment))
	mux.HandleFunc("GET /api/credits/{id}/history", s.engagementAuthenticated(s.creditHistory))
	mux.HandleFunc("GET /api/taxes/{id}/history", s.engagementAuthenticated(s.taxHistory))
	mux.HandleFunc("GET /api/insights", s.listInsights)

	// Domaines & tags
	mux.HandleFunc("GET /api/domains", s.listDomains)
	mux.HandleFunc("POST /api/domains", s.createDomain)
	mux.HandleFunc("PATCH /api/domains/{id}", s.updateDomain)
	mux.HandleFunc("DELETE /api/domains/{id}", s.deleteDomain)
	mux.HandleFunc("GET /api/tags", s.listTags)
	mux.HandleFunc("PATCH /api/tags/{name}", s.renameTag)
	mux.HandleFunc("DELETE /api/tags/{name}", s.deleteTag)
	mux.HandleFunc("GET /api/tag-folders", s.tagFolders)
	mux.HandleFunc("GET /api/tag-folders/{name}", s.tagFolderDetail)
	mux.HandleFunc("PUT /api/tag-folders/{name}/pref", s.setTagFolderPref)

	// Graphe par catégories & simulation
	mux.HandleFunc("GET /api/category-series", s.categorySeries)
	mux.HandleFunc("POST /api/simulate", s.simulate)

	return securityHeaders(logMiddleware(s.authMiddleware(mux)))
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.DB.Ping(); err != nil {
		writeErr(w, http.StatusServiceUnavailable, err)
		return
	}
	ocrOK, ocrDetail := parser.OCRStatus()
	aiProvider := "none"
	if c := s.getLLM(); c != nil {
		aiProvider = c.Name()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"ai_enabled":  s.getLLM() != nil,
		"ai_provider": aiProvider,
		"ocr":         ocrOK,
		"ocr_detail":  ocrDetail,
	})
}

// ---------- Knowledge Graph ----------

func (s *Server) kgHealth(w http.ResponseWriter, r *http.Request) {
	if s.getMemory() == nil || !s.getMemory().IsReady(r.Context()) {
		writeErr(w, http.StatusServiceUnavailable, fmt.Errorf("service knowledge graph indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) kgLearn(w http.ResponseWriter, r *http.Request) {
	if s.getMemory() == nil || !s.getMemory().IsReady(r.Context()) {
		writeErr(w, http.StatusServiceUnavailable, fmt.Errorf("service knowledge graph indisponible"))
		return
	}
	var in struct {
		Facts []string `json:"facts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	if len(in.Facts) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("au moins un fait requis"))
		return
	}
	owner := s.ownerID(r)
	groupID := fmt.Sprint(owner)
	entities, edges, err := s.getMemory().Learn(r.Context(), groupID, "transaction_import", in.Facts)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"group_id":       groupID,
		"entities_added": entities,
		"edges_added":    edges,
	})
}

// ---------- Relevés ----------

func (s *Server) uploadStatement(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/statements/previews" {
		w.Header().Set("Cache-Control", "no-store")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20) // 20 Mo max
	file, header, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("champ 'file' manquant: %w", err))
		return
	}
	defer file.Close()
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	data, err := io.ReadAll(file)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	// Progression : si le client a fourni un jeton, publier chaque étape
	// réelle du pipeline pour qu'il puisse l'afficher pendant l'attente.
	progressToken := r.FormValue("progress_token")
	ownerForProgress := s.ownerID(r)
	setUploadProgress(progressToken, ownerForProgress, "Lecture du fichier", "", 2, false)
	ctx = parser.WithProgress(ctx, func(stage, detail string, pct int) {
		setUploadProgress(progressToken, ownerForProgress, stage, detail, pct, false)
	})
	var res *models.ParseResult
	if strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		res, err = parser.ParseCSV(data)
	} else {
		res, err = parser.Parse(ctx, data, s.getLLM())
	}
	if err != nil {
		setUploadProgress(progressToken, ownerForProgress, "Échec du parsing", "", 100, true)
		writeErr(w, http.StatusUnprocessableEntity, fmt.Errorf("parsing impossible: %w", err))
		return
	}

	owner := s.ownerID(r)
	// Compte cible : celui demandé s'il appartient à l'utilisateur, sinon son
	// compte principal (on ne dépose jamais dans le compte d'autrui).
	var accountID int64
	if v := r.FormValue("account_id"); v != "" {
		if id, e := strconv.ParseInt(v, 10, 64); e == nil {
			var ok bool
			_ = s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1 AND owner_id = $2)`, id, owner).Scan(&ok)
			if ok {
				accountID = id
			}
		}
	}
	if accountID == 0 {
		_ = s.DB.QueryRow(`SELECT id FROM accounts WHERE owner_id = $1 ORDER BY id LIMIT 1`, owner).Scan(&accountID)
	}
	if r.URL.Path == "/api/statements/previews" {
		// Aucun changement silencieux de compte dans le parcours de vérification.
		if requested := r.FormValue("account_id"); requested != "" {
			requestedID, err := strconv.ParseInt(requested, 10, 64)
			if err != nil || requestedID != accountID {
				writeErr(w, http.StatusBadRequest, fmt.Errorf("compte cible indisponible"))
				return
			}
		}
		s.createImportPreview(w, r, header.Filename, res, accountID, progressToken)
		return
	}
	s.finishStatementImport(w, r, header.Filename, res, accountID, owner, progressToken)
}

// Chemin d'enregistrement commun à l'API historique et à la confirmation.
func (s *Server) finishStatementImport(w http.ResponseWriter, r *http.Request, filename string, res *models.ParseResult, accountID, owner int64, progressToken string) {
	ownerForProgress := owner
	setUploadProgress(progressToken, ownerForProgress, "Enregistrement des transactions", "", 88, false)
	inserted, skipped, stmtID, err := s.persist(filename, res, accountID, owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if inserted == 0 {
		// Relevé déjà importé : on ne garde pas d'enregistrement vide
		_, _ = s.DB.Exec(`DELETE FROM statements WHERE id = $1 AND owner_id = $2`, stmtID, owner)
		setUploadProgress(progressToken, ownerForProgress, "Terminé", "", 100, true)
		writeJSON(w, http.StatusOK, map[string]any{
			"statement_id": nil,
			"parser":       res.Parser,
			"validated":    res.Validated,
			"inserted":     0,
			"duplicates":   skipped,
			"message":      "relevé déjà importé, aucune nouvelle transaction",
		})
		return
	}
	// Les règles explicites sont déterministes. Les propositions IA restent
	// séparées et ne modifient jamais les catégories/tags à l'import.
	// Appliquer TOUTES les règles (catégories manquantes + tags) sur les
	// nouvelles transactions. C'est ce qui garantit que les tags d'une règle
	// (« Bouygues → Télécom + [factures, mobile] ») arrivent dès l'import,
	// sans avoir à rééditer la règle après coup.
	setUploadProgress(progressToken, ownerForProgress, "Application des règles", "", 90, false)
	if _, err := s.applyAllRulesForStatement(owner, stmtID); err != nil {
		log.Print("application des règles à l'import interrompue")
	}
	aiSuggested := 0
	if s.getLLM() != nil && s.autoCategorizeEnabled() && inserted > 0 {
		setUploadProgress(progressToken, ownerForProgress, "Suggestions de catégories par l'IA", "", 93, false)
		enr, err := ai.AutoEnrich(r.Context(), s.DB, s.getLLM(), owner, stmtID, "")
		if err != nil {
			log.Print("suggestions IA à l'import indisponibles")
		} else {
			aiSuggested = enr.Suggested
		}
	}

	setUploadProgress(progressToken, ownerForProgress, "Terminé", "", 100, true)

	// Apprendre les habitudes financières dans le graphe de connaissances.
	// Non bloquant : une erreur KG n'empêche pas l'import.
	if s.getMemory() != nil && s.getMemory().IsReady(r.Context()) && inserted > 0 {
		go func() {
			facts := memory.BuildFinancialFacts(res, owner)
			if len(facts) > 0 {
				_, _, _ = s.getMemory().Learn(context.Background(), fmt.Sprint(owner), "transaction_import", facts)
			}
		}()
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"statement_id":   stmtID,
		"parser":         res.Parser,
		"validated":      res.Validated,
		"inserted":       inserted,
		"duplicates":     skipped,
		"ai_categorized": 0, // legacy: proposals are not applied categories
		"ai_suggested":   aiSuggested,
		"period_start":   res.PeriodStart,
		"period_end":     res.PeriodEnd,
	})
}

// aiEnrich stores unverified proposals for explicit user review; no assignment.
func (s *Server) aiEnrich(w http.ResponseWriter, r *http.Request) {
	if s.getLLM() == nil {
		writeErr(w, http.StatusNotImplemented,
			fmt.Errorf("aucun fournisseur d'IA configuré (OLLAMA_URL pour du 100 %% local, ou ANTHROPIC_API_KEY)"))
		return
	}
	var in struct {
		Month string `json:"month"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil && err != io.EOF {
		writeErr(w, 400, fmt.Errorf("demande IA invalide"))
		return
	}
	if in.Month != "" && !validCalendarMonth(in.Month) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("month doit être au format YYYY-MM"))
		return
	}
	res, err := ai.AutoEnrich(r.Context(), s.DB, s.getLLM(), s.ownerID(r), 0, in.Month)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) persist(filename string, res *models.ParseResult, accountID, ownerID int64) (inserted, skipped int, stmtID int64, err error) {
	// Refuser toute ambiguïté avant la première écriture. Les empreintes
	// historiques restent inchangées : aucune réécriture des données existantes.
	seen := map[string]bool{}
	for _, t := range res.Transactions {
		if _, e := money.FromLegacy(t.Amount); e != nil {
			return 0, 0, 0, fmt.Errorf("montant invalide : %w", e)
		}
		if t.Hash == "" || seen[t.Hash] {
			return 0, 0, 0, fmt.Errorf("empreinte absente ou opérations identiques ambiguës : import à vérifier")
		}
		seen[t.Hash] = true
	}
	for _, balance := range []*float64{res.OpeningBalance, res.ClosingBalance} {
		if balance != nil {
			if _, e := money.FromLegacy(*balance); e != nil {
				return 0, 0, 0, fmt.Errorf("solde invalide : %w", e)
			}
		}
	}
	if res.Validated && !parser.Validate(res) {
		return 0, 0, 0, fmt.Errorf("validation comptable incohérente")
	}
	eng, err := categorize.Load(s.DB, ownerID)
	if err != nil {
		return 0, 0, 0, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback()
	// Sérialiser les imports d'un propriétaire pour que la vérification de
	// portée et la contrainte historique de déduplication soient atomiques.
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock($1)`, ownerID); err != nil {
		return 0, 0, 0, err
	}
	for _, t := range res.Transactions {
		var existingAccount sql.NullInt64
		e := tx.QueryRow(`SELECT st.account_id FROM transactions t LEFT JOIN statements st ON st.id=t.statement_id AND st.owner_id=t.owner_id WHERE t.owner_id=$1 AND t.hash=$2`, ownerID, t.Hash).Scan(&existingAccount)
		if e != nil && e != sql.ErrNoRows {
			return 0, 0, 0, e
		}
		if e == nil && (!existingAccount.Valid || existingAccount.Int64 != accountID) {
			return 0, 0, 0, fmt.Errorf("opération similaire sur un autre compte ou de provenance inconnue : déduplication ambiguë, import refusé")
		}
	}
	err = tx.QueryRow(`
  INSERT INTO statements (filename, period_start, period_end, opening_balance, closing_balance, parser, validated, account_id, owner_id)
		VALUES ($1, NULLIF($2,'')::date, NULLIF($3,'')::date, $4, $5, $6, $7, $8, $9) RETURNING id`,
		filename, res.PeriodStart, res.PeriodEnd, res.OpeningBalance, res.ClosingBalance, res.Parser, res.Validated, accountID, ownerID,
	).Scan(&stmtID)
	if err != nil {
		return 0, 0, 0, err
	}

	for _, t := range res.Transactions {
		var domID *int64
		if id, ok := eng.Match(t.Label); ok {
			domID = &id
		}
		norm := normalize.Label(t.Label, t.Amount)
		disp := norm.Display
		if alias, ok := s.aliasFor(ownerID, norm.Merchant, t.Label); ok {
			disp = alias // un surnom existant s'applique dès l'import
		}
		exactAmount, _ := money.FromLegacy(t.Amount)
		r, err := tx.Exec(`
			INSERT INTO transactions (statement_id, tx_date, label, amount, domain_id, hash, owner_id, display_label, merchant, label_kind, classification_source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (owner_id, hash) DO NOTHING`,
			stmtID, t.Date, t.Label, exactAmount.String(), domID, t.Hash, ownerID, disp, norm.Merchant, norm.Kind, classificationForRule(domID))
		if err != nil {
			return 0, 0, 0, err
		}
		if n, _ := r.RowsAffected(); n == 1 {
			inserted++
		} else {
			skipped++
		}
	}
	return inserted, skipped, stmtID, tx.Commit()
}

func (s *Server) listStatements(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`
		SELECT s.id, s.filename, s.period_start::text, s.period_end::text,
		       s.opening_balance, s.closing_balance, s.parser, s.validated, s.uploaded_at::text,
		       (SELECT COUNT(*) FROM transactions t WHERE t.statement_id = s.id)
		FROM statements s WHERE s.owner_id = $1 ORDER BY s.period_start DESC NULLS LAST, s.id DESC`, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []models.Statement{}
	for rows.Next() {
		var st models.Statement
		if err := rows.Scan(&st.ID, &st.Filename, &st.PeriodStart, &st.PeriodEnd,
			&st.OpeningBalance, &st.ClosingBalance, &st.Parser, &st.Validated, &st.UploadedAt, &st.TxCount); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, st)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) deleteStatement(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if _, err := s.DB.Exec(`DELETE FROM statements WHERE id = $1 AND owner_id = $2`, id, s.ownerID(r)); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- Transactions ----------

var reMonth = regexp.MustCompile(`^\d{4}-\d{2}$`)

// ---------- Catégories & règles ----------

// listCategories est conservé comme ALIAS renvoyant les domaines de vie, pour
// que les écrans existants (graphe, budgets) disposent d'une liste { id, name,
// color }. La catégorie n'existe plus : le domaine est la taxonomie unique.
func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`SELECT id, name, color, is_transfer FROM domains WHERE parent_id IS NULL AND owner_id = $1 ORDER BY name`, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	out := []models.Category{}
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.IsTransfer); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, c)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	// Chaque règle est renvoyée avec le nom + la couleur de son domaine cible et
	// le nombre de transactions dont le libellé correspond au motif (pour que
	// l'utilisateur voie la portée de chaque règle avant de la modifier).
	rows, err := s.DB.Query(`
		SELECT r.id, r.pattern, r.domain_id, d.name, d.color, r.tags
		FROM rules r
		JOIN domains d ON d.id = r.domain_id AND d.owner_id=r.owner_id
		WHERE r.owner_id = $1
		ORDER BY d.name, r.pattern`, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	type ruleOut struct {
		ID         int64    `json:"id"`
		Pattern    string   `json:"pattern"`
		DomainID   int64    `json:"domain_id"`
		DomainName string   `json:"domain_name"`
		Color      string   `json:"color"`
		Tags       []string `json:"tags"`
		Matches    int      `json:"matches"`
	}
	out := []ruleOut{}
	for rows.Next() {
		var ru ruleOut
		var tagsCSV string
		if err := rows.Scan(&ru.ID, &ru.Pattern, &ru.DomainID, &ru.DomainName, &ru.Color, &tagsCSV); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		ru.Tags = []string{}
		if tagsCSV != "" {
			ru.Tags = strings.Split(tagsCSV, ",")
		}
		out = append(out, ru)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, 500, fmt.Errorf("règles indisponibles"))
		return
	}
	rows.Close()
	labels, err := s.DB.QueryContext(r.Context(), `SELECT label FROM transactions WHERE owner_id=$1`, s.ownerID(r))
	if err != nil {
		writeErr(w, 500, fmt.Errorf("portée des règles indisponible"))
		return
	}
	defer labels.Close()
	matchers := make([]func(string) bool, len(out))
	for i, rule := range out {
		matchers[i], _ = categorize.CompilePattern(rule.Pattern)
	}
	for labels.Next() {
		var label string
		if err = labels.Scan(&label); err != nil {
			writeErr(w, 500, fmt.Errorf("portée des règles indisponible"))
			return
		}
		for i, match := range matchers {
			if match != nil && match(label) {
				out[i].Matches++
			}
		}
	}
	if labels.Err() != nil {
		writeErr(w, 500, fmt.Errorf("portée des règles indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// createRule crée une règle motif → domaine et, si apply=true, l'applique aux
// transactions sans domaine.
func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Pattern  string   `json:"pattern"`
		DomainID int64    `json:"domain_id"`
		Tags     []string `json:"tags"`
		Apply    bool     `json:"apply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Pattern == "" || body.DomainID == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("pattern et domain_id requis"))
		return
	}
	owner := s.ownerID(r)
	if err := s.validateRule(r.Context(), owner, body.DomainID, body.Pattern); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	tagsCSV := normalizeTagList(body.Tags)
	var id int64
	var replaced bool
	// Upsert : si le motif existe déjà pour CET utilisateur (insensible à la
	// casse), on met à jour sa catégorie ET ses tags plutôt que de créer un doublon.
	err := s.DB.QueryRow(`
		INSERT INTO rules (pattern, domain_id, tags, owner_id) VALUES ($1, $2, $3, $4)
		ON CONFLICT (owner_id, lower(pattern)) DO UPDATE SET domain_id = EXCLUDED.domain_id, tags = EXCLUDED.tags
		RETURNING id, (xmax <> 0) AS updated`,
		body.Pattern, body.DomainID, tagsCSV, owner).Scan(&id, &replaced)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	applied := int64(0)
	if body.Apply {
		applied, err = s.applyRuleToTransactions(owner, body.Pattern, body.DomainID, tagsCSV)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "applied": applied, "replaced": replaced})
}

// normalizeTagList nettoie une liste de tags : trim, retrait des vides, dédup
// insensible à la casse, et renvoie une chaîne CSV (le format de stockage).
func normalizeTagList(tags []string) string {
	seen := map[string]bool{}
	out := []string{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		k := strings.ToLower(t)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, t)
	}
	return strings.Join(out, ",")
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if _, err := s.DB.Exec(`DELETE FROM rules WHERE id = $1 AND owner_id = $2`, id, s.ownerID(r)); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// updateRule modifie le motif et/ou le domaine d'une règle. Si apply=true, la
// règle révisée est immédiatement appliquée aux transactions sans domaine.
func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var body struct {
		Pattern  string   `json:"pattern"`
		DomainID int64    `json:"domain_id"`
		Tags     []string `json:"tags"`
		Apply    bool     `json:"apply"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Pattern == "" || body.DomainID == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("pattern et domain_id requis"))
		return
	}
	owner := s.ownerID(r)
	if err := s.validateRule(r.Context(), owner, body.DomainID, body.Pattern); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	tagsCSV := normalizeTagList(body.Tags)
	res, err := s.DB.Exec(`UPDATE rules SET pattern = $1, domain_id = $2, tags = $3 WHERE id = $4 AND owner_id = $5`,
		body.Pattern, body.DomainID, tagsCSV, id, owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeErr(w, http.StatusNotFound, fmt.Errorf("règle introuvable"))
		return
	}
	applied := int64(0)
	if body.Apply {
		applied, err = s.applyRuleToTransactions(owner, body.Pattern, body.DomainID, tagsCSV)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "applied": applied})
}

// ---------- Résumé & analyses ----------

// summary retourne les séries mensuelles (revenus/dépenses) et la répartition
// par catégorie pour chaque mois des N derniers mois (défaut 12).
// Inclut maintenant les infos de période (mois courant, complet, incomplet).
// Le paramètre 'month' (YYYY-MM) limite la série aux mois jusqu'à ce mois.
func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	months := 12
	if m, err := strconv.Atoi(r.URL.Query().Get("months")); err == nil && m > 0 && m <= 60 {
		months = m
	}
	// Limiter la série aux mois jusqu'au mois sélectionné (pour éviter d'afficher
	// des mois futurs lorsque l'utilisateur navigue vers un mois historique).
	maxMonth := r.URL.Query().Get("month")

	type monthly struct {
		Completeness  string  `json:"completeness"`
		Month         string  `json:"month"`
		Income        float64 `json:"income"`
		Expenses      float64 `json:"expenses"`
		IsCurrent     bool    `json:"is_current"`
		Status        string  `json:"status"` // current | complete | incomplete
		DaysObserved  int     `json:"days_observed"`
		DaysInMonth   int     `json:"days_in_month"`
		CoverageRatio float64 `json:"coverage_ratio"`
	}
	series := []monthly{}
	rows, err := s.DB.Query(`
		SELECT to_char(tx_date, 'YYYY-MM') AS m,
		       COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0),
		       COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0)
		FROM transactions
		WHERE owner_id = $2 AND ($3='' OR to_char(tx_date,'YYYY-MM') <= $3)`+notTransfer+`
		GROUP BY 1 ORDER BY 1 DESC LIMIT $1`, months, owner, maxMonth)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	for rows.Next() {
		var m monthly
		if err := rows.Scan(&m.Month, &m.Income, &m.Expenses); err != nil {
			rows.Close()
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		info, periodErr := s.getPeriodInfo(owner, m.Month)
		if periodErr != nil {
			rows.Close()
			writeErr(w, http.StatusInternalServerError, periodErr)
			return
		}
		m.Completeness = info.Completeness
		m.IsCurrent = info.IsCurrent
		m.Status = info.Status
		m.DaysObserved = info.DaysObserved
		m.DaysInMonth = info.DaysInMonth
		m.CoverageRatio = info.CoverageRatio
		series = append(series, m)
	}
	rows.Close()
	// remettre en ordre chronologique
	for i, j := 0, len(series)-1; i < j; i, j = i+1, j-1 {
		series[i], series[j] = series[j], series[i]
	}

	// Si un mois est sélectionné, filtrer la série pour ne pas dépasser ce mois
	// (utile pour les graphiques historiques : on ne veut pas montrer de mois futurs).
	if maxMonth != "" {
		filtered := []monthly{}
		for _, s := range series {
			if s.Month <= maxMonth {
				filtered = append(filtered, s)
			}
		}
		series = filtered
	}

	type catAgg struct {
		Month    string  `json:"month"`
		Category string  `json:"category"`
		Color    string  `json:"color"`
		Total    float64 `json:"total"`
	}
	cats := []catAgg{}
	selectedMonths := make([]string, 0, len(series))
	for _, m := range series {
		selectedMonths = append(selectedMonths, m.Month)
	}
	crows, err := s.DB.Query(`
		SELECT to_char(t.tx_date, 'YYYY-MM'), COALESCE(d.name, 'Sans domaine'),
		       COALESCE(d.color, '#8896A6'), -SUM(t.amount)
		FROM transactions t LEFT JOIN domains d ON d.id = t.domain_id
		WHERE t.amount < 0 AND t.owner_id = $2`+notTransferT+`
		  AND to_char(t.tx_date, 'YYYY-MM') = ANY($1)
		GROUP BY 1, 2, 3 ORDER BY 1, 4 DESC`, pq.Array(selectedMonths), owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer crows.Close()
	for crows.Next() {
		var c catAgg
		if err := crows.Scan(&c.Month, &c.Category, &c.Color, &c.Total); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		cats = append(cats, c)
	}

	var uncategorized int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM transactions WHERE domain_id IS NULL AND owner_id = $1`, owner).Scan(&uncategorized)

	writeJSON(w, http.StatusOK, map[string]any{
		"monthly":       series,
		"by_category":   cats,
		"uncategorized": uncategorized,
	})
}

func (s *Server) getAnalysis(w http.ResponseWriter, r *http.Request) {
	month := r.PathValue("month")
	var content, createdAt string
	err := s.DB.QueryRow(`SELECT content, created_at::text FROM analyses WHERE month = $1 AND owner_id = $2`, month, s.ownerID(r)).Scan(&content, &createdAt)
	if err == sql.ErrNoRows {
		writeErr(w, http.StatusNotFound, fmt.Errorf("pas d'analyse pour %s", month))
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"month": month, "content": content, "created_at": createdAt})
}

func (s *Server) createAnalysis(w http.ResponseWriter, r *http.Request) {
	month := r.PathValue("month")
	if !reMonth.MatchString(month) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("format attendu: YYYY-MM"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	client := s.getLLM()
	content, err := ai.Analyze(ctx, s.DB, client, s.ownerID(r), month, s.aiContext())
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	model := ""
	if client != nil {
		model = client.Name()
	}
	writeJSON(w, http.StatusCreated, map[string]string{"month": month, "content": content, "model": model})
}

// ---------- Réglages (layout du dashboard, préférences) ----------

var reSettingKey = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

func (s *Server) getSetting(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "ai_config" {
		writeErr(w, 403, fmt.Errorf("utiliser la configuration IA administrative"))
		return
	}
	if !reSettingKey.MatchString(key) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("clé invalide"))
		return
	}
	var raw []byte
	err := s.DB.QueryRow(`SELECT value FROM settings WHERE key = $1 AND owner_id = $2`, key, s.ownerID(r)).Scan(&raw)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusOK, map[string]any{"key": key, "value": nil})
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"key":%q,"value":%s}`, key, raw)
}

func (s *Server) putSetting(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "ai_config" {
		writeErr(w, 403, fmt.Errorf("utiliser la configuration IA administrative"))
		return
	}
	if !reSettingKey.MatchString(key) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("clé invalide"))
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10)) // 64 Ko max
	if err != nil || !json.Valid(body) {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("corps JSON invalide"))
		return
	}
	_, err = s.DB.Exec(`
		INSERT INTO settings (key, value, owner_id) VALUES ($1, $2, $3)
		ON CONFLICT (COALESCE(owner_id, 0), key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		key, body, s.ownerID(r))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
