# Relevés — tests et vérifications

Exécutés le 25 octobre 2026, branche `feat/statements-documentary-hardening`, worktree courant. Données et comptes exclusivement synthétiques, schémas PostgreSQL temporaires/isolés. Aucune migration ou modification sur les données réelles ; aucun fournisseur IA appelé.

## Résultats

| Vérification | Résultat constaté |
|---|---|
| Go 1.23 : `go test ./...` | Réussi sur tous les packages avec PostgreSQL de test |
| `go vet ./...` | Réussi |
| `CGO_ENABLED=1 go test -race ./...` | Réussi, aucun accès concurrent signalé |
| Derniers contrôles ciblés Relevés/export après ajout du mode à l’export | Réussis ; `go vet ./...` réussi |
| `npm ci` | Réussi, 0 vulnérabilité rapportée par npm |
| `node --test src/components/*.test.mjs` | 58 réussis, 0 échec, 0 ignoré |
| `npm run build` | Réussi ; avertissement existant de chunk > 500 kB conservé |
| Playwright Relevés | Réussi : parcours, erreurs, filtres, revue, suppression, provenance, clavier, 5 largeurs et Axe clair/sombre |
| Playwright Transactions | Réussi, y compris drawer/focus et édition multiple |
| Playwright Budgets | Réussi, 6 largeurs |
| Playwright Simulation | Réussi, 6 largeurs, 0 violation Axe |
| Playwright Analyse | Réussi, navigation/périodes/graphiques/IA facultative et Axe |
| Playwright Crédits & taxes | Réussi, paiements/historique/erreurs et 5 largeurs |
| Playwright Configuration IA | Réussi, tous les fournisseurs, brouillons, sauvegarde/diagnostics et mémoire |
| Semgrep `p/security-audit --error` | Réussi, 0 résultat bloquant ; scan des sources nouvelles effectué avec `--no-git-ignore` sur backend/React/tests/configurations |
| Trivy fs, vulnérabilités + configurations HIGH/CRITICAL, `--ignore-unfixed` | Réussi, code 0 |
| Deux configurations Compose (`--quiet`, sans démarrage) | Réussies |
| Image `themis-statements:review` via `backend/Dockerfile` | Build réussi, aucun déploiement |
| `git diff --check` et empreinte `AGENTS.md` préexistant | Réussis |

Les commandes Go utilisent le conteneur d’outillage `themis-backend-test` et le PostgreSQL 16 dédié `themis-ai-config-test-db`, jamais la base réelle. Les jeux d’intégration utilisent des tables temporaires ou des schémas nommés uniques supprimés en fin de test. Les tests de migration construisent leur propre schéma, y compris les migrations historiques.

Le premier téléchargement Trivy a échoué sur la limite du tmpfs `/tmp`. Il a été relancé avec un volume de cache Docker dédié et a réussi ; aucun contrôle n’a été désactivé ou assoupli pour contourner cet échec d’environnement.

## Scénarios backend ajoutés

Complément **Actions et couleurs**, vérifié après retour utilisateur :

- `statement_recheck_test.go` : relance déterministe sans appel IA, mode combiné consultatif, absence d’écriture financière/de validation, contrôle de session et propriétaire avant appel fournisseur, JSON strict, délai borné, absence de données bancaires/contexte/identifiants dans le prompt, indisponibilité/timeout/réponse inventée, conservation des échecs et priorité du rouge même en présence d’un chevauchement, confirmation manuelle unique d’un résultat automatique.
- Payload IA limité à 20 constats prioritaires ; les échecs comptables restent affichés même si le modèle choisit des constats positifs.
- Tests Node conservés : **58 réussis**. Tests navigateur Relevés enrichis : couleurs calculées distinctes pour vert/orange/rouge dans les deux thèmes, filtre des échecs, actions accessibles en haut, relance sans IA et combinée, panne de fournisseur/requête, validation toujours bloquée après avis IA en présence d’un écart.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, build frontend et Semgrep réexécutés avec succès sur ce complément. Les six autres suites navigateur, Trivy et le build Docker du tableau correspondent au lot précédent ; ils n’ont pas été répétés pour cette seule évolution ciblée. Aucun test avec fournisseur réel, aucun nouveau déploiement, commit ou merge.

