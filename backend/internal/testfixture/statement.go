// Package testfixture builds anonymous PDFs, including the custom Type 3 font
// and ToUnicode mappings that defeated the previous native extractor. No real
// banking document or identifier is embedded in this fixture.
package testfixture

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

func Statement009PDF() []byte {
	var pages [][]string
	credit := map[int]int{8: 100000, 30: 94058, 60: 102527}
	debitIndex := 0
	for page := 0; page < 3; page++ {
		lines := []string{"Releve TEST du 21/08/2026 au 21/09/2026", fmt.Sprintf("%-64s %12s %12s", "Date Valeur Libelle", "Debit", "Credit")}
		if page == 0 {
			lines = append(lines, fmt.Sprintf("%-77s %12s", "Ancien solde au 21.08.2026", "2 492,88"))
		}
		for j := 0; j < 27; j++ {
			i := page*27 + j
			date := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i/3)
			if i == 80 {
				date = time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
			}
			d, c := "", ""
			if v, ok := credit[i]; ok {
				c = fr(v)
			} else {
				v := 5000
				if debitIndex == 75 {
					v = 29816
				}
				if debitIndex == 76 {
					v = 269
				}
				if debitIndex == 77 {
					v = 24300
				}
				debitIndex++
				d = fr(v)
			}
			label := fmt.Sprintf("ACHAT TEST %03d", i+1)
			// Identical legitimate occurrences, same amount/date/label.
			if i == 3 || i == 4 {
				label = "DEUX ACHATS IDENTIQUES TEST"
			}
			lines = append(lines, fmt.Sprintf("%-64s %12s %12s", date.Format("02.01")+" "+date.Format("02.01")+" "+label, d, c))
		}
		if page == 2 {
			lines = append(lines, fmt.Sprintf("%-64s %12s %12s", "Total des operations", "4 293,85", "2 965,85"), fmt.Sprintf("%-77s %12s", "Nouveau solde au 21.09.2026", "1 164,88"))
		}
		pages = append(pages, lines)
	}
	return customFontPDF(pages)
}
func fr(c int) string { return fmt.Sprintf("%d,%02d", c/100, c%100) }

func customFontPDF(pages [][]string) []byte {
	objects := []string{"", "", ""}
	add := func(s string) int { objects = append(objects, s); return len(objects) }
	stream := func(s string) string { return fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(s), s) }
	regular := add("<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>")
	var mapping, encoding, procs, widths strings.Builder
	mapping.WriteString("/CIDInit /ProcSet findresource begin 12 dict begin begincmap /CIDSystemInfo << /Registry (TEST) /Ordering (TEST) /Supplement 0 >> def /CMapName /TEST def /CMapType 2 def 1 begincodespacerange <01> <5F> endcodespacerange 95 beginbfchar\n")
	for c := 32; c <= 126; c++ {
		code := 1 + (c-32+37)%95
		fmt.Fprintf(&mapping, "<%02X> <%04X>\n", code, c)
		fmt.Fprintf(&encoding, "%d /glyph%d ", code, code)
		glyph := string(rune(c))
		glyph = strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(glyph)
		id := add(stream(fmt.Sprintf("600 0 d0 BT /G 1000 Tf (%s) Tj ET\n", glyph)))
		fmt.Fprintf(&procs, "/glyph%d %d 0 R ", code, id)
		widths.WriteString("600 ")
	}
	mapping.WriteString("endbfchar endcmap CMapName currentdict /CMap defineresource pop end end\n")
	cmap := add(stream(mapping.String()))
	objects[2] = fmt.Sprintf("<< /Type /Font /Subtype /Type3 /FontBBox [0 -250 600 1000] /FontMatrix [0.001 0 0 0.001 0 0] /FirstChar 1 /LastChar 95 /Widths [%s] /Encoding << /Type /Encoding /Differences [%s] >> /ToUnicode %d 0 R /CharProcs << %s >> /Resources << /Font << /G %d 0 R >> >> >>", widths.String(), encoding.String(), cmap, procs.String(), regular)
	var kids []string
	for _, lines := range pages {
		var content strings.Builder
		for line, s := range lines {
			var encoded strings.Builder
			for _, c := range s {
				fmt.Fprintf(&encoded, "%02X", 1+(int(c)-32+37)%95)
			}
			fmt.Fprintf(&content, "BT /F1 9 Tf 1 0 0 1 20 %d Tm <%s> Tj ET\n", 790-line*18, encoded.String())
		}
		body := add(stream(content.String()))
		page := add(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 842] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", body))
		kids = append(kids, fmt.Sprintf("%d 0 R", page))
	}
	objects[0] = "<< /Type /Catalog /Pages 2 0 R >>"
	objects[1] = fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(pages), strings.Join(kids, " "))
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, s := range objects {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, s)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return pdf.Bytes()
}
