package api

import (
	"encoding/json"
	"testing"
	"time"

	"themis/internal/money"
)

func recurringTestDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
func recurringTestSource(id int64, date, label string, amount money.Cents) recurringReviewSource {
	return recurringReviewSource{ID: id, AccountID: 1, Date: recurringTestDate(date), Label: label, Amount: amount}
}
func recurringTestSpans() map[int64][]heroDateSpan {
	return map[int64][]heroDateSpan{1: {{recurringTestDate("2020-01-01"), recurringTestDate("2020-03-31")}}}
}
func recurringNetflixSources() []recurringReviewSource {
	return []recurringReviewSource{recurringTestSource(1, "2020-01-10", "PRLV NETFLIX REF 11111", -1299), recurringTestSource(2, "2020-02-10", "NETFLIX 22222", -1299), recurringTestSource(3, "2020-03-10", "Paiement Netflix 33333", -1299)}
}
func TestRecurringReviewExplainsPerTransactionAverageAndMedian(t *testing.T) {
	sources := recurringNetflixSources()
	sources[0].Amount = -1000
	sources[1].Amount = -1500
	sources[2].Amount = -9000
	out := buildRecurringReview(sources, recurringTestSpans(), recurringTestDate("2020-03-31"))
	if len(out) != 1 {
		t.Fatalf("normalized merchant not grouped: %+v", out)
	}
	item := out[0]
	if item.AveragePerTransaction != json.Number("38.33") || item.TypicalAmount != json.Number("15.00") || item.AverageDenominator != 3 || item.OccurrenceCount != 3 {
		t.Fatalf("different denominators or hidden outlier: %+v", item)
	}
	if item.Nature != "subscription" || item.Confidence != "probable" || !item.CoverageComplete || item.Frequency == nil || *item.Frequency != "monthly" {
		t.Fatalf("documented cadence missing: %+v", item)
	}
	if len(item.TransactionIDs) != 3 || item.SourceAmounts[2] != 9000 {
		t.Fatal("all sources and the atypical value must remain traceable")
	}
}
func TestRecurringReviewNeverCallsCashOrSavingsSubscriptions(t *testing.T) {
	for _, label := range []string{"Mens.pel 123456789", "Ret DAB X4519 Les Ulis"} {
		sources := []recurringReviewSource{recurringTestSource(1, "2020-01-10", label, -10000), recurringTestSource(2, "2020-02-10", label, -10000), recurringTestSource(3, "2020-03-10", label, -10000)}
		out := buildRecurringReview(sources, recurringTestSpans(), recurringTestDate("2020-03-31"))
		if len(out) != 1 {
			t.Fatal("expected repeated movement")
		}
		if out[0].Nature == "subscription" || out[0].EstimatedNextDate != nil && out[0].Nature == "variable_recurring" {
			t.Fatalf("false subscription or due date: %+v", out[0])
		}
		detected := out[0].Nature
		applyRecurringReview(&out[0], recurringReviewDecision{Status: "confirmed", Nature: "subscription"})
		if out[0].Nature != detected {
			t.Fatal("protected bank movement reclassified by review metadata")
		}
	}
}
func TestRecurringReviewGroupsDirectionsAccountsAndCategoriesSeparately(t *testing.T) {
	sources := []recurringReviewSource{}
	var id int64
	for _, date := range []string{"2020-01-10", "2020-02-10", "2020-03-10"} {
		for _, account := range []int64{1, 2} {
			for _, amount := range []money.Cents{-10000, 10000} {
				id++
				tx := recurringTestSource(id, date, "VERSEMENT PEL", amount)
				tx.AccountID = account
				sources = append(sources, tx)
			}
		}
	}
	out := buildRecurringReview(sources, recurringTestSpans(), recurringTestDate("2020-03-31"))
	if len(out) != 4 {
		t.Fatalf("accounts or transfer directions merged: %+v", out)
	}
	for _, item := range out {
		if item.AveragePerTransaction != "100.00" || item.OccurrenceCount != 3 || item.Nature != "transfer_savings" {
			t.Fatalf("signed transfers netted: %+v", item)
		}
	}
}
func TestRecurringReviewStabilityDoesNotEstablishCadence(t *testing.T) {
	sources := []recurringReviewSource{recurringTestSource(1, "2020-01-02", "CARREFOUR", -5000), recurringTestSource(2, "2020-01-03", "CARREFOUR", -5000), recurringTestSource(3, "2020-02-20", "CARREFOUR", -5000)}
	out := buildRecurringReview(sources, recurringTestSpans(), recurringTestDate("2020-03-31"))
	if len(out) != 1 || out[0].Frequency != nil || out[0].Confidence != "insufficient" || out[0].Nature != "variable_recurring" || out[0].EstimatedNextDate != nil {
		t.Fatalf("stable purchases became a subscription: %+v", out)
	}
}
func TestRecurringReviewMultipleSameDayOccurrencesArePreservedWithoutFalseCadence(t *testing.T) {
	sources := recurringNetflixSources()
	duplicate := sources[1]
	duplicate.ID = 20
	sources = append(sources, duplicate)
	out := buildRecurringReview(sources, recurringTestSpans(), recurringTestDate("2020-03-31"))
	if len(out) != 1 || out[0].OccurrenceCount != 4 || len(out[0].TransactionIDs) != 4 || out[0].Frequency != nil {
		t.Fatalf("distinct same-day purchases lost or false cadence: %+v", out)
	}
}
func TestRecurringReviewMissingCoverageAndUserConfirmationStayDistinct(t *testing.T) {
	out := buildRecurringReview(recurringNetflixSources(), nil, recurringTestDate("2020-03-31"))
	if out[0].CoverageComplete || out[0].Confidence != "insufficient" || out[0].EstimatedNextDate != nil {
		t.Fatal("missing documentation hidden")
	}
	applyRecurringReview(&out[0], recurringReviewDecision{Status: "confirmed", Nature: "subscription", UpdatedAt: "2020-04-01"})
	if out[0].Status != "confirmed" || out[0].Confidence != "insufficient" || out[0].CoverageComplete {
		t.Fatal("confirmation fabricated coverage")
	}
}
func TestRecurringReviewHistoricalCutoffAndNoRolledForwardDueDate(t *testing.T) {
	sources := recurringNetflixSources()
	sources = append(sources, recurringTestSource(4, "2020-04-10", "NETFLIX", -9999))
	out := buildRecurringReview(sources, recurringTestSpans(), recurringTestDate("2020-03-31"))
	if out[0].LastAmount != "12.99" || out[0].LastDate != "2020-03-10" || out[0].EstimatedNextDate == nil || *out[0].EstimatedNextDate != "2020-04-10" {
		t.Fatalf("future source leaked: %+v", out[0])
	}
	spans := map[int64][]heroDateSpan{1: {{recurringTestDate("2020-01-01"), recurringTestDate("2020-06-30")}}}
	out = buildRecurringReview(sources[:3], spans, recurringTestDate("2020-06-30"))
	if out[0].EstimatedNextDate != nil {
		t.Fatal("an overdue hypothetical payment was silently rolled into the future")
	}
}
func TestRecurringReviewCalendarClampsMonthEnd(t *testing.T) {
	sources := []recurringReviewSource{recurringTestSource(1, "2020-01-31", "NETFLIX", -1000), recurringTestSource(2, "2020-02-29", "NETFLIX", -1000), recurringTestSource(3, "2020-03-31", "NETFLIX", -1000)}
	out := buildRecurringReview(sources, recurringTestSpans(), recurringTestDate("2020-03-31"))
	if out[0].Frequency == nil || *out[0].Frequency != "monthly" || out[0].EstimatedNextDate == nil || *out[0].EstimatedNextDate != "2020-04-30" {
		t.Fatalf("month-end cadence: %+v", out[0])
	}
}
func TestRecurringReviewVariableFixedChargeAndTwoObservations(t *testing.T) {
	sources := []recurringReviewSource{recurringTestSource(1, "2020-01-10", "EDF", -10000), recurringTestSource(2, "2020-02-10", "EDF", -14000), recurringTestSource(3, "2020-03-10", "EDF", -12000)}
	out := buildRecurringReview(sources, recurringTestSpans(), recurringTestDate("2020-03-31"))
	if out[0].Nature != "fixed_charge" || out[0].TypicalAmount != "120.00" {
		t.Fatalf("fixed charge incorrectly treated as a subscription: %+v", out[0])
	}
	out = buildRecurringReview(sources[:2], recurringTestSpans(), recurringTestDate("2020-03-31"))
	if out[0].Frequency != nil || out[0].Confidence != "insufficient" {
		t.Fatal("two observations asserted a frequency")
	}
}
