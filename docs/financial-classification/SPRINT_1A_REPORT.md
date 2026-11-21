# Sprint 1A — Fiabilisation du moteur de classification

> Publication autorisée séparément le 21 novembre 2026 : les états Git et validations locales ci-dessous relatent la remise initiale. Le sprint 0 est publié dans la PR #68 ; 1A est préparé comme PR dépendante, sans fusion automatique. L’addendum final décrit cette étape de publication.

Date : 21 novembre 2026. État : implémentation locale et validation ; **lot D partiellement satisfait**, limites de persistance explicites. Aucun sprint 1B/1C commencé, aucune publication Git ni migration nouvelle.

## 1. Base, périmètre et préservation

Consignes et livrables du sprint 0 relus : `AGENTS.md`, `CODEBASE_GRAPH.md`, [audit](../CLASSIFICATION_AUDIT.md), [concepts](FINANCIAL_CONCEPTS.md), [métriques](CLASSIFICATION_METRICS.md), [taxonomie](CLASSIFICATION_TAXONOMY.md), [corpus](FINANCIAL_TEST_CASES.md), [matrice](AGGREGATION_MATRIX.md), [plan](SPRINT_1_PLAN.md). Les décisions validées dans la mission 1A remplacent les options encore ouvertes de ces documents historiques : tags gagnants seulement et convention d’imputation des remboursements notamment.

Travail isolé dans `/workspace/work/classification-sprint1a`, branche `feat/classification-sprint1a`, base `d1bdec7a91d2719ccf67f755a8fef68dd46f0190` (référence locale origin/main). Le checkout `feat/agent-natural-dialogue` reste sur `3d944f5`, avec ses fichiers non suivis du sprint 0 intacts. Audit, six documents et outil du sprint 0 ont été copiés dans le worktree sans commit intermédiaire. Seul le diagnostic copié est adapté au moteur commun ; l’original du sprint 0 n’est pas écrasé. Aucun fetch, commit, staging, push, PR ou merge dans ce sprint.

Les changements touchent moteur, application des règles, audit de cohérence, suggestions IA, protections existantes de Transactions/Agent et interfaces existantes. Aucun changement de nomenclature, modèle SQL, nature économique, données réelles, remboursements, budgets, simulations ou formules d’agrégats. Aucun backfill, réimport, recalcul persistant sur installation réelle ni accès au KG. Les seuls accès DB sont des tests synthétiques dans un conteneur PostgreSQL dédié, avec tables temporaires/schémas éphémères. Leurs fixtures utilisent les migrations existantes pour créer les schémas isolés ; aucune migration nouvelle ni migration d’installation utilisateur n’est effectuée.

## 2. Architecture modifiée

| Élément | Responsabilité et évolution |
|---|---|
| `backend/internal/categorize/categorize.go` | `LoadFrom` charge les règles dont domaine et propriétaire concordent ; `Load` conserve son interface pour l’import. `CompilePattern` reste le matcher effectif, sans changement de sémantique. |
| `backend/internal/categorize/plan.go` | `New` ordonne/compile ; `Evaluate` distingue gagnante, correspondantes et invalides ; `Plan` prépare domaine/tags proposés, conflits, protections connues et opérations autorisées. Pas de montant ni copie du brut dans le plan retourné. |
| `api/transaction_rules.go` | Évaluation globale des règles propriétaires, sélection éventuelle d’une règle persistée, lecture/verrou des transactions et préparation avant application atomique. Les tags sont ajoutés sans suppression. |
| `api/admin.go`, `handlers.go` | Application globale, création/édition avec application individuelle passent par ce même service. GET règles conserve le tableau existant, ajoute comptage des matches gagnants actuels et état de motif invalide. |
| `api/coherence.go` | Snapshot lecteur, matcher et plan Go, règles gagnantes uniquement pour le diagnostic d’application ; choix manuels explicitement conservés, regex invalides visibles. Sources par ID, bornes et erreur au lieu de résultat tronqué présenté comme complet. |
| `internal/classificationaudit/audit.go` | Le diagnostic lecteur du sprint 0 utilise aussi `Engine.Evaluate`. Il conserve les métriques de chevauchement et d’apports potentiels des règles perdantes : ces dernières décrivent un risque/ancien cumul possible, pas des tags désormais effectivement appliqués. |
| `api/transactions.go`, `classification_version.go` | Précondition d’état de classification dérivée et vérifiée sous verrou ; réponse 409 en cas de divergence, avant mutation. Aucun stockage de révision ajouté. |
| `api/agent_workspace.go` | La provenance courante entre dans le snapshot d’une action préparée. Un changement de décision manuel, même sans changement de domaine, rend l’aperçu périmé. |
| `ai/enrich.go` | Les NULL manuels sont exclus des candidats et du recontrôle après appel modèle ; l’IA ne modifie toujours pas domaines/tags effectifs. |
| `ai/suggestion_identity.go` | Contrat pur d’identité d’une suggestion, testé mais non relié à une persistance de rejet. Il ne fournit pas une déduplication durable déjà opérationnelle. |

