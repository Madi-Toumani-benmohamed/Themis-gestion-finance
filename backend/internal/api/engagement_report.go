package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"themis/internal/money"
)

type engagementReport struct {
	AmountsCents          map[string]*int64 `json:"amounts_cents"`
	AsOf                  string            `json:"as_of"`
	Source                string            `json:"source"`
	Credits               []creditView      `json:"credits"`
	Taxes                 []taxView         `json:"taxes"`
	CreditsRemaining      *json.Number      `json:"credits_remaining"`
	KnownCreditsRemaining json.Number       `json:"known_credits_remaining"`
	UnknownCredits        int               `json:"unknown_credits_count"`
	InvalidCredits        int               `json:"invalid_credits_count"`
	MonthlyLoad           *json.Number      `json:"monthly_load"`
	TaxesDue              json.Number       `json:"taxes_due"`
	TaxesSoon             int               `json:"taxes_due_soon"`
	TaxesToVerify         int               `json:"taxes_to_verify"`
	TotalDebt             *json.Number      `json:"total_debt"`
	KnownTotal            json.Number       `json:"known_total_debt"`
}

func readCreditStates(ctx context.Context, reader analysisReader, owner int64, archived bool) ([]creditState, error) {
	rows, err := reader.QueryContext(ctx, `SELECT `+creditStateColumns+` FROM credits c WHERE c.owner_id=$1 AND ($2 OR c.archived_at IS NULL) ORDER BY c.id`, owner, archived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []creditState{}
	for rows.Next() {
		c, e := scanCredit(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func readTaxStates(ctx context.Context, reader analysisReader, owner int64, archived bool) ([]taxState, error) {
	rows, err := reader.QueryContext(ctx, `SELECT `+taxStateColumns+` FROM taxes t WHERE t.owner_id=$1 AND ($2 OR t.archived_at IS NULL) ORDER BY t.paid,t.due_date,t.id`, owner, archived)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []taxState{}
	for rows.Next() {
		t, e := scanTax(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (s *Server) loadEngagementReport(ctx context.Context, owner int64, now time.Time) (engagementReport, error) {
	out := engagementReport{AsOf: engagementToday(now).Format("2006-01-02"), Source: "user_declarations", Credits: []creditView{}, Taxes: []taxView{}}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	credits, err := readCreditStates(ctx, tx, owner, false)
	if err != nil {
		return out, err
	}
	taxes, err := readTaxStates(ctx, tx, owner, false)
	if err != nil {
		return out, err
	}
	var known, monthly, taxesDue money.Cents
	monthlyKnown := true
	for _, c := range credits {
		v := buildCredit(c, now)
		out.Credits = append(out.Credits, v)
		if v.Status == "invalid" {
			out.InvalidCredits++
			monthlyKnown = false
		}
		if v.Remaining == nil {
			out.UnknownCredits++
		} else {
			remaining, e := engagementAmount(string(*v.Remaining), false)
			if e != nil {
				return out, e
			}
			known, err = engagementAdd(known, remaining)
			if err != nil {
				return out, err
			}
		}
		if v.Status != "closed" && v.Status != "paid" && v.Status != "invalid" && c.StartMonth <= now.Format("2006-01") && v.EndMonth >= now.Format("2006-01") {
			monthly, err = engagementAdd(monthly, c.Monthly)
			if err != nil {
				return out, err
			}
		}
	}
	for _, t := range taxes {
		v, e := buildTax(t, now)
		if e != nil {
			return out, e
		}
		if v.Issue != "" {
			return out, fmt.Errorf("statut fiscal incohérent")
		}
		out.Taxes = append(out.Taxes, v)
		remaining, e := engagementAmount(string(v.Remaining), false)
		if e != nil {
			return out, e
		}
		taxesDue, err = engagementAdd(taxesDue, remaining)
		if err != nil {
			return out, err
		}
		if !v.Paid && v.DaysLeft >= 0 && v.DaysLeft <= 30 {
			out.TaxesSoon++
		}
		if !v.Paid && v.DaysLeft < 0 && !v.OverdueConfirmed {
			out.TaxesToVerify++
		}
	}
	out.KnownCreditsRemaining = engagementNumber(known)
	out.TaxesDue = engagementNumber(taxesDue)
	total, err := engagementAdd(known, taxesDue)
	if err != nil {
		return out, err
	}
	out.KnownTotal = engagementNumber(total)
	if out.UnknownCredits == 0 && out.InvalidCredits == 0 {
		out.CreditsRemaining = engagementPtr(known)
		out.TotalDebt = engagementPtr(total)
	}
	if monthlyKnown {
		out.MonthlyLoad = engagementPtr(monthly)
	}
	out.AmountsCents = engagementCentsFields(map[string]*json.Number{"credits_remaining": out.CreditsRemaining, "known_credits_remaining": &out.KnownCreditsRemaining, "monthly_load": out.MonthlyLoad, "taxes_due": &out.TaxesDue, "total_debt": out.TotalDebt, "known_total_debt": &out.KnownTotal})
	return out, tx.Commit()
}
func (s *Server) obligations(w http.ResponseWriter, r *http.Request) {
	out, err := s.loadEngagementReport(r.Context(), s.ownerID(r), time.Now())
	if err != nil {
		writeErr(w, 500, fmt.Errorf("engagements indisponibles ou incohérents"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}
func (s *Server) listCredits(w http.ResponseWriter, r *http.Request) {
	states, err := readCreditStates(r.Context(), s.DB, s.ownerID(r), r.URL.Query().Get("include_archived") == "1")
	if err != nil {
		writeErr(w, 500, fmt.Errorf("crédits indisponibles"))
		return
	}
	now := time.Now()
	out := []creditView{}
	for _, c := range states {
		out = append(out, buildCredit(c, now))
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}
func (s *Server) listTaxes(w http.ResponseWriter, r *http.Request) {
	states, err := readTaxStates(r.Context(), s.DB, s.ownerID(r), r.URL.Query().Get("include_archived") == "1")
	if err != nil {
		writeErr(w, 500, fmt.Errorf("échéances indisponibles"))
		return
	}
	now := time.Now()
	out := []taxView{}
	for _, t := range states {
		v, e := buildTax(t, now)
		if e != nil {
			writeErr(w, 500, fmt.Errorf("échéance incohérente"))
			return
		}
		out = append(out, v)
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, out)
}
