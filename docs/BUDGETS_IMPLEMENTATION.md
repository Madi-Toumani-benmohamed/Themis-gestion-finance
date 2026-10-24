# Budgets : compte rendu d'implémentation

Le diagnostic initial, établi avant modification et classé par criticité, est dans `BUDGETS_AUDIT.md`. Ce document décrit le résultat et ses limites. Les développements ont été répartis entre audit backend, analyse des récurrences/finance, frontend/UX ; la coordination et les validations transversales ont été réalisées par l'agent principal.

## Découpage des changements

| Lot | Branche | Périmètre |
|---|---|---|
| 1 | `fix/budget-financial-invariants` | Audit, isolation propriétaire, service budgétaire exact, sources, couverture et calculs partagés |
| 2 | `feat/budgets-ux` | Page de pilotage, gestion contextuelle des enveloppes, navigation et présentation de la revue |
| 3 | `feat/recurring-review` | Moteur/revue des récurrences, migration, adaptateur des consommateurs, pagination complète des sources et CI navigateur |

Les branches sont empilées : 2 cible 1 et 3 cible 2. Elles partent de `origin/main`. La fusion du simulateur arrivée pendant cette mission a ensuite été intégrée dans les trois branches ; ses tests restent présents. La section récurrences introduite dans le lot 2 nécessite le contrat serveur du lot 3 ; pendant la transition elle indique son indisponibilité explicitement. Déployer l'ensemble coordonné pour disposer de tous les parcours.

## Architecture et composants

- `backend/internal/api/budgets.go` : service `loadBudgetReport`, lecture cohérente des plafonds et transactions dans une transaction SQL de lecture, CRUD, ancien tableau `GET /api/budgets` conservé et nouveau `GET /api/budgets/report`.
- `transfer.go` : exclusion commune des catégories de transfert, de leurs enfants et des opérations normalisées `savings`. Les sommes de synthèse et les alertes budgétaires du Hero, du journal et des insights utilisent les mêmes périmètres.
- `handlers.go` et `transaction_sources.go` : provenance par enveloppe (`budget_id`) ou liste exacte de sources (`ids`), propriétaire toujours imposé par la session, pagination additive `offset`, aucune jointure de tags susceptible de multiplier les opérations financières.
- `frontend/src/pages/Budgets.jsx` et CSS local : synthèse à six indicateurs, mois explicite, couverture documentaire, enveloppes de dépenses séparées des transferts/épargne, création/édition/suppression dans des dialogues accessibles, recherche et filtre des dépassements.
- `money.mjs`, `budgetPageModel.mjs` et modèle du widget : centimes entiers, réconciliation des totaux, validation des périmètres, données absentes refusées au lieu de devenir des zéros certains.
- `App.jsx`/`Transactions.jsx` : navigation exacte vers les contributeurs, crédits visibles avec leur signe, pages de sources de 1 000 opérations et protection contre les réponses obsolètes. Les récurrences transmettent tous leurs identifiants, répartis en requêtes limitées à 1 000.
- `recurring_review.go` : moteur déterministe, preuves, cadence, natures économiques et décisions de revue. L'ancien `GET /api/recurring` garde sa forme via un adaptateur vers ce moteur ; les consommateurs partagés ne fondent plus `regular` sur la seule stabilité d'un montant.

## Définitions et invariants financiers

### Enveloppes

Une opération de dépense appartient à **une seule** enveloppe : celle de sa catégorie si elle existe, sinon celle du parent. La taxonomie existante autorise un niveau d'enfants. L'enveloppe parent n'additionne pas une deuxième fois les opérations déjà affectées à une enveloppe enfant.

- Alloué : somme des plafonds des seules enveloppes de dépenses.
- Consommé : somme exacte de leurs débits bruts observés, hors transferts internes et mouvements normalisés d'épargne.
- Restant : alloué − consommé ; ce n'est pas le solde bancaire.
- Dépassement : consommé **strictement supérieur** au plafond. L'égalité signifie « atteint ».
- Taux de consommation : consommé / alloué, indisponible sans plafond.
- Couverture budgétaire : débits observés affectés à une enveloppe / tous les débits observés éligibles, indisponible sans débits. Ce ratio mesure les opérations importées couvertes par des plafonds ; il ne mesure pas la complétude bancaire.

Les montants sont lus en décimal depuis PostgreSQL, convertis en centimes via `internal/money` et sérialisés en nombres JSON exacts. Les plafonds sous-centimes sont refusés, sans arrondi implicite. Les ratios sont des statistiques sans unité, pas des opérations monétaires en virgule flottante.

Les crédits sont exposés séparément. Aucun attribut de remboursement ou d'annulation fiable n'existe dans le modèle : un crédit de 10 € ne prouve pas qu'il rembourse un débit particulier. Les paires débit/crédit ne sont donc pas rapprochées automatiquement et la consommation est explicitement **brute**. Les opérations de signe nul n'inventent aucun mouvement d'argent.

