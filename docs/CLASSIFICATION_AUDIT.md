# Audit du système de classification financière de Thémis

Date : 21 novembre 2026. Statut : audit et propositions à valider, aucune implémentation.

## 1. Périmètre, preuve et synthèse

`AGENTS.md` et `CODEBASE_GRAPH.md` ont été consultés avant inspection. Les constats reposent sur les migrations et le code Go/React/Python, pas sur les seuls documents d’architecture. Le snapshot inspecté est `dd198abe53c89b3eb2f972f3988ed86de8067c27`, dans le worktree `agent-comparison-followups`, contenu de la PR #67 fusionnée sur `main` par `d1bdec7a91d2719ccf67f755a8fef68dd46f0190`. Le checkout principal, resté sur `feat/agent-natural-dialogue` au commit `3d944f55779bfd8447feaa7156005ed29bb82d53`, n’a pas été déplacé. Les liens ci-dessous désignent les chemins du dépôt au snapshot audité.

**Conclusion : privilégier B, une évolution additive, après clarification des conventions de calcul.** Les domaines et tags sont réutilisables. Le problème principal est que le sens économique est aujourd’hui déduit du signe, d’un drapeau de domaine et d’un type de libellé. Une nomenclature plus propre ne suffit pas à distinguer consommation, transfert, acquisition d’actif, remboursement et retour de capital. Une refonte comptable complète n’est pas justifiée à ce stade ; quelques relations structurées sont toutefois nécessaires pour les transferts rapprochés et les ventilations.

Légende : **V** = comportement vérifié par lecture du code ; **R** = risque ou conséquence déduite, sans fréquence mesurée ; **P** = proposition, non implémentée ; **ND** = donnée réelle non disponible dans cet audit. Aucun service applicatif n’a été démarré : le démarrage exécute migrations et backfill de normalisation (`backend/cmd/server/main.go`), incompatibles avec la consigne d’absence d’écriture. Aucun accès SQL, endpoint applicatif, appel IA ou KG n’a été réalisé. Aucun test n’a été lancé : certaines suites créent des données. La lecture de tests existants ne constitue pas leur exécution.

Les métriques des utilisateurs, de la VM et de FalkorDB restent **ND**, et non zéro. Aucun libellé bancaire, identifiant de compte, montant réel ou secret n’a été extrait ni transmis. Ce document est le seul fichier ajouté ; aucun commit, push, merge, migration ou changement de classification n’est effectué.

## 2. Architecture actuelle vérifiée

### 2.1 Modèle et tables

| Objet | Représentation vérifiée | Références |
|---|---|---|
| Mouvement bancaire | `transactions` : identifiant, `owner_id`, `statement_id`, `tx_date`, `amount NUMERIC(12,2)` signé, `label` brut, `domain_id` nullable, empreinte. Provenance documentaire ajoutée : index/ligne/page, date de valeur, empreinte métier/source. Aucun champ général de nature économique ni ventilation économique d’une ligne. | `internal/models/models.go`, migrations `001_init`, `010_data_isolation`, `026_import_fidelity`; `api/import_fidelity.go::persistDocument` |
| Présentation/type partiel | `display_label`, `merchant`, `label_kind` ; ce dernier mélange épargne, taxe, prestation, santé et retrait d’espèces. Le modèle Go historique porte encore des `float64`; les rapports récents utilisent texte décimal/centimes. | Migrations 014/015, `normalize/normalize.go::Label`, `api/transactions.go` |
| Domaine/catégorie | `domains(id,owner_id,name,parent_id,color,is_transfer)` ; une affectation au plus par mouvement. Racine et un niveau d’enfant autorisés par `createDomain`. `/api/categories` est un alias vers les racines de `domains`, pas une deuxième taxonomie. La table historique `categories` a été supprimée par 006. | `api/features2.go::{listDomains,createDomain,updateDomain,deleteDomain}`, `api/handlers.go::listCategories`, migrations 004–008/010/013 |
| Tags | `tags(id,owner_id,name)`, relation N–N `transaction_tags(transaction_id,tag_id)` à clé composite. Minuscules et trim, dédoublonnage ; PATCH : maximum 50 tags, longueur 40 octets selon `len` Go. Pas d’ontologie ni de synonymes canoniques. | `api/transactions.go::{validateTransactionTags,replaceTransactionTags}`, migrations 004/010 |
| Règles | `rules(id,owner_id,pattern,domain_id,tags TEXT)` : tags séparés par virgules, et non relation normalisée. Motif brut, sans condition native sur signe, compte, montant ou période. | Migrations 006/012, `categorize/categorize.go`, `api/transaction_rules.go` |
| Provenance de classification | `classification_source` : `unknown`, `rule`, `manual`, `ai`. Valeur courante, pas un journal complet/versionné de toutes les décisions. Les historiques antérieurs restent `unknown` par défaut. | Migration 021, `api/transactions.go::updateTransaction` |
| Proposition IA | `transaction_suggestions`, une proposition par transaction, domaine/tags/source/date, propriétaire. Séparée de l’affectation effective. | Migration 021, `ai/enrich.go::AutoEnrich` |
| Compte et position | `accounts.kind` peut décrire courant/livret/PEL/bourse ; compte d’une transaction via son relevé. `account_snapshots` garde des positions manuelles. Cela ne constitue pas un registre des acquisitions/cessions ni une relation entre deux jambes de virement. | Migration 004, `api/features2.go::{listAccounts,addSnapshot}` |
| Remboursements | `reimbursement_cases`, `reimbursement_allocations`, `reimbursement_history` : qualification explicite, allocations en centimes, snapshots, révocations et isolation propriétaire. | Migration 027, `api/reimbursements.go`, `api/reimbursement_service.go` |
| Dette/engagements | `credits`, `credit_payments`, historiques d’engagement ; échéances et paiements déclarés distincts des transactions importées. | Migrations 022–024, `api/engagement_{models,payments,report,money}.go` |
| Projets | `financial_projects`, `project_rules`, `project_transactions`, `project_history` : association confirmée d’une transaction entière, pas ventilation multi-domaines de son montant. | Migration 031, `api/project_{links,report}.go` |

