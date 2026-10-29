# Validation de la reprise du dashboard — 29 octobre 2026

Branche : `fix/cinematic-dashboard-review`, worktree `themis-dashboard-review`.
La fusion de `origin/main` (`fe53795`) était déjà engagée à la reprise. Les
conflits de fichiers sont résolus. Les corrections ont été validées avant publication.
Ces résultats sont locaux et ne constituent pas un résultat GitHub Actions.

## Résultats exécutés

| Contrôle | Résultat |
| --- | --- |
| `node --test src/components/*.test.mjs` | Réussi, dont les quatre cas du nouveau `dashboardModel` |
| `npm run build` | Réussi après correction finale ; avertissement sur le chunk JavaScript supérieur à 500 kB |
| `transactions.browser.cjs`, `budgets.browser.cjs` | Réussis : édition, remboursements, erreurs, clavier, responsive et Axe |
| `simulation.browser.cjs` | Réussi après correction de la double entrée de page ; six tailles, résultats visibles, zéro violation Axe |
| `analysis.browser.cjs`, `credits.browser.cjs` | Réussis : périodes, données exactes, interactions, responsive et Axe |
| `ai-configuration.browser.cjs` | Réussi : configuration, requêtes obsolètes, clavier, cinq tailles et Axe |
| `statements.browser.cjs`, `import-fidelity.browser.cjs` | Réussis : contrôles, import, confirmation, réimport, dialogues, responsive et Axe |
| `npm run test:motion` | Réussi : montants exacts, requêtes retardées, navigation rapide, absence de seconde entrée CSS après navigation native, préférences, états absents/erreur, Axe |
| `npm run test:cinematic` | Réussi : vrais tracés SVG achevés, quatre cœurs en mode automatique, disposition sérialisée/vide, périodes absentes, montants invalides, clavier, clair/sombre et 320–1440 px |
| `npm run test:e2e` puis reprise ciblée du scénario d'expiration | Les douze scénarios ont réussi : onze dans le dernier passage complet, le scénario d'expiration dans sa reprise ciblée |
| `go test ./...` | Réussi avec PostgreSQL isolé et `THEMIS_REQUIRE_PDF_TESTS=1` |
| `go vet ./...` | Réussi |
| `git diff --check`, `git ls-files -u` | Aucun défaut d'espacement ni fichier en conflit |

Les parcours navigateur ont été exécutés en plusieurs lots. Le premier
`npm run test:browser` s'arrêtait sur Simulation ; les suites suivantes et le
scénario corrigé ont ensuite été exécutés explicitement, sans désactiver de test.

## Échecs diagnostiqués et corrigés

- **Simulation :** le retrait du drapeau global de navigation réactivait
  `motion-page-in` après la transition native. Le panneau de résultat était alors
  décalé de 12 px, au-delà du premier écran. `PageTransition` mémorise désormais
  le mode d'entrée de la page. La règle CSS reste désactivée pour cette instance
  après la transition native ; la transition de secours reste disponible.
- **Référence du dashboard :** la capture attendue correspondait à l'ancienne
  composition. La nouvelle capture a été examinée puis enregistrée ; les quatre
  références visuelles E2E passent sans changement de seuil.
- **Configuration IA :** un premier passage a perdu son onglet pendant le
  rechargement Vite occasionné par la correction du composant partagé. La reprise
  sur les fichiers stabilisés passe ; aucune modification du scénario nécessaire.
- **Expiration de session E2E :** Go n'est pas installé sur l'hôte. Le lanceur
  Docker temporaire perdait l'entrée standard du processus lancé en arrière-plan.
  Le descripteur est maintenant conservé et transmis au conteneur. Le scénario
  réel réussit après correction de ce lanceur, sans changement d'authentification.

## Environnement et artefacts

Vite sur `127.0.0.1:5176`, Chromium Playwright et données intégralement synthétiques
pour les parcours simulés. Les E2E utilisent l'API Go et PostgreSQL 16 sur un port
local distinct, avec les bases temporaires `themis_e2e` et `themis_test`.
Go 1.23.12 et Poppler proviennent de l'image locale
`themis-dashboard-review-toolchain`. Aucun service bancaire n'est contacté et
aucun volume de données utilisateur n'est monté pour PostgreSQL.

Captures et mesures de cette exécution :

- `/tmp/themis-dashboard-debug/cinematic/` : bureau/mobile, clair/sombre, rapports Axe et échantillon de frames ;
- `/tmp/themis-dashboard-debug/motion/` : captures des transitions et `performance.json` ;
- `/tmp/themis-dashboard-*.log` : résultats locaux ;
- `frontend/tests/e2e/visual.spec.ts-snapshots/dashboard-linux.png` : référence versionnable mise à jour.

## Limites

Le build final produit environ 1 054 kB de JavaScript minifié (294 kB gzip) et
208 kB de CSS (37 kB gzip). Le découpage du bundle reste hors périmètre.

Sur les 180 intervalles échantillonnés par scénario, le p95 est de 16,8 ms pour
le changement de période et 33,4 ms pour la navigation, avec une pointe de
166,7 ms pendant celle-ci. Aucune animation permanente ne reste active au repos.
Ces mesures Chromium headless sur cette machine ne garantissent pas 60 FPS sur
un appareil physique. L'ouverture cinématique présente aussi des pointes sous
charge ; le profilage sur téléphone réel reste à faire.

Axe ne remplace pas une revue avec lecteur d'écran. Firefox, Safari et les
appareils physiques n'ont pas été contrôlés.

## Construction et lancement local demandés après le débogage

Le 29 octobre, l'image applicative a été reconstruite sans cache depuis ce
worktree, puis démarrée avec Docker Compose sur le port 8080. Les volumes
existants ont été réutilisés. Le frontend, son JavaScript et `/api/health`
répondent HTTP 200 ; le bundle servi contient le dashboard cinématique et la
correction `data-page-entrance`. L'image précédente reste disponible sous
`themis-gestion-finance-app:before-dashboard-debug-20261029`.
