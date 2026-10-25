# Conception — remboursements dans Thémis

## Décision proposée

Retenir un modèle hybride minimal :

1. **La transaction de crédit reste l’écriture bancaire originale.** Un utilisateur peut explicitement la qualifier de remboursement reçu; cela ne transforme pas son montant en revenu.
2. **Un dossier de remboursement** garde cette décision, sa nature et son état de rapprochement (non rapproché, partiel, rapproché ou à examiner).
3. **Des allocations exactes** relient ce crédit à une ou plusieurs transactions de dépense, pour un montant au centime. Plusieurs dossiers peuvent compenser la même dépense.
4. Le domaine conserve sa fonction actuelle : catégoriser la nature de l’opération et déterminer, via `is_transfer`, si un flux est un transfert interne. Les tags restent descriptifs. Un remboursement n’est pas un nouveau domaine de transfert.

La relation confirmée est la preuve du remboursement. Une ressemblance, une règle ou l’IA peuvent seulement suggérer le lien. Les transactions brutes, les montants, dates, relevés, soldes et identifiants ne sont jamais réécrits.

## Comparaison des options

| Option | Avantages | Limites métier et techniques | Décision |
|---|---|---|---|
| **A — Domaine « Remboursement »** | Facile à choisir, filtrer et associer à une règle; réutilise l’interface de domaines. | `is_transfer` est un booléen d’exclusion des flux internes : lui faire porter aussi les remboursements confond deux réalités. Un domaine ne relie pas le crédit à une dépense, ne porte pas de montant partiel, ne permet pas le N:M, n’explique pas un excédent et n’assure pas le plafond remboursable. Marquer le crédit en transfert l’exclurait sans calculer le coût net. | **Rejeter** comme mécanisme financier. Conserver le domaine d’origine (restaurant, transport, etc.). |
| **B — Relations transaction↔transaction uniquement** | Traçabilité claire, opérations intactes, 1:1 / 1:N / N:1 par allocations; calcul précis de reste. | Une relation seule ne sait pas distinguer un remboursement d’un remboursement de prêt ou d’une restitution de capital; ne représente pas un crédit connu mais sans dépense source; un montant entier ne couvre pas les crédits mixtes; elle ne précise pas la politique des résidus et du revenu économique. | **Insuffisant seul**; bon cœur de rapprochement. |
| **C — Dossier/qualification + allocations** | Distingue la nature déclarée du lien et son état; traite les crédits non rapprochés, remboursements partiels/multiples, commercial vs personnel, excédent en revue; peut être étendu aux avances sans falsifier le domaine. | Ajoute un concept et des tables; nécessite atomiques, verrouillage et audit; migration additive. La qualification d’un crédit ne doit pas être inférée silencieusement. | **Recommander**, en limitant le MVP aux remboursements d’une dépense déjà présente et aux retours commerciaux. |

La qualification est nécessaire et ne concurrence pas `is_transfer` : elle répond à « que représente ce crédit / quel montant compense quelle dépense? », tandis que le domaine répond à la catégorisation et que `is_transfer` retire les mouvements entre comptes propres des flux économiques. Une qualification ne peut jamais changer automatiquement le domaine d’une transaction.

### Prérequis : parité des filtres transfert

Avant de comparer ou d’enrichir des indicateurs, aligner les trois vues périphériques recensées dans l’audit (`hero.go` hausse de catégorie, `insights.go` pic, `TopWidget`) sur la sémantique parent/enfant de `financialsql.ExpenseT`. Ce travail ne change pas la définition du transfert; il évite qu’un sous-domaine marqué par un domaine parent apparaisse dans une découverte ou un top de dépense, alors qu’il est déjà exclu des agrégats centraux. Ajouter des tests synthétiques pour un transfert parent avec transaction classée sur son enfant. Les transactions restent visibles et présentes dans la réconciliation brute.

## Modèle de données conceptuel

Proposition additive, à confirmer avant toute migration.

### Dossier de remboursement

`reimbursement_cases` : un dossier est rattaché à un crédit bancaire positif déjà importé.

