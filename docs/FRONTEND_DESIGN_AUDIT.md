# Frontend Design Audit — Thémis

**Date :** 2026-11-07  
**Scope :** Frontend React (Vite + Recharts + Motion)  
**Méthode :** Audit UX/UI (design tokens, ergonomie), animations existantes (performance, accessibilité, cohérence), opportunités d'animation (Gate à 4 questions), performance (layout thrash, main thread, reduced-motion).

---

## Résumé exécutif

| Catégorie | P0 | P1 | P2 | Total |
|-----------|----|----|----|-------|
| Design tokens / UI | 2 | 7 | 6 | 15 |
| Animations existantes | 3 | 5 | 3 | 11 |
| Accessibilité / reduced-motion | 2 | 1 | 0 | 3 |
| **Total** | **7** | **13** | **9** | **29** |

**Constat global :** Thémis a une base solide (design system cohérent dans `styles.css`, motion grammar dans `motion/motion.css`), mais souffre de **duplications non consolidées** (styles.css vs cockpit.css vs refonte.css vs cinematic.css), de **conflits d'animation** (Toast double système), et d'une **stratégie reduced-motion fragmentée** en 2-3 couches contradictoires.

---

## 1. Incohérences de design system (UX/UI)

### 1.1 Typographie — tailles divergentes

| # | Before | After | Why | Fichier |
|---|--------|-------|-----|---------|
| D-01 | `styles.css:83` h1 = 34px | `cockpit.css:101` h1 = `clamp(30px, 3.3vw, 46px)` | Deux échelles pour le même niveau hiérarchique. | cockpit.css:101 |
| D-02 | `styles.css:84` h2 = 22px | `Budgets.css:11` h2 = `clamp(26px, 2.5vw, 36px)` | Page Budgets utilise une échelle plus grande sans justification. | Budgets.css:11 |
| D-03 | `styles.css:86` h3 = 17px | `Analyse.css:115` h2 = 20px (sans h3 défini) | Titres de section Analyse utilisent h2 au lieu de h3. | Analyse.css:115 |
| D-04 | `--font-editorial` défini | Utilisé seulement dans cockpit.css et refonte.css | Police éditoriale sous-exploitée, incohérente. | styles.css:56 |

### 1.2 Espacements & Padding

| # | Before | After | Why | Fichier |
|---|--------|-------|-----|---------|
| D-05 | `styles.css:244` `.page` padding 36px 28px 80px | `cockpit.css:45` `.app-main .page` padding 38px 40px 64px | Différence 2-12px sans raison apparente. | cockpit.css:45 |
| D-06 | `styles.css:285` masthead 14px 28px | `cockpit.css:82` topbar 15px 40px | Incohérence header global vs cockpit. | cockpit.css:82 |

### 1.3 Rayons (border-radius)

| # | Before | After | Why | Fichier |
|---|--------|-------|-----|---------|
| D-07 | `styles.css:45` `--r-lg: 16px` | `cockpit.css:13` `--r-lg: 20px` | Cockpit redéfinit le token global. | cockpit.css:13 |
| D-08 | `styles.css:103` `var(--r-sm)` = 8px | `refonte.css:75` `var(--r-xl)` = 22px | Hero utilise un rayon différent sans token dédié. | refonte.css:75 |

### 1.4 Ombres & Depth — deux systèmes parallèles

| # | styles.css (global) | cockpit.css (cockpit) | Risque |
|---|----------------------|------------------------|--------|
| D-09 | `--sh-md: 0 4px 12px rgba(22,22,29,.07)` | `--sh-md: 0 12px 28px -18px rgba(38,34,80,.25)` | Couleur, flou, offset complètement différents. Ombres cockpit plus "lourdes". |
| D-10 | `--sh-lg: 0 12px 32px rgba(22,22,29,.10)` | `--sh-lg: 0 24px 60px rgba(0,0,0,.4)` | Cockpit utilise rgba(0,0,0) au lieu du token indigo. |

### 1.5 Couleurs hardcodées vs tokens

