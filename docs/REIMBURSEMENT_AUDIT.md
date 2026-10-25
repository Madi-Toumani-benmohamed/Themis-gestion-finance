# Audit — remboursements et flux financiers

Date de l’audit : 25 octobre 2026. Inspection du code et de la documentation du dépôt sur `main`. Aucune donnée bancaire réelle n’a été consultée. Aucun changement applicatif, schéma ou base n’est effectué dans cette mission.

## Résumé

Thémis conserve les opérations bancaires telles qu’importées, avec un montant signé exact en base (`NUMERIC(12,2)`, négatif pour un débit, positif pour un crédit). Les domaines organisent le contexte et portent le booléen `is_transfer`; ce booléen, y compris celui du domaine parent, est le mécanisme partagé d’exclusion des transferts internes des flux économiques. Les tags ajoutent du contexte mais ne déclenchent pas le traitement financier.

Les remboursements ne sont pas représentés comme tels aujourd’hui. Un crédit ordinaire, qu’il s’agisse d’un salaire, d’un remboursement commercial ou d’une participation d’un ami, est agrégé avec les crédits/revenus. Il n’existe ni relation entre ce crédit et la dépense compensée, ni part remboursée, ni état « non rapproché ». Les budgets suivent les débits de l’enveloppe et exposent les crédits séparément sans les déduire de la consommation.

La réconciliation bancaire, elle, additionne les écritures brutes. La future fonction doit donc annoter et relier des opérations, sans toucher à leurs montants, dates, relevés ou empreintes. L’approche la plus compatible est une relation de remboursement confirmée avec montant alloué, complétée par une qualification explicite du crédit lorsque l’utilisateur le déclare comme remboursement. Aucun nouveau domaine `Remboursement` n’est recommandé.

## Modèle et traitement actuels

### Transactions, domaines, tags et règles

- `backend/internal/db/migrations/001_init.sql` établit le signe bancaire et rattache les transactions au relevé; les migrations ultérieures ajoutent `owner_id`, le domaine, la provenance de catégorie, l’affichage et la traçabilité documentaire. La transaction demeure la source des montants et dates.
- `backend/internal/db/migrations/013_transfer_category.sql` ajoute `domains.is_transfer`. `backend/internal/financialsql/predicates.go` centralise le périmètre ordinaire : un domaine de transfert ou son parent est exclu, ainsi que les mouvements normalisés `label_kind='savings'`. `backend/internal/api/transfer.go` expose ce même prédicat à la plupart des handlers historiques. Le domaine continue à être la décision de transfert; les tags n’interviennent pas dans ce filtre.
- Les règles de `backend/internal/api/transaction_rules.go` affectent un domaine et des tags, mais aucun type « remboursement ». L’utilisateur peut corriger le domaine et les tags par `PATCH /api/transactions/{id}`; ce PATCH ne modifie pas le montant, la date, le libellé ou la provenance brute.
- `transaction_suggestions` sépare les suggestions IA de l’état confirmé. Ce mécanisme de proposition puis confirmation est réutilisable pour des rapprochements candidats, mais n’est pas un registre de remboursement.

### Où le transfert est exclu, et où l’écriture reste présente

