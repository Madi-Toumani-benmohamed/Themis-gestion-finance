# Validation du prototype cinématique

Branche : `feat/cinematic-dashboard`. Aucun merge automatique.
Le résultat décrit cette machine et le prototype ; aucun statut GitHub Actions
n'est déduit des exécutions locales.

## Résultats locaux

| Contrôle | Résultat |
| --- | --- |
| `npm run build` | Réussi ; avertissement existant sur le chunk JavaScript supérieur à 500 kB |
| Tests de modèles `financialSummaryModel` et `budgetWidgetModel` | Deux fichiers exécutés avec succès, aucun échec |
| `BASE_URL=http://127.0.0.1:5173 npm run test:cinematic` | Réussi avec Chromium système 154 et API entièrement simulée |
| Montants | Valeurs exactes dès insertion dans le DOM, vérifiées par MutationObserver ; aucun montant interpolé |
| Périodes | Mois courant et précédent différenciés, avertissements conservés, réponses arrivant hors ordre ignorées |
| Navigation | Clavier et focus conservés, entrée principale non rejouée au retour au Dashboard |
| Préférences | Off persistant après rechargement, changements live, priorité de reduced-motion système |
| Donut | Vue plane actionnable au clavier, secteurs nommés, légende accessible |
| Mobile tactile | Largeur 390 px, montants et contrôles disponibles, aucun débordement horizontal, suivi pointeur inactif |
| Axe | Zéro violation détectée sur `main` en clair, sombre et mobile, règles WCAG 2 A/AA et 2.1 AA |
| Console | Aucune erreur JavaScript non interceptée pendant le scénario |
| `git diff --check` | Réussi |

Les tests backend, sécurité, Docker et E2E complets ne sont pas relancés pour cette
modification de présentation. Le nouveau test est intégré à `test:browser`, déjà
appelé par GitHub Actions. Le workflow CI lui-même n'est pas modifié.

## Comparaison visuelle

La version de départ a été exécutée séparément depuis une archive du HEAD, avec
les mêmes dépendances et les mêmes données synthétiques. Aucun worktree ni reset
du checkout de travail n'a été utilisé.

Les captures avant/après sont produites par le test dans
`/tmp/themis-cinematic` lorsque `THEMIS_CINEMATIC_ARTIFACTS` est défini :

- `dashboard-baseline-light.png`, `dashboard-baseline-dark.png` : version initiale ;
- `dashboard-cinematic-light-viewport.png` : vue principale du prototype ;
- `dashboard-cinematic-light.png` : composition complète ;
- `dashboard-cinematic-dark-reduced.png` : sombre avec mouvement réduit ;
- `dashboard-cinematic-mobile-viewport.png`, `dashboard-cinematic-mobile.png` :
  présentation mobile.

La lecture visuelle confirme une transformation concentrée sur l'entrée : fond
spatial, géométrie lumineuse en perspective, synthèse suspendue et KPI séparés.
Les surfaces de lecture restent calmes. Les captures ne démontrent pas la qualité
des transitions : les interactions et leur annulation sont vérifiées dans le test
navigateur. La validation artistique finale appartient à la revue du prototype.

Les captures pleine page prises après scroll peuvent placer sidebar et topbar
fixes au milieu du bitmap ; utiliser les vues viewport pour juger la composition.
Les captures gardent les animations normales : le fast-forward de Playwright
avec `animations: disabled` provoquait un artefact de composition des couches 3D
sur Chromium headless ; il ne correspondait pas au rendu interactif observé.

## Coût et fluidité

Comparaison de builds de production, mêmes dépendances, gzip Python niveau 9
appliqué aux deux sorties (les chiffres gzip par défaut de Vite diffèrent) :

| Sortie | Avant | Prototype | Variation |
| --- | ---: | ---: | ---: |
| JavaScript minifié | 892 106 octets | 901 031 octets | +8 925 octets |
| JavaScript gzip | 238 455 octets | 241 070 octets | +2 615 octets |
| CSS minifié | 184 576 octets | 198 885 octets | +14 309 octets |
| CSS gzip | 32 237 octets | 35 415 octets | +3 178 octets |

Aucune dépendance n'a été ajoutée. Le gros chunk initial demeure ; son découpage
n'est pas entrepris dans ce prototype. Le build ne constitue pas à lui seul un
test de fluidité.

Le test collecte 90 intervalles `requestAnimationFrame` pendant l'ouverture, dans
Chromium headless. Les résultats bruts sont écrits dans `frame-sample.json`.
Le dernier passage mesure une médiane de **16,7 ms**, un p95 de **33,4 ms** et un
maximum de **83,3 ms**. Des pointes sont donc présentes pendant l'ouverture ;
la cadence n'est pas constamment de 60 FPS dans cet environnement.
Ces mesures dépendent de la VM, du mode développement et du navigateur logiciel :
elles ne mesurent ni le temps GPU par frame ni un appareil physique et ne
garantissent pas 60 FPS. L'objectif reste à confirmer en profilant le build de
production sur un ordinateur récent et un téléphone réel.

Les protections vérifiées sont concrètes : entrée inférieure à deux secondes,
particules à durée finie, pas de boucle de rendu React liée au pointeur, nettoyage
des animations, suppression des mouvements sur demande, décor scroll observé
uniquement lorsqu'il est visible, interaction possible sans attendre l'animation.

## Limites connues et suite

- Axe est un audit automatisé ciblé du contenu principal ; il ne certifie pas
  toute l'application et ne remplace pas une revue avec lecteur d'écran.
- Les tests utilisent des données synthétiques ; aucune donnée bancaire réelle
  n'est nécessaire pour valider ces changements visuels.
- L'historique URL/Retour n'est pas géré par l'application initiale ; aucun routeur
  n'a été ajouté. Les comportements de navigation existants sont conservés.
- La 3D est une composition CSS/SVG. Les graphiques de données restent plans pour
  préserver leur lecture. Il n'y a pas de moteur WebGL.
- Les pages Analyse, Transactions, Budgets, Simulation, Crédits et Assistant
  reçoivent uniquement la transition commune pour l'instant. Les extensions
  adaptées à chacune sont détaillées dans `CINEMATIC_DASHBOARD.md`.
