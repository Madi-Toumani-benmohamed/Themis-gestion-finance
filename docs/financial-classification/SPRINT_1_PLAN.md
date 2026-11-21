# Sprint 1 proposé — contrats et protection de la classification

21 novembre 2026. **Plan uniquement ; le sprint 1 n’est pas commencé.** Direction B additive envisagée, pas autorisation de migration. Les six documents du sprint 0 se complètent : [concepts et décisions](FINANCIAL_CONCEPTS.md), [mesures facultatives](CLASSIFICATION_METRICS.md), [taxonomie](CLASSIFICATION_TAXONOMY.md), [vingt cas synthétiques](FINANCIAL_TEST_CASES.md), [matrice et chemin complet](AGGREGATION_MATRIX.md), présent plan. L’[audit initial](../CLASSIFICATION_AUDIT.md) reste inchangé.

## Objectif et exclusions

Rendre les décisions actuelles prévisibles et vérifiables avant toute qualification économique nouvelle. Préserver l’historique et l’interface Domaine + Tags. Ne pas inclure une refonte de portefeuille, une reclassification générale, une reprise du KG, des allocations multi-domaines ou un nouveau moteur fiscal. Aucune suppression de domaine/tag/règle.

Le sprint 0 a ajouté seulement ces documents et un diagnostic lecteur indépendant. Il n’a modifié ni matcher de production, ni politique de tags, ni API, modèle SQL, migration ou worker. Les propositions suivantes exigent une validation métier puis une tâche d’implémentation distincte.

## Lot 1.1 — Contrat commun des flux et des preuves

**Problème.** Les mêmes noms income/expense peuvent décrire des flux ordinaires plutôt que revenu/coût économiques. `financialsql` centralise déjà une exclusion importante ; le remplacer sans mesurer les consommateurs créerait des régressions. Certains endpoints legacy sont en flottants/ignorent des erreurs. Référence : matrice d’agrégation et S08/S13/S15/S19.

**Fichiers/dépendances.** `backend/internal/financialsql/predicates.go`, `api/transfer.go`, `transactions.go`, `budgets.go`, `analysis_report.go`, `simulation.go`, `handlers.go::summary`, `pulse.go`, `remaining.go`, `agent_tools.go`/`agent_read_scope.go`, `ai/analyze.go` ; consommateurs frontend et sources/export. Compte, propriétaire, périodes, couverture documentaire et conventions centimes sont des dépendances, pas un simple filtre SQL.

**Solution minimale proposée.** Documenter puis tester la politique existante par une interface/contrat commun explicable, sans étendre les exclusions par nouveaux mots-clés. Auditer les requêtes réellement consommatrices et remplacer seulement les duplications démontrées. Distinguer flux bruts, flux ordinaires et qualification économique future. Préserver les champs API existants pendant la transition ; toute extension d’étiquette/statut sera additive et validée. Traiter erreurs et inconnus explicitement avant une nouvelle métrique économique.

**Tests.** Matrice même propriétaire/période/source sur dépense/crédit ordinaire, transfert parent/enfant, savings, montant nul, catégorie étrangère, mois partiel, erreur SQL. Comparer listes d’IDs incluses et sommes exactes ; vérifier sources budget/export/Assistant et références de simulation sur trois mois. Tests de non-régression synthétiques existants Transactions/Budgets/Analyse/Simulation, puis E2E concernés.

**Acceptation.** Aucun mouvement ni classement historique modifié ; résultat legacy expliqué et stable tant que politique non changée ; aucune fuite propriétaire ou conversion inconnue→zéro ; les écarts sémantiques restants sont nommés, pas masqués. Une fois approuvée, toute différence de résultat est liée à un cas de référence et visible dans l’aperçu.

**Risque/retour arrière.** Élevé par nombre de consommateurs ; petits lots, comparaison parallèle en lecture, maintien de l’ancienne lecture derrière drapeau. Pas de backfill nécessaire à la centralisation.

## Lot 1.2 — Traçabilité des décisions

**Problème.** La source courante ne dit pas quelle règle a gagné, d’où vient un tag, qui a remplacé une décision, ni pourquoi une suggestion a été rejetée. Une suggestion acceptée via PATCH devient manual ; les historiques spécialisés ne couvrent pas toutes les décisions.