Les chemins abrégés `api/...` et `ai/...` désignent `backend/internal/api/...` et `backend/internal/ai/...` ; le diagnostic est dans `backend/internal/classificationaudit`.

## 3. Politique finale des règles et comportements visibles

1. **Priorité inchangée :** longueur en caractères Unicode décroissante, puis ID croissant. Pas de score de spécificité ajouté. Une regex plus longue peut donc encore être prioritaire sans être plus spécifique métier.
2. **Même matcher :** sous-chaîne littérale insensible à la casse ; `%` et `_` littéraux ; `re:` compile une regex Go insensible à la casse, y compris les ancrages. Aucune conversion vers regex SQL ni ILIKE dans l’audit déterministe.
3. **Une gagnante seulement :** son domaine et ses tags composent la proposition. Les autres matches expliquent le conflit mais ne produisent pas leurs tags, même si leur domaine est identique. Gagnante sans tags = aucun tag automatique ajouté.
4. **Domaine conservateur :** une attribution non nulle n’est pas réécrite par une règle. Un NULL provenant de `manual` reste NULL. Un ancien NULL `unknown` reste éligible : aucun refus volontaire n’est inventé.
5. **Tags autorisés :** ajouts de la gagnante si le domaine courant est absent et automatisable, ou s’il correspond au domaine gagnant. Un domaine courant différent empêche aussi l’ajout de tags incompatibles avec cette proposition. Un NULL manuel ne reçoit pas les tags d’une règle dont il refuse le domaine. Une correction manuelle avec le même domaine n’interdit pas tous les ajouts de tags.
6. **Préservation :** aucun tag existant n’est supprimé, manuel ou historique. Un ancien tag VTC inconnu reste donc présent ; il n’est simplement plus ajouté par la règle perdante UBER face à UBER EATS. Modifier une règle n’efface ni domaines déjà affectés ni tags anciens.
7. **Application individuelle :** une règle sélectionnée n’agit que là où elle gagne contre l’ensemble des règles propriétaires. La sélectionner ne lui donne pas une priorité spéciale. Créer/modifier une règle avec `apply=true` utilise son ID persisté, pas une règle ad hoc échappant au jeu courant.
8. **Invalidité :** les motifs invalides sont signalés dans GET règles et l’audit, et ignorés dans l’évaluation comme au chargement initial historique. L’application globale ne s’interrompt plus sur une regex invalide au milieu de sa boucle ; les erreurs SQL ou tags invalides font en revanche annuler toute l’application du périmètre demandé. Les règles ne sont pas corrigées/supprimées silencieusement.

L’application globale devient une transaction unique pour son périmètre, au lieu d’une succession de transactions par règle. Le compteur `applied` conserve son sens historique : nombre de domaines complétés, pas nombre de tags ajoutés. La mutation de définition d’une règle (création/édition) et l’application demandée ensuite restent deux étapes : un échec d’application n’annule pas la définition déjà enregistrée. Aucun nouveau journal ne masque cette limite.

