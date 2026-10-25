# Analyse — réalisation et vérifications

Travail sur `feat/analysis-exploration`, les 24–25 octobre 2026. Diagnostic initial dans [ANALYSIS_AUDIT.md](ANALYSIS_AUDIT.md). Aucune lecture de données bancaires réelles pour l’audit ou les tests, aucune migration, aucune modification de montants, catégories ou imports historiques. Les tests emploient des données synthétiques isolées.

## Diagnostic et organisation retenue

La page initiale affichait neuf entrées sans mois commun et une synthèse dépendant exclusivement du LLM. Les agrégats de nature ignoraient des erreurs SQL, les comparaisons utilisaient des dénominateurs différents, les marchands dépendaient d’aujourd’hui et les tags étaient présentés comme une partition malgré leurs chevauchements. Des jointures de tags n’exigeaient pas le propriétaire des transactions. Le récit IA recalculait ses propres faits et utilisait un contexte d’administration global.

Les neuf capacités utiles restent accessibles dans quatre familles, avec au maximum une navigation secondaire :

| Famille | Contenu conservé et amélioré |
|---|---|
| Synthèse | Causes principales des sorties, catégories contributrices, qualité de l’affectation, contexte documentaire et lecture IA facultative |
| Flux financiers | Nature des flux, catégories multi-séries avec barres/courbes et mois/trimestre/année, concentration par marchand et panier moyen |
| Habitudes | Comparaison de catégories ; récurrences, montants inhabituels, services potentiellement redondants, évolutions par marchand/tag, motifs annuels |
| Pilotage | Projets possibles par tag, épinglage/masquage, détails catégories/marchands, croisement catégorie/tag, historique de tags, renommage/fusion/suppression des tags |

La synthèse explique les postes contributeurs et renvoie aux sources, sans copier les cartes du Dashboard. L’IA n’est pas sollicitée au chargement. Les projets par tag sont nommés selon leur fonction réelle : ce ne sont pas des budgets avec plafond. La gestion des tags annonce explicitement sa portée globale.

## Architecture et fichiers

- `backend/internal/api/analysis_report.go` : rapport déterministe partagé, contrôles de réconciliation et ratios exacts ; `intelligence.go` : adaptateurs des trois endpoints existants.
- `reliability.go` : lecteur réutilisable avec DB ou transaction, et variante sur un intervalle pour vérifier deux fenêtres de marchands. Le comportement de l’endpoint de fiabilité reste inchangé.
- `analytics.go`, `features3.go`, `tag_folders.go` : dates bornées, calculs exacts, jointures par propriétaire, totaux uniques, erreurs explicites ; concentration de projets sans multiplication des occurrences du même jour.
- `analysis_discoveries.go` et `discoveries.go` : repérages déterministes bornés ; les anciennes heuristiques non bornées sont remplacées par leurs versions contextualisées. La détection de projet reste dans Pilotage, sans prétendre qu’un projet est terminé.
- `narrative.go` : réutilise les faits du rapport. Aucun recours au Knowledge Graph ni à la note globale d’administration comme source personnelle.
- `Analyse.jsx/.css`, `Intelligence.jsx` : mois central, quatre familles, synthèse immédiate, contexte, tableaux et sources. `CategoryChart`, `TagAnalysis`, `TagFolders`, `TagManager` réutilisent les contrôles existants en corrigeant dates, états et accessibilité.
- `analysisPageModel.mjs` : présentation en centimes et vérification des invariants. `analysisResources.js` : annulation, rejet des réponses obsolètes et masquage des anciennes données dès le changement de paramètres.
- Fichiers partagés `App.jsx`, `Transactions.jsx`, `api.js` : transmission des filtres exacts de provenance (période/catégorie/tag/type/marchand). `TagPickerNavigation.jsx` expose l’état sélectionné au lecteur d’écran. Le filtre marchand reste visible et retirable dans Transactions et accompagne l’export via les filtres partagés.
- CI existante : le job navigateur ajoute Analyse après Transactions/Budgets/Simulation. Les nouveaux tests Node sont pris automatiquement par le glob existant ; les tests PostgreSQL utilisent le service isolé déjà configuré. Aucun contrôle retiré ou assoupli.

