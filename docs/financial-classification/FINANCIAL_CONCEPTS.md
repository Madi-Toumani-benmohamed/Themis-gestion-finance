# Référentiel financier — sprint 0

21 novembre 2026. Proposition métier à valider, pas un nouveau contrat SQL/API. Références : [audit initial](../CLASSIFICATION_AUDIT.md), [corpus synthétique](FINANCIAL_TEST_CASES.md), [matrice des indicateurs](AGGREGATION_MATRIX.md). `V` signifie vérifié par lecture du code ; `P` proposition ; `ND` non mesuré. Aucun résultat de production n’est revendiqué.

## Périmètre vérifié et évolution depuis l’audit

Les consignes `AGENTS.md`, le graphe `CODEBASE_GRAPH.md` et l’audit ont été relus. Le checkout de ce sprint reste `feat/agent-natural-dialogue`, commit `3d944f55779bfd8447feaa7156005ed29bb82d53`. L’audit initial avait examiné `dd198abe53c89b3eb2f972f3988ed86de8067c27` (contenu de PR #67). La comparaison Git entre ces snapshots montre des évolutions du dialogue, du calendrier, des observations/briefing et de la console Agent. Les fichiers du moteur de règles, normaliseur, import, transactions, budgets, analyses, simulation, remboursements et KG examinés ici ne diffèrent pas entre ces deux snapshots. L’ajout du briefing dans `agent_workspace.go` ne modifie pas la validation des actions de classification. On ne prétend pas que le checkout est la dernière version distante : aucun fetch ou déploiement n’est nécessaire au sprint 0.

La référence locale `origin/main` pointe vers `d1bdec7a91d2719ccf67f755a8fef68dd46f0190` ; son contenu est identique au snapshot `dd198ab` de l’audit (diff Git vide). Aucun changement supplémentaire n’est donc constaté dans cette référence locale depuis l’audit ; cela ne constitue pas une interrogation de la branche distante en temps réel.

L’audit initial est conservé intact. Il décrit une observation statique ; ses propositions n’étaient pas une autorisation d’implémenter B. Aucune correction de calcul, nouvelle nature persistée ou mesure réelle n’est effectuée ici.

## Dictionnaire et invariants

| Concept | Définition et invariant proposés | Traitement actuel vérifié |
|---|---|---|
| Mouvement bancaire | Preuve datée et signée d’un mouvement d’un compte, liée à une source documentaire. Classification ne change jamais montant, date, brut, occurrence ni compte. | `transactions`, compte via `statements.account_id`, provenance 026 ; les occurrences légitimes identiques restent distinctes. Le PATCH de classification ne modifie pas le mouvement. |
| Nature économique | Interprétation d’une opération ou d’une portion : `expense`, `income`, `transfer`, `asset_operation`, `unclassified`. Indépendante du signe bancaire et du domaine. | Aucun enum général stocké. Type financier calculé à la lecture via exclusion et signe. Ne pas introduire ces valeurs en base à ce sprint. |
| `expense` | Consommation/coût : bien ou service consommé, intérêt ou frais. Achat d’actif et principal de dette ne sont pas automatiquement consommation. | Débits ordinaires utilisés comme approximation ; sémantique plus large que cette définition. |
| `income` | Accroissement économique disponible tel salaire, intérêt ou dividende identifié ; distinct d’un encaissement de capital ou d’emprunt. | Crédits ordinaires observés, sans preuve systématique de leur nature. |
| `transfer` | Déplacement entre comptes/réserves d’un même propriétaire, neutre pour revenu et consommation consolidés. Frais éventuels séparés. | Domaine/parent `is_transfer` ou `label_kind=savings` exclut des flux ordinaires. Pas de paire de mouvements structurée prouvant le transfert. |
| `asset_operation` | Acquisition, cession ou transformation d’actif/passif ; principal reçu/remboursé relève de la variation de dette. Sous-type nécessaire pour expliquer le cas. | Compte bourse, domaine Épargne et Investissements et positions manuelles ne forment pas un registre d’actifs/dettes lié aux mouvements. |
| `unclassified` | Information insuffisante ou décision volontaire de ne pas qualifier. Ne prouve aucune dépense/revenu économique nul. | `domain_id=NULL` et provenance `unknown` sont distincts, mais ne constituent pas une nature. Un choix manuel nul peut être repris par une règle. |
| Domaine | Fonction du coût/revenu : logement, alimentation, transport, santé… Ni bénéficiaire, ni réserve, ni périodicité. Un mouvement mixte peut nécessiter des domaines par portion. | Une seule FK `domain_id`, racine/enfant ; alias `/api/categories`. Certains domaines de contexte/patrimoine mélangent ces axes. |
| Tag | Contexte libre facultatif pour recherche ; plusieurs tags sont possibles et leurs totaux peuvent se chevaucher. | `tags` + `transaction_tags`, aucune allocation de montant par tag. Un tag remboursement/transfert ne pilote pas l’agrégation. |
| Comportement budgétaire | Fixe/prévisible, variable ou indéterminé ; indépendant de la nature. Une dépense fixe peut varier dans son montant exact. | Pas d’attribut général indépendant ; enveloppes et revue de récurrences constituent des signaux distincts. |
| Récurrence | Observation de répétition avec période/dispersion et couverture explicites. Ne démontre ni engagement futur ni contrat. | `recurring_review.go` et revues persistées, règles de regroupement ; couverture et exclusions à conserver. |
| Abonnement | Relation contractuelle/payante confirmée, pouvant être périodique ou non. Cinq repas de cantine par semaine ne suffisent pas. | Récurrences et catégories/tags peuvent suggérer une lecture ; aucune preuve générale de contrat par fréquence seule. |
| Crédit/revenu observé | Crédit bancaire signé positif, éventuellement après exclusions annoncées. Dire « crédit observé » si la qualification économique manque. | Plusieurs champs API sont nommés income bien qu’ils totalisent les crédits ordinaires. |
| Revenu économique | Partie qualifiée `income`, sans remboursement de dépense, capital restitué ni dette contractée. | Pas garanti par le signe ; fiscalité calculée séparément sur bases explicites. |
| Dépense brute | Débit entier observé dans un périmètre annoncé ; ne soustrait pas implicitement un crédit. | Budgets/rapports de dépenses utilisent surtout des débits bruts ordinaires. |
| Coût net supporté | Coût brut éligible moins remboursements explicitement alloués, à date et dans une vue économique. Ne modifie pas les lignes bancaires. | Allocations de remboursement en centimes et coût net au détail ; pas de réduction générale des agrégats du MVP. |
| Transfert confirmé/supposé | Confirmé par données de propriété/contrepartie et décision traçable, avec possibilité d’une jambe non importée ; supposé par motif/signaux seulement. | Drapeau de domaine et heuristique de normalisation ne portent pas ce degré de preuve. Une seule jambe ne prouve ni revenu ni anomalie. |
| Épargne/désépargne | Déplacement de réserve vers/depuis un compte détenu, pas consommation/revenu. À distinguer des intérêts de cette réserve. | PEL/Livret A/LDD/LDDS reconnus textuellement et exclus dans les deux sens ; couverture incomplète et ambiguïté sur libellés d’intérêts. |
| Apport investissement | Transfert de trésorerie vers une enveloppe détenue, pas achat d’actif en soi. | Compte et domaine possibles ; pas de chaîne structurée apport→achat. |
| Acquisition/cession | Achat transforme trésorerie en actif ; cession restitue capital et résultat éventuel. Résultat inconnu sans prix de revient. | Flux bancaires représentables, pas ventilation produit de cession/capital/résultat garantie. |
| Principal/intérêts/frais | Principal modifie la dette ; intérêts/frais sont des coûts. Somme des portions doit égaler l’échéance bancaire. | Paiements de `credit_payments` déclarés, sans lien obligatoire à une ligne importée ni ventilation générique. |
| Remboursement partiel/total | Crédit qualifié relié à une ou plusieurs dépenses ; allocations bornées par crédit et coûts éligibles, révocables et sourcées. Reliquat non rapproché explicite. | Tables `reimbursement_cases`, `reimbursement_allocations`, `reimbursement_history` ; limites/verrous/idempotence prévus. Ne pas confondre « paiement de dette » et remboursement marchand. |
| Transaction mixte/allocation | Une source bancaire, plusieurs portions économiques dont la somme exacte égale le mouvement. Part inconnue visible et non inventée. | Domaine unique ; allocations spécialisées de remboursement et liens projets ne sont pas un découpage multi-domaines générique. |
| Date bancaire | Date de transaction observée, date de valeur distincte lorsqu’elle existe. | `tx_date`, `value_date`, source du relevé ; bornes de période selon endpoints. |
| Date économique | Date d’imputation d’un coût/revenu, distincte de l’encaissement lorsque la convention l’exige. Toujours afficher convention et date de calcul. | Pas de date économique générique ; choix budgétaire de remboursement encore à valider. |
| Manquant/inconnu/zéro | Manquant : champ/source absent ; inconnu : état non établi ; zéro : mesure connue égale à zéro. Ne pas remplacer l’un par l’autre. | Rapports récents distinguent couverture/inconnus, certains endpoints legacy conservent COALESCE/erreurs ignorées. Une ligne de montant zéro n’est pas une classification connue. |
| Provenance | Source et version de la décision, acteur, instant, preuve et ancien état. | `classification_source` courant, historique spécialisé Agent/remboursements ; pas de journal complet de chaque changement manuel/règle/tag. |
| Confiance | Qualité de la preuve ou estimation du modèle, séparée de l’autorité d’appliquer. | Suggestions IA non vérifiées, pas de score uniforme par axe. Un score élevé ne vaut pas validation. |
| Validation | Décision autorisée explicite sur le périmètre proposé ; différente de la validation documentaire d’un relevé. | PATCH manuel, validation Agent et allocations ; `statements.validated` vérifie le document, pas la justesse de ses domaines. |

## Conventions de calcul à valider

Les montants de preuve sont signés : débit négatif, crédit positif. Dans une vue de flux, D = somme des opposés des montants négatifs (positive), C = somme des montants positifs, N = C − D. Ces trois nombres n’établissent pas à eux seuls consommation/revenu/patrimoine. L’outil du sprint mesure ces **flux bancaires bruts**, sans appliquer une nouvelle politique économique.

Dans la cible proposée : revenu économique = portions `income` qualifiées ; coût = portions `expense` qualifiées après convention explicite de remboursement ; transferts internes exclus des deux. Il ne faut ni enlever de la trésorerie les mouvements réels de transfert, ni les additionner au revenu consolidé. Positions par compte et total des flux ne sont pas interchangeables, surtout si tous les comptes ne sont pas documentés.

Un apport PEA de 500 € puis un achat ETF de 500 € représente 500 € acquis, pas 1 000 €. Une dette reçue de 1 000 € augmente trésorerie et passif, pas le revenu. Une échéance de 300 € avec preuves de 250 € de principal et 50 € d’intérêts représente un seul débit de 300 €, une dette réduite de 250 € et un coût de 50 €. Un paiement déclaré ne s’ajoute pas au débit correspondant. Ces règles ne sont pas des calculs fiscaux.

Une vente de 600 € sans prix de revient : trésorerie +600 €, résultat **inconnu**, pas revenu de 600 € ni résultat zéro. Un remboursement de 40 € sur une dépense de 100 € donne 60 € de coût à date si allocation confirmée ; le débit et le crédit restent inchangés. Ne pas simultanément réduire la dépense et ajouter le remboursement au revenu économique.

## Limite d’un enum unique

Un mouvement peut contenir principal, intérêt et frais ; un achat peut relever de plusieurs domaines. Même une nature unique correcte ne décrit pas ses portions. Préférer ultérieurement une nature principale explicable et des allocations vérifiées pour les cas mixtes, ou des qualifications de portions, sans multiplier les mouvements bancaires. Décider du modèle au sprint approprié, sans l’implémenter ici. Un remboursement peut être un ajustement de coût lié à sa source plutôt qu’une nature isolée : cette décision reste ouverte.

## Décisions humaines restantes

Valider B comme direction sans migration immédiate ; lexique visible pour les crédits ordinaires ; mois économique des remboursements et révision à date ; principal de dette sous-type patrimonial ; caisse espèces et transferts à une jambe ; statut « non qualifié volontaire » ; récurrence vs abonnement ; degré de preuve requis pour automatisation. L’absence de donnée ne doit jamais être arbitrée par un classement forcé.
