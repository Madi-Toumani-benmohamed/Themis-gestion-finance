package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"themis/internal/money"
)

type creditState struct {
	ID                     int64
	Name, StartMonth, Note string
	Monthly                money.Cents
	Months                 int
	Opening                sql.NullInt64
	Recorded               money.Cents
	PaymentCount           int
	InconsistentPayments   bool
	Closed, Archived       sql.NullString
}
type creditView struct {
	AmountsCents       map[string]*int64 `json:"amounts_cents"`
	ID                 int64             `json:"id"`
	Name               string            `json:"name"`
	MonthlyAmount      json.Number       `json:"monthly_amount"`
	MonthsTotal        int               `json:"months_total"`
	StartMonth         string            `json:"start_month"`
	Note               string            `json:"note"`
	TotalScheduled     *json.Number      `json:"total_scheduled"`
	OpeningPaid        *json.Number      `json:"opening_paid"`
	PaymentsRecorded   json.Number       `json:"payments_recorded"`
	PaymentCount       int               `json:"payment_count"`
	PaidTotal          *json.Number      `json:"paid_total"`
	Remaining          *json.Number      `json:"remaining"`
	MonthsPaid         *int              `json:"months_paid"`
	MonthsRemaining    *int              `json:"months_remaining"`
	Progress           *json.Number      `json:"progress_pct"`
	ScheduledElapsed   *int              `json:"scheduled_months_elapsed"`
	ScheduledRemaining *int              `json:"scheduled_months_remaining"`
	ScheduledAmount    *json.Number      `json:"scheduled_remaining"`
	EndMonth           string            `json:"end_month"`
	NextMonth          *string           `json:"next_scheduled_month"`
	HistoryComplete    bool              `json:"payment_history_complete"`
	Status             string            `json:"status"`
	Issue              string            `json:"issue,omitempty"`
	ClosedAt           *string           `json:"closed_at"`
	ArchivedAt         *string           `json:"archived_at"`
}

func engagementPtr(c money.Cents) *json.Number { v := engagementNumber(c); return &v }
func engagementString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}
func buildCredit(c creditState, now time.Time) (out creditView) {
	out = creditView{ID: c.ID, Name: c.Name, MonthlyAmount: engagementNumber(c.Monthly), MonthsTotal: c.Months, StartMonth: c.StartMonth, Note: c.Note, PaymentsRecorded: engagementNumber(c.Recorded), PaymentCount: c.PaymentCount, Status: "invalid", ClosedAt: engagementString(c.Closed), ArchivedAt: engagementString(c.Archived)}
	defer func() {
		out.AmountsCents = engagementCentsFields(map[string]*json.Number{"monthly_amount": &out.MonthlyAmount, "total_scheduled": out.TotalScheduled, "opening_paid": out.OpeningPaid, "payments_recorded": &out.PaymentsRecorded, "paid_total": out.PaidTotal, "remaining": out.Remaining, "scheduled_remaining": out.ScheduledAmount})
	}()

	total, err := engagementProduct(c.Monthly, c.Months)
	if err != nil {
		out.Issue = "Échéancier hors limites ; vérifier mensualité et durée."
		return out
	}
	out.TotalScheduled = engagementPtr(total)
	start, err := engagementMonth(c.StartMonth)
	if err != nil {
		out.Issue = "Date de début non documentée ou invalide."
		return out
	}
	end := start.AddDate(0, c.Months-1, 0)
	if end.Year() > 9999 {
		out.Issue = "Date de fin hors limites."
		return out
	}
	out.EndMonth = end.Format("2006-01")
	elapsed := (now.Year()-start.Year())*12 + int(now.Month()-start.Month()) + 1
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > c.Months {
		elapsed = c.Months
	}
	scheduledLeft := c.Months - elapsed
	out.ScheduledElapsed = &elapsed
	out.ScheduledRemaining = &scheduledLeft
	out.ScheduledAmount = engagementPtr(c.Monthly * money.Cents(scheduledLeft))
	if c.InconsistentPayments || c.Recorded < 0 || c.Recorded > total {
		out.Issue = "Paiements incohérents : un contrôle des sources est nécessaire."
		return out
	}
	current := now.Format("2006-01")
	next := current
	if next < c.StartMonth {
		next = c.StartMonth
	}
	if next <= out.EndMonth {
		out.NextMonth = &next
	}
	out.Status = "unknown"
	if c.Opening.Valid {
		opening := money.Cents(c.Opening.Int64)
		if opening < 0 || opening > engagementMaxCents {
			out.Status = "invalid"
			out.Issue = "Montant historique hors limites."
			return out
		}
		out.OpeningPaid = engagementPtr(opening)
		paid, e := engagementAdd(opening, c.Recorded)
		if e != nil || opening < 0 || paid > total {
			out.Issue = "Le total des paiements déclarés dépasse l’échéancier."
			out.Status = "invalid"
			return out
		}
		remaining := total - paid
		out.PaidTotal = engagementPtr(paid)
		out.Remaining = engagementPtr(remaining)
		paidMonths := int(paid / c.Monthly)
		leftMonths := c.Months - paidMonths
		out.MonthsPaid = &paidMonths
		out.MonthsRemaining = &leftMonths
		out.Progress = analysisPercent(paid, total)
		out.HistoryComplete = true
		out.Status = "active"
		if c.StartMonth > current {
			out.Status = "future"
		}
		if remaining == 0 {
			out.Status = "paid"
			out.NextMonth = nil
		}
	}
	if c.Closed.Valid {
		if out.Remaining == nil || string(*out.Remaining) != "0.00" {
			out.Status = "invalid"
			out.Issue = "Clôture sans solde déclaré nul : vérifier l’historique."
			return out
		}
		out.Status = "closed"
		out.NextMonth = nil
	}
	if c.Archived.Valid {
		out.Status = "archived"
		out.NextMonth = nil
	}
	return out
}

