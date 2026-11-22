# Sprint 1B — Décisions de classification persistées

Date : 22 novembre 2026. Branche locale : `feat/classification-sprint1b`.

## État et périmètre

Le prérequis 1A est satisfait : PR #68 (sprint 0), puis PR #69 (1A), intégrées dans `main`, référence de travail `048c9c37d0ac41937448652549a11d5c568db524`. La CI de référence et le workflow de sécurité ont été vérifiés avant reprise. Le précédent rapport de préparation bloquée a été sauvegardé hors du dépôt avant son remplacement.

Validation locale achevée. Le propriétaire a explicitement autorisé la publication et la fusion dans main le 22 novembre 2026 ; la fusion doit attendre les contrôles CI et sécurité de la PR. Aucune connexion à une base de production, aucun traitement de données bancaires réelles. La migration et son inverse ont été exécutés uniquement dans des schémas synthétiques isolés, sur PostgreSQL 16 local. Les autres worktrees ont été préservés. Le sprint 1C n'a pas commencé.

Périmètre : révisions, verrous indépendants, décisions de tags, rejets de propositions, journal, intégration des chemins d'écriture et panneau avancé Transactions. Aucun changement de nature économique, montants, dates, libellés, calculs de remboursement, crédits, patrimoine, agrégats ou prévisions.

## Architecture réellement implémentée

Une autorité partagée combine `backend/internal/classification/service.go` et les primitives PostgreSQL de `035_classification_decisions.sql`. Les projections courantes déterminent les protections ; les événements ne sont pas rejoués pour décider des écritures. Ce n'est pas un système d'event sourcing.

La commande manuelle ou Agent suit ce chemin :

1. Validation du contrat et du propriétaire ; verrou sur la transaction bancaire concernée.
2. Recherche d'un reçu d'opération déjà validée, avant la précondition de révision.
3. Vérification de la révision attendue ou de l'empreinte compatible 1A.
4. Application explicite des domaines/tags et des décisions de protection.
5. Incrément de révision et journal par les primitives communes, dans la même transaction SQL.
6. Écriture du reçu puis commit. Tout échec annule l'ensemble.

Les triggers couvrent aussi les écritures d'import, de règles et de dictionnaire : un oubli dans un handler ne suffit pas à désactiver ces protections. Les opérations automatiques gardent leur plan métier propre ; `Automatic` leur attribue une origine et un identifiant commun de lot. Les appels réseau IA restent hors transaction et hors verrou bancaire.

### Fichiers principaux

| Responsabilité | Fichiers et fonctions |
|---|---|
| Autorité, versions, contexte, commandes | `internal/classification/service.go` : `Read`, `Begin`, `Finish`, `Automatic`, `Rule`, `Locks`, `Context`, `Reject`, `Reevaluate` |
| Invariants et persistance partagée | Migration 035 : `classification_begin`, `classification_record`, `classification_snapshot`, `classification_set_locks`, `classification_tag_decide`, triggers transactions/liens/dictionnaire/règles |
| PATCH et opérations de lot UI | `api/transactions.go::updateTransaction`, `replaceTransactionTags`; `pages/Transactions.jsx::guardedSave` |
| Règles individuelles/globales/par relevé | `api/transaction_rules.go::applyClassificationRules`, `categorize/plan.go::Plan` |
| Import classique/documentaire | `api/handlers.go::persist`, `api/import_fidelity.go::persistDocument` |
| Agent | `api/agent_workspace.go::agentSnapshots`, `agentActionDecision` |
| Propositions IA | `ai/enrich.go::AutoEnrich`, identité canonique inchangée `ai/suggestion_identity.go::SuggestionIdentity` |
| Dictionnaires | `api/tags_manage.go::renameTag`, `deleteTag`; `api/features2.go::deleteDomain` |
| Lecture cohérente et confidentialité | `api/transactions.go::writeTransactionReport`, `api/classification_history.go::classificationHistory`, `api/export.go::exportAll` |
| UX | `components/ClassificationAdvanced.jsx`, `components/TransactionDetails.jsx`, `pages/Transactions.jsx`, `api.js` |

