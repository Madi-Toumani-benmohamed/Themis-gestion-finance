// Importaudit is deliberately read-only. Redirect the financial report to a
// private file; it never emits bank labels, credentials or raw PDF contents.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"themis/internal/db"
	"themis/internal/importaudit"
	"time"
)

func main() {
	owner := flag.Int64("owner", 0, "propriétaire à auditer (obligatoire)")
	flag.Parse()
	if *owner <= 0 {
		fmt.Fprintln(os.Stderr, "Utilisation : importaudit -owner ID ; DATABASE_URL requise. Aucun changement en base.")
		os.Exit(2)
	}
	d, err := db.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Connexion à la base d'audit impossible")
		os.Exit(1)
	}
	defer d.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	results, err := importaudit.Run(ctx, d, *owner)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Audit en lecture seule impossible")
		os.Exit(1)
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	if e.Encode(results) != nil {
		os.Exit(1)
	}
}