Les chemins `internal/...` de ce tableau sont sous `backend/`. Les contrôles propriétaires sont présents dans les principaux handlers et jointures financières. Plusieurs anciennes FK restent simples : ne pas confondre validation API et garantie composite généralisée en base. Les migrations récentes de remboursements/projets ajoutent des garanties composites explicites.

### 2.2 Attribution, ordre et mutations

**V — Import documentaire normal.** `handlers.go::finishStatementImport` appelle `persist`, qui délègue à `import_fidelity.go::persistDocument` lorsqu’une empreinte documentaire est disponible. Le chemin historique de `persist` subsiste. Les deux chargent `categorize.Load`, classent sur **le libellé brut**, normalisent avec `normalize.Label`, appliquent l’alias propriétaire à l’affichage puis persistent la ligne. La normalisation ne fournit pas le domaine à la place des règles.

**V — Priorité des règles.** `categorize.Load` trie par `length(pattern) DESC, id`. `Engine.Match` prend le premier motif correspondant. Un motif `re:` est une regex Go insensible à la casse ; sinon c’est une sous-chaîne littérale insensible à la casse, y compris `%` et `_`. Une regex invalide historique est ignorée au chargement ; l’édition/réapplication la refuse. La longueur du texte d’une regex n’est pas une mesure métier de spécificité.

**V — Deuxième passage.** Après persistance, `admin.go::applyAllRulesForStatement` parcourt toutes les règles dans le même ordre. `transaction_rules.go::applyRuleToTransactionsScoped` n’affecte le domaine que si `domain_id IS NULL`, mais ajoute les tags de **chaque règle correspondante même si un autre domaine est déjà attribué**. Les tags ne suivent donc pas exclusivement la règle gagnante. À l’import, le périmètre est le nouveau relevé ; une réapplication globale peut enrichir l’historique. Une application de règle est atomique ; la boucle de toutes les règles n’est pas une transaction globale avec l’import. L’échec post-import est journalisé et ne défait pas les lignes déjà persistées.

**V — Manuel.** `transactions.go::updateTransaction` valide propriétaire/domaine, verrouille la transaction, affecte `manual`, remplace ou ajoute/retire les tags atomiquement. Domaine et tags sont éditables, pas le montant/date/libellé bancaire via ce PATCH. Une affectation manuelle non nulle n’est pas écrasée par les règles. En revanche, remettre volontairement le domaine à `NULL` ne protège pas d’une future règle : le test porte sur la nullité, pas sur la provenance. Une modification des tags seule ne change pas `classification_source`.

**V — IA.** `AutoEnrich` cible les opérations sans domaine, maximum 500 par appel et lots de 40 ; domaines autorisés limités aux racines, liste d’au plus 200 tags existants, maximum quatre tags proposés. Il ne change aucun domaine/tag effectif et revérifie propriété et absence de domaine après l’appel. L’option d’auto-catégorisation à l’import produit donc des **suggestions**, pas des affectations automatiques. `TransactionDetails.jsx` reprend une suggestion dans le formulaire ; Enregistrer applique un PATCH manuel. Le refus supprime la proposition, sans registre durable de rejet empêchant une nouvelle proposition. La source `ai` existe dans le schéma, mais cette validation ne l’utilise pas comme provenance finale.

**V — Agent existant.** `agent_workspace.go::{prepareAgentAction,agentActionDecision}` prépare les actions de catégorisation/tag/règle, présente un aperçu, vérifie les données sous verrou et exige une décision explicite. La catégorisation validée écrit `manual`. Une règle créée par l’agent annonce un effet sur les prochains imports sans reclassement immédiat de l’historique. Il ne faut pas assimiler cette préparation à un apprentissage automatique des corrections.

**R — Apprentissage limité.** Une correction isolée ne crée pas spontanément une règle générale ; les alias et règles sont les mécanismes durables explicites. Une suggestion IA n’est ni une preuve ni une instruction comptable. Les libellés sont des données non fiables, pas des consignes.

### 2.3 Normalisation des commerçants

`normalize/normalize.go::Label` traite d’abord les retraits DAB explicites, puis le premier dictionnaire correspondant, le nettoyage technique (cartes, dates, heures, références, longs numéros), puis la casse et les sigles. PEL, Livret A, LDD/LDDS produisent `kind=savings`, avec affichage dépendant du signe. Un retrait produit `cash_withdrawal`, sans marchand. CPAM/AMELI produisent un libellé de remboursement et `health` : ce texte ne constitue pas une allocation de remboursement.

`merchant_alias.go::aliasFor` applique un alias exact sur la clé marchand (ou brut à défaut), puis les motifs par longueur décroissante et clé. Il modifie l’affichage, pas le domaine ni nécessairement l’identité `merchant` utilisée pour les regroupements. `setAlias` enregistre et propage l’affichage dans une transaction. **R :** sa propagation applique directement l’alias édité à toutes ses correspondances sans recalculer l’ensemble des alias concurrents ; l’affichage immédiat peut différer du choix de priorité à l’import/backfill en cas de chevauchement. `deleteAlias`/`renormalizeMatching` ne constituent pas une réconciliation générale de tous les motifs concurrents.