| Module | Traitement actuel des transferts internes | Conséquence pour un remboursement |
|---|---|---|
| Réconciliation de relevé | `statement_controls.go` rapproche le solde d’ouverture avec la somme de toutes les opérations brutes et le solde de clôture. Aucun filtre de domaine n’est appliqué à ce calcul. | Les crédits et débits liés restent dans les soldes bancaires exacts, quelle que soit leur qualification économique. |
| Solde projeté courant | `remaining.go` part du solde de clôture bancaire puis ajoute toutes les opérations postérieures, transferts inclus. | Un remboursement reçu améliore le solde réellement projeté. |
| Vue Transactions | Le rapport expose dépenses/crédits ordinaires et débits/crédits de transfert séparément. Le filtre de transfert ne retire pas les écritures de la liste; les crédits ordinaires sont explicitement présentés comme susceptibles d’inclure remboursements/annulations. | Les sources brutes restent inspectables; le crédit de remboursement est actuellement dans le total de revenus/crédits ordinaires. |
| Dashboard, pulse, Hero, Journal et séries mensuelles | Les flux globaux utilisent le prédicat partagé pour exclure transferts et épargne normalisée. `summary.go`, `pulse.go`, `hero.go` et `journal.go` somment toutefois tous les crédits ordinaires par signe. Exceptions de descendant constatées : le signal de hausse de catégorie dans `hero.go` ne teste que `d.is_transfer`; `insights.go` exclut les seuls IDs marqués et ne teste pas les parents; `TopWidget` compare le domaine transactionnel à la liste des domaines racines (`listCategories`) et peut manquer un enfant d’un transfert. | Les remboursements augmentent les revenus/crédits du mois où ils sont reçus et peuvent modifier net, tonalité et comparaisons. Les trois exceptions peuvent faire réapparaître un sous-domaine de transfert dans un repérage/top de dépense. Le widget « Solde net cumulé » est un cumul de flux économiques, pas le solde bancaire de tous les comptes. |
| Analyse | `analysis_report.go` distingue débits, crédits ordinaires, impôts, épargne et autres transferts. Les dépenses par catégorie portent sur les débits; les crédits ordinaires sont observés séparément et inclus dans `net = income - expenses`. Les repérages de tendance marchand/tag sont calculés sur les débits. | Un remboursement n’abaisse pas le coût de la dépense d’origine; son crédit peut être lu comme revenu. Les signaux dépenses utilisent donc le montant brut débité. |
| Budgets | `budgets.go` attribue chaque transaction à une enveloppe par domaine et ne compte comme consommé que les débits. Les crédits d’une enveloppe sont accumulés dans `credit_total` à titre informatif; ils ne réduisent pas `spent`. Les transferts ont une section distincte. | Un remboursement ne rend pas aujourd’hui de plafond disponible, même quand le crédit est classé dans le même domaine que l’achat. |
| Simulations | `simulation.go` prend les crédits positifs ordinaires comme revenu observé moyen et les débits ordinaires comme dépenses par catégorie sur trois mois. | Un remboursement ponctuel peut gonfler la référence de revenu et faire paraître le scénario récurrent plus favorable. La dépense d’origine reste brute. |
| Découvertes/récurrences | `analysis_discoveries.go` et le détecteur de récurrence focalisent leurs signaux sur des débits; les crédits ne deviennent pas des abonnements. Des découvertes de baisse/hausse de crédits ou d’habitudes de consommation n’ont toutefois aucune sémantique de remboursement. | Une dépense avancée puis remboursée peut rester présentée comme dépense personnelle complète; le crédit peut fausser les indicateurs d’entrées. |

Le « transfert interne » est donc une exclusion des agrégats économiques, pas une suppression de la transaction ou du mouvement de trésorerie. Les agrégats centraux et budgets utilisent le prédicat parent/enfant partagé; l’exclusion dans tous les signaux n’est pas parfaitement uniforme, comme le montrent les trois exceptions du tableau. Toute nouvelle vue remboursement doit utiliser le prédicat central et le plan d’implémentation doit inclure un correctif de parité/test des consommateurs dérogatoires, sans en faire un nouveau type de remboursement. Les tags ne remplacent pas le mécanisme de transfert. Le remboursement devra ajouter une relation/qualification distincte et préserver le domaine des transactions.

## Composants réutilisables

- Le registre de transactions, l’identité propriétaire, les montants en centimes (`internal/money`) et les mises à jour atomiques propriétaire-scoped fournissent la base des relations exactes.
- `transaction_suggestions` et le motif de revue des récurrences montrent comment séparer proposition et décision confirmée.
- `transaction_sources.go`, `transactions.go` et l’Analyse savent déjà naviguer vers un ensemble d’identifiants sources. Ce mécanisme peut servir à ouvrir les dépenses et crédits liés.
- Les vues de relevé conservent les sommes de flux bruts et les événements de validation. Les remboursements n’ont pas à modifier ces contrôles.
- Les rapports d’Analyse et de Budget ont des instantanés cohérents en lecture seule, des montants exacts et des états de couverture. Une extension devrait réutiliser ces couches plutôt que recalculer les règles dans React.
- L’historique transactionnel de crédits (`credit_payments`, annulation tracée, idempotence, intégrité propriétaire) offre des conventions utiles, mais ces paiements déclarés représentent une dette modélisée, pas des lignes de transactions bancaires ni des remboursements de consommation.

## Lacunes et risques établis

