package classificationaudit

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestSyntheticOverlapsTagsManualAndExactFlows(t *testing.T) {
	d := Dataset{
		Domains: []Domain{{ID: 1, Name: "Transport"}, {ID: 2, Name: "Alimentation"}, {ID: 3, Name: "Réserve", Transfer: true}, {ID: 4, Name: "Enfant", Parent: 3}},
		Tags:    []Tag{{ID: 1, Name: "synthetic-context"}, {ID: 2, Name: "TRANSPORT"}},
		Rules:   []Rule{{ID: 2, Domain: 1, Pattern: "UBER", Tags: "commun, trajet"}, {ID: 1, Domain: 2, Pattern: "UBER EATS", Tags: "commun, repas"}},
		Transactions: []Transaction{
			{ID: 1, Domain: 1, Source: "manual", Label: "UBER EATS synthetic-private-label", Merchant: "synthetic-private-merchant", Amount: "-10.01", Tags: []int64{1, 2}},
			{ID: 2, Source: "manual", Label: "UBER EATS", Amount: "20.02"},
			{ID: 3, Domain: 4, Kind: "savings", Source: "rule", Amount: "-0.01"},
			{ID: 4, Domain: 1, Kind: "savings", Source: "unknown", Amount: "0"},
		},
	}
	r, e := Analyze(d)
	if e != nil {
		t.Fatal(e)
	}
	if r.ConflictingTransactions != 2 || r.ManualDisagreements != 1 || r.ManualUnassignedMatches != 1 || r.LosingRuleAddsTags != 2 || r.RuleTagsRepeated != 1 {
		t.Fatalf("wrong conflict metrics: %+v", r)
	}
	if len(r.RulePairs) != 1 || r.RulePairs[0].Count != 2 || !r.RulePairs[0].Contradictory {
		t.Fatal(r.RulePairs)
	}
	if r.Rules[0].Domain != "domain-2" || r.Rules[0].FirstMatches != 2 || r.Rules[1].FirstMatches != 0 {
		t.Fatal(r.Rules)
	}
	if r.BankFlow != (Flow{"1002", "2002", "1000"}) || r.UnassignedFlow != (Flow{"0", "2002", "2002"}) {
		t.Fatal(r.BankFlow, r.UnassignedFlow)
	}
	if r.SavingsWithoutTransferFlag != 1 || r.TransferFlagWithoutSavings != 0 || r.DomainParents != 1 || r.DomainChildren != 1 {
		t.Fatal("scope counters")
	}
	if len(r.TagPairs) != 1 || !r.TagPairs[0].Systematic || !r.Tags[0].SingleMerchant || !r.Tags[1].DomainNameDuplicate {
		t.Fatal(r.Tags, r.TagPairs)
	}
	encoded, _ := json.Marshal(r)
	for _, secret := range []string{"synthetic-private", "UBER", "Alimentation", "synthetic-context", "repas"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("raw text leaked: %s", secret)
		}
	}
	if d.Rules[0].ID != 2 || d.Transactions[0].Domain != 1 || d.Transactions[1].Domain != 0 {
		t.Fatal("input modified")
	}
}

func TestRealMatcherLiteralRegexUnicodePriorityAndInvalidLinks(t *testing.T) {
	d := Dataset{Domains: []Domain{{ID: 1}}, Rules: []Rule{
		{ID: 4, Domain: 1, Pattern: "A_B%"}, {ID: 3, Domain: 1, Pattern: "re:^éé$"},
		{ID: 2, Domain: 1, Pattern: "re:["}, {ID: 1, Domain: 99, Pattern: "FOO"},
	}, Transactions: []Transaction{{ID: 1, Amount: "-1", Label: "AxxB test"}, {ID: 2, Amount: "1", Label: "A_B%"}, {ID: 3, Amount: "0", Label: "ÉÉ", Domain: 99, Tags: []int64{99}}}}
	r, e := Analyze(d)
	if e != nil {
		t.Fatal(e)
	}
	var matches, wildcards, invalid, regex int
	for _, m := range r.Rules {
		matches += m.Matches
		if m.LiteralSQLWildcard {
			wildcards++
		}
		if m.InvalidPattern {
			invalid++
		}
		if m.Regex {
			regex++
		}
	}
	if matches != 2 || wildcards != 1 || invalid != 1 || regex != 2 || r.InvalidDomainLinks != 1 || r.InvalidTagLinks != 1 {
		t.Fatalf("unexpected metrics: %+v", r)
	}
	// PostgreSQL length counts characters, not UTF-8 bytes; tie broken by ID.
	r, e = Analyze(Dataset{Domains: []Domain{{ID: 1}}, Rules: []Rule{{ID: 2, Domain: 1, Pattern: "éé"}, {ID: 1, Domain: 1, Pattern: "AB"}}})
	if e != nil || r.Rules[0].Ref != "rule-1" {
		t.Fatal(e)
	}
	if r.Rules[0].Priority != 1 {
		t.Fatal("priority")
	}
	// Both match: same character count, smaller ID wins, despite byte length.
	r, e = Analyze(Dataset{Domains: []Domain{{ID: 1}, {ID: 2}}, Rules: []Rule{{ID: 2, Domain: 2, Pattern: "éé"}, {ID: 1, Domain: 1, Pattern: "AB"}}, Transactions: []Transaction{{Amount: "0", Label: "AB éé"}}})
	if e != nil || r.Rules[0].Domain != "domain-1" || r.Rules[0].FirstMatches != 1 {
		t.Fatal(r.Rules, e)
	}
}

