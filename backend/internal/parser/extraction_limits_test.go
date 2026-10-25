package parser

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakePoppler(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, "pdftotext"), []byte("#!/bin/sh\n"+script), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", dir)
}
func TestExtractionFailureIsSanitized(t *testing.T) {
	fakePoppler(t, "echo PRIVATE_BANK_DATA >&2; exit 1\n")
	_, e := ExtractRowsContext(context.Background(), []byte("TEST"))
	if e == nil || strings.Contains(e.Error(), "PRIVATE_BANK_DATA") {
		t.Fatal("extraction error missing or leaks document data")
	}
}
func TestExtractionCancellation(t *testing.T) {
	fakePoppler(t, "while :; do :; done\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := ExtractRowsContext(ctx, []byte("TEST"))
	if e == nil {
		t.Fatal("cancelled extraction succeeded")
	}
}
func TestExtractionOutputBound(t *testing.T) {
	var b limitedPDFBuffer
	if _, e := b.Write(make([]byte, 32*1024*1024+1)); e == nil || b.Len() != 0 {
		t.Fatal("unbounded extraction output")
	}
}
func TestUnknownColumnsNeverValidate(t *testing.T) {
	rows := sampleRows()[1:]
	res, e := ParseLocal(rows)
	if e != nil {
		t.Fatal(e)
	}
	if res.Validated || res.Report.Completeness != "failed" {
		t.Fatal("assumed debit signs cannot validate an import")
	}
}
func TestExplicitOperationCount(t *testing.T) {
	rows := sampleRows()
	rows = append(rows, row(10, w("Nombre d'opérations : 3", 50, 150)))
	res, e := ParseLocal(rows)
	if e != nil {
		t.Fatal(e)
	}
	if res.Report.ExpectedCount == nil || *res.Report.ExpectedCount != 3 || res.Report.CountControl != "failed" || res.Validated {
		t.Fatal("explicit count mismatch ignored")
	}
	rows[len(rows)-1].Words[0].S = "Nombre d'opérations : 2"
	res, e = ParseLocal(rows)
	if e != nil {
		t.Fatal(e)
	}
	if res.Report.CountControl != "passed" || res.Report.Completeness != "verified" {
		t.Fatal("independent count not used")
	}
}

func TestConflictingIndependentDeclarationsBlockImport(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rows  []Row
		issue string
	}{
		{
			name: "count",
			rows: []Row{
				row(12, w("Nombre d'opérations : 3", 50, 160)),
				row(11, w("Nombre d'opérations : 2", 50, 160)),
			},
			issue: "conflicting_declared_count",
		},
		{
			name: "totals",
			rows: []Row{
				row(12, w("Total des opérations", 50, 160), w("33,10", 405, 38), w("310,00", 488, 42)),
				row(11, w("Total des opérations", 50, 160), w("23,10", 405, 38), w("300,00", 488, 42)),
			},
			issue: "conflicting_declared_totals",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ParseLocal(append(sampleRows(), tc.rows...))
			if err != nil {
				t.Fatal(err)
			}
			if !Validate(res) || res.Validated || res.Report.Completeness != "failed" {
				t.Fatal("contradictory independent evidence accepted despite balanced transactions")
			}
			found := false
			for _, issue := range res.Report.Issues {
				found = found || issue == tc.issue
			}
			if !found {
				t.Fatalf("missing diagnostic %s", tc.issue)
			}
			RefreshImportReport(res)
			if res.Validated || res.Report.Completeness != "failed" {
				t.Fatal("refresh discarded the conflicting evidence before persistence")
			}
		})
	}
}

func TestRepeatedIdenticalDeclarationsRemainValid(t *testing.T) {
	rows := sampleRows()
	rows = append(rows, rows[4],
		row(12, w("Nombre d'opérations : 2", 50, 160)),
		row(11, w("Nombre d'opérations : 2", 50, 160)))
	res, err := ParseLocal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Validated || res.Report.Completeness != "verified" || len(res.Report.Issues) != 0 {
		t.Fatal("consistent repeated page totals should remain valid")
	}
}
