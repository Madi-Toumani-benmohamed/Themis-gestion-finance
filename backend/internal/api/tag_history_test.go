package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestTagHistoryBoundsAndCalendar(t *testing.T) {
	for _, tc := range []struct{ query, from, to string }{{"month=2020-02&months=6", "2019-09-01", "2020-02-29"}, {"month=2020-02&months=24", "2018-03-01", "2020-02-29"}, {"month=2020-02&months=120", "2010-03-01", "2020-02-29"}, {"from=2018-06-01&to=2020-02-29", "2018-06-01", "2020-02-29"}} {
		from, to, err := analysisSeriesWindow(httptest.NewRequest("GET", "/api/analytics/tag-series?"+tc.query, nil))
		if err != nil || from != tc.from || to != tc.to {
			t.Fatalf("%s => %s / %s (%v)", tc.query, from, to, err)
		}
	}
	for _, query := range []string{"month=2020-02&months=601", "month=2020-02&from=2020-01-01", "months=24", "from=2999-01-01&to=2999-01-31", "month=2020-02&months=0", "month=2020-02&months=not-a-number", "month=0001-01&months=24", "from=1900-01-01&to=2020-01-01", "from=2020-03-01&to=2020-02-01", "from=2020-02-30&to=2020-03-01"} {
		if _, _, err := analysisSeriesWindow(httptest.NewRequest("GET", "/api/analytics/tag-series?"+query, nil)); err == nil {
			t.Fatalf("invalid accepted: %s", query)
		}
	}
	buckets := analysisSeriesBuckets("2019-12-12", "2020-03-21", "month")
	if len(buckets) != 4 || buckets[1] != "2020-01" || buckets[3] != "2020-03" {
		t.Fatal(buckets)
	}
	if buckets = analysisSeriesBuckets("2019-12-12", "2021-01-01", "year"); len(buckets) != 3 || buckets[1] != "2020" {
		t.Fatal(buckets)
	}
}
func TestTagHistoryLongWindowAndMissingMonthsPostgres(t *testing.T) {
	db := analysisTestDB(t)
	srv := &Server{DB: db}
	_, err := db.Exec(`INSERT INTO transactions(id,statement_id,owner_id,tx_date,amount,domain_id) VALUES(100,1,1,'2015-01-10',-1.23,1),(101,1,1,'2018-03-10',-2.34,1),(102,1,1,'2020-03-10',-999,1),(103,2,2,'2018-03-10',-9999,1);
 INSERT INTO transaction_tags VALUES(100,1),(101,1),(102,1),(103,1);`)
	if err != nil {
		t.Fatal(err)
	}
	call := func(query string) struct {
		Buckets []string `json:"buckets"`
		Series  []struct {
			Tag    string         `json:"tag"`
			Values []*json.Number `json:"values"`
		} `json:"series"`
		From string `json:"query_from"`
		To   string `json:"query_to"`
	} {
		r := httptest.NewRequest("GET", "/api/analytics/tag-series?tags=projet%20test&"+query, nil)
		r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, &authUser{ID: 1}))
		w := httptest.NewRecorder()
		srv.tagSeries(w, r)
		var out struct {
			Buckets []string `json:"buckets"`
			Series  []struct {
				Tag    string         `json:"tag"`
				Values []*json.Number `json:"values"`
			} `json:"series"`
			From string `json:"query_from"`
			To   string `json:"query_to"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil {
			t.Fatalf("%s => %d %s", query, w.Code, w.Body)
		}
		return out
	}
	long := call("month=2020-02&months=24")
	if len(long.Buckets) != 24 || long.From != "2018-03-01" || long.To != "2020-02-29" || len(long.Series) != 1 {
		t.Fatalf("bounds %+v", long)
	}
	if long.Series[0].Values[0] == nil || string(*long.Series[0].Values[0]) != "2.34" || long.Series[0].Values[1] != nil {
		t.Fatalf("gap lost: %+v", long.Series[0])
	}
	ten := call("month=2020-02&months=120")
	if len(ten.Buckets) != 120 {
		t.Fatal("ten-year calendar not returned")
	}
	var total int64
	for _, amount := range ten.Series[0].Values {
		if amount != nil {
			c, e := parseAnalysisAmount(string(*amount))
			if e != nil {
				t.Fatal(e)
			}
			total += int64(c)
		}
	}
	if total != 13260 {
		t.Fatalf("bounded history is not reconciled: %d", total)
	} // 1.23 + 2.34 + 30.03 + 99.00; no future, foreign owner or credits.
	custom := call("from=2018-03-01&to=2018-05-31")
	if len(custom.Buckets) != 3 || custom.Series[0].Values[1] != nil || custom.Series[0].Values[2] != nil {
		t.Fatal("custom gaps replaced by zeros")
	}
}