`normalize_backfill.go::backfillDisplayLabels`, version 5, renormalise au démarrage les colonnes de présentation et `label_kind` selon la version. **Point financier important :** même sans changement du montant/domaine, changer `label_kind` peut changer l’inclusion dans les agrégats. Un futur changement du dictionnaire doit donc être traité comme une évolution métier avec comparaison de périmètre, pas seulement cosmétique.

### 2.4 Exclusion des transferts

La référence commune est `financialsql/predicates.go::{Expense,ExpenseT}`, réexportée dans `api/transfer.go` : une ligne est ordinaire si `label_kind != 'savings'` et si ni son domaine propriétaire ni son parent propriétaire n’est `is_transfer`. Cela vaut pour **débits et crédits**. Un nom « Transfert interne » seul ne suffit pas ; un tag ne pilote pas ce calcul. Les liens de domaine étranger ne peuvent pas servir à exclure une ligne.

Le rapport Transactions calcule `financial_type` à la lecture : `transfer` si exclu, sinon débit/crédit selon le signe, sinon neutre. Ce champ n’est pas une nature économique persistée. Un remboursement non lié, un prêt reçu ou une vente d’actif positif restent des crédits ordinaires s’ils ne satisfont pas l’exclusion.

Le seed `api/seed.go::seedUserData` crée une racine `Transfert interne` avec le drapeau, contrairement à `Épargne et Investissements`, créée sans drapeau. `features2.go::{createDomain,updateDomain}` ne proposent pas l’édition de `is_transfer` ; renommer une catégorie ne change donc pas son comportement. L’absence de rapprochement entre comptes empêche de prouver que les deux jambes sont internes et complètes. La reconnaissance textuelle d’épargne peut exclure une ligne même sans preuve de compte destinataire.

### 2.5 Consommateurs et impacts

| Consommateur | Chemins vérifiés | Conséquence de classification |
|---|---|---|
| Liste/export Transactions | `transactions.go::{parseTransactionFilter,writeTransactionReport}`, `features.go::exportTransactions` | Filtres domaine/tag/type et totaux exacts ; proposition distincte de l’affectation. La source permet de contrôler les lignes réellement incluses. |
| Budgets | `budgets.go::{budgetAssignmentSQL,loadBudgetReport}` | Une enveloppe au plus : domaine exact prioritaire, sinon parent. Dépenses brutes, crédits séparés ; transferts/épargne exclus de consommation. Un budget de transfert garde un périmètre distinct. Un changement de domaine/parent peut déplacer une dépense ou changer sa couverture budgétaire. |
| Analyse | `analysis_report.go::{loadAnalysisReport,analysisReportFrom}`, `analytics.go::{tagsInCategory,tagSeries}` | Rapport en snapshot de lecture seule, flux ordinaires, taxes/épargne/transferts séparés. Les tags se chevauchent : leurs sous-totaux ne sont pas additifs. Le signe positif ne prouve pas un revenu économique. |
| Dashboard/signaux | `handlers.go::summary`, `pulse.go::pulse`, `hero.go`, `remaining.go::remaining`, `intelligence_jobs.go`, `behavior_discoveries.go` | Synthèses et signaux dépendent du même périmètre sur les chemins étudiés. Les endpoints historiques gardent des calculs `float64`/erreurs SQL parfois ignorées (`remaining` notamment) : ne pas étendre la garantie des rapports exacts à tous les widgets. Les recalculs d’intelligence post-mutation sont asynchrones et distincts du KG. |
| Simulation budgétaire | `simulation.go::{simulate,calculateSimulation,simulationMonths}` | Référence sur trois mois civils terminés, exclusions communes, montants exacts/rationnels. Réduction sélectionnée par nom de catégorie : renommage/fusion change les scénarios à réconcilier. Les crédits ordinaires non qualifiés peuvent gonfler le revenu de référence ; les remboursements alloués ne le corrigent pas actuellement. |
| Fiscalité | `fiscal/engine.go`, `api/fiscal.go` | Base fiscale saisie explicitement, moteur indépendant des banques : un crédit bancaire n’est pas automatiquement une base imposable. Conserver cette séparation. |
| Projets | `project_report.go::loadProjectReport`, `project_links.go` | Coût observé de lignes confirmées éligibles ; domaine et tags alimentent partitions/recherche, pas une comptabilité d’actifs. Ne pas cumuler des coûts de projets qui partagent des sources. |
| Assistant | `agent_tools.go`, `agent_read_scope.go`, observations et `chat.go` | Outils SQL avec preuves/périmètres ; une classification incorrecte change les faits fournis, même si le calcul est exact. La mémoire n’est pas la source des montants transactionnels exacts. |

**Limite d’audit :** absence de preuve d’une divergence dans tous les écrans ne démontre pas la parité de chaque SQL legacy. Une matrice exhaustive de parité par endpoint reste un critère d’acceptation avant changement du périmètre financier.

### 2.6 Knowledge Graph et FalkorDB

`handlers.go::finishStatementImport` envoie de manière asynchrone les faits d’un import validé à `memory.BuildFinancialFacts`, puis `knowledge_activity.go::learnKnowledge`. Un import non validé est signalé comme ignoré pour cet apprentissage. Les faits sont construits depuis le **ParseResult**, pas depuis une relecture des affectations finales SQL : date, brut, marchand du résultat, montant signé et direction, plus total d’import en flottants. **V : aucun domaine ni tag n’est inclus par `BuildFinancialFacts`.** Il n’est donc pas établi que le KG apprend la taxonomie effective simplement parce que le service est actif.