## 4. Parité des chemins et différences légitimes

| Chemin | Politique / différence légitime |
|---|---|
| Import initial | `Load`/`Match` utilisent les règles compilées et ordonnées communes pour le domaine d’une nouvelle ligne ; le passage post-import prépare/applique les tags de la gagnante au seul nouveau relevé. Une nouvelle ligne sans décision n’a pas de protection manuelle historique. |
| Import documentaire et fallback historique | Interfaces d’import conservées ; ni parsing, normalisation, signe, date, occurrence ou hash bancaire modifiés. Les tests d’import existants et une fixture supplémentaire vérifient le domaine et les tags finaux. |
| Classification automatique déterministe | Même plan ; ne remplace pas les domaines existants ni le NULL manuel. |
| Réapplication globale | Toutes les règles évaluées avant sélection des changements ; périmètre propriétaire demandé explicitement. Jamais exécutée sur des données réelles dans cette intervention. |
| Règle individuelle, création/édition avec application | Filtre par ID de gagnante ; mêmes règles concurrentes, même protection. Pas d’effet de « règle sélectionnée gagne forcément ». |
| Correction classique / lot | Décision utilisateur explicite, pas attribution automatique ; garde le verrou de ligne et envoie maintenant une précondition d’état depuis l’écran Transactions. Ajouts/retraits de tags travaillent sur l’état courant sous verrou. |
| Actions Agent | Décision préparée puis approuvée, propriétaire et snapshot vérifiés ; modification de provenance rend l’aperçu périmé. La création de règle Agent ne classe pas rétroactivement les transactions. Aucun moteur alternatif de tags ajouté. |
| Aperçus / audit / diagnostic | Même compilation et ordre. L’audit n’annonce plus les tags de règles perdantes comme « manquants ». Les décisions calculées aujourd’hui ne sont pas une provenance historique retrouvée. |

L’import reste constitué d’une persistance puis d’étapes post-import ; le lot de tags n’est pas atomique avec toute la chaîne documentaire et les appels IA. Une modification des règles entre ces étapes peut modifier la proposition actuelle. On conserve le domaine déjà persisté et on n’affirme pas une identité de règle historique qui n’est pas stockée. Une version durable de décision est une proposition 1B, pas un backfill autorisé.

## 5. Protections effectivement implémentées et limites

| Exigence | Résultat 1A |
|---|---|
| Domaine manuel A face à règle B | Protégé ; domaine non nul inchangé, tags de proposition incompatible non ajoutés. |
| Choix explicite Sans domaine | Représentable par `domain_id=NULL, classification_source='manual'` ; protégé contre règles et suggestion tardive. Cette combinaison existe déjà, aucun enum ajouté. |
| Ancien NULL sans décision | `unknown` n’est pas un refus ; toujours classifiable. Aucun backfill de source. |
| Domaine manuel avec tags encore automatisables | Oui si règle gagnante compatible avec ce domaine ; source du domaine n’est pas un verrou global des tags. |
| Tags manuels/historiques déjà présents | Préservés par ajout uniquement ; origine passée reste inconnue lorsque non tracée. |
| Tag retiré volontairement | **Non satisfait durablement.** PATCH retire le lien mais aucune tombstone ne distingue ce retrait d’un tag jamais présent. Le test documente sa réintroduction possible lors d’une réapplication explicite. |
| Verrou de domaine indépendant / verrou de tags / déverrouillage | **Non représentables dans le schéma actuel.** Aucun flag volatile, enum ambigu ni bouton inopérant créé. Ne pas assimiler manual à locked. |
| Deux corrections classiques concurrentes depuis Transactions | Précondition vérifiée sous verrou ; une mise à jour devenue périmée est refusée avec 409, sans mutation. Opérations par lot utilisent la même précondition et conservent leurs erreurs partielles explicites. |
| Action Agent périmée | Snapshots existants étendus à la source ; refus 409. Les actions préparées avant cette évolution peuvent nécessiter un nouvel aperçu plutôt qu’une application avec ancien fingerprint. |
| Suggestion IA tardive | Recontrôle SQL atomique empêche la proposition de remplacer un choix manuel, même NULL. Suggestions toujours non appliquées automatiquement. |
| Rejet IA durable | Hors implémentation 1A : identité spécifiée/testée, persistance proposée ci-dessous. Aucun rejet durable opérationnel n’est revendiqué. |