## Définitions, formules et limites de validité

Tous les montants sont des décimaux PostgreSQL lus en texte puis des centimes entiers Go. Les calculs de ratios/moyennes utilisent des entiers et des rationnels. Les sorties JSON monétaires ont deux décimales ; les totaux dépassant la précision exacte des centimes JavaScript sont refusés explicitement. React vérifie les sommes de la synthèse au centime avant affichage.

| Indicateur | Définition et formule | Périmètre et validité |
|---|---|---|
| Débits ordinaires | Somme des valeurs absolues des montants négatifs respectant `financialsql.ExpenseT` | Mois demandé, du premier jour à sa fin ou à aujourd’hui pour le mois courant ; source `transactions` du propriétaire |
| Crédits ordinaires | Somme des montants positifs avec la même exclusion des transferts/épargne | Crédits observés, pouvant inclure remboursements et annulations non rapprochés ; pas uniquement du salaire |
| Flux ordinaire net | Crédits ordinaires − débits ordinaires | Montant observé, jamais solde bancaire ni diagnostic de déficit définitif |
| Impôts identifiés | Débits ordinaires dont `label_kind=tax` | Classification normalisée disponible, pas garantie que tous les impôts soient identifiés |
| Autres débits ordinaires | Débits ordinaires hors `tax` | Avec le poste précédent, somme exactement égale aux dépenses |
| Épargne sortante | Débits `label_kind=savings` | Distincts des autres transferts ; ne prouve pas à lui seul le solde ou la propriété économique du compte destinataire |
| Autres transferts sortants | Débits exclus par le prédicat commun, hors épargne normalisée | Domaines de transfert et parent appartenant au même utilisateur |
| Total des sorties | Débits ordinaires + épargne sortante + autres transferts sortants | Postes disjoints, chaque écriture comptée une fois |
| Entrées internes/épargne | Crédits exclus des flux ordinaires par le prédicat partagé | Ne pas ajouter aux revenus ordinaires |
| Catégories et parts | Sommes par catégorie racine appartenant au propriétaire, enfants inclus ; part = montant / dépenses × 100 | Catégories étrangères traitées comme non documentées, source consultable via `domain=none` ; somme des catégories = dépenses |
| Panier moyen marchand | Total des débits du marchand / nombre de débits de ce marchand | Moyenne par opération, jamais moyenne mensuelle ; arrondi au centime, demi-centime à l’écart de zéro |
| Part marchand | Total marchand / total de tous les marchands de la fenêtre | Les 40 premiers sont affichés, mais le dénominateur utilise toutes les dépenses ; troncature signalée |
| Comparaison catégories | Dépense du mois vs moyenne des trois mois civils précédents, diviseur commun 3 | Seulement quatre mois intégralement documentés, sources attribuables ; les mois documentés sans transaction comptent pour zéro |
| Variation marchand | (Total fenêtre − total référence) / total référence × 100 | Deux fenêtres de même durée (1/3/6 mois dans l’UI, 1–24 dans l’API), intégralement documentées et sources attribuables ; nul si référence égale à zéro ou non comparable |
| Catégorie × tag | Total unique des débits de la catégorie, puis sommes indépendantes par tag | Les lignes multitag se chevauchent et ne s’additionnent pas ; barres et tableau remplacent le camembert ; nombre d’opérations multitag explicite |
| Séries historiques | Débits observés par intervalle, catégorie ou tag | Fenêtre de six mois ancrée sur le mois choisi ; graphiques descriptifs, couverture de chaque intervalle non certifiée ; absence ≠ zéro, trous non interpolés |
| Projet par tag | Débits ordinaires d’un tag jusqu’à la fin réellement interrogée du mois choisi | Historique importé à date, pouvant être incomplet et chevaucher d’autres tags ; pas coût total définitif ni date de fin de projet |