Le client Go appelle `memory/main.py::{learn_facts,query_kg}` ; Graphiti transforme les épisodes en entités/relations dans FalkorDB, groupés par propriétaire (`memory.Group`, groupe vérifié côté Python). `inspection.py` et le journal d’activité donnent une observabilité bornée ; `SearchFacts`/les handlers de chat joignent des faits au contexte assistant. Le KG est facultatif et ne bloque pas l’import.

**V/R :** les chemins de correction `updateTransaction`, règle et alias inspectés n’émettent pas une mise à jour versionnée de ces faits KG. Des faits textuels peuvent donc être incomplets ou périmés par rapport aux classifications actuelles. Graphiti peut inférer une relation, mais celle-ci ne prouve ni une affectation approuvée ni un rapprochement comptable. Aucune métrique de cohérence du graphe réel n’est disponible ici. Toute future synchronisation doit gérer versions, révocations, suppressions et source SQL, pas réinjecter aveuglément l’historique.

## 3. Inventaire taxonomique et qualité disponible

### 3.1 Inventaire statique, pas inventaire des utilisateurs

`seed.go` définit neuf catégories ordinaires : Logement, Transport, Alimentation, Santé et Bien-être, Divertissement, Épargne et Investissements, Vie Professionnelle, Famille, Autres ; une dixième racine technique Transfert interne est ajoutée séparément. Les règles initiales associent notamment supermarchés à Alimentation, énergie à Logement, UBER à Transport, services vidéo/musique à Divertissement, COTISATION à Autres. Aucun catalogue initial de tags n’est imposé par ce seed. Les utilisateurs peuvent avoir une nomenclature différente.

Les migrations 006 et 008 ont déjà changé la sémantique des catégories : anciennes catégories → domaines de vie → catégories actuelles. `Vie perso`/`Vie amoureuse` ont été remappées vers Autres ; certains anciens budgets ont été supprimés. **V :** ces scripts existent ; **ND :** leur effet réel sur chaque installation. Ne pas déduire le domaine historique d’origine à partir du seul domaine actuel, ni rejouer ces scripts pour une nouvelle rationalisation.

| Ambiguïté vérifiée dans la nomenclature | Risque sémantique à mesurer |
|---|---|
| Épargne et Investissements | Mélange transfert d’épargne, acquisition d’actif, frais et revenus financiers. Le nom ne pilote pas l’exclusion ; seuls drapeau/type le font. |
| Famille, Vie Professionnelle | Contextes transversaux : un achat alimentaire pour la famille ou un déplacement professionnel peut être rangé dans plusieurs domaines plausibles. |
| Santé et Bien-être | Soins, prévention et achats de confort regroupés sans distinction structurée ; `health` peut aussi désigner un crédit CPAM. |
| Autres | Regroupe inconnus, catégories volontairement diverses et héritage de migrations ; « Autres » n’est pas identique à sans catégorie. |
| Abonnement, facture, remboursement en tag | Attributs de récurrence/document/qualification, pas domaines exclusifs. Certains sont déjà partiellement structurés ailleurs. |

**ND :** aucun doublon effectif, fréquence d’utilisation incohérente ou taux d’erreur utilisateur n’est affirmé. Candidats à rechercher localement : tags équivalents à domaine, marchand, compte, signe, année/mois, projet, transfert, abonnement ou remboursement ; variantes accent/pluriel/abréviation ; même marchand dans plusieurs domaines. Cette dernière situation peut être correcte (supermarché, marketplace, achat mixte), et n’est pas à corriger automatiquement.

### 3.2 Requêtes proposées, non exécutées

À lancer ultérieurement seulement sur une copie locale autorisée avec rôle SQL strictement lecteur. `:owner_id` est un paramètre de l’opérateur local, jamais publié. La transaction est explicitement en lecture seule ; ni `EXPLAIN ANALYZE` applicatif, ni endpoint IA d’audit, ni export de lignes bancaires. Les sorties sont des comptes/ratios, sans noms, identifiants, dates individuelles ou montants. Les cellules de moins de cinq observations ne sont pas publiées ; aucune table de correspondance d’identifiants pseudonymisés ne doit être transmise. Les agrégats restent des données à protéger.

