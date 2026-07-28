package parser

import (
	"fmt"
	"context"
	"log"

	"themis/internal/llm"
	"themis/internal/models"
)

// Parse applique la stratégie hybride, du moins coûteux au plus coûteux :
//  1. extraction native + parseur positionnel ; si le résultat est validé
//     comptablement, terminé
//  2. sinon, OCR (pdftoppm + tesseract) + parseur positionnel — nécessaire
//     pour les relevés sans couche texte (Crédit Agricole…), utile aussi
//     quand la couche texte est présente mais inexploitable
//  3. sinon, fallback IA (API Claude ou Ollama local) si configuré
//  4. sinon, meilleur résultat local disponible (marqué non validé)
func Parse(ctx context.Context, pdfData []byte, client llm.Client) (*models.ParseResult, error) {
	report(ctx, "Extraction du texte du PDF", "", 4)
	rows, err := ExtractRows(pdfData)
	if err != nil {
		return nil, err
	}
	nativeUseful := HasUsefulText(rows)

	// Le parseur positionnel est quasi gratuit : toujours le tenter sur
	// l'extraction native, même si elle semble pauvre — le seuil de texte
	// utile ne sert qu'à formuler la bonne erreur en l'absence d'OCR.
	var best *models.ParseResult
	if local, localErr := ParseLocal(rows); localErr == nil {
		if local.Validated {
			return local, nil
		}
		best = local
	}

	// Le natif n'a pas suffi : OCR
	var ocrErr error
	if ok, detail := OCRStatus(); !ok {
		if !nativeUseful {
			ocrErr = fmt.Errorf("ce PDF n'a pas de couche texte (caractères en bitmap) et l'OCR n'est pas opérationnel : %s", detail)
			log.Print(ocrErr)
		}
	} else {
		log.Printf("parseur natif insuffisant → extraction OCR")
		if oRows, oErr := ExtractRowsOCR(ctx, pdfData); oErr == nil {
			if oLocal, oLocalErr := ParseLocal(oRows); oLocalErr == nil {
				if oLocal.Validated {
					return oLocal, nil
				}
				// préférer le résultat OCR s'il a trouvé plus de transactions
				if best == nil || len(oLocal.Transactions) > len(best.Transactions) {
					best = oLocal
					rows = oRows
				}
			}
		} else {
			ocrErr = fmt.Errorf("échec de l'OCR : %w", oErr)
			log.Print(ocrErr)
		}
	}

	if best != nil {
		log.Printf("parseur local: %d transactions mais validation comptable KO", len(best.Transactions))
	}

	if client != nil {
		ai, aiErr := ParseWithLLM(ctx, client, PlainText(rows))
		if aiErr == nil {
			return ai, nil
		}
		log.Printf("fallback IA (%s): %v", client.Name(), aiErr)
	}

	// Dernier recours : résultat local non validé plutôt que rien
	if best != nil {
		return best, nil
	}
	if ocrErr != nil {
		if client == nil {
			return nil, fmt.Errorf("%v — installer les dépendances OCR ou configurer une IA (OLLAMA_URL ou ANTHROPIC_API_KEY) pour le fallback", ocrErr)
		}
		return nil, fmt.Errorf("%v (et le fallback IA a échoué)", ocrErr)
	}
	return nil, fmt.Errorf("aucune transaction détectée par le parseur local")
}
