package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"time"

	"themis/internal/money"
)

type analysisCategory struct {
	ID    *int64       `json:"id"`
	Name  string       `json:"name"`
	Color string       `json:"color"`
	Spent json.Number  `json:"spent"`
	Count int          `json:"transaction_count"`
	Share *json.Number `json:"share_pct"`
}
type analysisReport struct {
	Available          bool               `json:"available"`
	Month              string             `json:"month"`
	From               string             `json:"query_from"`
	To                 string             `json:"query_to"`
	Period             periodInfo         `json:"period"`
	Coverage           reliabilityReport  `json:"coverage"`
	Income             json.Number        `json:"income"`
	Expenses           json.Number        `json:"expenses"`
	Net                json.Number        `json:"net"`
	RealSpend          json.Number        `json:"real_spend"`
	Mandatory          json.Number        `json:"mandatory"`
	Savings            json.Number        `json:"savings"`
	TransferOut        json.Number        `json:"transfer_out"`
	TransferIn         json.Number        `json:"transfer_in"`
	TotalOut           json.Number        `json:"total_out"`
	Count              int                `json:"transaction_count"`
	UncategorizedCount int                `json:"uncategorized_count"`
	UncategorizedSpent json.Number        `json:"uncategorized_spent"`
	UnattributedCount  int                `json:"unattributed_count"`
	Categories         []analysisCategory `json:"categories"`
}

// The JSON/React boundary also needs exact integer cents. Reject an aggregate
// beyond the shared presentation limit instead of silently losing precision.
const analysisMaxCents money.Cents = 9007199254740991

func parseAnalysisAmount(raw string) (money.Cents, error) {
	amount, err := parseTransactionAggregate(raw)
	if err != nil || amount > analysisMaxCents || amount < -analysisMaxCents {
		return 0, fmt.Errorf("agrégat monétaire hors limites exactes")
	}
	return amount, nil
}
func validateAnalysisAmounts(values ...money.Cents) error {
	for _, amount := range values {
		if amount > analysisMaxCents || amount < -analysisMaxCents {
			return fmt.Errorf("total monétaire hors limites exactes")
		}
	}
	return nil
}

