package api

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSimulationDisplayedAmountsReconcile(t *testing.T) {
	in := simulationInput{}
	result, err := calculateSimulation(in, 324947, []simulationBase{{"Dépenses", 419714}})
	if err != nil {
		t.Fatal(err)
	}
	if result["current_net"] != json.Number("-947.67") || result["simulated_net"] != json.Number("-947.67") || result["delta"] != json.Number("0.00") {
		t.Fatalf("unreconciled: %+v", result)
	}
}
func TestSimulationIndependentCutsAndExpenses(t *testing.T) {
	income := json.Number("4000")
	in := simulationInput{Income: &income, Cuts: []simulationCut{{"A", "10"}, {"B", "25.25"}}, Expenses: []simulationExpense{{"Ajout", "10.01"}}, Loan: &simulationLoan{"1200", "0", 12}}
	result, err := calculateSimulation(in, 324947, []simulationBase{{"A", 300000}, {"B", 119714}})
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]json.Number{"monthly_saved": "602.28", "base_spend": "4197.14", "simulated_spend": "3704.87", "current_net": "-947.67", "simulated_net": "295.13", "delta": "1242.80", "loan_payment": "100.00"} {
		if result[key] != want {
			t.Errorf("%s: %v != %s", key, result[key], want)
		}
	}
}
func TestSimulationRoundingAndLoan(t *testing.T) {
	result, err := calculateSimulation(simulationInput{Cuts: []simulationCut{{"A", "50"}, {"B", "50"}}}, 100, []simulationBase{{"A", 1}, {"B", 1}})
	if err != nil || result["monthly_saved"] != json.Number("0.02") {
		t.Fatalf("rounding: %+v %v", result, err)
	}
	result, err = calculateSimulation(simulationInput{Loan: &simulationLoan{"15000", "4.5", 48}}, 0, nil)
	if err != nil || result["loan_payment"] != json.Number("342.05") {
		t.Fatalf("loan: %+v %v", result, err)
	}
}
func TestSimulationInvalidInputs(t *testing.T) {
	for _, in := range []simulationInput{
		{Cuts: []simulationCut{{"A", "101"}}}, {Cuts: []simulationCut{{"A", "-1"}}},
		{Cuts: []simulationCut{{"A", "1.001"}}}, {Cuts: []simulationCut{{"A", "10"}, {"A", "20"}}},
		{Cuts: []simulationCut{{"Unknown", "10"}}}, {Expenses: []simulationExpense{{"X", "1.001"}}},
		{Loan: &simulationLoan{"100", "0", 0}}, {Loan: &simulationLoan{"100", "0", 601}},
	} {
		if _, err := calculateSimulation(in, 100, []simulationBase{{"A", 100}}); err == nil {
			t.Fatalf("accepted invalid input: %+v", in)
		}
	}
}
func TestSimulationThreeClosedCalendarMonths(t *testing.T) {
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	for _, sample := range []struct{ last, first, end string }{{"2026-10-24", "2026-07-01", "2026-10-01"}, {"2026-09-21", "2026-07-01", "2026-10-01"}, {"2026-01-21", "2025-11-01", "2026-02-01"}} {
		last, _ := time.Parse("2006-01-02", sample.last)
		first, end := simulationMonths(last, now)
		if first.Format("2006-01-02") != sample.first || end.Format("2006-01-02") != sample.end {
			t.Fatalf("bad window %s: %s %s", sample.last, first, end)
		}
	}
}

func TestSimulationZeroRateRoundingNeverCreatesNegativeInterest(t *testing.T) {
	result, err := calculateSimulation(simulationInput{Loan: &simulationLoan{"100", "0", 3}}, 0, nil)
	if err != nil || result["loan_payment"] != json.Number("33.33") || result["loan_total_cost"] != json.Number("100.00") || result["loan_interest"] != json.Number("0.00") {
		t.Fatalf("loan rounding: %+v %v", result, err)
	}
}
