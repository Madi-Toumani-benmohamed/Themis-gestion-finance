package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"themis/internal/models"
	"themis/internal/money"
)

var (
	reDate = regexp.MustCompile(`^(\d{2})[./](\d{2})(?:[./](\d{2,4}))?$`)
	// Décimales à virgule uniquement (format FR) : indispensable pour ne pas
	// confondre une date « 22.05 » avec un montant.
	reAmount       = regexp.MustCompile(`^[+-]?\d{1,3}(?:[ \x{00A0}.]\d{3})*,\d{2}$|^[+-]?\d+,\d{2}$`)
	reSoldeOld     = regexp.MustCompile(`(?i)(ancien\s*solde|solde\s*pr[ée]c[ée]dent|solde\s*au\s*d[ée]but|solde\s*initial)`)
	reSoldeNew     = regexp.MustCompile(`(?i)(nouveau\s*solde|solde\s*final|solde\s*cr[ée]diteur|solde\s*en\s*fin|solde\s*au\s*\d{2})`)
	rePeriod       = regexp.MustCompile(`(\d{2}/\d{2}/\d{4})\s*(?:au|-|à)\s*(\d{2}/\d{2}/\d{4})`)
	reSoldeOldDate = regexp.MustCompile(`(?i)(?:ancien\s*solde|solde\s*pr[ée]c[ée]dent)[^\d]*(\d{2})[./](\d{2})[./](\d{4})`)
	reSoldeNewDate = regexp.MustCompile(`(?i)nouveau\s*solde[^\d]*(\d{2})[./](\d{2})[./](\d{4})`)
)