**Fichiers/dépendances.** `api/transactions.go::updateTransaction`, `transaction_rules.go`, `handlers.go` (import/règles), `admin.go`, `ai/enrich.go`, `agent_workspace.go`, modèle d’export et tests propriétaires. Tables actuelles : transactions, tags/liens, rules, transaction_suggestions, agent_actions. Un éventuel journal durable demandera un schéma additif séparément autorisé ; il n’est pas créé au sprint 0.

**Solution minimale proposée.** Définir un événement métier versionné avec propriétaire, acteur/origine (manuel/règle/suggestion acceptée/Agent), identifiant source, précédent/nouvel état, raison et date. Tags : origine par décision, pas forcément un flag définitif sur un nom de tag. Journal atomique avec mutation et idempotence, sans recopier le brut bancaire dans les logs. Conserver `classification_source` pour compatibilité. Avant approbation du stockage durable, ne pas prétendre rendre l’historique traçable par de simples logs texte ou traces temporaires.

**Tests.** Rollback mutation+journal sur erreur, décisions répétées idempotentes, concurrence manuelle/règle, isolation propriétaire, acceptation/refus IA distincts, cohérence Agent/PATCH et export. Pas de valeurs bancaires dans logs.

**Acceptation.** Chaque nouvelle décision approuvée se relie à sa source ; absence de journal antérieur explicitement unknown, jamais un passé reconstruit fictif. Zéro classement réécrit pour remplir le journal. Événements conservés lors d’une correction compensatrice.

**Risque/retour arrière.** Moyen à élevé : atomicité et confidentialité. Déploiement du lecteur/journal sous drapeau ; conserver les événements en désactivant la fonctionnalité plutôt que les supprimer. Migration additive éventuelle = décision à part avant code.

## Lot 1.3 — Choix manuels, y compris l’abstention

**Problème.** Domaine manuel non nul protégé, choix manuel NULL réattribuable. Les règles peuvent ajouter des tags à une correction manuelle. Un domaine non nul ne dit pas à lui seul si l’utilisateur souhaite figer la décision.

**Fichiers/dépendances.** `transactions.go`, `transaction_rules.go`, `admin.go::applyAllRulesForStatement`, `ai/enrich.go`, `agent_workspace.go`, `frontend/src/components/TransactionDetails.jsx`, opérations par lot de Transactions. Dépend de 1.2 pour expliquer/protéger les décisions.

**Solution minimale proposée.** Définir une décision explicite « garder non qualifié » et un périmètre de protection (domaine et/ou tags). À court terme, une règle peut proposer, mais ne doit pas écraser un choix protégé, même NULL. Ne pas utiliser définitivement le seul `classification_source=manual` comme verrou générique : des corrections de tags ne changent pas cette source. Proposer les divergences en aperçu, ne pas reclasser automatiquement les lignes existantes. Politique de refus durable des suggestions à décider avec l’utilisateur.

**Tests.** Manuel A vs règle B, manuel NULL vs règle, tags retirés puis réapplication, reprise explicite autorisée, proposition IA tardive après correction, mise à jour en concurrence, actions Agent périmées, lot partiellement échoué. Vérifier montant/date/brut identiques.

**Acceptation.** Zéro écrasement de décision protégée, aucune réintroduction silencieuse d’un tag retiré dans le périmètre protégé, proposition expliquée ; choix volontaire distinct d’un historique simplement incomplet. Pas d’inférence de choix volontaire pour tous anciens NULL.

**Risque/retour arrière.** Moyen : moins de remplissage automatique attendu mais correct. Activation progressive, retrait du mécanisme futur sans perte de décisions ; l’inactivation ne doit pas autoriser une réapplication aveugle de tout l’historique.

## Lot 1.4 — Tags de règles conflictuelles

**Problème.** Première règle choisit le domaine, toutes les règles ajoutent leurs tags, ce qui mélange par exemple repas/trajet. Un tag identique ajouté deux fois n’est pas un doublon de lien ; le problème est sémantique, pas un double montant SQL.

**Fichiers/dépendances.** `categorize/categorize.go`, `transaction_rules.go`, `admin.go`, `handlers.go` (création/édition/appliquer), UI de gestion des règles existante, diagnostic lecteur. Dépend des protections manuelles et du journal.