```sql
BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL statement_timeout = '15s';

-- Couverture et provenance ; unknown n’est pas une classification erronée.
SELECT classification_source, count(*) AS n,
       count(*) FILTER (WHERE domain_id IS NULL) AS sans_domaine
FROM transactions WHERE owner_id=:owner_id
GROUP BY classification_source HAVING count(*)>=5;

-- Profondeur/type et utilisation des domaines, sans exposer leurs noms.
WITH usage AS (
 SELECT d.id,d.parent_id,d.is_transfer,count(t.id) AS n
 FROM domains d LEFT JOIN transactions t
   ON t.domain_id=d.id AND t.owner_id=d.owner_id
 WHERE d.owner_id=:owner_id GROUP BY d.id
)
SELECT (parent_id IS NOT NULL) AS enfant,is_transfer,
       count(*) AS domaines,count(*) FILTER(WHERE n=0) AS inutilises
FROM usage GROUP BY 1,2 HAVING count(*)>=5;

-- Variantes typographiques exactes ; pas une détection de synonymes.
WITH names AS (
 SELECT lower(btrim(name)) AS cle,count(*) AS n
 FROM tags WHERE owner_id=:owner_id GROUP BY 1
)
SELECT count(*) AS groupes_de_variantes FROM names WHERE n>1;

-- Tags reproduisant un nom de domaine (test lexical conservateur).
SELECT count(*) AS tags_egaux_a_un_domaine
FROM tags tg WHERE tg.owner_id=:owner_id AND EXISTS(
 SELECT 1 FROM domains d WHERE d.owner_id=tg.owner_id
 AND lower(btrim(d.name))=lower(btrim(tg.name)));

-- Répartition du nombre de tags par opération, y compris zéro.
WITH n AS (
 SELECT t.id,count(tg.id) AS k FROM transactions t
 LEFT JOIN transaction_tags tt ON tt.transaction_id=t.id
 LEFT JOIN tags tg ON tg.id=tt.tag_id AND tg.owner_id=t.owner_id
 WHERE t.owner_id=:owner_id GROUP BY t.id
)
SELECT CASE WHEN k=0 THEN '0' WHEN k=1 THEN '1' ELSE '2+' END AS groupe,
 count(*) AS operations FROM n GROUP BY 1 HAVING count(*)>=5;

-- Écart potentiel normalisation/flag : aucune conclusion sur l’intention.
WITH s AS (
 SELECT t.label_kind,coalesce(d.is_transfer,false) OR
 coalesce(p.is_transfer,false) AS domaine_transfert
 FROM transactions t LEFT JOIN domains d
 ON d.id=t.domain_id AND d.owner_id=t.owner_id
 LEFT JOIN domains p ON p.id=d.parent_id AND p.owner_id=d.owner_id
 WHERE t.owner_id=:owner_id
)
SELECT (label_kind='savings') AS epargne,domaine_transfert,count(*) AS n
FROM s GROUP BY 1,2 HAVING count(*)>=5;

-- Marchands multi-domaines : signal de revue, pas taux d’erreur.
WITH m AS (
 SELECT merchant,count(*) AS n,count(DISTINCT domain_id) AS domaines
 FROM transactions WHERE owner_id=:owner_id AND merchant<>''
 GROUP BY merchant HAVING count(*)>=5
)
SELECT count(*) AS marchands_multidomaines FROM m WHERE domaines>1;

-- Liens historiques de propriétaire incohérents : ne pas les réparer ici.
SELECT count(*) AS domaines_hors_proprietaire FROM transactions t
JOIN domains d ON d.id=t.domain_id
WHERE t.owner_id=:owner_id AND d.owner_id IS DISTINCT FROM t.owner_id;

ROLLBACK;
```

Compléments à préparer après validation : recouvrement des règles avec **le matcher Go réel** dans un outil lecteur, nombre de propositions acceptées/rejetées quand une trace durable existera, couverture du rapprochement des transferts, part non ventilée des crédits et actifs. Une conversion aveugle de regex Go en regex PostgreSQL ne garantit pas la même sémantique. Une extraction de noms anonymisés ne suffit pas à qualifier des synonymes métier ; la revue sémantique reste locale. Le taux de précision exige un échantillon de référence validé par l’utilisateur, pas seulement le taux de remplissage.

## 4. Flux particuliers et fiabilité des agrégats

| Cas | Capacité actuelle vérifiée | Fiabilité/limite et information manquante |
|---|---|---|
| Virement courant ↔ compte personnel | Deux mouvements peuvent être importés sur des relevés/comptes différents ; domaine marqué ou `savings` les exclut des flux ordinaires. | Exclusion correcte si qualification correcte ; pas de paire structurée, bénéficiaire personnel confirmé, délai de rapprochement ni traitement dédié des frais/change. Une jambe manquante n’est pas preuve d’erreur. Ne pas dédupliquer les jambes comme doublons. |
| Épargne/désépargne | PEL/Livret A/LDD/LDDS reconnus par texte, exclusion dans les deux sens ; comptes et snapshots disponibles. | Pas une preuve universelle de transfert : intérêt de livret portant le même mot peut être exclu à tort ; autre formulation peut rester crédit/revenu ou débit/dépense. Pas de séparation apport/retrait/intérêt. |
| Investissement/acquisition d’actif | Domaine Épargne et Investissements et compte bourse permettent de ranger/afficher un flux et une position. | Aucun registre titre/quantité/prix de revient, lien apport→achat ou décomposition capital/frais. Achat ordinaire non exclu peut gonfler la consommation. Une catégorie marquée transfert exclurait aussi les frais/revenus qui y seraient rangés. |
| Cession/retour de capital | Crédit bancaire positif représentable. | Produit de cession ≠ revenu : capital restitué et plus-value non séparés ; revenu économique et fiscal non déductibles du seul crédit. |
| Remboursement marchand ou avance personnelle | Dossier explicite, un crédit et plusieurs allocations partielles vers dépenses, plafonds, verrous, idempotence, historique/révocation. | Coût net consultable au détail ; budgets/analyses/simulation restent bruts et crédit ordinaire. Qualification ne réécrit pas les mouvements. Manque politique commune de mois économique, « à date », reliquat non affecté et propagation aux agrégats. |
| Crédit reçu | Encaissement et engagement déclaratif possibles. | Pas un revenu économique ; pas de relation obligatoire transaction↔dette. Le crédit ordinaire peut être compté en revenu. |
| Mensualité de crédit | Débit bancaire + paiement déclaré dans `credit_payments`, suivi d’engagement exact. | Pas de lien transactionnel ni ventilation principal/intérêts/assurance/frais. Le suivi d’échéances ne démontre pas le capital restant dû bancaire. Additionner paiement déclaré et débit importé doublerait la sortie. |
| Revenus financiers | Dividende/intérêt positif représentable, domaine/tag facultatif. | Pas de type structuré brut/net/prélèvements ni distinction revenu/retour de capital. Le normaliseur d’épargne peut masquer des intérêts ; ne pas déduire la fiscalité des tags. |
| Transaction mixte | Domaine unique, tags multiples ; allocations de remboursement spécialisées et projets existent. | Pas de découpage générique de 100 € en deux postes réconciliés. Mettre deux tags compte potentiellement 100 € dans chaque vue ; ils ne sont pas des allocations de 60/40 €. |
| Retrait d’espèces | `cash_withdrawal` identifié sans marchand. | Pas exclu par le prédicat commun sauf domaine marqué ; sortie bancaire peut être présentée en dépense. Si les achats en espèces sont saisis ensuite, risque de double consommation. Une caisse espèces explicite manque. |

