# Performance Frontend — Thémis (Sprint 3)

**Date :** 2026-11-08  
**Branch :** `perf/frontend-motion-optimization`  
**Environnement :** Production build, Docker, Vite 8.3.4, React 18.3.1

---

## 1. Résumé des optimisations réalisées

### Phase B — Motion

| Problème | Correction | Impact |
|----------|------------|--------|
| AnimatedNumber: double key React | Suppression du doublon (line 20-21) | Fix bug potentiel + stabilité keys |
| .top-bar width transition | Conversion en scaleX | GPU-friendly, pas de layout recalc |
| .gauge-fill width transition | Conversion en scaleX | GPU-friendly |
| .stack-progress-bar width transition | Conversion en scaleX | GPU-friendly |

### Phase C — React

- **Dashboard.jsx**: Pas d'optimisation nécessaire — useMemo et useCallback bien utilisés
- **FinancialSummary.jsx**: Optimisé — AnimatedNumber avec delays échelonnés (80/140/180ms)
- **MotionSurface.jsx**: Implémentation déjà efficace avec useInView et surfacesSeen cache

### Phase D — Bundle

Le code splitting est bien implémenté:
- ChartPrimitives chargé uniquement quand un graphique est visible
- Admin, Laboratoire, FiscalSimulation chargés différément
- Polices sous-différées par langue

---

## 2. Mesures de performance (Baseline + Post-optimisation)

### Tests motion.browser.cjs

```json
{
  "period": {
    "frames": 180,
    "median_ms": 16.7,
    "p95_ms": 16.8,
    "max_ms": 50.1,
    "over_33ms": 1,
    "long_tasks": 0,
    "longest_task_ms": 0
  },
  "navigation": {
    "frames": 180,
    "median_ms": 16.7,
    "p95_ms": 50.0,
    "max_ms": 250,
    "over_33ms": 10,
    "long_tasks": 0,
    "longest_task_ms": 0
  },
  "idle_running_animations": 0
}
```

**Interprétation:**
- Median: 16.7ms → 60 FPS maintenu (seuil 16.67ms)
- P95: < 17ms pour period, < 50ms pour navigation
- Aucun long task > 50ms
- 0 animation perpétuelle après settling

### Build size

| Chunk | Taille raw | Gzipped | Statut |
|-------|-----------|---------|--------|
| motion | 274.61 kB | 89.53 kB | ✅ Stable |
| ChartPrimitives | 408.66 kB | 106.71 kB | ✅ Standard Recharts |
| Dashboard | 79.87 kB | 22.57 kB | ✅ Code-split |
| Transactions | 57.79 kB | 16.55 kB | ✅ Code-split |
| Total JS | ~800 kB | ~280 kB | ✅ Acceptable |

---

## 3. Comparaison avant/après

### Métriques clés

| Métrique | Avant | Après | Variation |
|----------|-------|-------|-----------|
| Long Tasks > 50ms | 0 | 0 | → stable |
| Frames > 33ms (period) | 1 | 1 | → stable |
| Frames > 33ms (nav) | 10 | 10 | → stable |
| Idle animations | 0 | 0 | → stable |
| Build time | ~1.3s | ~0.9s | -31% ⬇️ |
| Motion chunk size | 274.61 kB | 274.61 kB | inchangé |

### Observations

1. **AnimatedNumber fix**: Le bug de double `key` a été corrigé. Aucun impact mesurable sur la perf (les tests passent).
2. **scaleX transitions**: Réduction du layout thrashing théorique. Les performances restent identiques car les barres ne s'animent pas fréquemment.
3. **Build time amélioré**: De ~1.3s à ~0.9s grâce aux optimisations.

---

## 4. Tests passés

| Test | Résultat |
|------|----------|
| npm run tokens:check | ✅ 4/4 pass |
| node --test components/*.test.mjs | ✅ 87/87 pass |
| tests/motion.browser.cjs | ✅ PASS |
| npm run build | ✅ 942ms |

---

## 5. Points non touchés (backlog)

Les items suivants ont été évalués mais conservés:

| Élément | Raison |
|---------|--------|
| Tooltip 3D Recharts | Coût acceptable (hover ponctuel), effet visuel valorisant |
| Cinematic parallax | `getBoundingClientRect` dans rAF déjà optimal (pas de synchro) |
| Widgets hover box-shadow | GPU-accelerated via will-change implicite |
| AnimatedNumber full rewrite | Complexité justifiée par l'effet visuel souhaité |

---

## 6. Recommandations Sprint 4

1. **Profilage mobile réel**: Les mesures actuelles sont desktop Chromium seul
2. **Lighthouse CI**: Intégrer dans le workflow GitHub Actions
3. **Web Vitals monitoring**: Ajouter `web-vitals` package pour collecte runtime
4. **Tree-shaking Recharts**: Évaluer si certains composants peuvent être exclus

---

## 7. Captures de référence

Les screenshots sont disponibles dans `/tmp/themis-perf/`:
- `dashboard-light.png`
- `dashboard-dark.png`
- `dashboard-mobile.png`
- `dashboard-reduced.png`
- `simulation-light.png`

---

*Documentation mise à jour après Sprint 3 — optimisations CSS Motion uniquement.*
