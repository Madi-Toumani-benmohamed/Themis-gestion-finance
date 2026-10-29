package api

import (
	"net/http/httptest"
	"testing"
)

func TestIntelligenceMutationTriggerOnlySelectsRelevantCommits(t *testing.T) {
	cases := []struct{ method, path, want string }{
		{"POST", "/api/statements", "statement_import"},
		{"POST", "/api/statements/previews", ""},
		{"DELETE", "/api/statements/previews/token", ""},
		{"POST", "/api/statements/previews/token/confirm", "statement_import"},
		{"PATCH", "/api/transactions/42", "transaction_update"},
		{"PATCH", "/api/statements/42/validation", "statement_validation"},
		{"POST", "/api/intel/discoveries/id/feedback", "discovery_feedback"},
		{"PATCH", "/api/domains/7", "category_change"},
		{"DELETE", "/api/reimbursements/3/allocations/5", "reimbursement_change"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if got := intelligenceMutationTrigger(req); got != tc.want {
			t.Errorf("%s %s trigger=%q want %q", tc.method, tc.path, got, tc.want)
		}
	}
}
