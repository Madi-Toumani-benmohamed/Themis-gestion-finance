# Transactions — livraison et validation

Référence initiale : `a38d6e1`. Diagnostic et priorités : [TRANSACTIONS_AUDIT.md](TRANSACTIONS_AUDIT.md). Aucun relevé bancaire réel n'a été consulté pour les tests. Aucun montant, date, empreinte ou libellé brut historique n'est réécrit par la migration.

## Résultat UX

La page conserve ses outils de classement, règles, catégories, surnoms marchands et tri en pile. Elle les organise autour d'un titre/période, de trois indicateurs, de filtres serveur et d'une liste paginée de 100 opérations. Les outils de gestion secondaires sont repliables. La largeur desktop est portée à 1 600 px uniquement sur cette route.

Les journées restent regroupées. Les montants sont alignés à droite, les noms normalisés précèdent les libellés bancaires, et le compte est visible. Les détails complets, la provenance du relevé, la validation de sa période et l'origine connue de la catégorie sont consultables dans un drawer natif. Les catégories/tags se modifient sans toucher aux écritures originales. La catégorie ne devient « validée manuellement » que si l'utilisateur la modifie ou la valide explicitement ; éditer uniquement les tags ne change pas cette origine.

Le drawer conserve le focus, se ferme par Échap et rétablit un point de navigation après sauvegarde. Sur mobile, catégories et tags restent visibles sous le libellé et éditables dans le détail. Le nombre de colonnes des groupes journaliers suit le responsive : les colonnes masquées ne créent pas de colonnes fantômes réduisant le libellé. Les filtres se réorganisent et aucun débordement horizontal n'a été observé aux six tailles testées.

La sélection porte sur la page visible, pas sur toutes les pages. Une boîte récapitule le nombre et l'action avant confirmation. Les catégories sont modifiées par PATCH existant ; les ajouts/retraits de tags sont calculés côté serveur sous verrou sur l'état actuel. Les erreurs sont détaillées par identifiant et les succès ne sont pas annoncés comme annulés si une autre opération échoue.

## Définitions financières

Toutes les sommes du nouveau rapport proviennent des transactions PostgreSQL du propriétaire authentifié, après application des **mêmes prédicats** pour la liste, le dénombrement, les agrégats et le CSV.

| Indicateur | Formule et périmètre | Limites |
|---|---|---|
| Dépenses observées | `-SUM(amount)` des montants négatifs hors transferts/épargne, après filtres | Débits bruts, non diminués d'un crédit sans rapprochement confirmé. |
| Revenus et crédits | `SUM(amount)` des montants positifs hors transferts/épargne, après filtres | Un crédit n'est pas une preuve de revenu : peut comprendre remboursement/annulation. |
| Flux net hors transferts | crédits observés moins dépenses observées, au centime | Flux sur le périmètre interrogé, pas solde bancaire ni reste à vivre prévisionnel. |
| Sorties/entrées de transfert | Débits/crédits isolés par la même classification, toujours conservés dans la liste | Classification issue du domaine propriétaire, de son parent ou du marqueur normalisé `savings`. Une catégorie étrangère invalide ne peut définir cette nature. |
| Total d'opérations | `COUNT(*)` des sources filtrées, avant pagination | Ne multiplie pas les lignes à cause des tags. Aucun dédoublonnage de ressemblances. |
| Total d'une journée | Somme signée des écritures présentes **sur la page**, transferts inclus | Ce sous-total est explicitement celui de la page, jamais celui de toute la journée si celle-ci dépasse une page. |

Les calculs utilisent SQL `NUMERIC`, Go en centimes entiers et JSON décimal exact. Le frontend contrôle les centimes et l'égalité `revenus - dépenses = net` avant affichage. Des valeurs incohérentes ou hors limites provoquent une erreur explicite. Le cas `3249.47 - 4197.14` donne **−947.67**, jamais −947.66.

Les remboursements et annulations ne sont pas certifiés à partir du signe ou d'un libellé approximatif. Les deux lignes−22/+22 restent consultables ; seules des catégories de transfert déjà établies les excluent des flux ordinaires. Il n'existe actuellement aucun lien de rapprochement validé dans le modèle : le drawer le dit explicitement.

## Périodes et filtres

