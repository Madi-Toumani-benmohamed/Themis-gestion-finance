// Package intelligence contains shared, deterministic contracts for financial
// discoveries. It deliberately has no dependency on an LLM or Graphiti.
package intelligence

import (
	"sort"
	"time"
)

type MetricKind string

const (
	MetricMoneyCents MetricKind = "money_cents"
	MetricCount      MetricKind = "count"
	MetricPercent    MetricKind = "percent"
	MetricDuration   MetricKind = "duration_days"
)

type Metric struct {
	Name  string     `json:"name"`
	Kind  MetricKind `json:"kind"`
	Value int64      `json:"value"` // Percent metrics use basis points; durations use whole days.
}

type Evidence struct {
	TransactionIDs []int64  `json:"transaction_ids,omitempty"`
	Periods        []string `json:"periods,omitempty"`
	Source         string   `json:"source,omitempty"`
}

type Coverage struct {
	Status       string `json:"status"` // complete, partial, unknown
	Periods      int    `json:"periods,omitempty"`
	Transactions int    `json:"transactions,omitempty"`
	Description  string `json:"description,omitempty"`
}

type Discovery struct {
	ID             string    `json:"id"` // Stable identity of the phenomenon.
	ObservationID  string    `json:"observation_id"`
	OwnerID        int64     `json:"owner_id,omitempty"`
	Kind           string    `json:"kind"`
	SubjectKey     string    `json:"subject_key"`
	Period         string    `json:"period,omitempty"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary"`
	Severity       string    `json:"severity,omitempty"`
	Confidence     int       `json:"confidence"` // Evidence quality, 0..100.
	RelevanceScore int       `json:"relevance_score"`
	ImpactScore    int       `json:"impact_score"`
	Metrics        []Metric  `json:"metrics,omitempty"`
	Evidence       Evidence  `json:"evidence"`
	Coverage       Coverage  `json:"coverage"`
	Limitations    []string  `json:"limitations,omitempty"`
	RuleVersion    string    `json:"rule_version"`
	State          string    `json:"state"`
	DetectedAt     time.Time `json:"detected_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Factors struct {
	Impact   int `json:"impact"`
	Evidence int `json:"evidence"`
	Novelty  int `json:"novelty"`
	Context  int `json:"context"`
	Action   int `json:"action"`
}

type Weights struct {
	Impact   int
	Evidence int
	Novelty  int
	Context  int
	Action   int
}

var DefaultWeights = Weights{Impact: 30, Evidence: 25, Novelty: 20, Context: 15, Action: 10}

// Score calculates priority, never confidence. Missing evidence or unknown
// partial/unknown coverage caps priority below the default feed threshold.
func Score(f Factors, w Weights, hasEvidence bool, coverageStatus string) (int, bool) {
	if !validFactor(f.Impact) || !validFactor(f.Evidence) || !validFactor(f.Novelty) || !validFactor(f.Context) || !validFactor(f.Action) {
		return 0, false
	}
	if !validFactor(w.Impact) || !validFactor(w.Evidence) || !validFactor(w.Novelty) || !validFactor(w.Context) || !validFactor(w.Action) || w.Impact+w.Evidence+w.Novelty+w.Context+w.Action != 100 {
		return 0, false
	}
	weighted := f.Impact*w.Impact + f.Evidence*w.Evidence + f.Novelty*w.Novelty + f.Context*w.Context + f.Action*w.Action
	score := (weighted + 50) / 100
	if !hasEvidence || coverageStatus != "complete" {
		if score > 49 {
			score = 49
		}
	}
	return score, true
}

func validFactor(v int) bool { return v >= 0 && v <= 100 }

// SubjectKey hashes a detector anchor so merchant/tag text is not duplicated
// into the registry identity column.
func SubjectKey(kind, anchor string) string { return digest(kind + "\x00" + anchor) }

// Prioritize deduplicates by stable phenomenon ID, removes hidden items, and
// returns a deterministic relevance-ordered slice up to limit.
func Prioritize(items []Discovery, limit int) []Discovery {
	if limit <= 0 {
		return []Discovery{}
	}
	best := make(map[string]Discovery, len(items))
	for _, item := range items {
		if item.State != "active" || item.ID == "" || item.Confidence < 0 || item.Confidence > 100 || item.RelevanceScore < 0 || item.RelevanceScore > 100 {
			continue
		}
		previous, exists := best[item.ID]
		if !exists || item.RelevanceScore > previous.RelevanceScore || (item.RelevanceScore == previous.RelevanceScore && item.UpdatedAt.After(previous.UpdatedAt)) {
			best[item.ID] = item
		}
	}
	out := make([]Discovery, 0, len(best))
	for _, item := range best {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RelevanceScore != out[j].RelevanceScore {
			return out[i].RelevanceScore > out[j].RelevanceScore
		}
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return out[i].ObservationID < out[j].ObservationID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
