package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"themis/internal/models"
)

func TestImportAndReapplicationWinningRuleParityPostgres(t *testing.T) {
	db := reimbursementTestDB(t)
	s := &Server{DB: db}
	if _, e := db.Exec(`INSERT INTO rules(owner_id,pattern,domain_id,tags) VALUES(1,'UBER EATS',9001,'livraison'),(1,'UBER',9002,'vtc'); SELECT setval(pg_get_serial_sequence('transactions','id'),(SELECT max(id) FROM transactions))`); e != nil {
		t.Fatal(e)
	}
	res := &models.ParseResult{Parser: "local", PeriodStart: "2026-03-01", PeriodEnd: "2026-03-31", Transactions: []models.Transaction{{Date: "2026-03-10", Label: "UBER EATS SYNTHETIC IMPORT", Amount: -12.34, Hash: "synthetic-classification-plan-import"}}}
	inserted, _, statement, e := s.persist("SYNTHETIC PLAN.csv", res, 11, 1)
	if e != nil || inserted != 1 {
		t.Fatal(inserted, e)
	}
	if _, e = s.applyAllRulesForStatement(1, statement); e != nil {
		t.Fatal(e)
	}
	var domain int64
	var raw, date, amount, source, tags string
	e = db.QueryRow(`SELECT t.domain_id,t.label,t.tx_date::text,t.amount::text,t.classification_source,COALESCE((SELECT string_agg(tg.name,',' ORDER BY tg.name) FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id WHERE tt.transaction_id=t.id),'') FROM transactions t WHERE t.statement_id=$1 AND t.owner_id=1`, statement).Scan(&domain, &raw, &date, &amount, &source, &tags)
	if e != nil || domain != 9001 || raw != res.Transactions[0].Label || date != "2026-03-10" || amount != "-12.34" || source != "rule" || tags != "livraison" {
		t.Fatal(domain, raw, date, amount, source, tags, e)
	}
	// Existing historical labels and tags are not modified by the import scope.
	var n int
	if e = db.QueryRow(`SELECT count(*) FROM transaction_tags tt JOIN transactions t ON t.id=tt.transaction_id WHERE t.statement_id<>$1`, statement).Scan(&n); e != nil || n != 0 {
		t.Fatal("historical import side effect", e)
	}
}

