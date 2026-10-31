# Themis — Codebase Knowledge Graph
*Graphe de référence pour navigation rapide. À mettre à jour à chaque changement significatif.*

---

## Architecture Globale

```
themis/
├── backend/                 # Go 1.23 — API REST + logique métier
│   ├── cmd/server/main.go   # Entrée unique, wire up tout
│   ├── internal/
│   │   ├── api/             # Handlers HTTP (1 fichier = 1 domaine)
│   │   ├── auth/            # Auth JWT + password hash
│   │   ├── ai/              # Enrichissement IA (categorize, analyze)
│   │   ├── categorize/      # Moteur de règles (patterns → domaines)
│   │   ├── config/          # Config centralisée (env + DB)
│   │   ├── db/              # PG connection + migrations embed
│   │   ├── llm/             # Abstraction multi-provider (Ollama, Anthropic, etc.)
│   │   ├── memory/          # Client Knowledge Graph (Graphiti REST)
│   │   ├── models/          # Structs partagées (Transaction, Statement, etc.)
│   │   ├── normalize/       # Nettoyage libellés marchands
│   │   └── parser/          # CSV/PDF/OCR → ParseResult
├── frontend/                # React 18 + Vite + Recharts
│   ├── src/
│   │   ├── pages/           # Routes (Dashboard, Transactions, Assistant, etc.)
│   │   ├── widgets/         # Composants graphiques (Flux, KPI, Donut, etc.)
│   │   ├── api.js           # Client HTTP unifié
│   │   └── components/      # UI réutilisable
├── memory/                  # Service Python Graphiti (FastAPI)
│   ├── main.py              # API /kg/learn, /kg/query, /kg/status
│   ├── Dockerfile
│   └── requirements.txt
├── docker-compose.yml       # Stack complète (db, falkordb, kg, app, ollama)
└── backend/Dockerfile       # Multi-stage: front build + back build + runtime
```

---

## Backend — Fichiers Clés par Domaine

### API Handlers (`backend/internal/api/`)

| Fichier | Responsabilité | Endpoints principaux |
|---------|---------------|---------------------|
| `handlers.go` | Router + handlers core | `/api/health`, `/api/statements*`, `/api/transactions*`, `/api/summary`, `/api/upload/progress`, `/api/period`, `/api/period/compare` |
| `statement_controls.go` | Réconciliation exacte, contrôles de périodes, similitudes documentaires informatives, revue tracée, suppression atomique et mode documentaire | `GET/DELETE /api/statements*`, `PATCH /api/statements/{id}/validation`, `PATCH /api/accounts/{id}` |
| `import_preview.go`, `import_fidelity.go` | Aperçu privé, rejeu idempotent par empreinte source, traçabilité des occurrences, similitudes consultatives et rapprochement avant validation automatique | Aperçu/confirmation des imports |
| `transactions.go` | Filtres partagés, rapport exact paginé, édition atomique et provenance | `GET /api/transactions[?view=report]`, `PATCH /api/transactions/{id}` |
| `transaction_rules.go` | Validation des règles et application atomique, périmètre d'import | Services des règles existantes |
| `knowledge_activity.go` | Journal mémoire propriétaire, inspection FalkorDB bornée, recherche et trace des faits joints au modèle | `GET /api/knowledge/overview`, `POST /api/knowledge/search`, `POST /api/knowledge/learn` |
| `chat.go` | Assistant conversationnel | `POST /api/chat` (orchestration outils) |
| `pulse.go` | Synthèse rapide (mois demandé) | `GET /api/pulse?month=YYYY-MM` |
| `hero.go` | Conclusion analyse (ton + points) | `GET /api/hero?month=YYYY-MM` |
| `anomalies.go` | Détection signaux (abos, unusual) | `GET /api/anomalies?month=YYYY-MM` |
| `discoveries.go` | Découvertes vagues 2 (duplicatas, drift) | `GET /api/intel/discoveries` |
| `journal.go` | Fil chronologique unifié | `GET /api/journal?month=YYYY-MM` |
| `remaining.go` | Reste-à-vivre (proj. ou hist.) | `GET /api/remaining?month=YYYY-MM` |
| `features.go` | Budgets, catégories, règles | `/api/budgets*`, `/api/rules*`, `/api/categories` |
| `features2.go` | Comptes, domaines et tags | `/api/accounts*`, `/api/domains*`, `/api/tags*` |
| `engagement_*.go` | Crédits/taxes : centimes exacts, paiements déclarés, propriétaire, historique atomique et synthèse cohérente | `/api/credits*`, `/api/taxes*`, `/api/obligations` |
| `export.go` | Export propriétaire, y compris paiements/historiques d’engagements retirés | `/api/export` |
| `features3.go` | Analytics avancées | `/api/analytics/*`, `/api/intel/*` |
| `narrative.go` | Synthèse narrative IA | `GET /api/intel/narrative` |
| `coherence.go` | Vérif cohérence données | `GET /api/coherence` |
| `merchant_alias.go` | Renommage marchands | `/api/merchants/*` |
| `admin.go`, `ai_configuration.go`, `ai_diagnostics.go` | Clients IA, configuration globale masquée, validation administrative, révisions et contrôles explicites | `/api/admin/config`, `/api/admin/ai/test`, `/api/admin/ollama/models`, `/api/admin/ai/memory/test` |
| `period.go` | Infos période + comparaison | `GET /api/period`, `GET /api/period/compare` |

### Modèles (`backend/internal/models/models.go`)
```go
Transaction { ID, Date, Label, Display, Merchant, Amount, DomainID, Tags, Hash }
Statement   { ID, Filename, PeriodStart/End, Balances, Parser, Validated }
Category    { ID, Name, Color, IsTransfer }
Rule        { ID, Pattern, DomainID }
```

### Validation documentaire des relevés
- Un comptage absent du document est `not_applicable` ; aucune référence n’est déduite du nombre extrait.
- Les débits et crédits déclarés sont comparés séparément au solde rapproché. La traçabilité de chaque opération et l’absence de lecture rejetée/ambiguë sont également requises pour la validation automatique.
- L’exhaustivité demeure `unknown` sans comptage indépendant, même quand les autres preuves concordent.
- Les occurrences répétées dans un document sont conservées. Seule la réimportation byte-identique avec un résultat inchangé est reconnue comme doublon confirmé ; les similarités entre imports sont consultatives.
- Le rapport d’audit historique est en lecture seule ; ces contrôles ne réécrivent pas les relevés déjà validés.

### Config (`backend/internal/config/config.go`)
- `Load()` lit `.env` + defaults
- Timeouts IA par opération
- **Nouveau** : `MemoryServiceURL`, `MemoryEnabled`

### Memory Client (`backend/internal/memory/memory.go`)
```go
Client{Learn, LearnSource, Query, Inspect, SearchFacts, BuildFinancialFacts}
```
- Appelé par `knowledge_activity.go` : contexte assistant et apprentissage après import confirmé/réconcilié. `inspection.go` valide les réponses, les limites et les groupes propriétaires ; erreur ≠ résultat vide.
- `memory/inspection.py` lit le graphe nommé par le groupe via `GRAPH.RO_QUERY`, sans cloner le driver ni créer de graphe absent. Graphiti est fixé à 0.31.2.
- UI : page `Knowledge`, `KnowledgeActivity` (également Admin → IA → Mémoire), `KnowledgeFacts` et validation `knowledgeModel.mjs`. Voir `docs/KNOWLEDGE_ACTIVITY.md`.

### Migrations (`backend/internal/db/migrations/`)
- `001_init.sql` → `033_knowledge_activity.sql` (appliquées 1x via `schema_migrations`, rollbacks explicites et gardés hors du dossier embarqué)

### Remboursements manuels
- `reimbursements.go` expose les routes authentifiées `/api/reimbursements*`; `reimbursement_service.go` porte les snapshots, allocations en centimes, verrous, statuts et historique.
- Migration 027 ajoute dossiers, allocations, historique, clés d’idempotence et contraintes multi-propriétaires; `db/rollbacks/027_reimbursements.sql` est un rollback gardé qui refuse la perte de données.
- `statement_controls.go` bloque la suppression d’un relevé dont les opérations ont des liens actifs; les triggers PostgreSQL préservent cette règle hors API.
- `TransactionReimbursements.jsx` est monté dans `TransactionDrawer` de `TransactionDetails.jsx`; `api.js` fournit les appels et `reimbursementModel.mjs` vérifie les montants en centimes. Les coûts nets affichés au détail n’alimentent aucun agrégat du MVP.
- Tests : `reimbursements_integration_test.go`, `reimbursement_migration_test.go` et `reimbursementModel.test.mjs`.

