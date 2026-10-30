package api

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"themis/internal/intelligence"
	"themis/internal/money"
)

type behaviorFacts struct {
	id          int64
	month       string
	merchant    string
	categoryKey string
	category    string
	amount      money.Cents
	weekend     bool
}

type behaviorAggregate struct {
	label   string
	amounts map[string]money.Cents
	counts  map[string]int64
	ids     []int64
}

// loadBehaviorDiscoveries evaluates conservative six-month behavioral signals
// from one owner-scoped, repeatable-read snapshot. Incomplete periods produce
// no comparative claims.
func (s *Server) loadBehaviorDiscoveries(ctx context.Context, owner int64, month string, now time.Time) ([]intelligence.Discovery, bool, error) {
	selected, err := time.Parse("2006-01", month)
	if err != nil || selected.Format("2006-01") != month {
		return nil, false, fmt.Errorf("invalid behavioral month")
	}
	first := selected.AddDate(0, -5, 0)
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	months := make([]string, 6)
	for i := range months {
		m := first.AddDate(0, i, 0)
		months[i] = m.Format("2006-01")
		report, e := analysisReportFrom(ctx, tx, owner, months[i], now)
		if e != nil {
			return nil, false, e
		}
		if report.Period.Completeness != "complete" {
			return []intelligence.Discovery{}, false, nil
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT t.id,to_char(t.tx_date,'YYYY-MM'),
 COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label),
	 COALESCE(COALESCE(p.id,d.id)::text,'uncategorized'),
 COALESCE(p.name,d.name,'Sans catégorie'),(-t.amount)::text,
	 EXTRACT(DOW FROM t.tx_date)::int IN (0,6)
 FROM transactions t LEFT JOIN domains d ON d.id=t.domain_id AND d.owner_id=t.owner_id
 LEFT JOIN domains p ON p.id=d.parent_id AND p.owner_id=d.owner_id
 WHERE t.owner_id=$1 AND t.amount<0 AND t.tx_date >= $2::date
 AND t.tx_date < ($3::date + interval '1 month') AND `+financialExpenseSQLT+`
 ORDER BY t.tx_date,t.id`, owner, first.Format("2006-01-02"), selected.Format("2006-01-02"))
	if err != nil {
		return nil, false, err
	}
	facts := []behaviorFacts{}
	for rows.Next() {
		var f behaviorFacts
		var raw string
		if err = rows.Scan(&f.id, &f.month, &f.merchant, &f.categoryKey, &f.category, &raw, &f.weekend); err != nil {
			rows.Close()
			return nil, false, err
		}
		f.amount, err = parseAnalysisAmount(raw)
		if err != nil {
			rows.Close()
			return nil, false, err
		}
		facts = append(facts, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, false, err
	}
	return buildBehaviorDiscoveries(owner, month, months, facts), true, nil
}

func buildBehaviorDiscoveries(owner int64, period string, months []string, facts []behaviorFacts) []intelligence.Discovery {
	merchants, categories := map[string]*behaviorAggregate{}, map[string]*behaviorAggregate{}
	get := func(groups map[string]*behaviorAggregate, key, label, month string) *behaviorAggregate {
		if groups[key] == nil {
			groups[key] = &behaviorAggregate{label: label, amounts: map[string]money.Cents{}, counts: map[string]int64{}}
		}
		return groups[key]
	}
	for _, f := range facts {
		m := get(merchants, f.merchant, f.merchant, f.month)
		m.amounts[f.month] += f.amount
		m.counts[f.month]++
		m.ids = append(m.ids, f.id)
		if f.categoryKey == "" {
			f.categoryKey = "uncategorized"
		}
		c := get(categories, f.categoryKey, f.category, f.month)
		c.amounts[f.month] += f.amount
		if f.weekend {
			c.counts[f.month]++
		}
		c.ids = append(c.ids, f.id)
	}
	out := []intelligence.Discovery{}
	appendDiscovery := func(kind, anchor, title, summary string, metrics []intelligence.Metric, ids []int64, limitations []string, amount money.Cents) {
		if len(ids) > 500 {
			ids = ids[:500]
		}
		impact := recurringImpactScore(amount)
		relevance, _ := intelligence.Score(intelligence.Factors{Impact: impact, Evidence: 90, Novelty: 50, Context: 0, Action: 55}, intelligence.DefaultWeights, len(ids) > 0, "complete")
		source := "behavior_facts_v1"
		if kind == "progressive_trend" && strings.HasPrefix(anchor, "merchant:") {
			source += ":merchant"
		}
		out = append(out, intelligence.Discovery{OwnerID: owner, Kind: kind, SubjectKey: intelligence.SubjectKey(kind, anchor), Period: period, Title: title, Summary: summary, Severity: "information", Confidence: 85, RelevanceScore: relevance, ImpactScore: impact, Metrics: metrics, Evidence: intelligence.Evidence{TransactionIDs: append([]int64(nil), ids...), Periods: append([]string(nil), months...), Source: source}, Coverage: intelligence.Coverage{Status: "complete", Periods: 6, Transactions: len(ids), Description: "Six mois calendaires complets, établis par les relevés validés."}, Limitations: limitations, State: "active"})
	}
	trendLimit := []string{"Six mois d’observation ne permettent pas d’exclure une saisonnalité.", "Une hausse comptable ne démontre pas sa cause."}
	for _, kindGroups := range []struct {
		kind   string
		groups map[string]*behaviorAggregate
	}{{"category", categories}, {"merchant", merchants}} {
		keys := sortedAggregateKeys(kindGroups.groups)
		for _, key := range keys {
			a := kindGroups.groups[key]
			series := make([]intelligence.MonthlyAmount, 6)
			for i, m := range months {
				series[i] = intelligence.MonthlyAmount{Month: m, Amount: a.amounts[m]}
			}
			if trend, ok := intelligence.DetectProgressiveTrend(series); ok {
				metrics := []intelligence.Metric{{Name: "first_period_cents", Kind: intelligence.MetricMoneyCents, Value: int64(trend.FirstCents)}, {Name: "last_period_cents", Kind: intelligence.MetricMoneyCents, Value: int64(trend.LastCents)}, {Name: "change_cents", Kind: intelligence.MetricMoneyCents, Value: int64(trend.DeltaCents)}, {Name: "change_bps", Kind: intelligence.MetricPercent, Value: trend.ChangeBPS}}
				appendDiscovery("progressive_trend", kindGroups.kind+":"+key, "Hausse progressive : "+a.label, fmt.Sprintf("Les débits ordinaires observés augmentent sur six mois consécutifs, de %s € à %s € (%s € d’écart).", trend.FirstCents.String(), trend.LastCents.String(), trend.DeltaCents.String()), metrics, a.ids, trendLimit, trend.DeltaCents)
			}
		}
	}
	baseline, current := months[:3], months[3:]
	rowsForVariation := []intelligence.MonthlySubjectAmount{}
	for key, a := range merchants {
		for _, m := range months {
			rowsForVariation = append(rowsForVariation, intelligence.MonthlySubjectAmount{Month: m, Key: key, Label: a.label, Amount: a.amounts[m]})
		}
	}
	if variation, ok := intelligence.ExplainVariation(rowsForVariation, baseline, current); ok && variation.Delta >= 10000 && (variation.BaselineCents == 0 || variation.Delta*100 >= variation.BaselineCents*15) {
		contrib := []string{}
		var leading money.Cents
		ids := []int64{}
		for _, item := range variation.Contributions {
			if item.DeltaCents != 0 {
				ids = append(ids, merchants[item.Key].ids...)
			}
			if item.DeltaCents > 0 && len(contrib) < 3 {
				contrib = append(contrib, fmt.Sprintf("%s (+%s €)", item.Label, item.DeltaCents.String()))
				leading += item.DeltaCents
			}
		}
		if len(contrib) > 0 {
			metrics := []intelligence.Metric{{Name: "baseline_three_months_cents", Kind: intelligence.MetricMoneyCents, Value: int64(variation.BaselineCents)}, {Name: "current_three_months_cents", Kind: intelligence.MetricMoneyCents, Value: int64(variation.CurrentCents)}, {Name: "observed_change_cents", Kind: intelligence.MetricMoneyCents, Value: int64(variation.Delta)}, {Name: "top_contributors_cents", Kind: intelligence.MetricMoneyCents, Value: int64(leading)}, {Name: "other_contributors_cents", Kind: intelligence.MetricMoneyCents, Value: int64(variation.Delta - leading)}}
			appendDiscovery("spend_variation_contributors", "ordinary-spend", "Principaux contributeurs à la variation", fmt.Sprintf("Les débits ordinaires totalisent %s € sur %s–%s contre %s € sur %s–%s. Principales contributions comptables : %s. Cela ne démontre pas une cause.", variation.CurrentCents.String(), current[0], current[2], variation.BaselineCents.String(), baseline[0], baseline[2], joinLabels(contrib)), metrics, ids, []string{"La contribution comptable n’établit pas une cause comportementale.", "Les six mois ne permettent pas d’exclure une saisonnalité."}, variation.Delta)
		}
	}
	for key, a := range categories {
		previous, currentCounts := make([]intelligence.MonthlyCount, 3), make([]intelligence.MonthlyCount, 3)
		for i := 0; i < 3; i++ {
			previous[i] = intelligence.MonthlyCount{Month: months[i], Count: a.counts[months[i]]}
			currentCounts[i] = intelligence.MonthlyCount{Month: months[i+3], Count: a.counts[months[i+3]]}
		}
		if change, ok := intelligence.DetectFrequencyChange(previous, currentCounts); ok {
			metrics := []intelligence.Metric{{Name: "previous_weekend_transactions", Kind: intelligence.MetricCount, Value: change.PreviousCount}, {Name: "current_weekend_transactions", Kind: intelligence.MetricCount, Value: change.CurrentCount}, {Name: "additional_weekend_transactions", Kind: intelligence.MetricCount, Value: change.Delta}}
			appendDiscovery("weekend_frequency_change", key, "Achats du week-end plus fréquents : "+a.label, fmt.Sprintf("%d débits ordinaires le week-end sur %s–%s, contre %d sur les trois mois précédents. Il s’agit d’une fréquence observée, pas d’un abonnement.", change.CurrentCount, months[3], months[5], change.PreviousCount), metrics, a.ids, []string{"Le décompte n’établit pas une récurrence contractuelle."}, 0)
		}
	}
	for key, a := range merchants {
		previous, currentCounts := make([]intelligence.MonthlyCount, 3), make([]intelligence.MonthlyCount, 3)
		for i := 0; i < 3; i++ {
			previous[i] = intelligence.MonthlyCount{Month: months[i], Count: a.counts[months[i]]}
			currentCounts[i] = intelligence.MonthlyCount{Month: months[i+3], Count: a.counts[months[i+3]]}
		}
		if change, ok := intelligence.DetectFrequencyChange(previous, currentCounts); ok {
			metrics := []intelligence.Metric{{Name: "previous_transactions", Kind: intelligence.MetricCount, Value: change.PreviousCount}, {Name: "current_transactions", Kind: intelligence.MetricCount, Value: change.CurrentCount}, {Name: "additional_transactions", Kind: intelligence.MetricCount, Value: change.Delta}}
			appendDiscovery("purchase_frequency_change", key, "Achats plus fréquents chez "+a.label, fmt.Sprintf("%d débits ordinaires observés chez ce marchand sur %s–%s, contre %d sur les trois mois précédents. Cette fréquence ne signifie pas un abonnement.", change.CurrentCount, months[3], months[5], change.PreviousCount), metrics, a.ids, []string{"La fréquence observée ne démontre pas un abonnement ni une intention d’achat."}, 0)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].SubjectKey < out[j].SubjectKey
	})
	return out
}

// A six-month merchant trend subsumes the older four-month merchant drift
// signal for the same merchant, preventing duplicate feed cards.
func removeSubsumedMerchantDrifts(signals []map[string]any, discoveries []intelligence.Discovery) []map[string]any {
	trending := map[string]bool{}
	const prefix = "Hausse progressive : "
	for _, d := range discoveries {
		if d.Kind == "progressive_trend" && d.Evidence.Source == "behavior_facts_v1:merchant" && strings.HasPrefix(d.Title, prefix) {
			trending[strings.TrimPrefix(d.Title, prefix)] = true
		}
	}
	out := make([]map[string]any, 0, len(signals))
	for _, signal := range signals {
		if signal["kind"] == "merchant_drift" {
			if source, ok := signal["source"].(map[string]any); ok {
				if merchant, ok := source["merchant"].(string); ok && trending[merchant] {
					continue
				}
			}
		}
		out = append(out, signal)
	}
	return out
}

func sortedAggregateKeys(groups map[string]*behaviorAggregate) []string {
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func joinLabels(values []string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}
