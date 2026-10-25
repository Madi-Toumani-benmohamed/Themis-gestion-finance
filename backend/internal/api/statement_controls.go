package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"themis/internal/models"
)

// One exact predicate for coverage consumers. The legacy flag alone is not proof
// after deduplication. All aliases and SQL fragments are compile-time constants.
const statementDocumentedSQL = `st.validated AND st.period_start IS NOT NULL AND st.period_end>=st.period_start
 AND EXISTS(SELECT 1 FROM accounts va WHERE va.id=st.account_id AND va.owner_id=st.owner_id)
 AND st.opening_balance IS NOT NULL AND st.closing_balance IS NOT NULL
 AND st.opening_balance+(SELECT COALESCE(SUM(v.amount),0) FROM transactions v WHERE v.statement_id=st.id AND v.owner_id=st.owner_id)=st.closing_balance
 AND NOT EXISTS(SELECT 1 FROM statements prior WHERE prior.owner_id=st.owner_id AND prior.account_id=st.account_id
 AND prior.id<>st.id AND prior.period_end IN (st.period_start,st.period_start-1)
 AND prior.closing_balance IS NOT NULL AND prior.closing_balance<>st.opening_balance)`

type statementControl struct {
	Code      string `json:"code"`
	Status    string `json:"status"` // passed, failed, unknown, suspected, informational, not_applicable
	Severity  string `json:"severity"`
	Detail    string `json:"detail"`
	RelatedID int64  `json:"related_id,omitempty"`
}
type statementReview struct {
	ID     int64  `json:"id"`
	Action string `json:"action"`
	Note   string `json:"note"`
	At     string `json:"at"`
}
type checkedStatement struct {
	ImportReport *models.ImportReport `json:"import_report,omitempty"`
	models.Statement
	AccountID          *int64             `json:"account_id"`
	AccountName        string             `json:"account_name"`
	TrackingMode       string             `json:"tracking_mode"`
	OpeningCents       *int64             `json:"opening_balance_cents"`
	ClosingCents       *int64             `json:"closing_balance_cents"`
	SumCents           int64              `json:"sum_cents"`
	CreditsCents       int64              `json:"credits_cents"`
	DebitsCents        int64              `json:"debits_cents"`
	TheoreticalCents   *int64             `json:"theoretical_balance_cents"`
	DifferenceCents    *int64             `json:"difference_cents"`
	EffectiveValidated bool               `json:"effective_validated"`
	Status             string             `json:"status"`
	ValidationSource   string             `json:"validation_source"`
	Controls           []statementControl `json:"controls"`
	History            []statementReview  `json:"history"`
	CanValidate        bool               `json:"can_validate"`
}

