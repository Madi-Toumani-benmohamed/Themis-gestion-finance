# Simulateur budgétaire — refonte du 24 octobre 2026

## Diagnostic et choix UX

La page initiale empilait tous les curseurs, même sans réglage actif. Les montants de référence apparaissaient sous chaque curseur, les résultats attendaient un clic « Calculer », les formulaires secondaires occupaient toujours de la place et les champs manquaient de labels. La synthèse financière n'était disponible que dans le panneau latéral. Les classes de tableau partagées imposaient des hauteurs et fonds adaptés aux transactions, pas à une comparaison compacte.

| Organisation étudiée | Avantage | Limite / décision |
|---|---|---|
| Grille permanente de curseurs | Plusieurs postes en parallèle | Beaucoup de contrôles visibles, colonnes étroites au zoom ; écartée |
| Groupes par domaine | Navigation par famille | Nécessite une taxonomie de regroupement fiable non fournie par cet endpoint ; écartée |
| Liste compacte avec réglages dépliables | Référence et réduction lisibles sans ouvrir ; curseur précis sur demande | Retenue, avec recherche et filtre des catégories modifiées |

La liste conserve l'ordre décroissant des dépenses fourni par l'API : les postes principaux sont accessibles en premier. Plusieurs postes peuvent rester ouverts. Le défilement de la liste est borné à 410 px sur desktop et 360 px sur petits écrans ; sa région reçoit le focus clavier. Les réglages restent stockés indépendamment des filtres et de l'ouverture des sections.

La synthèse supérieure affiche cinq indicateurs. Le panneau latéral présente le résultat estimé, la comparaison actuel/simulé et la décomposition de la variation. Son décalage sticky suit la hauteur réelle du header via ResizeObserver. Sur petits écrans, le panneau repasse dans le flux ; un rappel visuel fixe du résultat et de sa variation reste visible pendant les réglages, sur tablette comme sur mobile et dans la disposition à 200 %. Son alignement suit les dimensions réelles de la page. Les annonces aux lecteurs d'écran restent dans une seule région `status`.

Un graphique à deux barres a été évalué, puis retiré : il répétait les dépenses déjà comparées dans le tableau et repoussait la décomposition sous le premier écran. Les valeurs et leur différence sont plus utiles ici qu'une surface graphique supplémentaire.

Le revenu hypothétique, les nouvelles dépenses et le crédit sont dans des sections natives `details/summary`. Les curseurs fonctionnent au clavier et ont une saisie directe du pourcentage à deux décimales. Réinitialisation par catégorie, effacement du crédit, ajout/retrait de dépenses et réinitialisation globale sont conservés. Le calcul est immédiat : plus de bouton à cliquer après chaque réglage. Aucune requête réseau n'est nécessaire pendant les ajustements.

## Anomalies financières constatées et corrigées

1. La fenêtre glissante de trois mois à partir de la dernière transaction pouvait toucher quatre mois civils et inclure un mois courant sans salaire. La référence utilise désormais **trois mois civils consécutifs terminés**, ancrés sur le dernier mois importé et bornés avant le mois en cours. Les trois mois exacts sont affichés.
2. L'ancien diviseur des catégories dépendait du nombre de mois où chaque catégorie apparaissait. Une dépense présente un seul mois était comparée à un revenu moyenné sur une autre période. Toutes les sommes utilisent désormais le même diviseur **3**, y compris les mois sans opération de cette catégorie.
3. Les crédits de transferts étaient inclus dans le revenu automatique. Les deux côtés des transferts catégorisés `is_transfer` sont maintenant exclus, par les clauses métier existantes.
4. Le net était calculé sur des moyennes non arrondies, puis arrondi séparément des nombres affichés. Cela explique un écart possible d'un centime. Le cas `3 249,47 − 4 197,14` produit maintenant exactement **−947,67 €**, contrôlé en Go, en JavaScript et dans le navigateur.
5. Les erreurs SQL ne doivent pas produire une référence nulle : elles interrompent le calcul. Sans données ni couverture documentée, l'API répond `available: false` ; un zéro documentaire reste possible.
6. La personnalisation du revenu modifiait aussi la situation dite actuelle, rendant la variation trompeuse. La situation actuelle utilise maintenant toujours le revenu observé ; la situation simulée utilise l'hypothèse de revenu.
7. Les intérêts à taux zéro pouvaient devenir négatifs par multiplication de mensualités arrondies (100 € / 3). Le total à taux zéro conserve le principal ; une dernière échéance peut ajuster l'arrondi. Les intérêts estimés ne deviennent pas négatifs lorsque le principal est très petit.