| # | Fichier:ligne | Couleur hardcodée | Token attendu |
|---|---------------|-------------------|---------------|
| D-11 | `refonte.css:88` | `#B7791F`, `#D69E2E` | `--warn`, `--warn-soft` gradient |
| D-12 | `refonte.css:110` | `#B54545`, `#D66A6A` | `--neg`, `--neg-soft` gradient |
| D-13 | `AssistantFloating.css:33` | `var(--neg)` pour bouton "ouvert" | `--accent` (fonction assistant ≠ danger) |
| D-14 | `widgets/cinematicCharts.css:7` | `var(--accent) !important` | `!important` non documenté |
| D-15 | `components/FinancialSummary.css:10` | `#25243F`, `#34325B`, `#D8D8EE` | Ne respecte pas dark mode |

### 1.6 Z-index — pas de hiérarchie centralisée

| Élément | z-index | Fichier | Problème |
|---------|---------|---------|----------|
| `.drop-veil` | 100 | styles.css:254 | Base overlay |
| `.stamp-overlay` | 120 | styles.css:263 | Au-dessus drop-veil |
| `.cmdk-overlay` | 200 | styles.css:830 | Command palette |
| `.scroll-top` | 90 | styles.css:960 | **Sous** les overlays |
| `.assistant-float-btn` | 90 | AssistantFloating.css:10 | Même z-index que scroll-top |
| `.assistant-panel` | 95 | AssistantFloating.css:51 | Peut être masqué par overlays 100+ |
| `topbar` | aucun | cockpit.css | Risque superposition |

### 1.7 Tokens de durée dupliqués

| Token | styles.css | motion/tokens.js | Delta |
|-------|------------|------------------|-------|
| Fast/press | `--t-fast: 140ms` | `press: 0.12s` = 120ms | 20ms |
| Base/feedback | `--t-base: 220ms` | `feedback: 0.18s` = 180ms | 40ms |
| Slow/reveal | `--t-slow: 380ms` | `reveal: 0.52s` = 520ms | 140ms |

---

## 2. Défauts d'ergonomie

| # | Problème | Fichier:ligne | Impact |
|---|----------|---------------|--------|
| E-01 | Focus outline hardcodé `#C1C1FF` dans cockpit.css | cockpit.css:64 | Ne suit pas thème dark |
| E-02 | `:has(input:checked)` utilise `box-shadow` au lieu d'`outline` | motion.css:83 | Accessibilité lecteurs d'écran |
| E-03 | ImportReport.css utilise `--border`, `--text-muted` non définis | ImportReport.css:1 | CSS cassé potentiel |
| E-04 | Transaction table row height incohérent (62px vs 54px) | Transactions.css:35 vs refonte.css:231 | Saut visuel entre pages |
| E-05 | KpiRow `.kpi-period-badge` display diffère | styles.css:331 vs cockpit.css:113 | `inline-flex` vs `block` |
| E-06 | Dialog backdrops notation `#0009` peu standard | Statements.css:221 | Maintenance |
| E-07 | Double implémentation bulles Assistant | Assistant.css vs AssistantFloating.css | Divergence styles messages |

---

## 3. Animations existantes — audit détaillé

### 3.1 P0 — Bloquants

| # | Élément | Problème | Localisation | Fix |
|---|---------|----------|--------------|-----|
| A-01 | **Toast double animation** | CSS keyframe `toast-in` + Motion `AnimatePresence` simultanés. Entrée double, exit bloqué. | styles.css:646, 672, 1237, 1241 ; Toast.jsx:62 | Garder UN seul système (recommandé: Motion pour interruptibilité). |
| A-02 | **Deux `@keyframes toast-in`** | Même nom, comportement différent (`translateX` vs `translateY`). La seconde écrase la première. | styles.css:672, 1237 | Supprimer le duplicata (l.1237-1241). |
| A-03 | **Reduced-motion : 2 stratégies contradictoires** | `styles.css:703` (globale `*`) + `cinematic.css:141` (`[data-motion]`). Cascade imprévisible, éléments potentiellement non couverts (Recharts WAAPI). | styles.css:279,703,1251 ; cinematic.css:141-147,176 ; motion.css:124-132 | Uniformiser sur `[data-motion]` géré par `preference.js`. Supprimer règles `*, *` de styles.css. |