### Précondition de concurrence, sans révision persistée

Le rapport Transactions ajoute `classification_version`, empreinte de l’ID propriétaire/transaction, domaine nullable, provenance et tags normalisés triés. PATCH accepte en option `expected_classification_version`. L’UI et les lots l’envoient lorsque disponible ; les anciens clients sans ce champ restent compatibles et ne bénéficient pas de ce contrôle optimiste. Il ne s’agit ni d’un secret ni d’une autorisation ; les contrôles de propriétaire restent obligatoires.

La vérification se fait après verrou de transaction, avant domaine, tags ou suppression de proposition. L’état bancaire n’entre pas dans l’empreinte de classification. Sans révision persistée, un changement A→B→A n’est pas détectable ; la précondition n’est pas un historique complet. Aucun contournement n’est utilisé pour prétendre satisfaire ce critère plus fort. En cas de 409, le formulaire conserve le brouillon et affiche le refus ; aucune nouvelle tentative automatique n’est déclenchée.

### UX ciblée

Transactions garde Domaine + Tags. La liste avancée des règles indique les matches gagnants actuels et les motifs invalides. Conseils IA → Cohérence explique les choix manuels conservés, les propositions et les motifs invalides. Les liens vers Transactions utilisent les IDs exacts des sources au lieu de rechercher un texte de regex dans les libellés. Les listes de sources sont limitées à 1 000 par constat, distinctes du nombre total de cas. Aucun ancien mouvement ne reçoit une provenance de règle inventée.

## 6. Rejets : contrat préparé pour 1B, non implémenté durablement

État existant : `discard_suggestion` supprime `transaction_suggestions`, sans trace durable de rejet. Le générateur peut recréer la même proposition au prochain appel. Une action Agent `dismissed` conserve le refus de cette instance dans `agent_actions`, mais cela n’empêche pas de préparer une autre action sémantiquement identique ; ce n’est pas un registre général de rejets.

Identité de proposition définie par `SuggestionIdentity` : SHA-256 d’une représentation versionnée v1 contenant propriétaire, transaction, domaine proposé et tags normalisés (trim/minuscules, uniques, triés). Même ensemble de tags dans un ordre différent = même identité. Changer propriétaire, transaction, domaine ou tags = identité différente. Validation des IDs, taille et cardinalité de tags selon la proposition IA actuelle. Libellé bancaire, instant, fournisseur, tentative, modèle et nouvel import sont absents de cette identité. Cette fonction pure n’est pas encore branchée comme mécanisme de suppression de suggestions.

La future politique doit comparer **identité + révision de contexte pertinente**, distinctes de l’identité sémantique. Révision incrémentée lors d’une correction explicite, d’un changement significatif des règles/contexte qui concerne cette transaction ou d’une demande explicite de réévaluation. Un changement de modèle, une relance, un import différent ou une modification cosmétique ne suffit pas. Ne pas invalider tous les rejets parce qu’une règle sans rapport a changé.

Avant enregistrement d’un résultat LLM tardif : revérifier propriétaire, choix manuel, protections et révision capturée avant l’appel ; puis supprimer la proposition identique rejetée dans le même contexte. L’Agent et l’UI doivent pouvoir partager ce contrat sans transformer un refus d’action différent en rejet automatique de toute proposition future. Les tests 1A vérifient la canonicalisation et rendent visible la persistance manquante ; ils ne prétendent pas qu’elle existe.

