# Design System Thémis

Consolidation Sprint 2, basée sur les valeurs **effectivement rendues** après le Sprint 1 (`be87da2`). Cette migration conserve l'identité, le DOM, les calculs, les contrats API et les animations. Elle n'est pas une refonte.

## 1. Architecture des styles

Ordre explicite dans `frontend/src/main.jsx` :

1. Polices auto-hébergées.
2. `design-tokens.css` : seule source des tokens visuels globaux, clair puis sombre.
3. `motion/tokens.css` : fichier généré depuis `motion/tokens.js`.
4. `styles.css` : reset, typographie, primitives partagées, styles historiques.
5. `refonte.css` : composition historique (navigation, hero, tables).
6. `cockpit.css` : shell actuel et variantes ciblées `.dashboard`, responsive.
7. `motion/motion.css`, puis `motion/cinematic.css` : comportement animé et adaptations d'accessibilité.

Les pages sont chargées à la demande par `PageLoader.jsx`. Leurs CSS et ceux des composants peuvent donc arriver après le shell. Ils doivent cibler leur composant/page, jamais redéfinir `:root` ou un token global. Ne pas dépendre de l'ordre de visite des pages.

Cette migration ne déplace pas toute la cascade dans des layers : cela changerait la priorité des règles existantes. Les variantes contextuelles restent en place ; les propriétés réellement contradictoires des boutons et les définitions globales doublonnées ont été fusionnées.

## 2. Organisation et nomenclature des tokens

Les noms existants sont conservés pour ne pas casser leurs consommateurs CSS/React/Recharts.

| Famille | Tokens | Usage |
| --- | --- | --- |
| Surfaces | `--page`, `--paper`, `--paper-2`, `--sunken` | Fond, carte, surface secondaire, retrait |
| Texte | `--ink`, `--body`, `--muted`, `--faint` | Hiérarchie, sans inférer une validité métier |
| Bordures | `--line`, `--line-strong` | Séparation et contrôles |
| Accent/information | `--accent`, `--accent-hover`, `--accent-soft`, `--accent-ring` | Sélection, information neutre, focus |
| Action pleine | `--action`, `--action-hover`, `--on-action` | Indigo foncé + blanc dans les deux thèmes |
| États | `--pos`, `--warn`, `--neg`, suffixes `-soft`, `-ink` | Signal, fond, texte contrasté sur fond soft |
| Navigation | `--nav-bg`, `--nav-text`, `--nav-muted`, `--nav-focus` | Sidebar volontairement sombre dans les deux thèmes |
| Données | `--debit`, `--credit`, `--viz-1` à `--viz-6` | Flux signés et palette catégorielle |

L'accent sombre devient clair pour le texte et les graphiques. Il ne doit pas servir automatiquement de fond sous un texte blanc. Les tokens d'action distincts préservent ce contraste. Il n'y a pas de nouveau token `info` : l'accent répond déjà à ce besoin.

Les alias historiques `--green` (accent, malgré son nom), `--red`, `--gold` restent compatibles. Ne pas les utiliser pour de nouveaux composants. Ne pas renommer `--green` ni changer sa valeur sans examiner tous ses consommateurs.

## 3. Thèmes

`data-theme="dark"` sur la racine sélectionne le thème sombre ; l'absence de cet attribut utilise le clair. Le bootstrap HTML et `App.jsx` conservent leur initialisation du fond avant/après React. Les valeurs critiques du premier paint y restent présentes : toute évolution future de `--page` doit vérifier ces deux consommateurs.

Les tokens de rayons, spacing, typographie et z-index ne varient pas selon le thème. Les surfaces, textes, états et ombres varient dans le bloc sombre du fichier de tokens.

Exceptions intentionnelles :

- Sidebar, bouton d'import, boutons primaires et en-tête Assistant : fonds foncés dans les deux thèmes.
- `FinancialSummary.css` : palette locale `--summary-*`, panneau éditorial sombre neutre dans les deux thèmes. Les blancs/lavandes, halos et ornements associés restent locaux ; ils ne codent aucun gain/perte.
- Décors de `cinematic.css`, marque et logo : palette éditoriale locale, à évaluer comme composition, pas à remplacer aveuglément par des surfaces génériques.
- Couleurs de catégories et séries fournies par les données : ne pas remplacer leur identité par l'accent.

## 4. Couleurs financières

`--credit` référence `--pos`, `--debit` référence `--neg`. Ne jamais déduire une anomalie, une dépense ou un revenu uniquement d'une couleur. Préserver les libellés, signes, montants exacts, périodes et légendes.