- `id`, `owner_id`, `credit_transaction_id` (unique par propriétaire), `kind` (par exemple `personal_expense` ou `merchant_refund`), `status` (`unmatched`, `partial`, `allocated`, `needs_review`, `voided`), note facultative, créateur et horodatages.
- `kind` doit être explicitement confirmé. Le dossier classe le crédit entier comme crédit de remboursement ou retour commercial, même si son allocation reste partielle. Son résidu ouvert n’est donc pas traité en revenu.
- Ne pas créer de dossier automatiquement à l’import. Un crédit ordinaire qui n’est pas qualifié garde le statut historique/« crédit non qualifié ».

### Allocations de remboursement

`reimbursement_allocations` : chaque ligne porte un montant positif en centimes, le dossier source et la transaction de dépense cible.

- `id`, `owner_id`, `case_id`, `expense_transaction_id`, `amount_cents > 0`, créateur et date de confirmation, `voided_at`/motif pour annuler sans effacer l’historique.
- Plusieurs allocations peuvent partir d’un crédit vers plusieurs dépenses (1:N); plusieurs crédits peuvent viser une dépense (N:1); le N:M est la combinaison des deux. Une relation 1:1 est simplement un cas particulier.
- La somme des allocations actives d’un dossier ne dépasse pas le crédit. La somme affectée à une dépense ne dépasse pas son débit négatif moins les allocations déjà actives. Un solde non alloué demeure visible comme à rapprocher; aucun montant de transaction n’est modifié.
- Le montant disponible de la dépense est l’absolu du débit admissible, moins les allocations actives. Ce modèle autorise un remboursement partiel et calcule le reste exact.

### Intégrité et audit

- Chaque dossier est lié à une transaction existante positive (`amount > 0`); chaque cible doit être négative, même propriétaire, non nulle, et admissible selon le périmètre convenu (dépense ordinaire, hors transfert/épargne). Les domaines/tags ne prouvent pas à eux seuls l’éligibilité.
- Prévoir des clés étrangères composites `(transaction_id, owner_id)` (avec contrainte/index unique composite côté transactions) afin que PostgreSQL empêche un rattachement inter-propriétaire. Pas uniquement une vérification au niveau React.
- La somme de plusieurs lignes ne se garantit pas par un simple `CHECK`. L’API verrouille le crédit et les dépenses concernées, en ordre d’ID stable, recalcule les sommes actives et écrit les allocations/décisions/historique dans une transaction. Toute erreur entraîne un rollback.
- Une modification de montant d’allocation est une révocation tracée de l’ancienne ligne suivie d’une nouvelle. `reimbursement_history` conserve avant/après, acteur, horodatage et motif. Le budget et les rapports se recalculent depuis les allocations actives, sans conserver d’agrégat monétaire dupliqué.
- Ne pas supprimer silencieusement les relations par cascade quand un relevé est supprimé. Le MVP bloque la suppression d’un relevé contenant des transactions dans des dossiers/allocations actifs; l’utilisateur doit d’abord annuler explicitement ces liens. Les identifiants/snapshots nécessaires à l’audit sont conservés après révocation.

Les montants sont `BIGINT` en centimes, calculés via `internal/money`. Aucun `float64` pour valider ou additionner des allocations. Les réponses JSON suivent le contrat exact employé par les rapports existants.

## Règles de calcul proposées

Pour une dépense `D<0` et les allocations de remboursement actives `A_i>0` :

- **Débit bancaire brut** = `abs(D)` à la date de la transaction. Inaltéré.
- **Remboursement confirmé affecté** = `Σ A_i`, dans la limite de `abs(D)`.
- **Dépense nette supportée** = `abs(D) − Σ A_i`, attribuée à la date et à la catégorie de la dépense d’origine.
- **Crédit bancaire brut** = montant positif au mois de réception, inchangé dans la trésorerie.
- **Crédit classé comme remboursement** = montant brut du dossier; exclu du revenu économique. La partie non affectée reste dans le dossier et apparaît « à rapprocher / à qualifier », pas comme remboursement déjà associé à une dépense.
- **Revenus économiques confirmés** ne doivent pas assimiler silencieusement tout crédit positif à un revenu. Ajouter des postes `credits_observed`, `confirmed_income`, `reimbursements_received`, `unmatched_credits` (ou équivalents) en conservant le flux bancaire brut. La règle de compatibilité pour les anciens crédits non qualifiés reste une décision produit explicite (voir ci-dessous).
- **Solde bancaire et validation de relevé** restent calculés à partir de toutes les lignes originales signées, sans exclusion des remboursements ni déduction des relations.

