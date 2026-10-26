package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAdminStatsExposeCategoryCountConsumedByFrontend(t *testing.T) {
	dsn := os.Getenv("THEMIS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("THEMIS_TEST_DATABASE_URL non défini")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TEMP TABLE statements(owner_id bigint);
CREATE TEMP TABLE transactions(owner_id bigint, domain_id bigint, tx_date date);
CREATE TEMP TABLE domains(owner_id bigint, parent_id bigint);
CREATE TEMP TABLE rules(owner_id bigint);
CREATE TEMP TABLE analyses(owner_id bigint);
CREATE TEMP TABLE budgets(owner_id bigint);
INSERT INTO domains VALUES (7,NULL),(7,NULL),(7,1),(8,NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/admin/stats", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: 7}))
	response := httptest.NewRecorder()
	(&Server{DB: db}).adminStats(response, req)
	if response.Code != 200 {
		t.Fatalf("status admin stats=%d", response.Code)
	}
	var stats map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &stats); err != nil {
		t.Fatal("réponse JSON admin invalide")
	}
	if stats["categories"] != float64(2) {
		t.Fatalf("le contrat frontend attend le nombre de catégories racines du propriétaire; categories=%v réponse=%v", stats["categories"], stats)
	}
}