`--pos-ink`, `--warn-ink`, `--neg-ink` portent les teintes de texte contrastées sur fonds soft. En sombre, ils référencent les teintes sémantiques déjà éclaircies. Les tokens `--pos/neg/warn` ne garantissent pas un contraste de texte normal sur **tous** les fonds.

Les six couleurs `--viz-*` sont une rampe catégorielle existante, pas une échelle universelle de gravité ni une garantie de contraste entre séries adjacentes. Conserver légendes, valeurs exactes et tooltips. Ne pas ajouter de graphiques ni changer la géométrie Recharts dans une correction de style.

## 5. Espacement, rayons, ombres

Spacing existant : `--sp-1/2/3/4/5/6/8/10/12` = `4/8/12/16/20/24/32/40/48px`. Réutiliser ces valeurs lorsque l'intention correspond ; ne pas arrondir les espacements éditoriaux pour forcer leur inclusion dans l'échelle.

Le shell aligne topbar/page/footer sur `--sp-10` à grand écran. Il conserve 28px sous 1100px, puis 14px pour les pages et 12px pour la topbar sous 640px. Les paddings verticaux restent spécifiques à leur composition.

Rayons globaux : `--r-sm` 8px, `--r-md` 12px, `--r-lg` 20px, `--r-xl` 26px, `--r-full` 999px. Ce sont les valeurs finales de l'ancien cockpit. Les rayons 16/18/22px des widgets/synthèses restent des variantes locales : substituer `--r-lg` changerait leur rendu.

Ombres : `--sh-xs/sm/md/lg/xl`, plus `--sh-accent`. Utiliser les petites ombres pour les surfaces, les grandes pour les éléments flottants. Les niveaux sont visuels, sans correspondance automatique avec un z-index. `--ring` est réservé au focus, `--glow-accent` et `--blur-glass` aux usages décoratifs existants.

## 6. Typographie

`--font` : Inter Variable ; `--mono` : IBM Plex Mono ; `--font-editorial` : Fraunces. Les familles restent auto-hébergées.

Base : `--text-base` 15px et `--leading-body` 1.55. Titres globaux : `--text-h1/h2/h3` 34/22/17px, `--weight-heading` 650, `--leading-heading` 1.15. Les titres éditoriaux fluides du Dashboard, Analyse et Budgets sont des variantes locales, pas des tokens globaux concurrents. Ne pas changer un niveau HTML uniquement pour obtenir une taille.

Conserver les chiffres tabulaires des montants et tableaux. Les tailles des badges, légendes et tables restent spécifiques à leur densité d'information.

## 7. Z-index et overlays

Échelle du Sprint 1 conservée :

| Tokens `--z-*` | Valeurs / rôle |
| --- | --- |
| `bg`, `base` | 0, 1 : décor et contexte de base |
| `dropdown`, `sticky` | 30, 40 : menus, éléments sticky |
| `masthead`, `header` | 50, 60 : en-têtes |
| `overlay` | 70 : carte flottante |
| `float`, `float-panel` | 90, 95 : bouton et panneau Assistant |
| `veil`, `modal-backdrop` | 100, 120 : voiles |
| `dialog` | 150 : dialogue personnalisé |
| `toast` | 200 : notifications/palette de commande |
| `max` | 2147483647 : usage historique exceptionnel |

Les nombres s'appliquent **dans un contexte d'empilement**. Un `<dialog>.showModal()` appartient à la top layer native, au-dessus des z-index ordinaires ; `--z-max` ne permet pas de la dépasser. Les `-1` décoratifs dans un conteneur isolé sont locaux. Ne pas ajouter un grand nombre pour résoudre un problème de contexte.

## 8. Conventions CSS et motion

