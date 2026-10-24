package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"themis/internal/money"
	"themis/internal/normalize"
)

// A review is metadata about a deterministic detection, never a mutation of a
// transaction and never proof of the completeness of the underlying imports.
type recurringReviewReport struct {
	Month            string                `json:"month"`
	ObservationStart string                `json:"observation_start"`
	ObservationEnd   string                `json:"observation_end"`
	Period           periodInfo            `json:"period"`
	Items            []recurringReviewItem `json:"items"`
}
type recurringReviewItem struct {
	Key                   string        `json:"key"`
	Label                 string        `json:"label"`
	Merchant              string        `json:"merchant"`
	AccountID             int64         `json:"account_id"`
	DomainID              *int64        `json:"domain_id"`
	MerchantKind          string        `json:"merchant_kind,omitempty"`
	Direction             string        `json:"direction"`
	Nature                string        `json:"nature"`
	DetectedNature        string        `json:"detected_nature"`
	Status                string        `json:"status"`
	Confidence            string        `json:"confidence"`
	ConfidenceReason      string        `json:"confidence_reason"`
	OccurrenceCount       int           `json:"occurrence_count"`
	MonthsCount           int           `json:"months_count"`
	Months                []string      `json:"months"`
	FirstDate             string        `json:"first_date"`
	LastDate              string        `json:"last_date"`
	LastAmount            json.Number   `json:"last_amount"`
	AveragePerTransaction json.Number   `json:"average_per_transaction"`
	AverageDenominator    int           `json:"average_denominator"`
	TypicalAmount         json.Number   `json:"typical_amount"`
	Frequency             *string       `json:"frequency"`
	EstimatedNextDate     *string       `json:"estimated_next_date"`
	CoverageStart         string        `json:"coverage_start"`
	CoverageEnd           string        `json:"coverage_end"`
	CoverageComplete      bool          `json:"coverage_complete"`
	TransactionIDs        []int64       `json:"transaction_ids"`
	ReviewedAt            *string       `json:"reviewed_at"`
	SourceAmounts         []money.Cents `json:"-"`
	protected             bool
}
type recurringReviewSource struct {
	ID, AccountID                          int64
	DomainID                               *int64
	Date                                   time.Time
	Label, Merchant, Display, Kind, Domain string
	Amount                                 money.Cents // signed source value; directions must never be netted
	IsTransfer                             bool
}
type recurringReviewDecision struct{ Status, Nature, UpdatedAt string }

