package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"themis/internal/intelligence"
	"themis/internal/money"
)

func (s *Server) intelligenceFeed(w http.ResponseWriter, r *http.Request) {
	limit := 3
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 50 {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("limit doit être compris entre 1 et 50"))
			return
		}
		limit = parsed
	}
	items, err := intelligence.Feed(r.Context(), s.DB, s.ownerID(r), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("feed intelligence indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"discoveries": items, "limit": limit})
}

func (s *Server) intelligenceDiscovery(w http.ResponseWriter, r *http.Request) {
	owner := s.ownerID(r)
	id := r.PathValue("id")
	item, err := intelligence.Get(r.Context(), s.DB, owner, id)
	if errors.Is(err, intelligence.ErrNotFound) {
		writeErr(w, http.StatusNotFound, intelligence.ErrNotFound)
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("découverte indisponible"))
		return
	}
	history, err := intelligence.History(r.Context(), s.DB, owner, id, 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("historique de découverte indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"discovery": item, "observations": history})
}

func (s *Server) intelligenceFeedback(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Action string `json:"action"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(new(any)) != io.EOF {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("feedback invalide"))
		return
	}
	state, err := intelligence.Feedback(r.Context(), s.DB, s.ownerID(r), r.PathValue("id"), input.Action)
	if errors.Is(err, intelligence.ErrNotFound) {
		writeErr(w, http.StatusNotFound, intelligence.ErrNotFound)
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("action de feedback invalide"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"action": input.Action, "state": state})
}

func (s *Server) recomputeIntelligence(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Month string `json:"month"`
	}
	if r.Body != nil {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil && err != io.EOF {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("requête de recalcul invalide"))
			return
		}
		if decoder.Decode(new(any)) != io.EOF {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("requête de recalcul invalide"))
			return
		}
	}
	now := time.Now()
	if _, _, _, err := reliabilityWindow(input.Month, now); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	owner := s.ownerID(r)
	releaseLock, acquired, err := intelligence.AcquireRecomputeLock(r.Context(), s.DB, owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("verrou de recalcul indisponible"))
		return
	}
	if !acquired {
		writeErr(w, http.StatusConflict, intelligence.ErrRecomputeBusy)
		return
	}
	defer releaseLock()
	month, err := s.analysisMonth(r.Context(), owner, input.Month, now)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("période analytique indisponible"))
		return
	}
	report, err := s.loadRecurringReview(r.Context(), owner, month, now)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("analyse récurrente indisponible"))
		return
	}
	legacyRecurringSignals := analysisRecurringSignals(report)
	analysisSignals, _, err := s.loadAnalysisSignals(r.Context(), owner, month, now)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("signaux analytiques indisponibles"))
		return
	}
	discoveries := make([]intelligence.Discovery, 0, len(report.Items)+len(analysisSignals))
	for _, item := range report.Items {
		if item.Status == "ignored" || item.Direction != "debit" || item.protected || item.Nature == "transfer_savings" || item.Nature == "cash_withdrawal" {
			continue
		}
		discoveries = append(discoveries, recurringDiscovery(owner, item))
	}
	for _, signal := range append(legacyRecurringSignals, analysisSignals...) {
		kind, _ := signal["kind"].(string)
		if kind == "recurring" {
			continue
		}
		discovery, err := discoveryFromAnalysisSignal(owner, month, signal)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Errorf("adaptation d’un signal analytique impossible"))
			return
		}
		discoveries = append(discoveries, discovery)
	}
	if len(discoveries) > 500 {
		writeErr(w, http.StatusUnprocessableEntity, fmt.Errorf("trop de signaux pour un recalcul interactif"))
		return
	}
	currentIDs := map[string][]string{"recurring": {}, "price_creep": {}, "duplicate_subs": {}, "merchant_drift": {}, "tag_surge": {}, "seasonal": {}}
	for _, discovery := range discoveries {
		ruleVersion := "analysis-signals-v1"
		if discovery.Kind == "recurring" {
			ruleVersion = "recurring-v1"
		}
		stored, err := intelligence.Record(r.Context(), s.DB, discovery, ruleVersion)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Errorf("enregistrement du recalcul impossible"))
			return
		}
		currentIDs[stored.Kind] = append(currentIDs[stored.Kind], stored.ID)
	}
	for kind, ids := range currentIDs {
		if err := intelligence.ReconcileActive(r.Context(), s.DB, owner, kind, ids); err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Errorf("synchronisation des découvertes impossible"))
			return
		}
	}
	prioritized, err := intelligence.Feed(r.Context(), s.DB, owner, 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("feed intelligence indisponible"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"month": month, "discoveries": prioritized, "count": len(prioritized), "rule_version": "recurring-v1"})
}

func discoveryFromAnalysisSignal(owner int64, month string, signal map[string]any) (intelligence.Discovery, error) {
	kind, _ := signal["kind"].(string)
	title, _ := signal["title"].(string)
	summary, _ := signal["detail"].(string)
	source, _ := signal["source"].(map[string]any)
	if kind == "" || title == "" || summary == "" || source == nil {
		return intelligence.Discovery{}, fmt.Errorf("signal structure incomplete")
	}
	anchor := ""
	for _, key := range []string{"key", "merchant", "tag", "segment"} {
		if value, ok := source[key].(string); ok && value != "" {
			anchor = value
			break
		}
	}
	if anchor == "" {
		return intelligence.Discovery{}, fmt.Errorf("signal identity unavailable")
	}
	if kind == "price_creep" {
		anchor = "recurrence:" + anchor
	}
	anchor = strings.ToLower(strings.TrimSpace(anchor))
	ids, _ := source["ids"].([]int64)
	periods, _ := source["periods"].([]string)
	for _, key := range []string{"from", "to", "month"} {
		if value, ok := source[key].(string); ok && value != "" {
			periods = append(periods, value)
		}
	}
	periods = uniqueStrings(periods)
	coverage := "unknown"
	switch kind {
	case "merchant_drift", "tag_surge":
		coverage = "complete" // These detectors emit only after the four-month documentary gate.
	case "price_creep", "duplicate_subs":
		if complete, ok := source["coverage_complete"].(bool); ok && complete {
			coverage = "complete"
		}
	}
	confidence, evidenceFactor := 45, 45
	if coverage == "complete" {
		confidence = 70
		if len(ids) > 0 {
			confidence = 85
		}
		evidenceFactor = confidence
	}
	metrics, _ := signal["metrics"].([]intelligence.Metric)
	impactCents := int64(0)
	for _, metric := range metrics {
		if metric.Kind == intelligence.MetricMoneyCents && strings.Contains(metric.Name, "increase") || metric.Kind == intelligence.MetricMoneyCents && metric.Name == "observed_difference_cents" {
			impactCents = metric.Value
		}
	}
	if impactCents < 0 {
		impactCents = -impactCents
	}
	impact := recurringImpactScore(money.Cents(impactCents))
	relevance, _ := intelligence.Score(intelligence.Factors{Impact: impact, Evidence: evidenceFactor, Novelty: 50, Context: 0, Action: 50}, intelligence.DefaultWeights, len(ids) > 0 || len(periods) > 0, coverage)
	period := month
	return intelligence.Discovery{
		OwnerID: owner, Kind: kind, SubjectKey: intelligence.SubjectKey(kind, anchor), Period: period,
		Title: title, Summary: summary, Severity: "information", Confidence: confidence,
		RelevanceScore: relevance, ImpactScore: impact, Metrics: metrics,
		Evidence:    intelligence.Evidence{TransactionIDs: ids, Periods: periods, Source: "analysis_discoveries:" + kind},
		Coverage:    intelligence.Coverage{Status: coverage, Periods: len(periods), Transactions: len(ids), Description: analysisCoverageDescription(kind, coverage)},
		Limitations: analysisSignalLimitations(kind), State: "active",
	}, nil
}

func analysisCoverageDescription(kind, coverage string) string {
	switch {
	case kind == "seasonal":
		return "Motif calculé sur les opérations observées; l’exhaustivité documentaire de chaque année n’est pas établie."
	case kind == "tag_surge":
		return "Quatre périodes mensuelles documentées; les tags peuvent se chevaucher."
	case kind == "merchant_drift":
		return "Quatre périodes mensuelles complètes comparées; la contribution comptable ne démontre pas une cause."
	case coverage == "complete":
		return "Les sources du détecteur couvrent les périodes référencées."
	default:
		return "La couverture documentaire complète n’est pas établie."
	}
}

func analysisSignalLimitations(kind string) []string {
	switch kind {
	case "seasonal":
		return []string{"Une répétition calendaire possible ne prédit ni une prochaine opération ni son montant."}
	case "price_creep":
		return []string{"Un écart de montant observé ne prouve pas un changement de tarif."}
	case "duplicate_subs":
		return []string{"Des services dans un même segment ne prouvent ni des contrats actifs ni une dépense inutile."}
	case "merchant_drift":
		return []string{"La contribution comptable ne démontre pas la cause de la variation."}
	case "tag_surge":
		return []string{"Les tags peuvent se chevaucher; cette somme n’est pas additive au total des dépenses."}
	default:
		return []string{}
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func recurringDiscovery(owner int64, item recurringReviewItem) intelligence.Discovery {
	confidence, evidenceFactor := 45, 45
	coverageStatus := "unknown"
	if item.CoverageComplete {
		coverageStatus = "complete"
	}
	if item.Confidence == "probable" && item.CoverageComplete {
		confidence, evidenceFactor = 85, 85
	}
	amount := medianCents(item.SourceAmounts)
	impact := recurringImpactScore(amount)
	relevance, _ := intelligence.Score(intelligence.Factors{Impact: impact, Evidence: evidenceFactor, Novelty: 50, Context: 0, Action: 50}, intelligence.DefaultWeights, len(item.TransactionIDs) > 0, coverageStatus)
	frequency := "non déterminée"
	if item.Frequency != nil {
		frequency = *item.Frequency
	}
	label := item.Label
	if label == "" {
		label = item.Merchant
	}
	summary := fmt.Sprintf("%d opérations observées sur %d mois ; montant médian de %s € par opération. Cadence détectée : %s. Une récurrence observée ne prouve pas un abonnement.", item.OccurrenceCount, item.MonthsCount, item.TypicalAmount, frequency)
	limitations := []string{item.ConfidenceReason}
	if item.Nature == "subscription" {
		limitations = append(limitations, "La catégorie ou le marchand suggère une nature d’abonnement, sans preuve de contrat actif.")
	}
	var detected time.Time
	if item.FirstDate != "" {
		detected, _ = time.Parse("2006-01-02", item.FirstDate)
	}
	period := item.LastDate
	if len(period) >= 7 {
		period = period[:7]
	}
	return intelligence.Discovery{
		OwnerID: owner, Kind: "recurring", SubjectKey: item.Key, Period: period,
		Title: "Récurrence observée · " + label, Summary: summary, Severity: "information",
		Confidence: confidence, RelevanceScore: relevance, ImpactScore: impact,
		Metrics:     []intelligence.Metric{{Name: "occurrences", Kind: intelligence.MetricCount, Value: int64(item.OccurrenceCount)}, {Name: "typical_amount", Kind: intelligence.MetricMoneyCents, Value: int64(amount)}},
		Evidence:    intelligence.Evidence{TransactionIDs: item.TransactionIDs, Periods: item.Months, Source: "recurring_review"},
		Coverage:    intelligence.Coverage{Status: coverageStatus, Periods: item.MonthsCount, Transactions: item.OccurrenceCount, Description: item.ConfidenceReason},
		Limitations: limitations, State: "active", DetectedAt: detected,
	}
}

// recurringImpactScore is an initial, reviewable ranking scale. It does not
// imply a saving opportunity or a statistical confidence level.
func recurringImpactScore(amount money.Cents) int {
	switch {
	case amount >= 200000:
		return 100
	case amount >= 50000:
		return 75
	case amount >= 10000:
		return 50
	case amount >= 2500:
		return 30
	case amount > 0:
		return 15
	}
	return 0
}

func medianCents(values []money.Cents) money.Cents {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]money.Cents(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	middle := len(ordered) / 2
	if len(ordered)%2 != 0 {
		return ordered[middle]
	}
	return money.Cents((int64(ordered[middle-1]) + int64(ordered[middle]) + 1) / 2)
}
