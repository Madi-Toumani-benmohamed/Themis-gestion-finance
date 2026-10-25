package parser

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"
)

// ExtractRowsContext uses Poppler's Unicode/font mapping before the legacy
// decoder. It preserves the positions needed to distinguish debit and credit.
// The runtime image already includes Poppler; installations without it retain
// the existing native decoder and OCR fallback.
func ExtractRowsContext(ctx context.Context, data []byte) ([]Row, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return ExtractRows(data)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	f, err := os.CreateTemp("", "themis-extract-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("préparation extraction PDF impossible")
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return nil, fmt.Errorf("préparation extraction PDF impossible")
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-enc", "UTF-8", f.Name(), "-")
	// Neither stderr nor the document text may reach application logs.
	var output limitedPDFBuffer
	cmd.Stdout = &output
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("délai d'extraction PDF dépassé")
		}
		return nil, fmt.Errorf("extraction PDF positionnelle impossible")
	}
	return decodePopplerLayout(output.String()), nil
}

type limitedPDFBuffer struct{ bytes.Buffer }

func (b *limitedPDFBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 32*1024*1024 {
		return 0, fmt.Errorf("extraction PDF trop volumineuse")
	}
	return b.Buffer.Write(p)
}

// Poppler's layout renderer aligns the baselines even when a PDF font reports
// an oversized glyph bounding box. Coordinates below are text columns, not PDF
// points; using one coordinate system for headers and amounts is essential.
func decodePopplerLayout(text string) []Row {
	var rows []Row
	for page, content := range strings.Split(text, "\f") {
		for line, text := range strings.Split(content, "\n") {
			runes := []rune(text)
			var words []Word
			for i := 0; i < len(runes); {
				if unicode.IsSpace(runes[i]) {
					i++
					continue
				}
				start := i
				for i < len(runes) && !unicode.IsSpace(runes[i]) {
					i++
				}
				words = append(words, Word{S: string(runes[start:i]), X: float64(start), Y: float64(line), W: float64(i - start), FontSize: 1})
			}
			if len(words) > 0 {
				rows = append(rows, Row{Words: mergeThousands(words), Y: float64(line), Page: page + 1, Line: line + 1})
			}
		}
	}
	return rows
}
