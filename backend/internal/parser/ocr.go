package parser

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// OCRAvailable indique si le pipeline OCR est complet : pdftoppm, tesseract
// ET le pack de langue français.
func OCRAvailable() bool {
	ok, _ := OCRStatus()
	return ok
}

// OCRStatus détaille l'état du pipeline OCR pour le diagnostic
// (/api/health et logs de démarrage).
func OCRStatus() (bool, string) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return false, "pdftoppm introuvable (installer poppler-utils)"
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return false, "tesseract introuvable (installer tesseract-ocr)"
	}
	out, err := exec.Command("tesseract", "--list-langs").CombinedOutput()
	if err != nil {
		return false, fmt.Sprintf("tesseract inutilisable: %v", err)
	}
	for _, l := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(l) == "fra" {
			return true, "ok"
		}
	}
	return false, "pack de langue français absent (installer tesseract-ocr-fra)"
}

// HasUsefulText estime si l'extraction native contient assez de texte pour
// être exploitable (certains relevés — Crédit Agricole notamment — dessinent
// chaque caractère en bitmap : aucune couche texte n'existe).
func HasUsefulText(rows []Row) bool {
	n := 0
	for _, r := range rows {
		for _, c := range r.Text() {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
				n++
				if n >= 300 {
					return true
				}
			}
		}
	}
	return false
}

// ocrWord est un mot OCR avec sa boîte englobante complète.
type ocrWord struct {
	Word
	top, height float64
}

// ExtractRowsOCR rasterise le PDF (pdftoppm, 480 dpi — multiple entier des
// 240 ppi des glyphes bitmap des relevés, donc sans interpolation floue) puis
// OCRise chaque page (tesseract, TSV) et reconstruit des lignes positionnées
// compatibles avec le parseur positionnel. Chaque montant détecté est vérifié
// par une seconde passe OCR ciblée (psm 7 + whitelist chiffres), nettement
// plus fiable sur les nombres — mais pas infaillible : les deux lectures sont
// conservées et le parseur choisit la combinaison qui satisfait l'équation
// comptable. Coordonnées en pixels.
func ExtractRowsOCR(ctx context.Context, data []byte) ([]Row, error) {
	dir, err := os.MkdirTemp("", "solde-ocr-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	pdfPath := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(pdfPath, data, 0o600); err != nil {
		return nil, err
	}
	report(ctx, "Conversion du PDF en images", "", 8)
	if out, err := exec.Command("pdftoppm", "-r", "480", "-gray", "-png", pdfPath, filepath.Join(dir, "page")).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %v (%s)", err, out)
	}
	pages, _ := filepath.Glob(filepath.Join(dir, "page-*.png"))
	sort.Strings(pages)
	if len(pages) == 0 {
		return nil, fmt.Errorf("pdftoppm n'a produit aucune page")
	}

	var rows []Row
	for i, pngPath := range pages {
		// L'OCR page par page est la partie la plus longue de l'import : on
		// publie la vraie avancée (page i/N), mappée sur la plage 12→82 %.
		report(ctx, "Lecture du relevé (OCR)",
			fmt.Sprintf("page %d sur %d", i+1, len(pages)),
			12+int(70*float64(i)/float64(len(pages))))
		out, err := exec.Command("tesseract", pngPath, "stdout", "-l", "fra", "--psm", "3", "tsv").Output()
		if err != nil {
			return nil, fmt.Errorf("tesseract page %d: %v", i+1, err)
		}
		words := tsvToWords(string(out))
		verifyAmounts(words, pngPath, dir)
		rows = append(rows, clusterRows(words, i+1)...)
	}
	report(ctx, "Analyse des lignes extraites", "", 84)
	return rows, nil
}

// tsvToWords parse la sortie TSV de tesseract (niveau 5 = mot).
// Colonnes : level page block par line word left top width height conf text.
func tsvToWords(tsv string) []*ocrWord {
	var words []*ocrWord
	for _, l := range strings.Split(tsv, "\n") {
		f := strings.Split(l, "\t")
		if len(f) != 12 || f[0] != "5" {
			continue
		}
		text := strings.TrimSpace(f[11])
		if text == "" {
			continue
		}
		conf, _ := strconv.ParseFloat(f[10], 64)
		if conf < 30 { // mots trop incertains : bruit
			continue
		}
		left, _ := strconv.ParseFloat(f[6], 64)
		top, _ := strconv.ParseFloat(f[7], 64)
		width, _ := strconv.ParseFloat(f[8], 64)
		height, _ := strconv.ParseFloat(f[9], 64)
		words = append(words, &ocrWord{
			Word: Word{S: text, X: left, W: width, FontSize: height},
			top:  top, height: height,
		})
	}
	return words
}

