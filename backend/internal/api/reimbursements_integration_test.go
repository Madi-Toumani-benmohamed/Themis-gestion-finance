package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"
	database "themis/internal/db"
)

func reimbursementTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("reimbursement_api_%d", time.Now().UnixNano())
	if _, err = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	t.Cleanup(func() {
		db.Close()
		admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`)
		admin.Close()
	})
	if err = database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(id,username,password_hash,role) VALUES(1,'REFUND OWNER TEST','TEST','admin'),(2,'OTHER REFUND TEST','TEST','user');
	 INSERT INTO accounts(id,owner_id,name,kind,tracking_mode) VALUES(11,1,'Refund TEST','courant','statements'),(21,2,'Foreign TEST','courant','statements');
	 INSERT INTO domains(id,owner_id,name,color,is_transfer,parent_id) VALUES
	 (9001,1,'Restaurants TEST','#123456',false,NULL),(9002,1,'Transfer TEST','#123456',true,NULL),(9003,1,'Child transfer TEST','#123456',false,9002),
	 (9004,1,'Savings TEST','#123456',false,NULL),(9201,2,'Secret domain TEST','#123456',false,NULL);
	 INSERT INTO statements(id,owner_id,account_id,filename,period_start,period_end,opening_balance,closing_balance,validated) VALUES
	 (101,1,11,'Synthetic expenses TEST','2025-12-01','2025-12-31',1000,773,true),
	 (102,1,11,'Synthetic credits TEST','2026-01-01','2026-01-31',0,232.01,true),
	 (103,1,11,'Synthetic tiny TEST','2026-02-01','2026-02-28',10,5,true),
	 (201,2,21,'Private synthetic TEST','2025-12-01','2025-12-31',1000,940,true);
	 INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,display_label,merchant,amount,domain_id,hash,label_kind) VALUES
	 (1,101,1,'2025-12-10','TEST RESTAURANT 60','Restaurant TEST','restaurant',-60,9001,'refund-expense-1',''),
	 (2,102,1,'2026-01-10','TEST FRIEND REFUND 30','Friend TEST','friend',30,9001,'refund-credit-2',''),
	 (3,101,1,'2025-12-11','TEST GROUP EXPENSE 120','Group expense TEST','restaurant',-120,9001,'refund-expense-3',''),
	 (4,102,1,'2026-01-12','TEST REFUND 40','Refund TEST','friend',40,9001,'refund-credit-4',''),
	 (5,102,1,'2026-01-13','TEST REFUND 35','Refund TEST','friend',35,9001,'refund-credit-5',''),
	 (6,101,1,'2025-12-12','TEST PURCHASE 10','Purchase TEST','shop',-10,9001,'refund-expense-6',''),
	 (7,102,1,'2026-01-14','TEST BATCH REFUND 100','Batch refund TEST','friend',100,9001,'refund-credit-7',''),
	 (8,103,1,'2026-02-10','TEST PENNY EXPENSE','Penny TEST','shop',-0.01,9001,'refund-expense-8',''),
	 (9,102,1,'2026-01-15','TEST PENNY REFUND','Penny refund TEST','friend',0.01,9001,'refund-credit-9',''),
	 (10,101,1,'2025-12-15','TEST TRANSFER CHILD EXPENSE','Transfer child TEST','own',-22,9003,'refund-transfer-child',''),
	 (11,102,1,'2026-01-16','TEST TRANSFER CREDIT','Transfer TEST','own',22,9002,'refund-transfer-credit',''),
	 (12,201,2,'2025-12-10','SECRET EXPENSE','SECRET DISPLAY','SECRET MERCHANT',-60,9201,'refund-foreign-expense',''),
	 (13,201,2,'2025-12-11','SECRET CREDIT','SECRET DISPLAY','SECRET MERCHANT',30,9201,'refund-foreign-credit',''),
	 (14,101,1,'2025-12-20','TEST EXTRA EXPENSE','Extra TEST','shop',-15,9001,'refund-expense-14',''),
	 (15,103,1,'2026-02-12','TEST SAVINGS DEBIT','Savings debit TEST','bank',-5,9004,'refund-savings-debit','savings'),
	 (16,102,1,'2026-01-17','TEST SAVINGS CREDIT','Savings credit TEST','bank',5,9001,'refund-savings-credit','savings'),
	 (17,103,1,'2026-02-13','TEST ZERO OPERATION','Zero TEST','bank',0,9001,'refund-zero','');`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func reimbursementRequest(s *Server, handler http.HandlerFunc, method, path, body string, owner int64, key string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: owner, Role: "user"}))
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		if part == "reimbursements" && i+1 < len(parts) && parts[i+1] != "eligible-expenses" {
			r.SetPathValue("id", parts[i+1])
		}
		if part == "allocations" && i+1 < len(parts) {
			r.SetPathValue("allocation_id", parts[i+1])
		}
	}
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