**Exemples de conventions cibles à valider (P).** Un transfert de 500 € entre deux comptes détenus produit deux jambes de trésorerie mais zéro consommation et zéro revenu consolidés. Un apport au compte titres puis un achat de titres de 500 € ne doivent pas devenir deux investissements de 500 € ; seul l’achat acquiert l’actif, l’apport déplace la trésorerie. Les frais de 2 € sont un coût séparé, pas 502 € de consommation.

Une échéance de 300 € comprenant 250 € de principal et 50 € d’intérêts donne 300 € de sortie bancaire, 250 € de réduction de dette et 50 € de coût économique (hors assurance/frais éventuels). Une déclaration de paiement documente ce même mouvement et ne s’y additionne pas.

Une dépense de 100 € remboursée à hauteur de 40 € conserve son débit de 100 € et un crédit bancaire de 40 € ; son coût supporté peut être 60 € une fois l’allocation confirmée. Ne pas simultanément déduire 40 € de dépense et les ajouter à un revenu économique. Le mois du crédit reste celui de la banque ; le mois d’imputation du coût net doit être une convention explicite et révisable.

## 5. Problèmes classés par gravité

Les niveaux décrivent l’impact possible sur la compréhension/calcul, pas une incidence mesurée en production. Aucun défaut n’est corrigé dans ce sprint.

| Niveau | Constat | Preuve et suite proposée |
|---|---|---|
| Élevé | Absence de nature économique indépendante ; crédits ordinaires assimilables à revenus et acquisitions à dépenses. | Prédicat commun, `financial_type`, simulation. Définir contrats trésorerie/coût/revenu avant reclassement. |
| Élevé | Épargne exclue par texte, sans rapprochement ; effet financier d’un backfill présenté comme normalisation. | `Label`, `financialsql`, backfill v5. Mesurer inclusions/exclusions et introduire une qualification explicable. |
| Élevé | Remboursements structurés non intégrés aux agrégats ; principal/intérêts non ventilés. | Rapports SQL et services spécialisés. Garder brut/net séparés, lier les preuves avant tout changement de total. |
| Élevé | Suppression d’un domaine peut désaffecter transactions et supprimer règles/budgets ; parent peut entraîner ses enfants. | FK 004/006, `deleteDomain`. Une fusion taxonomique doit conserver identifiants, liens et historique, pas utiliser suppression/recréation. |
| Moyen | Priorité par longueur et cumul des tags de règles perdantes ; absence de journal complet des décisions. | Moteur et application. Ajouter simulation de conflits/provenance avant automatisation accrue. |
| Moyen | « Sans domaine » manuel peut être reclassé ; rejet IA non durable ; sous-domaines non proposés par l’IA. | PATCH, `domain_id IS NULL`, `AutoEnrich`. Définir abstention/verrou manuel et politique de rejet. |
| Moyen | Nomenclature mélange fonction, contexte et patrimoine ; mappings historiques irréversibles à partir du seul état courant. | Seeds/migrations 006/008. Cartographie versionnée non destructive et revue utilisateur. |
| Moyen | KG ne reçoit pas les domaines/tags finaux et ne suit pas leurs corrections dans les chemins étudiés. | `BuildFinancialFacts` et handlers. Références de source/version ; aucune autorité comptable accordée au graphe. |
| Moyen | Audit de cohérence et moteur n’ont pas exactement la même sémantique ; alias superposés peuvent diverger selon chemin. | `coherence.go` utilise `ILIKE`, ignore `re:` ; `setAlias` vs `aliasFor`. Unifier les contrats, tester les chevauchements. |
| Moyen | Certains endpoints legacy sont en flottants/ignorent des erreurs, contrairement aux rapports exacts. | `remaining`, `pulse`, structures historiques. Vérifier la parité et les états inconnus avant extension. |
| Faible à moyen | Tags redondants/synonymes possibles, identité marchand textuelle et renoms non équivalents à fusion. | Normalisation/tags/alias. Fréquence ND ; métriques et revue locale nécessaires. |

## 6. Comparaison des architectures

| Critère | A — modèle actuel amélioré | B — évolution légère additive | C — séparation complète |
|---|---|---|---|
| Technique/maintenance | Faible effort initial ; ambiguïtés et exceptions persistantes dans les consommateurs. | Effort moyen : nature, attributs typés, politique de calcul et liens ciblés ; réutilise API/domaines/tags/allocations. | Effort élevé : mouvements, événements économiques, allocations, positions/dettes, rapprochement et multiples vues. |
| Qualité analytique | Meilleurs rangements ; ne résout pas principal/intérêts, actifs et opérations mixtes. | Bonne pour cas fréquents si qualification prouvée et agrégations centralisées ; limites patrimoniales annoncées. | Potentiel maximal ; exige données et conventions comptables que l’historique ne contient pas toujours. |
| Historique | Compatibilité simple si pas de remapping automatique ; reclasser les domaines change malgré tout les lectures. | Ancien modèle conservé, nature inconnue admise, projections versionnées et opt-in. | Adaptateurs/backfill complexes ; beaucoup de liens non reconstructibles sans intervention. |
| Fiabilité des sommes | Exactitude arithmétique possible, sémantique toujours dépendante de drapeaux/textes. | Périmètres explicites, sources et réconciliation ; validation des ventilations indispensable. | Forte si invariants prouvés ; risque initial supérieur pendant coexistence de plusieurs livres. |
| Automatisation/IA | Règles/suggestions plus soignées, mais peu de contexte économique structuré. | Suggestions typées, confiance par axe, apprentissage supervisé de décisions approuvées. | Raisonnement riche sur événements/liens ; coût important de validation et explication. |
| UX | Très simple ; correction des exceptions difficile à expliquer. | Nature déduite/proposée avec validation des ambiguïtés ; domaine simple, tags optionnels, détails progressifs. | UX lourde si concepts comptables exposés ; nécessite une couche produit forte. |
| Migration | Faible à moyenne ; danger de fusion/suppression de domaines. | Moyenne mais maîtrisable par ajouts, shadow reads et activation progressive. | Élevée ; rollback et réconciliation historique coûteux. |