func analysisPercent(n, d money.Cents) *json.Number {
	if d <= 0 {
		return nil
	}
	// Percentage with two decimal places, integer/rational rounding only.
	value := new(big.Rat).Mul(big.NewRat(int64(n), int64(d)), big.NewRat(10000, 1))
	if value.Cmp(big.NewRat(int64(analysisMaxCents), 1)) > 0 || value.Cmp(big.NewRat(-int64(analysisMaxCents), 1)) < 0 {
		return nil
	}
	result := simulationNumber(simulationRound(value))
	return &result
}
func analysisMean(n money.Cents, count int) json.Number {
	return simulationNumber(simulationRound(big.NewRat(int64(n), int64(count))))
}
func (s *Server) analysisMonth(ctx context.Context, owner int64, month string, now time.Time) (string, error) {
	if month == "" {
		var latest sql.NullString
		if err := s.DB.QueryRowContext(ctx, `SELECT to_char(max(tx_date),'YYYY-MM') FROM transactions WHERE owner_id=$1 AND tx_date<=$2::date`, owner, now.Format("2006-01-02")).Scan(&latest); err != nil {
			return "", err
		}
		month = latest.String
		if month == "" {
			month = now.Format("2006-01")
		}
	}
	_, _, _, err := reliabilityWindow(month, now)
	return month, err
}
func (s *Server) loadAnalysisReport(ctx context.Context, owner int64, month string, now time.Time) (analysisReport, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return analysisReport{}, err
	}
	defer tx.Rollback()
	report, err := analysisReportFrom(ctx, tx, owner, month, now)
	if err != nil {
		return report, err
	}
	return report, tx.Commit()
}
func analysisReportFrom(ctx context.Context, reader analysisReader, owner int64, month string, now time.Time) (analysisReport, error) {
	first, last, current, err := reliabilityWindow(month, now)
	out := analysisReport{Month: month, From: first.Format("2006-01-02"), To: last.Format("2006-01-02"), Categories: []analysisCategory{}}
	if err != nil {
		return out, err
	}
	out.Coverage, err = loadReliabilityFrom(ctx, reader, owner, month, now)
	if err != nil {
		return out, err
	}
	const query = `WITH source AS (
 SELECT t.*,` + financialExpenseSQLT + ` AS ordinary,
 EXISTS(SELECT 1 FROM statements st JOIN accounts a ON a.id=st.account_id AND a.owner_id=st.owner_id
 WHERE st.id=t.statement_id AND st.owner_id=t.owner_id) AS attributed
 FROM transactions t WHERE t.owner_id=$1 AND t.tx_date BETWEEN $2::date AND $3::date)
 SELECT count(*),COALESCE(max(tx_date)::text,''),count(*) FILTER(WHERE NOT attributed),
 COALESCE(SUM(amount) FILTER(WHERE amount>0 AND ordinary),0)::text,
 COALESCE(-SUM(amount) FILTER(WHERE amount<0 AND ordinary AND COALESCE(label_kind,'')<>'tax'),0)::text,
 COALESCE(-SUM(amount) FILTER(WHERE amount<0 AND ordinary AND label_kind='tax'),0)::text,
 COALESCE(-SUM(amount) FILTER(WHERE amount<0 AND label_kind='savings'),0)::text,
 COALESCE(-SUM(amount) FILTER(WHERE amount<0 AND NOT ordinary AND COALESCE(label_kind,'')<>'savings'),0)::text,
 COALESCE(SUM(amount) FILTER(WHERE amount>0 AND NOT ordinary),0)::text FROM source`
	var lastTx string
	raw := make([]string, 6)
	err = reader.QueryRowContext(ctx, query, owner, out.From, out.To).Scan(&out.Count, &lastTx, &out.UnattributedCount, &raw[0], &raw[1], &raw[2], &raw[3], &raw[4], &raw[5])
	if err != nil {
		return out, err
	}
	amounts := make([]money.Cents, 6)
	for i, v := range raw {
		amounts[i], err = parseAnalysisAmount(v)
		if err != nil {
			return out, err
		}
	}
	income, real, tax, savings, transferOut, transferIn := amounts[0], amounts[1], amounts[2], amounts[3], amounts[4], amounts[5]
	expenses := real + tax
	if err = validateAnalysisAmounts(expenses, income-expenses, expenses+savings+transferOut); err != nil {
		return out, err
	}
	out.Income = simulationNumber(income)
	out.Expenses = simulationNumber(expenses)
	out.Net = simulationNumber(income - expenses)
	out.RealSpend = simulationNumber(real)
	out.Mandatory = simulationNumber(tax)
	out.Savings = simulationNumber(savings)
	out.TransferOut = simulationNumber(transferOut)
	out.TransferIn = simulationNumber(transferIn)
	out.TotalOut = simulationNumber(expenses + savings + transferOut)
	out.UncategorizedSpent = "0.00"
	out.Period = periodFromReliability(out.Coverage, lastTx, first.AddDate(0, 1, -1).Day(), current, now.Day())
	if out.UnattributedCount > 0 && out.Period.Completeness == "complete" {
		out.Period.Completeness = "partial"
		out.Period.Status = "incomplete"
	}
	out.Available = out.Count > 0 || out.Period.Completeness == "complete"
	rows, err := reader.QueryContext(ctx, `SELECT COALESCE(p.id,d.id),COALESCE(p.name,d.name,'Sans catégorie'),COALESCE(p.color,d.color,'#777777'),(-SUM(t.amount))::text,count(*)
 FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
 LEFT JOIN domains p ON p.id=d.parent_id AND p.owner_id=t.owner_id
 WHERE t.owner_id=$1 AND t.tx_date BETWEEN $2::date AND $3::date AND t.amount<0`+notTransferT+`
 GROUP BY COALESCE(p.id,d.id),COALESCE(p.name,d.name,'Sans catégorie'),COALESCE(p.color,d.color,'#777777')
 ORDER BY -SUM(t.amount) DESC,COALESCE(p.id,d.id) NULLS LAST`, owner, out.From, out.To)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	var categoryTotal money.Cents
	for rows.Next() {
		var c analysisCategory
		var id sql.NullInt64
		var value string
		if err = rows.Scan(&id, &c.Name, &c.Color, &value, &c.Count); err != nil {
			return out, err
		}
		amount, e := parseAnalysisAmount(value)
		if e != nil {
			return out, e
		}
		categoryTotal += amount
		if id.Valid {
			v := id.Int64
			c.ID = &v
		} else {
			out.UncategorizedCount = c.Count
			out.UncategorizedSpent = simulationNumber(amount)
		}
		c.Spent = simulationNumber(amount)
		c.Share = analysisPercent(amount, expenses)
		out.Categories = append(out.Categories, c)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if categoryTotal != expenses {
		return out, fmt.Errorf("répartition analytique non réconciliée")
	}
	return out, nil
}

// Shared denominator, then largest-remainder distribution. The sum of rounded
// category means equals the rounded total mean; ties follow stable category IDs.
func analysisApportion(values []money.Cents, denominator int) []money.Cents {
	out := make([]money.Cents, len(values))
	type remainder struct {
		index int
		value money.Cents
	}
	rest := []remainder{}
	var total, allocated money.Cents
	for i, v := range values {
		total += v
		out[i] = v / money.Cents(denominator)
		allocated += out[i]
		rest = append(rest, remainder{i, v % money.Cents(denominator)})
	}
	target := simulationRound(big.NewRat(int64(total), int64(denominator)))
	sort.SliceStable(rest, func(i, j int) bool { return rest[i].value > rest[j].value })
	for i := money.Cents(0); i < target-allocated; i++ {
		out[rest[int(i)].index]++
	}
	return out
}
