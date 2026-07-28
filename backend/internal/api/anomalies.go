package api

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

// anomaly décrit un signal remonté spontanément à l'utilisateur.
type anomaly struct {
	Kind     string  `json:"kind"`     // subscription_up | new_recurring | unusual | subscription_gone
	Severity string  `json:"severity"` // info | warn | alert
	Title    string  `json:"title"`
	Detail   string  `json:"detail"`
	Amount   float64 `json:"amount"`
	Label    string  `json:"label"`
	MerchantKind string `json:"merchant_kind,omitempty"`
}

// anomalies inspecte les dépenses récentes et remonte ce qui mérite l'attention :
// abonnements en hausse, nouveaux prélèvements récurrents, dépenses
// inhabituellement élevées, abonnements disparus. Tout est déterministe et
// prudent : mieux vaut manquer un signal que crier au loup à tort.
func (s *Server) anomalies(w http.ResponseWriter, r *http.Request) {
	rec, err := computeRecurring(s.DB, s.ownerID(r), 2)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := detectAnomalies(rec)
	writeJSON(w, http.StatusOK, map[string]any{"anomalies": out, "count": len(out)})
}

// detectAnomalies applique les règles de détection à un ensemble de dépenses
// récurrentes et renvoie les signaux, triés par sévérité puis montant. Fonction
// pure (pas d'accès base) pour être partagée entre le handler /api/anomalies et
// le journal d'événements.
func detectAnomalies(rec []recurring) []anomaly {
	out := []anomaly{}
	seen := map[string]bool{} // évite qu'un même marchand génère plusieurs signaux

	for _, g := range rec {
		n := len(g.Amounts)
		if n < 2 {
			continue
		}
		last := g.Amounts[n-1]
		prev := g.Amounts[n-2]

		// 1. Abonnement en hausse : montants antérieurs stables, puis le dernier
		//    grimpe de 5 % à 60 %. Au-delà de 60 %, ce n'est pas une revalorisation
		//    d'abonnement mais une dépense ponctuelle (traitée en cas 3).
		if n >= 3 && stableSeries(g.Amounts[:n-1]) && last > prev*1.05 && last <= prev*1.6 && last-prev >= 1 {
			pct := (last - prev) / prev * 100
			out = append(out, anomaly{
				MerchantKind: g.MerchantKind,
				Kind: "subscription_up", Severity: "warn",
				Title:  fmt.Sprintf("%s a augmenté", g.Label),
				Detail: fmt.Sprintf("%s → %s par mois (+%.0f %%).", euro(prev), euro(last), pct),
				Amount: -last, Label: g.Label,
			})
			seen[g.Label] = true
			continue
		}

		// 2. Nouveau prélèvement récurrent : apparu il y a 2-3 mois seulement,
		//    montant stable → probable abonnement récemment souscrit. On exige
		//    que le dernier prélèvement soit récent (sinon c'est peut-être fini).
		if g.Regular && len(g.Months) >= 2 && len(g.Months) <= 3 && recentDate(g.LastDate, 40) {
			out = append(out, anomaly{
				MerchantKind: g.MerchantKind,
				Kind: "new_recurring", Severity: "info",
				Title:  fmt.Sprintf("Nouveau prélèvement régulier : %s", g.Label),
				Detail: fmt.Sprintf("Un paiement récurrent semble apparaître : %s par mois depuis %d mois.", euro(last), len(g.Months)),
				Amount: -last, Label: g.Label,
			})
			seen[g.Label] = true
			continue
		}

		// 3. Dépense inhabituelle : dernier montant très au-dessus de la moyenne
		//    du marchand (marchand non régulier, sinon c'est une hausse d'abo).
		if !g.Regular && g.AvgAmount > 0 && last > g.AvgAmount*2 && last-g.AvgAmount >= 30 {
			out = append(out, anomaly{
				MerchantKind: g.MerchantKind,
				Kind: "unusual", Severity: "warn",
				Title:  fmt.Sprintf("Dépense inhabituelle : %s", g.Label),
				Detail: fmt.Sprintf("%s cette fois, soit %.0f %% au-dessus de ta moyenne habituelle (%s).", euro(last), (last-g.AvgAmount)/g.AvgAmount*100, euro(g.AvgAmount)),
				Amount: -last, Label: g.Label,
			})
			seen[g.Label] = true
		}
	}

	// 4. Abonnement disparu : régulier, présent sur ≥3 mois, mais rien depuis
	//    plus de ~40 jours → résiliation ou incident. On ignore ceux déjà
	//    signalés autrement (pas de double message pour un même marchand).
	for _, g := range rec {
		if seen[g.Label] || !g.Regular || len(g.Months) < 3 {
			continue
		}
		if !recentDate(g.LastDate, 40) {
			out = append(out, anomaly{
				MerchantKind: g.MerchantKind,
				Kind: "subscription_gone", Severity: "info",
				Title:  fmt.Sprintf("%s ne prélève plus", g.Label),
				Detail: fmt.Sprintf("Dernier prélèvement de %s le %s. Résiliation ou incident ?", euro(g.Amount), frDate(g.LastDate)),
				Amount: -g.Amount, Label: g.Label,
			})
		}
	}

	// ordre : alert, puis warn, puis info ; montant décroissant à sévérité égale
	rank := map[string]int{"alert": 0, "warn": 1, "info": 2}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			a, b := out[j-1], out[j]
			if rank[a.Severity] > rank[b.Severity] || (rank[a.Severity] == rank[b.Severity] && a.Amount < b.Amount) {
				out[j-1], out[j] = out[j], out[j-1]
			} else {
				break
			}
		}
	}
	return out
}

