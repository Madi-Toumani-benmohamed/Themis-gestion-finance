# Métriques locales de classification — sprint 0

21 novembre 2026. **Aucune mesure réelle exécutée.** Aucun accès à PostgreSQL utilisateur, aucun export de libellés, aucun appel IA/KG. Les tests utilisent exclusivement des objets synthétiques et un faux driver SQL. L’[inventaire statique](CLASSIFICATION_TAXONOMY.md) n’est pas un inventaire des données présentes.

## Outil facultatif et garanties

Commande ajoutée : `backend/cmd/classificationaudit`, service isolé `backend/internal/classificationaudit`. Réutilise `categorize.CompilePattern` et `money.Parse` existants ; aucune nouvelle dépendance. Aucun import des packages serveur, migrateur, configuration `.env`, normaliseur/backfill, worker, IA ou KG. Elle ne lance pas Thémis.

`Run` exige un propriétaire positif, ouvre une transaction `REPEATABLE READ READ ONLY`, exécute quatre SELECT paramétrés sur domaines/tags/règles/transactions et liens de tags, puis fait toujours ROLLBACK. Aucune écriture DML/DDL, table temporaire, procédure applicative ou commit. Les requêtes sont bornées à 50 001 lignes par collection pour détecter le dépassement ; plafond global 50 000 objets, 5 millions d’essais règle×transaction et 5 millions de paires cumulées, taille de libellé/marchand/motif bornée. Dépassement ou donnée invalide : erreur, **aucun rapport partiel**. Contexte de 30 secondes pour lectures SQL ; calcul mémoire borné en quantité de travail, non interrompu à chaque match. Pour des volumes supérieurs, concevoir un traitement paginé validé, pas tronquer silencieusement les résultats.

Le mode lecture seule côté serveur est obligatoire ; recommander en plus un rôle PostgreSQL existant strictement lecteur avec accès limité. Ne pas créer un rôle, modifier des droits, activer un service ou appliquer des migrations pour cet audit. Utiliser une copie locale autorisée si possible. Les limites réduisent la charge mais une lecture peut prendre des verrous légers et consommer des ressources : lancer hors charge sensible.

Le rapport n’émet pas libellés, marchands, noms de domaine/tag, motifs de règles, ID utilisateur/compte/transaction/relevé ou dates de mouvements. Il donne des références ordinales `domain-N`, `tag-N`, `rule-N`, et des comptes/montants. Ce sont des **agrégats locaux sensibles, pas une anonymisation irréversible** : petits groupes, montants et références stables peuvent permettre des inférences. Ne pas les publier, les ajouter à Git, les coller dans un assistant ou un log CI. Aucun filtrage de petites cellules n’est appliqué dans cette sortie privée ; tout partage ultérieur exige suppression/agrégation des petites cellules et revue explicite.

Les noms et libellés nécessaires au matcher sont lus en mémoire locale seulement. Les rapports JSON ne sont produits qu’après réussite complète ; les erreurs CLI sont génériques et ne reprennent pas le DSN/texte SQL/valeurs. `DATABASE_URL` est fourni par l’opérateur, sans argument de ligne de commande contenant le secret. Il n’y a pas de découverte automatique de base réelle.

## Exécution locale proposée, non exécutée sur données réelles

Depuis `backend/`, avec Go compatible et dépendances existantes :

```sh
go test -race ./internal/classificationaudit ./cmd/classificationaudit ./internal/categorize
go vet ./internal/classificationaudit ./cmd/classificationaudit
go build -o /tmp/themis-classificationaudit ./cmd/classificationaudit
```

Après autorisation locale et avec `DATABASE_URL` déjà injectée de manière privée par l’opérateur :

```sh
umask 077
read -r -p 'ID du propriétaire local : ' THEMIS_AUDIT_OWNER
/tmp/themis-classificationaudit -owner "$THEMIS_AUDIT_OWNER" > /tmp/themis-classification-private.json
```

Ne pas versionner le résultat ; le conserver dans un emplacement privé adapté à la machine (le chemin `/tmp` est illustratif). Vérifier le code de sortie avant de lire le fichier : un shell peut créer un fichier vide sur échec, ce n’est pas un rapport zéro. L’outil n’effectue aucun upload ni télémétrie. Aucune commande d’accès à une base réelle n’a été lancée pendant ce sprint.

## Définitions des champs et interprétation

Périmètre : **un propriétaire, toutes les dates importées**, pas un bilan mensuel ni une preuve de complétude des relevés. Les comptes absents/non importés restent hors mesure. Les champs de comptes commencent par majuscule dans JSON (`DomainTotal`, etc.) ; les tableaux et flux utilisent les clés explicites en minuscules. Les montants sont des **chaînes de centimes entiers** pour éviter la perte de précision des lecteurs JSON JavaScript.

