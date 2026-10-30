# Reprise — analyse financière par projet

- Branche : `feat/project-financial-analysis`.
- Départ / HEAD : `7a28d795930b1f34201f04e5a071247c7949811a`.
- État initial propre. Publication de la branche et fusion via PR autorisées explicitement par l’utilisateur après revue du résultat local. Vérifier les contrôles avant fusion ; leur état final est porté par GitHub.
- Architecture : projet stable distinct, confirmations uniques comme source de vérité, règles bornées proposant uniquement, historique et snapshots. Tables 031 additives ; aucun montant bancaire ni dossier historique migré automatiquement.
- Décisions et contrats : `PROJECT_ANALYSIS_AUDIT.md`, `PROJECT_ANALYSIS_API.md`.

## Lots

0. Audit / sécurisation : terminé, constats historiques revérifiés, propriété et intégrité testées.
1. Modèle / budget : terminé, migration additive, centimes et réconciliation.
2. Associations / suggestions : terminé, batch atomique, correction/retrait, règles déterministes et worker durable existant post-import.
3. Observations : terminé dans le périmètre documenté (dépassement et concentration sourcés, couverture prudente, outil IA de lecture facultatif). Aucune tendance non comparable ou remboursement inféré.
4. UX : terminé, global/détail, formulaires, sources et historique, tableaux et graphiques accessibles, erreurs/vides.
5. Validation locale : terminée avec la réserve visuelle décrite ci-dessous ; CI distante et image Docker non exécutées.

## Tests réellement exécutés

- Go `test -race ./...` : réussi avec PostgreSQL synthétique ; API/DB revalidées après les dernières corrections de migration, sources sans catégorie et cascade de suppression utilisateur.
- `go vet ./...` : réussi.
- Modèles Node : 16 fichiers réussis.
- Les 13 suites navigateur de la chaîne ont chacune réussi : Transactions, Budgets, Simulation, Analyse, Projets, Crédits, Configuration IA, Relevés, Import, Motion, Cinématique, Intelligence et chargement de production.
- Projets : responsive et Axe clair/sombre à 1440/768/390/320 px ; association manuelle, batch, correction/retrait, budget absent, sources, historique, erreurs.
- E2E réels : 9 scénarios fonctionnels réussis, dont le nouveau projet avec vraie API, authentification, import PDF synthétique et propriétaire étranger.
- Références visuelles : 4 échecs locaux reproduits sur main de départ. Les 4 captures refonte/main sont identiques octet pour octet (0 pixel différent), voir `PROJECT_ANALYSIS_VISUAL_COMPARISON.md`. Aucun seuil ni référence modifié. Ne pas les déclarer verts localement.
- Build frontend, test des chunks de production, 2 tests Python, compilation Knowledge Graph, Compose config et diff check : réussis.
- Rapport complet : `PROJECT_ANALYSIS_REPORT.md`.

## Environnement et limitations

Les E2E ont tourné dans une copie isolée du code (`/workspace/work/projects-e2e-run`) à cause des permissions de rapports préexistants dans le dépôt. Seuls le cache Vite et l'autorisation locale des modules/polices ont été adaptés dans la copie. Baseline main indépendante dans `/workspace/work/projects-e2e-baseline`.

Pas d'allocation partielle, coût net, causalité, prévision ou confirmation automatique ; pas de notification hors application. Le LLM réel n'a pas été appelé, seul l'adaptateur déterministe propriétaire est testé. Les containers PostgreSQL de test et le serveur Vite de cette mission sont arrêtés en fin de travail. Les rapports préexistants sont préservés.

## Fichiers modifiés ou ajoutés

- `CODEBASE_GRAPH.md`
- `backend/internal/api/chat.go`
- `backend/internal/api/export.go`
- `backend/internal/api/features2_test.go`
- `backend/internal/api/handlers.go`
- `backend/internal/api/intelligence_jobs.go`
- `backend/internal/api/project_chat.go`
- `backend/internal/api/project_links.go`
- `backend/internal/api/project_report.go`
- `backend/internal/api/project_sources.go`
- `backend/internal/api/projects.go`
- `backend/internal/api/projects_integration_test.go`
- `backend/internal/api/statement_controls.go`
- `backend/internal/api/tag_folders.go`
- `backend/internal/db/migrations/031_financial_projects.sql`
- `backend/internal/db/project_migration_test.go`
- `backend/internal/db/reimbursement_migration_test.go`
- `backend/internal/db/rollbacks/031_financial_projects.sql`
- `docs/PROJECT_ANALYSIS_API.md`
- `docs/PROJECT_ANALYSIS_AUDIT.md`
- `docs/PROJECT_ANALYSIS_PROGRESS.md`
- `docs/PROJECT_ANALYSIS_REPORT.md`
- `frontend/package.json`
- `frontend/src/api.js`
- `frontend/src/components/FinancialProjectForms.jsx`
- `frontend/src/components/FinancialProjectLinks.jsx`
- `frontend/src/components/FinancialProjectUI.jsx`
- `frontend/src/components/FinancialProjects.jsx`
- `frontend/src/components/financialProjects.css`
- `frontend/src/components/projectModel.mjs`
- `frontend/src/components/projectModel.test.mjs`
- `frontend/src/pages/TagFolders.jsx`
- `frontend/tests/analysis.browser.cjs`
- `frontend/tests/e2e/projects.spec.ts`
- `frontend/tests/projects.browser.cjs`

- `docs/PROJECT_ANALYSIS_VISUAL_COMPARISON.md`

## Prochaine action précise

Revoir le patch local et le rapport. Après instruction explicite de publication uniquement : créer le commit, pousser `feat/project-financial-analysis`, ouvrir une PR vers main et surveiller la CI (notamment le navigateur de référence et l'image Docker). Ne pas fusionner automatiquement. Ne pas recréer la refonte ni recalculer les références visuelles pour masquer l'écart local.