### 3.2 P1 — Performance / cohérence

| # | Élément | Problème | Localisation | Fix |
|---|---------|----------|--------------|-----|
| A-04 | `transition: all` ×3 | Déclenche recalc style complet sur hover/focus. | styles.css:679 (`.seg button`), 2046 (`.merchant-tab`), TagPickerNavigation.css:160 | Remplacer par propriétés explicites (`background, color, border-color`). |
| A-05 | **Width transitions** (layout thrash) | 6 occurrences de `transition: width` sur barres de progression. Layout recalc à chaque frame. | styles.css:378, 394, 759, 1647, 1661 ; motion.css:111 | Remplacer par `scaleX()` + `transform-origin: left` pour barres fréquentes. Garder `width` pour upload/gauges (ponctuels). |
| A-06 | **AnimatedNumber per-character spans** | 10+ `m.span` Motion par montant. Re-render complet à chaque changement de valeur. | motion/AnimatedNumber.jsx:20-23 | Regrouper par groupes (milliers, décimales) ou utiliser `clip-path` horizontal. |
| A-07 | **Cinematic parallax** `getBoundingClientRect()` dans rAF | Peut forcer layout sync pendant paint. | motion/CinematicDashboard.jsx:55 | Déplacer mesure dans `useLayoutEffect` sur resize, ne garder que calculs relatifs en rAF. |
| A-08 | **Tooltip 3D perspective** | `perspective(800px) rotateX(-4deg)` sur tooltip Recharts fréquent. Composite layer coût. | widgets/cinematicCharts.css:9-10 | Remplacer par `opacity` + `translateY` simple. |

### 3.3 P2 — Mineurs

| # | Élément | Observation | Localisation |
|---|---------|-------------|--------------|
| A-09 | Sidebar nav icon `rotate(-5deg)` hover | Fréquence élevée mais `transform` seul (GPU), 240ms, impact négligeable. | motion/motion.css:30-31 |
| A-10 | View Transitions Firefox | Déjà géré par fallback dans `motion/index.jsx:33`. | motion/index.jsx:33 |
| A-11 | Deux skeleton shimmers | `motion-loading` (2 itérations, finit) vs `shimmer` (infini). | motion/motion.css:78 ; styles.css:1216 |

---

## 4. Accessibilité & reduced-motion

| # | Problème | Localisation | Sévérité |
|---|----------|--------------|----------|
| ACC-01 | Stratégie reduced-motion fragmentée (voir A-03) | styles.css, cinematic.css, motion.css | **P0** |
| ACC-02 | Recharts animations non explicitement arrêtées par CSS media query | widgets/cinematicCharts.css | **P1** (dépend du flag JS) |
| ACC-03 | Focus outline cockpit hardcodé `#C1C1FF` | cockpit.css:64 | **P1** |
| ACC-04 | Toast timer `toast-bar` continue pendant reduced-motion ? | styles.css:664-665 | À vérifier |
| ACC-05 | Command Palette — animation sur ⌘K (clavier) | CommandPalette.jsx | **P0** si animé (voir §5 rejet) |

---

## 5. Opportunités d'animation — Gate validé

Seules **5 suggestions** passent les 4 questions du Gate (Frequency, Purpose, Speed, Function).

