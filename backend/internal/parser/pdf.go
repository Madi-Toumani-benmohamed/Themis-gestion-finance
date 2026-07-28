package parser

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// Word est un fragment de texte positionné dans la page.
type Word struct {
	S        string
	Alt      string // lecture alternative (2e passe OCR) quand elle diffère
	X        float64
	Y        float64
	W        float64 // largeur réelle en points
	FontSize float64
}

// Row est une ligne reconstituée (fragments à Y proche, triés par X).
type Row struct {
	Words []Word
	Y     float64
	Page  int
}

func (r Row) Text() string {
	parts := make([]string, len(r.Words))
	for i, w := range r.Words {
		parts[i] = w.S
	}
	return strings.Join(parts, " ")
}

// ExtractRows lit un PDF et reconstitue les lignes de texte avec positions.
func ExtractRows(data []byte) ([]Row, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	var rows []Row
	for pageNum := 1; pageNum <= r.NumPage(); pageNum++ {
		p := r.Page(pageNum)
		if p.V.IsNull() {
			continue
		}
		content := p.Content()
		// Regroupe les fragments par Y (tolérance 2.5pt)
		byY := map[int][]Word{}
		for _, t := range content.Text {
			if t.S == "" {
				continue
			}
			key := int(t.Y / 2.5)
			byY[key] = append(byY[key], Word{S: t.S, X: t.X, Y: t.Y, W: t.W, FontSize: t.FontSize})
		}
		keys := make([]int, 0, len(byY))
		for k := range byY {
			keys = append(keys, k)
		}
		// Y décroît du haut vers le bas dans les coordonnées PDF
		sort.Sort(sort.Reverse(sort.IntSlice(keys)))
		for _, k := range keys {
			ws := byY[k]
			sort.Slice(ws, func(i, j int) bool { return ws[i].X < ws[j].X })
			// Fusionne les fragments contigus (lettres découpées par le moteur PDF)
			merged := mergeAdjacent(ws)
			rows = append(rows, Row{Words: merged, Y: ws[0].Y, Page: pageNum})
		}
	}
	return rows, nil
}

// mergeAdjacent recolle les fragments en segments : seuils relatifs à la
// taille de police (gap < 0.22em → même mot, < 1.4em → même segment avec
// espace, au-delà → nouveau segment, typiquement une autre colonne).
func mergeAdjacent(ws []Word) []Word {
	if len(ws) == 0 {
		return ws
	}
	out := []Word{ws[0]}
	for _, w := range ws[1:] {
		last := &out[len(out)-1]
		fs := last.FontSize
		if fs <= 0 {
			fs = 10
		}
		gap := w.X - (last.X + last.W)
		switch {
		case gap < fs*0.22:
			last.S += w.S
			last.W = (w.X + w.W) - last.X
		case gap < fs*1.4:
			last.S += " " + w.S
			last.W = (w.X + w.W) - last.X
		default:
			out = append(out, w)
		}
	}
	// Nettoyage final : trim des segments, suppression des segments vides
	cleaned := out[:0]
	for _, seg := range out {
		seg.S = strings.Join(strings.Fields(seg.S), " ")
		if seg.S != "" {
			cleaned = append(cleaned, seg)
		}
	}
	return cleaned
}

// PlainText retourne le texte brut du PDF, ligne par ligne (pour le fallback IA).
func PlainText(rows []Row) string {
	var b strings.Builder
	page := 0
	for _, r := range rows {
		if r.Page != page {
			page = r.Page
			fmt.Fprintf(&b, "\n--- page %d ---\n", page)
		}
		b.WriteString(r.Text())
		b.WriteString("\n")
	}
	return b.String()
}