Les chemins abrégés Go désignent `backend/internal/...`, les chemins UI `frontend/src/...`.

## Schéma additif 035

| Table | Rôle et clés |
|---|---|
| `classification_state` | PK propriétaire/transaction ; révision monotone, génération de contexte et deux verrous |
| `classification_tag_decisions` | PK propriétaire/transaction/tag stable ; décision `added` ou `removed`, révision, origine, date |
| `classification_rejections` | PK propriétaire/transaction/identité/contexte ; preuve durable de rejet |
| `classification_events` | Événements ordonnés, unicité propriétaire/transaction/révision ; ancien/nouvel état, origine/acteur, règle/tag/suggestion, motif et opération |
| `classification_operations` | Reçus idempotents par propriétaire/transaction/opération ; empreinte de requête et révision obtenue, y compris sur no-op |
| `classification_rule_versions` | Version sémantique par règle/propriétaire ; dépendances de contexte sans invalidation globale |

Trois colonnes sont ajoutées au cache `transaction_suggestions` : `identity`, `context_key`, `base_revision`. La valeur vide identifie un ancien cache sans contexte durable ; aucune identité ni provenance historique n'est fabriquée.

Les états, décisions, reçus et rejets ont des FK composites vers une transaction du même propriétaire. Les décisions de tags référencent un tag du même propriétaire. Les index composites tags/règles reprennent les identités existantes. La migration ne met à jour aucune transaction et ne reconstruit aucun événement historique.

Le journal conserve ses événements après suppression légitime d'un relevé ou d'une transaction : il référence alors seulement l'ancien ID. Il ne possède donc pas de FK bloquante vers la transaction. La suppression du propriétaire efface ses événements par FK vers `users` ; c'est l'exception explicite à l'immutabilité pour l'effacement du compte. Les autres projections suivent la suppression de la transaction. Les tests de suppression existants restent valides.

## Révisions et concurrence

La révision logique initiale vaut **0** sans ligne d'état. Une première mutation effective crée l'état. Une transaction nouvellement importée avec un domaine affecté peut recevoir immédiatement un événement d'affectation initiale ; les lignes déjà présentes lors de la migration n'en reçoivent aucun.

Toute mutation effective de domaine/source, lien de tag, décision de tag, verrou, rejet nouveau ou demande de réévaluation incrémente la révision. Un renommage de tag touche également les transactions qui le portent ou conservent une décision sur son ID. Les mises à jour strictement identiques et les réapplications sans delta ne l'incrémentent pas.

**Arbitrage :** une commande HTTP peut produire plusieurs événements et plusieurs incréments (par exemple lien de tag puis décision explicite). La garantie est une révision strictement croissante par delta atomique, pas `+1` exactement par clic. Tous ces incréments sont validés ensemble. Un no-op de classification reste à la même révision ; son reçu technique peut être créé.

Le verrou parent sérialise les corrections. Une ancienne révision ne redevient pas valide après A → B → A. L'Agent inclut la révision dans son empreinte d'aperçu, donc une modification de protection invalide aussi son aperçu.

Un même `Idempotency-Key` et la même requête rejouent le succès sans réappliquer l'écriture. Réutiliser la clé avec une autre requête renvoie 409. Le reçu est vérifié avant la précondition devenue ancienne après le premier succès. L'UI conserve la clé après un échec et la renouvelle quand la requête change. Les opérations automatiques restent idempotentes par déduplication d'import et absence de delta.

Les lots de règles et l'Agent verrouillent les transactions dans l'ordre croissant des IDs. Les renommages verrouillent leurs transactions concernées dans le même ordre. Un conflit ou deadlock SQL annule la transaction ; aucune relance silencieuse de commande manuelle n'est ajoutée.

## Verrous et priorité manuelle

`domain_locked` et `tags_locked` sont indépendants de `classification_source`. Une correction explicite manuelle/Agent peut modifier une classification verrouillée. Les automatismes ne peuvent pas changer un domaine verrouillé ni les liens de tags verrouillés.