- Le mois courant du rapport est borné du premier jour à aujourd'hui ; les bornes réellement interrogées sont renvoyées et affichées.
- Les mois historiques couvrent leurs bornes calendaires. Une plage personnalisée du rapport nécessite deux dates valides et ordonnées.
- « Tous les mois » est une demande explicite `all=1`. Sans période ni identifiants ni `all=1`, le rapport prend le mois courant. Les anciennes listes gardent leur défaut historique pour compatibilité.
- Compte, catégorie/enfants, absence de catégorie, tag/absence de tag, débit/crédit/transfert/montant nul et recherche sont filtrés côté serveur. La recherche porte sur marchand, affichage, libellé brut, tags et montant signé exact quand le texte est un montant. `%` et `_` restent des caractères littéraux.
- Les réponses de liste et de synthèse sont lues dans une transaction SQL readonly RepeatableRead. L'export lit toutes les lignes de ce même périmètre, indépendamment de la pagination. Les requêtes obsolètes sont annulées et leurs réponses ignorées.
- La complétude mensuelle réutilise `/api/period`, donc les relevés validés et les comptes connus. Elle est conservative et globale aux comptes, même en présence d'un filtre compte. Pour une plage personnalisée ou tout l'historique, elle reste inconnue.
- Le mois courant demeure provisoire, y compris avant le salaire de fin de mois. Le dernier relevé disponible ou une dernière transaction ne prouve pas la couverture. Aucun déficit observé ne déclenche ici une qualification négative automatique.
- La navigation depuis budgets et récurrences conserve budget/identifiants exacts et dates. Au-delà de 1000 identifiants, la provenance est découpée en lots, chacun paginé de 100 lignes : **synthèse et export sont explicitement limités au lot courant**, pas annoncés comme le total global des sources.

## Contrats et migration

Aucun nouvel endpoint nécessaire. `GET /api/transactions` conserve son tableau historique. L'option `view=report` retourne :

```text
transactions, total, limit, offset, query_from, query_to,
summary: expenses, income, net, transfer_debits, transfer_credits, uncategorized_count
```

Les lignes du rapport ajoutent `amount_cents`, le compte, le relevé et ses bornes/validation, `financial_type`, `classification_source` et une `suggestion` éventuelle. Une catégorie introuvable ou étrangère est exposée comme non catégorisée dans cette vue, sans modifier sa référence stockée. La pagination est de 100 par défaut, bornée entre 1 et 1 000 ; les filtres sont validés, les erreurs ne deviennent pas une liste vide réussie.

`PATCH /api/transactions/{id}` conserve le succès204. Il accepte `domain_id` (dont null), `tags` (remplacement), `add_tags`, `remove_tags` et `discard_suggestion:true`. Remplacement et ajout/retrait ne se mélangent pas. Le verrou de transaction, la catégorie propriétaire et toutes les écritures de tags sont validés dans une même transaction SQL. Les champs financiers ou inconnus sont refusés. Une opération étrangère renvoie404.

Migration additive `021_transaction_classification.sql` :

- `transactions.classification_source` vaut historiquement `unknown`. Seules les nouvelles règles/appréciations explicites posent respectivement `rule`/`manual`. `ai` est réservé à une origine documentée, jamais inférée rétroactivement.
- `transaction_suggestions` stocke les propositions séparément des catégories/tags appliqués.
- Index `(owner_id,tx_date DESC,id DESC)` pour les périodes et l'ordre paginé.

Rollback manuel : `backend/internal/db/rollbacks/021_transaction_classification.sql`, avec la version applicative précédente. Il retire seulement les métadonnées/propositions et l'index, sans retirer les classifications que l'utilisateur a effectivement appliquées. Migration, rollback et réapplication sont testés dans un schéma isolé.

`POST /api/enrich/ai` reste disponible, mais sa sémantique est volontairement fiabilisée : il génère des **propositions** sans affectation automatique. `domains_set`/`tags_set` restent présents à zéro et `suggested` indique les propositions. L'import expose `ai_suggested` et conserve `ai_categorized:0`. L'option existante `AI_AUTO_CATEGORIZE` déclenche désormais ces propositions ; Admin et README l'expliquent. Reprendre la proposition dans le formulaire ne fait aucune écriture ; Enregistrer l'applique explicitement. Ignorer la retire ; une nouvelle demande IA peut produire une nouvelle proposition.

## Corrections de cohérence et confidentialité

