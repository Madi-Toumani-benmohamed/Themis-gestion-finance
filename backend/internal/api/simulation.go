package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"themis/internal/money"
)

type simulationCut struct {
	Category string      `json:"category"`
	Percent  json.Number `json:"reduce_pct"`
}
type simulationExpense struct {
	Label  string      `json:"label"`
	Amount json.Number `json:"amount"`
}
type simulationLoan struct {
	Amount json.Number `json:"amount"`
	Rate   json.Number `json:"rate"`
	Months int         `json:"months"`
}
type simulationInput struct {
	Income   *json.Number        `json:"base_income"`
	Cuts     []simulationCut     `json:"cuts"`
	Expenses []simulationExpense `json:"new_expenses"`
	Loan     *simulationLoan     `json:"loan"`
}
type simulationCategory struct {
	Category string      `json:"category"`
	Before   json.Number `json:"before"`
	After    json.Number `json:"after"`
	Percent  json.Number `json:"reduce_pct"`
}
type simulationBase struct {
	Name   string
	Amount money.Cents
}

func simulationNumber(c money.Cents) json.Number { return json.Number(c.String()) }

// Arrondi au centime le plus proche, demi-centime à l'écart de zéro.
func simulationRound(r *big.Rat) money.Cents {
	n := new(big.Int).Set(r.Num())
	negative := n.Sign() < 0
	n.Abs(n)
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(n, r.Denom(), rem)
	if new(big.Int).Lsh(rem, 1).Cmp(r.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if negative {
		q.Neg(q)
	}
	return money.Cents(q.Int64())
}
func simulationRatio(raw json.Number, max int64) (*big.Rat, error) {
	if raw == "" {
		raw = "0"
	}
	value, err := money.Parse(string(raw))
	if err != nil || value < 0 || value > money.Cents(max*100) {
		return nil, fmt.Errorf("pourcentage invalide : 0 à %d, deux décimales maximum", max)
	}
	return big.NewRat(int64(value), 100), nil
}
func simulationAmount(n json.Number) (money.Cents, error) {
	c, e := money.Parse(string(n))
	if e != nil || c < 0 {
		return 0, fmt.Errorf("montant positif ou nul, au centime, attendu")
	}
	return c, nil
}
func calculateSimulation(in simulationInput, observed money.Cents, baseline []simulationBase) (map[string]any, error) {
	income := observed
	var err error
	if in.Income != nil {
		income, err = simulationAmount(*in.Income)
		if err != nil {
			return nil, err
		}
	}
	cuts := map[string]json.Number{}
	known := map[string]bool{}
	for _, b := range baseline {
		known[b.Name] = true
	}
	for _, c := range in.Cuts {
		if !known[c.Category] {
			return nil, fmt.Errorf("catégorie indisponible dans la référence")
		}
		if _, exists := cuts[c.Category]; exists {
			return nil, fmt.Errorf("catégorie répétée dans le scénario")
		}
		if _, err := simulationRatio(c.Percent, 100); err != nil {
			return nil, err
		}
		cuts[c.Category] = c.Percent
	}
	categories := []simulationCategory{}
	var baseSpend, saved, added, payment, total, principal money.Cents
	for _, b := range baseline {
		pct := cuts[b.Name]
		if pct == "" {
			pct = "0"
		}
		ratio, _ := simulationRatio(pct, 100)
		saving := simulationRound(new(big.Rat).Mul(big.NewRat(int64(b.Amount), 100), ratio))
		baseSpend += b.Amount
		saved += saving
		categories = append(categories, simulationCategory{b.Name, simulationNumber(b.Amount), simulationNumber(b.Amount - saving), pct})
	}
	for _, e := range in.Expenses {
		if strings.TrimSpace(e.Label) == "" {
			return nil, fmt.Errorf("intitulé de dépense requis")
		}
		c, err := simulationAmount(e.Amount)
		if err != nil {
			return nil, err
		}
		added += c
	}
	if in.Loan != nil {
		principal, err = simulationAmount(in.Loan.Amount)
		if err != nil {
			return nil, err
		}
		rate, err := simulationRatio(in.Loan.Rate, 100)
		if err != nil {
			return nil, err
		}
		if in.Loan.Months < 1 || in.Loan.Months > 600 {
			return nil, fmt.Errorf("durée de crédit : 1 à 600 mois")
		}
		if rate.Sign() == 0 {
			payment = simulationRound(big.NewRat(int64(principal), int64(in.Loan.Months)))
		} else {
			monthly := new(big.Rat).Quo(rate, big.NewRat(1200, 1))
			factor := new(big.Rat).Add(big.NewRat(1, 1), monthly)
			exponent := big.NewInt(int64(in.Loan.Months))
			power := new(big.Rat).SetFrac(new(big.Int).Exp(factor.Num(), exponent, nil), new(big.Int).Exp(factor.Denom(), exponent, nil))
			numerator := new(big.Rat).Mul(big.NewRat(int64(principal), 1), monthly)
			numerator.Mul(numerator, power)
			payment = simulationRound(numerator.Quo(numerator, new(big.Rat).Sub(power, big.NewRat(1, 1))))
		}
		total = payment * money.Cents(in.Loan.Months)
		if rate.Sign() == 0 || total < principal {
			total = principal
		}
	}
	current := observed - baseSpend
	spend := baseSpend - saved + added + payment
	net := income - spend
	return map[string]any{
		"income": simulationNumber(income), "observed_income": simulationNumber(observed), "base_spend": simulationNumber(baseSpend),
		"current_net": simulationNumber(current), "simulated_spend": simulationNumber(spend), "simulated_net": simulationNumber(net),
		"monthly_saved": simulationNumber(saved), "new_expenses": simulationNumber(added), "loan_payment": simulationNumber(payment),
		"loan_total_cost": simulationNumber(total), "loan_interest": simulationNumber(total - principal), "delta": simulationNumber(net - current), "categories": categories,
	}, nil
}

// Trois mois civils consécutifs, terminés, ancrés au dernier mois importé.
// Les catégories et les crédits partagent le même diviseur, y compris mois vides.
func simulationMonths(last, now time.Time) (time.Time, time.Time) {
	end := time.Date(last.Year(), last.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	current := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if end.After(current) {
		end = current
	}
	return end.AddDate(0, -3, 0), end
}
func (s *Server) simulate(w http.ResponseWriter, r *http.Request) {
	var in simulationInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
	if err := decoder.Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("scénario JSON invalide"))
		return
	}
	owner := s.ownerID(r)
	var last string
	err := s.DB.QueryRowContext(r.Context(), `SELECT COALESCE(max(d)::text,'') FROM (
 SELECT tx_date AS d FROM transactions WHERE owner_id=$1 AND tx_date<=CURRENT_DATE
 UNION ALL SELECT period_end AS d FROM statements WHERE owner_id=$1 AND period_end<=CURRENT_DATE) sources`, owner).Scan(&last)
	if err != nil {
		writeErr(w, 500, fmt.Errorf("référence financière indisponible"))
		return
	}
	if last == "" {
		writeJSON(w, 200, map[string]any{"available": false, "reason": "Importe des relevés pour établir une référence."})
		return
	}
	date, err := time.Parse("2006-01-02", last)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	first, end := simulationMonths(date, time.Now())
	periods := []periodInfo{}
	documented := true
	for month := first; month.Before(end); month = month.AddDate(0, 1, 0) {
		info, err := s.getPeriodInfo(owner, month.Format("2006-01"))
		if err != nil {
			writeErr(w, 500, fmt.Errorf("couverture de référence indisponible"))
			return
		}
		periods = append(periods, info)
		if info.Completeness != "complete" {
			documented = false
		}
	}
	var incomeText string
	var count int
	err = s.DB.QueryRowContext(r.Context(), `SELECT COALESCE(SUM(amount) FILTER (WHERE amount>0),0)::text,COUNT(*) FROM transactions WHERE owner_id=$1 AND tx_date >= $2 AND tx_date < $3`+notTransfer, owner, first, end).Scan(&incomeText, &count)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	if count == 0 && !documented {
		writeJSON(w, 200, map[string]any{"available": false, "reason": "Aucune référence exploitable sur les trois mois terminés. Les opérations du mois en cours sont exclues.", "reference_periods": periods})
		return
	}
	incomeSum, err := money.Parse(incomeText)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	observed := simulationRound(big.NewRat(int64(incomeSum), 3))
	rows, err := s.DB.QueryContext(r.Context(), `SELECT COALESCE(d.name,'Sans domaine'),SUM(-t.amount)::text FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id WHERE t.owner_id=$1 AND t.amount<0 AND t.tx_date >= $2 AND t.tx_date < $3`+notTransferT+` GROUP BY 1 ORDER BY SUM(-t.amount) DESC,1`, owner, first, end)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	baseline := []simulationBase{}
	for rows.Next() {
		var name, amount string
		if err = rows.Scan(&name, &amount); err != nil {
			break
		}
		var total money.Cents
		total, err = money.Parse(amount)
		if err != nil {
			break
		}
		baseline = append(baseline, simulationBase{name, simulationRound(big.NewRat(int64(total), 3))})
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	result, err := calculateSimulation(in, observed, baseline)
	if err != nil {
		writeErr(w, 400, err)
		return
	}
	result["available"] = true
	result["reference_periods"] = periods
	result["reference_complete"] = documented
	result["rounding"] = "Moyennes et économies arrondies par catégorie au centime ; totaux calculés à partir de ces montants."
	writeJSON(w, 200, result)
}
