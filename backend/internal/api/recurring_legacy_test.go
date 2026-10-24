package api

import "testing"

func TestLegacyRecurrenceUsesEconomicEvidence(t *testing.T) {
	items := buildRecurringReview(recurringNetflixSources(), recurringTestSpans(), recurringTestDate("2020-03-31"))
	out, err := legacyRecurringItems(items, 2)
	if err != nil || len(out) != 1 || !out[0].Regular || out[0].Nature != "subscription" || out[0].EstimatedNextDate == nil {
		t.Fatalf("documented legacy adapter: %+v %v", out, err)
	}
	items[0].Confidence = "insufficient"
	out, err = legacyRecurringItems(items, 2)
	if err != nil || len(out) != 1 || out[0].Regular {
		t.Fatal("incomplete imports fabricated an engagement")
	}
	items[0].Status = "ignored"
	out, err = legacyRecurringItems(items, 2)
	if err != nil || len(out) != 0 {
		t.Fatal("ignored recurrence leaked into consumers")
	}
	items[0].Status = "detected"
	items[0].Nature = "transfer_savings"
	out, err = legacyRecurringItems(items, 2)
	if err != nil || len(out) != 0 {
		t.Fatal("savings counted as subscriptions")
	}
	items[0].Nature = "variable_recurring"
	items[0].protected = true
	out, err = legacyRecurringItems(items, 2)
	if err != nil || len(out) != 0 {
		t.Fatal("protected cash movement counted as subscription")
	}
}