func readReimbursementResponse(t *testing.T, w *httptest.ResponseRecorder) reimbursementCaseView {
	t.Helper()
	var out reimbursementCaseView
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid response %s: %v", w.Body, err)
	}
	return out
}

func TestReimbursementManualAllocationsAuditAndOwnerIsolationPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	var originalAmount, originalDate, originalLabel, originalHash, originalStatementNet string
	if err := db.QueryRow(`SELECT amount::text,tx_date::text,label,hash FROM transactions WHERE id=1`).Scan(&originalAmount, &originalDate, &originalLabel, &originalHash); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COALESCE(sum(amount),0)::text FROM transactions WHERE owner_id=1 AND statement_id=101`).Scan(&originalStatementNet); err != nil {
		t.Fatal(err)
	}
	create := func(credit int, kind string) reimbursementCaseView {
		t.Helper()
		w := reimbursementRequest(s, s.createReimbursement, "POST", "/api/reimbursements", fmt.Sprintf(`{"credit_transaction_id":%d,"kind":%q}`, credit, kind), 1, "")
		if w.Code != 201 {
			t.Fatalf("create case %d: %d %s", credit, w.Code, w.Body)
		}
		return readReimbursementResponse(t, w)
	}
	allocate := func(caseID, expense int64, cents int64, key string, owner int64) *httptest.ResponseRecorder {
		t.Helper()
		path := fmt.Sprintf("/api/reimbursements/%d/allocations", caseID)
		body := fmt.Sprintf(`{"expense_transaction_id":%d,"amount_cents":%d}`, expense, cents)
		return reimbursementRequest(s, s.createReimbursementAllocation, "POST", path, body, owner, key)
	}

	partial := create(2, "personal_expense")
	if partial.Status != "unmatched" || partial.UnallocatedCents != 3000 || partial.Credit.AmountCents != 3000 {
		t.Fatalf("initial case: %+v", partial)
	}
	// Repeating the same natural-key qualification is idempotent.
	replay := reimbursementRequest(s, s.createReimbursement, "POST", "/api/reimbursements", `{"credit_transaction_id":2,"kind":"personal_expense"}`, 1, "")
	if replay.Code != 200 || strings.Count(replay.Body.String(), `"id":`) < 1 {
		t.Fatalf("case replay created a duplicate: %d %s", replay.Code, replay.Body)
	}
	if err := db.QueryRow(`SELECT count(*) FROM reimbursement_cases WHERE credit_transaction_id=2 AND owner_id=1`).Scan(new(int)); err != nil {
		t.Fatal(err)
	}
	tooMuch := allocate(partial.ID, 1, 3001, "expense-over-limit-request-0001", 1)
	if tooMuch.Code != 409 {
		t.Fatalf("over-available allocation accepted: %d %s", tooMuch.Code, tooMuch.Body)
	}
	first := allocate(partial.ID, 1, 1500, "expense-partial-allocation-0001", 1)
	if first.Code != 201 {
		t.Fatalf("partial allocation: %d %s", first.Code, first.Body)
	}
	firstView := readReimbursementResponse(t, first)
	if firstView.Status != "partial" || firstView.AllocatedCents != 1500 || firstView.UnallocatedCents != 1500 || len(firstView.Allocations) != 1 {
		t.Fatalf("partial amounts not exact: %+v", firstView)
	}
	eligible := reimbursementRequest(s, s.eligibleReimbursementExpenses, "GET", fmt.Sprintf("/api/reimbursements/eligible-expenses?case_id=%d", partial.ID), "", 1, "")
	if eligible.Code != 200 {
		t.Fatalf("eligible expense list: %d %s", eligible.Code, eligible.Body)
	}
	var eligibleResponse struct {
		Expenses []eligibleReimbursementExpense `json:"expenses"`
	}
	if err := json.Unmarshal(eligible.Body.Bytes(), &eligibleResponse); err != nil || len(eligibleResponse.Expenses) == 0 {
		t.Fatalf("eligible expenses missing: %s %v", eligible.Body, err)
	}
	foundRemaining := false
	for _, expense := range eligibleResponse.Expenses {
		if expense.TransactionID == 1 && expense.RemainingCents == 4500 {
			foundRemaining = true
		}
	}
	if !foundRemaining {
		t.Fatalf("remaining amount not computed exactly: %+v", eligibleResponse.Expenses)
	}
	// Exact replay returns the existing allocation. Reusing the key differently conflicts.
	replayAlloc := allocate(partial.ID, 1, 1500, "expense-partial-allocation-0001", 1)
	if replayAlloc.Code != 200 {
		t.Fatalf("allocation replay: %d %s", replayAlloc.Code, replayAlloc.Body)
	}
	keyConflict := allocate(partial.ID, 1, 1499, "expense-partial-allocation-0001", 1)
	if keyConflict.Code != 409 {
		t.Fatalf("idempotency key reused for different request: %d", keyConflict.Code)
	}
	remainder := allocate(partial.ID, 1, 1500, "expense-final-allocation-0002", 1)
	if remainder.Code != 201 || readReimbursementResponse(t, remainder).Status != "allocated" {
		t.Fatalf("complete reimbursement: %d %s", remainder.Code, remainder.Body)
	}
	if excessExpense := allocate(partial.ID, 1, 1, "expense-excess-target-000003", 1); excessExpense.Code != 409 {
		t.Fatalf("fully reimbursed expense accepted extra amount: %d", excessExpense.Code)
	}
	allocationID := firstView.Allocations[0].ID
	voidPath := fmt.Sprintf("/api/reimbursements/%d/allocations/%d", partial.ID, allocationID)
	void := reimbursementRequest(s, s.voidReimbursementAllocation, "DELETE", voidPath, `{"reason":"Correction manuelle TEST"}`, 1, "")
	if void.Code != 204 {
		t.Fatalf("allocation revoke: %d %s", void.Code, void.Body)
	}
	voidReplay := reimbursementRequest(s, s.voidReimbursementAllocation, "DELETE", voidPath, `{"reason":"Correction manuelle TEST"}`, 1, "")
	if voidReplay.Code != 204 {
		t.Fatalf("allocation revoke replay: %d", voidReplay.Code)
	}
	var eventCount int
	if err := db.QueryRow(`SELECT count(*) FROM reimbursement_history WHERE owner_id=1 AND case_id=$1 AND event_type='allocation_voided'`, partial.ID).Scan(&eventCount); err != nil || eventCount != 1 {
		t.Fatalf("revoke history count=%d err=%v", eventCount, err)
	}
	replacement := allocate(partial.ID, 1, 1500, "replacement-allocation-after-revoke-001", 1)
	if replacement.Code != 201 || readReimbursementResponse(t, replacement).Status != "allocated" {
		t.Fatalf("traced allocation modification: %d %s", replacement.Code, replacement.Body)
	}

	// Two credits compensate one expense; one credit is split across two expenses.
	c40 := create(4, "personal_expense")
	c35 := create(5, "merchant_refund")
	if got := allocate(c40.ID, 3, 4000, "multi-credit-allocation-0000001", 1); got.Code != 201 {
		t.Fatalf("first of multiple credits: %d %s", got.Code, got.Body)
	}
	if got := allocate(c35.ID, 3, 3500, "multi-credit-allocation-0000002", 1); got.Code != 201 {
		t.Fatalf("second of multiple credits: %d %s", got.Code, got.Body)
	}
	batch := create(7, "personal_expense")
	if got := allocate(batch.ID, 6, 1000, "one-credit-many-expenses-00001", 1); got.Code != 201 {
		t.Fatalf("batch first target: %d %s", got.Code, got.Body)
	}
	if got := allocate(batch.ID, 14, 1500, "one-credit-many-expenses-00002", 1); got.Code != 201 {
		t.Fatalf("batch second target: %d %s", got.Code, got.Body)
	}
	if detail := reimbursementRequest(s, s.getReimbursement, "GET", fmt.Sprintf("/api/reimbursements/%d", batch.ID), "", 1, ""); detail.Code != 200 || readReimbursementResponse(t, detail).AllocatedCents != 2500 {
		t.Fatalf("N:M dossier detail: %d %s", detail.Code, detail.Body)
	}
	if excessiveTarget := allocate(batch.ID, 3, 4501, "target-limit-across-cases-0001", 1); excessiveTarget.Code != 409 {
		t.Fatalf("aggregate expense limit ignored: %d %s", excessiveTarget.Code, excessiveTarget.Body)
	}

	// Transfer, savings, wrong-sign, absent and cross-owner transactions are rejected.
	for _, credit := range []int{1, 11, 16, 17, 99999, 13} {
		w := reimbursementRequest(s, s.createReimbursement, "POST", "/api/reimbursements", fmt.Sprintf(`{"credit_transaction_id":%d,"kind":"personal_expense"}`, credit), 1, "")
		if w.Code != 400 && w.Code != 404 {
			t.Errorf("ineligible credit %d: %d %s", credit, w.Code, w.Body)
		}
	}
	for i, expense := range []int64{2, 10, 15, 17, 12, 99998} {
		w := allocate(batch.ID, expense, 1, fmt.Sprintf("invalid-target-request-%08d", i), 1)
		if w.Code != 400 && w.Code != 404 {
			t.Errorf("ineligible expense %d: %d %s", expense, w.Code, w.Body)
		}
	}
	foreign := reimbursementRequest(s, s.getReimbursement, "GET", fmt.Sprintf("/api/reimbursements/%d", batch.ID), "", 2, "")
	if foreign.Code != 404 {
		t.Fatalf("cross-owner case read: %d", foreign.Code)
	}
	unauthReq := httptest.NewRequest("GET", "/api/reimbursements", nil)
	unauthW := httptest.NewRecorder()
	s.Router().ServeHTTP(unauthW, unauthReq)
	if unauthW.Code != 401 {
		t.Fatalf("unauthenticated endpoint: %d", unauthW.Code)
	}

	history := reimbursementRequest(s, s.reimbursementHistory, "GET", fmt.Sprintf("/api/reimbursements/%d/history", partial.ID), "", 1, "")
	if history.Code != 200 || !strings.Contains(history.Body.String(), "allocation_voided") {
		t.Fatalf("history missing revocation: %d %s", history.Code, history.Body)
	}

	voidCase := reimbursementRequest(s, s.voidReimbursement, "POST", fmt.Sprintf("/api/reimbursements/%d/void", batch.ID), `{"reason":"Annulation du dossier TEST"}`, 1, "")
	if voidCase.Code != 204 {
		t.Fatalf("case void: %d %s", voidCase.Code, voidCase.Body)
	}
	voidCaseAgain := reimbursementRequest(s, s.voidReimbursement, "POST", fmt.Sprintf("/api/reimbursements/%d/void", batch.ID), `{"reason":"Annulation du dossier TEST"}`, 1, "")
	if voidCaseAgain.Code != 204 {
		t.Fatalf("case void replay: %d", voidCaseAgain.Code)
	}
	var voidEvents int
	if err := db.QueryRow(`SELECT count(*) FROM reimbursement_history WHERE owner_id=1 AND case_id=$1 AND event_type='case_voided'`, batch.ID).Scan(&voidEvents); err != nil || voidEvents != 1 {
		t.Fatalf("case cancellation history count=%d err=%v", voidEvents, err)
	}
	var afterAmount, afterDate, afterLabel, afterHash, afterStatementNet string
	if err := db.QueryRow(`SELECT amount::text,tx_date::text,label,hash FROM transactions WHERE id=1`).Scan(&afterAmount, &afterDate, &afterLabel, &afterHash); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COALESCE(sum(amount),0)::text FROM transactions WHERE owner_id=1 AND statement_id=101`).Scan(&afterStatementNet); err != nil {
		t.Fatal(err)
	}
	if originalAmount != afterAmount || originalDate != afterDate || originalLabel != afterLabel || originalHash != afterHash || originalStatementNet != afterStatementNet {
		t.Fatalf("bank source or reconciliation changed: original=%s/%s/%s/%s net=%s after=%s/%s/%s/%s net=%s", originalAmount, originalDate, originalLabel, originalHash, originalStatementNet, afterAmount, afterDate, afterLabel, afterHash, afterStatementNet)
	}
}

