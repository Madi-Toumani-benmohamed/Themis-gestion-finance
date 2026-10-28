# Thémis — prototype cinématique du Dashboard

## Périmètre et audit initial

Le prototype porte sur le Dashboard. La transition du conteneur de page et les
préférences d'animation sont partagées. Les pages métier conservent leur contenu,
leurs filtres et leurs interactions ; aucun endpoint ni calcul backend ne change.

L'audit de `AGENTS.md`, `CODEBASE_GRAPH.md`, `App.jsx`, `Dashboard.jsx`,
`FinancialSummary`, des widgets et de `ChartPrimitives` a établi :

- React 18, Recharts 2 et Vite sont déjà présents ; aucune bibliothèque de motion
  ou de WebGL n'est installée.
- Les styles se superposent dans `styles.css`, `refonte.css`, puis `cockpit.css`.
  Les anciens effets sont surtout des apparitions 2D. Une révélation `clip-path`
  globale masque temporairement les axes des graphiques ; le prototype l'annule.
- `Reveal` utilise déjà IntersectionObserver. Les animations JavaScript Recharts
  ne sont cependant pas désactivées par une simple media query CSS.
- L'ancien `useCountUp` montre des montants interpolés pendant 750 ms, y compris
  après un changement de période. Il est remplacé par les valeurs exactes : seule
  leur présentation s'anime.
- La navigation utilise un état `tab`, sans routeur ni gestion d'URL/Retour dans
  l'existant. Le `main` est remonté à chaque changement de page ou rafraîchissement.
  Le prototype conserve ce contrat, sans délai ni interception de l'historique.
- Les avertissements, l'incomplétude et le résultat des flux sont portés par
  `FinancialSummary` et son modèle indépendant. Ils restent visibles et inchangés.

## Direction artistique et stratégie

La scène associe un ciel bleu nuit, des lumières froides, un cristal facetté
original, trois orbites en perspectives différentes et des particules discrètes.
Le cristal est purement décoratif : sa forme, sa taille et ses mouvements ne
représentent aucune donnée financière. Aucun asset de Final Fantasy n'est utilisé.
Le thème clair conserve ses surfaces de lecture ; le thème sombre conserve ses
tokens et ses contrastes.

CSS 3D, SVG, IntersectionObserver et Web Animations API suffisent pour ce prototype.
Motion/Framer Motion ou GSAP ajouteraient ici une dépendance pour des séquences
finies et courtes. Three.js/React Three Fiber ajouteraient un renderer, un canvas
et un coût GPU sans besoin de vraie scène volumétrique interactive.

La chorégraphie installe le décor, approche la géométrie, révèle le titre, puis les
surfaces et les KPI. L'entrée principale dure moins de deux secondes. Les
particules ont une dérive finie, sans boucle permanente. Aucun écran de transition
ne bloque l'interaction. Les montants exacts sont présents dès leur réception.

## Composants réutilisables

| Élément | Responsabilité |
| --- | --- |
| `motion/Motion.jsx` — `MotionProvider` | Préférence persistante, priorité système et mémoire de session des révélations |
| `MotionControl` | Sélecteur Automatique / Cinématique / Réduites / Désactivées |
| `useSceneReveal` | Déclenche une amélioration au premier passage visible ; ne masque jamais le contenu en attente |
| `MotionSurface` | Arrivée en perspective, reflet et inclinaison limitée à ±1,5°, mémoire par `revealKey` |
| `PageTransition` | Arrivée spatiale de 320 ms, annulation au démontage, sans déplacement du focus |
| `motion/CinematicDashboard.jsx` | Géométrie SVG originale et mémoire de l'entrée principale |
| `motion/cinematic.css` | Tokens de durée/easing, composition, profondeur et fallbacks |
| `widgets/index.jsx` — `ChartStage` | Révélation unique de tracés exacts, sans morphing des données |
| `widgets/cinematicCharts.css` | Éclairage, anneau du donut, légende clavier et tooltips |

La mémoire de session empêche de rejouer toute la séquence au retour au Dashboard
ou après un rafraîchissement d'import. Un nouveau chargement complet de
l'application constitue une nouvelle ouverture. Les préférences seules sont
persistées en stockage local ; aucune donnée financière ne l'est par ces effets.

La légende du donut met en évidence un secteur à la souris comme au clavier.
« Vue plane » retire sa couronne décorative et son ombre ; les secteurs restent
toujours orthographiques, sans inclinaison ni modification de leurs valeurs.
Les courbes sont dessinées sur leurs coordonnées finales ; les axes restent
stables. Les animations Recharts natives sont désactivées, notamment pour éviter
une interpolation lors des changements de mois ou de la brosse.