`api/statement_controls_test.go` :

- Un ou plusieurs comptes soumis au contrôle documentaire, comptes manuels et ancien pointage sans dégradation artificielle.
- Modification du mode persistée, refus d’accès croisé ; absence de compte requis non assimilée à une preuve de complétude.
- Relevés couvrant deux mois autour du 21, périodes partielles et sources non connues.
- Écart exact d’un centime, soldes inconnus et fin de période manquante ; aucune valeur absente remplacée par zéro de contrôle.
- Continuité correcte, borne partagée sans faux chevauchement, chevauchement suspect, interruption et rupture comptable malgré des relevés individuellement équilibrés.
- Validation/réouverture avec justification, refus d’une transition répétée, refus d’un contrôle incohérent et isolation par propriétaire.
- Historique atomique en cas d’erreur d’insertion ou d’erreur SQL tardive après écriture : ni statut ni preuve partielle conservés.
- Suppression avec cascade, historique conservé par identifiant de ressource, cache propriétaire invalidé, données de l’autre utilisateur conservées et rattachement étranger bloqué.
- Import partiellement dédupliqué : contrôle du parsing non conservé comme preuve des écritures stockées. Relevé sans opération conservé.
- Rapport des transactions filtré par relevé sans mois implicite, revenu exact, provenance ne faisant pas confiance à un ancien flag incohérent, source étrangère inaccessible.
- Export contenant le mode manuel sans divulgation d’autres ressources utilisateur.

`api/import_preview_test.go` : égalité exacte `0,10 + 0,20 = 0,30`, signe de l’écart d’un centime, doublons et absence de solde conservés.

`db/statement_migrations_test.go` : reprise du schéma 024, attribution selon le type et non le nom, montants/flags historiques inchangés, migration répétable via le runner, rollback avant utilisation, refus atomique après personnalisation du mode ou ajout d’une revue. Les tests de migration existants vérifient aussi une installation neuve avec 025.

Les anciennes fixtures analytiques déclarant des relevés documentés reçoivent des soldes synthétiques réellement cohérents et des modes explicites. Les cas modifiant les observations ajustent leurs données sources de test en conséquence ; les assertions sur les montants et les fenêtres financières ne sont pas affaiblies. Les tests nouveaux vérifient précisément qu’un écart non corrigé fait échouer la preuve documentaire.

## Scénarios frontend

`statementPageModel.test.mjs` et `dataReliabilityModel.test.mjs` : centimes reçus réconciliables, sommes crédit/débit cohérentes, valeurs inconnues explicites, dénominateur sans comptes manuels, synthèse, filtrage par période bancaire/compte/statut/recherche/identifiants.

`tests/statements.browser.cjs` intercepte toutes les API : lecture échouée/reprise et chargement, filtres et sélection de mois, compteur documentaire, contrôles non exécutés, validation refusée puis réussie, historique visible, accès aux opérations par provenance sans mois imposé, suppression annulée/échouée/réussie, état vide. Test clavier, focus contenu/restauré par le gestionnaire natif réutilisé, absence de débordement global et du détail, tableau exploitant la largeur desktop, cinq largeurs (1920/960/768/390/320), Axe sombre/clair. Les captures locales sont dans `/tmp/themis-statements-ui` et ne contiennent que les fixtures.

Les six autres parcours navigateur vérifient les consommateurs partagés et le gestionnaire de dialogue extrait de Transactions. Les scans Axe du nouveau parcours ciblent la page Relevés et son dialogue ; ils ne constituent pas un audit d’accessibilité de toute la page Comptes ou de toute l’application.

## CI et reproduction

Les tests Go et migrations seront exécutés par le job PostgreSQL existant, les tests Node par le glob existant. Le nouveau parcours Relevés est ajouté à la suite navigateur CI sans retrait ni assouplissement des six parcours précédents. Les outils Playwright/Axe suivent la même installation temporaire que les autres tests.