func centsPointer(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}
func legacyEuro(n *int64) *float64 {
	if n == nil {
		return nil
	}
	v := float64(*n) / 100
	return &v
}
func periodDates(st checkedStatement) (time.Time, time.Time, bool) {
	if st.PeriodStart == nil || st.PeriodEnd == nil {
		return time.Time{}, time.Time{}, false
	}
	a, e1 := time.Parse("2006-01-02", *st.PeriodStart)
	b, e2 := time.Parse("2006-01-02", *st.PeriodEnd)
	return a, b, e1 == nil && e2 == nil && !b.Before(a)
}
func (st *checkedStatement) control(code, status, severity, detail string, related int64) {
	st.Controls = append(st.Controls, statementControl{code, status, severity, detail, related})
}
func finalizeStatementStatus(st *checkedStatement) {
	for _, c := range st.Controls {
		if c.Status == "failed" && c.Severity == "error" {
			st.Status = "failed"
			st.CanValidate = false
			st.EffectiveValidated = false
			return
		}
	}
	// A manual review can acknowledge non-blocking warnings such as similar
	// occurrences or dates outside the statement period. Keep
	// those controls visible, but do not overwrite the owner's recorded status.
	if st.EffectiveValidated && st.ValidationSource == "manual" {
		st.Status = "validated"
	}
}
func assessStatement(st *checkedStatement, outside, similar int) {
	_, _, known := periodDates(*st)
	if known {
		st.control("period", "passed", "info", "Période bancaire identifiée ; elle peut chevaucher deux mois calendaires.", 0)
	} else {
		st.control("period", "unknown", "warning", "Début ou fin de période absent ou incohérent : couverture non établie.", 0)
	}
	if st.DifferenceCents == nil {
		st.control("reconciliation", "unknown", "warning", "Soldes de contrôle absents : rapprochement non exécuté.", 0)
	} else if *st.DifferenceCents != 0 {
		st.control("reconciliation", "failed", "error", "Solde final différent du solde initial augmenté des opérations conservées. Comparer à l’original ; une extraction ou attribution incomplète après déduplication est possible.", 0)
	} else {
		st.control("reconciliation", "passed", "info", "Égalité exacte au centime. Ce contrôle ne prouve pas à lui seul la complétude de l’extraction.", 0)
	}
	if outside > 0 {
		st.control("operation_dates", "suspected", "warning", fmt.Sprintf("%d opération(s) datée(s) hors période bancaire. Vérifier les dates d’opération et de valeur sur l’original ; aucune correction automatique.", outside), 0)
	}
	if similar > 0 {
		st.control("similar_transactions", "informational", "info", fmt.Sprintf("%d opération(s) du relevé partagent date, montant et libellé. Les occurrences sont conservées ; une ressemblance ne confirme pas un doublon.", similar), 0)
	}
	st.CanValidate = known && st.DifferenceCents != nil && *st.DifferenceCents == 0
	if st.AccountID == nil {
		st.control("account", "unknown", "warning", "Compte propriétaire non identifiable : rattachement à vérifier.", 0)
		st.CanValidate = false
	}
	st.EffectiveValidated = st.Validated && st.CanValidate
	st.Status = "analyzed"
	if !st.CanValidate || outside > 0 {
		st.Status = "to_review"
	} else if st.EffectiveValidated {
		st.Status = "validated"
	}
	st.ValidationSource = "none"
	if st.Validated {
		st.ValidationSource = "automatic"
		if st.ImportReport == nil {
			st.ValidationSource = "automatic_legacy"
		}
	}
	if len(st.History) > 0 {
		if st.History[0].Action == "validate" {
			st.ValidationSource = "manual"
		} else {
			st.ValidationSource = "reopened"
		}
	}
	if r := st.ImportReport; r != nil {
		if r.Rejected > 0 || len(r.Issues) > 0 || r.Parsed != st.TxCount {
			st.control("import_candidates", "failed", "error", "Extraction incomplète, lecture ambiguë ou nombre enregistré différent du rapport. Consulter les références sources du rapport d’import.", 0)
		}
		if r.DeclaredDebits != nil && r.DeclaredCredits != nil {
			status, severity := "passed", "info"
			if *r.DeclaredDebits != st.DebitsCents || *r.DeclaredCredits != st.CreditsCents {
				status, severity = "failed", "error"
			}
			st.control("declared_totals", status, severity, "Débits et crédits enregistrés comparés séparément aux totaux indépendants du document.", 0)
		} else {
			st.control("declared_totals", "unknown", "warning", "Totaux indépendants non disponibles : contrôle non exécuté.", 0)
		}
		if r.ExpectedCount == nil {
			st.control("declared_count", "not_applicable", "info", "Nombre attendu non fourni explicitement par le document : ce contrôle ne s’applique pas.", 0)
		} else {
			status, severity := "passed", "info"
			if *r.ExpectedCount != st.TxCount {
				status, severity = "failed", "error"
			}
			st.control("declared_count", status, severity, "Nombre enregistré comparé au nombre explicitement déclaré par le document.", 0)
		}
		if r.TraceabilityControl != "" {
			if r.TraceabilityControl == "failed" {
				st.control("source_traceability", "failed", "error", "Une ou plusieurs opérations n’ont pas de référence de page et de ligne source.", 0)
			} else {
				st.control("source_traceability", "passed", "info", "Chaque opération interprétée est rattachée à sa page et à sa ligne source.", 0)
			}
		}
	}
	finalizeStatementStatus(st)
}
func compareStatementPeriods(statements []checkedStatement) {
	defer func() {
		for i := range statements {
			finalizeStatementStatus(&statements[i])
		}
	}()
	ordered := make([]int, 0, len(statements))
	for i, st := range statements {
		if _, _, ok := periodDates(st); ok && st.AccountID != nil {
			ordered = append(ordered, i)
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := statements[ordered[i]], statements[ordered[j]]
		if *a.AccountID != *b.AccountID {
			return *a.AccountID < *b.AccountID
		}
		if *a.PeriodStart != *b.PeriodStart {
			return *a.PeriodStart < *b.PeriodStart
		}
		return a.ID < b.ID
	})
	for n, i := range ordered {
		st := &statements[i]
		start, _, _ := periodDates(*st)
		for p := n - 1; p >= 0; p-- {
			other := &statements[ordered[p]]
			if *other.AccountID != *st.AccountID {
				break
			}
			a, b, _ := periodDates(*other)
			if !b.After(start) {
				continue
			}
			// A shared single boundary is not an overlapping bank period.
			if !a.After(start) {
				st.control("overlap", "suspected", "warning", "Période recouvrant un autre relevé au-delà d’une borne commune. Vérifier les sources sans supprimer d’écriture.", other.ID)
				other.control("overlap", "suspected", "warning", "Période recouvrant un autre relevé ; une double importation est possible, sans être établie.", st.ID)
				st.Status = "to_review"
				other.Status = "to_review"
			}
		}
		if n == 0 {
			continue
		}
		previous := &statements[ordered[n-1]]
		if *previous.AccountID != *st.AccountID {
			continue
		}
		_, end, _ := periodDates(*previous)
		if start.After(end.AddDate(0, 0, 1)) {
			st.control("continuity", "unknown", "warning", "Intervalle non documenté entre les deux relevés. La continuité des soldes ne peut pas être conclue.", previous.ID)
			continue
		}
		if start.Before(end) {
			continue
		}
		if previous.ClosingCents == nil || st.OpeningCents == nil {
			st.control("balance_continuity", "unknown", "warning", "Un solde manque pour vérifier la continuité entre relevés consécutifs.", previous.ID)
		} else if *previous.ClosingCents != *st.OpeningCents {
			st.control("balance_continuity", "failed", "error", "Le solde d’ouverture diffère du solde final du relevé précédent. Vérifier l’original et les régularisations ; aucun montant modifié.", previous.ID)
			st.Status = "to_review"
			st.CanValidate = false
			st.EffectiveValidated = false
		} else {
			st.control("balance_continuity", "passed", "info", "Solde d’ouverture égal au solde final précédent, au centime.", previous.ID)
		}
	}
}