func TestReimbursementConcurrencyRollbackAndStatementDeletionPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	// A one-cent target shared by two separately qualified credits is serialized.
	for _, credit := range []int{9} {
		w := reimbursementRequest(s, s.createReimbursement, "POST", "/api/reimbursements", fmt.Sprintf(`{"credit_transaction_id":%d,"kind":"personal_expense"}`, credit), 1, "")
		if w.Code != 201 {
			t.Fatalf("penny case: %d %s", w.Code, w.Body)
		}
	}
	// Create a second synthetic one-cent credit so both requests contend for one target cent.
	if _, err := db.Exec(`INSERT INTO transactions(id,statement_id,owner_id,tx_date,label,display_label,amount,domain_id,hash) VALUES(18,103,1,'2026-02-11','TEST SECOND PENNY','SECOND PENNY',0.01,9001,'refund-credit-18')`); err != nil {
		t.Fatal(err)
	}
	w := reimbursementRequest(s, s.createReimbursement, "POST", "/api/reimbursements", `{"credit_transaction_id":18,"kind":"personal_expense"}`, 1, "")
	if w.Code != 201 {
		t.Fatalf("second penny case: %d %s", w.Code, w.Body)
	}
	var cases []int64
	caseRows, err := db.QueryContext(context.Background(), `SELECT id FROM reimbursement_cases WHERE owner_id=1 AND credit_transaction_id IN (9,18) ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	for caseRows.Next() {
		var caseID int64
		if err = caseRows.Scan(&caseID); err != nil {
			break
		}
		cases = append(cases, caseID)
	}
	if err == nil {
		err = caseRows.Err()
	}
	caseRows.Close()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 2 {
		t.Fatalf("penny cases: %v", cases)
	}
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i, caseID := range cases {
		wg.Add(1)
		go func(i int, caseID int64) {
			defer wg.Done()
			key := fmt.Sprintf("parallel-penny-request-%08d", i)
			response := reimbursementRequest(s, s.createReimbursementAllocation, "POST", fmt.Sprintf("/api/reimbursements/%d/allocations", caseID), `{"expense_transaction_id":8,"amount_cents":1}`, 1, key)
			results <- response.Code
		}(i, caseID)
	}
	wg.Wait()
	close(results)
	created, conflicts := 0, 0
	for code := range results {
		if code == 201 {
			created++
		}
		if code == 409 {
			conflicts++
		}
	}
	if created != 1 || conflicts != 1 {
		t.Fatalf("concurrent final cent: created=%d conflicts=%d", created, conflicts)
	}
	var allocated int64
	if err := db.QueryRow(`SELECT COALESCE(sum(amount_cents),0) FROM reimbursement_allocations WHERE owner_id=1 AND expense_transaction_id=8 AND voided_at IS NULL`).Scan(&allocated); err != nil || allocated != 1 {
		t.Fatalf("concurrent sum=%d err=%v", allocated, err)
	}

	// Any failure to append the audit event rolls the dossier creation back.
	if _, err := db.Exec(`CREATE FUNCTION fail_reimbursement_history() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.note='FAIL HISTORY TEST' THEN RAISE EXCEPTION 'injected'; END IF; RETURN NEW; END $$;
	 CREATE TRIGGER fail_reimbursement_history BEFORE INSERT ON reimbursement_history FOR EACH ROW EXECUTE FUNCTION fail_reimbursement_history()`); err != nil {
		t.Fatal(err)
	}
	failed := reimbursementRequest(s, s.createReimbursement, "POST", "/api/reimbursements", `{"credit_transaction_id":2,"kind":"personal_expense","note":"FAIL HISTORY TEST"}`, 1, "")
	if failed.Code != 500 {
		t.Fatalf("history failure hidden: %d %s", failed.Code, failed.Body)
	}
	var partialWrite int
	if err := db.QueryRow(`SELECT count(*) FROM reimbursement_cases WHERE credit_transaction_id=2 AND owner_id=1`).Scan(&partialWrite); err != nil || partialWrite != 0 {
		t.Fatalf("case survived failed history: %d %v", partialWrite, err)
	}
	if _, err := db.Exec(`DROP TRIGGER fail_reimbursement_history ON reimbursement_history; DROP FUNCTION fail_reimbursement_history()`); err != nil {
		t.Fatal(err)
	}

	// Statement deletion is blocked while the source/target relationship is active.
	debit := reimbursementRequest(s, s.createReimbursement, "POST", "/api/reimbursements", `{"credit_transaction_id":2,"kind":"personal_expense"}`, 1, "")
	if debit.Code != 201 {
		t.Fatalf("case after rollback: %d %s", debit.Code, debit.Body)
	}
	if allocated := reimbursementRequest(s, s.createReimbursementAllocation, "POST", fmt.Sprintf("/api/reimbursements/%d/allocations", readReimbursementResponse(t, debit).ID), `{"expense_transaction_id":1,"amount_cents":100}`, 1, "delete-check-active-allocation-0001"); allocated.Code != 201 {
		t.Fatalf("allocation: %d %s", allocated.Code, allocated.Body)
	}
	deleteReq := httptest.NewRequest("DELETE", "/api/statements/101", nil).WithContext(context.WithValue(context.Background(), ctxUserKey, &authUser{ID: 1, Role: "user"}))
	deleteReq.SetPathValue("id", "101")
	deleteW := httptest.NewRecorder()
	s.deleteStatement(deleteW, deleteReq)
	if deleteW.Code != 409 {
		t.Fatalf("active reimbursement statement deletion: %d %s", deleteW.Code, deleteW.Body)
	}

	// Revoke/cancel explicitly, then deletion can proceed; history keeps snapshots
	// while the composite FK nulls only the transaction reference.
	caseView := readReimbursementResponse(t, debit)
	void := reimbursementRequest(s, s.voidReimbursement, "POST", fmt.Sprintf("/api/reimbursements/%d/void", caseView.ID), `{"reason":"Retrait avant suppression du relevé TEST"}`, 1, "")
	if void.Code != 204 {
		t.Fatalf("void before delete: %d %s", void.Code, void.Body)
	}
	deleteReq = httptest.NewRequest("DELETE", "/api/statements/101", nil).WithContext(context.WithValue(context.Background(), ctxUserKey, &authUser{ID: 1, Role: "user"}))
	deleteReq.SetPathValue("id", "101")
	deleteW = httptest.NewRecorder()
	s.deleteStatement(deleteW, deleteReq)
	if deleteW.Code != 204 {
		t.Fatalf("statement deletion after revoke: %d %s", deleteW.Code, deleteW.Body)
	}
	var txRef sql.NullInt64
	var snapshot string
	if err := db.QueryRow(`SELECT a.expense_transaction_id,a.expense_snapshot::text FROM reimbursement_allocations a JOIN reimbursement_cases c ON c.id=a.case_id WHERE c.id=$1`, caseView.ID).Scan(&txRef, &snapshot); err != nil || txRef.Valid || !strings.Contains(snapshot, "TEST RESTAURANT 60") {
		t.Fatalf("historical allocation reference/snapshot: %v %s %v", txRef, snapshot, err)
	}

	// Original transaction records and statement cash totals are not rewritten by linking.
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM transactions WHERE owner_id=1 AND id IN (2,4,5,7,8,9,11,15,16,17,18)`).Scan(&count); err != nil || count != 11 {
		t.Fatalf("unrelated raw transactions changed: count=%d err=%v", count, err)
	}
}
