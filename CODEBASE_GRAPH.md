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
| `transactions.go` | Filtres partagés, rapport exact paginé, édition atomique et provenance | `GET /api/transactions[?view=report]`, `PATCH /api/transactions/{id}` |
| `transaction_rules.go` | Validation des règles et application atomique, périmètre d'import | Services des règles existantes |
| `chat.go` | Assistant conversationnel | `POST /api/chat` (orchestration outils) |
| `pulse.go` | Synthèse rapide (mois demandé) | `GET /api/pulse?month=YYYY-MM` |
| `hero.go` | Conclusion analyse (ton + points) | `GET /api/hero?month=YYYY-MM` |
| `anomalies.go` | Détection signaux (abos, unusual) | `GET /api/anomalies?month=YYYY-MM` |
| `discoveries.go` | Découvertes vagues 2 (duplicatas, drift) | `GET /api/intel/discoveries` |
| `journal.go` | Fil chronologique unifié | `GET /api/journal?month=YYYY-MM` |
| `remaining.go` | Reste-à-vivre (proj. ou hist.) | `GET /api/remaining?month=YYYY-MM` |
| `features.go` | Budgets, catégories, règles | `/api/budgets*`, `/api/rules*`, `/api/categories` |
| `features2.go` | Tags, récurrences, exports | `/api/tags*`, `/api/recurring`, `/api/export` |
| `features3.go` | Analytics avancées | `/api/analytics/*`, `/api/intel/*` |
| `narrative.go` | Synthèse narrative IA | `GET /api/intel/narrative` |
| `coherence.go` | Vérif cohérence données | `GET /api/coherence` |
| `merchant_alias.go` | Renommage marchands | `/api/merchants/*` |
| `admin.go` | Config IA persistée + stats | `/api/admin/*` |
| `period.go` | Infos période + comparaison | `GET /api/period`, `GET /api/period/compare` |

### Modèles (`backend/internal/models/models.go`)
```go
Transaction { ID, Date, Label, Display, Merchant, Amount, DomainID, Tags, Hash }
Statement   { ID, Filename, PeriodStart/End, Balances, Parser, Validated }
Category    { ID, Name, Color, IsTransfer }
Rule        { ID, Pattern, DomainID }
```

### Config (`backend/internal/config/config.go`)
- `Load()` lit `.env` + defaults
- Timeouts IA par opération
- **Nouveau** : `MemoryServiceURL`, `MemoryEnabled`