**Recommandation P :** B comme trajectoire, A comme première étape de diagnostic/rationalisation, C réservé à un besoin confirmé de portefeuille, multi-devises, positions et comptabilité de dette. Ne pas promettre la gestion patrimoniale complète par un simple enum. Les relations de remboursement existantes ne doivent pas être jetées ni dupliquées ; les réutiliser avec adaptateurs vers les vues économiques futures.

## 7. Taxonomie cible proposée

Les trois axes proposés sont compatibles avec les domaines/tags existants, à condition de distinguer **mouvement** et **interprétation/portion économique**. La nature ne doit pas être dérivée durablement du domaine. Une ligne mixte peut porter plusieurs portions : un enum unique sur la ligne ne suffit pas aux cas de crédit ou d’achat mixte.

- Nature économique : dépense, revenu, transfert, opération patrimoniale ; état **non qualifié** explicite, jamais conversion forcée en dépense/revenu. Le signe décrit le sens bancaire, pas la nature. Un remboursement peut être un ajustement de dépense lié à sa source ; ce traitement requiert un sous-type/lien, pas un cinquième domaine technique.
- Domaine fonctionnel : Logement, Alimentation, Transport, Santé, Loisirs/culture, Éducation, Communications, Services/frais, Impôts/taxes, Autres ; liste indicative à valider, sous-domaines facultatifs et bornés. Les coûts/frais d’investissement peuvent avoir un domaine sans ranger l’acquisition de capital dans la consommation. Conserver « sans domaine » distinct d’Autres.
- Tags libres : contexte volontaire, par exemple voyage, cadeau ou événement. Famille/professionnel sont candidats à contexte ou attribut structuré lorsqu’ils pilotent une règle métier ; pas de déplacement automatique des anciennes affectations.
- Attributs structurés seulement lorsqu’ils gouvernent un calcul : sous-type patrimonial, compte/contrepartie détenue confirmée, relation de transfert, remboursement alloué, dette, portion principal/intérêts/frais, dates bancaire et économique, source/confiance/version/validation. Récurrence confirmée n’est pas un tag qui prouve un abonnement ; projet conserve son identifiant propre.

**Invariants proposés :** montant/date/brut/source du mouvement conservés ; somme des portions signées égale au montant de la ligne, sans surallocation ; aucun mouvement dupliqué pour classifier ; relations toujours propriétaires ; pas de plus-value calculée sans prix de revient ; transfert rapproché non additionné à consommation/revenu ; rapport brut et rapport économique séparés et réconciliables. En l’absence de données, portion non qualifiée visible plutôt que résultat inventé. Distinguer complétude documentaire, complétude de classification et confiance de suggestion.

## 8. Migration non destructive, validation et rollback

Cette section est un plan futur, pas une autorisation ni un script exécuté.

1. Figer les définitions des agrégats, les cas synthétiques et le mapping proposé. Produire un inventaire local autorisé. Sauvegarde chiffrée et restauration testée dans un environnement isolé avant écriture future.
2. Ajouter des structures versionnées sans renommer/supprimer domaines, règles ou tags existants. Préserver `domain_id`, provenance documentaire, liens de remboursements/projets et identifiants. Les anciens enregistrements restent non qualifiés tant qu’aucune preuve ne permet mieux.
3. Calculer les nouvelles vues en parallèle, en lecture seule : comparer par transaction, période, compte, propriétaire, domaine et consommateur. Un écart économique attendu doit avoir une explication et des sources ; un écart de solde bancaire est interdit.
4. Présenter les changements de taxonomie sous forme de propositions de correspondance, avec nombre de lignes et effet sur budgets/scénarios/règles. Validation propriétaire explicite ; pas d’équivalence déduite seulement d’un nom. Conserver alias/historique et possibilité d’afficher la taxonomie à date.
5. Activer par fonctionnalité et propriétaire après critères d’acceptation. Une seule source d’agrégation par vue ; ne pas additionner anciennes et nouvelles lignes. Pas de recalcul silencieux de tous les mois historiques.
6. Synchroniser ensuite les faits KG approuvés par source/version, avec invalidation des versions périmées. En cas d’échec KG, les vues SQL restent disponibles et leur état de synchronisation est explicite.

**Validation future :** jeu synthétique à deux propriétaires, deux comptes, deux jambes à dates différentes, une jambe absente, frais, change non résolu, intérêts de livret, achat/cession d’actif, crédit et échéance ventilée, remboursements partiels/multiples/révoqués, tags chevauchants, opération mixte, mois partiel, limites calendaires, import répété et occurrences légitimes identiques. Tester concurrence, reprises idempotentes, erreurs SQL, refus d’IDs étrangers, règles et alias concurrents, modifications manuelles, historique et export. Vérifier les centimes et l’ensemble des sources incluses, pas uniquement le total final.