Exemple par périodes : dépense `−60,00 €` en septembre, remboursement reçu `+30,00 €` en octobre. Trésorerie : septembre `−60,00 €`, octobre `+30,00 €`. Coût économique affiché à la date du calcul : septembre `30,00 €` net de remboursement confirmé; octobre n’a pas de dépense de consommation de `30,00 €`. L’écriture de septembre reste `−60,00 €`, et le solde bancaire de chaque mois reste celui du relevé. La vue économique historique doit afficher « calculée au … » et peut donc réviser le coût d’un mois antérieur.

## Traitement des cas étudiés

| Cas | Traitement proposé |
|---|---|
| **1. Remboursement partiel : dépense 60 €, ami 30 €** | Conserver `−60` et `+30`; confirmer 30 € d’allocation; sortie brute 60 €, entrée brute 30 €, coût personnel net 30 €, revenu économique issu de cette entrée 0 €. |
| **2. Remboursement total : avance de consommation 100 €, retour 100 €** | Allocation 100 € à la dépense; coût net 0 €. Deux lignes de trésorerie inchangées. |
| **3. Plusieurs remboursements : dépense 120 €, retours 40 € et 35 €** | Deux dossiers/crédits peuvent allouer 40 et 35 au même débit; coût net 45 €. Chaque dossier garde son propre état et son reliquat. |
| **4. Un crédit couvrant plusieurs dépenses** | Un dossier peut allouer, par exemple, 50 € à deux achats de 20 € et 30 €. Le débit source total et chaque crédit cible gardent leurs plafonds. Le domaine de chaque dépense reste son contexte d’origine. |
| **5. Remboursement intermensuel** | Vue trésorerie par date réelle d’écriture; vue économique par date de dépense d’origine, recalculée à une date d’observation explicite. La comparaison historique indique que les remboursements postérieurs peuvent réviser les mois antérieurs. Le budget recommandé suit aussi le mois de la dépense d’origine; le crédit réel reste affiché dans le mois de réception. |
| **6. Crédit sans dépense source présente** | L’utilisateur peut qualifier le crédit comme remboursement et créer un dossier `unmatched`, sans créer de dépense fictive. Le crédit brut reste visible; aucun coût n’est compensé jusqu’à une allocation ou une qualification différente. |
| **7. Avance ou prêt personnel de 200 €** | Ne pas appliquer le dossier de remboursement de consommation. La sortie est une créance/avance (pas une dépense de consommation); retour du principal : diminution de créance, pas revenu; intérêts : revenu économique. Un modèle d’avances/prêts à tiers est un chantier séparé, avec rapprochement éventuel vers une dépense externe seulement si celle-ci est réellement documentée. Ne pas le classer en transfert interne. |
| **8. Retour ou annulation commerciale** | Même relation comptable débit↔crédit quand l’achat source existe, mais `kind=merchant_refund`. Prioriser une suggestion déterministe par marchand/libellé, signe, montant disponible et proximité de date; jamais fusionner ni confirmer automatiquement. Un geste commercial excédant l’achat reste un résidu à examiner. |
| **9. Remboursement d’une dette/crédit** | Ne pas confondre paiement de prêt et remboursement d’une dépense. `credits`/`credit_payments` sont des déclarations d’obligation séparées des transactions importées. Une éventuelle association transaction↔versement doit être une preuve de rapprochement, non un second flux; le capital diminue la dette, les intérêts sont une charge. La ventilation capital/intérêts doit être fournie ou confirmée, pas inventée depuis une mensualité seule. |
| **Dépense payée pour le compte d’un ami, puis réglée par l’utilisateur à l’ami** | Si le débit est le paiement par l’utilisateur de sa part, il reste une dépense personnelle. S’il rend un principal prêté ou une avance, traiter comme retour d’actif futur, non comme remboursement de consommation. Le seul sens du débit ou le libellé « remboursement » ne tranche pas. |

Les remboursements effectués par l’utilisateur à un tiers n’ont pas tous le même sens. Un paiement de sa propre part d’un restaurant demeure un coût; un remboursement de capital d’une avance est une sortie d’actif. Le MVP ne doit pas déduire l’un de l’autre. Une éventuelle annulation de remboursement (l’utilisateur renvoie un crédit reçu) est une nouvelle écriture de débit, liée à une correction explicite, pas la suppression du crédit historique.

