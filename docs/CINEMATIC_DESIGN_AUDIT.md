# Audit — expérience cinématique du Dashboard

## Périmètre et question de l’écran

Le Dashboard répond d’abord à : **quels flux ont été observés sur le mois choisi, et que faut-il regarder ensuite ?** Le contexte principal est un usage personnel, au quotidien, sur les thèmes clair et sombre, avec des données mensuelles pouvant être absentes ou partielles. Le sélecteur de mois, l’import de relevés, la personnalisation du tableau et les actions vers les transactions et les relevés restent des fonctions de premier plan.

## Inventaire observé

- Les objets visibles sont les flux mensuels revenus/dépenses/net, le reste à vivre disponible, les catégories, transactions, budgets, abonnements, signaux et événements du journal.
- Les montants sont exprimés en euros ; les graphiques sont chronologiques au mois. L’API peut distinguer mois courant, incomplet et complet.
- Le titre, le mois consulté et l’action d’import sont dans la zone supérieure de l’application.
- `FinancialSummary` présente déjà le résultat de flux, les revenus et dépenses, les points d’attention et la couverture documentaire.
- `KpiRow` reprend actuellement revenus, dépenses et résultat déjà présents dans la synthèse ; seuls le reste à vivre et les variations ajoutent une information distincte.
- `FluxWidget` compose trois séries sémantiquement distinctes et conserve un brush Recharts mémoïsé. `SoldeWidget` est un cumul des flux du jeu de données, et non un solde bancaire.
- `CategoriesWidget` possède une légende interactive et un mode de profondeur décoratif ; `TopWidget` exclut les transferts.
- `Journal` trie les événements par importance fournie par le backend, distingue les tons et expose les actions de navigation.
- La disposition par défaut contient une occurrence du widget « Top dépenses du mois ». La disposition personnalisable accepte plusieurs occurrences légitimes ; aucune suppression automatique d’une préférence utilisateur n’est justifiée.
- Les widgets peuvent être retirés, déplacés, redimensionnés, configurés et ajoutés. Ces capacités doivent rester disponibles.
- Le shell dispose déjà des thèmes, de l’import global, de la navigation persistante, de `MotionControl` et de la préférence mouvement réduit.

## Motion déjà présent

- `MotionSurface` / `Reveal` révèlent une fois les sections lorsqu’elles entrent dans la fenêtre.
- `AnimatedChart` utilise WAAPI pour dessiner les courbes SVG à partir de leur géométrie finale, révéler les aires et faire croître barres et secteurs ; le zoom du brush ne doit pas relancer les tracés.
- `ChartFocusDot` et `ChartTooltip` donnent un repère au survol. Recharts porte l’interaction de pointage et de sélection.
- Les montants sont rendus par `AnimatedNumber`, avec une valeur exacte accessible. Le Hero possède des orbites SVG à entrée finie ; la navigation utilise View Transitions avec repli.
- La réduction du mouvement est suivie dans `preference.js` et les composants partagés ; les états lisibles ne dépendent pas des animations.

## Limites identifiées

1. Le dessin initial des courbes, des aires et des autres formes est lancé dans une même passe, avec des retards par élément. Les axes et repères n’ont pas de chorégraphie dédiée.
2. Quand `revision` change, l’effet recrée son `WeakSet` et peut redessiner des formes SVG déjà présentes. Ce traitement ne distingue pas un premier dessin d’une nouvelle période ou d’une série mise à jour.
3. Une interpolation arbitraire entre les valeurs de deux périodes pourrait suggérer des points mensuels observés qui n’existent pas. L’API n’offre pas de géométrie de morphing fiable à réutiliser sans risque.
4. Les interactions dépendent surtout des comportements par défaut de Recharts ; le guide vertical, le focus des séries et la mise en évidence de la série active sont peu affirmés.
5. Le résumé de période et la rangée de KPI font répéter trois grands montants. Le flux conserve des couleurs sémantiques, mais les trois séries visuellement proches ne sont pas sélectionnables depuis la légende.
6. La grille des widgets est principalement pilotée par deux tailles (`half`, `full`) et les cartes partagent le même traitement. Les grands écrans n’expriment pas encore de niveau Focus/Standard/Quiet.
7. `AttentionWidget` existe dans la disposition par défaut, mais les points forts du journal et les insights de widgets ont une hiérarchie plus faible que les grands panneaux analytiques.
8. Le hero existe déjà (bandeau indigo, orbites, cristallographie abstraite, sélecteur de mois) ; il a une identité mais son décor ne répond pas au pointeur. Les animations d’entrée sont finies, pas en boucle.
9. Le flux principal ne crée pas de point de calendrier pour un mois absent. Recharts peut donc relier les observations de part et d’autre d’une absence pourtant connue.

## Résultat du prototype intégré

- Les grands montants des KPI ne répètent plus la synthèse : les variations comparables, le taux d’épargne déjà modélisé et le reste à vivre disponible donnent les indicateurs secondaires.
- Le flux principal reçoit une largeur dominante sur desktop ; l’attention occupe la colonne attenante. Les surfaces Focus, Standard, Quiet et éditoriales sont ciblées au Dashboard.
- La première entrée distingue grille/axes, tracés décalés, aires et légende. Un changement de révision ne rejoue plus le dessin initial ; il montre brièvement le nouvel état exact.
- Les séries revenus, dépenses et net ont un guide vertical, une légende clavier pour les isoler et un tableau accessible avec les mois sans observation explicitement indiqués.
- Les dates absentes entre deux lignes existantes sont représentées par des points `null`, sans interpolation ni liaison de courbe.
- Le cumul net respecte les mêmes ruptures de calendrier et souligne sa dernière valeur observée.
- L’orbite du Hero répond au pointeur souris en desktop par un déplacement faible ; elle reste fixe au tactile et en mouvement réduit.
- Le breakpoint mobile contraint le décor orbital de la synthèse pour éviter qu’il ne crée de défilement horizontal.

## Contraintes et surfaces touchées

- Pas de changement d’API, backend, modèles, migrations, calcul de séries ou convention financière.
- Les widgets gardent leur fonctionnement et l’édition de disposition. Aucun doublon d’une configuration utilisateur n’est supprimé automatiquement.
- Les fichiers partagés touchés sont `Dashboard.jsx`, `widgets/index.jsx` et `AnimatedChart.jsx`. Les modifications de style restent sous le scope `.cinematic-dashboard` ou `.dashboard`.
- Les validations navigateur existantes fournissent des fixtures synthétiques et couvrent notamment période, brush, thèmes, mouvement réduit, exactitude des valeurs, navigation et accessibilité Axe.

## Vérification de référence

La disposition par défaut est définie dans `frontend/src/widgets/index.jsx`. L’animation des géométries et le focus sont dans `frontend/src/motion/AnimatedChart.jsx`. Les hooks d’entrée et de révision sont documentés dans `docs/MOTION_DESIGN.md` et `CODEBASE_GRAPH.md`. Les calculs des KPI sont dans `frontend/src/components/dashboardModel.mjs` ; ils comparent seulement deux périodes consécutives complètes et gardent absentes les valeurs indisponibles.