**Critères de passage :** zéro changement de montant/date/brut/empreinte/identité historique ; zéro fuite propriétaire ; zéro surallocation ; zéro double compte ; réconciliation bancaire identique ; différences économiques expliquées ; incohérences/inconnus visibles ; sources navigables ; application et retour arrière rejouables sans perte. La précision de l’IA se mesure sur décisions de référence locales, par nature/domaine, avec taux d’abstention et erreurs coûteuses séparés ; un taux de remplissage élevé n’est pas un succès suffisant.

**Rollback futur :** désactiver la lecture économique/nouvelle automatisation par drapeaux ; conserver les données additionnelles et revenir aux lectures legacy documentées. Annuler une décision par événement compensateur/version, pas par suppression de preuve. Restaurer la projection précédente des mappings, sans écraser les corrections manuelles postérieures ; ces conflits nécessitent une revue. Pas de rollback SQL destructif d’un schéma déjà utilisé. Une restauration complète de sauvegarde est un dernier recours coordonné car elle peut perdre les imports intervenus depuis ; tester reprise/journal avant tout déploiement. Réindexer le KG à partir des versions approuvées après rollback, jamais restaurer des faits périmés comme vérité financière.

## 9. Sprints proposés

| Sprint | Périmètre | Acceptation et non-régression |
|---|---|---|
| 0 — Décisions et mesures locales | Valider ce rapport, conventions consommation/revenu/patrimoine, métriques et corpus synthétique. Aucune reclassification. | Inventaire agrégé local autorisé ; lexique et cas d’acceptation signés ; liste des inconnus conservée. |
| 1 — Intégrité et contrats | Centraliser la politique des flux, préciser brut/net/observé, parité des consommateurs, journal de classification et protection des décisions manuelles. | Tests propriétaires, centimes, alias/règles, import/export et périodes ; aucune modification des transactions historiques ; erreurs distinctes des zéros. |
| 2 — Nature explicite en parallèle | Nature/attributs versionnés additifs, état inconnu, aperçu d’impact et lectures shadow ; aucune bascule générale. | Ancien rendu identique sous flag éteint ; différences sourcées sous flag actif ; nullité manuelle/rejets protégés ; rollback de lecture testé. |
| 3 — Rationalisation domaine/tag | Correspondances proposées, synonymes, rôle des domaines contextuels, règles expliquées et détection de conflits. | Aucun effacement/fusion automatique ; budgets/scénarios/liens préservés ; comptages avant/après et historique réconciliés ; tags non additifs explicités dans l’UX. |
| 4 — Relations et ventilations prioritaires | Transferts confirmés, réutilisation des remboursements, principal/intérêts/frais et cas mixtes bornés. | Sommes des portions réconciliées ; paiements déclarés jamais ajoutés deux fois ; deux jambes/une jambe/frais/annulation/concurrence ; coût économique distinct de trésorerie. |
| 5 — Automatisation supervisée | Suggestions typées, confiance par axe, abstention, rejet durable et apprentissage sur corrections approuvées ; règles conservatrices. | Mesure de précision sur corpus local, zéro application sur ambiguïté coûteuse, limites explicites et arrêt automatique ; pas d’envoi externe non validé. |
| 6 — Analyses, prévisions et KG | Activer progressivement les vues économiques par consommateur ; scénarios cohérents ; faits KG versionnés et révocables. | Matrice Transactions/Budgets/Analyse/Dashboard/Simulation/Assistant/Projets ; E2E et Axe, sources exactes ; reprise KG en panne et absence de données sans invention. |

Suites existantes utiles à préserver/étendre lors de ces futurs sprints : `categorize/categorize_test.go`, `api/transactions_test.go`, `transaction_quality_integration_test.go`, `budgets*_test.go`, `simulation_test.go`, `reimbursements_integration_test.go`, migrations/rollbacks et tests de projets ; parcours `frontend/tests/{transactions,budgets,simulation,analysis,projects,credits,agent}.browser.cjs`. Leur présence a été examinée ; aucun résultat d’exécution n’est revendiqué dans cet audit.

## 10. Décisions à valider avant implémentation

1. Retenir B comme direction, sans imposer la comptabilité patrimoniale complète de C ; préciser les cas hors périmètre de la première version.
2. Définir les mots et indicateurs visibles : crédit bancaire observé, revenu économique, sortie bancaire, consommation, coût supporté et variation patrimoniale.
3. Choisir l’imputation des remboursements et le traitement des révisions historiques : mois d’origine/à date et mois du crédit doivent rester consultables séparément.
4. Approuver la qualification explicite des transferts et le traitement des intérêts d’épargne, des retraits d’espèces, des retours de capital et du principal de dette.
5. Valider la liste de domaines et le déplacement éventuel des contextes Famille/Vie professionnelle, avec mappings présentés au propriétaire et conservation de l’historique.
6. Définir la protection d’un choix manuel « non qualifié », la priorité des règles, le cumul des tags, la mémoire des rejets et les seuils d’automatisation.
7. Autoriser séparément, si souhaité, les mesures agrégées locales et leur périmètre ; aucune extraction de données réelles n’est nécessaire pour accepter l’architecture de principe.
8. Valider la stratégie additive, les critères de bascule/rollback et la synchronisation KG sans lui déléguer l’autorité comptable.

**État final de ce sprint :** rapport documentaire uniquement, aucun code ou fichier partagé modifié, aucune base consultée/modifiée, aucune migration, aucun changement de règle/tag/domaine, aucune opération Git de publication. Les observations fonctionnelles sont statiques et les métriques de production restent à établir localement si autorisées.
