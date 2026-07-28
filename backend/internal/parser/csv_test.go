package parser

import "testing"

func TestParseCSVFormats(t *testing.T) {
	cases := []struct {
		name    string
		csv     string
		wantN   int
		wantFst float64 // montant de la 1re tx
	}{
		{"pointvirgule_frdate", "dateOp;label;amount\n15/06/2026;CARREFOUR;-42,50\n16/06/2026;SALAIRE;2500,00", 2, -42.50},
		{"virgule_debitcredit", "Date,Libellé,Débit,Crédit\n2026-06-15,Auchan,42.50,\n2026-06-16,Paie,,2500.00", 2, -42.50},
		{"format_EN", "Date,Description,Amount\n06/15/2026,Store,-1234.56\n06/16/2026,Pay,3000.00", 2, -1234.56},
		{"entete_decale_euro", "Export\nSolde 1000\nDate;Nature;Montant\n15/06/2026;DAB;-50,00 €\n16/06/2026;EDF;-89,90 €", 2, -50.00},
		{"tabulation", "Date\tLibelle\tMontant\n15/06/2026\tTest\t-10,00", 1, -10.00},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := ParseCSV([]byte(c.csv))
			if err != nil {
				t.Fatalf("erreur: %v", err)
			}
			if len(res.Transactions) != c.wantN {
				t.Fatalf("attendu %d tx, obtenu %d", c.wantN, len(res.Transactions))
			}
			if res.Transactions[0].Amount != c.wantFst {
				t.Errorf("1re tx: attendu %.2f, obtenu %.2f", c.wantFst, res.Transactions[0].Amount)
			}
		})
	}
}

func TestParseCSVRejectsGarbage(t *testing.T) {
	if _, err := ParseCSV([]byte("bonjour\nceci n'est pas un csv")); err == nil {
		t.Error("aurait dû rejeter un contenu non tabulaire")
	}
	if _, err := ParseCSV([]byte("")); err == nil {
		t.Error("aurait dû rejeter un fichier vide")
	}
}
