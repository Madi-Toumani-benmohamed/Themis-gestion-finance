package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	database "themis/internal/db"
)

func labRequest(s *Server, method, path, body string, owner int64) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: owner}))
	parts := strings.Split(path, "/")
	if len(parts) > 3 {
		r.SetPathValue("id", parts[3])
	}
	w := httptest.NewRecorder()
	switch {
	case path == "/api/laboratory" && method == "GET":
		s.laboratoryList(w, r)
	case method == "POST" || method == "PUT":
		s.laboratorySave(w, r)
	case method == "DELETE":
		s.laboratoryDelete(w, r)
	default:
		s.laboratoryGet(w, r)
	}
	return w
}
func labInput() laboratoryInput {
	return laboratoryInput{ClientKey: "d38fde4d-509a-4c38-9080-b5e3b03c1025", Title: "Feuille TEST", Type: "workbook", SchemaVersion: 1, Content: laboratoryContent{Rows: 20, Columns: 8, Cells: map[string]laboratoryCell{"A1": {Input: "0.1", Format: "currency"}, "B1": {Input: "=A1+0.2", Format: "currency"}}, History: []laboratoryCalculation{}, SourceOperands: []string{}}}
}
func labJSON(t *testing.T, in laboratoryInput) string {
	t.Helper()
	data, e := json.Marshal(in)
	if e != nil {
		t.Fatal(e)
	}
	return string(data)
}
func TestLaboratoryOwnedRevisionsAndReplayPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	in := labInput()
	w := labRequest(s, "POST", "/api/laboratory", labJSON(t, in), 1)
	var first laboratoryDocument
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &first) != nil {
		t.Fatal(w.Code, w.Body)
	}
	path := fmt.Sprintf("/api/laboratory/%d", first.ID)
	w = labRequest(s, "POST", "/api/laboratory", labJSON(t, in), 1)
	var replay laboratoryDocument
	json.Unmarshal(w.Body.Bytes(), &replay)
	if w.Code != 200 || replay.ID != first.ID || !replay.Replayed {
		t.Fatal("POST replay", w.Code, w.Body)
	}
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		body := labJSON(t, in)
		if method == "PUT" {
			in.Revision = 1
			body = labJSON(t, in)
		}
		if method == "DELETE" {
			body = `{"revision":1}`
		}
		w = labRequest(s, method, path, body, 2)
		if w.Code != 404 {
			t.Fatal("foreign ID", method, w.Code, w.Body)
		}
	}
	w = labRequest(s, "GET", "/api/laboratory", "", 2)
	if w.Code != 200 || strings.Contains(w.Body.String(), "Feuille TEST") {
		t.Fatal("foreign list", w.Code, w.Body)
	}
	in.Revision = 1
	in.Title = "Renommée TEST"
	in.Content.Cells["A1"] = laboratoryCell{"<img src=x onerror=alert(1)>", "number"}
	w = labRequest(s, "PUT", path, labJSON(t, in), 1)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	w = labRequest(s, "PUT", path, labJSON(t, in), 1)
	if w.Code != 409 {
		t.Fatal("stale revision accepted", w.Code, w.Body)
	}
	w = labRequest(s, "DELETE", path, `{"revision":1}`, 1)
	if w.Code != 409 {
		t.Fatal("stale deletion", w.Code)
	}
	w = labRequest(s, "GET", path, "", 1)
	var current laboratoryDocument
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &current) != nil || current.Revision != 2 || current.Title != in.Title {
		t.Fatal("saved document lost", w.Body)
	}
	// Text is stored as data; no formula or HTML is ever executed in Go.
	var content laboratoryContent
	json.Unmarshal(current.Content, &content)
	if content.Cells["A1"].Input != in.Content.Cells["A1"].Input {
		t.Fatal("text changed")
	}
	r := httptest.NewRequest("GET", "/api/export", nil).WithContext(context.WithValue(context.Background(), ctxUserKey, &authUser{ID: 2}))
	export := httptest.NewRecorder()
	s.exportAll(export, r)
	if export.Code != 200 || strings.Contains(export.Body.String(), "Renommée TEST") {
		t.Fatal("document export isolation", export.Code, export.Body)
	}
	w = labRequest(s, "DELETE", path, `{"revision":2}`, 1)
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body)
	}
	w = labRequest(s, "GET", path, "", 1)
	if w.Code != 404 {
		t.Fatal("delete", w.Code)
	}
	if _, e := db.Exec(`DROP TABLE laboratory_documents`); e != nil {
		t.Fatal(e)
	}
	w = labRequest(s, "GET", "/api/laboratory", "", 1)
	if w.Code != 500 {
		t.Fatal("SQL failure represented as empty", w.Code)
	}
}
func TestLaboratoryInputLimits(t *testing.T) {
	changes := []func(*laboratoryInput){func(i *laboratoryInput) { i.SchemaVersion = 2 }, func(i *laboratoryInput) { i.Content.Rows = 101 }, func(i *laboratoryInput) { i.Content.Columns = 27 }, func(i *laboratoryInput) { i.Content.Cells["Z100"] = laboratoryCell{"1", "number"} }, func(i *laboratoryInput) { i.Content.Cells["A1"] = laboratoryCell{strings.Repeat("é", 513), "number"} }, func(i *laboratoryInput) { i.ClientKey = "not-a-uuid" }, func(i *laboratoryInput) { i.Content.History = nil }}
	for n, change := range changes {
		in := labInput()
		change(&in)
		w := labRequest(&Server{}, "POST", "/api/laboratory", labJSON(t, in), 1)
		if w.Code != 400 {
			t.Fatal("invalid input", n, w.Code)
		}
	}
	for _, body := range []string{`{"owner_id":2}`, `{} {}`, strings.Repeat("x", 270000)} {
		w := labRequest(&Server{}, "POST", "/api/laboratory", body, 1)
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
}
func TestFiscalAPIValidationNoPersistence(t *testing.T) {
	s := &Server{}
	body := `{"income_year":2025,"family":"single","children":0,"scope_confirmed":true,"basis":"final_taxable","income_period":"annual","observed_months":12,"income_cents":3000000,"deduction":"standard"}`
	call := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.fiscalSimulate(w, httptest.NewRequest("POST", "/api/fiscal/simulate", strings.NewReader(body)))
		return w
	}
	w := call(body)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"tax_cents":210400`) {
		t.Fatal(w.Code, w.Body)
	}
	w = call(strings.Replace(body, "2025", "2026", 1))
	if w.Code != 422 {
		t.Fatal("unknown fiscal year", w.Code, w.Body)
	}
	w = call(strings.Replace(body, "3000000", "1.25", 1))
	if w.Code != 400 {
		t.Fatal("fractional cents accepted")
	}
	s = &Server{DB: reimbursementTestDB(t)}
	for _, path := range []string{"/api/fiscal/rules", "/api/laboratory"} {
		w := httptest.NewRecorder()
		s.Router().ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 401 {
			t.Fatal("unauthenticated access", path, w.Code)
		}
	}
}

func TestLaboratoryMigrationAndConcurrentRevisionPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	rollback, e := os.ReadFile("../db/rollbacks/032_laboratory.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(string(rollback)); e != nil {
		t.Fatal("empty rollback", e)
	}
	for range 2 {
		if e = database.Migrate(db); e != nil {
			t.Fatal("replay migration", e)
		}
	}
	s := &Server{DB: db}
	in := labInput()
	w := labRequest(s, "POST", "/api/laboratory", labJSON(t, in), 1)
	var d laboratoryDocument
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &d) != nil {
		t.Fatal(w.Code, w.Body)
	}
	if _, e = db.Exec(string(rollback)); e == nil {
		t.Fatal("rollback must preserve existing documents")
	}
	in.Revision = d.Revision
	body := labJSON(t, in)
	path := fmt.Sprintf("/api/laboratory/%d", d.ID)
	codes := make(chan int, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() { defer wait.Done(); codes <- labRequest(s, "PUT", path, body, 1).Code }()
	}
	wait.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal("lost concurrent update", counts)
	}
	var amount string
	if e = db.QueryRow("SELECT amount::text FROM transactions WHERE id=1").Scan(&amount); e != nil || amount != "-60.00" && amount != "-60" {
		t.Fatal("bank source modified", amount, e)
	}
}
