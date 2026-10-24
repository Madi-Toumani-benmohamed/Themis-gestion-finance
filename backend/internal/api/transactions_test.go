package api

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTransactionFiltersRejectInvalidInputs(t *testing.T) {
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	for _, query := range []string{"month=2020-13", "month=nope", "from=2020-02-30&to=2020-03-01", "from=2020-01-02&to=2020-01-01", "from=2020-01-01", "to=2020-01-01", "limit=0", "limit=1001", "offset=-1", "offset=abc", "domain=wat", "account=-1", "type=refund", "ids=1,0", "all=yes", "budget_id=wat"} {
		v, _ := url.ParseQuery(query)
		if _, err := parseTransactionFilter(v, 1, true, now); err == nil {
			t.Errorf("accepted %s", query)
		}
	}
	f, err := parseTransactionFilter(url.Values{}, 1, true, now)
	if err != nil || f.from != "2026-10-01" || f.to != "2026-10-24" || f.limit != 100 {
		t.Fatalf("default scope: %+v %v", f, err)
	}
	v, _ := url.ParseQuery("month=2020-02&from=2020-02-10&to=2020-03-05&limit=2&offset=4")
	f, err = parseTransactionFilter(v, 1, true, now)
	if err != nil || f.from != "2020-02-10" || f.to != "2020-02-29" || f.limit != 2 || f.offset != 4 || strings.Contains(f.where, "to_char") {
		t.Fatalf("bounded leap-month scope: %+v %v", f, err)
	}
	for _, query := range []string{"all=1", "ids=1,2"} {
		v, _ = url.ParseQuery(query)
		f, err = parseTransactionFilter(v, 1, true, now)
		if err != nil || f.from != "" || f.to != "" {
			t.Fatalf("explicit source scope: %+v %v", f, err)
		}
	}
	legacy, err := parseTransactionFilter(url.Values{}, 1, false, now)
	if err != nil || legacy.limit != 1000 || legacy.from != "" {
		t.Fatal("legacy contract changed")
	}
	currentBudget, err := parseTransactionFilter(url.Values{"month": {"2026-10"}, "budget_id": {"1"}}, 1, false, now)
	if err != nil || currentBudget.to != "2026-10-24" {
		t.Fatalf("budget/current sources diverge: %+v %v", currentBudget, err)
	}
}

func TestTransactionAggregateAndTagValidation(t *testing.T) {
	for raw, want := range map[string]int64{"0": 0, "0.01": 1, "-947.67": -94767, "99999999999.99": 9999999999999, "1000000000000.1": 100000000000010} {
		got, err := parseTransactionAggregate(raw)
		if err != nil || int64(got) != want {
			t.Fatalf("%q = %d, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"", "NaN", "1.001", "1.0.0", "92233720368547758.08", "1e2", "--1"} {
		if _, err := parseTransactionAggregate(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	tags, err := validateTransactionTags([]string{" TaxES ", "taxes", "Maison"})
	if err != nil || !reflect.DeepEqual(tags, []string{"taxes", "maison"}) {
		t.Fatalf("tags %+v %v", tags, err)
	}
	for _, names := range [][]string{{""}, {strings.Repeat("a", 41)}, make([]string, 51)} {
		if _, err := validateTransactionTags(names); err == nil {
			t.Fatal("invalid tags accepted")
		}
	}
}
