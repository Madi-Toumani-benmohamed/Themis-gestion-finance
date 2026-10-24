# Audit et refonte Budgets — 24 octobre 2026

## Diagnostic initial vérifié avant modification

Audit statique coordonné du backend, de la finance et du frontend. Aucun accès aux opérations bancaires réelles ; les constats ci-dessous sont établis à partir du code. L'hypothèse d'un doublon dans les données de production n'a pas été vérifiée et n'est pas présentée comme un fait.

| Priorité | Anomalie établie et cause | Correctif prévu |
|---|---|---|
| P0 | `features.go:upsertBudget` accepte une catégorie sans vérifier son propriétaire ; `listBudgets` joint le nom du domaine sans cloisonnement. | Contrôle de propriété à la création/modification et dans les jointures ; tests interutilisateurs. |
| P1 | Projection uniforme `dépense × jours_du_mois / jour_actuel`, indépendamment des relevés disponibles. | Retirer cette extrapolation ; conserver un champ de compatibilité `projected:null`, jamais une projection certaine. |
| P1 | Budget sur domaine exact, alors que le filtre Transactions inclut les enfants. | Affectation unique à l'enveloppe la plus précise ; filtre de consultation identique, sans double comptage parent/enfant. |
| P1 | Transferts et `label_kind=savings` non exclus des budgets ; calcul du Hero dupliqué. | Service de calcul partagé, distinction des mouvements internes, consommation brute explicitement définie. |
| P1 | Montants Go flottants ; saisies sous-centimes arrondies implicitement par PostgreSQL. | Centimes exacts via `internal/money`, entrées décimales validées et JSON numérique exact. |
| P1 | Tous les crédits ignorés, sans attribut fiable de remboursement/annulation. | Exposer les crédits séparément ; ne pas supposer qu'un crédit annule un débit ni que toute entrée est un revenu. |
| P1 | Mois invalide remplacé par le mois actuel ; regex acceptant un mois impossible ; pas de complétude. | Validation calendaire, mois explicite et réutilisation de la couverture documentaire existante. |
| P1 | `Budgets.jsx` considère `ratio >= 1` comme dépassement ; le widget utilise `>1`. | Un plafond exactement atteint n'est pas dépassé ; statut textuel commun. |
| P1 | Réponses asynchrones de mois successifs non protégées ; chargement/échec assimilables à une liste vide. | Protection contre réponses obsolètes, chargement/erreur/vide distincts et reprise explicite. |
| P1 | `recurring()` ne reçoit pas le mois ; moteur borné à `now()-12months`, sans borne supérieure. | Nouvelle vue déterministe des récurrences, bornée au mois choisi, sans opérations ultérieures. |
| P1 | `Regular` ne mesure que la stabilité du montant (<15 %) ; pas d'intervalles ou de couverture. DAB et versements PEL peuvent être assimilés à des abonnements. | Nature économique séparée de cadence, variabilité et preuve ; règles conservatrices, correction humaine traçable. |
| P1 | `AvgAmount=sum/nombre_transactions` additionné comme montant mensuel par Budgets et SubsWidget ; lignes affichant parfois le dernier montant. | Séparer moyenne par occurrence, médiane habituelle et dernier montant ; aucun total mensuel inventé. |
| P1 | Groupe de récurrence par marchand seul ; nature prise sur la première ligne, catégorie sur la dernière ; ordre par date seul. | Groupes par compte/marchand/périmètre économique, ordre déterministe et sources accessibles. |
| P2 | Création seule, modification par suppression/recréation ; suppression sans confirmation ; formulaire permanent sans labels. | Actions contextuelles et dialogue accessible, édition du plafond, suppression confirmée. |
| P2 | Mois courant remplacé par dernier mois avec transactions ; mois vides difficilement accessibles. | Mois courant initial explicite, navigation mensuelle indépendante de la présence d'opérations. |
| P2 | Branche 85–100 % historique sans texte de disponible ; absence de synthèse globale. | Synthèse réconciliable et état textuel pour chaque enveloppe. |
| P2 | Erreurs de curseur non vérifiées ; suppression de budget absent indiquée réussie. | Propagation des erreurs, statuts HTTP adaptés, tests isolés. |

Le SQL initial des budgets n'effectue pas de jointure avec les tags : aucun double comptage structurel par tags n'est établi. L'unicité propriétaire/catégorie existe déjà. La déduplication de l'import n'est pas remplacée par une suppression heuristique de transactions similaires.

## Organisation en trois lots reviewables

1. **Fiabilisation métier et audit** : contrats additifs, service budgétaire exact, propriété, période, périmètre des sources, couverture, tests backend ; base `origin/main`, indépendante du simulateur non encore fusionné.
2. **Refonte de l'interface** : synthèse, enveloppes contextuelles, navigation vers sources, états et accessibilité, tests frontend. S'appuie sur le contrat du lot 1.
3. **Récurrences** : moteur de revue déterministe et mois de référence, natures économiques, moyennes explicables, confirmation/correction/ignorance en métadonnées séparées et migration réversible ; tests de faux positifs et isolation. La page ne doit pas déguiser l'ancienne heuristique en information vérifiée durant cette transition.

Les limites historiques d'annulation/remboursement et de déduplication sont conservées explicitement. Aucun objectif d'épargne n'est inventé : les anciennes enveloppes de transfert sont séparées des enveloppes de dépenses ; les vrais objectifs nécessiteraient un modèle métier spécifique.
