package ai

import (
	"database/sql"
	_ "github.com/lib/pq"
	"os"
	"strings"
	"testing"
)

func TestMonthlyPromptExactSharedFinancialScopePostgres(t *testing.T) {
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
	_, err = db.Exec(`CREATE TEMP TABLE domains(id bigint PRIMARY KEY,owner_id bigint,parent_id bigint,name text,is_transfer bool);
 CREATE TEMP TABLE transactions(id bigint PRIMARY KEY,owner_id bigint,tx_date date,amount numeric(12,2),domain_id bigint,label_kind text);
 INSERT INTO domains VALUES(1,1,NULL,'Courses',false),(2,1,NULL,'Transfert',true),(3,1,2,'Livret',false),(4,2,NULL,'SECRET',false);
 INSERT INTO transactions VALUES(1,1,'2020-03-10',3249.47,1,''),(2,1,'2020-03-10',-4197.14,1,''),(3,1,'2020-03-11',22,1,''),(4,1,'2020-03-11',-22,1,''),(5,1,'2020-03-10',-100,2,''),(6,1,'2020-03-10',100,2,''),(7,1,'2020-03-10',-50,3,''),(8,1,'2020-03-10',50,3,''),(9,1,'2020-03-10',-25,1,'savings'),(10,1,'2020-03-10',25,1,'savings'),(11,2,'2020-03-10',9999,4,''),(12,1,'2020-02-10',-30.01,1,''),(13,1,'2020-01-10',-60.02,1,'');`)
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := BuildMonthlyPrompt(db, 1, "2020-03")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Crédits observés : 3271.47 € | Débits observés : 4219.14 € | Flux net : -947.67 €", "Courses : 4219.14 € (2 opérations)", "45.02 € en moyenne par mois avec transactions (mois absents exclus, arrondi au centime)", "Couverture documentaire : inconnue", "diviseur : 2 mois sur la fenêtre de 3 mois, 2 débits inclus"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("missing %q in %s", want, prompt)
		}
	}
	for _, secret := range []string{"SECRET", "Livret :", "Transfert :", "9999", "écart observé", " vs ", "EN HAUSSE", "EN BAISSE"} {
		if strings.Contains(prompt, secret) {
			t.Fatalf("excluded data leaked: %s", prompt)
		}
	}
	if _, err = BuildMonthlyPrompt(db, 1, "2020-13"); err == nil {
		t.Fatal("invalid month accepted")
	}
	if _, err = BuildMonthlyPrompt(db, 1, "2010-01"); err == nil {
		t.Fatal("absent data presented as zero")
	}
}
