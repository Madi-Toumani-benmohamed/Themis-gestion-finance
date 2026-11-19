package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"themis/internal/money"
)

// All read tools use this scope, so selecting a category, merchant, tag or
// amount cannot disappear when moving from source rows to their exact totals.
const agentReadWhereSQL = ` WHERE t.owner_id=$1 AND t.tx_date BETWEEN $2::date AND $3::date AND ` + financialExpenseSQLT + `
 AND (COALESCE(cardinality($4::bigint[]),0)=0 OR t.id=ANY($4))
 AND ($5='' OR t.label ILIKE $5 OR t.merchant ILIKE $5 OR t.display_label ILIKE $5)
 AND ($6::bigint IS NULL OR t.domain_id=$6)
 AND ($7='' OR EXISTS(SELECT 1 FROM transaction_tags tt JOIN tags g ON g.id=tt.tag_id AND g.owner_id=t.owner_id WHERE tt.transaction_id=t.id AND g.name=$7))
 AND ($8::numeric IS NULL OR abs(t.amount)>=$8::numeric)
 AND ($9::numeric IS NULL OR abs(t.amount)<=$9::numeric)
 AND ($10='all' OR ($10='expense' AND t.amount<0) OR ($10='income' AND t.amount>0)) `

type agentReadOptions struct {
	DomainName string      `json:"-"`
	Start      string      `json:"start_date"`
	End        string      `json:"end_date"`
	Merchant   string      `json:"merchant"`
	DomainID   *int64      `json:"domain_id"`
	Tag        string      `json:"tag"`
	Min        json.Number `json:"min_amount"`
	Max        json.Number `json:"max_amount"`
	Direction  string      `json:"direction"`
	Group      string      `json:"group_by"`
	Sort       string      `json:"sort"`
	Limit      *int        `json:"limit"`
}

func agentReadArguments(args map[string]any, mode string) (agentReadOptions, error) {
	var o agentReadOptions
	data, e := json.Marshal(args)
	if e != nil {
		return o, fmt.Errorf("arguments de lecture invalides")
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&o) != nil {
		return o, fmt.Errorf("filtre de lecture invalide ou non pris en charge ; consulter la définition de l’outil")
	}
	if o.Direction == "" {
		o.Direction = "all"
		if mode == "categories" {
			o.Direction = "expense"
		}
	}
	if o.Direction != "all" && o.Direction != "income" && o.Direction != "expense" {
		return o, fmt.Errorf("direction requise : all, income ou expense")
	}
	if o.Sort != "" && o.Sort != "date" && o.Sort != "amount" {
		return o, fmt.Errorf("tri requis : date ou amount")
	}
	if mode == "comparison" && (o.Start != "" || o.End != "") {
		return o, fmt.Errorf("utiliser start_date_a/end_date_a et start_date_b/end_date_b pour comparer")
	}
	if mode != "search" && (o.Sort != "" || o.Limit != nil) {
		return o, fmt.Errorf("sort et limit concernent la recherche des opérations, pas le total complet")
	}
	if o.Limit != nil && (*o.Limit < 1 || *o.Limit > 50) {
		return o, fmt.Errorf("limite de recherche : 1 à 50 opérations")
	}
	if o.Group != "" && o.Group != "none" && o.Group != "category" && o.Group != "merchant" {
		return o, fmt.Errorf("regroupement invalide")
	}
	if mode != "summary" && o.Group != "" && o.Group != "none" {
		return o, fmt.Errorf("group_by concerne la synthèse financière")
	}
	if mode == "categories" && o.Direction == "all" {
		return o, fmt.Errorf("choisir expense ou income pour une ventilation")
	}
	if len([]rune(o.Merchant)) > 200 {
		return o, fmt.Errorf("critère marchand trop long")
	}
	if o.DomainID != nil && *o.DomainID <= 0 {
		return o, fmt.Errorf("identifiant de catégorie invalide")
	}
	if _, present := args["tag"]; present {
		tags, e := validateTransactionTags([]string{o.Tag})
		if e != nil {
			return o, e
		}
		o.Tag = tags[0]
	}
	var bounds [2]money.Cents
	for i, v := range []json.Number{o.Min, o.Max} {
		if v == "" {
			continue
		}
		n, e := money.Parse(string(v))
		if e != nil || n < 0 {
			return o, fmt.Errorf("montants de filtre positifs, au centime près, requis")
		}
		bounds[i] = n
		if i == 0 {
			o.Min = json.Number(n.String())
		} else {
			o.Max = json.Number(n.String())
		}
	}
	if o.Min != "" && o.Max != "" && bounds[0] > bounds[1] {
		return o, fmt.Errorf("bornes de montant inversées")
	}
	return o, nil
}
func agentReadDomain(ctx context.Context, db *sql.DB, owner int64, o *agentReadOptions) error {
	if o.DomainID == nil {
		return nil
	}
	if err := db.QueryRowContext(ctx, `SELECT name FROM domains WHERE id=$1 AND owner_id=$2`, *o.DomainID, owner).Scan(&o.DomainName); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("catégorie indisponible pour ce compte")
		}
		return fmt.Errorf("catégories indisponibles")
	}

	return nil
}
func (o agentReadOptions) params(owner int64, p agentPeriod, ids []int64) []any {
	pattern := ""
	if o.Merchant != "" {
		pattern = "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(o.Merchant) + "%"
	}
	var domain, min, max any
	if o.DomainID != nil {
		domain = *o.DomainID
	}
	if o.Min != "" {
		min = string(o.Min)
	}
	if o.Max != "" {
		max = string(o.Max)
	}
	return []any{owner, p.From, p.To, pq.Array(ids), pattern, domain, o.Tag, min, max, o.Direction}
}
func (o agentReadOptions) metadata() map[string]any {
	out := map[string]any{"direction": o.Direction}
	if o.Merchant != "" {
		out["merchant"] = o.Merchant
	}
	if o.DomainID != nil {
		out["domain_id"] = *o.DomainID
		out["category"] = o.DomainName
	}
	if o.Tag != "" {
		out["tag"] = o.Tag
	}
	if o.Min != "" {
		out["min_amount"] = string(o.Min)
	}
	if o.Max != "" {
		out["max_amount"] = string(o.Max)
	}
	return out
}
func agentScopeIDs(scope [][]int64) []int64 {
	if len(scope) > 0 {
		return scope[0]
	}
	return nil
}
