# Analyse — audit initial et plan

Audit du code le24octobre2026, base `3746b29`, sans lecture des données bancaires réelles. Instructions AGENTS et CODEBASE_GRAPH consultées avant modification. Les scénarios de validation utiliseront des fixtures isolées.

## Rubriques réelles et décision

| Rubrique | Composant et sources | Constat vérifié | Décision |
|---|---|---|---|
| Synthèse | `Analyse.jsx` → `Narrative`, `/api/intel/narrative` | Aucun chiffre ni constat initial : le seul contenu attend un clic LLM. Backend choisit le dernier mois en base, pas celui demandé. | Synthèse déterministe immédiate ; lecture IA facultative et sourcée en complément. |
| Ce que Thémis a repéré | `Discoveries`, `/api/intel/discoveries`, récurrences | Heuristiques ancrées sur CURRENT_DATE/maxdate ; erreurs remplacées par vide ; comparaisons sans couverture. | Conserver les signaux vérifiables, borner l'observation et distinguer hypothèse/indisponibilité. |
| Où va ton argent | `NatureBreakdown`, `/api/intel/nature` | Exclusions des transferts incomplètes, QueryRow ignoré peut produire des zéros réussis ; calculs float et ambiguïté «tout transfert = épargne». | Réutiliser l'endpoint comme rapport analytique exact additif ; séparer dépenses, impôts identifiés, épargne normalisée et autres transferts. |
| Ce mois-ci vs d'habitude | `CategoryComparison`, `/api/intel/comparison` | Référence maxdate, historique non borné, diviseur par catégorie selon seuls mois avec opérations ; couverture absente. | Comparaisons sur trois mois civils communs documentés ; sinon valeurs comparatives nulles avec raison. |
| Chez qui tu dépenses | `MerchantsView`, `/api/intel/merchants` | Fenêtre glissante depuis aujourd'hui, sans borne haute, panier et variation float ; arrondi à l'euro. | Fenêtre1/3/6mois ancrée sur le mois choisi, montants exacts, panier par transaction explicite, tendances seulement comparables. |
| Budgets projet | `TagFolders`, `/api/tag-folders` et détail/préférences | Agrégats de tags, pas plafonds budgétaires ; liens sources par texte approximatif. Filtre du propriétaire des transactions manquant, comptage de concentration multiplié par les opérations du mêmejour, erreurs ignorées. | Conserver et nommer «Projets par tag» ; historique observé à date explicite, source par tag exact, montants uniques et revue de détection. |
| Dépenses par catégorie | `CategoryChart`, `/api/category-series` | Requête tout historique ; from/to existants non transmis ; IDs invalides ignorés peuvent élargir la requête ; loading assimilé à vide. | Conserver barres/courbes et filtres, fenêtre bornée ancrée, états distincts, catégories réutilisées, critères stricts. |
| Analyse par tag | `TagAnalysis`, `/api/analytics/tags-in-category` et `/tag-series` | Pas de mois transmis ; parts/Pie calculés sur SUMslices malgré chevauchement possible des tags ; erreurs assimilées à vide. | Barres/tableaux non additifs sur total unique de catégorie ; séries bornées, pas de fausse partition ni de moyenne supposée. |
| Gérer les tags | `TagManager`, `/api/tags` et mutations | Utilitaire global, pas analyse du mois ; erreurs remplacées par liste vide, chargements invisibles. | Garder accessible sous Pilotage/outils, portée globale claire, erreurs et confirmations visibles. |

## Anomalies prioritaires

- **P1 — isolation** : les dossiers de tags filtrent le propriétaire du tag sans exiger celui des transactions ; certaines jointures de catégories/tags ne vérifient pas le propriétaire. Les jointures doivent être concordantes partout.
- **P1 — temporalité** : aucune période commune au frontend ; les endpoints utilisent maxdate ou aujourd'hui et peuvent faire dépendre un mois historique d'opérations ultérieures.
- **P1 — comparaisons** : moyenne sur des mois observés différents et couverture non prouvée ; libellé «nouveau» déduit d'une absence de base. Ne pas comparer des couvertures inconnues ou un mois partiel à un mois complet.
- **P1 — récit IA** : `gatherFacts` duplique des agrégats float et peut transmettre «mois déficitaire» pour une période partielle. Le fallback perd les garde-fous ; une note globale d'administration est présentée comme contexte personnel. Ne pas l'utiliser comme donnée d'un propriétaire particulier.
- **P1 — erreurs et absences** : plusieurs erreurs SQL et réseau produisent des tableaux vides ou des zéros. Les états indisponible/vide/observé doivent être séparés.
- **P1 — tags** : une transaction multitag figure dans plusieurs lignes ; additionner ces lignes comme un total de catégorie ou les montrer en partition à100% est trompeur.
- **P2 — présentation/performance** : navigation9rubriques peu hiérarchisée, largeur limitée, vues vides ; chargements de référentiels répétés et pas de protection systématique contre les réponses obsolètes.

## Architecture cible et lots

1. Rapport déterministe partagé derrière `/api/intel/nature`, sans nouvel endpoint ni migration : période réellement interrogée, couverture existante, flux exacts et décomposition réconciliable. Contrats historiques conservés par champs additifs.
2. Mois central et quatre familles : **Synthèse**, **Flux**, **Habitudes**, **Pilotage**. Navigation secondaire compacte seulement quand utile ; détails montés à la demande ; outils globaux clairement identifiés.
3. Comparaisons, marchands, catégories, tags et projets : mêmes exclusions des transferts par propriétaire ; bornes calendaires strictes ; aucune donnée future utilisée pour un mois historique.
4. IA facultative : sélectionner/prioriser des constats déterministes identifiés et des questions d'investigation, sans produire de chiffres ou conclusions libres non vérifiables. Preuves et accès aux sources, pas de remplacement par le Knowledge Graph.
5. Tests Go/PostgreSQL, modèles Node, parcours navigateur, accessibilité et responsive ; régressions Budgets/Transactions/Simulation, builds et contrôles de sécurité existants.

Les agents se répartissent les services analytiques déterministes, l'UX et l'IA/repérages ; le coordinateur prend les agrégats tags/projets, l'intégration, les tests navigateur et le compte rendu. Les contrats sont partagés avant les modifications, les fichiers édités par un seul responsable.

## Critères financiers

Dépenses = débits ordinaires observés, crédits séparés (remboursements/annulations non rapprochés possibles), transferts hérités et épargne normalisée exclus des flux ordinaires. Sommes et moyennes monétaires exactes, arrondi explicite. Les décompositions ne comptent chaque écriture qu'une fois ; les catégories non documentées restent identifiables.

Un mois courant reste provisoire, notamment avant le salaire de fin de mois et en attente du relevé autour du21. Une dernière opération ou un mois calendaire écoulé n'établit pas la complétude. Un mois documenté sans transaction peut contribuer pour zéro à une comparaison ; un mois non documenté n'est jamais assimilé à zéro. Les graphiques historiques ne prouvent pas une évolution financière lorsque la couverture est inconnue.

Limites établies : pas de lien de rapprochement confirmé des annulations ni de date de comptabilisation distincte ; les textes d'analyse déjà sauvegardés n'ont pas d'empreinte de fraîcheur. Ils ne seront ni effacés ni utilisés comme faits de la nouvelle synthèse.