Déverrouiller ne change ni le domaine, ni sa source manuelle, ni les décisions de tags. Le NULL manuel conserve sa priorité. Les règles ne remplacent toujours pas un domaine existant ; seules les propositions de la règle gagnante restent éligibles. Un verrou de domaine, à lui seul, n'est pas un verrou de tags. La règle conserve néanmoins les protections métier 1A lorsque le domaine est manuellement NULL ou différent de celui proposé.

Le diagnostic de cohérence lit ces protections et les retraits tracés, afin de ne pas annoncer comme applicables des tags bloqués. Le diagnostic CLI d'audit historique n'est pas une commande de mutation et ne remplace pas l'autorité transactionnelle.

## Tombstones et dictionnaires

Un retrait explicite d'un tag connu crée ou actualise une décision `removed` par **ID**, même si le lien était déjà absent. L'insertion automatique de ce tag est bloquée. Un ajout explicite autorisé enregistre `added`, ce qui lève ce retrait. Aucun tag n'est créé uniquement pour retirer un nom encore inconnu : aucune identité stable n'existe dans ce cas.

Le remplacement manuel des tags utilise des deltas, sans supprimer/réinsérer les liens inchangés. Un tag simultanément présent dans `add_tags` et `remove_tags` est refusé en 400.

Le renommage conserve l'ID et donc le retrait. Un renommage vers un nom déjà utilisé renvoie **409**, sans fusion implicite. La suppression d'un tag lié ou référencé par une décision est refusée ; un tag réellement inutilisé reste supprimable. La suppression d'un domaine utilisé par une transaction ou ses enfants immédiats est également refusée. Ces restrictions protègent les preuves existantes ; une éventuelle fusion/archivage exige un contrat ultérieur et n'est pas simulée ici.

## Rejets IA : identité et contexte

L'identité canonique v1 de 1A est réutilisée sans modification de calcul : propriétaire, transaction, domaine proposé, tags normalisés/dédupliqués/triés. Fournisseur, modèle, heure et relance sont exclus.

Le contexte est une empreinte distincte : version de politique 1, génération pertinente de la transaction et dépendances des seules règles qui correspondent à son libellé. Pour ces règles, ID, version, domaine, motif et tags canoniques sont pris en compte. Une règle sans rapport, un modèle différent ou un import sans effet sur cette transaction ne réinitialise pas ses rejets. Les variations sans effet de casse d'un motif littéral et d'ordre/doublons des tags ne créent pas de version de règle supplémentaire.

Un rejet incrémente la révision pour invalider les résultats tardifs mais **pas** la génération de contexte : il ne s'invalide pas lui-même. Une correction pertinente, un renommage concerné ou une demande explicite de réévaluation fait évoluer le contexte. Les preuves de rejets précédentes restent stockées.

`AutoEnrich` capture révision/contexte avant l'appel modèle. Au retour, il verrouille la transaction, revalide propriétaire, révision, contexte et rejet puis stocke uniquement une proposition. Il n'affecte ni domaine ni tags. Le rapport Transactions revalide aussi les propositions présentées dans son snapshot de lecture : une ancienne suggestion ne reste pas affichée après changement de révision ou de règles pertinentes.

La réévaluation explicite est journalisée, invalide le cache et peut cibler une transaction déjà classée. Elle ne déverrouille rien et n'applique pas le résultat. L'indisponibilité du fournisseur reste visible. Le refus d'une action Agent reste distinct du rejet d'une proposition de classification : aucune écriture de `classification_rejections` n'est ajoutée à ce refus.

## Journal et confidentialité

Chaque delta écrit état, révision et événement dans la même transaction. Une panne du journal annule la mutation. Une panne de mutation ne laisse pas d'événement de succès. Les UPDATE/DELETE ordinaires du journal sont refusés ; une correction crée un nouvel événement.

