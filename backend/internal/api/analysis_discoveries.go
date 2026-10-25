package api

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"themis/internal/money"
)

func analysisSignalStart(month string) string {
	first, _ := time.Parse("2006-01", month)
	return first.AddDate(0, -35, 0).Format("2006-01-02")
}

// Recurrence facts describe observed occurrences, never an inferred monthly bill.
func analysisRecurringSignals(report recurringReviewReport) []map[string]any {
	out := []map[string]any{}
	segments := map[string][]recurringReviewItem{}
	for _, item := range report.Items {
		if item.Status == "ignored" || item.Direction != "debit" {
			continue
		}
		source := map[string]any{"ids": item.TransactionIDs, "from": report.ObservationStart, "to": report.ObservationEnd}
		out = append(out, map[string]any{"kind": "recurring", "title": item.Label, "detail": fmt.Sprintf("%d occurrences observées ; médiane par opération : %s €. %s", item.OccurrenceCount, item.TypicalAmount, item.ConfidenceReason), "status": item.Status, "nature": item.Nature, "confidence": item.Confidence, "source": source})
		if item.Nature != "subscription" {
			continue
		}
		// Compare the last observed charge with the PREVIOUS occurrences, excluding
		// the current observation from its own reference. No tariff change is asserted.
		if n := len(item.SourceAmounts); n >= 3 {
			previous := append([]money.Cents(nil), item.SourceAmounts[:n-1]...)
			sort.Slice(previous, func(i, j int) bool { return previous[i] < previous[j] })
			median := previous[len(previous)/2]
			last := item.SourceAmounts[n-1]
			if len(previous)%2 == 0 {
				median = recurringReviewRound(previous[len(previous)/2-1]+median, 2)
			}
			if median > 0 && last-median >= 150 && (last-median)*100 > median*15 {
				out = append(out, map[string]any{"kind": "price_creep", "title": "Montant à vérifier : " + item.Label, "detail": fmt.Sprintf("Dernier débit observé : %s € ; médiane des %d occurrences précédentes : %s €. Cet écart ne prouve pas un changement de tarif.", simulationNumber(last), len(previous), simulationNumber(median)), "source": source})
			}
		}
		for segment, members := range serviceSegments {
			for _, member := range members {
				if strings.Contains(strings.ToLower(item.Merchant), strings.ToLower(member)) {
					segments[segment] = append(segments[segment], item)
					break
				}
			}
		}
	}
	keys := []string{}
	for key := range segments {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		names := map[string]bool{}
		ids := []int64{}
		for _, item := range segments[key] {
			names[strings.ToLower(item.Merchant)] = true
			ids = append(ids, item.TransactionIDs...)
		}
		if len(names) < 2 {
			continue
		}
		labels := []string{}
		for name := range names {
			labels = append(labels, name)
		}
		sort.Strings(labels)
		out = append(out, map[string]any{"kind": "duplicate_subs", "title": "Services potentiellement redondants : " + key, "detail": "Plusieurs services observés : " + strings.Join(labels, ", ") + ". Cela ne prouve ni des contrats actifs ni des écritures en doublon ; vérifier leur utilité et les dernières occurrences.", "source": map[string]any{"ids": ids, "from": report.ObservationStart, "to": report.ObservationEnd}})
	}
	return out
}