| Indicateur/champ | Définition exacte | Limite / interprétation |
|---|---|---|
| `transactions`, `DomainTotal`, `TagTotal` | Lignes propriétaires lues, domaines et tags existants | Zéro = snapshot lu vide ; indisponible = commande en erreur, pas zéro. |
| `DomainUsed`, `TagUsed` | Au moins un mouvement propriétaire directement associé | Domaine parent sans affectation directe peut servir à ses enfants, budgets et règles. |
| `DomainParents`, `DomainChildren` | Parents ayant un enfant propriétaire ; domaines ayant un parent déclaré | Pas un classement sémantique ; les liens parentaux anormaux doivent être revus séparément. |
| `domains[].transactions`, `.bank_flow` | Nombre de lignes et D/C/N pour chaque domaine exact | Aucun rollup parent : pas de double compte parent/enfant. Aucun filtre transfert/épargne pour ces flux bancaires. |
| `bank_flow` | D positif = −somme débits ; C positif = somme crédits ; net signé = C−D | Comprend transferts et capital : ce n’est pas la consommation/revenu. |
| `unassigned_bank_flow`, `WithoutDomain` | Lignes ayant domain_id NULL | Sans domaine ≠ nature inconnue garantie, ni erreur. |
| `invalid_domain_bank_flow`, `InvalidDomainLinks`, `InvalidTagLinks` | Référence absente de la collection propriétaire | Ne lit aucun nom étranger ; ne répare pas. Les domaines valides + sans domaine + invalides réconcilient le flux brut. |
| `classification_sources` | Comptage de unknown/manual/rule/ai ; autres valeurs regroupées en other | Provenance courante, pas historique des décisions ; une modification de tag seule n’est pas identifiée comme source séparée. |
| `tags[].transactions`, `.domains` | Fréquence unique du tag et distribution par domaine exact | Plusieurs tags sur une ligne se chevauchent : pas de somme additive des sous-totaux. |
| `observed_tag_pairs`, `always_together` | Cooccurrences ; vrai si les deux tags ont exactement le même ensemble observé | Corrélation observée, pas preuve de redondance. Petits effectifs sensibles. |
| `single_nonempty_merchant`, `unknown_merchant_transactions` | Toutes occurrences du tag avec un même marchand non vide ; compte des inconnus | Vrai seulement si aucun marchand manquant ; identité textuelle actuelle, pas une entité commerciale certifiée. |
| `domain_name_candidate` | Nom normalisé égal à un domaine local | Signal lexical, ne pas supprimer un tag. |
| `structured_attribute_candidate` | Petit lexique : revenu/dépense/transfert/épargne/remboursement/abonnement/mensuel/crédit/débit/virement selon variantes du code | Détecteur volontairement limité ; ne reconnaît pas tous les comptes, projets, mois ou synonymes. Revue locale nécessaire. |
| `TagLexicalGroups`, `DomainLexicalGroups` | Plusieurs noms égaux après trim/minuscules | Ne traite ni accents/pluriels ni synonymes ; ne détecte pas tous les recouvrements de sens. |
| `rules[].executable`, `invalid_pattern` | Motif compilable et domaine propriétaire disponible ; invalidité du motif distincte | Pas de drapeau actif en base. Une règle sans match reste exécutable, pas inactive. |
| `rules[].priority`, `matches`, `first_matches` | Ordre longueur Unicode décroissante/ID croissant ; nombre de matches ; premier parmi règles exécutables | Ordinal inclut les règles invalides pour inventaire ; elles ne participent pas à la décision. Contrôle le comportement du matcher, pas une mutation. |
| `observed_rule_overlaps`, `different_domains` | Une paire matche la même ligne ; domaines distincts | Collision réellement observée sur le libellé, pas recherche de toutes les intersections possibles de regex. |
| `ConflictingTransactions` | Au moins deux domaines différents proposés par les règles exécutables | Pas nécessairement une erreur : priorité spécifique/générique parfois intentionnelle. |
| `RuleTagsRepeated` | Nombre de noms de tags présents dans plusieurs règles exécutables | Unicité des liens empêche de doubler un tag identique ; ce compteur n’est pas un double montant. |
| `LosingRuleAddsTags` | Au moins une règle moins prioritaire propose un tag absent de la première règle sur la même ligne | Effet potentiel de la boucle d’application ; ne démontre pas l’origine réelle des tags ni que toute la boucle a terminé. |
| `ManualDisagreements` | Source manual, domaine non nul différent du domaine de la règle gagnante observée | Manuel actuellement protégé sur le domaine, pas preuve qu’il faut l’écraser. |
| `ManualUnassignedMatches` | Source manual, domaine NULL et au moins une règle correspondante | Le choix nul peut être repris par une réapplication ; pas de modification dans cet outil. |
| `literal_sql_wildcard`, `regex_skipped_by_legacy_coherence` | Motif simple contenant %/_ ; regex re: | Dénombre des écarts de sémantique possibles avec coherence.go, sans exécuter cet audit ni remplacer le matcher. |
| `SavingsWithoutTransferFlag`, `TransferFlagWithoutSavings` | Désaccord entre type savings et drapeau effectif domaine/parent | Deux mécanismes d’exclusion peuvent légitimement couvrir des cas différents. Ne prouve ni faux positif ni faux négatif. |

## Tests et résultats de ce sprint

Tests synthétiques : exactitude des centimes et signes, chevauchement UBER/UBER EATS, tags gagnants/perdants, correction manuelle et NULL, regex Go et littéraux `%/_`, priorité Unicode/tie-break, données invalides et limites, tags sans marchand, non-mutation du dataset et absence de textes privés dans le JSON. Faux driver : propriétaire requis, SELECT paramétrés uniquement, options READ ONLY/REPEATABLE READ, rollback sur succès/erreur et absence de détails SQL sensibles dans l’erreur remontée.

Les commandes effectivement exécutées et leur résultat final sont consignés dans [SPRINT_1_PLAN.md](SPRINT_1_PLAN.md#validation-du-sprint-0). Pas de connexion PostgreSQL synthétique ou réelle, pas de test des droits d’un rôle réel ni du transport TLS pendant ce sprint. Le faux driver vérifie le contrat de `database/sql`, pas le comportement d’une instance PostgreSQL ou son schéma installé.

L’outil ne mesure pas précision sémantique, fraude, qualité d’un panier, dates économiques, performance IA ni cohérence réelle FalkorDB. Ces états restent ND. Une mesure de précision nécessiterait un corpus validé localement et une autorisation distincte ; le corpus du sprint fixe les conventions attendues, pas un taux de réussite en production.