## 7. Convention de remboursement validée, sans changement de calcul

Le mouvement bancaire reste à sa date réelle. Le coût net économique est imputé au mois de la dépense d’origine, révisable et consultable à date ; brut et net sont séparés. Une allocation confirmée ne doit pas être comptée simultanément comme baisse de coût et revenu économique. Exemple : débit de 100 € en septembre, crédit de 40 € en octobre ; coût de septembre révisé à 60 € à date après allocation, flux bancaires inchangés dans leurs mois respectifs.

Cette convention est désormais une décision métier pour la suite, **pas une fonctionnalité activée par 1A**. Services/tables de remboursements, budgets et agrégats sont inchangés ; leur net général ne doit pas être déduit de la seule existence des allocations. Les tests existants de remboursements et d’intégrité sont conservés.

## 8. Proposition SQL minimale pour 1B — aucune migration ajoutée/appliquée

Spécification à valider séparément, pas script exécutable :

| Structure additive candidate | Données et contraintes nécessaires |
|---|---|
| État de classification par transaction | PK propriétaire/transaction, FK composite vers transaction propriétaire ; révision monotone, `domain_locked` et `tags_locked` indépendants de `classification_source`, acteur/date. Absence d’état initial ≠ ancien refus volontaire ; pas de verrou manuel déduit de l’historique. |
| Décisions de tags par transaction | Clé canonique de tag, décision ajout/retrait manuel, propriétaire, transaction, révision et acteur. Sert aux protections ciblées/tombstones. Définir traitement des renoms/alias de tags et conflits avant stockage ; ne pas reconstruire l’origine historique inconnue. |
| Décisions de propositions | Propriétaire/transaction, identité v1, révision de contexte, décision de rejet et date/acteur ; unicité/idempotence de cette identité dans ce contexte. Pas de copie du brut bancaire. Une réévaluation explicite laisse une trace, pas une suppression générale des rejets. |
| Journal des décisions | Avant/après de classification, origine/acteur/règle/identité, révision et événement d’annulation compensateur. Écriture atomique avec mutation ; règles/version de décision sourcées pour ne pas confondre règle actuellement gagnante et règle ayant effectivement classé hier. |

Réutiliser l’index composite transaction/propriétaire existant et vérifier les FK réellement installées. Tester concurrence, idempotence, refus des liens étrangers, révision lors de chaque décision, autorisation de déverrouillage et conservation d’événements. Aucun nouveau domaine, tag ou enum de nature économique nécessaire à cette proposition. Pas de backfill des décisions historiques ; état inconnu explicite. Rollback fonctionnel par désactivation des lecteurs/automatismes, conservation des nouvelles preuves, aucune suppression de journal/tombstone après usage.

## 9. Validation et résultats

Validation finale : toutes les commandes listées ci-dessous ont réussi. Journaux locaux sous `/workspace/work/classification1a-*.log`, captures sous `/workspace/output/classification-sprint1a` ; ne pas confondre ces données synthétiques avec une mesure de l’installation réelle.

Commandes adaptées à l’organisation réelle (`internal/api`, pas `./api/...`), avec Go 1.26.11 et cache local explicite :

```sh
go test ./...
go test -race ./internal/categorize ./internal/classificationaudit ./internal/api ./internal/ai
go vet ./...
npm --prefix frontend run build
node frontend/tests/transactions.browser.cjs
node frontend/tests/agent.browser.cjs
```

Les commandes navigateur utilisent Vite local et Chromium système avec APIs entièrement interceptées ; aucun modèle externe appelé. Les tests Go utilisent PostgreSQL 16 éphémère réservé aux données synthétiques, pas de base utilisateur.

