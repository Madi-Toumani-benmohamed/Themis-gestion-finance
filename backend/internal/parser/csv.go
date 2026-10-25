package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"themis/internal/models"
	"themis/internal/money"
)

// ParseCSV lit un export bancaire CSV en détectant automatiquement le format :
// séparateur (`,` `;` ou tab), colonnes (date / libellé / montant, ou
// débit+crédit séparés), format de date et de montant (virgule ou point
// décimal). L'objectif est d'avaler la plupart des exports français sans
// configuration de la part de l'utilisateur.
func ParseCSV(data []byte) (*models.ParseResult, error) {
	text := stripBOM(string(data))
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("fichier CSV vide")
	}

	sep := detectSeparator(text)
	rows, err := readCSV(text, sep)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("CSV trop court : au moins un en-tête et une ligne attendus")
	}

	// Trouver la ligne d'en-tête : la première dont on sait mapper les colonnes.
	headerIdx, cols := findHeader(rows)
	if cols == nil {
		return nil, fmt.Errorf("colonnes non reconnues : il faut au moins une date, un libellé et un montant (ou débit/crédit)")
	}

	res := &models.ParseResult{Parser: "csv", Report: &models.ImportReport{Candidates: []models.ImportCandidate{}, Issues: []string{}}}
	var minDate, maxDate string

	for rowIndex, row := range rows[headerIdx+1:] {
		if len(row) == 0 || allEmpty(row) {
			continue
		}
		date := getCol(row, cols.date)
		label := getCol(row, cols.label)
		iso := normalizeDate(date)
		for _, col := range cols.dates {
			other := getCol(row, col)
			if other != "" && normalizeDate(other) != iso {
				return nil, fmt.Errorf("ligne CSV %d : dates d'opération et de comptabilisation différentes ; format non pris en charge sans perte de provenance", headerIdx+rowIndex+2)
			}
		}
		if iso == "" || label == "" {
			return nil, fmt.Errorf("ligne CSV %d : date ou libellé invalide", headerIdx+rowIndex+2)
		}

		var amount float64
		var ok bool
		if cols.amount >= 0 {
			amount, ok = parseCSVAmount(getCol(row, cols.amount))
		} else {
			// colonnes débit / crédit séparées
			deb, dok := parseCSVAmount(getCol(row, cols.debit))
			cred, cok := parseCSVAmount(getCol(row, cols.credit))
			if dok && cok && deb != 0 && cred != 0 {
				return nil, fmt.Errorf("ligne CSV %d : débit et crédit simultanés", headerIdx+rowIndex+2)
			}
			if (getCol(row, cols.debit) != "" && !dok) || (getCol(row, cols.credit) != "" && !cok) {
				return nil, fmt.Errorf("ligne CSV %d : débit ou crédit invalide", headerIdx+rowIndex+2)
			}
			switch {
			case dok && deb != 0:
				amount, ok = -absF(deb), true
			case cok && cred != 0:
				amount, ok = absF(cred), true
			}
		}
		if !ok {
			return nil, fmt.Errorf("ligne CSV %d : montant invalide ou absent", headerIdx+rowIndex+2)
		}

		tx := models.Transaction{Date: iso, Label: cleanLabel(label), Amount: amount, SourcePage: 1, SourceLine: headerIdx + rowIndex + 2}
		res.Report.Candidates = append(res.Report.Candidates, models.ImportCandidate{Index: len(res.Transactions), Page: 1, Line: tx.SourceLine, Status: "parsed"})
		tx.Hash = TxHash(tx)
		res.Transactions = append(res.Transactions, tx)

		if minDate == "" || iso < minDate {
			minDate = iso
		}
		if maxDate == "" || iso > maxDate {
			maxDate = iso
		}
	}

	if len(res.Transactions) == 0 {
		return nil, fmt.Errorf("aucune transaction exploitable trouvée dans le CSV")
	}
	res.PeriodStart = minDate
	res.PeriodEnd = maxDate
	// Un CSV ne porte pas de solde d'ouverture/clôture fiable : pas de validation
	// au centime (contrairement au PDF). On marque comme non validé.
	res.Validated = false
	reconcileImport(res, nil)
	documentIdentity(res, data)
	return res, nil
}

type colMap struct {
	date, label, amount, debit, credit int
	dates                              []int
}

func stripBOM(s string) string {
	return strings.TrimPrefix(s, "\ufeff")
}

// detectSeparator choisit le séparateur le plus fréquent sur les 1res lignes.
func detectSeparator(text string) rune {
	sample := text
	if i := nthLineEnd(text, 5); i > 0 {
		sample = text[:i]
	}
	counts := map[rune]int{';': 0, ',': 0, '\t': 0}
	for _, c := range sample {
		if _, ok := counts[c]; ok {
			counts[c]++
		}
	}
	best, bestN := ',', -1
	for r, n := range counts {
		if n > bestN {
			best, bestN = r, n
		}
	}
	return best
}

