// Package normalize transforme les libellés bancaires bruts en noms lisibles.
//
// Un libellé bancaire est écrit pour les systèmes, pas pour les humains :
// « X4519 Urssaf Marseille 09/01 » doit devenir « Cotisation URSSAF ».
//
// Le pipeline a trois étages :
//  1. Reconnaissance : un dictionnaire de marchands et d'opérations connues
//     (Amazon, Carrefour City, URSSAF, PEL…) court-circuite tout le reste.
//     Certaines entrées sont INTERPRÉTÉES, pas seulement renommées :
//     « Mens.pel 446904… » → « Versement PEL » (kind=savings).
//  2. Nettoyage : références de carte (X4519), dates (09/01), numéros de
//     compte, préfixes techniques (CB, PRLV SEPA, VIR…) sont retirés.
//  3. Mise en forme : Title Case avec préservation des sigles (EDF, SNCF…).
//
// Le libellé BRUT reste la source de vérité pour les règles de tri et la
// déduplication ; le nom d'affichage ne sert qu'aux humains.
package normalize

import (
	"regexp"
	"strings"
)

// Result est le produit de la normalisation d'un libellé.
type Result struct {
	Display  string // ce que l'utilisateur voit
	Merchant string // clé de regroupement (canonique pour les marchands connus)
	Kind     string // "" | savings | tax | benefit | health : opérations interprétées
}

type entry struct {
	needle   string // motif en MAJUSCULES, espaces normalisés
	word     bool   // exiger des frontières de mots (noms courts risqués : FREE, CAF…)
	display  string // nom affiché ("" = dépend du sens, voir dispNeg/dispPos)
	dispNeg  string // affichage si débit (versement…)
	dispPos  string // affichage si crédit (remboursement…)
	merchant string
	kind     string
}