// ParseLocal tente d'extraire les transactions par analyse positionnelle :
// repère les colonnes Débit/Crédit dans l'en-tête, puis classe chaque montant
// selon sa position X. Retourne une erreur si le résultat est inexploitable.
func ParseLocal(rows []Row) (*models.ParseResult, error) {
	res := &models.ParseResult{Parser: "local", Report: &models.ImportReport{Candidates: []models.ImportCandidate{}, Issues: []string{}}}

	rows = normalizePageColumns(rows)
	debitX, creditX, creditRight := findColumns(rows)
	rows = sanitizeRows(rows, creditRight)
	rows = stripMarginCodes(rows)
	fullText := ""
	for _, r := range rows {
		fullText += r.Text() + "\n"
	}

	// Période du relevé : « du jj/mm/aaaa au jj/mm/aaaa », ou à défaut les
	// dates portées par les lignes « Ancien/Nouveau solde au jj.mm.aaaa »
	if m := rePeriod.FindStringSubmatch(fullText); m != nil {
		res.PeriodStart = frDateToISO(m[1])
		res.PeriodEnd = frDateToISO(m[2])
	}
	if res.PeriodStart == "" {
		if m := reSoldeOldDate.FindStringSubmatch(fullText); m != nil {
			res.PeriodStart = m[3] + "-" + m[2] + "-" + m[1]
		}
	}
	if res.PeriodEnd == "" {
		if m := reSoldeNewDate.FindStringSubmatch(fullText); m != nil {
			res.PeriodEnd = m[3] + "-" + m[2] + "-" + m[1]
		}
	}

	var current *models.Transaction
	var hasAmount bool
	candidateIndex := -1
	startY, startM := yearMonth(res.PeriodStart)
	endY, endM := yearMonth(res.PeriodEnd)
	chooseYear := func(month int) int {
		if endY == 0 {
			return 0
		}
		if startY != 0 && startY != endY && (month == startM || month > endM) {
			return startY
		}
		return endY
	}
	flush := func() {
		if current == nil {
			return
		}
		current.Label = cleanLabel(current.Label)
		if hasAmount && current.Label != "" {
			res.Transactions = append(res.Transactions, *current)
		} else {
			res.Report.Candidates[candidateIndex].Status = "uninterpretable"
			res.Report.Candidates[candidateIndex].Reason = "missing_amount_or_label"
		}
		current = nil
		hasAmount = false
	}
	setAmount := func(row Row) bool {
		amount, alt, x, ok := trailingAmount(row)
		if !ok {
			return false
		}
		count := 0
		for _, w := range row.Words {
			if reAmount.MatchString(normalizeAmountStr(w.S)) {
				count++
			}
		}
		if count != 1 {
			res.Report.Issues = appendUnique(res.Report.Issues, "multiple_amounts_on_operation")
			return false
		}
		if debitX < 0 || creditX < 0 {
			res.Report.Issues = appendUnique(res.Report.Issues, "unknown_debit_credit_columns")
		}
		sign := signFromColumns(x, debitX, creditX)
		current.Amount = sign * amount
		if alt != nil {
			v := sign * (*alt)
			current.AltAmount = &v
		}
		return true
	}
	pageLines := map[int]int{}
	inTable := false
	for _, row := range rows {
		pageLines[row.Page]++
		line := row.Line
		if line == 0 {
			line = pageLines[row.Page]
		}
		text := row.Text()
		if reSoldeOld.MatchString(text) || reSoldeNew.MatchString(text) {
			flush()
			if v, _, ok := lastAmount(row); ok {
				if reSoldeOld.MatchString(text) {
					res.OpeningBalance = &v
				} else {
					res.ClosingBalance = &v
				}
			}
			inTable = reSoldeOld.MatchString(text)
			continue
		}
		if strings.Contains(strings.ToUpper(removeAccents(text)), "TOTAL DES OPERATIONS") {
			flush()
			inTable = false
			continue
		}
		if len(row.Words) >= 2 {
			if day, month, y, ok := parseDayMonth(row.Words[0].S); ok {
				flush()
				inTable = true
				candidateIndex = len(res.Report.Candidates)
				res.Report.Candidates = append(res.Report.Candidates, models.ImportCandidate{Index: candidateIndex, Page: row.Page, Line: line, Status: "parsed"})
				if y == 0 {
					y = chooseYear(month)
				}
				date := fmt.Sprintf("%04d-%02d-%02d", y, month, day)
				if _, e := time.Parse("2006-01-02", date); e != nil {
					res.Report.Candidates[candidateIndex].Status = "uninterpretable"
					res.Report.Candidates[candidateIndex].Reason = "invalid_or_unknown_date"
					continue
				}
				current = &models.Transaction{Date: date, SourcePage: row.Page, SourceLine: line}
				if len(row.Words) > 1 {
					if d, m, y, ok := parseDayMonth(row.Words[1].S); ok {
						if y == 0 {
							y = chooseYear(m)
						}
						value := fmt.Sprintf("%04d-%02d-%02d", y, m, d)
						if _, e := time.Parse("2006-01-02", value); e != nil {
							res.Report.Issues = appendUnique(res.Report.Issues, "invalid_value_date")
						} else {
							current.ValueDate = value
						}
					}
				}
				hasAmount = setAmount(row)
				current.Label = labelBetween(row, hasAmount)
				continue
			}
		}
		// Repeated headers, page furniture and totals do not terminate a pending
		// operation: the amount/description may continue on the following page.
		d, c, _ := findColumns([]Row{row})
		if d >= 0 && c >= 0 {
			inTable = true
			continue
		}
		if looksLikeNoise(text) {
			continue
		}
		if current != nil {
			if _, _, ok := lastAmount(row); ok {
				if !hasAmount {
					hasAmount = setAmount(row)
					for _, w := range row.Words {
						if !reAmount.MatchString(normalizeAmountStr(w.S)) {
							current.Label += " " + w.S
						}
					}
				} else {
					flush()
					res.Report.Candidates = append(res.Report.Candidates, models.ImportCandidate{Index: len(res.Report.Candidates), Page: row.Page, Line: line, Status: "uninterpretable", Reason: "amount_without_operation_date"})
				}
			} else {
				current.Label += " " + text
			}
		} else if inTable {
			if _, _, ok := lastAmount(row); ok {
				res.Report.Candidates = append(res.Report.Candidates, models.ImportCandidate{Index: len(res.Report.Candidates), Page: row.Page, Line: line, Status: "uninterpretable", Reason: "amount_without_operation_date"})
			}
		}
	}
	flush()

	if len(res.Transactions) == 0 && len(res.Report.Candidates) == 0 {
		return nil, fmt.Errorf("aucune transaction détectée par le parseur local")
	}

	reconcileImport(res, rows)

	// Hash de déduplication (après réconciliation : les montants sont définitifs)
	for i := range res.Transactions {
		res.Transactions[i].Hash = TxHash(res.Transactions[i])
	}
	return res, nil
}