func loadCheckedStatements(ctx context.Context, reader analysisReader, owner int64) ([]checkedStatement, error) {
	rows, err := reader.QueryContext(ctx, `WITH grouped_operations AS (
 SELECT t.statement_id,count(*) OVER(PARTITION BY t.statement_id,t.tx_date,t.amount,lower(trim(t.label))) n
 FROM transactions t JOIN statements source ON source.id=t.statement_id AND source.owner_id=t.owner_id WHERE t.owner_id=$1
 ), similar_groups AS (SELECT statement_id,count(*) n FROM grouped_operations WHERE n>1 GROUP BY statement_id)
 SELECT st.id,st.filename,st.period_start::text,st.period_end::text,
 (st.opening_balance*100)::bigint,(st.closing_balance*100)::bigint,st.parser,st.validated,st.uploaded_at::text,
 a.id,COALESCE(a.name,'Compte non identifié'),COALESCE(a.tracking_mode,'statements'),
 x.n,x.total,x.credits,x.debits,
 CASE WHEN st.opening_balance IS NOT NULL THEN ((st.opening_balance+x.total/100::numeric)*100)::bigint END,
 CASE WHEN st.opening_balance IS NOT NULL AND st.closing_balance IS NOT NULL THEN ((st.closing_balance-st.opening_balance)*100-x.total)::bigint END,x.outside,
 COALESCE(sim.n,0),(to_jsonb(st)->'import_report')::text
 FROM statements st LEFT JOIN accounts a ON a.id=st.account_id AND a.owner_id=st.owner_id
 LEFT JOIN similar_groups sim ON sim.statement_id=st.id
 CROSS JOIN LATERAL(SELECT count(*) n,COALESCE(SUM(t.amount)*100,0)::bigint total,
 COALESCE(SUM(t.amount) FILTER(WHERE t.amount>0)*100,0)::bigint credits,
 COALESCE(-SUM(t.amount) FILTER(WHERE t.amount<0)*100,0)::bigint debits,
 count(*) FILTER(WHERE t.tx_date<st.period_start OR t.tx_date>st.period_end) outside
 FROM transactions t WHERE t.statement_id=st.id AND t.owner_id=st.owner_id) x
 WHERE st.owner_id=$1 ORDER BY st.period_start DESC NULLS LAST,st.id DESC`, owner)
	if err != nil {
		return nil, err
	}
	out := []checkedStatement{}
	indices := map[int64]int{}
	var counts [][2]int
	for rows.Next() {
		var st checkedStatement
		var opening, closing, theory, diff sql.NullInt64
		var outside, similar int
		var report sql.NullString
		if err = rows.Scan(&st.ID, &st.Filename, &st.PeriodStart, &st.PeriodEnd, &opening, &closing, &st.Parser, &st.Validated, &st.UploadedAt, &st.AccountID, &st.AccountName, &st.TrackingMode, &st.TxCount, &st.SumCents, &st.CreditsCents, &st.DebitsCents, &theory, &diff, &outside, &similar, &report); err != nil {
			rows.Close()
			return nil, err
		}
		if report.Valid {
			if err = json.Unmarshal([]byte(report.String), &st.ImportReport); err != nil {
				rows.Close()
				return nil, fmt.Errorf("rapport d’import invalide")
			}
		}
		st.OpeningCents = centsPointer(opening)
		st.ClosingCents = centsPointer(closing)
		st.TheoreticalCents = centsPointer(theory)
		st.DifferenceCents = centsPointer(diff)
		st.OpeningBalance = legacyEuro(st.OpeningCents)
		st.ClosingBalance = legacyEuro(st.ClosingCents)
		st.Controls = []statementControl{}
		st.History = []statementReview{}
		indices[st.ID] = len(out)
		out = append(out, st)
		counts = append(counts, [2]int{outside, similar})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = reader.QueryContext(ctx, `SELECT resource_id,id,action,note,created_at::text FROM (
 SELECT resource_id,id,action,note,created_at,row_number() OVER(PARTITION BY resource_id ORDER BY id DESC) n
 FROM statement_validation_history WHERE owner_id=$1) h WHERE n<=20 ORDER BY id DESC`, owner)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var review statementReview
		if err = rows.Scan(&id, &review.ID, &review.Action, &review.Note, &review.At); err != nil {
			rows.Close()
			return nil, err
		}
		if i, ok := indices[id]; ok {
			out[i].History = append(out[i].History, review)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range out {
		assessStatement(&out[i], counts[i][0], counts[i][1])
	}
	compareStatementPeriods(out)
	return out, nil
}
func (s *Server) listStatements(w http.ResponseWriter, r *http.Request) {
	tx, err := s.DB.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		writeErr(w, 500, fmt.Errorf("contrôles des relevés indisponibles"))
		return
	}
	defer tx.Rollback()
	out, err := loadCheckedStatements(r.Context(), tx, s.ownerID(r))
	if err != nil {
		writeErr(w, 500, fmt.Errorf("contrôles des relevés indisponibles"))
		return
	}
	if err = tx.Commit(); err != nil {
		writeErr(w, 500, fmt.Errorf("contrôles des relevés indisponibles"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}
func (s *Server) changeStatement(w http.ResponseWriter, r *http.Request, deleting bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, fmt.Errorf("identifiant invalide"))
		return
	}
	action, note := "delete", "Suppression explicitement confirmée par le propriétaire."
	if !deleting {
		var in struct {
			Action string `json:"action"`
			Note   string `json:"note"`
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		d.DisallowUnknownFields()
		if d.Decode(&in) != nil || d.Decode(new(any)) != io.EOF || (in.Action != "validate" && in.Action != "reopen") {
			writeErr(w, 400, fmt.Errorf("action de validation invalide"))
			return
		}
		action, note = in.Action, strings.TrimSpace(in.Note)
		if utf8.RuneCountInString(note) < 10 || utf8.RuneCountInString(note) > 1000 {
			writeErr(w, 400, fmt.Errorf("justification de 10 à 1 000 caractères requise"))
			return
		}
	}
	owner := s.ownerID(r)
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("modification indisponible"))
		return
	}
	defer tx.Rollback()
	// Same owner lock as imports, plus resource lock: consistent sources and transitions.
	if _, err = tx.ExecContext(r.Context(), `SELECT pg_advisory_xact_lock($1)`, owner); err != nil {
		writeErr(w, 500, fmt.Errorf("modification indisponible"))
		return
	}
	var locked int64
	err = tx.QueryRowContext(r.Context(), `SELECT id FROM statements WHERE id=$1 AND owner_id=$2 FOR UPDATE`, id, owner).Scan(&locked)
	if err == sql.ErrNoRows {
		writeErr(w, 404, fmt.Errorf("relevé introuvable"))
		return
	}
	if err != nil {
		writeErr(w, 500, fmt.Errorf("modification indisponible"))
		return
	}
	all, err := loadCheckedStatements(r.Context(), tx, owner)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("contrôles indisponibles"))
		return
	}
	var st checkedStatement
	for _, v := range all {
		if v.ID == id {
			st = v
			break
		}
	}
	if action == "validate" && !st.CanValidate {
		writeErr(w, 409, fmt.Errorf("validation refusée : période, rapprochement ou continuité des soldes à vérifier"))
		return
	}
	// A parsing-only validation may be explicitly attested by the owner once.
	alreadyManual := action == "validate" && st.Validated && st.ValidationSource == "manual"
	if !deleting && (alreadyManual || action == "reopen" && !st.Validated) {
		writeErr(w, 409, fmt.Errorf("ce statut est déjà enregistré"))
		return
	}
	if deleting {
		var foreign bool
		if err = tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM transactions WHERE statement_id=$1 AND owner_id IS DISTINCT FROM $2)`, id, owner).Scan(&foreign); err != nil || foreign {
			writeErr(w, 409, fmt.Errorf("rattachements incohérents : suppression bloquée pour préserver les données"))
			return
		}
		// Serialize statement deletion with allocation creation. The service locks
		// source and target transaction rows before checking allocation capacity.
		lockedTransactions, lockErr := tx.QueryContext(r.Context(), `SELECT id FROM transactions WHERE statement_id=$1 AND owner_id=$2 ORDER BY id FOR UPDATE`, id, owner)
		if lockErr != nil {
			writeErr(w, 500, fmt.Errorf("contrôle des remboursements indisponible"))
			return
		}
		for lockedTransactions.Next() {
			var transactionID int64
			if lockErr = lockedTransactions.Scan(&transactionID); lockErr != nil {
				break
			}
		}
		if lockErr == nil {
			lockErr = lockedTransactions.Err()
		}
		lockedTransactions.Close()
		if lockErr != nil {
			writeErr(w, 500, fmt.Errorf("contrôle des remboursements indisponible"))
			return
		}
		var linked bool
		if err = tx.QueryRowContext(r.Context(), `SELECT
		 EXISTS(SELECT 1 FROM reimbursement_cases c JOIN transactions t ON t.id=c.credit_transaction_id AND t.owner_id=c.owner_id WHERE t.statement_id=$1 AND c.owner_id=$2 AND c.status<>'voided')
		 OR EXISTS(SELECT 1 FROM reimbursement_allocations a JOIN transactions t ON t.id=a.expense_transaction_id AND t.owner_id=a.owner_id WHERE t.statement_id=$1 AND a.owner_id=$2 AND a.voided_at IS NULL)`, id, owner).Scan(&linked); err != nil {
			writeErr(w, 500, fmt.Errorf("contrôle des remboursements indisponible"))
			return
		}
		if linked {
			writeErr(w, 409, fmt.Errorf("suppression bloquée : annule d’abord les rapprochements de remboursement actifs"))
			return
		}
	}
	snapshot, _ := json.Marshal(st)
	if _, err = tx.ExecContext(r.Context(), `INSERT INTO statement_validation_history(statement_id,resource_id,owner_id,actor_id,action,previous_validated,note,snapshot) VALUES($1::integer,$1::bigint,$2,$2,$3,$4,$5,$6)`, id, owner, action, st.Validated, note, snapshot); err != nil {
		writeErr(w, 500, fmt.Errorf("historique non enregistré ; modification annulée"))
		return
	}
	if deleting {
		_, err = tx.ExecContext(r.Context(), `DELETE FROM statements WHERE id=$1 AND owner_id=$2`, id, owner)
	} else {
		_, err = tx.ExecContext(r.Context(), `UPDATE statements SET validated=$3 WHERE id=$1 AND owner_id=$2`, id, owner, action == "validate")
	}
	if err != nil {
		writeErr(w, 500, fmt.Errorf("modification annulée"))
		return
	}
	if _, err = tx.ExecContext(r.Context(), `DELETE FROM analyses WHERE owner_id=$1`, owner); err != nil {
		writeErr(w, 500, fmt.Errorf("actualisation des analyses impossible ; modification annulée"))
		return
	}
	if err = tx.Commit(); err != nil {
		writeErr(w, 500, fmt.Errorf("modification annulée"))
		return
	}
	w.WriteHeader(204)
}
func (s *Server) deleteStatement(w http.ResponseWriter, r *http.Request) {
	s.changeStatement(w, r, true)
}
func (s *Server) validateStatement(w http.ResponseWriter, r *http.Request) {
	s.changeStatement(w, r, false)
}
func (s *Server) updateAccountTracking(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, 400, fmt.Errorf("identifiant invalide"))
		return
	}
	var in struct {
		Mode string `json:"tracking_mode"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(new(any)) != io.EOF || (in.Mode != "manual" && in.Mode != "statements") {
		writeErr(w, 400, fmt.Errorf("mode de suivi invalide"))
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("mode non enregistré"))
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), `UPDATE accounts SET tracking_mode=$3 WHERE id=$1 AND owner_id=$2`, id, s.ownerID(r), in.Mode)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("mode non enregistré"))
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		writeErr(w, 404, fmt.Errorf("compte introuvable"))
		return
	}
	if _, err = tx.ExecContext(r.Context(), `DELETE FROM analyses WHERE owner_id=$1`, s.ownerID(r)); err != nil {
		writeErr(w, 500, fmt.Errorf("actualisation des analyses impossible ; mode inchangé"))
		return
	}
	if err = tx.Commit(); err != nil {
		writeErr(w, 500, fmt.Errorf("mode non enregistré"))
		return
	}
	w.WriteHeader(204)
}
