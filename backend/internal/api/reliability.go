package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"time"
)

type reliabilityGap struct {
	From string `json:"from"`
	To   string `json:"to"`
}
type reliabilityStatement struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	From        string `json:"from"`
	To          string `json:"to"`
	Validated   bool   `json:"validated"`
	PeriodKnown bool   `json:"period_known"`
}
type reliabilityAccount struct {
	ID              int64                  `json:"id"`
	Name            string                 `json:"name"`
	LastImport      string                 `json:"last_import"`
	LastTransaction string                 `json:"last_transaction"`
	Status          string                 `json:"status"`
	Gaps            []reliabilityGap       `json:"gaps"`
	Statements      []reliabilityStatement `json:"statements"`
	ToReview        int                    `json:"to_review"`
	UnknownPeriods  int                    `json:"unknown_periods"`
}
type reliabilityReport struct {
	Month    string               `json:"month"`
	Current  bool                 `json:"current"`
	Through  string               `json:"through"`
	Accounts []reliabilityAccount `json:"accounts"`
}

// Un seul calcul d'intervalles pour la synthèse et le centre de fiabilité.
// Les trous indiquent l'absence de preuve documentaire, pas de transactions.
func uncoveredStatementDates(spans []heroDateSpan, first, last time.Time) []reliabilityGap {
	out := []reliabilityGap{}
	if last.Before(first) {
		return out
	}
	ordered := append([]heroDateSpan(nil), spans...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].start.Before(ordered[j].start) })
	next := first
	for _, span := range ordered {
		if span.end.Before(span.start) || span.end.Before(next) || span.start.After(last) {
			continue
		}
		if span.start.After(next) {
			out = append(out, reliabilityGap{next.Format("2006-01-02"), span.start.AddDate(0, 0, -1).Format("2006-01-02")})
		}
		next = span.end.AddDate(0, 0, 1)
		if next.After(last) {
			return out
		}
	}
	if !next.After(last) {
		out = append(out, reliabilityGap{next.Format("2006-01-02"), last.Format("2006-01-02")})
	}
	return out
}