// Le dictionnaire est ordonné : les motifs les plus spécifiques d'abord
// (CARREFOUR CITY avant CARREFOUR, FREE MOBILE avant FREE).
var dict = []entry{
	// ---- Opérations interprétées ----
	{needle: "URSSAF", word: true, display: "Cotisation URSSAF", merchant: "URSSAF", kind: "tax"},
	{needle: "PEL", word: true, dispNeg: "Versement PEL", dispPos: "Retrait PEL", merchant: "PEL", kind: "savings"},
	{needle: "LIVRET A", dispNeg: "Versement Livret A", dispPos: "Retrait Livret A", merchant: "Livret A", kind: "savings"},
	{needle: "LDDS", word: true, dispNeg: "Versement LDDS", dispPos: "Retrait LDDS", merchant: "LDDS", kind: "savings"},
	{needle: "LDD", word: true, dispNeg: "Versement LDD", dispPos: "Retrait LDD", merchant: "LDD", kind: "savings"},
	{needle: "DGFIP", display: "Impôts (DGFiP)", merchant: "DGFiP", kind: "tax"},
	{needle: "IMPOT", display: "Impôts (DGFiP)", merchant: "DGFiP", kind: "tax"},
	{needle: "TRESOR PUBLIC", display: "Trésor public", merchant: "Trésor public", kind: "tax"},
	{needle: "CAF", word: true, dispNeg: "CAF", dispPos: "Prestation CAF", merchant: "CAF", kind: "benefit"},
	{needle: "CPAM", display: "Remboursement CPAM", dispNeg: "CPAM", dispPos: "Remboursement CPAM", merchant: "CPAM", kind: "health"},
	{needle: "AMELI", display: "Remboursement CPAM", dispNeg: "CPAM", dispPos: "Remboursement CPAM", merchant: "CPAM", kind: "health"},
	{needle: "FRANCE TRAVAIL", dispNeg: "France Travail", dispPos: "Allocation France Travail", merchant: "France Travail", kind: "benefit"},
	{needle: "POLE EMPLOI", dispNeg: "Pôle emploi", dispPos: "Allocation Pôle emploi", merchant: "France Travail", kind: "benefit"},

	// ---- Marchands connus (renommage simple) ----
	{needle: "AMAZON PRIME", display: "Amazon Prime", merchant: "Amazon Prime"},
	{needle: "AMAZON", display: "Amazon", merchant: "Amazon"},
	{needle: "CARREFOUR CITY", display: "Carrefour City", merchant: "Carrefour City"},
	{needle: "CARREFOUR MARKET", display: "Carrefour Market", merchant: "Carrefour Market"},
	{needle: "CARREFOUR EXPRESS", display: "Carrefour Express", merchant: "Carrefour Express"},
	{needle: "CARREFOUR", display: "Carrefour", merchant: "Carrefour"},
	{needle: "AUCHAN", display: "Auchan", merchant: "Auchan"},
	{needle: "LECLERC", display: "E.Leclerc", merchant: "E.Leclerc"},
	{needle: "INTERMARCHE", display: "Intermarché", merchant: "Intermarché"},
	{needle: "LIDL", display: "Lidl", merchant: "Lidl"},
	{needle: "ALDI", word: true, display: "Aldi", merchant: "Aldi"},
	{needle: "MONOPRIX", display: "Monoprix", merchant: "Monoprix"},
	{needle: "FRANPRIX", display: "Franprix", merchant: "Franprix"},
	{needle: "GRAND FRAIS", display: "Grand Frais", merchant: "Grand Frais"},
	{needle: "PICARD", word: true, display: "Picard", merchant: "Picard"},
	{needle: "NETFLIX", display: "Netflix", merchant: "Netflix"},
	{needle: "SPOTIFY", display: "Spotify", merchant: "Spotify"},
	{needle: "DISNEY", display: "Disney+", merchant: "Disney+"},
	{needle: "DEEZER", display: "Deezer", merchant: "Deezer"},
	{needle: "CANAL", word: true, display: "Canal+", merchant: "Canal+"},
	{needle: "FREE MOBILE", display: "Free Mobile", merchant: "Free Mobile"},
	{needle: "FREE", word: true, display: "Free", merchant: "Free"},
	{needle: "ORANGE", word: true, display: "Orange", merchant: "Orange"},
	{needle: "SOSH", word: true, display: "Sosh", merchant: "Sosh"},
	{needle: "RED BY SFR", display: "RED by SFR", merchant: "RED by SFR"},
	{needle: "SFR", word: true, display: "SFR", merchant: "SFR"},
	{needle: "BOUYGUES TELECOM", display: "Bouygues Telecom", merchant: "Bouygues Telecom"},
	{needle: "PAYPAL", display: "PayPal", merchant: "PayPal"},
	{needle: "UBER EATS", display: "Uber Eats", merchant: "Uber Eats"},
	{needle: "UBER", word: true, display: "Uber", merchant: "Uber"},
	{needle: "DELIVEROO", display: "Deliveroo", merchant: "Deliveroo"},
	{needle: "SNCF", display: "SNCF", merchant: "SNCF"},
	{needle: "RATP", display: "RATP", merchant: "RATP"},
	{needle: "TOTALENERGIES", display: "TotalEnergies", merchant: "TotalEnergies"},
	{needle: "ENGIE", display: "Engie", merchant: "Engie"},
	{needle: "EDF", word: true, display: "EDF", merchant: "EDF"},
	{needle: "VEOLIA", display: "Veolia", merchant: "Veolia"},
	{needle: "DECATHLON", display: "Decathlon", merchant: "Decathlon"},
	{needle: "FNAC", display: "Fnac", merchant: "Fnac"},
	{needle: "DARTY", display: "Darty", merchant: "Darty"},
	{needle: "BOULANGER", word: true, display: "Boulanger", merchant: "Boulanger"},
	{needle: "IKEA", display: "IKEA", merchant: "IKEA"},
	{needle: "ACTION", word: true, display: "Action", merchant: "Action"},
	{needle: "ZARA", word: true, display: "Zara", merchant: "Zara"},
	{needle: "KIABI", display: "Kiabi", merchant: "Kiabi"},
	{needle: "SEPHORA", display: "Sephora", merchant: "Sephora"},
	{needle: "DOCTOLIB", display: "Doctolib", merchant: "Doctolib"},
	{needle: "BASIC FIT", display: "Basic-Fit", merchant: "Basic-Fit"},
	{needle: "MCDONALD", display: "McDonald's", merchant: "McDonald's"},
	{needle: "BURGER KING", display: "Burger King", merchant: "Burger King"},
	{needle: "KFC", word: true, display: "KFC", merchant: "KFC"},
	{needle: "STEAM", word: true, display: "Steam", merchant: "Steam"},
	{needle: "NINTENDO", display: "Nintendo", merchant: "Nintendo"},
	{needle: "PLAYSTATION", display: "PlayStation", merchant: "PlayStation"},
	{needle: "ITUNES", display: "Apple", merchant: "Apple"},
	{needle: "APPLE", word: true, display: "Apple", merchant: "Apple"},
	{needle: "GOOGLE", word: true, display: "Google", merchant: "Google"},
	{needle: "MICROSOFT", display: "Microsoft", merchant: "Microsoft"},
	{needle: "AIRBNB", display: "Airbnb", merchant: "Airbnb"},
	{needle: "BOOKING", word: true, display: "Booking.com", merchant: "Booking.com"},
	{needle: "VINTED", display: "Vinted", merchant: "Vinted"},
	{needle: "LEBONCOIN", display: "Leboncoin", merchant: "Leboncoin"},
	{needle: "LA POSTE", display: "La Poste", merchant: "La Poste"},
	{needle: "HARMONIE MUTUELLE", display: "Harmonie Mutuelle", merchant: "Harmonie Mutuelle"},
	{needle: "DIRECT ASSURANCE", display: "Direct Assurance", merchant: "Direct Assurance"},
	{needle: "MAIF", word: true, display: "MAIF", merchant: "MAIF"},
	{needle: "MACIF", word: true, display: "MACIF", merchant: "MACIF"},
	{needle: "MATMUT", display: "Matmut", merchant: "Matmut"},
	{needle: "MGEN", word: true, display: "MGEN", merchant: "MGEN"},
	{needle: "AXA", word: true, display: "AXA", merchant: "AXA"},
}