La moyenne par catégorie est arrondie par répartition des plus grands restes, avec départage stable par identifiant : la somme des moyennes de catégories égale la moyenne globale arrondie. Une base égale à zéro ne produit ni pourcentage infini ni libellé « nouveau ».

## Périodes et provenance

- Le mois est validé strictement et les mois futurs sont refusés. L’UI transmet systématiquement le mois ; les appels historiques sans mois des endpoints intel gardent un repli sur le dernier mois transactionnel non futur.
- Le rapport et sa couverture sont lus dans une transaction `REPEATABLE READ` en lecture seule. Les comparaisons partagent aussi un même instant de lecture. Les sources consultées ultérieurement dans Transactions peuvent naturellement refléter une correction effectuée entre les deux requêtes.
- La couverture réutilise les relevés validés et leurs intervalles par compte. Une dernière opération ne prouve jamais la couverture. Une opération sans relevé/compte attribuable empêche la certification complète du rapport.
- Un mois courant reste provisoire ; salaire de fin de mois et disponibilité différée des relevés sont rappelés sans qualification négative automatique.
- Un mois complet documenté sans opération peut afficher un zéro observé. Sans couverture documentaire complète et sans opération, la synthèse signale des données insuffisantes.
- Les accès aux sources utilisent les filtres existants ou le nouveau filtre marchand exact, avec des dates bornées. Les récurrences utilisent leurs identifiants exacts et la pagination par lots déjà disponible dans Transactions.

## Repérages et IA

Les récurrences réutilisent le moteur de revue existant (12 mois), avec nature économique, statut, confiance, médiane et sources. Les services potentiellement redondants ne prouvent ni contrats actifs ni écritures dupliquées. Les hausses de prélèvement comparent la dernière occurrence à la médiane des occurrences précédentes, sans affirmer un changement de tarif.

Les variations mensuelles de marchands/tags nécessitent quatre mois documentés. Les motifs annuels sont limités à 36 mois civils et décrivent des observations sur plusieurs années ; ils ne promettent aucune échéance ou dépense future. Les projets restent accessibles dans Pilotage. Les erreurs SQL ne deviennent plus une absence réussie de signaux.

Le LLM existant sélectionne/priorise des identifiants de constats calculés. Le serveur rejette toute sortie libre, identifiant inconnu, doublon ou JSON invalide ; le texte affiché est construit depuis les seuls constats autorisés. Couverture et réserves temporelles sont obligatoires. La réponse indique ce mode limité : il s’agit d’une lecture assistée, pas d’une analyse libre générant de nouveaux faits. Une indisponibilité du modèle laisse toutes les analyses déterministes utilisables. Aucun contexte personnel global ni mémoire KG ne remplace les transactions de référence.

## Validations exécutées