func reliabilityWindow(month string, now time.Time) (first, last time.Time, current bool, err error) {
	if month == "" {
		month = now.Format("2006-01")
	}
	first, err = time.Parse("2006-01", month)
	if err != nil || first.Year() < 1 || first.Format("2006-01") != month {
		return first, last, false, fmt.Errorf("mois invalide : format YYYY-MM attendu")
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if first.After(today) {
		return first, last, false, fmt.Errorf("les mois futurs ne peuvent pas être vérifiés")
	}
	current = month == now.Format("2006-01")
	last = first.AddDate(0, 1, -1)
	if current {
		last = today
	}
	return
}

func assessReliabilityAccount(a *reliabilityAccount, first, last time.Time) {
	spans := []heroDateSpan{}
	a.ToReview = 0
	a.UnknownPeriods = 0
	for i := range a.Statements {
		st := &a.Statements[i]
		from, e1 := time.Parse("2006-01-02", st.From)
		to, e2 := time.Parse("2006-01-02", st.To)
		st.PeriodKnown = e1 == nil && e2 == nil && !to.Before(from)
		if !st.Validated || !st.PeriodKnown {
			a.ToReview++
		}
		if !st.PeriodKnown {
			a.UnknownPeriods++
			continue
		}
		if st.Validated {
			spans = append(spans, heroDateSpan{from, to})
		}
	}
	a.Gaps = uncoveredStatementDates(spans, first, last)
	switch {
	case len(a.Gaps) == 0:
		a.Status = "documented"
	case a.ToReview > 0 || a.UnknownPeriods > 0:
		a.Status = "to_review"
	case len(a.Statements) == 0:
		a.Status = "no_statement"
	default:
		a.Status = "insufficient"
	}
}

func (s *Server) loadReliability(ctx context.Context, owner int64, month string, now time.Time) (reliabilityReport, error) {
	return loadReliabilityFrom(ctx, s.DB, owner, month, now)
}

// A report and its documentary coverage can share one read-only snapshot.
type analysisReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func loadReliabilityFrom(ctx context.Context, reader analysisReader, owner int64, month string, now time.Time) (reliabilityReport, error) {
	first, last, current, err := reliabilityWindow(month, now)
	if err != nil {
		return reliabilityReport{}, err
	}
	return loadReliabilityRangeFrom(ctx, reader, owner, first, last, current)
}

func loadReliabilityRangeFrom(ctx context.Context, reader analysisReader, owner int64, first, last time.Time, current bool) (reliabilityReport, error) {
	report := reliabilityReport{Month: first.Format("2006-01"), Current: current, Through: last.Format("2006-01-02"), Accounts: []reliabilityAccount{}}
	// Inclure tous les comptes connus comme dans heroMonthCovered. Les relevés
	// dont le compte n'est pas identifiable restent visibles sans nom externe.
	rows, err := reader.QueryContext(ctx, `WITH scope AS (
 SELECT id,name FROM accounts WHERE owner_id=$1
 UNION SELECT DISTINCT COALESCE(st.account_id,0),'Compte non identifié'
 FROM statements st WHERE st.owner_id=$1 AND NOT EXISTS (
 SELECT 1 FROM accounts a WHERE a.owner_id=$1 AND a.id=st.account_id)
 ) SELECT c.id,c.name,
 COALESCE((SELECT max(st.uploaded_at)::date::text FROM statements st WHERE st.owner_id=$1 AND COALESCE(st.account_id,0)=c.id),''),
 COALESCE((SELECT max(t.tx_date)::text FROM transactions t JOIN statements st ON st.id=t.statement_id AND st.owner_id=$1
 WHERE t.owner_id=$1 AND COALESCE(st.account_id,0)=c.id AND t.tx_date BETWEEN $2::date AND $3::date),'')
 FROM scope c ORDER BY c.id`, owner, first.Format("2006-01-02"), last.Format("2006-01-02"))
	if err != nil {
		return report, err
	}
	indices := map[int64]int{}
	for rows.Next() {
		a := reliabilityAccount{Statements: []reliabilityStatement{}}
		if err = rows.Scan(&a.ID, &a.Name, &a.LastImport, &a.LastTransaction); err != nil {
			rows.Close()
			return report, err
		}
		indices[a.ID] = len(report.Accounts)
		report.Accounts = append(report.Accounts, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	rows, err = reader.QueryContext(ctx, `SELECT id,COALESCE(account_id,0),filename,COALESCE(period_start::text,''),COALESCE(period_end::text,''),validated
 FROM statements WHERE owner_id=$1 AND (
 period_start IS NULL OR period_end IS NULL OR period_end<period_start OR
 (period_start<=$3::date AND period_end>=$2::date)) ORDER BY period_start NULLS LAST,id`, owner, first.Format("2006-01-02"), last.Format("2006-01-02"))
	if err != nil {
		return report, err
	}
	defer rows.Close()
	for rows.Next() {
		var accountID int64
		var st reliabilityStatement
		if err = rows.Scan(&st.ID, &accountID, &st.Filename, &st.From, &st.To, &st.Validated); err != nil {
			return report, err
		}
		if i, ok := indices[accountID]; ok {
			report.Accounts[i].Statements = append(report.Accounts[i].Statements, st)
		}
	}
	if err = rows.Err(); err != nil {
		return report, err
	}
	for i := range report.Accounts {
		assessReliabilityAccount(&report.Accounts[i], first, last)
	}
	return report, nil
}

func (s *Server) reliability(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	now := time.Now()
	if _, _, _, err := reliabilityWindow(month, now); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	report, err := s.loadReliability(r.Context(), s.ownerID(r), month, now)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("impossible de vérifier la couverture des relevés"))
		return
	}
	writeJSON(w, http.StatusOK, report)
}