- Surnoms marchands indexés par propriétaire ; réparation atomique des affichages anciennement mélangés, et version du normaliseur stockée avec la vraie clé globale de settings. Les surnoms exacts priment sur les motifs, puis les motifs les plus longs ; ceux-ci sont appliqués à l’import comme au redémarrage, sans traiter `%` ou `_` comme jokers SQL. Aucune valeur bancaire dans les nouveaux logs d'erreur.
- Motifs littéraux et regex partagés entre import, application et comptage des règles ; validation du domaine cible. L'import ne réapplique plus les tags aux anciennes écritures comme effet secondaire. Les expressions historiquement invalides ne bloquent pas le parsing ; elles doivent être corrigées avant leur réapplication explicite.
- Exclusions de transferts partagées avec l'analyse mensuelle et les séries de catégories. Les bornes annoncées par les séries sont effectivement appliquées.
- Analyse mensuelle : agrégats et moyennes exacts ; moyenne descriptive calculée sur les mois contenant des débits de la catégorie, dans les trois mois précédents, avec diviseur, nombre de débits et absence d'exclusion atypique explicités. Les mois absents ne sont pas des zéros. Aucune variation/conclusion comparative n'est fournie faute de preuve de couverture comparable.
- Recherche de transactions du chat corrigée, identifiants sources ajoutés ; synthèse principale et regroupements décimaux exacts. La qualité du résultat ne prétend plus être complète sans preuve documentaire. Le Knowledge Graph reste un contexte indicatif.
- Export protégé contre les formules de tableur ; le CSV est préparé dans un temporaire 0600 supprimé sur succès/échec. Une erreur SQL ou de lecture après une première ligne n'est pas renvoyée comme un CSV financier complet réussi.

## Fichiers et responsabilités