type taxState struct {
	ID               int64
	Name, DueDate    string
	Amount           money.Cents
	Paid             bool
	PaidAmount       sql.NullInt64
	OverdueConfirmed bool
	Archived         sql.NullString
}
type taxView struct {
	AmountsCents     map[string]*int64 `json:"amounts_cents"`
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	Amount           json.Number       `json:"amount"`
	DueDate          string            `json:"due_date"`
	Paid             bool              `json:"paid"`
	PaidAmount       json.Number       `json:"paid_amount"`
	Remaining        json.Number       `json:"remaining"`
	PaymentSource    string            `json:"payment_source"`
	DaysLeft         int               `json:"days_left"`
	OverdueConfirmed bool              `json:"overdue_confirmed"`
	Status           string            `json:"status"`
	Issue            string            `json:"issue,omitempty"`
	ArchivedAt       *string           `json:"archived_at"`
}

func buildTax(t taxState, now time.Time) (out taxView, err error) {
	out = taxView{ID: t.ID, Name: t.Name, Amount: engagementNumber(t.Amount), DueDate: t.DueDate, PaymentSource: "declared", ArchivedAt: engagementString(t.Archived), OverdueConfirmed: t.OverdueConfirmed}
	defer func() {
		out.AmountsCents = engagementCentsFields(map[string]*json.Number{"amount": &out.Amount, "paid_amount": &out.PaidAmount, "remaining": &out.Remaining})
	}()

	date, err := engagementDate(t.DueDate)
	if err != nil || t.Amount <= 0 {
		return out, fmt.Errorf("échéance incohérente")
	}
	out.DaysLeft = int((date.Unix() - engagementToday(now).Unix()) / 86400)
	paid := money.Cents(0)
	if t.PaidAmount.Valid {
		paid = money.Cents(t.PaidAmount.Int64)
	} else if t.Paid {
		paid = t.Amount
		out.PaymentSource = "legacy_confirmation"
	}
	if paid < 0 || paid > t.Amount {
		return out, fmt.Errorf("montant payé incohérent")
	}
	out.PaidAmount = engagementNumber(paid)
	out.Remaining = engagementNumber(t.Amount - paid)
	out.Paid = paid == t.Amount
	if t.PaidAmount.Valid && t.Paid != out.Paid {
		out.Issue = "Le statut enregistré et le montant déclaré sont incohérents."
	}
	switch {
	case out.Paid:
		out.Status = "paid"
	case paid > 0:
		out.Status = "partial"
	case out.DaysLeft < 0:
		out.Status = "to_verify"
	case out.DaysLeft == 0:
		out.Status = "today"
	default:
		out.Status = "upcoming"
	}
	if t.OverdueConfirmed && !out.Paid && out.DaysLeft < 0 {
		out.Status = "late_confirmed"
	}
	if t.Archived.Valid {
		out.Status = "archived"
	}
	return out, nil
}

// One shared scope for the report, individual views and locked mutations.
const creditStateColumns = `c.id,c.name,c.monthly_amount::text,c.months_total,btrim(c.start_month),c.note,c.opening_paid_cents,c.closed_at::text,c.archived_at::text,
 COALESCE((SELECT SUM(p.amount_cents) FROM credit_payments p WHERE p.credit_id=c.id AND p.owner_id=c.owner_id AND p.voided_at IS NULL),0)::text,
 (SELECT count(*) FROM credit_payments p WHERE p.credit_id=c.id AND p.owner_id=c.owner_id AND p.voided_at IS NULL),
 EXISTS(SELECT 1 FROM credit_payments p WHERE p.credit_id=c.id AND p.owner_id<>c.owner_id)`

type engagementScanner interface{ Scan(...any) error }

func scanCredit(row engagementScanner) (creditState, error) {
	var c creditState
	var monthly, recorded string
	err := row.Scan(&c.ID, &c.Name, &monthly, &c.Months, &c.StartMonth, &c.Note, &c.Opening, &c.Closed, &c.Archived, &recorded, &c.PaymentCount, &c.InconsistentPayments)
	if err != nil {
		return c, err
	}
	c.Monthly, err = money.Parse(monthly)
	if err != nil {
		return c, err
	}
	// SUM(BIGINT cents) is an integer count of cents, not euros.
	value, err := engagementParseCents(recorded)
	c.Recorded = money.Cents(value)
	return c, err
}

const taxStateColumns = `t.id,t.name,t.amount::text,t.due_date::text,t.paid,t.paid_amount_cents,t.overdue_confirmed,t.archived_at::text`

func scanTax(row engagementScanner) (taxState, error) {
	var t taxState
	var raw string
	err := row.Scan(&t.ID, &t.Name, &raw, &t.DueDate, &t.Paid, &t.PaidAmount, &t.OverdueConfirmed, &t.Archived)
	if err != nil {
		return t, err
	}
	t.Amount, err = money.Parse(raw)
	return t, err
}