---

## Frontend — Structure

- `PageLoader.jsx` charge les pages à la demande avec Suspense et une récupération en cas d’échec ; `App.jsx` précharge la destination au survol/focus. Auth reste immédiat. `tests/loading.browser.cjs` vérifie le build réel.
- `IntelligenceHighlights.jsx` suspend le polling des jobs dans les onglets masqués et le reprend au retour sans chevauchement. Mesures et limites : `docs/PERFORMANCE_OPTIMIZATION.md`.

### Mouvement et interactions (`frontend/src/motion/`)
- `index.jsx` centralise le provider Motion, les transitions de navigation (View Transitions avec repli), les révélations au scroll, cartes, onglets, panneaux et jauges. `PresencePanel` rend les éléments sortants inertes.
- `tokens.js` définit les rythmes et springs ; `preference.js` suit le mouvement réduit dès le premier rendu et à chaud ; `motion.css` harmonise les interactions après les feuilles existantes.
- `AnimatedNumber.jsx` anime les caractères déjà formatés, en gardant le montant exact dans le texte accessible/copiable. Les valeurs absentes restent absentes.
- `AnimatedChart.jsx` révèle la géométrie Recharts avec Web Animations, sans changer les séries ni remonter le graphique lors du zoom. `HeroFlow.jsx` présente les flux disponibles sur douze mois calendaires, avec ruptures pour les mois absents et tableau accessible.
- `FinancialSummary` et les KPI identifient la période des valeurs conservées pendant le chargement et ignorent les réponses obsolètes. Un mois absent ne reprend pas les KPI du mois précédent.
- `useModal.js` conserve le dialogue natif et la restauration du focus ; une copie décorative inerte termine la sortie après fermeture immédiate de l’état React, pour préserver les réouvertures rapides des transactions/relevés.
- Référence : `docs/MOTION_DESIGN.md`. Validation : `tests/motion.browser.cjs` (API synthétique, montants, périodes, navigation rapide, mouvement réduit, accessibilité, performances) et scénarios navigateur/E2E existants.

### Dashboard cinématique (reprise PR #39)
- `motion/preference.js` unifie le choix persistant et la priorité système pour Motion, WAAPI, navigation et dialogues ; `Motion.jsx` conserve les révélations terminées en session.
- `PageTransition` mémorise le mode d'entrée de chaque page ; la fin d'une transition native ne relance plus l'entrée CSS ni le décalage vertical de Simulation. Régression couverte par `motion.browser.cjs`.
- `CinematicDashboard.jsx` / `cinematic.css` portent un bandeau adaptatif et un décor SVG à animation finie. Le contenu et les contrôles ne dépendent pas du décor.
- `components/dashboardModel.mjs` partage la sélection du mois, le net exact et les comparaisons de périodes complètes entre les KPI du dashboard et le widget configurable.
- `Dashboard.jsx` sérialise les sauvegardes de disposition. `widgets/index.jsx` garde les gradients propres à chaque instance et la légende des catégories accessible.
- Validation : `dashboardModel.test.mjs`, `cinematic.browser.cjs`, `motion.browser.cjs`, parcours métier et E2E. Voir `docs/CINEMATIC_DASHBOARD.md` et `docs/CINEMATIC_VALIDATION.md`.

### Cinematic Experience 2.0
- `Dashboard.jsx` présente des KPI non redondants (variations comparables, taux d’épargne existant et reste à vivre) et différencie panneaux Hero/Focus/Standard/Quiet sans toucher aux calculs métier ou aux préférences personnalisées.
- `widgets/chartCalendar.mjs` complète uniquement la géométrie des courbes par des mois `null` pour ne pas relier deux observations séparées par une absence. `widgets/index.jsx` expose la légende de flux au clavier, un tableau de valeurs accessible et des moyennes qui gardent le dénominateur des périodes observées ; le cumul interrompt sa courbe au même endroit et marque sa dernière valeur.
- `motion/AnimatedChart.jsx` chorégraphie grille/axes, courbes décalées, aires et légende à l’entrée ; une nouvelle révision utilise un fondu bref de sa géométrie exacte, sans morphing de valeurs ni répétition du dessin au zoom.
- `motion/CinematicDashboard.jsx` et `motion/cinematic.css` ajoutent la parallaxe discrète de l’orbite au pointeur souris et une grille desktop asymétrique. Le mobile garde les décorations et les tables SR hors du défilement horizontal (`motion.css`).
- Audits/décisions : `docs/CINEMATIC_DESIGN_AUDIT.md` et `docs/CINEMATIC_DESIGN_SYSTEM.md`. Vérification ciblée : `npm run test:cinematic`, `npm run test:motion` et `node --test src/widgets/chartCalendar.test.mjs src/components/dashboardModel.test.mjs`.

### Pages (`frontend/src/pages/`)
| Page | Description |
|------|-------------|
| `Dashboard.jsx` | Vue principale (Hero, KPI, Journal, widgets grid) |
| `Transactions.jsx` | Synthèse filtrée, pagination serveur, classement, sélection et drawer |
| `Analyse.jsx` | Analyse mensuelle IA + graphiques |
| `Assistant.jsx` | Chat IA full-screen |
| `Budgets.jsx` | CRUD budgets + suivi |
| `MerchantList.jsx` / `MerchantRenamer.jsx` | Gestion marchands |
| `TagManager.jsx` / `TagFolders.jsx` | Tags + dossiers |
| `Simulate.jsx` | Simulateur projections |

### Widgets (`frontend/src/widgets/index.jsx`)
| Widget | Composant | Data attendue |
|--------|-----------|---------------|
| `kpi` | `KpiWidget` | `data.monthly[]` (avec `is_current`, `status`, `days_observed`) |
| `flux` | `FluxWidget` + `FluxChart` | `data.monthly[]` |
| `solde` | `SoldeWidget` | `data.monthly[]` |
| `categories` | `CategoriesWidget` | `data.monthly[]`, `data.by_category[]` |
| `trend` | `CategoryTrendWidget` | `data.by_category[]` |
| `top` | `TopWidget` | `api.transactions({ month, type: 'debit' })` (backend financial scope; owner, transfer descendants and savings) |
| `budgets` | `BudgetsWidget` | `api.budgets()` |
| `subs` | `SubsWidget` | `api.recurring()` |
| `attention` | `AttentionWidget` | `api.anomalies()` |

### API Client (`frontend/src/api.js`)
```js
export const api = {
  summary(months, month), hero(month), pulse(month), anomalies(month), journal(month),
  period(), periodCompare(metric), chat(messages, convId),
  remaining(month), budgets(), recurring(), categories(),
  transactions(params), statements(), upload(file, opts),
  // ...
}
```

---

## Knowledge Graph (Graphiti)

### Service Python (`memory/main.py`)
```python
# FastAPI sur :8100
POST /kg/learn     # group_id, facts[], source → add_episode()
POST /kg/query     # group_id, query, top_k → search()
GET  /kg/status/{group_id}
POST /kg/clear/{group_id}
```

### Modèles de données Graphiti
- **Nodes** : Merchant, Domain, Month, User, TransactionPattern
- **Edges** : `ACHETE_CHEZ`, `CATEGORISE_COMME`, `DANS_MOIS`, `RECURRENT_AVEC`
- **Temporal** : Chaque épisode a `reference_time`, relations versionnées

### Intégration Go
```go
// Dans chat.go::orchestrate()
memCtx := s.getMemory().SearchFacts(ctx, ownerID, userQuestion, 5)
systemPrompt += memCtx  // Injecté avant appel LLM

// Dans handlers.go::uploadStatement() (async, non bloquant)
go func() {
    facts := memory.BuildFinancialFacts(res, owner)
    s.Memory.Learn(ctx, groupID, "transaction_import", facts)
}()
```

---

## Base de Données — Tables Principales

| Table | Description |
|-------|-------------|
| `users` | Auth (id, email, password_hash, created_at) |
| `accounts` | Comptes bancaires (id, owner_id, name, type, currency) |
| `statements` | Relevés importés (id, owner_id, account_id, filename, period_*, balances, parser, uploaded_at) |
| `transactions` | Transactions (id, statement_id, owner_id, tx_date, label, display_label, merchant, amount, domain_id, hash, label_kind) |
| `domains` | Catégories (id, name, color, parent_id, is_transfer) |
| `tags` + `transaction_tags` | Tags utilisateur |
| `rules` | Règles de categorisation (pattern, domain_id, tags[]) |
| `budgets` | Budgets mensuels (owner_id, domain_id, monthly_limit) |
| `recurring` | Prélèvements détectés (merchant, avg_amount, months[], regular) |
| `chat_conversations` / `chat_messages` | Historique assistant |
| `settings` | Clé/valeur (layout dashboard, ai_config, etc.) |
| `schema_migrations` | Suivi migrations appliquées |