| # | Location | Today | Purpose | Frequency | Suggested motion |
|---|----------|-------|---------|-----------|------------------|
| O-01 | `pages/Accounts.jsx:76` `.account-card` grid | Apparition instantanée sans lien spatial | Preventing jarring change / Group entrance | Rare (page Comptes qq fois/sem) | `Reveal` par carte stagger : `initial {opacity:0, y:18, scale:0.98}` → `animate {opacity:1, y:0, scale:1}` `transition {duration: 0.52s, ease: [0.22,1,0.36,1], delay: index*0.055s max 0.275s}` |
| O-02 | `components/TransactionDetails.jsx:22` TransactionDrawer `<dialog>` | Ouvre/ferme sans transition, contenu téléporte | Spatial consistency / State indication | Occasional (< dizaines/jour) | Entrée depuis droite : `initial {x:'100%', opacity:0}` → `animate {x:0, opacity:1}` `transition {duration: 0.28s, ease: [0.22,1,0.36,1]}` `exit {x:'100%', opacity:0}` 0.22s. `transform-origin: right center`. Reduced → opacity 150ms seul. |
| O-03 | `App.jsx:39-58` ScrollTop button | Apparaît/disparaît via `if(!show) return null` — brutal | Preventing jarring change | Occasional (pages longues) | Fade+scale : `initial {opacity:0, scale:0.9}` → `animate {opacity:1, scale:1}` `transition {duration: 0.18s, ease: [0.22,1,0.36,1]}` Exit 0.16s. |
| O-04 | `pages/Dashboard.jsx:149-154` KpiRow variations | Chaînes `+5 %` changent instantanément au changement de mois | State indication | Occasional (navigation mois) | `AnimatedNumber` sur la partie numérique seule : `direction` de 85% selon signe, `transition {duration: 0.44s, ease: [0.22,1,0.36,1], delay: index*0.016}` Texte `%` fixe. |
| O-05 | `components/ImportPreview.jsx:27` ImportPreview dialog | Apparaît instantanément via `showModal()` | Spatial consistency | Rare (import qq fois/sem) | Entrée centrée : `initial {opacity:0, scale:0.96, y:12}` → `animate {opacity:1, scale:1, y:0}` `transition {duration: 0.32s, ease: [0.22,1,0.36,1]}` Exit 0.22s. |

### Rejected candidates (explicite)

| Location | Gate question échouée | Raison |
|----------|----------------------|--------|
| `CommandPalette.jsx:97-121` résultats | **Frequency** (100+/day, clavier-initié) | **Jamais animer une action clavier** — latence perçue. Raycast n'a pas d'animation. |
| `pages/Transactions.jsx:521-527` sélection/hover lignes | **Frequency** (tens/day, haute fréquence) | Interaction fonctionnelle — motion gêne la lecture. |
| `pages/Simulate.jsx:341-350` curseurs réduction | **Frequency** (tens/day, ajustement fréquent) | Feedback natif `<input type=range>` suffit. |
| `pages/Statements.jsx:614` `AnimatedRegion` tableau | **Déjà couvert** | Anime déjà opacity/translate 280ms via WAAPI. |
| `Toast.jsx:62` toasts | **Déjà animé** | `PresencePanel` 0.22s ease, layout position. |

**Verdict :** L'interface est déjà sobre. Seules les apparitions rares/occasionnelles méritent un pont spatial. La plus haute levier = TransactionDrawer (consulté régulièrement, moment "focus" où un slide cohérent améliore la lisibilité sans pénaliser la vitesse).

---

## 6. Problèmes de performance détaillés

### 6.1 Layout-triggering properties animées

| Propriété | Occurrences | Fichiers | Risque |
|-----------|-------------|----------|--------|
| `width` | 6 | styles.css:378,394,759,1647,1661 ; motion.css:111 | **HIGH** — layout recalc/frame |
| `height` | 1 | TagPickerNavigation.css:160 (via `transition: all`) | **MEDIUM** |
| `padding` | 0 direct | — | — |
| `margin` | 0 direct | — | — |

### 6.2 Paint-triggering properties

| Propriété | Occurrences | Fichiers | Note |
|-----------|-------------|----------|------|
| `box-shadow` | 10+ | styles.css (hover widgets), cockpit.css | Négligeable si isolé |
| `border-radius` | 0 animé | — | — |
| `filter: blur()` | 0 | — | Pas de blur animé (bon) |

### 6.3 Main thread pressure

| Source | Fréquence | Driver | Mitigation |
|--------|-----------|--------|------------|
| Cinematic parallax `pointermove` + rAF | Continue sur Dashboard | rAF + CSS custom props | Déplacer `getBoundingClientRect` hors rAF (voir A-07) |
| AnimatedNumber re-renders | À chaque changement montant | React + Motion spans | Regrouper spans (voir A-06) |
| Recharts tooltips 3D | Hover fréquent graphiques | CSS animation | Simplifier (voir A-08) |
| View Transitions | Navigation page | Browser native | Fallback OK (Firefox) |