| Validation réellement exécutée | Résultat / preuve locale |
|---|---|
| Suite Go complète avec PostgreSQL synthétique | PASS — `classification1a-release-full.log`, tous les packages réussis, API et tests de migrations existantes inclus. |
| Race categorize/classificationaudit/api/ai avec PostgreSQL synthétique | PASS — `classification1a-release-race.log`, API 31,029 s ; tests de concurrence, état périmé, import et rollback exécutés. |
| `go vet ./...` | PASS — `classification1a-release-vet.log` vide, code de sortie 0. |
| Build frontend | PASS — `classification1a-final-build.log`. |
| Navigateur Transactions + liens de Cohérence | PASS — `classification1a-browser.log`, six viewports, clavier, Axe clair/sombre, cas 409/brouillon et sources par ID ajoutés. |
| Navigateur Agent | PASS — `classification1a-agent-browser.log`, six layouts/thèmes, console principale et panneau flottant, Axe. |
| Intégrité de livraison | Diff sans erreur d’espacement ; aucun diff dans migrations, schéma, remboursements, budgets, simulation ou analyse financière ; originaux du sprint 0 préservés. |

Couverture ajoutée : littéral/regex/ancrage/Unicode/%/_, priorité longueur/ID, domaines contradictoires ou identiques, gagnante sans tags, tags identiques/gagnants/perdants, métadonnées immuables, protections connues et états non représentables ; parité import/global/individuel/périmètre de relevé, règle modifiée, propriétaires, source manuelle/NULL, preuves bancaires intactes, rollback sur erreur d’insertion de tag, concurrence PATCH (un succès/un 409), contexte Agent périmé par source, IA tardive et identité/rejet non durable.

Parcours navigateur étendu : précondition sur PATCH, 409 sans répétition automatique avec brouillon conservé, liens de cohérence vers IDs exacts ; anciens parcours de remboursements, lots partiellement échoués, dates/réponses périmées, clavier, six formats et Axe conservés. Les tests Agent restent nécessaires à cause du changement d’empreinte.

Échecs intermédiaires corrigés : restriction sandbox des sockets de test (relance autorisée pour les serveurs synthétiques), contexte d’authentification du nouveau test corrigé vers l’API réelle, séquences SQL de fixtures à IDs explicites corrigées ; un nom de variable local du rapport et un chemin de copie de dépendances corrigés. Le test de navigation Conseils a révélé un sélecteur de rôle inadapté aux onglets et une réponse d’analyse simulée invalide ; fixtures corrigées, pas de test désactivé. Ces tentatives ne sont pas comptées comme réussites.

Non exécutés : CI distante, déploiement, métriques réelles, import bancaire réel, inspection ou mutation KG, tests des futures migrations/verrous/rejets durables (non implémentés). Aucun test n’est supprimé pour contourner une erreur.

## 10. Acceptation, risques et suite

| Critère | Statut et portée |
|---|---|
| Matcher/ordre commun, correspondances et invalidités explicites | Implémenté et couvert par tests. |
| Une seule gagnante pour les tags, y compris application individuelle | Implémenté ; historique conservé, pas nettoyage automatique. |
| Domaines manuels représentables, NULL manuel et anciens NULL | Implémenté sans migration. |
| Préservation tags manuels/historiques présents et preuves financières | Implémenté ; additions seulement, tests de sources et rollback. |
| Propriétaires, concurrence contrôlée et état périmé Agent | Implémenté dans les chemins précisés ; clients sans précondition et ABA restent des limites. |
| Verrous indépendants persistés, déverrouillage, protection du tag retiré | Non satisfaits sans nouveau stockage : proposition 1B seulement. Ne pas accepter D comme entièrement terminé. |
| Rejet IA durable | Définition/test de contrat 1A livrés ; effet durable non implémenté, reporté à 1B conformément au périmètre. |
| Tests techniques finaux | Réussis sur données synthétiques selon tableau ci-dessus ; aucune validation de production revendiquée. |