### Memory Client (`backend/internal/memory/memory.go`)
```go
Client{Learn, Query, SearchFacts, BuildFinancialFacts}
```
- Appelé par `chat.go` (enrichit prompt) et `handlers.go` (apprend à l'import)

### Migrations (`backend/internal/db/migrations/`)
- `001_init.sql` → `021_transaction_classification.sql` (appliquées 1x via `schema_migrations`, rollbacks explicites hors du dossier embarqué)

---

## Frontend — Structure

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
| `top` | `TopWidget` | `api.transactions()` + `api.categories()` |
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
AI_PROVIDER=ollama|anthropic|mistral|freellm|none
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
| `CategoriesWidget`, `KpiWidget`, `TopWidget` | Filtrent `data` selon `selectedMonth` |
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
- `/api/hero` expose les calculs existants `income`, `expenses`, `net` et `period_complete`. `hero_diagnosis.go` vérifie une couverture continue par relevés validés pour tous les comptes connus, indépendamment du statut approximatif de `period.go`. En absence de preuve : bilan incomplet.
- Diagnostic provisoire indépendant du signe des flux et sans hypothèse de salaire. Alertes réelles conservées ; comparaisons limitées aux trois mois précédents documentés ; signaux récurrents limités au mois de leur dernière opération.
- Tests : `hero_diagnosis_test.go` (cycle de salaire, alertes, intervalles de relevés), tests Node de présentation.

### Dépendances frontend — sécurité (2026-10-23)

- Vite 8.3.4 et `@vitejs/plugin-react` 6.1.2 : mise à niveau conjointe pour respecter les peer dependencies. Build avec Node.js 22 (Docker et CI).
- Correctifs PostCSS, Browserslist et baseline-browser-mapping intégrés via Dependabot.

### Fournisseur OpenAI — Admin

- `backend/internal/llm/openai.go` : client Responses HTTP (`store: false`), sorties texte/JSON, détection des réponses incomplètes et erreurs sans corps sensible. Aucun SDK ajouté.
- `admin.go` : configuration persistée `openai_model` / `openai_key`, clé masquée (`openai_key_set`), conservation de la clé lors du test et de la sauvegarde avec champ vide, repli environnement.
- `config.go` + configurations Compose : `OPENAI_API_KEY` / `OPENAI_MODEL`. Modèle par défaut `gpt-4.1-mini`, modifiable.
- `frontend/src/pages/Admin.jsx` : choix OpenAI, champs modèle/clé et test existant. La mémoire Graphiti garde son fournisseur Ollama séparé.
- Tests `llm/openai_test.go`, `api/admin_openai_test.go`, `config/openai_test.go` : contrat Responses simulé, erreurs/refus/troncature, persistance, masquage et sélection du fournisseur.

### CI — consommation GitHub Actions

- `.github/workflows/ci.yml` exécute les tests Node `src/components/*.test.mjs` avant le build frontend.
- Le contrôle syntaxique Python (`compileall`) ne nécessite pas l’installation de Graphiti et de ses dépendances ; il ne valide pas les imports ni le comportement du service.
- Le build de l’image applicative attend backend, frontend, mémoire et Compose. Buildx utilise le cache GitHub Actions `themis-app` en mode `min`, sans publication d’image ; une erreur d’export de cache ne masque pas une erreur de build.

### Centre de fiabilité des données

- `GET /api/reliability?month=YYYY-MM` (`reliability.go`) : couverture par compte, intervalles sans preuve documentaire, relevés non validés ou de période inconnue, dernière transaction sur la période et dernier import toutes périodes. Requêtes limitées au propriétaire authentifié. Aucun montant bancaire exposé.
- Le mois courant est examiné du 1er à aujourd’hui ; un mois historique est examiné jusqu’à sa fin. Pas de mois futur. Les périodes inconnues sont signalées sans leur attribuer arbitrairement un mois.
- `uncoveredStatementDates` est partagé avec `heroSpansCover` : continuité des relevés validés, y compris cycles clôturés le 21. Pas de nouveau calcul financier ou de migration.
- `DataReliability.jsx` dans Relevés : choix du mois, détails par compte, bouton pour filtrer les relevés concernés et accès à l’import existant. Lien depuis le pied de FinancialSummary transmettant le mois du dashboard.
- Tests Go des intervalles/statuts et test PostgreSQL optionnel `TestReliabilityPostgres` (`THEMIS_TEST_DATABASE_URL`, tables temporaires), tests Node `dataReliabilityModel.test.mjs`.

### Prévisualisation des imports

- `POST /api/statements/previews` réutilise le parsing PDF/CSV, sans persister de relevé ou de transaction. `import_preview.go` calcule les doublons selon les empreintes existantes et décrit les soldes/écarts disponibles.
- Aperçus en mémoire du `Server`, liés au propriétaire et au compte : TTL 15 min, 3 par utilisateur, 32 au total et budget estimé de 32 Mio ; limite de 20 000 opérations par aperçu. Suppression sur expiration par timer, annulation ou confirmation.
- Confirmation `POST /api/statements/previews/{token}/confirm` : consommation atomique du jeton, revérification du compte, puis `finishStatementImport`, commun à l’import direct historique. Déduplication définitive par la contrainte SQL existante.
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
- Import : refus d'empreintes ambiguës dans un fichier et entre comptes, sans réécrire les empreintes historiques. Verrou transactionnel par propriétaire ; la déduplication historique ne distingue toujours pas toutes les opérations entre fichiers.
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