// Validate exige une égalité exacte au centime ; aucune tolérance implicite.
func Validate(res *models.ParseResult) bool {
	if res.OpeningBalance == nil || res.ClosingBalance == nil {
		return false
	}
	opening, e1 := money.FromLegacy(*res.OpeningBalance)
	closing, e2 := money.FromLegacy(*res.ClosingBalance)
	if e1 != nil || e2 != nil {
		return false
	}
	sum := opening
	for _, t := range res.Transactions {
		amount, err := money.FromLegacy(t.Amount)
		if err != nil {
			return false
		}
		sum += amount
	}
	return sum == closing
}

// TxHash calcule l'empreinte de déduplication d'une transaction.
// TxHash identifie une transaction pour la déduplication. La normalisation
// (casse, espaces multiples) rend le hash stable entre les chemins de
// parsing : « PRLV  FREE » natif et « Prlv Free » OCR donnent le même hash.
func TxHash(t models.Transaction) string {
	label := strings.ToUpper(strings.Join(strings.Fields(t.Label), " "))
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%.2f", t.Date, label, t.Amount)))
	return hex.EncodeToString(h[:])
}

// findColumns cherche la ligne d'en-tête contenant À LA FOIS « Débit » et
// « Crédit » (exiger la même ligne évite les faux positifs comme le nom de la
// banque « CREDIT AGRICOLE »). Retourne le milieu de chaque en-tête et le bord
// droit de la colonne Crédit (au-delà : cases à cocher et marges à ignorer).
func findColumns(rows []Row) (debitX, creditX, creditRight float64) {
	for _, r := range rows {
		d, c, cr := -1.0, -1.0, -1.0
		for _, w := range r.Words {
			u := strings.ToUpper(removeAccents(w.S))
			if strings.Contains(u, "DEBIT") && d < 0 {
				d = w.X + w.W/2
			}
			if strings.Contains(u, "CREDIT") && c < 0 {
				c = w.X + w.W/2
				cr = w.X + w.W
			}
		}
		if d >= 0 && c >= 0 && c > d {
			return d, c, cr
		}
	}
	return -1, -1, -1
}

// signFromColumns détermine le signe d'un montant selon la colonne la plus
// proche de son milieu. Sans en-têtes détectés, on suppose débit.
func signFromColumns(mid, debitX, creditX float64) float64 {
	if debitX >= 0 && creditX >= 0 {
		if math.Abs(mid-creditX) < math.Abs(mid-debitX) {
			return 1
		}
		return -1
	}
	return -1
}

// trailingAmount retourne le dernier montant de la ligne, le milieu de son
// segment, et l'éventuelle lecture OCR alternative de ce montant.
func trailingAmount(row Row) (float64, *float64, float64, bool) {
	for i := len(row.Words) - 1; i >= 0; i-- {
		w := row.Words[i]
		s := normalizeAmountStr(w.S)
		if reAmount.MatchString(s) {
			if v, err := parseAmount(s); err == nil {
				var alt *float64
				if a := normalizeAmountStr(w.Alt); a != "" && reAmount.MatchString(a) {
					if av, err := parseAmount(a); err == nil && av != v {
						alt = &av
					}
				}
				return v, alt, w.X + w.W/2, true
			}
		}
	}
	return 0, nil, 0, false
}

func lastAmount(row Row) (float64, *float64, bool) {
	v, alt, _, ok := trailingAmount(row)
	return v, alt, ok
}