- Tests Go de tous les packages, avec PostgreSQL 16 isolé : réussis. `go test -race ./...` et `go vet ./...` : réussis dans Go 1.23 en conteneur.
- Dix nouveaux tests Go ciblent réconciliation, arrondi, absence/zéro, mois incomplet sans revenu, propriétaires, annulations/transferts, filtres, tags multitag, sources, erreurs SQL, concentration de projets, repérages bornés, fenêtres de marchands et contrat IA. Les tables sont temporaires, sans lecture de données réelles.
- `node --test frontend/src/components/*.test.mjs` : **39 tests réussis**, dont quatre nouveaux tests du modèle Analyse.
- `frontend/tests/analysis.browser.cjs` : parcours synthétique isolé réussi : navigation, mois, réponses obsolètes, filtres de sources, comparaison indisponible et tendance comparable, graphiques et tooltip, lecture IA facultative/indisponible, projets, renommage unique, annulation/suppression de tag, états erreur/vide et reprise.
- Six formats (1920×1080, 1280×800, 960×540, 768×1024, 390×844, 320×700), zéro débordement de document pour la synthèse. Toutes les vues détaillées vérifiées à 1920 et 320 pixels, tables défilant dans leur conteneur. Axe sans violation sur ces parcours ; thèmes sombre/clair contrôlés. Recomposition équivalente à un zoom 200 % vérifiée via 960 pixels CSS et facteur 2, sans prétendre remplacer un essai manuel de lecteur d’écran ou de zoom navigateur natif.
- Parcours existants Transactions, Budgets et Simulation : réussis, y compris leurs contrôles responsive et Axe.
- Build Vite et build de l’image applicative Docker : réussis. Démarrage de cette image sur PostgreSQL de test : santé et SPA HTTP200, API Analyse HTTP401 sans session après création d’un utilisateur de test inutilisable pour la connexion ; aucun service de production redémarré. Avertissement existant de bundle supérieur à 500 ko conservé, aucun seuil masqué.
- Semgrep `p/security-audit --error` : aucune alerte, y compris contrôle explicite des nouveaux fichiers Go non suivis ; Actionlint : réussi. Les résultats ci-dessus correspondent aux validations locales ; les contrôles GitHub sont consultables sur la PR de publication.

Reproduction : utiliser le service PostgreSQL de CI ou une base exclusivement de test via `THEMIS_TEST_DATABASE_URL`, puis exécuter les commandes backend ci-dessus. Pour les navigateurs, réutiliser l’installation Playwright/Axe isolée du job `budgets-ui`, démarrer Vite et lancer les quatre scripts sous `frontend/tests` ; le nouveau script accepte `THEMIS_UI_URL` et `THEMIS_ANALYSIS_ARTIFACTS`. Toutes les requêtes `/api/**` sont interceptées dans les scripts, y compris l’authentification et le LLM.

## Limites et prochaines itérations possibles

1. Les chiffres réels de l’utilisateur n’ont pas été audités : les tests prouvent les règles sur des fixtures, pas la complétude de ses relevés. La couverture existante dépend des comptes connus et peut être conservatrice pour des comptes inactifs sans date d’activité.
2. Il n’existe toujours pas de rapprochement confirmé entre remboursement/annulation et débit source. Les crédits restent séparés, sans compensation automatique ni suppression d’écritures.
3. Les séries historiques affichent des observations avec réserves ; leur complétude par intervalle n’est pas calculée. Une comparaison chiffrée reste réservée aux vues qui la prouvent.
4. Une similarité de marchand ne prouve pas une même relation commerciale. Les hypothèses de récurrence/saisonnalité sont contrôlables via leurs sources ; confirmation/correction restent dans la page Budgets existante.
5. La génération IA libre a été remplacée par une priorisation vérifiable. Une interprétation plus riche demanderait des preuves structurées et une validation supplémentaires ; aucun framework ou service ajouté.
6. Les archives d’analyses IA sauvegardées auparavant ne sont pas réécrites ni utilisées comme faits de cette synthèse. Leur fraîcheur et certains anciens calculs hors page Analyse restent hors du présent périmètre.
7. La requête de concentration des projets est bornée dans le futur et agrégée côté serveur, mais parcourt leur historique observé à date. Mesurer les requêtes sur un volume anonymisé représentatif avant de proposer un index ou une limite supplémentaire. Aucun benchmark de base distante n’est revendiqué.
8. Le bundle Vite reste volumineux ; un découpage du chargement du code est envisageable séparément, sans changer la fraîcheur des données.
9. Changements de sémantique justifiés et documentés : `average`, `delta`, `delta_pct`, `prev_total` et valeurs historiques manquantes peuvent être `null` lorsque la comparaison n’est pas établie. Les consommateurs de cette page ont été adaptés. Aucun nouvel endpoint et aucune migration ; paramètre `merchant` ajouté au filtre de transactions et donc à l’export.