**Solution minimale proposée.** Faire calculer un plan de décision commun domaine+tags avant application. Politique recommandée à valider : tags de la règle gagnante par défaut ; cumul seulement lorsqu’une règle est explicitement déclarée compatible/additive dans un futur contrat. Une règle appliquée seule doit préciser son périmètre et ne pas contourner les protections. Conserver la priorité actuelle au début ; ne pas introduire simultanément un nouveau score de spécificité. Afficher différence de tags proposés, jamais nettoyer automatiquement les anciens tags dont l’origine est inconnue.

**Tests.** UBER EATS/UBER, regex vs littéral, égalités de longueur/ID et Unicode, règle sans tags, tags identiques, règles de mêmes domaines et domaines différents, invalidité en milieu de boucle, import nouveau relevé vs application globale, changement de domaine manuel. Vérifier que la nouvelle politique s’applique de façon identique à création/édition/application/import.

**Acceptation.** Tags déterministes expliqués dans aperçu ; aucun tag de règle perdante ajouté sans compatibilité explicite ; tags manuels/historiques préservés ; pas d’application aux anciens relevés lors d’un nouvel import. Échec atomique sur le périmètre annoncé, ou résultat partiel explicitement présenté si ce contrat est conservé.

**Risque/retour arrière.** Élevé sur attentes utilisateurs existantes de cumul ; comparer effets avant activation et valider la politique. Revenir à l’ancienne politique pour les nouvelles décisions seulement ; ne pas supprimer massivement les tags déjà écrits lors d’un rollback.

## Lot 1.5 — Cohérence et matcher identiques

**Problème.** `coherence.go` utilise ILIKE et ignore les regex alors que `CompilePattern` les exécute en Go et traite %/_ littéralement. L’audit peut donner une contradiction qui n’est pas celle du moteur.

**Fichiers/dépendances.** `categorize/categorize.go`, `api/coherence.go`, `handlers.go::listRules`, tests et outil de diagnostic du sprint 0. Borne de travail nécessaire pour un audit interactif ; propriétaires, secrets et non-envoi IA par défaut.

**Solution minimale proposée.** Réutiliser la compilation et l’évaluation réelle, avec ordre identique et résultats structurés de correspondance/conflit. Ne pas réécrire les regex en SQL. Une règle invalide est un état explicite cohérent entre chargement, aperçu et application ; définir le comportement d’échec par lot. L’audit reste lecteur ; une explication IA éventuelle reçoit seulement le périmètre autorisé, pas un export implicite de données bancaires.

**Tests.** Regex ancrée, caractères Unicode, préfixe `re:` exact, `%/_`, regex invalide, égalité de longueur/ID ; parité entre matcher, aperçu et audit, limite de volume sans résultats partiels présentés comme complets. Aucun appel réseau dans tests unitaires.

**Acceptation.** Une même entrée produit les mêmes matches/ordre partout ; aucune contradiction fondée sur une approximation SQL. Limites et couverture annoncées, pas de correction automatique depuis l’audit.

**Risque/retour arrière.** Moyen : signaux de cohérence visibles peuvent changer. Versionner l’explication et préserver l’accès à la preuve ; aucun rollback de transaction nécessaire à un audit lecteur.

## Lot 1.6 — Préparer les qualifications économiques sans les appliquer

**Problème.** Enum unique insuffisant pour crédit/mixte ; tags/domaines ne sont pas preuves de transfert, revenu ou coût. Les garanties de sources doivent précéder toute bascule.

**Fichiers/dépendances.** Référentiel/corpus/matrice, contrats d’agrégation ; services existants de remboursements/engagements/projets à inventorier, sans les remplacer. Futurs schémas à concevoir seulement après décision humaine.

**Solution minimale proposée.** Définir des contrats métier de preuve/portion/qualification versionnée, un état non qualifié, dates bancaire/économique et liens propriétaires. Produire une spécification et des fonctions pures de validation de portions sur fixtures, sans nouvel enum SQL ni qualification historique. Remboursements existants réutilisés, pas deuxième registre de crédits. Le modèle de portefeuille complet reste hors périmètre.

**Tests futurs.** S06–S19, conservation des centimes, acquis/non acquis, capital/résultat inconnu, dette/intérêts, révocation d’allocation et transfert à une jambe. Pour toute portion inconnue : refus d’une ventilation inventée, pas solde fictif.

**Acceptation.** Spécification compatible avec l’historique et des IDs stables ; aucune nouvelle nature déduite seulement du nom de domaine/tag. Aucune modification KG ou règle générale de reprise. Une validation séparée est requise avant ajout durable.