### 6.4 will-change — usage actuel

Aucun `will-change` explicite trouvé. Les éléments animant `transform`/`opacity` sont auto-promus par le navigateur. Ajouter `will-change: transform` uniquement si shift 1px observé au démarrage/fin (cas `.scene-orbit-system` possible candidat).

---

## 7. Plan d'action priorisé

### P0 — À corriger avant release (3 items)

| # | Action | Fichiers | Effort |
|---|--------|----------|--------|
| P0-1 | **Unifier Toast** : supprimer CSS keyframes `.toast { animation }`, garder Motion `AnimatePresence` | styles.css:646,672,1237-1241 ; Toast.jsx | 30 min |
| P0-2 | **Unifier reduced-motion** : supprimer `styles.css:703` et `styles.css:1251` `*, *` rules ; ne garder que `[data-motion]` de `cinematic.css` + sync `preference.js` | styles.css:279,703,1251 ; cinematic.css:141-147 ; motion/preference.js | 1h |
| P0-3 | **Supprimer `@keyframes toast-in` duplicata** (l.1237-1241) | styles.css | 2 min |

### P1 — Sprint suivant (8 items)

| # | Action | Fichiers | Effort |
|---|--------|----------|--------|
| P1-1 | Remplacer 3 `transition: all` par propriétés explicites | styles.css:679,2046 ; TagPickerNavigation.css:160 | 15 min |
| P1-2 | Convertir `width` → `scaleX` sur barres fréquentes (merchant-bar, nature-seg) | styles.css:1647,1661 | 30 min |
| P1-3 | Optimiser AnimatedNumber : grouper par milliers/décimales | motion/AnimatedNumber.jsx | 45 min |
| P1-4 | Cinematic parallax : sortir `getBoundingClientRect` du rAF | motion/CinematicDashboard.jsx | 30 min |
| P1-5 | Simplifier tooltip Recharts 3D → opacity+translateY | widgets/cinematicCharts.css:9-10 | 10 min |
| P1-6 | Focus outline cockpit : utiliser `var(--accent)` au lieu de hardcodé | cockpit.css:64 | 5 min |
| P1-7 | Z-index : définir système centralisé (CSS custom props `--z-*`) | styles.css, cockpit.css, AssistantFloating.css | 1h |
| P1-8 | Tokens durée : aligner `styles.css` sur `motion/tokens.js` (ou inversement) | styles.css:63-67 ; motion/tokens.js | 15 min |

### P2 — Backlog (9 items)

| # | Action | Fichiers |
|---|--------|----------|
| P2-1 | Consolider rayons : `--r-lg` unique (16px ou 20px) | styles.css, cockpit.css, refonte.css |
| P2-2 | Consolider ombres : un seul système (styles.css ou cockpit.css) | styles.css, cockpit.css |
| P2-3 | Remplacer couleurs hardcodées par tokens (15 occurrences) | refonte.css, AssistantFloating.css, FinancialSummary.css, widgets/cinematicCharts.css |
| P2-4 | Unifier skeleton shimmer (motion-loading vs shimmer) | motion/motion.css, styles.css |
| P2-5 | KpiRow period badge : display cohérent | styles.css:331, cockpit.css:113 |
| P2-6 | Transaction row height : valeur unique | Transactions.css, refonte.css |
| P2-7 | Assistant bulles : fusionner Assistant.css + AssistantFloating.css | components/Assistant*.css |
| P2-8 | Scrollbar épaisseur : unifier (10px vs 6px) | styles.css, TagPickerNavigation.css |
| P2-9 | Implémenter 5 opportunités d'animation validées (O-01 à O-05) | Accounts.jsx, TransactionDetails.jsx, App.jsx, Dashboard.jsx, ImportPreview.jsx |

---

## 8. Risques & Limitations