// Sigles à préserver en capitales dans le Title Case des marchands inconnus.
var acronyms = map[string]bool{
	"EDF": true, "GDF": true, "SNCF": true, "RATP": true, "SFR": true,
	"KFC": true, "GMF": true, "AXA": true, "MGEN": true, "TVA": true,
	"CIC": true, "BNP": true, "LCL": true, "HSBC": true, "ING": true,
}

var (
	reCardRef = regexp.MustCompile(`(?i)\bX\s?\d{3,6}\b`)
	reDate    = regexp.MustCompile(`\b\d{1,2}/\d{1,2}(?:/\d{2,4})?\b`)
	reLongNum = regexp.MustCompile(`\b\d{5,}\b`)
	reShortNum = regexp.MustCompile(`\b\d{2,4}\b`)
	// référence de transaction : « Ref-xxxx », « Ref xxxx », ou tout jeton
	// mixte lettres+chiffres d'au moins 5 caractères (identifiants opaques type
	// 927b2800281923c1, f24695gfgw…). Ces chaînes ne portent aucun sens humain.
	reRefKeyword = regexp.MustCompile(`(?i)\bref[-\s:]*[0-9a-z]+\b`)
	reSpaces  = regexp.MustCompile(`\s+`)
	rePrefix  = regexp.MustCompile(`^(?:CB|CARTE|PRLV(?:\s+SEPA)?|VIR(?:\s+SEPA|\s+INST|EMENT)?|PAIEMENT(?:\s+CB)?|ACHAT(?:\s+CB)?|RETRAIT(?:\s+DAB)?|TIP|CHQ|CHEQUE|FACTURE|ECH(?:EANCE)?|MENS(?:UALITE)?\.?)\s+`)
)

// jetons techniques sans valeur pour un humain, où qu'ils apparaissent
var stopTokens = map[string]bool{
	"SEPA": true, "PAYL": true, "EMIS": true, "RECU": true, "INST": true,
	"CB": true, "CARTE": true, "PRLV": true, "VIR": true, "VIREMENT": true,
	"PAIEMENT": true, "ACHAT": true, "RETRAIT": true, "DAB": true,
	"TIP": true, "CHQ": true, "CHEQUE": true,
	// Plateformes d'encaissement : elles préfixent le vrai commerçant
	// (« SumUp *Coiffure Léa » → « Coiffure Léa »). On retire la plateforme.
	"SUMUP": true, "ZETTLE": true, "IZETTLE": true, "SU": true,
	"STRIPE": true, "PAYPLUG": true, "MOLLIE": true, "LYDIA": true,
	"PAYPAL": true, "PAY": true,
}