- Les usages historiques `.btn.btn-secondary` conservent leur fond indigo primaire et leurs dimensions secondaires. `.btn-secondary` seul reste neutre ; ne pas supprimer cette compatibilité sans examiner ses consommateurs.
- Une primitive partagée a un bloc propriétaire. Ajouter une variante locale ciblée plutôt qu'une nouvelle surcharge globale en fin de fichier.
- Éviter `!important`. Restent justifiés : priorités reduced-motion/forced-colors, neutralisation des animations pilotées par JS, couleur de légende Recharts définie inline. Documenter la raison avant tout nouvel usage.
- `widgets/cinematicCharts.css` n'a plus besoin de `!important` sur la bordure du tooltip. Le sélecteur historique `.chart-tooltip` n'a actuellement pas de producteur React ; les tooltips actifs sont `.motion-chart-tooltip`. Ne pas réactiver l'ancien effet par renommage opportuniste.
- Pas de `transition: all` : lister les propriétés. Aucun nouveau driver, keyframe ou effet n'est ajouté par ce sprint.
- Les transitions CSS historiques `--t-fast/base/slow` restent 140/220/380ms, avec leurs courbes existantes. Ce ne sont pas des alias de `press/feedback/reveal`.
- `motion/tokens.js` est la source pour Motion et `motion/tokens.css`. `npm run tokens:generate` régénère le CSS ; `npm run tokens:check` et le build détectent une dérive. Durées en secondes côté JS, millisecondes côté CSS. `scene` 360ms est distinct de `page` 320ms.
- Les courbes, ressorts, entrées/sorties des toasts et animations de graphiques du Sprint 1 sont conservés. Pas de CSS `toast-in`, ni d'animation concurrente de Motion.

## 9. Accessibilité

Conserver le bootstrap `data-motion` avant React, `motion/preference.js`, les règles reduced/off et les garde-fous `prefers-reduced-motion`. Les `!important` de ces garde-fous ne sont pas une dette à supprimer arbitrairement.

Focus : `--nav-focus` sur la sidebar sombre, `--accent` dans le contenu, `--ring` sur les champs. Le focus doit rester visible au clavier et ne pas être rogné par un overflow. Ne pas substituer mécaniquement l'accent clair de marque au focus de la sidebar.

Contrôles : tester hover, focus, active, disabled, erreurs, chargement et données absentes. Les libellés restent nécessaires ; couleur seule et tooltips seuls ne suffisent pas. Vérifier au minimum les paires texte/fond à 4.5:1 et les indicateurs de focus à 3:1. Les tests de tokens couvrent des paires explicites, pas tous les composites/transparences possibles.

## 10. Ajouter un composant

1. Examiner les primitives et les variantes existantes, puis leurs consommateurs.
2. Réutiliser les tokens globaux sans les redéfinir. Une variante locale se déclare sur la classe du composant avec un préfixe explicite.
3. Ne créer un token global qu'en présence d'un rôle partagé réel. Conserver une valeur locale si elle répond à une composition spécifique.
4. Importer le CSS près du composant, cibler ses classes ; ne pas toucher au DOM ou à la logique financière pour corriger un détail visuel.
5. Tester clair/sombre, desktop/mobile, clavier, reduced-motion, chargement/erreur/vide et ordre de navigation des pages lazy.
6. Exécuter les contrôles ci-dessous et documenter les exceptions. Ne pas mettre à jour les goldens pour faire disparaître une régression.

```bash
cd frontend
npm run tokens:check
node --test src/components/*.test.mjs tests/design-tokens.test.mjs
npm run build
# Serveur Vite sur 5176 pour les scénarios à API synthétique interceptée :
npm run test:browser
# Go, Poppler et Docker requis ; PostgreSQL réservé à themis_e2e :
npm run test:e2e:local
```

`tests/e2e/design-system.spec.ts` couvre les neuf pages demandées dans les deux thèmes à 1440 et 390px, les tokens après chargement lazy, le focus et l'Assistant. Il accepte `THEMIS_DESIGN_BASELINE_URL` pour enregistrer des captures avant/après sur la même base synthétique. Aucun appel à un service IA n'est requis.

## Relecture de l'audit initial

| Before (constat confirmé) | After | Why |
| --- | --- | --- |
| Tokens globaux contradictoires dans styles/cockpit | Un propriétaire, valeurs finales conservées | Supprimer la dépendance aux redéfinitions |
| Boutons définis à plusieurs endroits | Primitives regroupées + palette d'action | Garder rendu et contraste sombre |
| Variables absentes dans ImportReport | `--line` et `--muted` | Bordure et texte résolus |
| Assistant ouvert en rouge | Palette d'action indigo | L'ouverture ne signale pas un danger |
| Trois `transition: all` | Propriétés explicites, durées inchangées | Éviter d'animer de futures propriétés de layout |
| Tokens Motion CSS recopiés | Génération statique contrôlée au build | Éviter la dérive CSS/JS sans injection runtime |

Les variantes de titres, badges KPI, hauteurs de tables, bulles Assistant et scrollbars ne sont pas fusionnées artificiellement : leurs contextes diffèrent. L'optimisation des barres animées en largeur, du parallax, des nombres et les cinq nouvelles animations restent au sprint Motion/performance. Verdict motion limité au diff : aucun nouvel effet ; les risques préexistants nécessitent toujours un profil sur appareil mobile réel.