## Formules et arrondis

- Les sommes sources sont calculées par PostgreSQL en `NUMERIC`, lues en décimal et converties via `internal/money`.
- Moyenne d'une catégorie : somme de ses débits sur la fenêtre / 3, arrondie au centime. Dépense de référence = somme de ces moyennes arrondies. Elle peut donc différer d'un centime de l'arrondi de la moyenne globale non ventilée : c'est une règle explicite, identique dans les deux versions du calcul.
- Revenu observé : somme des crédits hors transferts / 3, arrondie au centime. Il s'agit de crédits importés, pas exclusivement de salaires.
- Économie par catégorie : référence en centimes × réduction en points de base / 10 000, arrondie au centime. Les économies sont ensuite additionnées.
- Dépenses simulées = dépenses de référence − économies + nouvelles dépenses + mensualité de crédit.
- Net actuel = revenu observé − dépenses de référence. Net simulé = revenu hypothétique (ou observé) − dépenses simulées. Variation = net simulé − net actuel.
- Arrondi : au plus proche, demi-centime à l'écart de zéro. Montants entrés : deux décimales maximum, positifs ou nuls. Pourcentages : 0 à 100, deux décimales. Crédit : 1 à 600 mois, taux annuel nominal fixe, hors assurance et frais.
- Go utilise des centimes entiers et des rationnels `math/big`, y compris pour le crédit. JavaScript utilise des centimes entiers, des points de base et `BigInt` pour les quotients et puissances. Seul le formatage final en euros utilise Intl.

La référence est récupérée une fois à l'ouverture de la page. Le modèle frontend pur vérifie sa réconciliation avant de l'utiliser. Une réponse incohérente est affichée comme erreur, jamais corrigée silencieusement. Les formules frontend et backend ont des cas de test communs (dont crédit à 4,5 %, demi-centimes et cas signalé).

## API et fichiers

L'URL, les champs de requête et les champs numériques historiques de `POST /api/simulate` sont conservés. Ajouts : `available`, `reason` si absence, `observed_income`, `reference_periods`, `reference_complete`, `rounding`. Le corps de requête est borné ; les scénarios incohérents (catégorie répétée/inconnue, précision ou bornes invalides) sont refusés explicitement. Aucun changement de base ou de données existantes.

- `frontend/src/pages/Simulate.jsx` : orchestration, état du scénario et présentation ; petits composants de métrique et saisie monétaire.
- `frontend/src/pages/Simulate.css` : styles limités à la simulation, tokens du thème existant, responsive et focus visible.
- `frontend/src/components/simulationModel.mjs` : calcul pur et validation, indépendant de React.
- `backend/internal/api/simulation.go` : handler et calcul exact ; extrait de `features3.go` pour isoler ce domaine.
- Tests : `simulationModel.test.mjs`, `simulation_test.go`, `simulation_integration_test.go`, `frontend/tests/simulation.browser.cjs`.

## Validation

- Tests Node des modèles : calcul du cas à un centime, plusieurs catégories, réinitialisations, crédits et dépenses simultanés, revenu hypothétique, arrondis par catégorie, saisies invalides et référence incohérente.
- PostgreSQL isolé, tables temporaires : trois mois et même diviseur, catégorie présente un seul mois, mois sans salaire mais documenté, mois courant exclu, transferts des deux signes exclus, opérations similaires distinctes conservées, propriétaire isolé, couverture dégradée signalée, données absentes, zéro documenté et erreur SQL.
- Navigateur Chromium avec interception de **toutes** les routes `/api` : jeux synthétiques exclusivement, vingt catégories, accordéons au clavier, réductions indépendantes, filtre/recherche sans perte de réglages, reset, revenu/dépenses/crédit, état invalide puis reprise, absence de référence et référence incohérente.
- Géométrie vérifiée à 1920 × 1080 : cinq métriques, liste bornée, panneau de résultats complet dans le premier écran. Contrôles à 1280 × 800, 960 × 540 (espace CSS équivalent à 1920 × 1080 à 200 %), 768 × 1024, 390 × 844 et 320 × 700 ; absence de débordement horizontal.
- Axe WCAG A/AA sur la page, thèmes sombre/clair et mobile ; les résultats automatisés ne remplacent pas une revue complète avec lecteur d'écran et appareils tactiles physiques.

