# Dashboard — reprise de la PR #39

## Base et périmètre

La PR `4d907a0` partait de `1f60a5d`, avant le système Motion intégré par la PR #37 (`fe53795`). La reprise sur `fix/cinematic-dashboard-review` réconcilie les deux branches sans retirer les corrections de dialogues, de navigation ou de périodes de `main`. Aucun contrat API, calcul backend, migration ni service externe n'est ajouté. Motion 14.6 est déjà une dépendance de `main`.

## Corrections

- Un seul provider, une seule préférence pour les transitions de navigation, les graphiques, les nombres, les dialogues et le décor. Le choix explicite reste persistant ; la préférence système de mouvement réduit est prioritaire. Le nombre de cœurs ne désactive plus arbitrairement les animations automatiques. Save-Data est conservé.
- Chaque page mémorise son mode d'entrée : la fin d'une transition native ne déclenche plus une seconde entrée CSS. Le décalage de 12 px qui faisait déborder le résultat de Simulation est supprimé ; la navigation sans View Transitions garde son animation de secours.
- Le mois demandé est recherché explicitement ; les réponses obsolètes ne remplacent pas la période récente. Les anciennes valeurs encore affichées pendant le chargement portent leur mois. Aucun mois absent n'emprunte les chiffres d'un autre.
- `dashboardModel.mjs` partage les règles des deux présentations KPI : net calculé en centimes, absence distincte de zéro, comparaison uniquement entre deux mois consécutifs documentés complets, jours absents non inventés.
- Les sauvegardes de disposition sont sérialisées ; seule la dernière confirme l'enregistrement. Une disposition vide est conservée au rechargement. Une erreur reste visible.
- Les catégories sont identifiées par leur nom plutôt que par un ancien index de survol. La légende est utilisable au clavier ; les montants invalides produisent un message local au graphique, pas un écran blanc. Chaque instance de graphique possède ses propres identifiants de gradients.
- Les chiffres du journal conservent leurs centimes. Le sélecteur contient aussi un mois atteint en remontant au-delà des 24 derniers mois.

## Composition et animation

Le bandeau indigo a la hauteur de son contenu, y compris sur mobile. Sa géométrie décorative partage la palette de l'application. Les surfaces, espacements et rayons sont harmonisés en clair et en sombre. Le petit historique des flux occupe la colonne des points d'attention, réduisant la hauteur globale de la synthèse sans retirer une information.

La scène se construit en moins de deux secondes : tracé des orbites et des arêtes SVG, arrivée de la géométrie, passage lumineux fini. Les cartes apparaissent à leur entrée dans le viewport ; les graphiques révèlent leurs coordonnées finales ; les caractères des montants glissent vers leur valeur exacte. La disposition conserve les transitions de position lors des déplacements de widgets. Les cartes financières ne basculent plus sous le pointeur.

Les contrôles et valeurs restent présents pendant les animations. Le contenu ne dépend pas du déclenchement d'un observer. Une révélation terminée est mémorisée pour la session ; les animations annulées par StrictMode peuvent reprendre. Les effets sont nettoyés au démontage ou à la réduction des mouvements. Aucune boucle décorative permanente.

## Fichiers partagés et impacts

- `motion/index.jsx`, `preference.js`, `Motion.jsx` et `main.jsx` : préférences communes à toutes les pages ; validation de navigation, clavier, dialogues et modes réduits nécessaire.
- `FinancialSummary.jsx` / `HeroFlow.jsx` : position et couleurs de l'historique ; sources, trous calendaires et tableau accessible conservés.
- `widgets/index.jsx` : catégories, gradients et KPI ; doublons de widgets, brosse et changements de mois à couvrir.
- `Dashboard.jsx` : sélection mensuelle et sauvegardes de disposition.
- Les nouveaux styles sont limités au dashboard, sauf la désactivation des mouvements explicitement choisie par l'utilisateur.

## Vérification visuelle

Depuis `frontend`, servir le build avec `npm run preview -- --host 127.0.0.1 --port 5183`. Les tests `cinematic.browser.cjs` et `motion.browser.cjs` interceptent intégralement les API avec des données synthétiques. `THEMIS_UI_URL` sélectionne le serveur pour les parcours existants ; `BASE_URL` sélectionne celui du scénario cinématique.

Les contrôles vont de 320 à 1440 px, en clair/sombre, avec priorité système, erreurs, données absentes et écritures retardées. Les assertions d'animation attendent l'achèvement de vrais tracés SVG ; elles ne déduisent pas leur existence d'une classe CSS.

Les résultats exécutés et limites figurent dans `CINEMATIC_VALIDATION.md`. Les tests automatisés ne certifient ni tous les navigateurs ni tous les appareils physiques.
