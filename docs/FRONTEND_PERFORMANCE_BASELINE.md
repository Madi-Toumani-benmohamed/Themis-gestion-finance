# Baseline de Performance Frontend — Thémis

**Date :** 2026-11-08  
**Commit :** à définir après tests  
**Branch :** `perf/frontend-motion-optimization`  
**Environnement :** Production build, Docker, Vite 8.3.4, React 18.3.1

---

## 1. Résumé exécutif

| Métrique | Valeur | Seuil | Statut |
|----------|--------|-------|--------|
| Taille bundle principal | 274.61 kB (gz: 89.53 kB) | - | ✅ |
| ChartPrimitives | 408.66 kB (gz: 106.71 kB) | - | ✅ |
| Temps chargement initial | - | < 2s | À mesurer |
| FPS Dashboard (idle) | - | ≥ 60 | À mesurer |
| Long Tasks > 50ms | - | 0 | À mesurer |
| CLS | - | < 0.1 | À mesurer |

---

## 2. Méthodologie

### Outils utilisés

- **Chrome DevTools Performance** : enregistrement manuel
- **React Profiler** : identité des re-renders
- **Lighthouse** : métriques Core Web Vitals
- **Playwright Tests** : `tests/motion.browser.cjs`, `tests/e2e/*.spec.ts`

### Conditions de test

- Build : `npm run build` (production)
- Docker : `docker compose build --no-cache app`
- Navigateur : Chromium 130+ (via Playwright)
- Viewport : 1440x1000 (desktop), 390x844 (mobile)
- Données : fixtures synthétiques
- Cache : décoché
- Réseau : throttling "Regular 3G" pour LCP

### Scénarios testés

#### Dashboard
- [ ] Chargement initial (LCP)
- [ ] Changement de mois
- [ ] Hover graphiques Recharts
- [ ] Scroll Parallax CinematicDashboard
- [ ] Animation des KPI

#### Analyse
- [ ] Chargement visualisations
- [ ] Filtrage catégories/tags
- [ ] Tooltips Recharts

#### Transactions
- [ ] Chargement tableau
- [ ] Recherche
- [ ] TransactionDrawer ouvert/fermé

#### Simulation
- [ ] Modification sliders
- [ ] Recalcul résultats

---

## 3. Analyse du bundle

### Taille des chunks principaux (production)

```
ChartPrimitives-B9L4JjRi.js  408.66 kB    106.71 kB   # Graphiques Recharts
index-B7LdWkw2.js              80.17 kB     27.39 kB   # App root
motion-CKLdf5zG.js            274.61 kB     89.53 kB   # Animation Motion
Dashboard-CIBb-hT2.js         79.87 kB     22.57 kB   # Page Dashboard
Transactions-CfFnJIHp.js      57.79 kB     16.55 kB   # Page Transactions
Simulate-KcpofE4I.js          17.91 kB      5.85 kB   # Page Simulate
```

### Observations

- `ChartPrimitives` est le chunk le plus lourd (408 kB raw, 106 kB gz)
- `motion` bundle contient AnimatedNumber, CinematicDashboard, MotionProvider
- Code splitting par page fonctionne correctement

---

## 4. Analyse Animation Motion

### AnimatedNumber (motion/AnimatedNumber.jsx)

- Crée **un span par caractère** du nombre formaté
- Chaque span a une animation Motion séparée
- Pour un montant de 882,29 (5 chars) = 5 composants Motion
- FinancialSummary peut avoir 3+ nombres animés = 15+ spans

**Impact potentiel** : Re-renders multiples, surcharge DOM

### CinematicDashboard (motion/CinematicDashboard.jsx)

- `getBoundingClientRect()` appelé dans `pointermove` → `rAF`
- Animation parallax continue pendant le hover
- SVG paths animés avec Web Animations API

**Impact potentiel** : Layout sync si `getBoundingClientRect()` fréquent

### Transitions CSS (`styles.css`)

- `transition: width` sur 6 barres de progression (A-05)
- `box-shadow` animé sur widgets (GPU-friendly si isolé)

---

## 5. Points d'attention identifiés

### Haute priorité

1. **AnimatedNumber spans** : potentiellement 30+ spans simultanés
2. **Parallax CinematicDashboard** : `getBoundingClientRect` dans rAF
3. **Width transitions** : layout recalc si fréquentes

### Moyenne priorité

4. **Barres de progression** : `width` au lieu de `scaleX`
5. **Tooltip 3D Recharts** : perspective + rotateX
6. **Widget hover** : box-shadow animé

---

## 6. Actions de baseline

- [ ] Snapshot build bundle sizes
- [ ] Run motion.browser.cjs tests
- [ ] Run design-system.spec.ts tests  
- [ ] Chrome DevTools Performance Dashboard
- [ ] Lighthouse mobile/desktop
- [ ] Documenter résultats

---

## 7. Références

- Audit initial : `docs/FRONTEND_DESIGN_AUDIT.md`
- Design System : `docs/DESIGN_SYSTEM.md`
- Tests Motion : `frontend/tests/motion.browser.cjs`