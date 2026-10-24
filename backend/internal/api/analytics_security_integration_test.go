package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"themis/internal/money"
)

func TestTagSeriesBoundSQLAndOwnerScopePostgres(t *testing.T) {
	db := qualityTestDB(t)
	_, err := db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,is_transfer bool);
 CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,tx_date date,amount numeric(12,2),domain_id bigint,label_kind text);
 CREATE TEMP TABLE tags(id bigint PRIMARY KEY,owner_id bigint,name text);
 CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint);
 INSERT INTO domains VALUES(1,1,null,false),(2,1,null,true),(3,1,2,false);
 INSERT INTO transactions VALUES(1,1,'2020-01-01',-10.01,1,''),(2,1,'2020-02-01',-20.02,1,''),(3,1,'2020-02-02',-100,3,''),(4,1,'2020-02-03',-50,1,'savings'),(5,2,'2020-01-01',-999,1,'');
 INSERT INTO tags VALUES(1,1,'food'),(2,1,'quote''tag'),(3,2,'SECRET');
 INSERT INTO transaction_tags VALUES(1,1),(2,1),(1,2),(3,1),(4,1),(1,3),(5,1);`)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{DB: db}
	for _, tc := range []struct {
		query   string
		buckets int
		total   float64
	}{
		{"tags=food&period=month", 2, 30.03}, {"tags=food&period=year", 1, 30.03}, {"tags=quote%27tag", 1, 10.01}, {"tags=SECRET", 0, 0}, {"tags=x%27%29%3BDELETE+FROM+transactions%3B--", 0, 0}, {"", 2, 40.04},
	} {
		req := httptest.NewRequest("GET", "/api/analytics/tag-series?"+tc.query, nil)
		req = req.WithContext(context.WithValue(req.Context(), ctxUserKey, &authUser{ID: 1}))
		response := httptest.NewRecorder()
		server.tagSeries(response, req)
		var report struct {
			Buckets []string `json:"buckets"`
			Series  []struct {
				Tag    string    `json:"tag"`
				Values []float64 `json:"values"`
			} `json:"series"`
		}
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &report) != nil || len(report.Buckets) != tc.buckets {
			t.Fatalf("tag series %s: %d %s", tc.query, response.Code, response.Body)
		}
		var cents int64
		for _, series := range report.Series {
			if series.Tag == "SECRET" {
				t.Fatal("foreign tag disclosed")
			}
			for _, value := range series.Values {
				exact, err := money.FromLegacy(value)
				if err != nil {
					t.Fatal(err)
				}
				cents += int64(exact)
			}
		}
		expected, err := money.FromLegacy(tc.total)
		if err != nil {
			t.Fatal(err)
		}
		if cents != int64(expected) {
			t.Fatalf("tag scope %s: %d", tc.query, cents)
		}
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM transactions`).Scan(&count); err != nil || count != 5 {
		t.Fatal("bound tag query changed sources")
	}
}