Aucun dédoublonnage heuristique n'est ajouté : deux opérations similaires distinctes restent présentes. La déduplication historique appartient au pipeline d'import ; ses limites ne sont pas masquées par une suppression de sources dans Budgets.

### Périodes

Le mois sélectionné définit les dates effectivement interrogées. Le mois actuel s'arrête à aujourd'hui ; un mois terminé va jusqu'à son dernier jour. Un mois invalide/futur est refusé. Le mois courant ne revient pas automatiquement au dernier mois importé.

La complétude réutilise les intervalles des relevés validés par compte. Date de transaction, date d'import et preuve documentaire restent distinctes. Un mois vide n'est un zéro établi que si sa couverture est documentée. Aucun revenu absent avant le salaire de fin de mois ne déclenche une alerte Budgets. Aucune extrapolation uniforme des transactions importées n'est utilisée ; `projected` est conservé à `null`, `projection_available:false`.

Les plafonds existants restent des paramètres mensuels réutilisables **non versionnés** : les modifier change aussi la référence affichée sur les mois historiques. Le dialogue l'indique avant l'enregistrement. Les transactions historiques, elles, restent inchangées et sont interrogées sur leur propre mois.

### Récurrences et moyennes

La fenêtre d'analyse couvre jusqu'à douze mois civils, bornée à la fin du mois choisi ou à aujourd'hui. Chaque groupe distingue compte, catégorie/périmètre économique, sens du flux et marchand normalisé. Aucun rapprochement approximatif n'efface des opérations réelles ou ne fusionne les deux côtés d'un virement.

- Moyenne par opération : somme des montants absolus des sources / nombre d'opérations ; diviseur affiché.
- Montant habituel : médiane des montants par opération, arrondie au centime ; ce n'est pas une mensualité.
- Dernier montant : dernière source selon l'ordre date puis identifiant.
- Valeurs atypiques : conservées dans les sources et dans la moyenne ; la médiane offre un autre repère, sans exclusion cachée.
- Mois sans opération : aucun zéro n'est imputé à une moyenne **par opération**. Aucun total mensuel n'est dérivé de la somme de ces moyennes.
- Cadence : au moins trois observations aux intervalles compatibles (hebdomadaire, mensuelle ou trimestrielle). Plusieurs opérations le même jour sont conservées et peuvent rendre la cadence indéterminée.
- Confiance probable : cadence cohérente et intervalle documentaire couvert pour le compte depuis la première observation jusqu'à la borne choisie. Une nature économique détectée n'est pas une preuve de contrat.
- Échéance : uniquement estimée quand la cadence et la couverture le permettent ; pas de report artificiel d'une date théorique déjà dépassée.

Abonnements, charges fixes, dépenses variables et transferts/épargne restent séparés. La stabilité seule ne suffit pas. Les indices de retrait DAB et les mouvements d'épargne protégés ne deviennent pas des abonnements par une correction de détection. L'identité du marchand ou une catégorie peut suggérer une nature économique ; cette proposition reste « détectée » jusqu'à la revue humaine.

Confirmation, correction et ignorance sont des **métadonnées de revue**, indépendantes des montants et de la complétude. Une confirmation utilisateur ne remplit aucun trou documentaire. Ignorer masque la détection des listes usuelles et l'écarte des consommateurs partagés ; elle peut être réaffichée et réexaminée.

## Migration et retour arrière

`020_recurring_reviews.sql` ajoute uniquement `recurring_reviews(owner_id, recurrence_key, status, nature, updated_at)`. La clé de groupe est hachée ; aucun nouveau montant ou libellé bancaire n'est dupliqué dans cette table. Aucun relevé ou transaction existant n'est modifié.

Le rollback explicite est dans `backend/internal/db/rollbacks/020_recurring_reviews.sql`, hors du dossier auto-appliqué. Il supprime la table de décisions et son entrée de suivi ; les opérations bancaires sont conservées. Préserver les décisions de revue avant ce retour arrière si nécessaire, arrêter la version utilisant ce contrat, puis revenir à un code compatible : le serveur récent signale une erreur si la table requise est absente. Le test PostgreSQL applique réellement la migration et le rollback dans un schéma de test unique et vérifie le nombre inchangé de transactions.

## Validation et CI