| Ensemble | Fichiers |
|---|---|
| Page et API frontend | `frontend/src/pages/Transactions.jsx`, `Transactions.css`, `frontend/src/api.js` |
| Présentation/calculs | `frontend/src/components/TransactionDetails.jsx`, `transactionPageModel.mjs`, `transactionPageModel.test.mjs` |
| Intégration existante | `frontend/src/App.jsx` (largeur ciblée, noms accessibles et bouton import natif), `pages/Admin.jsx` (libellé de l'option IA) |
| Listing/édition | `backend/internal/api/transactions.go`, `transactions_test.go`, `transactions_integration_test.go`, extraction depuis `handlers.go`, suppression du helper concurrent de tags dans `features2.go` |
| Règles/import | `transaction_rules.go`, `transaction_rules_integration_test.go`, `handlers.go`, `admin.go`, `categorize/categorize.go`, son test ; fixture import adaptée aux métadonnées |
| Export | `api/features.go`, `transaction_export_integration_test.go` |
| Cohérence/qualité | `financialsql/predicates.go`, `api/transfer.go`, `normalize_backfill.go`, `merchant_alias.go`, `features3.go`, `chat.go`, `transaction_quality_integration_test.go`, `chat_quality_test.go`, fixture budgets isolée |
| IA existante | `ai/enrich.go`, `analyze.go` et leurs tests PostgreSQL ; commentaire de configuration |
| Schéma | Migration021, rollback021 et `db/transaction_classification_test.go` |
| Validation/documentation | `frontend/tests/transactions.browser.cjs`, adaptation `budgets.browser.cjs`, `.github/workflows/ci.yml`, README, CODEBASE_GRAPH et rapports Transactions |

## Validations réelles

- `go test ./...` avec PostgreSQL16 isolé : réussi, y compris budgets, simulation, imports, récurrences, filtres, édition/export, analyse, isolation et migration/rollback.
- `go test -race ./...`, `go vet ./...`, compilation Go : réussis avec l'environnement PostgreSQL de test.
- Frontend : **35 tests Node réussis**, dont 5 nouveaux tests du modèle Transactions ; build Vite réussi.
- Transactions navigateur : recherche/filtres, total 107 avec pagination 100, réconciliation au centime, périodes, réponses obsolètes, détail et focus, sauvegardes/échec, origine manuelle, propositions reprises/ignorées sans mutation implicite, confirmations et erreurs partielles de masse, ajout/retrait de tags, déclencheur import natif.
- Six tailles :1920×1080,1280×800,960×540,768×1024,390×844 et320×700. Contrôles des largeurs utiles du tableau, de la lisibilité de la colonne mobile, des débordements et du drawer. Axe sans violation sur les parcours testés en sombre/clair. Un rendu 960×540 à facteur 2 complète le contrôle de disposition équivalent à 200 % ; aucun test manuel de zoom natif ni de lecteur d'écran n'est revendiqué.
- Non-régression navigateur Budgets et Simulation : réussie, six tailles chacune. Le test Budgets vérifie les liens d'enveloppe, les sources de récurrence et leur deuxième lot au-delà de 1000 identifiants.
- Image Docker construite ; démarrage et migrations sur PostgreSQL isolé, page statique, santé et rapport vide vérifiés. Aucun remplacement du service Compose utilisant les données réelles.
- Workflow validé par actionlint ; configuration Compose validée. Les tests Node/Go sont déjà collectés ; le job navigateur existant exécute désormais Transactions, Budgets et Simulation et reste requis avant le build Docker. Aucun résultat GitHub distant n'est revendiqué avant publication/exécution.

Le warning Vite existant concernant un bundle supérieur à 500 kB subsiste. Aucun seuil ni assertion n'a été relâché.

## Limites et suites proposées

1. **Rapprochements et doublons potentiels** : aucune nouvelle relation confirmée ni badge supposé. La paire débit/crédit ne prouve pas une annulation. Les empreintes historiques restent ambiguës pour deux opérations strictement identiques du même compte sur des imports distincts. Une revue dédiée nécessitera un modèle minimal de liens candidats/confirmés, sans fusion ni suppression automatique.
2. **Dates/provenance** : pas de date de comptabilisation distincte conservée aujourd'hui ; certains imports n'ont pas de compte ou de période documentés. Ces absences sont affichées, pas inventées.
3. **Origine historique des catégories** : inconnue et non reconstruite par heuristique. PayPal/DGFiP ne sont pas recatégorisés automatiquement. Une règle déterministe doit elle-même être revue si sa logique est trop large.
4. **Couverture** : complétude de compte filtré/plage multimois non certifiée ; sources supérieures à1000 découpées en lots explicités. Aucun total global fictif ni projection uniforme.
5. **Concurrence** : ajouts/retraits de tags et écritures d'une opération atomiques. Les éditions concurrentes d'une catégorie ou les remplacements explicites de tags suivent encore le dernier enregistrement ; pas de journal de versions ni contrôle optimiste. La modification multiple est une succession d'opérations atomiques, pas une transaction globale.
6. **Performance** : pagination OFFSET et recherche ILIKE paramétrée, adaptées au périmètre attendu. Pas de benchmark sur des données bancaires réelles, d'index trigramme, de cache distribué ni de curseur de pagination ajouté sans besoin mesuré.
7. **IA et reste de l'application** : les propositions restent faillibles ; les anciens textes d'analyse déjà sauvegardés ne sont pas réécrits. Tous les endpoints analytiques historiques ne sont pas migrés ici vers une représentation monétaire exacte, notamment certains anciens outils de comparaison. Ne pas extrapoler les garanties du rapport Transactions à ces calculs restants.
8. **Validation humaine** : tests Chromium/Axe et images de fixtures, pas de certification d'accessibilité ni de test exhaustif sur tous les navigateurs. Une revue métier des rapprochements et des règles historiques reste nécessaire.

### Critères pour une future revue de rapprochement

La prochaine itération devra produire des **candidats**, pas appliquer une compensation : propriétaire identique, comptes documentés, montants exactement opposés et non nuls, dates dans une fenêtre explicite (par exemple sept jours, configurable après vérification métier), marchand normalisé identique ou référence bancaire commune réellement présente. Une annulation sur le même compte doit en plus disposer d'un indice documentaire dans son libellé ; un transfert entre comptes propres nécessite deux comptes distincts appartenant à la même personne. Un libellé générique ou le seul montant ne suffit pas. Si plusieurs écritures satisfont les critères, aucune paire n'est sélectionnée automatiquement.

La validation humaine devra conserver les deux sources, le motif et les éléments vérifiés dans une métadonnée réversible, empêcher l'utilisation d'une écriture dans plusieurs rapprochements confirmés et partager le traitement avec les agrégats concernés. Les simples ressemblances de débit/date/marchand devront rester des doublons potentiels ; les opérations similaires mais distinctes ne seront jamais supprimées. Ces règles de conception sont documentées ici ; aucun moteur de rapprochement n'est déclaré implémenté dans cette livraison.
