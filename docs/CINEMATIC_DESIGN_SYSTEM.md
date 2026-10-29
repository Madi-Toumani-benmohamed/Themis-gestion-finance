# Direction artistique — Cinematic Financial Intelligence

## Décision

Le Dashboard garde le vocabulaire propre à Thémis — indigo, Inter, Fraunces, chiffres tabulaires et bandeau orbital — et gagne en profondeur par la composition et le mouvement des données. La référence Taste Skill tirée pour une vue financière est **Achromatic Ledger** (seed `741176432`) : elle rappelle que l’accent doit expliquer les données. Son move monochrome complet est écarté car il contredirait la palette existante, les couleurs de revenus/dépenses et l’intention cinématique explicitement demandée. La palette/type du projet sont conservés ; le mouvement éditorial s’applique aux grandes séries observées uniquement.

**Question du Dashboard :** « Quels flux ont été observés pour le mois sélectionné et quel signal mérite mon attention ? »

**Visage de Thémis :** l’orbite cristalline devient l’armature atmosphérique du titre ; le tracé revenus/dépenses reste l’élément coloré majeur et concret. Elle est décorative et n’encode aucune valeur financière.

## Niveaux de panneaux

Ces niveaux s’appliquent au Dashboard, sans introduire un nouveau kit global.

| Niveau | Usage | Traitement |
| --- | --- | --- |
| Hero | Bandeau et conclusion financière de période | Indigo profond, contraste élevé, typographie ample, profondeur localisée, entrée finie |
| Focus | Flux principal et événement prioritaire | Largeur dominante, séries colorées sémantiques, espace de lecture, interaction précise |
| Standard | Catégories, cumul, analyse secondaire | Surface claire/sombre calme, règles fines, données et unités d’abord |
| Quiet | Détails, actions, états vides et widgets secondaires | Chrome réduit, pas de halo ni de relief permanent |

La grille de bureau emploie douze colonnes : le flux principal occupe huit colonnes et le widget d’attention quatre ; les analyses suivantes occupent la largeur selon leur taille utilisateur. Sous le seuil de bureau, le flux et l’attention reprennent toute la largeur. Les contrôles de personnalisation gardent la possibilité de modifier la largeur et l’ordre.

## Hiérarchie de la page

1. **Hero** — nom du produit, titre, période et contrôle de mouvement. Le halo orbital peut répondre légèrement au pointeur sur bureau ; aucune donnée n’est encodée par le décor.
2. **Synthèse de période** — résultat disponible, revenus/dépenses qui le composent, état de complétude et points d’attention. Le résultat garde sa distinction explicite avec le solde bancaire.
3. **Indicateurs non redondants** — variations de revenus et dépenses lorsque les deux mois comparés sont complets, taux d’épargne calculé depuis les valeurs existantes et reste à vivre disponible. Les pourcentages ne sont pas une prédiction ; les comparaisons impossibles restent explicites.
4. **Journal** — événement prioritaire du backend, contexte puis action vers l’écran de détail correspondant. Le ton de l’événement reste celui reçu ; pas de sévérité inventée.
5. **Exploration** — revenus/dépenses en Focus, catégories et cumul en Standard, top dépenses, budgets et abonnements en Quiet. L’utilisateur peut toujours personnaliser la sélection.

Le layout par défaut a un seul Top dépenses. Il ne faut pas corriger les dispositions enregistrées par l’utilisateur en supprimant des widgets de même type : deux instances peuvent avoir des filtres différents.

## Chorégraphie des graphiques

### Première apparition

1. Surface et axes deviennent lisibles immédiatement.
2. Grille et repères apparaissent doucement.
3. Les tracés exacts se dessinent depuis leur géométrie Recharts finale ; revenus, dépenses et net sont légèrement décalés.
4. Les aires s’éclairent après le tracé, sans halo permanent.
5. La légende et les interactions sont utilisables pendant toute la séquence. La légende du flux permet au pointeur ou au clavier d’isoler une série ; les autres séries s’atténuent, elles ne changent pas de valeur.

### Nouvelle période / série

Ne pas interpoler les valeurs entre deux périodes : Recharts ne fournit pas ici de correspondance de points garantissant que les observations, les lacunes calendaires et les unités sont comparables. La révision change avec un fondu bref du nouvel état exact. `withCalendarGaps` représente les périodes manquantes par des lignes `null`, et `connectNulls={false}` garde leurs ruptures sur le flux et le cumul ; l’axe garde les valeurs Recharts. Le cumul souligne seulement sa dernière observation et rappelle que les ruptures correspondent à une absence. Pour les changements rapides, l’animation précédente est annulée et la dernière série gagne. Le brush mémoïsé ne redessine pas pendant le zoom.

### Pointeur, clavier et toucher

- Le guide vertical léger suit l’index actif de Recharts ; le point actif reçoit un halo restreint et un rayon plus lisible.
- Le tooltip affiche période, noms de séries, signes et montants exacts, avec chiffres tabulaires. Il ne remplace pas le tableau accessible des visualisations qui en ont un.
- La légende garde ses signes/couleurs et son accès clavier. Toute sélection de série indique visuellement son état et ne masque pas la valeur exacte par défaut.
- Les changements de données gardent un fondu en opacité ; aucune position « devinée » n’est annoncée par le lecteur d’écran.

### Barres et anneaux

Les barres croissent depuis leur base réelle et les segments de donut apparaissent par léger décalage. Le survol met en évidence uniquement la sélection courante. La profondeur du donut reste désactivable et décorative ; ses données et son total restent à plat et exacts.

## Matière, thème et motion

- Accent indigo conservé pour l’action et la sélection ; revenus, dépenses et net gardent leurs rôles colorés actuels.
- Les surfaces Hero et Focus peuvent recevoir une lumière localisée ; les panneaux Standard et Quiet restent sans glow généralisé.
- Clair et sombre utilisent les variables du design system, avec ajustement des contours/contrastes, pas une inversion mécanique.
- Aucun canvas/WebGL ni boucle 3D n’est justifié pour des séries SVG 2D. Une perspective CSS est limitée à l’illustration du Hero.
- Le mouvement réduit coupe décor, dessin et transitions ; contenu, focus et sélection restent consultables. La révélation de scroll reste à déclenchement une fois.
- Les animations WAAPI sont annulées au démontage ou au changement de préférence. Les animations de layout gardent les mêmes composants React et le brush.

## Sources et limites

Les primitives existantes restent la référence : `styles.css`, `cockpit.css`, `motion/tokens.js`, `motion/preference.js`, `motion/AnimatedChart.jsx`, `components/ChartPrimitives.jsx` et `widgets/cinematicCharts.css`. Les valeurs de comparaison proviennent du modèle existant `dashboardModel.mjs`. Aucun écart n’est calculé dans le style ou dans la couche d’animation.

Les mesures de FPS dans Chromium headless sont des repères de comparaison, pas une garantie de 60 FPS sur appareil physique. Le morphing SVG ne sera réintroduit que si une correspondance stable des observations peut être prouvée sans combler une lacune.