- Tests Go unitaires et PostgreSQL isolés : propriété des catégories, création/modification/suppression, plafond vide/atteint/dépassé, parent/enfant sans double comptage, tags multiples, crédits, transferts hérités, épargne, source whitelist et pagination, mois historique/courant/absent, absence de salaire et erreur SQL.
- Tests récurrences : PEL et DAB, catégories/compte/sens distincts, montants variables et atypiques, occurrence multiple, intervalle manquant, confiance distincte de confirmation, échéance périmée, fin de mois, décisions persistantes et isolation propriétaire, migration/rollback, adaptateur legacy.
- Tests frontend : **30 tests Node** sur les modèles existants et nouveaux, réconciliation au centime et refus des données absentes/incohérentes.
- Test Chromium : création, modification, suppression annulée puis confirmée, changement de mois et réponse obsolète, chargement/erreur/reprise/vide, confirmation/correction/ignorance, navigation exacte vers les sources et pagination d'un groupe de 1 001 opérations.
- Géométrie : 1920×1080, 1280×800, 960×540 (espace CSS équivalent à 200 %), 768×1024, 390×844 et 320×700, sans débordement horizontal. Axe WCAG A/AA est exécuté sur la page en sombre, en clair et sur mobile.
- La CI conserve `go test -race ./...`, `go vet ./...`, les tests Node et le build frontend. Le filtre des PR inclut les bases des branches empilées pour tester aussi la chaîne de revue. Un job `budgets-ui` installe Playwright/Axe dans le répertoire temporaire du runner, sans dépendance runtime de l'application, et exécute le navigateur contre des API entièrement simulées. Le build Docker dépend aussi de ce contrôle ; aucun échec n'est ignoré.
- `actionlint` vérifie la configuration. Les commandes de validation locale et le build Docker sont exécutés avant publication des branches ; les résultats de GitHub Actions restent à distinguer des résultats locaux.

Reproduire le navigateur : démarrer Vite sur 127.0.0.1:5176, installer `playwright@1.64.1 @axe-core/playwright@4.14.0` dans `/tmp/themis-browser`, installer Chromium, puis exécuter `NODE_PATH=/tmp/themis-browser/node_modules node frontend/tests/budgets.browser.cjs`. `THEMIS_UI_URL` et `THEMIS_UI_ARTIFACTS` permettent de choisir serveur et captures. Toutes les API du test sont interceptées : aucune donnée financière réelle n'est lue.

## Limites, risques et prochaines itérations

1. Aucun historique des plafonds n'est disponible : une version des enveloppes par date serait une évolution métier distincte. Le comportement actuel est explicite dans l'UI.
2. Aucun lien bancaire fiable d'annulation/remboursement n'existe : conserver crédits et débits séparés jusqu'à un rapprochement documenté.
3. Les classements stockés peuvent être erronés ; les données sans catégorie sont accessibles pour correction. Les indices de nature de la revue ne modifient pas la classification transactionnelle de référence. Les anciennes opérations dont les métadonnées de normalisation manquent peuvent rester à examiner.
4. Les objectifs d'épargne ne sont pas modélisés : un plafond de transfert n'est pas présenté comme un objectif atteint. Ne pas inventer de progression d'épargne.
5. Le regroupement ne prouve pas un contrat : plusieurs contrats chez un marchand, surtout avec compte non identifié, peuvent rester indiscernables. En cas d'observations multiples ambiguës, ne pas attribuer de cadence certaine.
6. Les décisions de revue sont attachées au groupe, pas versionnées par mois. Leur date est conservée et ne modifie pas les opérations historiques.
7. La déduplication des imports et la fidélité des relevés ne sont pas certifiées par cette mission. Une couverture documentaire ne détecte pas un compte jamais déclaré.
8. Les autres analyses héritées peuvent encore employer des montants flottants à leur frontière de compatibilité ; le nouveau moteur, les totaux Budgets et ses modèles sont exacts. La projection bancaire multi-comptes de `/remaining` demeure un sujet distinct de la consommation d'enveloppes, malgré le raccordement prudent de ses récurrences.
9. La pagination des sources permet d'aller au-delà de 1 000 opérations ; elle n'est pas une photographie immuable entre plusieurs requêtes si des imports ou corrections surviennent simultanément.
10. Le contrôle automatique ne remplace pas une revue manuelle avec lecteurs d'écran, zoom navigateur natif et claviers virtuels sur appareils physiques.

## Résultats locaux constatés

- État final : `go test -race ./...`, tests PostgreSQL compris, `go vet ./...` et build Go **réussis**.
- Tests frontend : **30/30 réussis** ; build Vite **réussi** (avertissement préexistant sur la taille du bundle).
- Chromium : parcours CRUD, décisions de revue, sources exactes et pagination au-delà de 1 000 **réussis** ; **six formats** sans débordement horizontal ; **aucune violation Axe** relevée dans les contrôles sombre, clair et mobile.
- `actionlint` sur le workflow final : **réussi**.
- Image `themis-budgets-check:latest` : **build réussi**. Démarrage sur la base PostgreSQL temporaire vierge, migrations complètes dont 020, accès HTTP à `/` et `/api/health` : **réussis**. Cette image n'a pas remplacé l'application en cours ni utilisé sa base.
- Les résultats GitHub Actions seront ceux des PR publiées ; ils ne sont pas présentés comme déjà verts sur la seule base des validations locales.
- Les captures `/tmp/themis-budgets-ui` proviennent exclusivement des jeux de données synthétiques du test navigateur.