func TestWinningRulePathsManualNullAndFinancialIntegrityPostgres(t *testing.T) {
	for _, mode := range []string{"global", "statement", "individual-winner", "individual-loser"} {
		t.Run(mode, func(t *testing.T) {
			db := qualityTestDB(t)
			_, err := db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,name text,color text,parent_id bigint,is_transfer bool DEFAULT false);
CREATE TEMP TABLE rules(id bigserial PRIMARY KEY,owner_id bigint,domain_id bigint,pattern text,tags text);
CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,statement_id bigint,label text,domain_id bigint,classification_source text,tx_date date,amount numeric(12,2),display_label text);
CREATE TEMP TABLE tags(id bigserial PRIMARY KEY,owner_id bigint,name text,UNIQUE(owner_id,name));
CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint,UNIQUE(transaction_id,tag_id));
INSERT INTO domains(id,owner_id,name,color) VALUES(1,1,'Alimentation','#fff'),(2,1,'Transport','#fff'),(3,2,'Secret','#fff');
INSERT INTO rules(id,owner_id,domain_id,pattern,tags) VALUES(1,1,1,'UBER EATS','livraison'),(2,1,2,'UBER','vtc'),(3,2,3,'UBER EATS SECRET','secret');
INSERT INTO transactions VALUES
 (1,1,1,'UBER EATS',NULL,'unknown','2026-09-01',-10.01,'Uber Eats'),
 (2,1,2,'UBER EATS',NULL,'unknown','2026-09-01',-20.02,'Uber Eats'),
 (3,1,1,'UBER EATS',2,'manual','2026-09-01',-30.03,'Uber Eats'),
 (4,1,1,'UBER EATS',NULL,'manual','2026-09-01',-40.04,'Uber Eats'),
 (5,2,1,'UBER EATS',NULL,'unknown','2026-09-01',-50.05,'Uber Eats'),
 (6,1,1,'UBER EATS',1,'manual','2026-09-01',-60.06,'Uber Eats');
INSERT INTO tags(id,owner_id,name) VALUES(1,1,'manuel'),(2,1,'historique'),(3,1,'ancien-vtc');
INSERT INTO transaction_tags VALUES(1,1),(1,2),(1,3);
SELECT setval(pg_get_serial_sequence('tags','id'),3);`)
			if err != nil {
				t.Fatal(err)
			}
			s := &Server{DB: db}
			proof := func() string {
				var p string
				if e := db.QueryRow(`SELECT jsonb_agg(jsonb_build_array(id,owner_id,tx_date,label,amount) ORDER BY id)::text FROM transactions`).Scan(&p); e != nil {
					t.Fatal(e)
				}
				return p
			}
			before := proof()
			switch mode {
			case "global":
				_, err = s.applyAllRules(1)
			case "statement":
				_, err = s.applyAllRulesForStatement(1, 1)
			case "individual-winner":
				_, err = s.applyStoredRuleToTransactions(1, 1)
			case "individual-loser":
				_, err = s.applyStoredRuleToTransactions(1, 2)
			}
			if err != nil {
				t.Fatal(err)
			}
			if proof() != before {
				t.Fatal("bank proof changed")
			}
			var tags string
			if err = db.QueryRow(`SELECT COALESCE(string_agg(tg.name,',' ORDER BY tg.name),'') FROM transaction_tags tt JOIN tags tg ON tg.id=tt.tag_id WHERE tt.transaction_id=1`).Scan(&tags); err != nil {
				t.Fatal(err)
			}
			want := "ancien-vtc,historique,livraison,manuel"
			if mode == "individual-loser" {
				want = "ancien-vtc,historique,manuel"
			}
			if tags != want {
				t.Fatal(tags, want)
			}
			var safe bool
			if err = db.QueryRow(`SELECT (SELECT domain_id=2 AND classification_source='manual' FROM transactions WHERE id=3) AND (SELECT domain_id IS NULL AND classification_source='manual' FROM transactions WHERE id=4) AND (SELECT domain_id IS NULL FROM transactions WHERE id=5) AND NOT EXISTS(SELECT 1 FROM transaction_tags tt JOIN tags t ON t.id=tt.tag_id WHERE t.name IN('vtc','secret'))`).Scan(&safe); err != nil || !safe {
				t.Fatal("manual/owner/loser safety", err)
			}
			var oldNull bool
			if err = db.QueryRow(`SELECT domain_id IS NULL FROM transactions WHERE id=2`).Scan(&oldNull); err != nil {
				t.Fatal(err)
			}
			if oldNull != (mode == "statement" || mode == "individual-loser") {
				t.Fatal("scope diverged")
			}
			if mode != "individual-loser" {
				var n int
				db.QueryRow(`SELECT count(*) FROM transaction_tags tt JOIN tags t ON t.id=tt.tag_id WHERE tt.transaction_id=6 AND t.name='livraison'`).Scan(&n)
				if n != 1 {
					t.Fatal("manual domain incorrectly locks all tags")
				}
			}
			if _, err = s.applyStoredRuleToTransactions(1, 3); err == nil {
				t.Fatal("foreign rule selected")
			}
			if mode == "global" {
				// An edited rule does not rewrite assigned domains or strip old tags.
				if _, err = db.Exec(`UPDATE rules SET domain_id=2,tags='changed' WHERE id=1`); err != nil {
					t.Fatal(err)
				}
				if _, err = s.applyAllRules(1); err != nil {
					t.Fatal(err)
				}
				var n int
				db.QueryRow(`SELECT count(*) FROM transaction_tags tt JOIN tags t ON t.id=tt.tag_id WHERE tt.transaction_id=1 AND t.name='changed'`).Scan(&n)
				if n != 0 {
					t.Fatal("conflicting domain tags added")
				}
				if proof() != before {
					t.Fatal("proof rewritten")
				}
			}
		})
	}
}

func TestCoherenceUsesWinnerRegexLiteralsAndManualProtectionPostgres(t *testing.T) {
	db := qualityTestDB(t)
	_, e := db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,name text,color text);
CREATE TEMP TABLE rules(id bigint PRIMARY KEY,owner_id bigint,domain_id bigint,pattern text,tags text);
CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,label text,display_label text,domain_id bigint,classification_source text);
CREATE TEMP TABLE tags(id bigint PRIMARY KEY,owner_id bigint,name text);CREATE TEMP TABLE transaction_tags(transaction_id bigint,tag_id bigint);
INSERT INTO domains VALUES(1,1,'Alimentation','#fff'),(2,1,'Transport','#fff'),(3,2,'Secret','#fff');
INSERT INTO rules VALUES(1,1,1,'UBER EATS','livraison'),(2,1,2,'UBER','vtc'),(3,1,1,'re:^PRLV EDF$','facture'),(4,1,1,'A_B%','literal'),(5,1,1,'re:[','invalid'),(6,2,3,'SECRET','secret');
INSERT INTO transactions VALUES(1,1,'UBER EATS','Uber Eats',1,'rule'),(2,1,'PRLV EDF','EDF',2,'manual'),(3,1,'AXXB','AXXB',NULL,'unknown'),(4,2,'SECRET','Secret',NULL,'unknown');`)
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("GET", "/api/coherence", nil).WithContext(context.WithValue(context.Background(), ctxUserKey, &authUser{ID: 1}))
	w := httptest.NewRecorder()
	(&Server{DB: db}).coherence(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var out struct {
		Findings []coherenceFinding `json:"findings"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &out); e != nil {
		t.Fatal(e)
	}
	var regex, invalid bool
	for _, f := range out.Findings {
		if f.Rule == "UBER" || f.Rule == "A_B%" || strings.Contains(f.Detail, "Secret") {
			t.Fatal(f)
		}
		if f.Rule == "re:^PRLV EDF$" && f.Kind == "manual_choice" {
			regex = true
		}
		if f.Kind == "invalid_rule" {
			invalid = true
		}
	}
	if !regex || !invalid {
		t.Fatal(out)
	}
}