Risques : applications individuelles moins larges qu’avant ; certains anciens tags ne seront plus ajoutés ; application globale atomique garde des verrous plus longtemps mais évite les demi-applications par règle ; regex invalides sautées avec diagnostic au lieu d’arrêter la boucle ; nouveaux refus 409 en cas de concurrence ; empreintes Agent antérieures à repréparer. Les lectures de données financières peuvent changer après une application explicite de règles, sans changement de leurs formules : aucune modification des règles ou des historiques n’est exécutée sur des données réelles ici.

Décisions humaines restantes : autoriser ou non la spécification additive 1B ; définir les détails de protection par tag/renom, révision de contexte pertinente et interaction rejet Agent/proposition ; confirmer l’UX de déverrouillage et la portée de la compatibilité des anciens clients. Les décisions déjà validées (gagnante seule, manuel prioritaire, remboursement au mois d’origine) ne sont pas redemandées.

Sprint 1B recommandé : persistance des états, révisions, tombstones, rejets et journal ; activation explicite et tests de concurrence/isolation/rollback. Sprint 1C : extensions économiques/allocations et propagation aux consommateurs selon conventions validées, après garanties de persistance. Aucun de ces travaux n’est commencé par 1A.

## État de remise

Branche `feat/classification-sprint1a`, HEAD inchangé `d1bdec7a91d2719ccf67f755a8fef68dd46f0190` ; 15 fichiers suivis modifiés, nouveaux fichiers non suivis du moteur/tests/rapport et copies des livrables 0. Aucun commit, staging, push ou merge. Le checkout d’origine et ses documents 0 ont été comparés octet par octet avec leurs copies documentaires : inchangés. Diff SQL/dépendances/calculs financiers vide. Les journaux de validation finale sont aussi conservés avec les captures dans `/workspace/output/classification-sprint1a/`. Le serveur Vite de test est arrêté et le conteneur PostgreSQL dédié a été supprimé après la fin des tests. Les autres worktrees et services n’ont pas été touchés.

## Addendum — publication des sprints 0 et 1A

L’autorisation de publication permet commits, push et PR, sans fusion. La PR [#68](https://github.com/Madi-Toumani-benmohamed/Themis-gestion-finance/pull/68) publie les 11 fichiers du sprint 0 et étend uniquement les branches cibles acceptées par les workflows CI/Security pour vérifier la PR empilée. Aucun contrôle n’est retiré. Son commit `1d1d082070642a85c144fc3274552f2a8842559e` sert de base à 1A ; aucun développement non lié du dialogue Agent n’est repris de la branche d’origine.

Les 34 fichiers originaux de travail 1A ont été vérifiés par empreinte avant/après l’empilement : contenu identique, puis actualisation ciblée du présent rapport et du titre du graphe. Les fichiers du sprint 0 devenus suivis sont ainsi séparés des seuls changements 1A. Une sauvegarde locale hors Git protège les copies initiales ; elle n’est pas publiée. La branche d’origine et le worktree documentaire 1B restent intacts.

Une nouvelle validation prépublication est exécutée : suite Go complète avec `-race`, PostgreSQL 16 synthétique et extraction PDF requise, vet complet, build backend/frontend et parcours navigateur Transactions/Agent avec Axe. Les résultats et l’état des workflows distants sont fournis dans les PR et le bilan de publication ; les anciens journaux de remise ne sont pas présentés comme de nouveaux tests. Aucun journal, capture, secret, fichier temporaire ou donnée réelle n’est ajouté à Git.

Ordre recommandé : revue/fusion manuelle de #68 d’abord ; changer ensuite la base de la PR 1A de `feat/classification-sprint0` vers `main`, vérifier son diff et relancer les contrôles. Si #68 est squashée, une intégration non destructive du nouveau main dans la branche 1A pourra être nécessaire pour réconcilier l’ascendance ; aucun force-push requis. Ne pas supprimer prématurément la branche cible du sprint 0. Le sprint 1B ne peut commencer qu’après intégration et CI de 1A, et n’est pas implémenté pendant cette publication.
