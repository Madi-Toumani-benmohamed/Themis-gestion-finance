# Tests Playwright E2E

Les tests navigateur existants `frontend/tests/*.browser.cjs` restent des tests UI sur Vite avec API simulée. Ils sont exécutés séparément des nouveaux tests `frontend/tests/e2e/*.spec.ts`, qui passent par le backend Go et PostgreSQL.

## Lancement local reproductible

Prérequis : Docker Compose, Go 1.23+, Node 22+, et Poppler (`pdftotext`). Depuis `frontend` :

```bash
npm ci
npm run test:e2e:local
```

Le script recrée le conteneur PostgreSQL de test dédié `themis_e2e`, puis Playwright démarre le backend et Vite. La base est neuve à chaque lancement et supprimée à la fin. Il ne faut jamais remplacer l'URL de cette base par une URL personnelle ou de production.

Pour travailler avec une instance E2E déjà démarrée, définir `THEMIS_E2E_DATABASE_URL` vers une base **vide** nommée `themis_e2e`, installer Chromium (`npx playwright install chromium`), puis lancer `npm run test:e2e`.

Rapports : `npm run test:e2e:report`. Les erreurs conservent une trace, une capture et un rapport JUnit sous `test-results`; le rapport HTML est créé dans `playwright-report`.

## Fixtures et sécurité

La préparation utilise deux utilisateurs synthétiques et `backend/internal/testfixture.Statement009PDF`; aucun compte, mot de passe, relevé ou service distant réel n'est utilisé. Les routes normales d'authentification et d'import sont exercées. Les migrations sont appliquées au démarrage habituel du serveur. Le CLI `backend/cmd/e2efixtures` génère le PDF de test et ne modifie une session qu'après avoir confirmé que `current_database()` est `themis_e2e`.

Les fichiers d'état d'authentification, la base de test, les rapports et le relevé généré sont ignorés par Git.