```bash
# Avec une base PostgreSQL dédiée et THEMIS_TEST_DATABASE_URL défini
cd backend
go test ./...
go test -race ./...
go vet ./...

cd ../frontend
npm ci
node --test src/components/*.test.mjs
npm run build
# Vite et l’outillage navigateur isolé doivent être disponibles
NODE_PATH=/tmp/themis-browser/node_modules THEMIS_UI_URL=http://127.0.0.1:5182 \
  node tests/statements.browser.cjs
```

**Aucun résultat GitHub CI n’est revendiqué pour cette branche locale non publiée.** La réussite CI/Security de la PR #32 concerne la configuration IA précédemment fusionnée, pas ces modifications Relevés. La CI et une revue du diff devront être vérifiées sur une future PR avant merge. La migration doit être déployée avec le backend correspondant, pas sur une base réelle pour faire passer un test.

## Limites de validation

Pas de relevé bancaire réel, pas de test sur production, pas de contact avec une banque ou un LLM. La complétude intrinsèque d’un PDF ne peut pas être démontrée sans contrôles/source supplémentaires absents du modèle existant. Pas de restauration des transactions supprimées ni de purge de mémoire KG effectuée. La compatibilité financière repose sur les contrôles exacts, les tests d’intégration et les non-régressions synthétiques ; elle ne vaut pas certification d’exhaustivité documentaire. Voir `STATEMENTS_CHANGES.md` pour les limitations restantes.

## Reprise — fidélité des imports, 25 octobre 2026

Cette reprise conserve le chantier préexistant et ajoute le rejet des compteurs/totaux indépendants contradictoires. Les répétitions identiques sont acceptées. Les tests couvrent aussi le recalcul avant persistance ; le cas de rejet SQL vérifie maintenant le motif précis du rapport, sans pouvoir réussir accidentellement sur un chevauchement de période.

| Vérification réexécutée | Résultat |
|---|---|
| `CGO_ENABLED=1 THEMIS_REQUIRE_PDF_TESTS=1 go test -race ./...` avec PostgreSQL dédié | Réussi sur tous les packages, dont migrations, PDF synthétique de 81 opérations, persistance et API |
| `go vet ./...` | Réussi |
| `go build -o /tmp/themis-server ./cmd/server` dans le conteneur | Réussi |
| `node --test src/components/*.test.mjs` | Réussi, 13 fichiers de tests, aucun échec |
| `npm run build` | Réussi ; avertissement de chunk > 500 kB toujours présent |
| `tests/import-fidelity.browser.cjs` | Réussi : extraction bloquée, similarité, totaux indépendants, compteur inconnu, 81 références sources, confirmation, responsive et Axe |
| `tests/statements.browser.cjs` | Réussi : contrôles, validation, erreurs, suppression, provenance, clavier, cinq largeurs et Axe clair/sombre |
| `tests/transactions.browser.cjs` | Réussi : filtres, montants, pagination, provenance, édition, erreurs, six largeurs et Axe clair/sombre |
| `actionlint .github/workflows/ci.yml` via l’image 1.7.12 | Réussi |
| `git diff --check` | Réussi |

Le premier lancement Go a échoué faute de compilateur C dans l’image d’outillage. L’installation temporaire de `gcc`, `musl-dev` et `poppler-utils` dans le conteneur a permis de réexécuter la commande complète avec succès, sans désactiver les tests PDF ni la détection de concurrence. Les restrictions locales de lancement de Vite et Chromium ont été résolues par les autorisations d’exécution correspondantes.

Cette reprise utilise exclusivement les fixtures synthétiques et `themis-ai-config-test-db`. Les tests privés optionnels `THEMIS_REFERENCE_PDF` ne sont pas activés ; aucun accès aux relevés ou à la base réelle. Les autres parcours navigateur, les scans de sécurité et le build d’image Docker n’ont pas été répétés pour ce complément ciblé. Aucun résultat GitHub CI revendiqué, aucun commit, push, déploiement ou réparation historique. Les migrations 025/026 restent à déployer avec leur backend ; les validations de migration ci-dessus portent seulement sur les schémas isolés.
