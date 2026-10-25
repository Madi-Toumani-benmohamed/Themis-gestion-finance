package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"themis/internal/models"
	"themis/internal/money"
)

var operationCount = regexp.MustCompile(`(?i)nombre\s+(?:total\s+)?d['’]op[ée]rations\s*:?\s*(\d+)\b`)

func reconcileImport(res *models.ParseResult, rows []Row) {
	r := res.Report
	if r == nil {
		r = &models.ImportReport{Candidates: []models.ImportCandidate{}, Issues: []string{}}
		res.Report = r
	}
	r.Parsed = len(res.Transactions)
	r.Detected = len(r.Candidates)
	r.TraceabilityControl = "passed"
	r.Rejected = 0
	if r.Detected < r.Parsed {
		r.Issues = appendUnique(r.Issues, "candidate_count_mismatch")
		r.TraceabilityControl = "failed"
	} else {
		r.Rejected = r.Detected - r.Parsed
	}
	r.CountControl = "not_applicable"
	r.TotalsControl = "unverifiable"
	r.BalanceControl = "unverifiable"
	r.Completeness = "unknown"
	for _, row := range rows {
		text := row.Text()
		if m := operationCount.FindStringSubmatch(text); m != nil {
			n, e := strconv.Atoi(m[1])
			if e != nil {
				r.Issues = appendUnique(r.Issues, "invalid_declared_count")
			} else if r.ExpectedCount != nil && *r.ExpectedCount != n {
				r.Issues = appendUnique(r.Issues, "conflicting_declared_count")
			} else {
				r.ExpectedCount = &n
			}
		}
		if strings.Contains(strings.ToUpper(removeAccents(text)), "TOTAL DES OPERATIONS") {
			var amounts []int64
			for _, w := range row.Words {
				s := normalizeAmountStr(w.S)
				if reAmount.MatchString(s) {
					if v, e := parseAmount(s); e == nil {
						if c, e := money.FromLegacy(v); e == nil {
							amounts = append(amounts, int64(c))
						}
					}
				}
			}
			if len(amounts) == 2 {
				d, c := amounts[0], amounts[1]
				if (r.DeclaredDebits != nil && *r.DeclaredDebits != d) || (r.DeclaredCredits != nil && *r.DeclaredCredits != c) {
					r.Issues = appendUnique(r.Issues, "conflicting_declared_totals")
				} else {
					r.DeclaredDebits = &d
					r.DeclaredCredits = &c
				}
			}
		}
	}
	r.Debits = 0
	r.Credits = 0
	for _, t := range res.Transactions {
		if t.SourcePage < 1 || t.SourceLine < 1 {
			r.TraceabilityControl = "failed"
		}
		if (res.PeriodStart != "" && t.Date < res.PeriodStart) || (res.PeriodEnd != "" && t.Date > res.PeriodEnd) {
			r.RequiresReview = true
			r.Warnings = appendUnique(r.Warnings, "operation_date_outside_period")
		}
		c, e := money.FromLegacy(t.Amount)
		if e != nil {
			r.Issues = append(r.Issues, "invalid_amount")
			continue
		}
		if c < 0 {
			r.Debits -= int64(c)
		} else {
			r.Credits += int64(c)
		}
		if t.AltAmount != nil {
			r.Issues = appendUnique(r.Issues, "ambiguous_ocr_amount")
		}
	}
	if r.ExpectedCount != nil {
		r.CountControl = "passed"
		if *r.ExpectedCount != r.Parsed {
			r.CountControl = "failed"
		}
	}
	if r.DeclaredDebits != nil && r.DeclaredCredits != nil {
		r.TotalsControl = "passed"
		if *r.DeclaredDebits != r.Debits || *r.DeclaredCredits != r.Credits {
			r.TotalsControl = "failed"
		}
	}
	if res.OpeningBalance != nil && res.ClosingBalance != nil {
		o, e1 := money.FromLegacy(*res.OpeningBalance)
		c, e2 := money.FromLegacy(*res.ClosingBalance)
		if e1 == nil && e2 == nil {
			diff := int64(c-o) - r.Credits + r.Debits
			r.BalanceDifference = &diff
			r.BalanceControl = "passed"
			if diff != 0 {
				r.BalanceControl = "failed"
			}
		}
	}
	if r.Rejected > 0 || len(r.Issues) > 0 || r.CountControl == "failed" || r.TotalsControl == "failed" || r.BalanceControl == "failed" || r.TraceabilityControl == "failed" {
		r.Completeness = "failed"
	}
	if r.ExpectedCount != nil && r.CountControl == "passed" && r.TotalsControl == "passed" && r.BalanceControl == "passed" && r.TraceabilityControl == "passed" && r.Completeness != "failed" {
		r.Completeness = "verified"
	}
	// A balanced net alone cannot validate a missing debit/credit pair.
	res.Validated = r.BalanceControl == "passed" && r.TotalsControl == "passed" && r.TraceabilityControl == "passed" && r.Rejected == 0 && len(r.Issues) == 0 && r.Completeness != "failed" && !r.RequiresReview
}

func appendUnique(items []string, value string) []string {
	for _, s := range items {
		if s == value {
			return items
		}
	}
	return append(items, value)
}

func documentIdentity(res *models.ParseResult, data []byte) {
	h := sha256.Sum256(data)
	res.DocumentHash = hex.EncodeToString(h[:])
}

// SourceHash distinguishes occurrences inside a document. Account scoping is
// added at persistence. The historical business hash is only a similarity hint.
func SourceHash(document string, index int, account int64) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("source-v1|%s|%d|%d", document, index, account)))
	return hex.EncodeToString(h[:])
}

// RefreshImportReport recalculates every control from the candidate ledger and
// transaction values, retaining only independent totals read from the source.
func RefreshImportReport(res *models.ParseResult) { reconcileImport(res, nil) }