| Risque | Description | Mitigation |
|--------|-------------|------------|
| **Toast double animation** | Bug visuel probable : toast qui slide horizontal puis vertical, ou ne se ferme pas. | Corriger P0-1 en priorité absolue. |
| **Reduced-motion fragmenté** | Animations peuvent "fuiter" pour utilisateurs `prefers-reduced-motion` si `preference.js` n'est pas sync au premier paint. | Corriger P0-2 ; tester avec `prefers-reduced-motion: reduce` dans DevTools. |
| **Z-index collisions** | Overlays (command palette, assistant, import preview, toasts) peuvent se superposer imprévisiblement. | Implémenter P1-7 (système `--z-*`). |
| **Dark mode cockpit divergent** | `cockpit.css` redéfinit `--accent` `#A2A2FF` vs `#7C7CF0` global. Incohérence visuelle. | Aligner sur tokens globaux ou documenter l'écart intentionnel. |
| **AnimatedNumber sur mobile low-end** | 30+ spans Motion simultanés (FinancialSummary ×3) peuvent drop frames. | Implémenter P1-3 (regroupement) avant charge réelle. |
| **View Transitions Firefox** | Non supporté stable — fallback immédiat OK mais pas d'animation fluide. | Accepter (P2 info), surveiller caniuse. |

---

## 9. Fichiers concernés — index

| Fichier | Lignes clés | Catégories touchées |
|---------|-------------|---------------------|
| `styles.css` | 1-300, 646-680, 703, 1216-1251, 1647, 1661 | Design tokens, Toast, reduced-motion, width transitions |
| `cockpit.css` | 1-235 | Design tokens (shadows, radius, colors), focus outline, z-index |
| `motion/motion.css` | 1-148 | Motion grammar, sidebar nav, page transitions, width upload |
| `motion/cinematic.css` | 1-180 | Cinematic dashboard, reduced-motion strategy B, parallax |
| `motion/tokens.js` | 1-4 | Timing tokens, ease, spring |
| `motion/CinematicDashboard.jsx` | 48-75 | Parallax rAF, getBoundingClientRect |
| `motion/AnimatedNumber.jsx` | 20-23 | Per-char spans |
| `widgets/cinematicCharts.css` | 9-10 | Tooltip 3D animation |
| `Toast.jsx` | 62 | Double animation system |
| `components/TransactionDetails.jsx` | 22 | Drawer sans transition |
| `pages/Accounts.jsx` | 76 | Account cards group entrance |
| `pages/Dashboard.jsx` | 149-154 | KpiRow variations |
| `components/ImportPreview.jsx` | 27 | Dialog sans transition |
| `App.jsx` | 39-58 | ScrollTop brutal appear |
| `CommandPalette.jsx` | 97-121 | Rejeté (clavier) |
| `pages/Transactions.css` | 35, 48, 521 | Row height, hover, sélection rejetée |
| `pages/Simulate.jsx` | 341-350 | Sliders rejetés |

---

## 10. Validation & Next Steps

**Tests à exécuter après corrections P0 :**

```bash
# Backend
cd backend && go test ./... && go vet ./...

# Frontend
cd frontend && npm run build

# Reduced-motion test
# 1. Ouvrir DevTools > Rendering > "Emulate CSS prefers-reduced-motion: reduce"
# 2. Naviguer Dashboard → Transactions → ouvrir TransactionDrawer → importer → toasts
# 3. Vérifier : AUCUNE animation transform/opacity > 0ms, seulement color/opacity rapides

# Performance spot-check
# 1. Lighthouse (mobile throttling) sur Dashboard
# 2. DevTools Performance : enregistrer navigation + hover graphiques
# 3. Vérifier : 0 layout shifts, 0 long tasks > 50ms pendant animations
```

**Branche de travail recommandée :** `feat/frontend-design-audit-fixes`  
**Ordre d'exécution :** P0-1 → P0-2 → P0-3 → P1-1..P1-8 → P2-1..P2-9 (opportunités O-01..O-05 en dernier)

---

*Rapport généré par audit automatisé + revue manuelle. Aucun fichier applicatif modifié.*