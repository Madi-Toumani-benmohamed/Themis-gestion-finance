# Projets financiers — contrats et règles

## Source de vérité

`project_transactions` est l'unique source des liens confirmés. Clé unique `(project_id, transaction_id)` et FK composites sur le propriétaire. Une suggestion ou un rejet ne contribue jamais au coût. Les règles ne confirment aucune opération. Les données sources ne sont jamais modifiées par le module.

Coût brut = somme exacte des débits ordinaires confirmés, dans les dates du projet et jusqu'au mois consulté (aujourd'hui pour le mois courant). Transferts descendants et épargne normalisée exclus selon `financialsql`. Tous les montants sont des entiers en centimes ; aucune compensation de crédit, annulation ou remboursement n'est inférée. Aucun coût net n'est publié. Les opérations devenues inéligibles restent visibles et traçables, exclues du coût.

Budget absent = `null` pour reste, dépassement et pourcentage ; budget nul = reste négatif du coût, dépassement égal au coût, pourcentage `null`. Budget strictement positif : taux déterministe arrondi à deux décimales, calcul rationnel partagé. Si un agrégat dépasse la précision exacte JSON, le rapport échoue explicitement.

Les catégories, marchands et mois constituent trois partitions réconciliables du coût. Les tags se chevauchent. Une opération peut appartenir intégralement à plusieurs projets ; les coûts interprojets ne s'additionnent pas.

## Routes authentifiées

| Méthode et route | Contrat |
|---|---|
| `GET /api/projects?month=YYYY-MM&offset=0` | 100 projets au maximum, `has_more`, date limite et avertissement non additif. Agrégations restreintes aux projets de la page. |
| `POST /api/projects` | `{name,description,budget_cents:null|entier,start_date:""|YYYY-MM-DD,end_date:""|YYYY-MM-DD,status}` ; renvoie `{id}`. |
| `PUT /api/projects/{id}` | Remplacement explicite des métadonnées, même contrat ; verrou et historique atomique. Statuts `active`, `completed`, `archived`. Pas de suppression destructive : archiver conserve les preuves. |
| `GET /api/projects/{id}?month=YYYY-MM` | Projet, coût, indicateurs, partitions et observations depuis un snapshot REPEATABLE READ ; aucune mutation. |
| `GET /api/projects/{id}/transactions?month=YYYY-MM&state=confirmed|suggested|rejected&category=…&offset=0` | 50 opérations, `has_more`, raison persistée, éligibilité et inclusion actuelles. Les confirmations exclues du coût restent visibles. `category` est un nom exact du rapport, aucun filtre approximatif. |
| `POST /api/projects/{id}/decisions` | `{transaction_ids:[…],state:"confirmed"|"rejected"}` ; lot atomique de 1 à 100 IDs. Confirmer associe aussi manuellement une dépense non suggérée. Rejeter une confirmation la retire du coût. Un rejet se corrige en confirmant à nouveau. Répéter un état n'ajoute pas d'historique. |
| `GET/POST /api/projects/{id}/rules` | Lecture / création idempotente des critères actifs : `{kind,tag_id?|domain_id?|match_text?,from_date,to_date}`. Maximum 50 règles actives. |
| `DELETE /api/projects/{id}/rules/{rule_id}` | Désactive la règle ; les décisions passées restent intactes. |
| `POST /api/projects/{id}/suggestions` | Recherche explicite pour un projet actif ; `{inserted}`. Sérialisée avec éditions et décisions. |
| `GET /api/projects/{id}/history?offset=0` | 50 événements par page ; métadonnées et décisions, snapshot bancaire minimal des décisions. |

GET inconnus/propriétaire étranger : 404. Entrées invalides : 400. Mutations d'un projet archivé : 409 (les métadonnées permettent de le réactiver). Erreurs SQL : 500 sans faux résultat vide. Les listes vides réussies restent des listes vides.

## Règles et imports

Critères déterministes : tag possédé exact, catégorie possédée exacte (sans descendants implicites), marchand normalisé exact sans casse, sous-chaîne littérale sans casse dans le libellé bancaire. Chaque règle exige une période explicite de 0 à 3660 jours. Plusieurs correspondances produisent un seul lien, justifié par la règle d'ID minimal. Aucune règle générale de marchand ne confirme automatiquement une opération.

Le worker durable intelligence existant recherche les suggestions à partir des transactions persistées après les imports/mutations réussis. Les erreurs de worker sont réessayées sans annuler l'import. Les rejets et retraits restent exclus de tous les recalculs, même si une nouvelle règle correspond ; une confirmation explicite peut les corriger. Aucun nouveau service ou fournisseur IA.

Supprimer un tag ou une catégorie supprime ses règles dépendantes (l'historique et les raisons des décisions conservent leurs critères). Les confirmations explicites ne dépendent plus de l'existence d'un tag. Supprimer une source confirmée est interdit au niveau API et PostgreSQL jusqu'au retrait des confirmations. Après retrait, le lien peut être supprimé en cascade, mais l'historique bancaire de décision reste conservé.

## Observations et couverture

Deux observations prioritaires : dépassement strictement positif du budget ; concentration de la catégorie dominante à au moins 40 % du coût, avec au moins trois dépenses confirmées. Seuils déterministes dans `projectObservations`, sources par projet/état/période/catégorie. IDs stables par projet, type et date limite. Ni anomalie « inattendue » ni tendance ne sont inférées sans référence comparable.

Sans début explicite, le coût reste provisoire. Avec dates, la couverture commune des relevés vérifie tous les comptes du propriétaire, les soldes documentaires et les transactions non attribuées. Une période courante ou mal documentée reste provisoire. `completed` ne certifie rien. Même une période documentée ne prouve pas l'exhaustivité des associations du projet.

L'Assistant réutilise son orchestrateur et `get_project_analysis` : liste limitée des projets puis lecture du même rapport propriétaire avec `project_id` et `month`. Il ne change aucun calcul. Les observations UI fonctionnent sans IA ; noms et descriptions sont explicitement traités comme données non fiables. Aucun appel LLM au chargement.

## Migration et limites

031 est additive : aucune conversion automatique des hypothèses, aucune écriture bancaire modifiée. Les anciens dossiers restent accessibles via leurs endpoints et la vue par tag. Reprise volontaire : créer un projet, règle sur le tag, puis valider les suggestions. Le rollback 031 refuse toute base contenant un projet ; effectuer les rollbacks dans l'ordre inverse des migrations, avant 027 dont l'index d'identité est partagé.

Limites volontaires : pas d'allocation partielle, de net, de prévision de dépenses futures, de causalité, de tendance non comparable, de confirmation automatique ou de notification hors application. Recherche manuelle bornée au mois choisi, 20 résultats avec invitation à affiner. Les observations apparaissent directement au détail, sans appel IA. Pas de nouvelle mémoire ou de service externe.