// labelBetween assemble le libellé (tout sauf la date en tête et le montant en queue).
func labelBetween(row Row, hasAmount bool) string {
	end := len(row.Words)
	if hasAmount {
		end--
	}
	parts := []string{}
	for i := 1; i < end; i++ {
		s := row.Words[i].S
		if i == 1 {
			if _, _, _, ok := parseDayMonth(s); ok { // date de valeur
				continue
			}
		}
		if reAmount.MatchString(normalizeAmountStr(s)) {
			continue
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

// parseDayMonth reconnaît « jj.mm », « jj/mm », « jj.mm.aaaa », « jj/mm/aa »…
// Retourne (jour, mois, année ou 0 si absente, ok).
func parseDayMonth(s string) (int, int, int, bool) {
	m := reDate.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0, 0, false
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	if day < 1 || day > 31 || month < 1 || month > 12 {
		return 0, 0, 0, false
	}
	year := 0
	if m[3] != "" {
		year, _ = strconv.Atoi(m[3])
		if year < 100 {
			year += 2000
		}
	}
	return day, month, year, true
}

// yearMonth extrait (année, mois) d'une date ISO, (0, 0) si vide/invalide.
func yearMonth(iso string) (int, int) {
	if len(iso) < 7 {
		return 0, 0
	}
	y, err1 := strconv.Atoi(iso[:4])
	m, err2 := strconv.Atoi(iso[5:7])
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return y, m
}

func frDateToISO(d string) string {
	p := strings.Split(d, "/")
	if len(p) != 3 {
		return ""
	}
	return p[2] + "-" + p[1] + "-" + p[0]
}

func normalizeAmountStr(s string) string {
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "€")
	s = strings.TrimSuffix(s, "EUR")
	return strings.TrimSpace(s)
}

func parseAmount(s string) (float64, error) {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimLeft(s, "+-")
	// format français : 1 234,56 ou 1.234,56 → 1234.56
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else {
		s = strings.ReplaceAll(s, " ", "")
	}
	if neg {
		s = "-" + s
	}
	c, err := money.Parse(s)
	return float64(c) / 100, err
}

func cleanLabel(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func looksLikeNoise(s string) bool {
	u := strings.ToUpper(removeAccents(s))
	for _, kw := range []string{"PAGE", "RELEVE", "IBAN", "BIC", "TELEPHONE", "WWW.", "CAPITAL", "SIREN", "TOTAL DES"} {
		if strings.Contains(u, kw) {
			return true
		}
	}
	return false
}

func removeAccents(s string) string {
	r := strings.NewReplacer("é", "e", "è", "e", "ê", "e", "É", "E", "È", "E", "Ê", "E", "à", "a", "â", "a", "ç", "c")
	return r.Replace(s)
}

// sanitizeRows nettoie le bruit d'OCR des tableaux : caractères de bordure
// (« | ») collés aux mots, tokens ne contenant ni lettre ni chiffre, et tout
// ce qui se trouve à droite de la colonne Crédit (cases à cocher).
func sanitizeRows(rows []Row, creditRight float64) []Row {
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		ws := make([]Word, 0, len(r.Words))
		for _, w := range r.Words {
			w.S = strings.Trim(w.S, "|¦!")
			if w.S == "" || !hasAlnum(w.S) {
				continue
			}
			if creditRight > 0 && w.X > creditRight*1.005 && !reAmount.MatchString(normalizeAmountStr(w.S)) {
				continue // au-delà de la colonne Crédit et pas un montant : case à cocher, marge
			}
			ws = append(ws, w)
		}
		if len(ws) > 0 {
			out = append(out, Row{Words: ws, Y: r.Y, Page: r.Page, Line: r.Line})
		}
	}
	return out
}

func hasAlnum(s string) bool {
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return true
		}
	}
	return false
}

// PDF text layout can use a different character pitch on each page. Column
// positions are therefore established per page, never borrowed from page one.
func normalizePageColumns(rows []Row) []Row {
	pages := map[int][]Row{}
	for _, r := range rows {
		pages[r.Page] = append(pages[r.Page], r)
	}
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		d, c, _ := findColumns(pages[r.Page])
		if d >= 0 && c > d {
			scale := 80 / (c - d)
			ws := append([]Word(nil), r.Words...)
			for i := range ws {
				ws[i].X = 400 + (ws[i].X-d)*scale
				ws[i].W *= scale
				ws[i].FontSize *= scale
			}
			r.Words = ws
		}
		out = append(out, r)
	}
	return out
}

// Some statements print a vertical numeric reference in the left margin.
// OCR can attach it to a table row. Remove it only when two explicit date
// columns follow it; neither dates nor amounts are inferred or rewritten.
func stripMarginCodes(rows []Row) []Row {
	for i := range rows {
		ws := rows[i].Words
		if len(ws) < 4 {
			continue
		}
		if _, _, _, ok := parseDayMonth(ws[1].S); !ok {
			continue
		}
		if _, _, _, ok := parseDayMonth(ws[2].S); !ok {
			continue
		}
		margin := len(ws[0].S) >= 4 && ws[0].X+ws[0].W < ws[1].X
		for _, c := range ws[0].S {
			if c < '0' || c > '9' {
				margin = false
			}
		}
		if margin {
			rows[i].Words = ws[1:]
		}
	}
	return rows
}