---

## Flux de Données Clés

### 1. Import Relevé
```
POST /api/statements (multipart)
  → parser.Parse() ou parser.ParseCSV()
  → persist() : INSERT statements + transactions (ON CONFLICT hash)
  → applyAllRules() : domaines + tags via rules engine
  → ai.AutoEnrich() : domaines manquants via LLM (si activé)
  → memory.Learn() : facts → Graphiti (async)
  → Response {statement_id, inserted, ai_categorized}
```

### 2. Chat Assistant
```
POST /api/chat {messages[], conversation_id?}
  → resolve/create conversation
  → load history
  → orchestrate() loop (max 4 rounds):
      systemPrompt + toolsDef + history + memory facts
      → LLM → tool_calls ou answer
      → execute tools (search_transactions, financial_summary, etc.)
      → append results to context
  → persist assistant message
  → Response {conversation_id, answer, evidence}
```

### 3. Dashboard Load
```
GET /api/summary?months=24&month=YYYY-MM
GET /api/hero?month=YYYY-MM
GET /api/pulse?month=YYYY-MM
GET /api/journal?month=YYYY-MM
GET /api/anomalies?month=YYYY-MM
GET /api/remaining?month=YYYY-MM
→ Dashboard.jsx assemble + widgets grid (tous reçoivent selectedMonth)
```

---

## Patterns Récurrents

### Handler Standard
```go
func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
    owner := s.ownerID(r)
    // 1. Parse input
    // 2. Validate
    // 3. DB query (toujours avec owner_id)
    // 4. Transform
    // 5. writeJSON / writeErr
}
```

### Auth Middleware
```go
func (s *Server) authMiddleware(next http.Handler) http.Handler {
    // Cookie session → owner_id dans context
    // 401 si absent/invalide
}
```

### Pagination / Filtres Transactions
```go
// Query params: month, from, to, q, domain, tag, kind
// LIMIT 1000 hardcodé
```

### Réponses JSON Standard
```go
writeJSON(w, http.StatusOK, data)        // Succès
writeErr(w, http.StatusBadRequest, err)  // Erreur client
writeErr(w, http.StatusInternalServerError, err) // Erreur serveur
```

---

## Variables d'Environnement (`.env`)

```bash
# PostgreSQL
POSTGRES_USER=themis
POSTGRES_PASSWORD=xxx
POSTGRES_DB=themis
POSTGRES_PORT=5432
DATABASE_URL=postgres://...

# App
APP_PORT=8080
COOKIE_SECURE=false

# IA
AI_PROVIDER=openai|ollama|anthropic|mistral|freellm|none
AI_TRUSTED_ENDPOINTS=    # Bases HTTP(S) exactes autorisées côté serveur, séparées par virgules
OLLAMA_URL=http://ollama:11434/v1   # Variable legacy - préférer REMOTE_OLLAMA_BASE_URL
REMOTE_OLLAMA_BASE_URL=   # URL Ollama distante (Mac hôte), vide = conteneur local (profile local-ai)
OLLAMA_MODEL=qwen2.5:14b
ANTHROPIC_API_KEY=xxx
MISTRAL_API_KEY=xxx
FREELLM_API_KEY=xxx
AI_AUTO_CATEGORIZE=true

# Timeouts (secondes)
AI_OLLAMA_TIMEOUT=600
AI_CHAT_TIMEOUT=480
AI_CATEGORIZE_TIMEOUT=300

# Knowledge Graph
MEMORY_ENABLED=true
MEMORY_SERVICE_URL=http://knowledge-graph:8100
OLLAMA_EMBED_MODEL=nomic-embed-text

# Ollama (profile local-ai)
OLLAMA_IMAGE=ollama/ollama:latest
OLLAMA_PORT=11434
```

> **docker-compose.yml** : `REMOTE_OLLAMA_BASE_URL` est la variable clé pour pointer Ollama distant (Mac hôte). Le profile `local-ai` est découplé pour `knowledge-graph`/`falkordb`. Cohérence maintenue entre `docker-compose.yml` et `compose-supabase/docker-compose.yml`.

---

## Commandes Utiles

```bash
# Build + run complet (avec IA locale)
docker compose --profile local-ai up -d --build

# Backend seul (rebuild Go)
docker compose build app && docker compose up -d app

# Knowledge Graph seul
docker compose --profile local-ai up -d knowledge-graph falkordb ollama

# Logs
docker compose logs -f app
docker compose logs -f knowledge-graph

# Tests Go
docker run --rm -v $(pwd)/backend:/src -w /src golang:1.23-alpine go test ./...

# Migration DB manuelle
docker compose exec db psql -U themis -d themis -c "\dt"
```

---

## Points d'Attention / Dette Technique

1. **Pas de tests automatisés** — ajouter `*_test.go` critiques
2. **Frontend non typé** — migrer vers TypeScript
3. **Migrations SQL non versionnées dans le code** — `embed.FS` OK mais pas de rollback
4. **Knowledge Graph optionnel** — bien géré (nil checks partout)
5. **Périodes incomplètes** — **RÉSOLU** via `period.go` + champs `is_current`/`status` dans `summary`
6. **Comparaisons mois partiel vs complet** — **RÉSOLU** via `comparePeriod()` (mêmes jours)

---

## Navigation Temporelle Dashboard — Spécifications

### Objectif
Rendre le dashboard navigable dans le temps sans nouvelle page ni duplication de composants.

### Architecture
- **État unique** : `selectedMonth` (format `YYYY-MM`) dans `Dashboard.jsx`
- **Propagation** : Passé à tous les widgets via props + API calls
- **Backend** : Paramètre `month` optionnel sur tous les endpoints période-dépendants

### Composants Ajoutés/Modifiés

| Composant | Rôle |
|-----------|------|
| `MonthSelector` (dans `Dashboard.jsx`) | UI : prev/next, dropdown 24 mois, bouton "mois en cours" |
| `Hero`, `KpiRow`, `Journal` | Reçoivent `selectedMonth`, appellent API avec `month` |
| `CategoriesWidget`, `KpiWidget` | Filtrent `data` selon `selectedMonth` |
| `TopWidget` | Charge ses débits via l'API Transactions, avec le mois sélectionné et le filtre économique backend partagé |
| `BudgetsWidget`, `AttentionWidget` | Appellent API avec `selectedMonth` |
| `FluxWidget`/`FluxChart`, `SoldeWidget`, `CategoryTrendWidget` | Ligne verticale "Mois consulté" sur graphiques historiques |

### Endpoints Backend Mis à Jour

| Endpoint | Paramètre | Comportement |
|----------|-----------|--------------|
| `/api/pulse` | `?month=YYYY-MM` | Pulse pour le mois demandé |
| `/api/hero` | `?month=YYYY-MM` | Analyse pour le mois demandé |
| `/api/journal` | `?month=YYYY-MM` | Événements (salaire, wins) pour le mois |
| `/api/anomalies` | `?month=YYYY-MM` | Anomalies + `period` info |
| `/api/remaining` | `?month=YYYY-MM` | Mode historique (net réel) ou projection |
| `/api/summary` | `?month=YYYY-MM` | Série filtrée ≤ mois demandé |

### Gestion Périodes Incomplètes
Réutilise `period.go` (`getPeriodInfo`, `comparePeriod`) :
- `is_current` : mois en cours réel (date application)
- `status` : `current` | `complete` | `incomplete` (< 90% couverture)
- `days_observed` / `days_in_month` / `coverage_ratio`
- Badges visuels sur Hero, KPI, Donut, Top

### Séparation des Concepts
| Concept | Source | Affichage |
|---------|--------|-----------|
| **Mois sélectionné** | UI (`selectedMonth`) | Sélecteur + badges |
| **Dernier import** | `statements.uploaded_at` | Hero meta + coverage |
| **Couverture données** | `transactions.tx_date` max | Hero meta (`last_tx`, `stale`) |

