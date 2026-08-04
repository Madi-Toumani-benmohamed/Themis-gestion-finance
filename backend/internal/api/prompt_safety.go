package api

import (
	"regexp"
	"strings"
)

// Longueur maximale du contexte personnel (caractères). Au-delà, on tronque :
// un contexte légitime tient en quelques phrases ; une charge d'injection est
// souvent plus longue et bavarde.
const maxContextLen = 1000

// Motifs d'injection de prompt connus. On les NEUTRALISE (remplacement par un
// marqueur visible) plutôt que de rejeter tout le texte : ça préserve la partie
// légitime du contexte tout en désamorçant la tentative. La liste vise les
// tournures classiques en français et en anglais.
var injectionPatterns = []*regexp.Regexp{
	// « ignore/oublie (toutes) les instructions (précédentes/ci-dessus) »
	regexp.MustCompile(`(?i)\b(ignore|oublie|oubliez|disregard|forget|ignorez?)\b[^.\n]{0,40}\b(instruction|consigne|règle|prompt|contexte|ci-dessus|précédent|previous|above|system)`),
	// « à partir de maintenant/désormais, TU es/agis… » — seulement quand c'est
	// suivi d'une adresse au modèle (sinon « désormais je paie… » est légitime).
	regexp.MustCompile(`(?i)\b(à partir de maintenant|désormais|dorénavant|from now on|starting now)\b[,\s]*\b(tu|vous|you|ignore|oublie|agis|act|comporte)`),
	// « tu es (maintenant) … » / « you are now… » (redéfinition de rôle)
	regexp.MustCompile(`(?i)\b(tu es|vous êtes|you are|act as|agis comme|comporte-toi|pretend|fais semblant)\b[^.\n]{0,25}\b(maintenant|désormais|now|un|une|a|an|sans limite|libre|different|nouveau|pirate|hacker|jailbreak)`),
	// balises de rôle système/assistant injectées
	regexp.MustCompile(`(?i)</?(system|assistant|user|instruction)s?>`),
	// « nouvelles instructions : » / « new instructions: » / « system prompt: »
	regexp.MustCompile(`(?i)\b(nouvelles? instructions?|new instructions?|system prompt|prompt système|override)\s*:`),
	// injonctions à révéler le prompt
	regexp.MustCompile(`(?i)\b(révèle|montre|affiche|reveal|show|print|repeat)\b[^.\n]{0,30}\b(prompt|instruction|système|system|consigne)`),
	// injonctions à inventer / mentir sur les chiffres
	regexp.MustCompile(`(?i)\b(invente|fabrique|falsifie|mens|invent|fabricate|make up|lie about)\b[^.\n]{0,30}\b(chiffre|montant|nombre|donnée|number|amount|figure|data)`),
}

// sanitizeUserContext applique trois défenses à un contexte fourni par
// l'utilisateur avant son injection dans un prompt :
//  1. limite de taille (troncature),
//  2. neutralisation des motifs d'injection connus,
//  3. suppression des caractères de contrôle qui pourraient perturber le prompt.
//
// Le résultat reste du texte lisible : on ne cherche pas à bloquer un contexte
// légitime, seulement à désamorcer les tentatives de détournement. La
// délimitation (le fait d'encadrer ce texte comme "données non fiables") est
// gérée séparément, à l'injection.
func sanitizeUserContext(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// 1. Limite de taille.
	if len(s) > maxContextLen {
		s = s[:maxContextLen]
	}
	// 3. Retirer les caractères de contrôle (sauf retour à la ligne et tab),
	//    qui n'ont aucune raison d'être dans un contexte et peuvent servir à
	//    masquer une charge.
	s = stripControlChars(s)
	// 2. Neutraliser les motifs d'injection.
	for _, re := range injectionPatterns {
		s = re.ReplaceAllString(s, "[contenu ignoré]")
	}
	return strings.TrimSpace(s)
}

func stripControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1 // supprime
		}
		return r
	}, s)
}
