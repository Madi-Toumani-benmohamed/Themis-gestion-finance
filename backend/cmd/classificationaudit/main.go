// classificationaudit is an optional local, read-only diagnostic. Its output
// contains financial aggregates: redirect it to a private file, never CI logs.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"themis/internal/classificationaudit"
)

func main() {
	owner := flag.Int64("owner", 0, "propriétaire local requis")
	flag.Parse()
	if *owner <= 0 || os.Getenv("DATABASE_URL") == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage : classificationaudit -owner ID ; DATABASE_URL requise. Lecture seule, sortie privée.")
		os.Exit(2)
	}
	db, e := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if e != nil {
		fmt.Fprintln(os.Stderr, "Configuration locale invalide")
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, e := classificationaudit.Run(ctx, db, *owner)
	if e != nil {
		fmt.Fprintln(os.Stderr, "Audit non produit : lecture indisponible, données invalides ou limite atteinte. Aucun résultat partiel.")
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if enc.Encode(r) != nil {
		os.Exit(1)
	}
}