func nthLineEnd(s string, n int) int {
	c := 0
	for i, ch := range s {
		if ch == '\n' {
			c++
			if c == n {
				return i
			}
		}
	}
	return -1
}

func readCSV(text string, sep rune) ([][]string, error) {
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = sep
	r.FieldsPerRecord = -1 // tolère un nombre de colonnes variable
	r.LazyQuotes = true
	var out [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV malformé : %w", err)
		}
		out = append(out, rec)
	}
	return out, nil
}

var (
	reDateHdr   = regexp.MustCompile(`(?i)date`)
	reLabelHdr  = regexp.MustCompile(`(?i)lib|libell|label|description|nature|op[ée]ration|motif|d[ée]tail|intitul|wording|name`)
	reAmountHdr = regexp.MustCompile(`(?i)montant|amount|valeur`)
	reDebitHdr  = regexp.MustCompile(`(?i)d[ée]bit|retrait|sortie`)
	reCreditHdr = regexp.MustCompile(`(?i)cr[ée]dit|versement|entr[ée]e|d[ée]p[ôo]t`)
)

// findHeader repère la ligne d'en-tête et associe les colonnes.
func findHeader(rows [][]string) (int, *colMap) {
	for i, row := range rows {
		if i > 5 {
			break // l'en-tête est forcément dans les toutes premières lignes
		}
		cm := &colMap{date: -1, label: -1, amount: -1, debit: -1, credit: -1}
		for j, cell := range row {
			c := strings.TrimSpace(cell)
			if reDateHdr.MatchString(c) {
				cm.dates = append(cm.dates, j)
				if cm.date < 0 {
					cm.date = j
				}
				continue
			}
			switch {
			case cm.date < 0 && reDateHdr.MatchString(c):
				cm.date = j
			case cm.debit < 0 && reDebitHdr.MatchString(c):
				cm.debit = j
			case cm.credit < 0 && reCreditHdr.MatchString(c):
				cm.credit = j
			case cm.amount < 0 && reAmountHdr.MatchString(c):
				cm.amount = j
			case cm.label < 0 && reLabelHdr.MatchString(c):
				cm.label = j
			}
		}
		hasAmount := cm.amount >= 0 || cm.debit >= 0 || cm.credit >= 0
		if cm.date >= 0 && cm.label >= 0 && hasAmount {
			return i, cm
		}
	}
	return -1, nil
}

func getCol(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func allEmpty(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

var (
	reISODate = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})`)
	reFRDate  = regexp.MustCompile(`^(\d{1,2})[/.](\d{1,2})[/.](\d{2,4})`)
)

// normalizeDate convertit JJ/MM/AAAA, JJ.MM.AA ou AAAA-MM-JJ en ISO.
func normalizeDate(s string) string {
	s = strings.TrimSpace(s)
	if m := reISODate.FindStringSubmatch(s); m != nil {
		return validCSVDate(fmt.Sprintf("%s-%s-%s", m[1], m[2], m[3]))
	}
	if m := reFRDate.FindStringSubmatch(s); m != nil {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		year, _ := strconv.Atoi(m[3])
		if year < 100 {
			year += 2000
		}
		// Par défaut JJ/MM (européen). Si le 1er nombre est clairement un mois
		// impossible en jour-européen mais possible en mois (ex. 13+ en 2e
		// position → 1er = mois), on bascule sur MM/JJ (format US).
		day, mon := a, b
		if b > 12 && a <= 12 {
			day, mon = b, a // le 2e ne peut être un mois → format MM/JJ
		}
		if mon < 1 || mon > 12 || day < 1 || day > 31 {
			return ""
		}
		return validCSVDate(fmt.Sprintf("%04d-%02d-%02d", year, mon, day))
	}
	return ""
}

var csvMoneyFormat = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:[,.][0-9]{1,2})?|[0-9]{1,3}(?:[ .][0-9]{3})+,[0-9]{1,2}|[0-9]{1,3}(?: [0-9]{3})+(?:[,.][0-9]{1,2})?|[0-9]{1,3}(?:,[0-9]{3})+\.[0-9]{1,2})$`)

// Formats reconnus seulement : ne jamais nettoyer un texte arbitraire en montant.
func parseCSVAmount(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimSpace(strings.TrimSuffix(s, "€"))
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\u00a0", " "), "\u202f", " ")
	if !csvMoneyFormat.MatchString(s) {
		return 0, false
	}
	s = strings.ReplaceAll(s, " ", "")
	if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else {
		s = strings.ReplaceAll(s, ",", "")
	}
	amount, err := money.Parse(s)
	if err != nil {
		return 0, false
	}
	// Compatibility boundary only: no financial arithmetic uses this conversion.
	v, err := strconv.ParseFloat(amount.String(), 64)
	return v, err == nil
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

var _ = bytes.MinRead

func validCSVDate(s string) string {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return ""
	}
	return s
}