Le journal contient des IDs, source, verrous, décisions de tags, versions et motifs courts. Il ne copie ni montant, date bancaire, libellé brut, nom de marchand, texte libre LLM, prompt ni secret. L'identité de suggestion est un hash. L'acteur manuel/Agent correspond au propriétaire authentifié ; un automatisme n'invente pas un acteur humain.

L'endpoint authentifié `GET /api/transactions/{id}/classification/history` vérifie la propriété, lit état/événements dans un snapshot cohérent, pagine 50 événements avec `before_id`, et interdit le cache HTTP. L'export ajoute les états, décisions, rejets et événements filtrés par propriétaire.

L'immutabilité protège les chemins applicatifs et les écritures ordinaires. Elle n'est pas une garantie cryptographique contre un administrateur PostgreSQL capable de modifier les fonctions/triggers. Les accès DB privilégiés restent une responsabilité d'exploitation.

## Compatibilité API et UX

Le contrat GET historique en tableau est conservé. Le rapport ajoute `classification_revision` en chaîne décimale (précision bigint préservée), les deux verrous et `classification_configured`. L'empreinte `classification_version` reste fournie et inclut désormais la révision.

PATCH accepte `expected_revision` (entier JSON ou chaîne décimale), `expected_classification_version`, `domain_locked`, `tags_locked`, `reevaluate`, `expected_suggestion_identity` et les champs précédents. L'ancienne empreinte purement calculée n'est acceptée qu'à révision 0, sans état durable. Après activation d'une décision, elle ne contourne plus les révisions. Un ancien client sans précondition peut toujours faire une correction explicite ; il respecte les protections communes mais ne bénéficie pas d'une détection de brouillon périmé. Un ancien client muni d'une empreinte périmée reçoit 409.

Le panneau « Classification avancée » est replié par défaut. Il sépare verrous, réévaluation et historique, sans écran d'administration supplémentaire. Le formulaire simple Domaine + Tags reste en place. Le contrôle Axe supplémentaire du détail ouvert en thème clair a révélé deux textes de remboursement insuffisamment contrastés ; leur couleur a été corrigée localement dans `pages/Transactions.css`, sans changement du fonctionnement des remboursements. Les erreurs n'effacent pas le brouillon. Le bouton de rechargement actualise la transaction par ID même si la correction la fait sortir des filtres de liste. Aucune écriture n'est automatiquement répétée après un conflit.

## Déploiement et rollback fonctionnel

Analyse d'impact avant toute production :

- La migration est additive, mais installe des triggers sur des tables partagées et ajoute du travail à chaque delta de classification. Mesurer latence, volume de journal, durée des lots et contention avec un jeu synthétique représentatif avant activation de production.
- Les suppressions de domaines/tags utilisés et les fusions implicites deviennent des refus explicites. Prévenir les utilisateurs de ce changement de dictionnaire.
- Les anciennes suggestions sans contexte ne sont conservées à l'affichage que pour un état encore non configuré. Aucun rejet historique n'est inventé.
- Sauvegarde/restauration et export doivent inclure les nouvelles tables. Aucune purge de preuves ni durée de rétention automatique n'est introduite.

Ordre proposé : sauvegarde vérifiée → arrêt des anciens writers pendant installation → migration 035 → déploiement du backend compatible → UI → contrôles synthétiques de classification, agrégats et permissions. Aucun de ces gestes n'a été effectué en production.

Le rollback **fonctionnel** conserve tables, révisions et journal, et retire les nouvelles entrées UX/automatismes en gardant les primitives de compatibilité. Revenir aveuglément à un backend 1A sur le schéma 035 n'est pas un rollback validé : les anciens writers ne posent pas tous l'origine explicite attendue. Garder une version de compatibilité et les protections actives, ou restaurer une sauvegarde complète dans un environnement de reprise après décision d'exploitation.

L'inverse `db/rollbacks/035_classification_decisions.sql` est réservé à une installation inutilisée : il refuse toute ligne d'état, décision, journal, rejet, reçu, version de règle ou cache enrichi. Il ne supprime pas les index composites hérités. Les tests d'inverses 031/027 annulent désormais d'abord leur dépendance 035, sans contourner ses gardes.