## Effets sur les interfaces et analyses

### Dashboard

Conserver les flux bruts par date pour comprendre les entrées/sorties de trésorerie; compléter par des indicateurs distincts : dépenses brutes, remboursements confirmés reçus, coût net des dépenses, crédits non qualifiés, revenus économiques confirmés. Une entrée de remboursement ne doit pas gonfler le KPI de revenu économique ni être interprétée comme revenu disponible récurrent. Les transferts `is_transfer` restent exclus de ces indicateurs et visibles dans leurs mouvements séparés.

### Transactions

Chaque ligne demeure éditable comme catégorie/tags via l’interface existante. Ajouter un accès « Rapprochements » sur une dépense ou un crédit : montrer les opérations liées, le montant imputé, le reste, l’état à rapprocher et l’origine documentaire. L’utilisateur peut confirmer, répartir, modifier par annulation tracée, ou retirer un lien. Les suggestions sont identifiées comme suggestions; la ligne bancaire et son montant restent en lecture seule.

### Analyse et comparaisons

Afficher brut, affecté et net côte à côte, avec navigation vers les transactions brutes et les allocations. Les montants nets sont par dépense d’origine, calculés à la date « connu jusqu’au ». Ne pas sommer les débits bruts et remboursements comme deux dépenses ni compter le crédit à la fois en revenu et en remboursement. Les périodes historiques où un remboursement ultérieur a révisé le coût doivent être étiquetées comme telles.

### Budgets

Recommandation : le budget suit la consommation nette du mois d’achat, c’est-à-dire débit éligible moins les remboursements confirmés rattachés à ce débit. Un crédit reçu en octobre pour un achat de septembre révise l’enveloppe de septembre; il ne crée pas une dépense négative ou un plafond disponible d’octobre. Présenter distinctement le crédit observé d’octobre. La période budgétaire peut donc évoluer après sa clôture documentaire; l’UI doit indiquer la date de recalcul et laisser consulter le brut.

### Simulations et découvertes

Le simulateur utilise le coût net par dépense pour les références de consommation et exclut les remboursements ponctuels des revenus récurrents. Il garde le montant observé brut et la couverture. Les découvertes décrivent « débit brut X, remboursement confirmé Y, coût net Z » avec les sources. Une hausse de crédit ne devient pas une hausse de revenu ou un comportement. Sans assez d’éléments, aucune conclusion net/abonnement n’est produite.

## Automatisation et IA

1. **D’abord règles déterministes, en lecture seule :** rechercher les dépenses négatives antérieures ou proches, comparer les centimes restants et le montant positif; signaux textuels du marchand/contrepartie, compte et période; ne pas supposer qu’un montant identique prouve une relation.
2. Retourner des candidats avec motifs vérifiables (« même marchand », « montant restant égal », dates à N jours) et le montant maximal proposé. Ne pas afficher de score numérique tant qu’il n’est pas calibré sur des cas synthétiques représentatifs.
3. Aucun lien automatique confirmé. Confirmation manuelle et refus explicite. Les règles confirmées peuvent servir d’indice, pas d’écriture permanente à d’autres transactions sans exception par opération.
4. IA optionnelle ensuite pour ordonner ou reformuler des candidats déjà bornés. Ne lui transmettre que les libellés nécessaires et aucun relevé/PDF complet; ne jamais lui laisser produire ou confirmer le montant, l’identité du propriétaire ou l’écriture SQL.
5. Pas de Knowledge Graph obligatoire. Les corrections de l’utilisateur priment sur toute proposition et les rapports restent disponibles sans LLM.

## Migration et préservation de l’historique

- Seulement une migration **additive** après validation de la conception; aucune modification des colonnes bancaires existantes, des relevés validés ou des empreintes.
- Aucun rapprochement rétroactif automatique. Les crédits et dépenses historiques restent non qualifiés jusqu’à une décision utilisateur. Le rapport de migration compte les crédits positifs sans en exposer les libellés et ne les transforme pas en revenu/remboursement.
- Première version en lecture/écriture des dossiers et allocations, sans backfill, sans remigration analytique des anciennes périodes. Les endpoints et vues hérités restent compatibles; de nouveaux champs analytiques sont ajoutés avant qu’une UI remplace un libellé ancien.
- Tests de rollback de migration sur schéma synthétique, intégrité multi-propriétaire, concurrence d’allocations et conservation de la somme brute avant/après.

