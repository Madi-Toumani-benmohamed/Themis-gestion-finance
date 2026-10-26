// Command e2efixtures creates synthetic test input or performs narrowly scoped
// test-only database operations. It intentionally refuses non-e2e databases.
package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	_ "github.com/lib/pq"
	"themis/internal/testfixture"
)

func main() {
	if len(os.Args) != 3 {
		fatal("usage: e2efixtures statement <path> | expire-session <stdin>")
	}
	switch os.Args[1] {
	case "statement":
		if err := os.WriteFile(os.Args[2], testfixture.Statement009PDF(), 0o600); err != nil {
			fatal("écriture du relevé synthétique impossible")
		}
	case "expire-session":
		if os.Args[2] != "-" {
			fatal("le jeton de session doit être transmis sur stdin")
		}
		url := os.Getenv("THEMIS_E2E_DATABASE_URL")
		if url == "" {
			fatal("THEMIS_E2E_DATABASE_URL est requis")
		}
		token, err := io.ReadAll(io.LimitReader(os.Stdin, 4097))
		if err != nil || len(token) < 32 || len(token) > 4096 {
			fatal("jeton de session invalide")
		}
		db, err := sql.Open("postgres", url)
		if err != nil {
			fatal("connexion à la base E2E impossible")
		}
		defer db.Close()
		var name string
		if err = db.QueryRow(`SELECT current_database()`).Scan(&name); err != nil || name != "themis_e2e" {
			fatal("opération refusée hors de la base themis_e2e")
		}
		if _, err = db.Exec(`UPDATE sessions SET expires_at=now()-interval '1 minute' WHERE token=$1`, string(token)); err != nil {
			fatal("expiration de la session E2E impossible")
		}
	default:
		fatal("commande E2E inconnue")
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