**Risque/retour arrière.** Faible si limité aux contrats/test pures ; élevé si élargi aux écritures. Préparer l’activation par lecteur avec flag, compatibilité historique et rollback sans destruction.

## Ordre de livraison et porte de validation

Valider les définitions et politiques avant 1.1 ; réaliser 1.1 et spécifier 1.2 ; autoriser séparément le stockage durable éventuel ; protéger le manuel avant changer les tags ; harmoniser audit/matcher ; finaliser contrats économiques. Éviter un déploiement qui change simultanément normaliseur, taxonomie et agrégations.

Avant publication future : sauvegarde/restauration testée sur environnement isolé, baseline de sommes et identités, tests unitaires/intégration synthétiques puis parcours navigateur concernés, aperçu d’impact propriétaire, validation d’activation. Critères non négociables : intégrité des preuves, aucun double compte, propriétaires isolés, centimes réconciliés et inconnus visibles. Ce sprint ne lance aucun de ces déploiements ni migrations.

## Validation du sprint 0

Commandes exécutées dans le checkout indiqué, avec Go 1.26.11 et cache de build local sous `/workspace/work` :

```sh
go test -race ./internal/classificationaudit ./cmd/classificationaudit ./internal/categorize
go vet ./internal/classificationaudit ./cmd/classificationaudit
go build -o /tmp/themis-classificationaudit-sprint0 ./cmd/classificationaudit
```

Résultats finaux : tests de diagnostic et matcher réussis ; commande compile, vet sans diagnostic. Tests entièrement synthétiques, faux driver local vérifiant READ ONLY, REPEATABLE READ, requêtes paramétrées et rollback ; données simulées scannées avec tableaux PostgreSQL, calculées et réconciliées. Aucune base ni connexion réseau nécessaire aux tests. Un premier `go fmt` avait échoué sur le cache par défaut en lecture seule ; formatage réalisé ensuite avec `gofmt` et cache explicite, sans changement des droits système.

Contrôles documentaires : six livrables présents, vingt cas de sources fictives, liens locaux vérifiés, vocabulaire et conventions brut/économique recoupés ; fichiers ajoutés limités aux docs demandées et à l’outil/tests. Contrôle CLI sans DATABASE_URL avec propriétaire invalide : refus attendu, code 2, aucun accès base. L’audit préexistant reste non suivi et inchangé. Aucun fichier applicatif existant modifié, y compris graphe de code, schéma, dépendances ou frontend.

Non exécutés : outil sur PostgreSQL réel/synthétique, tests d’intégration DB, suite Go complète, build frontend et E2E applicatifs (aucun changement des fonctionnalités) ; tests du futur modèle économique (non implémenté), mesures utilisateurs et inspection KG. Les résultats du faux driver ne prouvent ni droits/TLS ni schéma d’une installation réelle.

État Git attendu à remise : branche existante `feat/agent-natural-dialogue` conservée, nouveaux fichiers non commités, aucun push/merge. L’audit existant est un livrable de la tâche précédente ; ne pas l’écraser ni l’inclure implicitement dans une publication. Aucun conflit introduit avec les autres worktrees ; aucune reclassification, réimport, backfill ou écriture base/KG.

## Décisions à valider avant de commencer le sprint 1

- Noms et sens des métriques visibles (crédits observés vs revenu économique), et conservation des anciennes conventions pendant transition.
- Protection explicite du choix manuel NULL, portée des verrous sur tags et mémoire des rejets IA.
- Tags de règle gagnante seuls par défaut ou cumul explicite ; traitement des invalidités et atomicité des lots.
- Besoin et contrat de stockage additif des décisions, rétention et export ; aucune migration n’est autorisée par ce plan seul.
- Imputation des remboursements au mois d’origine/à date, articulation avec trésorerie réelle ; principal de dette, intérêts, espèces et transferts supposés.
- Rôle de Famille/Vie professionnelle, taxonomie fonctionnelle et revue de correspondances ; aucun déplacement automatique.
- Autorisation locale distincte pour exécuter les métriques réelles et modalités de conservation privée. Les nombres de domaines/tags utilisateurs demeurent **non mesurés**.

Fin du sprint 0 après remise des livrables. Ne pas enchaîner l’implémentation du sprint 1 sans nouvelle instruction.