// Retains merchant/tag/annual investigations, with bounded dates and a shared
// documentary gate before comparing monthly totals. SQL errors remain errors.
func (s *Server) loadAnalysisSignals(ctx context.Context, owner int64, month string, now time.Time) ([]map[string]any, []string, error) {
	first, last, _, err := reliabilityWindow(month, now)
	if err != nil {
		return nil, nil, err
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	out := []map[string]any{}
	limits := []string{"Les projets détectés par tag restent consultables dans Pilotage → Projets par tag. Les regroupements ne prouvent pas la fin d’un projet."}
	comparable := true
	months := []string{}
	for i := 3; i >= 0; i-- {
		months = append(months, first.AddDate(0, -i, 0).Format("2006-01"))
	}
	for i := 3; i >= 0; i-- {
		report, e := analysisReportFrom(ctx, tx, owner, months[i], now)
		if e != nil {
			return nil, nil, e
		}
		if report.Period.Completeness != "complete" {
			comparable = false
			break
		}
	}

	if comparable {
		rows, e := tx.QueryContext(ctx, `WITH source AS(SELECT t.*,COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label) AS name
 FROM transactions t WHERE t.owner_id=$1 AND t.amount<0 AND t.tx_date BETWEEN $2::date AND $3::date`+notTransferT+`)
 SELECT 'merchant',name,to_char(tx_date,'YYYY-MM'),(-SUM(amount))::text FROM source GROUP BY name,to_char(tx_date,'YYYY-MM')
 UNION ALL SELECT 'tag',tg.name,to_char(t.tx_date,'YYYY-MM'),(-SUM(t.amount))::text FROM source t
 JOIN transaction_tags tt ON tt.transaction_id=t.id JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id
 GROUP BY tg.name,to_char(t.tx_date,'YYYY-MM')`, owner, first.AddDate(0, -3, 0).Format("2006-01-02"), last.Format("2006-01-02"))
		if e != nil {
			return nil, nil, e
		}
		type group struct {
			kind, name string
			values     map[string]money.Cents
		}
		groups := map[string]*group{}
		for rows.Next() {
			var kind, name, m, raw string
			if e = rows.Scan(&kind, &name, &m, &raw); e != nil {
				rows.Close()
				return nil, nil, e
			}
			amount, pe := parseAnalysisAmount(raw)
			if pe != nil {
				rows.Close()
				return nil, nil, pe
			}
			key := kind + ":" + name
			if groups[key] == nil {
				groups[key] = &group{kind, name, map[string]money.Cents{}}
			}
			groups[key].values[m] = amount
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, nil, e
		}
		keys := []string{}
		for key := range groups {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			g := groups[key]
			a, b, c, d := g.values[months[0]], g.values[months[1]], g.values[months[2]], g.values[months[3]]
			if g.kind == "merchant" && a > 0 && a < b && b < c && c < d && (d-a)*100 >= a*40 {
				out = append(out, map[string]any{"kind": "merchant_drift", "title": "Dépenses en hausse chez " + g.name, "detail": fmt.Sprintf("Quatre mois documentés %s à %s : %s €, %s €, %s €, %s €. Comparaison des débits ordinaires, sans déduire la cause de la hausse.", months[0], months[3], simulationNumber(a), simulationNumber(b), simulationNumber(c), simulationNumber(d)), "source": map[string]any{"merchant": g.name, "from": first.AddDate(0, -3, 0).Format("2006-01-02"), "to": last.Format("2006-01-02"), "type": "debit"}})
			}
			if g.kind == "tag" {
				sum := a + b + c
				avg := recurringReviewRound(sum, 3)
				if sum > 0 && d-avg >= 5000 && d*15 > sum*8 {
					out = append(out, map[string]any{"kind": "tag_surge", "title": "Dépenses taguées à examiner : " + g.name, "detail": fmt.Sprintf("%s € en %s, contre %s € par mois sur %s à %s (trois mois documentés, mois vides inclus). Les tags peuvent se chevaucher.", simulationNumber(d), month, simulationNumber(avg), months[0], months[2]), "source": map[string]any{"tag": g.name, "month": month, "type": "debit"}})
				}
			}
		}
	} else {
		limits = append(limits, "Variations mensuelles par marchand et par tag suspendues : les quatre mois nécessaires ne sont pas tous documentés. Une absence de salaire en cours de mois n’est pas une anomalie à elle seule.")
	}
	// Annual patterns are occurrence hypotheses only. No projected amount/date.
	rows, err := tx.QueryContext(ctx, `SELECT COALESCE(NULLIF(t.merchant,''),NULLIF(t.display_label,''),t.label),to_char(t.tx_date,'YYYY-MM'),count(*)
 FROM transactions t WHERE t.owner_id=$1 AND t.amount<=-80 AND t.tx_date BETWEEN $2::date AND $3::date`+notTransferT+`
 GROUP BY 1,2 ORDER BY 1,2`, owner, first.AddDate(0, -35, 0).Format("2006-01-02"), last.Format("2006-01-02"))
	if err != nil {
		return nil, nil, err
	}
	patterns := map[string][]string{}
	for rows.Next() {
		var name, m string
		var count int
		if err = rows.Scan(&name, &m, &count); err != nil {
			rows.Close()
			return nil, nil, err
		}
		patterns[name] = append(patterns[name], m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	names := []string{}
	for name := range patterns {
		names = append(names, name)
	}
	sort.Strings(names)
	next := first.AddDate(0, 1, 0).Format("01")
	for _, name := range names {
		years, calendar := map[string]bool{}, map[string]bool{}
		for _, m := range patterns[name] {
			calendar[m[5:7]] = true
			if m[5:7] == next {
				years[m[:4]] = true
			}
		}
		if len(years) >= 2 && len(calendar) <= 2 {
			out = append(out, map[string]any{"kind": "seasonal", "title": "Motif annuel possible : " + name, "detail": fmt.Sprintf("Au moins un débit d’au moins 80 € a été observé au mois %s sur %d années dans les trois dernières années. Une prochaine occurrence reste une hypothèse, sans montant ni échéance certains.", next, len(years)), "source": map[string]any{"merchant": name, "from": first.AddDate(0, -35, 0).Format("2006-01-02"), "to": last.Format("2006-01-02"), "type": "debit"}})
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, nil, err
	}
	return out, limits, nil
}