// stableSeries indique si des montants sont proches entre eux (écart < 8 %),
// signe d'un abonnement à prix fixe avant une éventuelle hausse.
func stableSeries(amts []float64) bool {
	if len(amts) < 2 {
		return false
	}
	min, max := math.MaxFloat64, 0.0
	for _, a := range amts {
		if a < min {
			min = a
		}
		if a > max {
			max = a
		}
	}
	return max > 0 && (max-min)/max < 0.08
}

// recentDate indique si une date ISO est à moins de `days` jours d'aujourd'hui.
func recentDate(iso string, days int) bool {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return false
	}
	return time.Since(t).Hours() <= float64(days)*24
}

// cleanMerchant met en forme un libellé marchand : retire les préfixes
// bancaires (Prlv, Cb…) puis passe en capitalisation douce.
// « PRLV FREE TELECOM » → « Free Telecom ».
func cleanMerchant(s string) string {
	fields := []string{}
	for _, f := range splitFields(s) {
		if recStopwords[toUpper(f)] {
			continue
		}
		fields = append(fields, f)
	}
	if len(fields) == 0 {
		fields = splitFields(s)
	}
	for i, f := range fields {
		fields[i] = titleWord(f)
	}
	return joinSpace(fields)
}

func titleWord(s string) string {
	r := []rune(s)
	for i, c := range r {
		if i == 0 {
			if c >= 'a' && c <= 'z' {
				r[i] = c - 'a' + 'A'
			}
		} else if c >= 'A' && c <= 'Z' {
			r[i] = c - 'A' + 'a'
		}
	}
	return string(r)
}

func splitFields(s string) []string {
	out := []string{}
	cur := ""
	for _, c := range s {
		if c == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func toUpper(s string) string {
	r := []rune(s)
	for i, c := range r {
		if c >= 'a' && c <= 'z' {
			r[i] = c - 'a' + 'A'
		}
	}
	return string(r)
}

func joinSpace(fs []string) string {
	out := ""
	for i, f := range fs {
		if i > 0 {
			out += " "
		}
		out += f
	}
	return out
}

func frDate(iso string) string {
	if t, err := time.Parse("2006-01-02", iso); err == nil {
		mois := []string{"janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."}
		return fmt.Sprintf("%d %s", t.Day(), mois[int(t.Month())-1])
	}
	return iso
}

var _ = math.Round