Les tests Node sont pris automatiquement par le glob frontend de la CI existante. Les tests Go et PostgreSQL sont pris par `go test -race ./...` et le service de test existants. Le script navigateur reste un contrôle explicite, sans dépendance ajoutée à l'application.

Commandes de reproduction :

```bash
cd frontend
node --test src/components/*.test.mjs
npm run build
# Dans un terminal séparé : npm run dev -- --host 127.0.0.1 --port 5176
# Outillage navigateur isolé (aucune dépendance runtime ajoutée au projet) :
npm install --prefix /tmp/themis-browser playwright@1.64.1 @axe-core/playwright@4.14.0
/tmp/themis-browser/node_modules/.bin/playwright install chromium
NODE_PATH=/tmp/themis-browser/node_modules node tests/simulation.browser.cjs
```

`THEMIS_UI_URL` permet de choisir le serveur et `THEMIS_UI_ARTIFACTS` le répertoire des captures (défaut `/tmp/themis-simulation-ui`). Les tests PostgreSQL exigent `THEMIS_TEST_DATABASE_URL` pointant vers une base de test isolée.

## Limites explicites

- Une fenêtre de trois mois ne prouve pas sa complétude. La couverture est évaluée avec les services existants par compte. Si une période n'est pas documentée, les résultats sont présentés comme **provisoires**, avec avertissement de sous-estimation possible. Le diviseur reste 3 ; l'absence d'opération importée n'est pas une preuve de revenu nul.
- L'indicateur « actuel » est un net moyen de flux importés, pas le solde bancaire courant. Aucun déficit temporaire n'entraîne un diagnostic négatif automatique.
- Les remboursements restent des crédits bruts, les virements mal catégorisés restent possibles. Le simulateur ne réinvente pas les règles de classification.
- La déduplication d'import reste celle du pipeline existant. Les opérations similaires distinctes sont conservées ; aucun `DISTINCT` ou dédoublonnage heuristique susceptible d'effacer de vraies dépenses n'est introduit ici. Les doublons historiques non reconnus ne sont pas réparés par la simulation.
- Le crédit est une estimation nominale, pas une offre bancaire. La dernière échéance peut différer à cause des arrondis. Le produit ne modélise ni assurance, ni frais, ni changement de taux.
- Les scénarios restent locaux et non persistés, comme auparavant. Une nouvelle importation nécessite de rouvrir/recharger la simulation pour renouveler la référence.
- Le zoom est testé par réduction de l'espace CSS disponible ; le zoom navigateur natif, les lecteurs d'écran et les claviers virtuels mobiles restent à contrôler sur appareils réels.

## Résultat de l'exécution locale

- `go test -race ./...` avec PostgreSQL temporaire : réussi, y compris les tests d'intégration du simulateur et les tests existants.
- `go vet ./...` et build du serveur Go : réussis.
- `node --test src/components/*.test.mjs` : **23 tests réussis**, aucun échec.
- Build Vite : réussi ; avertissement préexistant sur la taille du bundle.
- Contrôle Chromium : **6 formats réussis**, scénarios et clavier vérifiés, **0 violation Axe** dans les vérifications sombre, clair et mobile. Les textes secondaires sur fond accentué ont été ajustés localement pour respecter le contraste.
- Captures générées dans `/tmp/themis-simulation-ui` ; elles utilisent uniquement les jeux synthétiques du test navigateur.
- Aucun changement à `.gitignore` ou `AGENTS.md` préexistants, aucune donnée de production modifiée, aucune nouvelle dépendance runtime, aucune migration.
