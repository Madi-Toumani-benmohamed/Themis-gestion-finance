# Thémis — direction du mouvement

## Audit avant modification

Base : `1f60a5d`. React 18.3, Recharts 2.15 (version verrouillée), Vite 8. Aucune bibliothèque de mouvement dédiée. Les entrées CSS sont réparties entre `styles.css`, `refonte.css`, `cockpit.css` et les feuilles des pages. Plusieurs règles animent le même widget. `Reveal.jsx` utilise déjà IntersectionObserver, une fois par élément. `ScrollProgress` passe par un état React à chaque événement de scroll. Les compteurs des widgets repartent de zéro à chaque valeur ; ceux du Hero sont statiques.

La navigation est un état dans `App.jsx`, sans routeur, avec remontage explicite par `tab:refreshKey`. Les dialogues natifs et `useModal` portent la capture/restauration du focus. Les conserver est essentiel. Les graphiques historiques mémorisés conservent leur brosse ; leurs animations sont désactivées. Les séries de tags préservent les trous (`connectNulls=false`) et les points isolés. Les pages Simulation et Crédits n'offrent pas de séries temporelles de projection : ne pas en inventer.

Le Dashboard comporte un Hero (`FinancialSummary`), des indicateurs, un journal et une grille configurable. Plusieurs requêtes réagissent au mois indépendamment : toute transition doit identifier explicitement la période des valeurs affichées et ignorer une réponse devenue obsolète. Les calculs et contrats API restent ceux des composants existants.

## Direction : observatoire financier

Une ouverture en trois temps : typographie par lignes, éclairage du Hero, puis tracé des flux et révélation des repères. Palette, données et vocabulaire actuels conservés. Les ornements sont abstraits et ne représentent aucune valeur financière.

| Geste | Rythme | Usage |
| --- | --- | --- |
| Retour tactile | 90–140 ms | Compression des commandes, sélection |
| Déplacement | spring 420 / damping 38 / mass 0,8 | Indicateurs sidebar et onglets |
| Changement de vue | 220–360 ms, cubic-bezier(0.22,1,0.36,1) | Navigation, panneaux |
| Révélation | 420–620 ms, décalage 45–70 ms plafonné | Titres, cartes, journal |
| Données | 380–480 ms | Roulement des caractères des nombres exacts |
| Graphique | 700–1000 ms, décalage entre séries 70 ms | Tracé des chemins réellement calculés |

Les nombres sont formatés avant animation. Seuls les caractères se déplacent : pas de comptage depuis zéro, ni de montant intermédiaire inventé. Une valeur absente conserve son texte d'indisponibilité. Les barres de progression utilisent `scaleX`, sans dépassement élastique du ratio. Les chemins de courbes sont dessinés tels que calculés par Recharts ; les données, axes, trous et tooltips restent exacts.

## Choix techniques

Motion for React **14.6.0** est retenu pour les springs, éléments partagés, présence et révélations. C'est l'unique dépendance déclarée ajoutée. Chargement synchrone des fonctionnalités `domMax` par `LazyMotion` et composants `m` : le layout partagé nécessite ces fonctionnalités et le contenu ne dépend pas d’un téléchargement différé. Les transitions de navigation utilisent les snapshots natifs View Transitions lorsqu'ils sont disponibles, avec entrée immédiate de repli. Aucun ancien formulaire interactif n'est conservé derrière le nouveau. Les notifications et palettes deviennent inertes pendant leur sortie.

Les animations de chemins SVG utilisent Web Animations et un observer ciblé ; elles ne remontent pas le graphique et ne se rejouent pas pendant le déplacement d'une brosse. Les grands historiques limitent les effets. Aucun mouvement ambiant infini. Le mouvement réduit est lu dès le premier rendu et suivi quand la préférence change ; CSS, Motion et animations SVG doivent tous le respecter.

Les dialogues libèrent immédiatement leur état React et restaurent le focus à la fermeture. Une copie décorative, inerte, sans identifiants ni gestionnaires d’événements, termine la sortie en 140 ms puis est supprimée. Elle ne peut donc ni bloquer une nouvelle ouverture ni la fermer à la fin de l’animation précédente. Le passage au mouvement réduit termine aussi cette sortie immédiatement.