## Décisions produit à prendre avant implémentation

1. Les crédits historiques non qualifiés restent-ils dans l’ancien indicateur de revenus/crédits pour compatibilité, tout en étant montrés comme non qualifiés, ou faut-il un nouveau KPI « revenus économiques confirmés » basé sur une qualification explicite? Recommandation : ne plus appeler « revenus » le total brut des crédits; afficher une mesure additive de crédits et un revenu confirmé séparé, sans réécrire les anciens crédits.
2. Le budget mensuel net est-il rattaché au mois d’achat (recommandé) même s’il se révise plus tard, ou au mois de réception?
3. Quelles formes de remboursements sont dans le MVP : entre particuliers et retours marchands (recommandé), ou seulement un premier cas?
4. Quel libellé et quelle action permettent de qualifier un crédit incomplet/sans source sans prétendre à une dépense d’origine présente?
5. Les avances/prêts consentis à des tiers restent-ils hors MVP (recommandé) ou faut-il dès la première version distinguer principal et intérêts?

## Limites

Une relation confirmée prouve seulement la décision enregistrée par l’utilisateur et les montants déclarés; elle ne prouve ni la provenance économique réelle du crédit ni l’absence de dépenses manquantes. Les soldes de compte et les relevés restent les références pour la trésorerie. La relation ne certifie pas la complétude des opérations d’un relevé.

## Contrat réalisé — MVP lots 1 et 2

Le MVP implémenté suit le modèle hybride ci-dessus, sans rétroactivité ni adaptation des rapports :

- Migration additive `027_reimbursements.sql`; aucun backfill. Les snapshots JSONB conservent libellés, date et montant exacts pour l’audit quand une opération est ultérieurement supprimée après révocation/annulation explicite. `027_reimbursements.sql` dans `backend/internal/db/rollbacks/` refuse le rollback si des dossiers ou événements existent.
- `POST /api/reimbursements` qualifie explicitement `{credit_transaction_id, kind, note?}`. Une seule fiche par crédit/propriétaire; la répétition avec la même qualification retourne le dossier existant, une qualification différente est en conflit.
- `GET /api/reimbursements?transaction_id=&limit=&offset=` retourne les dossiers liés à un crédit ou une dépense. `GET /api/reimbursements/{id}` lit un dossier propriétaire. `GET /api/reimbursements/eligible-expenses?case_id=&limit=&offset=` énumère les débits ordinaires avec leur montant encore remboursable.
- `POST /api/reimbursements/{id}/allocations` reçoit `{expense_transaction_id, amount_cents}` et exige l’en-tête `Idempotency-Key`. Une clé répétée pour le même propriétaire/dossier rejoue la réponse existante; son réemploi pour un autre contenu est rejeté.
- `DELETE /api/reimbursements/{id}/allocations/{allocation_id}` révoque avec `{reason?}`; `POST /api/reimbursements/{id}/void` annule le dossier avec `{reason?}` et est idempotent; `GET /api/reimbursements/{id}/history?limit=&offset=` expose les décisions tracées.
- Les écritures/historiques sont atomiques et propriétaires. Clés étrangères composites interdisent les références inter-propriétaires. Les opérations concernées sont verrouillées en ordre d’identifiant; les plafonds crédit et dépense sont recalculés en centimes dans la transaction. PostgreSQL et l’API refusent les signes incorrects, zéro, transferts directs/descendants et épargne.
- La suppression d’un relevé contenant un lien actif retourne un conflit; il faut révoquer ou annuler explicitement avant suppression. Une fois les références révoquées, les snapshots et l’historique restent consultables.
- L’interface de Transactions ne qualifie aucun ancien crédit sans action; elle montre brut, alloué, restant et liens. Le coût net au détail d’une dépense est explicitement indicatif et n’affecte pas Dashboard, Analyse, Budgets ou Simulations dans le MVP.

Les limites de pagination sont de 1 à 100 dossiers par page. Les associations transactionnelles sont limitées à une requête par `Idempotency-Key`; l’application n’expose pas encore d’endpoint de remplacement en une seule action (le changement est révocation tracée puis nouvelle allocation). Les crédits mixtes, excédents et avances personnelles restent hors périmètre.
