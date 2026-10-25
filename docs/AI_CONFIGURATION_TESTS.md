# Configuration IA — validations

Exécutées le 25 octobre 2026 sur `feat/ai-configuration-hardening`, dans le worktree courant. Tous les montants, comptes, clés et paramètres de fournisseurs présents dans les tests sont synthétiques. Aucun service IA réel ni donnée bancaire réelle utilisés.

## Résultats locaux

| Validation | Résultat réel |
|---|---|
| Backend Go 1.23 : `go test ./...` | Réussi, tous les packages ; PostgreSQL de test fourni |
| `go vet ./...` | Réussi |
| `CGO_ENABLED=1 go test -race ./...` | Réussi, aucun accès concurrent signalé |
| Frontend : `node --test src/components/*.test.mjs` | 54 tests réussis, 0 échec, 0 ignoré |
| Frontend : `npm run build` | Réussi ; avertissement de chunk > 500 kB conservé |
| Playwright configuration IA | Réussi ; les 4 panneaux, 5 largeurs (1920, 960, 768, 390, 320), clavier et Axe clair/sombre |
| Playwright Transactions | Réussi ; parcours, montants exacts, filtres, drawer, édition multiple et 6 largeurs |
| Playwright Budgets | Réussi ; 6 largeurs |
| Playwright Simulation | Réussi ; 6 largeurs, aucune violation Axe |
| Playwright Analyse | Réussi ; navigation, périodes, sources, IA facultative, graphiques et 6 dispositions |
| Playwright Crédits & taxes | Réussi ; paiements, historique, erreurs, clavier, 5 largeurs et Axe |
| `docker compose config --quiet` | Réussi |
| Variante Supabase : `docker compose -f compose-supabase/docker-compose.yml config --quiet` | Réussi syntaxiquement ; variables SUPABASE_* absentes de cet environnement, avertissements explicites |
| `docker build -f backend/Dockerfile -t themis-ai-configuration:review .` | Réussi, frontend + binaire + image finale |
| `git diff --check` | Réussi |

Go est exécuté dans le conteneur d’outillage `themis-backend-test`. Les intégrations utilisent exclusivement le PostgreSQL 16 isolé `themis-ai-config-test-db` et la variable `THEMIS_TEST_DATABASE_URL`. Le détecteur de races utilise les compilateurs C installés dans le conteneur temporaire, sans dépendance supplémentaire dans le projet. Le build Docker ne remplace pas l’image ni le service de production.

Les tests navigateur interceptent toutes les API avec des fixtures ; aucune requête à un fournisseur ou à la base réelle. L’outillage Playwright/Axe est installé dans `/tmp/themis-browser`, conformément au mécanisme CI existant. Les captures sont des jeux synthétiques locaux, sans données financières réelles.

## Couverture ajoutée

`backend/internal/api/ai_configuration_test.go` :

- Six choix de fournisseur et conservation des clés, du contexte et des options lors d’un PUT partiel.
- Persistance/rechargement, clés d’environnement, retrait explicite, erreur SQL sans changement de configuration active ni fuite de secret.
- Configuration globale non masquable par une ligne utilisateur ; UPSERT réellement exécuté sur PostgreSQL isolé, ligne utilisateur conservée.
- Authentification, rôle administrateur et refus des cinq routes IA durant le bootstrap ; réglage `ai_config` réservé à l’API administrative.
- JSON invalide, valeurs nulles/types incorrects, modèle absent/invalide, URL interdite, contexte Unicode, clé manquante, conflit de révision.
- Test sans persistance, association du résultat au brouillon puis à la configuration enregistrée, absence de messages externes sensibles, deadline.
- Contrôles concurrents terminés dans l’ordre inverse, découverte bornée et filtrée, refus d’un HTTP 500 et des métadonnées réseau.
- Mémoire désactivée, disponible et non configurée.

`backend/internal/llm/network_test.go` : schémas/ports/identifiants/query/fragment invalides, IP privées et IPv6, métadonnées, autorisation exacte de base, DNS mixte, IP vérifiée, origine verrouillée, absence de proxy/redirection, erreurs fournisseurs sans secrets et maintien des longs délais de génération.

`backend/internal/memory/health_test.go` : aucun appel si désactivée, santé JSON réelle, slash final, HTTP 200 HTML refusé, redirection refusée, annulation.

`backend/internal/ai/context_test.go` : impossibilité de fermer le bloc déclaratif avec des balises de contexte/rôle. Tests OpenAI existants conservés ; le stockage simulé peut maintenant injecter une erreur SQL.

`aiConfigurationModel.test.mjs` : six tests sur les fournisseurs, brouillons, distinction connexion/comportement, payload de test sans contexte, révision/retrait des clés, validation et absence de métrique fictive.

`frontend/tests/ai-configuration.browser.cjs` : erreur de lecture/rechargement, aucun contrôle automatique, champs conditionnels, modèle saisi conservé après découverte, test échoué sans changement de fournisseur actif, sauvegarde échouée sans perte de clé, effacement après succès, contexte conservé pendant maintenance, annulation, réponse de test devenue obsolète, retrait explicite, diagnostic/mémoire, conflit HTTP 409 et rechargement volontaire, mémoire désactivée, désactivation IA répercutée dans la navigation. Vérifications responsive et Axe limitées au composant IA pour ne pas prétendre auditer toute l’administration.

## Reproduction

```bash
# PostgreSQL exclusivement dédié aux tests, DSN via THEMIS_TEST_DATABASE_URL
cd backend
go test ./...
go vet ./...
go test -race ./...

cd ../frontend
node --test src/components/*.test.mjs
npm run build
# Vite et outillage Playwright/Axe démarrés selon le job CI existant
NODE_PATH=/tmp/themis-browser/node_modules THEMIS_UI_URL=http://127.0.0.1:5182 \
  node tests/ai-configuration.browser.cjs
```

La CI exécute les tests Go avec PostgreSQL et `-race`, `go vet`, les tests Node/build et désormais le parcours navigateur IA en plus des cinq parcours financiers. Les contrôles sécurité existants restent inchangés. Leurs résultats distants seront contrôlés sur le commit publié avant toute fusion ; les résultats locaux ci-dessus ne valent pas affirmation de réussite des scans GitHub.

## Limites

Aucun test réel avec clé payante, modèle Ollama distant ou graphe réel. Pas de garantie de disponibilité future, ni de test d’inférence Ollama par le bouton de connectivité. Pas de test distribué multi-instance ou de coffre de secrets ajouté. Aucun test d’acceptation manuel avec technologie d’assistance ; les parcours clavier et Axe sont automatisés. Voir `AI_CONFIGURATION_AUDIT.md` pour les risques résiduels.
