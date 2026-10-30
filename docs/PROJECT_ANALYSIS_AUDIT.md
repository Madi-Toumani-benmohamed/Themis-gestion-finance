# Projets financiers — audit du 30 octobre 2026

Base : `7a28d795930b1f34201f04e5a071247c7949811a`. Audit du code uniquement, aucune donnée bancaire réelle.

## Architecture et constats vérifiés

`TagFolders.jsx` lit `/api/tag-folders` et son détail. Ce sont des vues des tags des débits ordinaires, jusqu'au mois sélectionné. `tag_folder_prefs` conserve uniquement un choix épinglé/masqué, identifié par texte ; il n'existe ni dossier persistant ni budget ni relation projet/transaction. La détection (150 €, deux opérations et 70 % concentrés sur deux mois) reste une hypothèse.

Les défauts historiques de `docs/ANALYSIS_AUDIT.md` ne sont plus tous présents : `tag_folders.go` exige déjà le propriétaire concordant des transactions/tags, utilise DISTINCT/EXISTS, des agrégats numériques convertis en centimes, et remonte les erreurs. La concentration utilise désormais des dates d'ancrage distinctes. Les tests PostgreSQL existants couvrent ces corrections. Les sources Transactions utilisent le nom exact du tag, pas une recherche libre.

Risques restants : identité par nom mutable, agrégation implicite de toutes les opérations d'un tag, aucune distinction suggéré/confirmé/rejeté, absence de budget et de preuve de couverture propre au projet. Les totaux de différents tags se chevauchent et ne sont pas additifs. Une modification de tag peut changer un regroupement historique.

## Réutilisation et impacts

- `money.Cents`, `parseAnalysisAmount`, `analysisPercent` : montants exacts et limite JSON sûre.
- `financialsql.ExpenseT` : exclusion commune des transferts descendants et de l'épargne normalisée. Les crédits ne deviennent pas des remboursements.
- `loadReliabilityRangeFrom` : preuves documentaires ; le statut manuel du projet ne prouve pas sa complétude.
- `useAnalysisResource` : chargement/erreur et rejet des réponses obsolètes.
- File PostgreSQL intelligence existante : déclenchement après import confirmé, retries, isolation, sans dépendance LLM.
- Analyse héberge le nouveau module. Les filtres et calculs Transactions, Budgets et Simulation ne changent pas. L'Assistant reçoit un outil de lecture déterministe ; aucun calcul n'est délégué au LLM ou au Knowledge Graph.

## Architecture retenue

Une table `financial_projects` est nécessaire : faire porter identité, dates et budget à une préférence textuelle confondrait une hypothèse automatique avec un choix financier. Tables minimales liées : décisions projet/transaction (source unique des confirmations), règles bornées, historique des décisions. Clés étrangères composites par propriétaire et unicité des liens. Les agrégats sont calculés à la lecture, sans copie de totaux.

Migration additive 031 ; aucune mutation des écritures bancaires, tags ou préférences. Les vues historiques restent consultables. Leur reprise est une action utilisateur : créer un projet puis sélectionner un tag et confirmer explicitement les suggestions. Aucun projet détecté automatiquement n'est déclaré confirmé par migration. Pas d'allocation partielle ni de coût net dans cette première version : une opération peut être confirmée entièrement dans plusieurs projets, avec avertissement de non-additivité.

Les décisions rejetées/retirées ne sont pas écrasées par un recalcul. Les règles proposent uniquement ; elles ne confirment rien. L'archivage conserve les preuves. Les opérations devenues inéligibles (transfert, hors dates) restent traçables mais sont exclues du coût courant.