// Label normalise un libellé bancaire brut. Le montant sert aux entrées
// interprétées dont l'affichage dépend du sens (versement vs retrait).
func Label(raw string, amount float64) Result {
	// clé de comparaison : majuscules, ponctuation → espaces (MENS.PEL → MENS PEL)
	up := strings.ToUpper(raw)
	up = strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return ' '
	}, up)
	up = " " + reSpaces.ReplaceAllString(strings.TrimSpace(up), " ") + " "

	// 1. Reconnaissance
	for _, e := range dict {
		hit := false
		if e.word {
			hit = strings.Contains(up, " "+e.needle+" ")
		} else {
			hit = strings.Contains(strings.ReplaceAll(up, " ", ""), strings.ReplaceAll(e.needle, " ", "")) ||
				strings.Contains(up, " "+e.needle+" ")
		}
		if !hit {
			continue
		}
		d := e.display
		if amount < 0 && e.dispNeg != "" {
			d = e.dispNeg
		} else if amount >= 0 && e.dispPos != "" {
			d = e.dispPos
		}
		if d == "" {
			d = e.merchant
		}
		return Result{Display: d, Merchant: e.merchant, Kind: e.kind}
	}

	// 2. Nettoyage
	c := reCardRef.ReplaceAllString(raw, " ")
	c = reRefKeyword.ReplaceAllString(c, " ") // « Ref-xxxx » explicite
	c = reDate.ReplaceAllString(c, " ")
	c = reLongNum.ReplaceAllString(c, " ")
	// séparateurs de plateformes collés au marchand : SumUp *Léa, Zettle_Garage
	c = strings.NewReplacer("*", " ", "_", " ", "/", " ").Replace(c)
	c = reShortNum.ReplaceAllString(c, " ") // codes courts résiduels (33, 4756…)
	c = reSpaces.ReplaceAllString(strings.TrimSpace(c), " ")
	for {
		next := rePrefix.ReplaceAllString(strings.TrimSpace(c), "")
		if next == c {
			break
		}
		c = next
	}
	words := []string{}
	for _, w := range strings.Fields(c) {
		clean := strings.Trim(w, ".,")
		if stopTokens[strings.ToUpper(clean)] {
			continue
		}
		if isOpaqueToken(clean) {
			continue // identifiant sans valeur humaine (927b28…, gfgw, bh4)
		}
		words = append(words, w)
	}
	if len(words) == 0 { // tout était du bruit : mieux vaut le brut que rien
		return Result{Display: strings.TrimSpace(raw), Merchant: strings.TrimSpace(raw)}
	}

	// 3. Mise en forme : Title Case, sigles préservés
	for i, w := range words {
		uw := strings.ToUpper(w)
		if acronyms[uw] {
			words[i] = uw
		} else if len(w) > 1 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		} else {
			words[i] = uw
		}
	}
	disp := strings.Join(words, " ")
	return Result{Display: disp, Merchant: disp}
}

// isOpaqueToken détecte un jeton sans valeur pour un humain : soit il mélange
// lettres et chiffres (identifiant type 927b2800, f24695gfgw), soit c'est une
// suite de lettres sans aucune voyelle d'au moins 3 caractères (gfgw, bh4 après
// retrait des chiffres → gfgw). Les vrais mots ont des voyelles.
func isOpaqueToken(w string) bool {
	if len(w) < 3 {
		return false
	}
	hasLetter, hasDigit, hasVowel := false, false, false
	for _, r := range strings.ToLower(w) {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r >= 'a' && r <= 'z':
			hasLetter = true
			if r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u' || r == 'y' {
				hasVowel = true
			}
		}
	}
	// mixte lettres+chiffres d'au moins 5 car → identifiant opaque
	if hasLetter && hasDigit && len(w) >= 3 {
		return true
	}
	// suite de consonnes sans voyelle (≥3) → charabia (gfgw, bh, zx…)
	if hasLetter && !hasVowel && !hasDigit {
		return true
	}
	return false
}