### Tests Validés
1. Affichage initial → mois courant
2. Navigation précédent/suivant
3. Sélection directe mois historique
4. Retour mois courant (bouton ●)
5. Janvier → Décembre année précédente
6. Cohérence entre tous composants
7. Mois historique incomplet → badge + avertissement
8. Mois sans transactions → data vide (pas d'erreur)
9. Mois sélectionné ≠ dernier import
10. Graphiques historiques cohérents (ligne "Mois consulté")
11. Aucune régression endpoints existants

---

## Dernières Modifications (Session Actuelle)

- ✅ `backend/internal/api/period.go` — Infos période + comparaison comparable
- ✅ `backend/internal/api/pulse.go` — Mois courant : compare aux mêmes jours, message adapté
- ✅ `backend/internal/api/hero.go` — Pas d'alerte structurelle sur mois courant, badge période
- ✅ `backend/internal/api/handlers.go` — `summary` enrichi + endpoints `/api/period*`
- ✅ `backend/internal/api/journal.go` — Paramètre `month` + helpers mis à jour
- ✅ `backend/internal/api/anomalies.go` — Paramètre `month` + renvoie `period`
- ✅ `backend/internal/api/remaining.go` — Paramètre `month` + mode historique/projection
- ✅ `frontend/src/api.js` — Toutes fonctions API acceptent `month`
- ✅ `frontend/src/pages/Dashboard.jsx` — **NOUVEAU** Sélecteur mois (`MonthSelector`) + état `selectedMonth`
- ✅ `frontend/src/widgets/index.jsx` — **NOUVEAU** 9 widgets acceptent `selectedMonth` + highlight graphiques
- ✅ `frontend/src/styles.css` — **NOUVEAU** Styles sélecteur mois + badges période (KPI, donut, top, hero)
- ✅ `memory/main.py` — Service Graphiti FastAPI (FalkorDB + Ollama)
- ✅ `backend/internal/memory/memory.go` — Client Go + BuildFinancialFacts
- ✅ `docker-compose.yml` — Services falkordb + knowledge-graph
- ✅ `backend/Dockerfile` — Build multi-stage inchangé

---

*Mise à jour : 2026-10-23 — Session navigation temporelle dashboard + Knowledge Graph + Correction périodes financières*
### Synthèse financière — refonte UX (2026-10-23)

- `frontend/src/components/FinancialSummary.jsx` + CSS dédié : remplace Hero, surface unique responsive, flux dominants, alertes hiérarchisées et dates distinctes. Modèle de présentation et tests Node dans `financialSummaryModel.*`.
- `/api/hero` expose les calculs existants `income`, `expenses`, `net` et `period_complete`. `hero_diagnosis.go` vérifie une couverture continue par relevés validés pour tous les comptes soumis au contrôle documentaire, indépendamment du statut approximatif de `period.go`. En absence de preuve : bilan incomplet.
- Diagnostic provisoire indépendant du signe des flux et sans hypothèse de salaire. Alertes réelles conservées ; comparaisons limitées aux trois mois précédents documentés ; signaux récurrents limités au mois de leur dernière opération.
- Tests : `hero_diagnosis_test.go` (cycle de salaire, alertes, intervalles de relevés), tests Node de présentation.

### Dépendances frontend — sécurité (2026-10-23)

- Vite 8.3.4 et `@vitejs/plugin-react` 6.1.2 : mise à niveau conjointe pour respecter les peer dependencies. Build avec Node.js 22 (Docker et CI).
- Correctifs PostCSS, Browserslist et baseline-browser-mapping intégrés via Dependabot.

### Configuration IA — Admin (consolidation 2026-10-25)

- `backend/internal/llm/openai.go` : client Responses HTTP (`store: false`), sorties texte/JSON, détection des réponses incomplètes et erreurs sans corps sensible. Aucun SDK ajouté.
- `ai_configuration.go` : configuration globale `settings.ai_config` (`owner_id IS NULL`), fusion des PUT partiels, secrets absents du DTO, `clear_keys` explicite, repli environnement et `expected_revision` facultatif pour compatibilité. Les endpoints IA imposent un administrateur authentifié même avant le premier compte ; la clé est interdite aux réglages génériques.
- `ai_diagnostics.go` : contrôles explicites et bornés, résultats datés et liés aux paramètres testés ; le test ne persiste pas la configuration. Contrôle mémoire facultatif distinct, sans lecture du graphe. Les diagnostics sont en mémoire et redeviennent non testés après redémarrage.
- `llm/network.go` : URL Ollama/FreeLLM validées, DNS vérifié à la connexion et IP fixée, sans proxy ambiant ni redirection. Bases privées/HTTP autorisées uniquement via `OLLAMA_URL`, `FREELLM_URL` ou `AI_TRUSTED_ENDPOINTS` côté opérateur. Métadonnées/link-local toujours refusées. Les erreurs des fournisseurs ne renvoient pas les corps externes.
- `config.go` + configurations Compose : `OPENAI_API_KEY` / `OPENAI_MODEL`. Modèle par défaut `gpt-4.1-mini`, modifiable.
- `frontend/src/components/AIConfiguration.jsx` + `.css` + `aiConfigurationModel.mjs` : panneaux Fournisseur, Comportement, Diagnostic et Mémoire ; brouillon, activation et connexion vérifiée distincts. `Admin.jsx` conserve les autres actions administratives. `App.jsx` répercute l’activation IA ; `api.js` annule les lectures et contrôles obsolètes. La mémoire Graphiti garde sa configuration serveur séparée.
- Tests `llm/openai_test.go`, `api/admin_openai_test.go`, `config/openai_test.go` : contrat Responses simulé, erreurs/refus/troncature, persistance, masquage et sélection du fournisseur.
- Tests supplémentaires : `api/ai_configuration_test.go`, `llm/network_test.go`, `memory/health_test.go`, tests du contexte, `aiConfigurationModel.test.mjs` et `tests/ai-configuration.browser.cjs` intégré à la CI. Rapports : `docs/AI_CONFIGURATION_{AUDIT,CHANGES,TESTS}.md`.

### CI — consommation GitHub Actions

- `.github/workflows/ci.yml` exécute les tests Node `src/components/*.test.mjs` avant le build frontend.
- Le job `E2E - Playwright` utilise une base PostgreSQL 16 dédiée `themis_e2e`, démarre le vrai serveur Go et Vite via `frontend/playwright.config.ts`, puis lance les tests Playwright authentifiés de `frontend/tests/e2e/`. Les tests navigateur `frontend/tests/*.browser.cjs` restent séparés et conservent leurs API simulées et vérifications d’accessibilité.
- `frontend/tests/e2e/global-setup.ts` crée deux propriétaires fictifs et importe le PDF anonyme de `backend/internal/testfixture`; les totaux, soldes et opérations sont vérifiés au niveau API avant les tests UI. `backend/cmd/e2efixtures` fournit le PDF et l'expiration de sessions uniquement pour la base nommée `themis_e2e`.
- `docs/PLAYWRIGHT_E2E.md` documente l'exécution locale reproductible et les garde-fous de confidentialité.

### Relevés — contrôle documentaire (2026-10-25)

- `accounts.tracking_mode` distingue `statements` et `manual`. Migration 025 : initialisation selon le type métier existant, jamais le nom ; modification explicite dans Comptes. Les pointages et valorisations restent séparés des transactions, sans exigence mensuelle de relevé.
- `statement_controls.go` enrichit le tableau historique de `/api/statements` : compte, centimes, réconciliation des écritures conservées, contrôles inconnus/échoués/soupçons, source de validation, 20 dernières revues. Les montants restent en NUMERIC/centimes ; les conversions flottantes ne servent que les anciens champs JSON.
- `statement_recheck.go` expose `POST /api/statements/{id}/checks` (`deterministic` ou `combined`) : photographie en lecture seule, contrôle session/propriétaire, relecture facultative via le client LLM existant et `selectAnalysisEvidence`. Seuls les constats sans données bancaires ni identifiants sont transmis, plafonnés à 20 ; aucun verdict IA ou modification financière. Les échecs restent prioritaires et la requête est bornée à 30 s. Résultats de relance non persistés.
- `statementDocumentedSQL` est partagé par la couverture et la provenance des Transactions : validation déclarée, période, propriétaire du compte, égalité comptable et absence de rupture confirmée entre soldes consécutifs. Les anciens flags incohérents restent stockés mais ne prouvent plus la couverture.
- Après une validation manuelle autorisée, `finalizeStatementStatus` conserve le statut `validated` malgré les avertissements non bloquants ; ils restent détaillés. Les contrôles en échec continuent de bloquer la validation.
- `reliability.go`, `hero_diagnosis.go`, `period.go` et la comparaison marchands excluent les comptes manuels du contrôle documentaire ; sans compte documentaire, la complétude reste inconnue. Les intervalles inclusifs partagés restent utilisés et les contrôles du mois courant s’arrêtent à aujourd’hui.
- `PATCH /api/statements/{id}/validation` : `validate`/`reopen` avec justification, verrou propriétaire, validation et historique atomiques. Suppression en cascade explicite, métadonnées conservées dans `statement_validation_history`, liens étrangers incohérents bloqués, cache d’analyses du propriétaire invalidé ; export de l’historique complet.
- `Statements.jsx/.css`, `statementPageModel.mjs` : synthèse, périmètres explicites, filtres/tri, détail natif clavier, revue et suppression confirmée. `DataReliability` reste le composant de couverture. `useModal.js` réutilise le gestionnaire de dialogue de Transactions. Adaptations minimales de Comptes, du texte de couverture d’Analyse et de la navigation.
- Actions du relevé : validation/revue en haut du panneau, relance seule ou avec relecture IA, statuts vert/orange/rouge avec texte et symbole. Une validation automatique peut être confirmée manuellement une fois ; les erreurs mathématiques restent bloquantes.
- `statement_id` devient un filtre commun au rapport paginé et à l’export des transactions ; un rebond depuis Relevés n’impose pas de mois qui masquerait les écritures hors période. L’aperçu d’import calcule son écart en centimes exacts. Les imports partiellement dédupliqués perdent la validation du parsing ; les relevés réellement sans opérations sont conservés.
- Tests : `statement_controls_test.go`, `statement_migrations_test.go`, `statementPageModel.test.mjs`, `statements.browser.cjs` intégré à la CI. Audit et rapport : `docs/STATEMENTS_{AUDIT,CHANGES,TESTS}.md`.
- Le contrôle syntaxique Python (`compileall`) ne nécessite pas l’installation de Graphiti et de ses dépendances ; il ne valide pas les imports ni le comportement du service.
- Le build de l’image applicative attend backend, frontend, mémoire et Compose. Buildx utilise le cache GitHub Actions `themis-app` en mode `min`, sans publication d’image ; une erreur d’export de cache ne masque pas une erreur de build.

### Centre de fiabilité des données

- `GET /api/reliability?month=YYYY-MM` (`reliability.go`) : couverture par compte, intervalles sans preuve documentaire, relevés non validés ou de période inconnue, dernière transaction sur la période et dernier import toutes périodes. Requêtes limitées au propriétaire authentifié. Aucun montant bancaire exposé.
- Le mois courant est examiné du 1er à aujourd’hui ; un mois historique est examiné jusqu’à sa fin. Pas de mois futur. Les périodes inconnues sont signalées sans leur attribuer arbitrairement un mois.
- `uncoveredStatementDates` est partagé avec `heroSpansCover` : continuité des relevés validés, y compris cycles clôturés le 21. Pas de nouveau calcul financier ou de migration.
- `DataReliability.jsx` dans Relevés : choix du mois, détails par compte, bouton pour filtrer les relevés concernés et accès à l’import existant. Lien depuis le pied de FinancialSummary transmettant le mois du dashboard.
- Tests Go des intervalles/statuts et test PostgreSQL optionnel `TestReliabilityPostgres` (`THEMIS_TEST_DATABASE_URL`, tables temporaires), tests Node `dataReliabilityModel.test.mjs`.

### Prévisualisation des imports

- `POST /api/statements/previews` réutilise le parsing PDF/CSV, sans persister de relevé ou de transaction. `import_preview.go` expose le rapport de réconciliation et distingue document source identique, similarité informative et période bancaire recouvrante. Aucun changement avant confirmation.
- Aperçus en mémoire du `Server`, liés au propriétaire et au compte : TTL 15 min, 3 par utilisateur, 32 au total et budget estimé de 32 Mio ; limite de 20 000 opérations par aperçu. Suppression sur expiration par timer, annulation ou confirmation.
- Confirmation `POST /api/statements/previews/{token}/confirm` : consommation atomique du jeton, revérification du compte, puis `finishStatementImport`, commun à l’import direct historique. Nouveaux uploads PDF/CSV : `import_fidelity.go` établit les occurrences par fichier/compte/indice ; rejeu identique sans modification des métadonnées ; similarité intersource consultative ; insertions atomiques sans conflit ignoré.
- Annulation `DELETE /api/statements/previews/{token}` : aucune écriture financière. Réponses de prévisualisation/confirmation non mises en cache. Pas de nouvelle table ni de dépendance.
- `ImportPreview.jsx` (dialog natif) : compte, période, validation, doublons, tableau filtrable/paginé et confirmation explicite. `reviewedImport.mjs` isole le passage de la décision utilisateur à la confirmation ; les imports multiples restent séquentiels.
- Tests `import_preview*_test.go` (dont PostgreSQL optionnel via `THEMIS_TEST_DATABASE_URL`, schéma de test isolé), `importPreviewModel.test.mjs` : annulation, absence d’écriture avant confirmation, isolation, expiration, rejeu, concurrence et déduplication à l’enregistrement.

### Refonte graphique du cockpit (2026-10-24)

- `frontend/src/cockpit.css`, chargé après les styles existants : palette violet, thèmes clair/sombre, navigation et mises en page responsive. Police Fraunces importée dans `main.jsx`.
- `App.jsx` : barre de recherche ouvrant la palette existante, navigation et menu de compte sur petits écrans. `Dashboard.jsx` : nouvelle hiérarchie visuelle et classes par type de widget.
- `FinancialSummary.jsx` et son CSS : présentation des flux et points d’attention ; les calculs et qualifications de périodes restent dans le modèle existant. `widgets/index.jsx` ajoute les montants par catégorie à la légende.
- Validation : tests Node des trois modèles frontend et build de production réussis ; image Docker reconstruite et application vérifiée par HTTP.

### Lisibilité des budgets du dashboard (2026-10-24)

- `BudgetsWidget` affiche `domain_name` (contrat API), les dépenses importées face au plafond mensuel, le pourcentage utilisé et le montant avant plafond ou du dépassement. La barre reste bornée à 100 %, le pourcentage conserve les dépassements.
- `components/budgetWidgetModel.mjs` et tests Node : catégorie API, consommation, plafond atteint et dépassement. Aucune modification du calcul backend ; le texte rappelle que le suivi dépend des opérations importées.

### Audit de fiabilité financière (2026-10-24)

- Rapport, définitions, plan et risques ouverts : `docs/FINANCIAL_DATA_AUDIT.md`. Les anciennes mentions « périodes résolues » ne constituent pas une garantie globale.
- `internal/money` : centimes entiers, précision et limites explicites. Validation/réconciliation du parsing exactes ; insertion des montants transactionnels en décimal. Les autres calculs Go/React restent à migrer.
- CSV : échec explicite sur ligne invalide, double débit/crédit ou dates multiples divergentes ; aucun abandon silencieux d'opération.
- Import : les nouveaux PDF/CSV utilisent une identité de document et d’occurrence ; les empreintes historiques restent inchangées. Verrou par propriétaire, détection des similitudes et suspension des réimports ambigus. Voir le durcissement P0 ci-dessous.
- `period.go` réutilise `loadReliability` et expose `completeness` (`complete`, `partial`, `unknown`) ; la dernière transaction ne prouve plus la couverture. Comparaisons limitées aux mois documentés. `summary` filtre le mois avant LIMIT et expose le même état.
- Pulse : aucune qualification négative d'un déficit provisoire ; comparaison catégorie suspendue faute de couverture homogène. Mémoire IA explicitement indicative, heuristique de complétude fondée sur les volumes supprimée.
- CI backend : PostgreSQL 16 isolé active les tests d'intégration auparavant optionnels. Tests financiers dans `money`, `parser/financial_invariants_test.go`, `api/period_invariants_test.go` et extensions du test d'import PostgreSQL.

### Budgets — audit et fiabilisation métier (2026-10-24)

- `api/budgets.go` centralise plafonds, consommation brute exacte, crédits séparés, couverture documentaire et affectation unique catégorie/enfant. `GET /api/budgets` conserve son tableau ; `GET /api/budgets/report` ajoute synthèse, dates et sources explicables.
- Les sources `/api/transactions?budget_id=...` réutilisent le même périmètre ; `ids` permet une liste exacte d'identifiants, limitée à 1 000 et toujours cloisonnée au propriétaire.
- Projections uniformes désactivées (`projected:null`) ; transferts hérités et mouvements normalisés d'épargne exclus des flux de consommation via `transfer.go`. Hero, journal et signaux de dépassement réutilisent le service budgétaire.
- `components/money.mjs` et modèle du widget : différences en centimes, égalité au plafond distincte du dépassement, état d'erreur explicite. Pas de migration de données financières.
- Audit initial et organisation en trois lots : `docs/BUDGETS_AUDIT.md`. Tests `budgets*_test.go`, `transaction_sources_test.go`, tests Node du widget.

### Budgets — interface de pilotage (2026-10-24)

- `pages/Budgets.jsx` et `Budgets.css` : mois explicite, six indicateurs réconciliés, enveloppes hiérarchisées, création/édition contextuelles et suppression confirmée, opérations non catégorisées et couverture documentaire détaillée.
- `budgetPageModel.mjs` vérifie les totaux au centime et distingue atteint/dépassé, dépenses et mouvements internes. Chargement, erreur, vide et réponses obsolètes sont traités séparément.
- Navigation `App.jsx` → `Transactions.jsx` transmet le périmètre d'enveloppe ou les identifiants exacts des sources, avec période et avertissement de la limite de 1 000 opérations.
- La section de revue prépare le contrat `/api/recurring/review` du troisième lot ; une indisponibilité reste une erreur explicite, sans repli sur les anciennes détections non fiabilisées.
- Test navigateur reproductible : `frontend/tests/budgets.browser.cjs` (API synthétiques isolées, CRUD, mois, revue, sources, six formats et Axe).

### Budgets — récurrences, revue et non-régression (2026-10-24)

- `recurring_review.go` : groupes déterministes compte/catégorie/sens/marchand, moyenne par occurrence et médiane exactes, cadence indépendante de la stabilité, couverture sur l'intervalle du groupe, échéance nullable et aucune date repoussée artificiellement.
- `GET /api/recurring/review?month=...`, `PUT /api/recurring/review/{key}` : confirmation/correction/ignorance en métadonnées, sans mutation des transactions. Migration additive `020_recurring_reviews.sql` ; rollback manuel séparé dans `db/rollbacks`, testé sur schéma isolé.
- `features.go` adapte le contrat historique au nouveau moteur ; Hero/anomalies/journal respectent le mois demandé. Widgets et estimations ne déduisent plus un total mensuel de moyennes par transaction. Les retraits/épargne protégés n'entrent pas dans les abonnements.
- Sources complètes par pages de 1 000 dans Transactions (`offset` pour enveloppes, lots d'identifiants pour récurrences), avec réponses obsolètes écartées. Aucun dédoublonnage heuristique des sources.
- Job CI `budgets-ui` : parcours, six formats et Axe via outillage temporaire isolé, requis avant le build Docker. Tests Go/PostgreSQL/rollback et Node existants conservés.
- Compte rendu, formules, migration, résultats réels et risques résiduels : `docs/BUDGETS_IMPLEMENTATION.md`.

### Simulateur budgétaire — UX et cohérence des calculs (2026-10-24)

- `pages/Simulate.jsx` + CSS local : synthèse à cinq indicateurs, catégories dépliables avec recherche/filtre des modifications, saisie de pourcentage, résultats immédiats, panneau sticky calé sur le header et rappel mobile. Hypothèses secondaires dans des sections natives.
- `components/simulationModel.mjs` : calcul pur en centimes/BigInt, réconciliation de la référence, crédits et réductions cohérents au centime. Tests Node pris par la CI existante.
- `api/simulation.go`, extrait de `features3.go` : trois mois civils terminés, diviseur commun, couverture documentaire explicite, transferts exclus des crédits et débits, rationnels exacts pour moyennes et crédit. URL et champs historiques conservés ; provenance et disponibilité ajoutées.
- Tests Go unitaires/PostgreSQL ; `frontend/tests/simulation.browser.cjs` vérifie interactions, accessibilité et six formats avec des API entièrement simulées. Pas de nouvelle dépendance runtime ni de migration.
- Diagnostic, alternatives UX, formules, reproduction des contrôles et limites : `docs/SIMULATOR_UX.md`.

### Transactions — gestion, traçabilité et fiabilité (2026-10-24)

- `api/transactions.go` extrait du routeur : GET tableau historique conservé, option `view=report` (total, page, agrégats NUMERIC exacts, centimes, compte, relevé, classification et propositions), mêmes filtres pour l'export CSV, requêtes obsolètes annulables. Dates calendaires et propriété des catégories/tags/comptes contrôlées.
- PATCH atomique catégorie/tags, ajout/retrait de tags depuis l'état verrouillé, décision explicite `discard_suggestion`. Aucune écriture de montant/date/libellé via cette API. Les origines historiques restent inconnues.
- Migration021 : provenance de catégorie, propositions séparées et index `(owner_id,tx_date,id)`. Rollback manuel conserve les transactions et leurs catégories/tags ; testé sur un schéma isolé.
- `categorize.CompilePattern` réutilisé par import, application et comptage des règles ; les tags à l'import sont limités au nouveau relevé. Domaines étrangers ignorés. `ai/enrich.go` génère des propositions sans appliquer catégorie/tag.
- `financialsql` partage les exclusions par propriétaire (transfert, parent, épargne) entre API et analyse IA. Les liens historiques vers une catégorie étrangère ne peuvent plus influencer les flux ; ils apparaissent sans catégorie dans le rapport. Analyse et principaux outils de synthèse/recherche du chat conservent des décimaux exacts et indiquent une couverture inconnue.
- `normalize_backfill.go` et `merchant_alias.go` : surnoms exacts/motifs séparés par propriétaire, priorité déterministe identique à l’import/redémarrage et version du normaliseur atomique. Réparation des affichages antérieurement mélangés, libellés bancaires bruts conservés.
- `Transactions.jsx/.css`, `TransactionDetails.jsx`, `transactionPageModel.mjs` : largeur ciblée1600, trois indicateurs, filtres communs export/liste, groupes journaliers, drawer clavier, édition multiple confirmée et résultats partiels. Mois courant/couverture distingués ; les groupes de plus de1000 sources sont des lots explicitement incomplets.
- Tests Go/PostgreSQL/API/export/IA/migration/rollback, Node et `frontend/tests/transactions.browser.cjs`. Job CI `budgets-ui` conservé et étendu aux trois pages financières Transactions/Budgets/Simulation ; aucun contrôle désactivé.
- Diagnostic priorisé : `docs/TRANSACTIONS_AUDIT.md`. Contrats, règles, validations et limites : `docs/TRANSACTIONS_IMPLEMENTATION.md`.

### Validation des contrôles Security après synchronisation main (2026-10-24)

- SQL du chat et des séries par tag paramétré ; quatre faux positifs d'audit Semgrep annotés et justifiés avec tests, sans désactivation de contrôle global.
- Images mémoire/PostgreSQL d'initialisation/frontend autonome non root ; Nginx autonome sur8080 avec PID temporaire. Contextes de sécurité Kubernetes et volumes temporaires explicités ; backup atomique et privé.
- Diagnostic Python historique : transport HTTP(S) contrôlé, TLS vérifié, cookies associés ; tests isolés `tests/test_navigation_transport.py` exécutés dans la CI.
- Vérifications, justification des annotations, port du frontend autonome et contraintes des volumes : `docs/DEPLOYMENT_SECURITY.md`. Aucun manifeste appliqué sur un cluster ni volume réel modifié pour ces essais.

### Analyse — exploration, périodes et preuves (2026-10-25)

- `api/analysis_report.go` centralise les flux exacts, catégories réconciliées, sources et couverture pour `/api/intel/nature?month=...`. `reliability.go` peut lire une DB ou une transaction et un intervalle ; rapport et couverture partagent un snapshot en lecture seule.
- `intelligence.go` borne marchands et comparaison au mois choisi : trois mois civils communs documentés pour les catégories, deux fenêtres de même durée documentées pour les marchands. Comparaisons inconnues nulles ; paniers par transaction, parts sur le total complet et limite de 40 marchands explicite.
- `analytics.go`, `features3.go`, `tag_folders.go` : propriétaires concordants, dates strictes, centimes exacts, tags non additifs sur total unique, concentration sans multiplication des occurrences, projets observés à date. Pas de migration.
- `analysis_discoveries.go` réutilise les récurrences et contextualise variations de marchands/tags et motifs annuels bornés. `narrative.go` fait sélectionner des constats vérifiés par le LLM existant ; le serveur produit le texte, impose les réserves et rejette les faits inconnus. Aucun KG ni contexte personnel global comme source transactionnelle.
- `Analyse.jsx/.css` : synthèse immédiate et quatre familles, mois central, couverture explicite, chargement des détails à la demande. `analysisResources.js` annule et écarte les réponses obsolètes ; `analysisPageModel.mjs` valide les sommes au centime. Catégories/tags gardent graphiques et tableaux accessibles ; projets et outils globaux restent disponibles.
- Navigation `App.jsx` → `Transactions.jsx` : filtres exacts catégorie/tag/type/marchand et dates. Paramètre `merchant` additif au filtre/à l’export existants, visible et retirable dans la page Transactions.
- Tests `analysis*_test.go`, `analysisPageModel.test.mjs`, `frontend/tests/analysis.browser.cjs` ; parcours ajouté au job navigateur CI sans retrait des trois autres suites. Diagnostic : `docs/ANALYSIS_AUDIT.md`, définitions/contrats/vérifications/limites : `docs/ANALYSIS_IMPLEMENTATION.md`.

### Analyse par tag — courbes et périodes longues (2026-10-25)

- `components/ChartPrimitives.jsx` partage les couleurs, dégradés, axes et infobulle du Dashboard avec `TagHistoryChart.jsx`. Courbes d’aires non empilées, ruptures pour les mois sans observation, points isolés, brosse de zoom clavier et tableau/source exacts.
- `pages/TagAnalysis.jsx` : durées 6/12/24/36/60/120 mois, ou deux mois personnalisés à valider ; sélection des tags conservée. La durée est propre à l’historique, distincte du croisement catégorie/tag mensuel.
- `api/analytics.go` : fenêtre `months` jusqu’à 600, dates strictes et calendrier complet pour les séries bornées, trous `null` sans ajout de zéros. Agrégations SQL et propriété inchangées ; aucun nouvel endpoint ou schéma.
- `tagHistoryModel.mjs` et tests, `api/tag_history_test.go`, parcours navigateur Analyse étendu : précision, bornes, données absentes, noms spéciaux, dix ans d’historique, zoom clavier et accessibilité.

### Intelligence financière v2 — audit et contrat commun (2026-10-29)

- `docs/INTELLIGENCE_V2_AUDIT.md` cartographie les détecteurs, critères, preuves/couverture, périodes comparables, transferts/remboursements, flottants legacy, mémoire et endpoints; il consigne également les lots d’intégration v2.
- `backend/internal/intelligence/` introduit le modèle Discovery commun. L’identité stable du phénomène (`ID`) est distincte de `ObservationID`; les métriques sont typées et l’évidence référence des transactions/périodes sans copier les libellés sensibles.
- `Score` applique les poids configurables proposés (30/25/20/15/10), reste séparé de la confiance et plafonne les découvertes sans preuve ou couverture complète à 49. `Prioritize` retire les éléments masqués, déduplique et borne la liste avec tri déterministe.
- `Registry.Record` écrit identité et observations immuables dans la migration additive 028, avec empreintes stables, limites explicites et vérification propriétaire des IDs de transaction. `Feed`/`Get`/`History` filtrent systématiquement par owner. `GET /api/intel/feed` (3 par défaut, 50 maximum) et `GET /api/intel/discoveries/{id}` ne recalculent rien au chargement.
- `POST /api/intel/recompute` réutilise la revue de récurrences et les signaux existants ainsi que les détecteurs comportementaux. Calcul borné à 500 découvertes et sérialisé par advisory lock propriétaire; les signaux absents ne sont résolus que lorsque la couverture est complète.
- Migration `030_intelligence_jobs.sql`, `internal/intelligence/jobs.go` et `api/intelligence_jobs.go` fournissent une file PostgreSQL coalescée par propriétaire, reprise avec bail/retry, worker démarré dans le serveur, déclencheurs post-mutation best effort et `GET /api/intel/jobs/status`. Le recalcul manuel demeure disponible.
- Le Dashboard suit le statut du job, montre l’historique versionné à la demande et expose les comparaisons comportementales en graphique et tableau accessible; il ne lance pas d’analyse lourde au chargement.
- `api/behavior_discoveries.go` ajoute les détecteurs Lot 2 au même recalcul. Un snapshot propriétaire en lecture seule exige six mois calendaires entièrement documentés; les faits sont les débits ordinaires exacts après exclusion des transferts. Les signaux couvrent tendances progressives marchand/catégorie, contributions de marchands à une variation (réconciliées aux centimes), fréquence d’achat par marchand et fréquence week-end par catégorie sur deux fenêtres de trois mois. Ils indiquent explicitement que contribution/fréquence ne démontrent ni cause ni abonnement et que la saisonnalité reste possible.
- `intelligence/behavior.go` contient les règles pures et testables : six points consécutifs avec quatre pas en hausse, fenêtres mensuelles sans recouvrement et rapprochement des contributions, ou changement de fréquence réparti sur deux mois récents au minimum.
- `POST /api/intel/discoveries/{id}/feedback` garde l’historique propriétaire des sept actions. `hide` masque; `unhide` attend un recalcul qui confirme à nouveau l’existence du phénomène.
- Migration 029 et `GET/PATCH /api/intel/preferences` conservent le retour courant corrigible/supprimable et une note explicite; au recalcul, les préférences affectent uniquement la pertinence, avec une pénalité similaire bornée pour les signaux jugés peu pertinents.
- `Dashboard.jsx` consomme `GET /api/intel/feed` dans la section `IntelligenceHighlights` (maximum trois cartes, preuves, limites et feedback). `chat.go` expose `get_verified_discoveries` aux questions de l’Assistant; textes bancaires et notes restent des données non fiables.
- Les suites Go avec PostgreSQL couvrent le score, les adaptateurs, l’idempotence, versions, historique, feedback, préférences, réconciliation, lock et isolation; le build frontend passe. La file durable déclenchée après mutations reste nécessaire pour rendre le recalcul automatique et indépendant du bouton Dashboard.


## Consolidation Crédits & taxes — octobre 2026

- `frontend/src/pages/Credits.jsx` conserve bandeau/cartes/tableau, avec rapport unique, statut déclaré/inconnu, formulaires contextuels et archives à la demande.
- `frontend/src/components/CreditDialogs.jsx` : édition, paiement partiel, clôture/réouverture, retrait conservateur, historique paginé et archives ; fenêtres natives et focus clavier.
- `frontend/src/components/creditsPageModel.mjs` : validation et formatage exacts, tris, statuts backend. Aucun agrégat financier de remplacement en frontend.
- `backend/internal/api/engagement_money.go`, `engagement_models.go` : invariants et vue financière partagés. Échéancier calendaire distinct des paiements déclarés ; historique manquant = solde inconnu.
- `engagement_report.go` : instantané PostgreSQL REPEATABLE READ. `engagement_credits.go`, `engagement_payments.go`, `engagement_taxes.go`, `engagement_history.go` : propriétaire, verrou, mutation/historique atomiques, idempotence et pagination.
- Migrations 022/023 reprises sans altération ; 024 additive avec contraintes de propriétaire et conservation des preuves. Rollback 024 gardé contre toute perte de déclarations.
- Tests : `features2_test.go`, `engagement_migrations_test.go`, `creditsPageModel.test.mjs`, `frontend/tests/credits.browser.cjs` (ajouté à la CI), `backend/tests/credits_http_smoke.py` pour une instance synthétique vide isolée.
- Diagnostic, contrats, formules, validations et limites : `docs/CREDITS_TAXES_CODEX_{AUDIT,CHANGES,TESTS}.md`.


### P0 — Fidélité de l’import PDF/CSV (2026-10-25)

- `parser/poppler.go` : extraction Unicode et colonnes `pdftotext -layout`, plafonnée/contextualisée, temporaire privé ; colonnes normalisées par page dans `local.go`. Le décodeur Go reste un secours en l’absence de Poppler.
- `parser/local.go` : références numériques de marge devant deux dates, opérations multiligne, date de valeur distincte, registre des candidats et absence de correction OCR choisie pour équilibrer les soldes.
- `parser/import_report.go`, `models.ImportReport` : sommes exactes, contrôles indépendants des débits/crédits, soldes, compteur explicitement déclaré ou `not_applicable`, et traçabilité des lignes. OCR en revue ; résultat IA non confirmé non persistable. Complétude inconnue conservée sans comptage indépendant.
- Des compteurs ou totaux déclarés contradictoires bloquent l’import, y compris après recalcul du rapport avant persistance. Les opérations similaires restent des occurrences distinctes et les ressemblances intersources sont informatives ; le rejeu identique inchangé reste idempotent.
- `api/import_fidelity.go`, migration additive `026_import_fidelity.sql` : identité fichier/compte/occurrence, preuve du parsing, rapport atomique, provenance des transactions ; aucune mutation/backfill de l’historique. Rollback gardé après utilisation.
- `api/statement_controls.go`, `transactions.go` : rapports et références source accessibles au propriétaire ; un total indépendant divergent bloque la validation, même si le net équilibre les soldes. Validation historique explicitement nommée.
- `ImportReport.jsx`, `ImportPreview.jsx`, `Statements.jsx`, `TransactionDetails.jsx` : rapport consultable, contrôles non applicables et similarités informatives visibles, import suspendu si une preuve échoue ou si la période recouvre un relevé existant ; date de valeur et références de document. Aucun déploiement automatique.
- `cmd/importaudit`, `internal/importaudit` : audit historique READ ONLY, propriétaire obligatoire ; totaux/écarts, limites des preuves, aucune réparation.
- `internal/testfixture/statement.go` : PDF synthétique Type 3/ToUnicode, trois pages et occurrences légitimes identiques. Tests réels extraction→parsing→PostgreSQL isolé→API ; PDF bancaire réel optionnel via `THEMIS_REFERENCE_PDF`, jamais en Git. CI exige Poppler (`THEMIS_REQUIRE_PDF_TESTS=1`) et exécute `import-fidelity.browser.cjs`.
- Diagnostic et restrictions de réparation : `docs/IMPORT_FIDELITY_AUDIT.md`. Validations : `docs/IMPORT_FIDELITY_TESTS.md`.

### Intelligence financière — audit et normalisation des retraits (2026-10-25)

- `docs/FINANCIAL_INTELLIGENCE_AUDIT.md` cartographie normalisation/alias, règles, tags, suggestions IA, Knowledge Graph, analyses/découvertes, Dashboard, simulations et qualité des données ; `docs/FINANCIAL_INTELLIGENCE_ROADMAP.md` classe les propositions P0–P3.
- `normalize.Label` reconnaît explicitement `Ret Distrib`, `Ret DAB` et `Retrait DAB` comme `label_kind=cash_withdrawal`, libellé stable `Retrait DAB`, marchand vide. Les variantes d’heure sont retirées du nom générique et le Title Case traite les runes UTF‑8. Le brut et les transactions distinctes demeurent intacts.
- `normalize_backfill.go` version 5 met à jour uniquement les colonnes de présentation/type par son transactionnel existant ; aucune migration SQL. `recurring_review.go` regroupe ces occurrences sous un libellé fonctionnel pour la revue, sans classer les retraits en abonnement.
- `merchant_alias.go::setAlias` exécute désormais l’upsert et sa propagation dans une transaction unique, renvoie les erreurs au lieu de masquer l’échec et aligne le prédicat motif sur les clés vérifiées à l’import. La liste expose `is_pattern`, conservé par correction dans `MerchantList.jsx`. Test d’intégration synthétique : propriétaire isolé et rollback alias+transactions.
- Tests ciblés : `normalize_test.go`, `recurring_review_test.go`, `transaction_quality_integration_test.go`. Résultats de validation et limitations consignés au compte rendu de mission.


### Projets financiers — octobre 2026

- `api/projects.go`, `project_report.go`, `project_links.go`, `project_sources.go` : projets stables, budget facultatif en centimes, partitions réconciliées, confirmations uniques et sources paginées. Le rapport utilise un snapshot REPEATABLE READ et les prédicats financiers/documentaires partagés.
- Migration additive `031_financial_projects.sql` : `financial_projects`, `project_rules`, `project_transactions`, `project_history`, FK composites propriétaire, contraintes/unicité/index et protection des sources confirmées. Rollback gardé, en ordre inverse avant 027.
- `intelligence_jobs.go` réutilise le worker durable post-import pour les suggestions déterministes bornées, sans confirmation automatique ni écrasement des rejets. Aucune mutation pendant GET.
- `project_chat.go` ajoute `get_project_analysis` à l'orchestrateur existant : mêmes faits propriétaire, montants formatés côté backend, IA facultative. `export.go` inclut projets, critères et décisions ; `statement_controls.go` refuse une suppression de source encore confirmée.
- `FinancialProjects.jsx` et `FinancialProject{Forms,Links,UI}.jsx` apportent vue globale/détail, règles, confirmation/rejet par lot et association manuelle ; `projectModel.mjs` valide la réconciliation à la réception et les saisies exactes. Les anciennes vues `TagFolders` restent accessibles ; reprise par règle et validation explicite.
- Tests : `projects_integration_test.go` (PostgreSQL synthétique, deux utilisateurs, argent, périodes, imports, erreurs, suppression, export/Assistant, concurrence), `db/project_migration_test.go`, `projectModel.test.mjs`, `tests/projects.browser.cjs` (Axe/responsive) et `tests/e2e/projects.spec.ts` (API/import/auth réels).
- Audit, contrats, limites et reprise : `docs/PROJECT_ANALYSIS_{AUDIT,API,PROGRESS,REPORT}.md` ; preuve visuelle comparative dans `PROJECT_ANALYSIS_VISUAL_COMPARISON.md`.

## Fiscalité et Laboratoire (V1 bornée, 2026-10-30)

- `backend/internal/fiscal/{rules,engine}.go` : moteur pur indépendant des banques et de l’IA. Règles revenus2024/imposition2025 et revenus2025/imposition2026, rationnels exacts, frais professionnels par déclarant, quotient/plafonnement/décote/arrondis et PAS observé/projeté distincts. Années et situations hors périmètre refusées. Sources : `docs/FISCAL_RULES.md`.
- `api/fiscal.go` : GET `/api/fiscal/rules`, POST `/api/fiscal/simulate`, authentifiés, aucune persistance. `api/laboratory.go` : CRUD propriétaire `/api/laboratory[/{id}]`, création idempotente par UUID, révisions optimistes PUT/DELETE, validation et tailles bornées. Export propriétaire complété.
- Migration additive `032_laboratory.sql` : JSONB/version/titre/type/propriétaire/révision, sans données initiales ni modification des banques. Rollback refuse toute perte de document.
- `App.jsx`, `PageLoader.jsx`, `navIcons.jsx`, `api.js`, `cockpit.css` : navigation Laboratoire différée, bouton global calculatrice, mode Budget/Fiscalité dans l’en-tête de Simulation, effacement des seuls brouillons Laboratoire à la déconnexion/session expirée. Une panne réseau conserve la récupération locale.
- `FiscalSimulation.jsx` et `fiscalModel.mjs` : saisie explicite des bases fiscales, rejet des résultats incohérents, invalidation des réponses devenues obsolètes, scénario comparé, hypothèses/sources et distinction inconnus/zéros.
- `components/laboratory/LaboratoryProvider.jsx` : maximum4 documents ouverts, récupération versionnée sessionStorage par utilisateur/onglet, pas d’écriture SQL par frappe. `QuickScratchpad.jsx` : modal natif, raccourci configurable, historique, copie, conversion conservatrice. `pages/Laboratory.jsx` : gestion explicite des documents, copies, rechargement et conflits sans écrasement du brouillon.
- `lib/laboratory/workbookModel.mjs` : contrat versionné, migrations locales, limites, références relatives/absolues, presse-papiers et suppressions structurelles en fin de grille. `calculationEngine.mjs` : sélection mathjs15.2.0, BigNumber40chiffres, AST autorisé, dépendances/cycles, budget de travail. `calculationWorker.js` + hook : chargement à la demande, calcul hors UI, interruption après3secondes. Aucun eval/Function, code utilisateur ou accès externe.
- Grille native `SpreadsheetWorkspace.jsx` : formules, plages, navigation clavier, copier/coller TSV et interne, annuler/rétablir20versions, formats décimal/monétaire/pourcentage. Maximum100lignes×26colonnes,1000cellules stockées,512caractères/cellule et200ko/document.
- Tests : Go pur/API/PostgreSQL, tests Node des contrats/moteur, `tests/fiscal-laboratory.browser.cjs` (simulé ou API réelle synthétique), `tests/e2e/fiscal-laboratory.spec.ts`, chargement différé de production étendu. Notices publiques `frontend/public/licenses/laboratory/` ; architecture/limites/validation dans `docs/FISCAL_LABORATORY_*`.

### Correctif accès VM / LAN et visibilité fiscale

- `lib/laboratory/documentKey.mjs` génère des UUIDv4 avec `crypto.randomUUID` si disponible, sinon `crypto.getRandomValues` : aucune dépendance à un contexte HTTPS pour les installations HTTP par IP, aucun recours à Math.random. En l’absence d’entropie sûre, l’erreur est explicite et les brouillons restent conservés.
- Le calcul rapide ne crée plus d’identifiant pendant son rendu : génération au moment de la sauvegarde uniquement. Création/copie de feuille et conservation d’un brouillon après suppression utilisent le même helper. Une erreur reste dans le gestionnaire d’action, sans vider la racine React.
- Nouvelle entrée `Impôts` (`fiscal`) dans App/PageLoader/navIcons, accès direct au formulaire existant sans devoir découvrir le sélecteur de Simulation. API fiscale inchangée.
- `documentKey.test.mjs` vérifie les deux chemins, la version/variante RFC4122 et le refus d’une source aléatoire absente. `tests/laboratory-http-check.cjs`, appelé par le test de chargement de production, utilise une vraie origine HTTP non sécurisée avec randomUUID absent (assets locaux, API synthétique) : ordinateur/mobile, mouvements normaux, feuille/calculatrice/sauvegarde/copie/conversion et accès Impôts.
