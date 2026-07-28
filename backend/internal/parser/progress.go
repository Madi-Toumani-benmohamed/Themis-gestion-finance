package parser

import "context"

// ProgressFunc reçoit l'étape courante du parsing, un détail lisible et un
// pourcentage approximatif (0-100). Le pourcentage est une estimation honnête
// fondée sur les étapes réelles — pas une animation cosmétique.
type ProgressFunc func(stage, detail string, pct int)

type progressKey struct{}

// WithProgress attache un rapporteur de progression au contexte. Le parseur
// l'appellera à chaque étape significative (extraction, conversion, OCR page
// par page…). Sans rapporteur, le parsing est strictement identique.
func WithProgress(ctx context.Context, f ProgressFunc) context.Context {
	return context.WithValue(ctx, progressKey{}, f)
}

// report émet une étape si un rapporteur est présent dans le contexte.
func report(ctx context.Context, stage, detail string, pct int) {
	if f, ok := ctx.Value(progressKey{}).(ProgressFunc); ok && f != nil {
		f(stage, detail, pct)
	}
}