## Validation

Les fichiers de tests spécifiques sont `api/classification_decisions_integration_test.go`, `api/classification_concurrency_test.go`, `db/classification_migration_test.go`, et `frontend/tests/transactions.browser.cjs`. Les anciens tests à tables temporaires chargent désormais les primitives de production dans un schéma isolé via `internal/testfixture/postgres.go` ; les tests sur schéma complet vérifient aussi les FK et les cascades. Aucun accès réel n'est utilisé et aucun contrôle de production n'est désactivé.

| Vérification locale | Résultat |
|---|---|
| `go test ./...` avec PostgreSQL synthétique | Réussi sur l'ensemble des packages lors de la passe complète |
| Scénarios 1B dédiés | Révisions/ABA, rejeu/no-op, propriétaire, verrous indépendants, tombstone/renommage/réajout, journal atomique/immuable, rejet durable/contexte/réévaluation/résultat tardif, transition legacy, Agent périmé/confirmation/refus distinct : réussis |
| Migration synthétique 034 → 035, réapplication, inverse gardé | Réussi ; sources historiques strictement inchangées, aucun backfill, propriétaire et suppression légitime vérifiés |
| `go vet ./...` | Réussi |
| `go test -race ./...` avec PostgreSQL | Réussi sur tous les packages, PostgreSQL synthétique inclus |
| `npm run build` | Réussi |
| Transactions navigateur Chromium | Réussi : révisions, clé idempotente, conflit sans relance, brouillon conservé, verrous/historique/réévaluation ; six tailles et Axe clair/sombre |
| Agent navigateur Chromium | Réussi : aperçus, validation explicite, action périmée, six tailles/thèmes, Axe plein écran/flottant |
| Axe panneau avancé ouvert à six tailles et en clair | Réussi : panneau ouvert aux six tailles, puis en thème clair ; aucune violation Axe |
| CI distante du sprint 1B | À vérifier sur la PR avant fusion ; résultats et SHA consultables dans GitHub |

Les contrôles de navigateur interceptent les API avec des données synthétiques. Ils ne remplacent pas les scénarios PostgreSQL qui testent les mutations réellement exécutées. La suite HTTP bout en bout complète avec fournisseur IA réel et KG n'a pas été exécutée : ni nouveau fournisseur ni modification du KG ne fait partie de ce sprint.

## Limites et décisions restantes

- Les choix antérieurs à l'activation restent de provenance inconnue : aucun verrou, tombstone ou rejet n'est inféré des données existantes.
- Les champs de classement manuels sans précondition restent acceptés pour compatibilité ; seuls les nouveaux clients ou ceux fournissant une empreinte bénéficient du conflit de brouillon.
- Le contexte ne prétend pas modéliser toutes les évolutions futures du KG, d'une politique économique ou d'un modèle personnalisé. Une telle dépendance devra être explicitement versionnée avant de servir aux décisions ; un changement de modèle seul n'est volontairement pas une invalidation.
- La fusion/archivage de tags protégés, les migrations de taxonomie, la nature économique et les allocations financières sont hors périmètre.
- Un journal retenu après suppression de transaction est consultable dans l'export propriétaire ; l'endpoint de détail d'une transaction disparue renvoie 404.
- Les snapshots du journal sont bornés à 64 Kio chacun ; un état exceptionnel dépassant cette limite provoque un échec atomique, jamais une écriture sans journal.
- Les performances à grande volumétrie et le plan de rétention doivent être validés avant production. Pas de preuve de charge représentative ni de garantie contre l'administration DB privilégiée.

Publication et fusion autorisées par le propriétaire le 22 novembre 2026. La migration, les restrictions de dictionnaire, le contrat legacy et la granularité des révisions sont documentés pour revue. La fusion passe par une PR après réussite de la CI et des contrôles de sécurité ; aucun déploiement manuel ni accès à la base de production n'est prévu par cette publication.