func TestEmptyMissingAndBoundedFailures(t *testing.T) {
	r, e := Analyze(Dataset{})
	if e != nil || r.Transactions != 0 || r.BankFlow.NetCents != "0" {
		t.Fatal(r, e)
	}
	for _, amount := range []string{"", "1.001", "NaN", "99999999999"} {
		if _, e = Analyze(Dataset{Transactions: []Transaction{{Amount: amount}}}); e == nil {
			t.Fatal("invalid amount accepted")
		}
	}
	if _, e = Analyze(Dataset{Transactions: make([]Transaction, MaxRows+1)}); e == nil {
		t.Fatal("size limit")
	}
	if _, e = Analyze(Dataset{Rules: make([]Rule, 101), Transactions: make([]Transaction, MaxRows-101)}); e == nil {
		t.Fatal("work limit")
	}
	r, e = Analyze(Dataset{Tags: []Tag{{ID: 1, Name: "x"}}, Transactions: []Transaction{{Amount: "0", Tags: []int64{1}}}})
	if e != nil || r.Tags[0].SingleMerchant || r.Tags[0].UnknownMerchantTransactions != 1 {
		t.Fatal(r.Tags, e)
	}
}

// A fake driver proves the SQL boundary without a database, credentials or
// schema setup. Any non-SELECT query, Exec, Commit or prepared statement fails.
type readDriver struct{ state *readState }
type readConnector struct{ state *readState }

func (c readConnector) Connect(context.Context) (driver.Conn, error) { return readConn{c.state}, nil }
func (c readConnector) Driver() driver.Driver                        { return readDriver{c.state} }

type readState struct {
	queries, rollbacks int
	options            driver.TxOptions
	fail               bool
	fixture            bool
}
type readConn struct{ state *readState }
type readTx struct{ state *readState }
type emptyRows struct{}

func (d readDriver) Open(string) (driver.Conn, error)  { return readConn{d.state}, nil }
func (c readConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("prepare forbidden") }
func (c readConn) Close() error                        { return nil }
func (c readConn) Begin() (driver.Tx, error)           { return nil, errors.New("untyped begin forbidden") }
func (c readConn) BeginTx(_ context.Context, o driver.TxOptions) (driver.Tx, error) {
	c.state.options = o
	return readTx{c.state}, nil
}
func (c readConn) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	if !c.state.options.ReadOnly || c.state.options.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) || !strings.HasPrefix(q, "SELECT ") || strings.Contains(q, ";") || len(args) != 1 || args[0].Value != int64(123) {
		return nil, errors.New("unsafe query")
	}
	c.state.queries++
	if c.state.fail {
		return nil, errors.New("private SQL detail must not escape")
	}
	if c.state.fixture {
		var values []driver.Value
		switch c.state.queries {
		case 1:
			values = []driver.Value{int64(10), int64(0), "synthetic-domain", false}
		case 2:
			values = []driver.Value{int64(20), "synthetic-tag"}
		case 3:
			values = []driver.Value{int64(30), int64(10), "SYNTHETIC", "synthetic-tag"}
		case 4:
			values = []driver.Value{int64(40), int64(10), "SYNTHETIC BANK DATA", "SYNTHETIC MERCHANT", "", "manual", "-10.01", "{20}"}
		}
		return &fixtureRows{values: values}, nil
	}
	return emptyRows{}, nil
}

type fixtureRows struct {
	values []driver.Value
	done   bool
}

func (r *fixtureRows) Columns() []string { return make([]string, len(r.values)) }
func (r *fixtureRows) Close() error      { return nil }
func (r *fixtureRows) Next(dst []driver.Value) error {
	if r.done {
		return io.EOF
	}
	copy(dst, r.values)
	r.done = true
	return nil
}
func (t readTx) Commit() error              { return errors.New("commit forbidden") }
func (t readTx) Rollback() error            { t.state.rollbacks++; return nil }
func (emptyRows) Columns() []string         { return []string{"unused"} }
func (emptyRows) Close() error              { return nil }
func (emptyRows) Next([]driver.Value) error { return io.EOF }

func TestReadOnlySnapshotOwnerAndErrorBoundary(t *testing.T) {
	for _, failure := range []bool{false, true} {
		s := &readState{fail: failure}
		db := sql.OpenDB(readConnector{s})
		var e error
		if _, e = Run(context.Background(), db, 0); e == nil || s.queries != 0 {
			t.Fatal("missing owner accepted")
		}
		r, e := Run(context.Background(), db, 123)
		if failure {
			if e == nil || strings.Contains(e.Error(), "private SQL") {
				t.Fatal(e)
			}
		} else if e != nil || r.Transactions != 0 || s.queries != 4 {
			t.Fatal(r, e, s)
		}
		if !s.options.ReadOnly || s.rollbacks != 1 {
			t.Fatal("snapshot not read-only/rolled back")
		}
		db.Close()
	}
}

func TestSyntheticSQLRowsAreScannedAndReconciled(t *testing.T) {
	s := &readState{fixture: true}
	db := sql.OpenDB(readConnector{s})
	defer db.Close()
	r, e := Run(context.Background(), db, 123)
	if e != nil {
		t.Fatal(e)
	}
	if r.Transactions != 1 || r.DomainTotal != 1 || r.TagUsed != 1 || r.Rules[0].Matches != 1 || r.BankFlow.DebitCents != "1001" || r.Domains[0].Flow != r.BankFlow || s.rollbacks != 1 {
		t.Fatal(r)
	}
}