Publication : branche de fonctionnalité `feat/analysis-exploration`, revue et fusion via une PR. La modification utilisateur locale de `AGENTS.md` est préservée hors du lot de publication.

## Évolution — courbes et périodes longues par tag (25 octobre 2026)

L’historique sous **Pilotage → Analyse par tag** reprend les éléments des courbes du Dashboard : aires non empilées, dégradés légers, traits lissés, axes discrets, légende et infobulle adaptées au thème. `ChartPrimitives.jsx` extrait les couleurs, dégradés, axes et infobulle déjà employés par les widgets ; Dashboard et historique partagent ainsi le même rendu. `TagHistoryChart.jsx` conserve les ruptures sans données et rend visibles les observations isolées. L’infobulle distingue explicitement « Non observé » de zéro. Les courbes ne s’additionnent pas lorsqu’un débit porte plusieurs tags.

Le choix **Période de l’historique** propose 6 mois, 1 an, 2 ans, 3 ans, 5 ans et 10 ans. Ces fenêtres se terminent au mois de référence sélectionné en haut de la page (jusqu’à aujourd’hui pour le mois courant). Le mode personnalisé permet de choisir deux mois puis d’appliquer les bornes ; il ne modifie pas la requête tant que la saisie n’est pas validée. La catégorie × tag située au-dessus reste explicitement rattachée au mois de référence. La sélection des tags est préservée lorsque la durée change.

Pour les séries de plus de 14 mois, la brosse Recharts permet de zoomer sur les observations déjà agrégées, sans requête réseau supplémentaire. L’intégralité de la période choisie est visible initialement ; les poignées se pilotent aussi avec les flèches du clavier. La période visible est annoncée, les montants restent accessibles dans le tableau complet et les liens sources transmettent le tag et les dates exactes.

`GET /api/analytics/tag-series` garde son URL et son contrat : le paramètre `months` accepte désormais 1 à 600 mois. Les dates personnalisées sont strictement validées, futures et inversées refusées ; une durée exige un mois de référence, qui ne peut pas être mélangé avec des bornes explicites. Une requête est limitée à 600 mois civils (50 ans) pour borner les séries renvoyées. Les appels historiques sans bornes conservent leur comportement ; l’interface utilise toujours une requête bornée. Aucun nouvel endpoint, migration, index ni dépendance runtime.

Une série bornée avec observations comprend tous les mois de sa fenêtre, y compris les mois où aucun tag sélectionné n’apparaît. Ces intervalles contiennent `null`, sans interpolation des données ni zéro automatique. `tagHistoryModel.mjs` vérifie les bornes de la réponse, les montants, l’ordre des mois et les longueurs ; des clés internes séparent les noms de tags des métadonnées du graphique. Les sommes restent calculées en NUMERIC/centimes côté backend, sur le propriétaire et hors transferts/épargne comme auparavant.

Validation : **43 tests Node réussis**, dont quatre nouveaux tests du modèle historique ; deux nouveaux tests Go, dont un PostgreSQL isolé sur dix ans, trous de données, filtres temporels, précision et propriété. Suite Go/PostgreSQL et `go vet` réussis. Le parcours Analyse couvre périodes longues, bornes personnalisées, erreurs de saisie, conservation des tags, infobulle, zoom clavier et Axe ; il vérifie aussi le mode long/personnalisé à 320 pixels. Les trois autres parcours financiers restent réussis. La complétude historique reste inconnue ; allonger la période ne crée aucune garantie de couverture ni nouvelle moyenne extrapolée.

Vérifications finales de cette évolution : `go test -race ./...` et `go vet ./...` réussis sur PostgreSQL de test ; Semgrep sans alerte sur les fichiers concernés. Image Compose reconstruite, conteneur applicatif remplacé, santé et interface HTTP200. Le JavaScript effectivement servi contient les nouveaux contrôles et le graphique ; comptes et volumes existants conservés.