1. Aucun lien N:M entre transaction créditrice et dépenses d’origine; aucune valeur restante à rembourser ou par dépense.
2. Le signe ne distingue pas revenu, remboursement, retour de capital, intérêt, avance ou transfert. Une règle de catégorie ou un tag ne fournit pas cette preuve.
3. Les crédits positifs alimentent les revenus du Dashboard, de l’Analyse et de la simulation tant qu’ils ne sont pas exclus comme transferts/épargne.
4. Un remboursement tardif apparaît dans le mois de trésorerie correct; aucune vue ne peut actuellement montrer son effet sur le coût du mois d’achat sans réécrire ou attribuer la dépense.
5. Les budgets n’affectent pas les crédits à la dépense initiale; changer ce comportement sur le mois du crédit produirait une logique temporelle différente de celle du coût de la dépense.
6. Des transferts personnels vers un ami ou un prêt accordé à un tiers ne sont pas des transferts internes. Réutiliser `is_transfer` pour les soustraire masquerait des flux réels et mélangerait les notions.
7. Un remboursement sans dépense source peut être une annulation d’un import incomplet, un retour d’achat ancien, une restitution d’avance ou un crédit inexpliqué. Il ne faut ni créer une dépense fictive ni forcer une relation.
8. Une relation ne doit pas dépasser le crédit disponible ni le montant de la dépense restante; l’application concurrente de deux remboursements exige verrouillage et recomptage dans une transaction.
9. La suppression d’un relevé supprime actuellement ses transactions par cascade. Une future relation exige une règle explicite : bloquer la suppression tant que des liens sont actifs, ou demander leur révocation tracée avant la suppression. Une cascade silencieuse détruirait la preuve du rapprochement.
10. Les écritures de dette dans `credit_payments` sont suivies séparément des opérations importées. Ajouter les deux à une même somme de dépenses pourrait doubler les mensualités; la part capital/intérêts n’est pas encore une qualification de transaction garantie.
11. Trois consommateurs périphériques contournent le prédicat partagé parent/enfant (`hero.go` pour une hausse de catégorie, `insights.go` pour un pic, `TopWidget` avec une liste des seuls domaines racines). Ils peuvent inclure un domaine enfant de transfert dans un signal de dépense. Il faut couvrir cette parité avant d’étendre les agrégats aux remboursements.

## Effets fonctionnels à prévoir

- **Trésorerie et soldes bancaires :** inchangés et toujours basés sur toutes les lignes signées. Un crédit remboursant un achat reste une entrée bancaire observée.
- **Revenu économique :** les allocations de remboursement confirmées ne sont pas du revenu. Les crédits non qualifiés doivent rester nommés « crédits observés/non rapprochés »; ils ne prouvent pas seuls un salaire.
- **Dépense supportée :** débit de la dépense d’origine moins remboursements confirmés affectés, au centime près. La somme remboursée ne peut être comptée qu’une fois.
- **Budget :** décision à expliciter. Recommandation : faire réduire la consommation de l’enveloppe de la dépense d’origine, pas créer du disponible dans le mois où l’argent revient. Montrer séparément le crédit en trésorerie dans son mois réel et signaler qu’un ancien mois budgétaire a été révisé.
- **Simulations :** isoler les crédits de remboursement/retour de capital des revenus récurrents; le scénario doit utiliser les dépenses nettes confirmées sur sa période de référence, avec la couverture de cette période.
- **Découvertes :** calculer les habitudes sur les dépenses nettes confirmées pour les analyses personnelles, mais garder une navigation vers les débits bruts, crédits et allocations qui justifient le montant.

Les rapports d’origine devront être « à date » : une allocation confirmée en octobre peut réviser le coût économique de septembre, avec une date de calcul explicite. Elle ne change ni le relevé de septembre, ni ses opérations, ni son solde.

## Conclusion d’audit

L’architecture dispose d’un périmètre fiable pour distinguer les transferts internes et de primitives exactes pour les transactions, budgets, périodes et preuves. Elle ne possède pas de sémantique de remboursement. Un nouveau domaine `is_transfer` n’est pas adapté : il exclurait un crédit du revenu mais n’associerait pas le crédit aux débits, n’autoriserait pas l’allocation partielle et pourrait faire apparaître un remboursement comme mouvement interne. Un tag ne convient pas non plus puisque les tags ne pilotent pas les calculs.

La conception détaillée est dans [REIMBURSEMENT_DESIGN.md](REIMBURSEMENT_DESIGN.md), le plan de lots dans [REIMBURSEMENT_ROADMAP.md](REIMBURSEMENT_ROADMAP.md). La principale décision produit restante concerne le traitement budgétaire et la présentation des crédits non qualifiés; aucune classification historique automatique n’est justifiée par les données actuelles.