Les réponses KPI et journal devenues obsolètes après un changement de mois sont
ignorées. Les anciennes données sont retirées pendant la nouvelle requête.

## Performance et accessibilité

- Aucun moteur WebGL, aucune dépendance supplémentaire, aucun nouvel appel API
  pour le décor. Le lockfile est inchangé.
- Les effets privilégient `transform` et `opacity`. Les tracés SVG utilisent
  `strokeDashoffset` uniquement pendant leur révélation ; les longueurs sont lues
  une seule fois. Les effets sur les données ne tournent pas en boucle.
- Le suivi du pointeur écrit des propriétés CSS au plus une fois par frame et
  ne provoque pas de rendu React. Il est désactivé pour les pointeurs tactiles.
  L'inclinaison est remise à zéro à la sortie, au scroll et au démontage.
- Le parallaxe du décor est limité à 18 px sur ordinateur ; son écoute du scroll
  est active uniquement lorsque le décor est visible. Il est désactivé sur tactile.
- Le mode automatique réduit les effets en cas de Save-Data ou de faible nombre
  de cœurs signalé par le navigateur. Ce signal est une précaution, pas une
  détection exhaustive des performances GPU.
- `prefers-reduced-motion` reste prioritaire, y compris si Cinématique est choisi.
  Les modes Réduites et Désactivées rendent la géométrie et les graphiques
  statiques. Les changements de préférence annulent les effets WAAPI actifs.
- Décor `aria-hidden`, sans interception du pointeur. Aucun compteur interpolé,
  aucune perte de focus, aucun défilement imposé, aucun flash agressif.
- Les observers, timers, listeners et frames sont nettoyés au démontage, y compris
  avec le StrictMode React existant.

## Validation ciblée

La validation locale suit le périmètre frontend ; les suites backend, sécurité,
Docker et E2E complètes restent à GitHub Actions.

```bash
# Terminal de développement, depuis frontend
npm run dev -- --host 127.0.0.1 --port 5173 --strictPort

# Autre terminal, depuis frontend
BASE_URL=http://127.0.0.1:5173 \
THEMIS_CINEMATIC_ARTIFACTS=/tmp/themis-cinematic \
npm run test:cinematic

npm run build
node --test src/components/financialSummaryModel.test.mjs \
  src/components/budgetWidgetModel.test.mjs
```

Le test utilise uniquement des données synthétiques et intercepte les API.
Il utilise Chromium installé (`THEMIS_CHROMIUM_PATH` peut sélectionner son chemin),
ou le Chromium Playwright de la CI. L'URL par défaut est celle des autres tests UI
en CI : port 5176. Il fait partie de `npm run test:browser`, déjà exécuté par la CI.
Pour comparer à une version antérieure avec les mêmes données, le mode
`THEMIS_CINEMATIC_BASELINE=1` capture l'existant puis quitte explicitement avant
les assertions du prototype ; ce mode ne valide pas les effets cinématiques.

Les contrôles couvrent les montants exacts dès leur apparition (MutationObserver),
les périodes différenciées, les avertissements incomplets, la navigation clavier,
les préférences persistantes, la priorité du système, l'absence de rejeu, la vue
plane du donut, les erreurs JavaScript, les débordements mobiles et Axe sur le
contenu principal en clair, sombre et mobile.

Résultats et limites mesurés : voir `CINEMATIC_VALIDATION.md`.

## Généralisation proposée après revue visuelle

Les effets sont volontairement concentrés sur le Dashboard pour permettre une
revue de la direction artistique avant de modifier simultanément toutes les pages.
La transition spatiale commune est déjà active sur la navigation existante.

| Page | Prochaine application des primitives | Garde-fou |
| --- | --- | --- |
| Analyse | Révélation des courbes et panneaux d'exploration | Garder axes et périodes fixes, conserver les avertissements |
| Transactions | Ouverture spatiale du détail et microtransitions des filtres | Aucun tilt des lignes, aucune animation par ligne sur les grandes listes |
| Budgets | Jauges avec éclairage et progression à l'entrée | Montants et dépassements exacts toujours lisibles |
| Simulation | Transition des panneaux d'hypothèses et projections | Ne pas interpoler deux scénarios comme une mesure réelle |
| Crédits & taxes | Relief décoratif autour des progressions | Aucune altération des échéances, paiements ou preuves |
| Assistant | Halo d'état et apparition des messages | État réel du traitement, pas de faux signal d'activité |

Avant chaque extension : contrôle visuel, test d'interaction ciblé, mode statique,
vérification sur mobile. Les sorties de page complexes sont différées : conserver
deux écrans montés pour une animation de sortie augmenterait ici les risques de
focus et de requêtes en double sans routeur dédié.