Documentation consultée : [Motion React](https://motion.dev/docs/react), [LazyMotion](https://motion.dev/docs/react-lazy-motion), [accessibilité](https://motion.dev/docs/react-accessibility), [transitions de layout](https://motion.dev/docs/react-layout-animations).

## Points de contrôle

- Focus et fermeture des dialogues, navigation rapide, période changée pendant une requête.
- Lecture des valeurs exactes pendant une animation et après changement de signe.
- Mois sans observation, première valeur inconnue, erreurs de chargement.
- Brosse de graphique, tableaux accessibles, thèmes clair/sombre, petits écrans.
- Animations finies, aucun travail récurrent après stabilisation, préférence de mouvement modifiée à chaud.

## Mesures initiales

Build production avant modification : JavaScript 892,10 kB / 241,67 kB gzip ; CSS 184,57 kB / 32,61 kB gzip. Les mesures navigateur comparatives utilisent uniquement les fixtures synthétiques du scénario motion. Les temps dépendent de la machine et ne constituent pas une promesse de 60 FPS sur tous les appareils.

## Implémentation

| Élément | Animation effective |
| --- | --- |
| Navigation | Snapshots de page, indicateur partagé sur spring, repli sans View Transitions, dernier choix prioritaire lors de clics rapides |
| Dashboard | Révélation du titre, passage lumineux fini et orbites du Hero, caractères monétaires, tracé de douze mois de flux, repères puis cartes au scroll |
| Graphiques | Dessin des chemins, aires et segments échelonnés, croissance des barres, halo de sélection, tooltip réactif ; brosse et trous conservés |
| Analyse | Indicateurs d’onglets partagés, transition du panneau, historiques et catégories animés ; pas de remontage au zoom |
| Transactions | Synthèse animée, transition groupée des résultats/pagination, sélection, drawer et fermeture avec retour du focus |
| Budgets | Enveloppes échelonnées, jauges sans dépassement, montant consommé animé, retour au clic |
| Simulation | Résultats et écarts animés, repères comparatifs dans le tableau, révélation des contrôles |
| Crédits/taxes | Cartes échelonnées, progression déclarée, changement du tri des échéances, repères de timeline sur l’historique existant |
| Relevés/admin | Progression d’import, résultats des contrôles, détail natif animé, transition des filtres, onglets de configuration et feedback |

Les calculs, modèles métier, contrats API et migrations ne sont pas modifiés. Les seules adaptations de données concernent l’affichage : conserver le mois associé aux anciennes valeurs, ignorer une réponse obsolète, afficher l’absence d’indicateur pour un mois absent et représenter les lacunes calendaires par `null` dans le nouveau mini-graphique.

Les fichiers partagés concernés sont `App.jsx`, `main.jsx`, `Toast.jsx`, `Reveal.jsx`, `ScrollProgress.jsx`, `components/{ChartPrimitives,useModal,FinancialSummary,TagHistoryChart,TransactionDetails}`, `widgets/index.jsx` et les styles du système. Leur validation doit couvrir navigation, focus, tooltips, budgets et vues mobiles, même pour un changement limité à une page.

## Vérification visuelle

Depuis le worktree `themis-cinematic-motion` :

```bash
cd frontend
npm ci
npm run dev -- --host 127.0.0.1 --port 5176 --strictPort
```

Le serveur de développement utilise le backend local `http://localhost:8080`, ou `THEMIS_API_PROXY_TARGET` si renseigné. Aucun déploiement n’est effectué par ces commandes.

Pour vérifier sans données personnelles, laisser Vite tourner et lancer dans un autre terminal :

```bash
cd frontend
npm run test:motion
```

Toutes les routes `/api/**` du navigateur de ce scénario sont interceptées avec des fixtures synthétiques. Les captures clair/sombre/mobile et `performance.json` sont écrites dans `/tmp/themis-motion` (configurable avec `THEMIS_MOTION_OUTPUT`). `THEMIS_UI_URL` permet de tester un build servi par `vite preview`. `THEMIS_MOTION_BASELINE=1` active les interactions communes à la version initiale pour la comparaison de performances.

Parcours visuel : ouvrir le dashboard ; changer de mois rapidement ; faire défiler les graphiques ; déplacer la brosse ; ouvrir une transaction au clavier puis fermer avec Échap ; modifier un paramètre de simulation ; basculer clair/sombre et émuler le mouvement réduit avant puis après chargement.

## Limites explicites

- La navigation utilise une entrée de page de repli lorsque View Transitions n’est pas disponible. Le repli est testé dans Chromium ; Safari et Firefox restent à vérifier sur leurs moteurs natifs.
- Les courbes se révèlent selon leur géométrie réelle. Les morphings arbitraires de séries, d’axes ou de périodes incompatibles ne sont pas activés.
- Simulation et Crédits ne disposent pas de séries temporelles fiables de projection/dette dans leur contrat actuel. Aucun historique ou échéancier financier n’a été inventé ; les résultats, progressions et historiques existants sont animés.
- Les tests visuels E2E utilisent le mouvement réduit pour comparer un état statique déterministe. Le scénario motion teste séparément les animations réelles. Il fige uniquement `Date`, car l’horloge Playwright peut désynchroniser `performance.now()` et la timeline native après rechargement.
- Le bundle JavaScript reste volumineux ; le dépassement du seuil Vite de 500 kB existait déjà. Les mesures chiffrées ci-dessous rendent visible le coût du système.

## Reprise du 27 octobre 2026

Le chantier est isolé dans `/home/localadm/projets/themis-cinematic-motion`, branche `feat/cinematic-motion`. La reprise a corrigé deux régressions de l’intégration :

- Une fermeture différée du dialogue supprimait une réouverture immédiate. `useModal.js`, partagé par Relevés et Transactions, ferme maintenant l’état actif sans attendre l’animation.
- Le conteneur du diagramme de répartition se réduisait à une largeur nulle dans la grille CSS. `motion.css` lui rend la largeur disponible ; les secteurs portent également leur catégorie et leur montant comme nom accessible. Le scénario motion vérifie leur présence sur desktop et mobile.

Les contrôles locaux restent ciblés. Les suites Go, les modèles frontend, les autres parcours navigateur et les E2E complets sont déjà déclenchés par `.github/workflows/ci.yml` ; `test:motion` fait désormais partie de `test:browser`, donc du job E2E. La CI se déclenche sur une pull request vers `main` ou manuellement ; le push d’une branche de fonctionnalité seul ne la lance pas. Ses résultats restent à obtenir pour cette branche.

### Validation de la reprise

| Commande / contrôle | Résultat |
| --- | --- |
| `npm run build` | Réussi ; avertissement de taille Vite documenté ci-dessous |
| `THEMIS_UI_URL=http://127.0.0.1:5177 npm run test:motion` | Réussi sur le build de production : périodes rapides, données absentes, montants exacts, brosse, secteurs de répartition, mouvement réduit, Axe clair/sombre, mobile et absence d’animations perpétuelles |
| `node tests/statements.browser.cjs` | Réussi : filtres, contrôles, validation, annulation puis réouverture immédiate, suppression, clavier, cinq largeurs et Axe |
| `THEMIS_UI_URL=http://127.0.0.1:5177 node tests/transactions.browser.cjs` | Réussi : détails, retour du focus, modifications, remboursements, erreurs, six largeurs et Axe |
| `npm run test:e2e -- tests/e2e/visual.spec.ts --grep dashboard --update-snapshots` | 1 scénario réussi ; référence du dashboard régénérée et inspectée, sur une base PostgreSQL dédiée et des relevés synthétiques |
| `git diff --check` | Réussi |

Le rapport retrouvé de la session précédente indiquait 12 E2E réussis, avec mise à jour des captures. La suite complète n’a pas été relancée à la reprise. Le scénario visuel ci-dessus met à jour la référence ; la comparaison de cette nouvelle référence sera exécutée par la CI.

Les premiers essais ont reproduit les régressions décrites plus haut, puis détecté les noms accessibles manquants des secteurs restaurés. Les autres interruptions locales provenaient du rechargement Vite lors de l’écriture du rapport HTML et d’une permission de lecture dans l’outil PDF temporaire ; les essais finaux ont utilisé le build de production et l’outil corrigé. Aucun test n’a été désactivé ou assoupli. Le conteneur PostgreSQL de validation a été supprimé après usage.

## Mesures finales

Build Vite de production du 27 octobre 2026, tailles décimales hors polices :

| Ressource | Base `1f60a5d` | Version finale | Écart gzip |
| --- | --- | --- | --- |
| JavaScript | 892,10 kB / 241,67 kB gzip | 1 047,82 kB / 292,64 kB gzip | +50,97 kB (+21,1 %) |
| CSS | 184,57 kB / 32,61 kB gzip | 195,72 kB / 34,99 kB gzip | +2,38 kB (+7,3 %) |

Le coût supplémentaire inclut Motion et les nouvelles primitives. Le découpage du bundle reste une optimisation possible ; l’avertissement Vite des chunks de plus de 500 kB est conservé.

Échantillon Chromium headless sur le build servi par `vite preview`, 180 intervalles de frames par interaction, fixtures synthétiques :

| Interaction | Médiane | 95e percentile | Maximum | Frames > 33,5 ms |
| --- | --- | --- | --- | --- |
| Changement de mois | 16,7 ms | 16,8 ms | 33,4 ms | 0 / 180 |
| Navigation Dashboard → Simulation | 16,7 ms | 33,4 ms | 133,4 ms | 6 / 180 |

Aucune tâche longue détectée par `PerformanceObserver` pendant ces deux échantillons ; aucune animation encore active après stabilisation. La médiane correspond à environ 60 FPS, mais les pointes de navigation ne permettent pas de promettre 60 FPS constants. Aucune comparaison de cadence avec la version initiale n’a été relancée pendant cette reprise.

Les captures clair/sombre/mobile et les mesures brutes sont disponibles localement dans `/tmp/themis-motion/` jusqu’au nettoyage des fichiers temporaires. Le scénario accepte `THEMIS_UI_URL=http://127.0.0.1:5177 npm run test:motion` pour vérifier un build servi par `npm run preview -- --host 127.0.0.1 --port 5177 --strictPort`. Ce mode évite les rechargements HMR déclenchés par l’écriture des rapports HTML Playwright.