var recurringCashPattern = regexp.MustCompile(`(?i)(\bDAB\b|\bDISTRIBUTEUR\b|\bATM\b|\bRETRAIT\b|\bRET\s+DAB\b)`)
var recurringReviewKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func recurringReviewAmount(c money.Cents) json.Number { return json.Number(c.String()) }
func recurringReviewRound(n money.Cents, denominator int) money.Cents {
	d := money.Cents(denominator)
	return (n + d/2) / d
}
func recurringReviewIdentity(t recurringReviewSource) (key, merchant, kind string, transfer, cash bool) {
	sign := -1.0
	if t.Amount > 0 {
		sign = 1
	}
	normalized := normalize.Label(t.Label, sign)
	kind = t.Kind
	if kind == "" {
		kind = normalized.Kind
	}
	merchant = strings.TrimSpace(t.Merchant)
	if merchant == "" {
		merchant = normalized.Merchant
	}
	merchant = strings.Join(strings.Fields(merchant), " ")
	if merchant == "" {
		merchant = strings.TrimSpace(t.Label)
	}
	transfer = t.IsTransfer || kind == "savings" || normalized.Kind == "savings"
	cash = recurringCashPattern.MatchString(t.Label) || recurringCashPattern.MatchString(t.Display) || recurringCashPattern.MatchString(merchant)
	scopeKind := kind
	if transfer {
		scopeKind = "transfer_savings"
	}
	if cash && !transfer {
		scopeKind = "cash_withdrawal"
	}
	domain := int64(0)
	if t.DomainID != nil {
		domain = *t.DomainID
	}
	direction := "debit"
	if t.Amount > 0 {
		direction = "credit"
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("v1|%d|%d|%s|%s|%s", t.AccountID, domain, scopeKind, direction, strings.ToUpper(merchant))))
	return hex.EncodeToString(digest[:]), merchant, kind, transfer, cash
}
func recurringEconomicNature(t recurringReviewSource, merchant string, transfer, cash bool) string {
	if transfer {
		return "transfer_savings"
	}
	if cash {
		return "variable_recurring"
	}
	name := strings.ToUpper(strings.TrimSpace(merchant))
	domain := strings.ToUpper(t.Domain)
	// Merchant identity/category are indications of economic nature, not proof of
	// a contract. Confidence and cadence are evaluated separately below.
	subscriptions := map[string]bool{"NETFLIX": true, "SPOTIFY": true, "DEEZER": true, "DISNEY+": true, "AMAZON PRIME": true, "CANAL+": true, "BASIC-FIT": true}
	if subscriptions[name] || strings.Contains(domain, "ABONNEMENT") {
		return "subscription"
	}
	fixed := map[string]bool{"EDF": true, "ENGIE": true, "VEOLIA": true, "URSSAF": true, "DGFIP": true, "MAIF": true, "MACIF": true, "MATMUT": true, "HARMONIE MUTUELLE": true, "DIRECT ASSURANCE": true}
	if fixed[name] || t.Kind == "tax" || strings.Contains(domain, "LOGEMENT") || strings.Contains(domain, "ASSURANCE") || strings.Contains(domain, "ÉNERGIE") || strings.Contains(domain, "ENERGIE") {
		return "fixed_charge"
	}
	return "variable_recurring"
}
func recurringCalendarNext(last time.Time, months int) time.Time {
	first := time.Date(last.Year(), last.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, months, 0)
	day := last.Day()
	if max := first.AddDate(0, 1, -1).Day(); day > max {
		day = max
	}
	return first.AddDate(0, 0, day-1)
}
func recurringCadence(dates []time.Time) *string {
	if len(dates) < 3 {
		return nil
	}
	weekly, monthly, quarterly := true, true, true
	for i := 1; i < len(dates); i++ {
		previous, current := dates[i-1], dates[i]
		days := int(current.Sub(previous).Hours() / 24)
		weekly = weekly && days >= 6 && days <= 8
		months := (current.Year()-previous.Year())*12 + int(current.Month()-previous.Month())
		dayDelta := current.Day() - previous.Day()
		if dayDelta < 0 {
			dayDelta = -dayDelta
		}
		previousEnd := previous.AddDate(0, 1, -previous.Day()).Day()
		currentEnd := current.AddDate(0, 1, -current.Day()).Day()
		nearEnd := previous.Day() >= previousEnd-2 && current.Day() >= currentEnd-2
		monthly = monthly && months == 1 && (dayDelta <= 5 || nearEnd)
		quarterly = quarterly && months == 3 && (dayDelta <= 5 || nearEnd)
	}
	name := ""
	if weekly {
		name = "weekly"
	} else if monthly {
		name = "monthly"
	} else if quarterly {
		name = "quarterly"
	}
	if name == "" {
		return nil
	}
	return &name
}
func buildRecurringReview(sources []recurringReviewSource, spans map[int64][]heroDateSpan, end time.Time) []recurringReviewItem {
	type group struct {
		sources        []recurringReviewSource
		merchant, kind string
		transfer, cash bool
	}
	groups := map[string]*group{}
	ordered := append([]recurringReviewSource(nil), sources...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Date.Equal(ordered[j].Date) {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Date.Before(ordered[j].Date)
	})
	for _, t := range ordered {
		if t.Amount == 0 || t.Date.After(end) {
			continue
		}
		key, merchant, kind, transfer, cash := recurringReviewIdentity(t)
		if t.Amount > 0 && !transfer {
			continue
		} // refunds and income are not spending subscriptions
		g := groups[key]
		if g == nil {
			g = &group{merchant: merchant, kind: kind, transfer: transfer, cash: cash}
			groups[key] = g
		}
		g.sources = append(g.sources, t)
	}
	out := []recurringReviewItem{}
	for key, g := range groups {
		dates := []time.Time{}
		monthSet := map[string]bool{}
		distinctDates := map[string]bool{}
		amounts := []money.Cents{}
		ids := []int64{}
		var sum money.Cents
		for _, t := range g.sources {
			amount := t.Amount
			if amount < 0 {
				amount = -amount
			}
			sum += amount
			amounts = append(amounts, amount)
			ids = append(ids, t.ID)
			dates = append(dates, t.Date)
			distinctDates[t.Date.Format("2006-01-02")] = true
			monthSet[t.Date.Format("2006-01")] = true
		}
		if len(amounts) < 2 || len(distinctDates) < 2 {
			continue
		}
		first, last := g.sources[0], g.sources[len(g.sources)-1]
		ranked := append([]money.Cents(nil), amounts...)
		sort.Slice(ranked, func(i, j int) bool { return ranked[i] < ranked[j] })
		median := ranked[len(ranked)/2]
		if len(ranked)%2 == 0 {
			median = recurringReviewRound(ranked[len(ranked)/2-1]+ranked[len(ranked)/2], 2)
		}
		months := make([]string, 0, len(monthSet))
		for month := range monthSet {
			months = append(months, month)
		}
		sort.Strings(months)
		label := last.Display
		if label == "" {
			label = g.merchant
		}
		if label == "" {
			label = last.Label
		}
		nature := recurringEconomicNature(last, g.merchant, g.transfer, g.cash)
		complete := first.AccountID != 0 && heroSpansCover(spans[first.AccountID], first.Date, end)
		confidence := "insufficient"
		reason := "La cadence nécessite au moins trois observations espacées de façon cohérente."
		frequency := recurringCadence(dates)
		if frequency != nil {
			if !complete {
				reason = "La cadence est compatible avec une récurrence, mais la couverture documentaire de l'intervalle n'est pas établie."
			} else {
				confidence = "probable"
				reason = "Au moins trois observations suivent une cadence cohérente dans un intervalle documenté."
			}
		}
		// Stability is explained, but alone never establishes an economic contract.
		stable := ranked[len(ranked)-1] > 0 && (ranked[len(ranked)-1]-ranked[0])*100 <= ranked[len(ranked)-1]*15
		if nature == "subscription" && !stable {
			reason += " Les montants sont variables ; la nature économique détectée ne prouve pas l'existence d'un contrat."
		}
		direction := "debit"
		if last.Amount > 0 {
			direction = "credit"
		}
		item := recurringReviewItem{Key: key, Label: label, Merchant: g.merchant, AccountID: last.AccountID, DomainID: last.DomainID, MerchantKind: g.kind, Direction: direction, Nature: nature, DetectedNature: nature, Status: "detected", Confidence: confidence, ConfidenceReason: reason, OccurrenceCount: len(amounts), MonthsCount: len(months), Months: months, FirstDate: first.Date.Format("2006-01-02"), LastDate: last.Date.Format("2006-01-02"), LastAmount: recurringReviewAmount(amounts[len(amounts)-1]), AveragePerTransaction: recurringReviewAmount(recurringReviewRound(sum, len(amounts))), AverageDenominator: len(amounts), TypicalAmount: recurringReviewAmount(median), Frequency: frequency, CoverageStart: first.Date.Format("2006-01-02"), CoverageEnd: end.Format("2006-01-02"), CoverageComplete: complete, TransactionIDs: ids, SourceAmounts: amounts, protected: g.transfer || g.cash}
		if frequency != nil && confidence == "probable" && nature != "variable_recurring" {
			next := last.Date.AddDate(0, 0, 7)
			if *frequency == "monthly" {
				next = recurringCalendarNext(last.Date, 1)
			} else if *frequency == "quarterly" {
				next = recurringCalendarNext(last.Date, 3)
			}
			if next.After(end) {
				date := next.Format("2006-01-02")
				item.EstimatedNextDate = &date
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Nature != out[j].Nature {
			return out[i].Nature < out[j].Nature
		}
		if out[i].Label != out[j].Label {
			return out[i].Label < out[j].Label
		}
		return out[i].Key < out[j].Key
	})
	return out
}
func applyRecurringReview(item *recurringReviewItem, decision recurringReviewDecision) {
	item.Status = decision.Status
	if decision.Nature != "" && !(item.protected && decision.Nature != item.DetectedNature) {
		item.Nature = decision.Nature
	}
	if decision.UpdatedAt != "" {
		item.ReviewedAt = &decision.UpdatedAt
	}
	// Confirmation is a user assertion, never a promotion of data quality.
	if item.Nature == "variable_recurring" || item.Nature == "transfer_savings" {
		item.EstimatedNextDate = nil
	}
}

func (s *Server) loadRecurringReview(ctx context.Context, owner int64, month string, now time.Time) (recurringReviewReport, error) {
	first, end, _, err := reliabilityWindow(month, now)
	report := recurringReviewReport{Month: first.Format("2006-01"), Items: []recurringReviewItem{}}
	if err != nil {
		return report, err
	}
	start := first.AddDate(0, -11, 0)
	report.ObservationStart = start.Format("2006-01-02")
	report.ObservationEnd = end.Format("2006-01-02")
	report.Period, err = s.getPeriodInfo(owner, report.Month)
	if err != nil {
		return report, err
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return report, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT t.id,COALESCE(st.account_id,0),t.tx_date::text,t.label,t.amount::text,t.domain_id,COALESCE(t.merchant,''),COALESCE(t.display_label,''),COALESCE(t.label_kind,''),concat_ws(' ',d.name,p.name),(COALESCE(d.is_transfer,false) OR COALESCE(p.is_transfer,false))
 FROM transactions t LEFT JOIN statements st ON st.id=t.statement_id AND st.owner_id=t.owner_id
 LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
 LEFT JOIN domains p ON p.id=d.parent_id AND p.owner_id=t.owner_id
 WHERE t.owner_id=$1 AND t.amount<>0 AND t.tx_date >= $2::date AND t.tx_date <= $3::date ORDER BY t.tx_date,t.id`, owner, report.ObservationStart, report.ObservationEnd)
	if err != nil {
		return report, err
	}
	sources := []recurringReviewSource{}
	for rows.Next() {
		var t recurringReviewSource
		var date, amount string
		if err = rows.Scan(&t.ID, &t.AccountID, &date, &t.Label, &amount, &t.DomainID, &t.Merchant, &t.Display, &t.Kind, &t.Domain, &t.IsTransfer); err != nil {
			break
		}
		t.Date, err = time.Parse("2006-01-02", date)
		if err != nil {
			break
		}
		t.Amount, err = money.Parse(amount)
		if err != nil {
			break
		}
		sources = append(sources, t)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return report, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT COALESCE(account_id,0),period_start::text,period_end::text FROM statements WHERE owner_id=$1 AND validated=true AND period_start <= $3::date AND period_end >= $2::date ORDER BY period_start`, owner, report.ObservationStart, report.ObservationEnd)
	if err != nil {
		return report, err
	}
	spans := map[int64][]heroDateSpan{}
	for rows.Next() {
		var account int64
		var from, to string
		if err = rows.Scan(&account, &from, &to); err != nil {
			break
		}
		a, e1 := time.Parse("2006-01-02", from)
		b, e2 := time.Parse("2006-01-02", to)
		if e1 == nil && e2 == nil && !b.Before(a) {
			spans[account] = append(spans[account], heroDateSpan{a, b})
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return report, err
	}
	report.Items = buildRecurringReview(sources, spans, end)
	rows, err = tx.QueryContext(ctx, `SELECT recurrence_key,status,COALESCE(nature,''),updated_at::text FROM recurring_reviews WHERE owner_id=$1`, owner)
	if err != nil {
		return report, err
	}
	decisions := map[string]recurringReviewDecision{}
	for rows.Next() {
		var key string
		var d recurringReviewDecision
		if err = rows.Scan(&key, &d.Status, &d.Nature, &d.UpdatedAt); err != nil {
			break
		}
		decisions[key] = d
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return report, err
	}
	for i := range report.Items {
		if d, ok := decisions[report.Items[i].Key]; ok {
			applyRecurringReview(&report.Items[i], d)
		}
	}
	return report, tx.Commit()
}
func (s *Server) listRecurringReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	month := r.URL.Query().Get("month")
	if _, _, _, err := reliabilityWindow(month, time.Now()); err != nil {
		writeErr(w, 400, err)
		return
	}
	result, err := s.loadRecurringReview(r.Context(), s.ownerID(r), month, time.Now())
	if err != nil {
		writeErr(w, 500, fmt.Errorf("les détections ne peuvent pas être chargées"))
		return
	}
	writeJSON(w, 200, result)
}
func (s *Server) updateRecurringReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	key := r.PathValue("key")
	if !recurringReviewKeyPattern.MatchString(key) {
		writeErr(w, 400, fmt.Errorf("clé de détection invalide"))
		return
	}
	var in struct {
		Status string `json:"status"`
		Nature string `json:"nature"`
		Month  string `json:"month"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	if err := decoder.Decode(&in); err != nil {
		writeErr(w, 400, fmt.Errorf("décision invalide"))
		return
	}
	if in.Status != "detected" && in.Status != "confirmed" && in.Status != "ignored" {
		writeErr(w, 400, fmt.Errorf("statut invalide"))
		return
	}
	validNature := map[string]bool{"subscription": true, "fixed_charge": true, "variable_recurring": true, "transfer_savings": true}
	if in.Nature != "" && !validNature[in.Nature] {
		writeErr(w, 400, fmt.Errorf("nature économique invalide"))
		return
	}
	month := r.URL.Query().Get("month")
	if month == "" {
		month = in.Month
	}
	if _, _, _, err := reliabilityWindow(month, time.Now()); err != nil {
		writeErr(w, 400, err)
		return
	}
	owner := s.ownerID(r)
	report, err := s.loadRecurringReview(r.Context(), owner, month, time.Now())
	if err != nil {
		writeErr(w, 500, fmt.Errorf("la détection ne peut pas être vérifiée"))
		return
	}
	var found *recurringReviewItem
	for i := range report.Items {
		if report.Items[i].Key == key {
			found = &report.Items[i]
			break
		}
	}
	if found == nil {
		writeErr(w, 404, fmt.Errorf("détection introuvable dans les sources de cette période"))
		return
	}
	if found.protected && in.Nature != "" && in.Nature != found.DetectedNature {
		writeErr(w, 400, fmt.Errorf("un transfert, mouvement d'épargne ou retrait ne peut pas devenir un abonnement par une correction de détection"))
		return
	}
	_, err = s.DB.ExecContext(r.Context(), `INSERT INTO recurring_reviews(owner_id,recurrence_key,status,nature) VALUES($1,$2,$3,NULLIF($4,'')) ON CONFLICT(owner_id,recurrence_key) DO UPDATE SET status=EXCLUDED.status,nature=EXCLUDED.nature,updated_at=now()`, owner, key, in.Status, in.Nature)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("la décision ne peut pas être enregistrée"))
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