var (
	reAmountCandidate = regexp.MustCompile(`\d,\d{2}$`)  // finit par ,dd → montant probable
	reDigitsOnly      = regexp.MustCompile(`^[\d .,]+$`) // sortie valide de la 2e passe
)

// verifyAmounts re-OCRise chaque montant probable dans sa propre boîte
// (psm 7 = ligne unique, whitelist chiffres). Les glyphes bitmap des relevés
// provoquent des confusions 5/9 ou 8/3 en pleine page que la passe ciblée corrige.
func verifyAmounts(words []*ocrWord, pngPath, dir string) {
	var img image.Image
	for _, w := range words {
		if !reAmountCandidate.MatchString(w.S) {
			continue
		}
		if img == nil { // décodage paresseux : seulement si la page a des montants
			f, err := os.Open(pngPath)
			if err != nil {
				return
			}
			img, err = png.Decode(f)
			f.Close()
			if err != nil {
				return
			}
		}
		sub, ok := img.(interface {
			SubImage(image.Rectangle) image.Image
		})
		if !ok {
			return
		}
		m := 10 // marge en pixels
		rect := image.Rect(int(w.X)-m, int(w.top)-m, int(w.X+w.W)+m, int(w.top+w.height)+m).
			Intersect(img.Bounds())
		cropPath := filepath.Join(dir, "amt.png")
		cf, err := os.Create(cropPath)
		if err != nil {
			return
		}
		err = png.Encode(cf, sub.SubImage(rect))
		cf.Close()
		if err != nil {
			continue
		}
		out, err := exec.Command("tesseract", cropPath, "stdout", "--psm", "7",
			"-c", "tessedit_char_whitelist=0123456789,. ").Output()
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(out))
		if s != "" && s != w.S && reDigitsOnly.MatchString(s) && reAmountCandidate.MatchString(s) {
			w.Alt = s // lecture concurrente : l'équation comptable arbitrera
		}
	}
}

// clusterRows reconstruit les lignes par regroupement vertical des mots.
// On n'utilise PAS la segmentation en lignes de tesseract : ses blocs peuvent
// séparer la colonne des montants du reste de la ligne. Deux mots appartiennent
// à la même ligne si leurs centres verticaux sont proches.
func clusterRows(words []*ocrWord, page int) []Row {
	if len(words) == 0 {
		return nil
	}
	sort.Slice(words, func(i, j int) bool {
		return words[i].top+words[i].height/2 < words[j].top+words[j].height/2
	})
	var rows []Row
	var cluster []*ocrWord
	clusterCenter := 0.0
	flush := func() {
		if len(cluster) == 0 {
			return
		}
		ws := make([]Word, len(cluster))
		for i, ow := range cluster {
			ws[i] = ow.Word
		}
		sort.Slice(ws, func(i, j int) bool { return ws[i].X < ws[j].X })
		rows = append(rows, Row{Words: mergeThousands(ws), Y: -clusterCenter, Page: page})
		cluster = nil
	}
	for _, w := range words {
		c := w.top + w.height/2
		tol := w.height * 0.6
		if tol < 8 {
			tol = 8
		}
		if len(cluster) == 0 || c-clusterCenter <= tol {
			cluster = append(cluster, w)
			clusterCenter += (c - clusterCenter) / float64(len(cluster)) // moyenne glissante
		} else {
			flush()
			cluster = append(cluster, w)
			clusterCenter = c
		}
	}
	flush()
	return rows
}

var (
	reThouHead = regexp.MustCompile(`^\d{1,3}$`)
	reThouTail = regexp.MustCompile(`^\d{3}(?:[.,]\d{2})?$`)
)

// mergeThousands recolle les montants coupés par le séparateur de milliers
// ("1" + "520,00" → "1 520,00"). Seuls des groupes de chiffres proches sont
// fusionnés, jamais deux dates ni deux colonnes distinctes.
func mergeThousands(ws []Word) []Word {
	out := make([]Word, 0, len(ws))
	for i := 0; i < len(ws); i++ {
		w := ws[i]
		for i+1 < len(ws) {
			parts := strings.Fields(w.S)
			head := parts[len(parts)-1]
			next := ws[i+1]
			gap := next.X - (w.X + w.W)
			if reThouHead.MatchString(head) && reThouTail.MatchString(next.S) && gap < next.FontSize*1.2 {
				if next.Alt != "" {
					w.Alt = w.S + " " + next.Alt
				}
				w.S += " " + next.S
				w.W = (next.X + next.W) - w.X
				i++
				continue
			}
			break
		}
		out = append(out, w)
	}
	return out
}
