package api

import (
	"encoding/json"
	"testing"
	"themis/internal/money"
)

func TestBudgetThresholdsExactCents(t *testing.T) {
	for _, sample := range []struct {
		spent     int64
		status    string
		remaining string
		exceeded  bool
	}{{0, "available", "10.01", false}, {1001, "reached", "0.00", false}, {1002, "over", "-0.01", true}} {
		row := budgetStatus{limit: 1001, spent: money.Cents(sample.spent)}
		finishBudgetStatus(&row)
		if row.Status != sample.status || row.Exceeded != sample.exceeded || row.Remaining != json.Number(sample.remaining) || row.Spent != row.DebitTotal || row.Projected != nil || row.ProjectionAvailable {
			t.Fatalf("bad threshold: %+v", row)
		}
	}
}
