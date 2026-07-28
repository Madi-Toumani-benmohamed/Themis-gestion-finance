package main

import (
	"context"
	"fmt"
	"os"

	"themis/internal/parser"
)

// CLI de test : pipeline local complet (natif + OCR), sans fallback IA.
func main() {
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	res, err := parser.Parse(context.Background(), data, nil)
	if err != nil {
		fmt.Println("ERREUR:", err)
		os.Exit(1)
	}
	for _, t := range res.Transactions {
		fmt.Printf("%s  %9.2f  %s\n", t.Date, t.Amount, t.Label)
	}
	sum := 0.0
	for _, t := range res.Transactions {
		sum += t.Amount
	}
	fmt.Printf("\nparseur=%s | période %s → %s\n", res.Parser, res.PeriodStart, res.PeriodEnd)
	ob, cb := "?", "?"
	if res.OpeningBalance != nil { ob = fmt.Sprintf("%.2f", *res.OpeningBalance) }
	if res.ClosingBalance != nil { cb = fmt.Sprintf("%.2f", *res.ClosingBalance) }
	fmt.Printf("%d transactions | somme %.2f | soldes %s → %s | validé: %v\n", len(res.Transactions), sum, ob, cb, res.Validated)
}
