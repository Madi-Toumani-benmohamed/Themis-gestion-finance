# Audit des données financières — 24 octobre 2026

## Conclusion et portée

Le critère « tout chiffre est exact, traçable et contextualisé » **n'est pas encore satisfait**. L'examen statique du pipeline et les premiers correctifs ci-dessous ne constituent pas une certification des données déjà importées. Aucune donnée de production n'a été consultée, corrigée ou supprimée. Les jeux de test sont synthétiques et isolés. Les fichiers `.gitignore` et `AGENTS.md` déjà modifiés/non suivis sont préservés.

## Parcours et diagnostic priorisé

| Priorité | Étape / preuve dans le code | Anomalie et conséquence | Traitement |
|---|---|---|---|
| P0 | `parser/csv.go`, `readCSV`, `ParseCSV` | Erreurs/lignes ignorées : import partiel sans échec ; date impossible acceptée ; crédit ignoré si débit simultané | Corrigé : refus atomique du CSV, numéro de ligne sans données bancaires dans l'erreur ; dates calendaires vérifiées ; double montant refusé |
| P0 | `models/models.go`, API, `memory.BuildFinancialFacts`, widgets | `float64` et `Number` pour les montants : calculs dérivés approximatifs malgré `NUMERIC(12,2)` en base | Partiel : centimes entiers pour validation/réconciliation et chaîne décimale pour insertion des transactions ; migration des calculs API et frontend reste requise |
| P0 | `parser/local.go:TxHash`, `handlers.go:persist`, migration 010 | Clé `(owner_id,hash)` sans compte ni référence bancaire ; même date/libellé/montant fusionnés | Protection : refuse les empreintes répétées dans un import et les collisions intercomptes ou de provenance inconnue ; sérialisation des imports par propriétaire. Risque résiduel : deux opérations réellement distinctes mais identiques dans deux fichiers du même compte restent indiscernables |
| P0 | `parser/local.go:Validate`, `reconcile` | Tolérance de presque deux centimes et somme flottante pouvaient valider un relevé faux | Corrigé : égalité exacte en centimes, précision excessive refusée ; aucune revalidation des relevés anciens |
| P1 | `api/period.go` | Dernière transaction proche de fin de mois ou absence de transactions ⇒ mois « complet » ; densité de transactions utilisée pour comparer | Corrigé : couverture documentaire par compte via `loadReliability` ; états explicites `complete/partial/unknown` ; comparaison exige des relevés couvrant les deux mois |
| P1 | `api/handlers.go:summary` | LIMIT appliqué avant borne mensuelle : navigation vers un ancien mois pouvait supprimer ses données | Corrigé : borne dans SQL avant LIMIT ; catégories alignées sur les mois de la série |
| P1 | `api/pulse.go` | Déficit provisoire qualifié négativement ; comparaison catégorie sur historique de couverture quelconque | Corrigé : qualification du déficit réservée au mois documenté terminé ; comparaison catégorie suspendue |
| P1 | `parser/csv.go:findHeader` | Plusieurs dates : la date valeur pouvait devenir une colonne montant ; perte de la seconde date | Protection : colonnes date exclues des montants ; dates différentes refusées explicitement. Stockage de deux dates encore à concevoir |
| P1 | `api/chat.go:get_data_quality`, injection mémoire | Complétude déduite du nombre de transactions ; faits KG potentiellement périmés injectés sans hiérarchie explicite | Heuristique de complétude retirée ; mémoire explicitement indicative ; consignes de vérification transactionnelle et citations renforcées. Pas de garantie déterministe sur les réponses générées |
| P1 | `api/remaining.go` | Projection de solde du dernier relevé d'un seul compte + delta global multi-comptes ; date non bornée au mois | Ouvert : ne pas assimiler cette projection au solde bancaire courant ; calcul par compte et date nécessaire |
| P1 | API : appels `Scan`/`QueryRow` ignorés dans hero, remaining, analytics, chat, journal | Erreur SQL transformée en zéro ou résultat incomplet | Partiel : propagation dans les nouvelles lectures de période/comparaison ; autres lectures à corriger |
| P1 | `parser/llmfallback.go`, `local.go:reconcile` | Équilibre comptable ne prouve pas fidélité à la source ; sélection de la première solution OCR équilibrée | Ouvert : conserver les alternatives et la provenance, ne pas confondre réconciliation et vérification humaine |
| P1 | `api/transfer.go`, classification et SQL par signe | Catégorie transfert exclue ; tout crédit restant devient revenu, remboursements inclus | Ouvert : crédits bruts ≠ salaires/revenus économiques ; règles de remboursement et appariement transfert à définir explicitement |
| P2 | `summary`, widgets, `getComparableMonths` | Mois sans transaction absent de la série ; widgets pouvant utiliser zéro ou une autre ligne ; mois documenté vide non inclus dans historique | Ouvert : calendrier explicite et distinction zéro documenté / absence de données nécessaires |
| P2 | `.github/workflows/ci.yml` | Tests PostgreSQL existants ignorés sans variable de connexion | Corrigé : service PostgreSQL de test et variable dédiés dans le job backend |

## Plan de correction et statut

1. **Lot A implémenté** : parsing CSV sans perte silencieuse, réconciliation au centime, refus des ambiguïtés d'import, insertion décimale exacte ; tests unitaires et imports PostgreSQL existants.
2. **Lot B implémenté** : statut documentaire partagé, comparaisons prudentes, correction du filtre avant LIMIT, prudence du pulse et de la mémoire IA ; tests de statut, CI PostgreSQL.
3. **Lot C requis, P0** : remplacer tous les montants des modèles et des API par un type décimal/centimes (y compris soldes et budgets), JSON numérique compatible sérialisé exactement ; supprimer toute arithmétique monétaire `float64`/`Number`. Tests de réconciliation base → chaque endpoint → présentation. Pas de changement global mécanique : risque de modifier les contrats et comportements validés.
4. **Lot D requis, P0/P1** : identité bancaire et provenance par occurrence, compte et fichier ; conservation des deux dates ; identité du document et des associations document/transaction. Migration additive réversible, lecture compatible des anciennes empreintes, rapport des ambiguïtés ; jamais de réécriture automatique des opérations existantes.
5. **Lot E requis, P1** : règles de remboursements/transferts, source et preuve de couverture distinctes de la simple validation comptable ; projections par compte ; propagation de toutes erreurs SQL.
6. **Lot F requis, P1/P2** : statuts consommés par tous les widgets et endpoints analytics/journal/narrative, calendrier vide explicite ; contrôle déterministe des chiffres et références des réponses IA.

## Invariants et définitions

- Montant EUR exact en centimes, limite actuelle NUMERIC(12,2). Une précision supérieure au centime est une erreur, jamais un arrondi implicite. Le nouveau package `internal/money` centralise cette règle pour les chemins corrigés.
- Réconciliation documentaire : `solde initial + somme algébrique de toutes les opérations = solde final`, transferts inclus. Soldes absents ⇒ validation impossible. Un relevé équilibré peut toujours être incomplet ou mal transcrit.
- Flux importés : `crédits bruts = SUM(amount > 0)`, `débits bruts = -SUM(amount < 0)`, hors catégories `is_transfer`. `net = crédits bruts - débits bruts`. Le net n'est ni un solde bancaire ni une épargne certaine. Un remboursement positif reste actuellement dans les crédits ; il n'est pas identifié comme salaire.
- Période complète : mois historique documenté continûment par relevés validés pour tous les comptes connus. Cette preuve utilise les métadonnées actuelles ; elle n'authentifie pas le document et ne détecte pas un compte jamais déclaré.
- Période partielle : mois en cours documenté à date, ou présence de relevés avec couverture insuffisante/non validée. Période inconnue : aucune preuve documentaire exploitable. `status` ancien reste compatible ; `completeness` distingue les trois états.
- `last_tx_date` est une date d'opération observée, jamais une borne certaine de couverture. `days_observed` courant est un nombre de jours écoulés ; le ratio zéro sans preuve n'affirme pas zéro transaction. Toujours lire `completeness` et les intervalles du centre de fiabilité.
- Le versement du salaire fin de mois et la disponibilité des relevés le 21 ne permettent aucune imputation de montant ni de date d'opération. Deux cycles de relevés peuvent être nécessaires pour documenter un mois civil.
- Comparaisons : exigent une couverture documentaire des deux mois ; une densité de transactions n'est pas une preuve. Les comparaisons du mois courant sont conservativement indisponibles sans preuve du mois entier. L'absence d'historique comparable n'est pas une variation de zéro.
- Import identique d'un même compte : aucun nouvel enregistrement transactionnel si les empreintes correspondent. L'API supprime le relevé créé si aucune transaction n'a été insérée ; cette suppression peut perdre une preuve documentaire et reste à réviser. L'identité documentaire n'est pas résolue. Les empreintes historiques sont conservées.
- Mémoire IA : contexte indicatif pouvant être périmé. Les outils PostgreSQL sont la référence transactionnelle. Toute réponse doit identifier outil et dates ; une hypothèse ou projection doit être annoncée. Les consignes seules ne garantissent pas la conformité du modèle.

## Catalogue des indicateurs exposés

Les périodes sont des mois civils selon `tx_date`, sauf mention contraire. La source est `transactions`, limitée au propriétaire authentifié. Tous les agrégats actuels doivent être lus comme **flux importés**, avec les limites monétaires et de couverture ci-dessus.

| Indicateur / endpoints | Source et formule | Période et validité | Limite d'interprétation |
|---|---|---|---|
| Revenus, dépenses, net — summary, hero, pulse, outils chat | Sommes par signe hors transferts ; net = crédits − débits | Mois demandé ; comparaison seulement si documenté | Remboursement assimilé au crédit ; pas un solde bancaire ; API encore flottantes |
| Dépenses par catégorie — summary, analytics, chat, widgets | Somme des débits regroupés par domaine ; certaines vues utilisent somme signée | Période filtrée dans chaque requête | Écarts possibles avec crédits de remboursement ; audit d'alignement encore requis |
| Variation historique — period/compare, pulse | `(valeur − moyenne) / abs(moyenne) × 100` ; moyenne de mois candidats documentés | Borne au jour demandé, six mois retenus maximum | Indisponible sans couverture ; les mois sans opération sont encore absents des candidats |
| Synthèse et tonalité — hero | Règles sur flux et signaux ; `heroDiagnosis` réserve le bilan aux mois terminés documentés | Mois demandé | Alertes réelles conservées sur mois provisoire ; pas de diagnostic de solvabilité |
| Budgets — budgets, hero, widgets | Débits importés du domaine / plafond ; dépassement = consommation − plafond | Mois sélectionné, plafond mensuel | Plafond normatif utilisateur, pas une dépense prévue certaine ; couverture des widgets à généraliser |
| Solde/flux historiques — widgets | Séries income/expenses et différences calculées côté React | Mois de chaque point | Flux ≠ solde ; calculs `Number` non exacts ; pas de calendrier explicite des données absentes |
| Reste à vivre — remaining | Historique : net ; courant : dernier solde + delta − engagements détectés | Mois choisi / dernier relevé | Projection, pas un solde certain ; multi-comptes/date encore non fiabilisés |
| Récurrences/anomalies — recurring, anomalies, discoveries, journal | Agrégats marchands, répétitions, seuils heuristiques | Fenêtres propres aux détecteurs ; attribution historique partiellement filtrée | Hypothèses statistiques, pas preuves de prélèvements futurs ni de fraude |
| Couverture — period, reliability, hero | Union des intervalles des relevés validés, par compte ; trous documentaires | Mois demandé (courant jusqu'à aujourd'hui dans reliability) | Validation comptable ≠ exhaustivité authentifiée ; compte inconnu non détectable |
| Narration/assistant — narrative, chat, KG | Outils SQL et faits mémorisés | Dates explicites nécessaires | Provenance au niveau transactionnel et vérification déterministe encore incomplètes |

## Validation et limites des tests

Tests ajoutés : montants exacts et limites, un centime d'écart, solde manquant, CSV avec ligne invalide après ligne valide, date impossible, débit/crédit simultanés, dates comptables différentes refusées, empreintes stables et opérations similaires distinctes, remboursement positif, mois vide non documenté, mois vide documenté, mois avant salaire, compte non couvert.

La CI exécute les tests Go avec `-race`, les tests Node des modèles et le build React ; PostgreSQL de test active désormais les tests d'import (rejeu, isolation, concurrence, confirmation) et de fiabilité existants. Ces tests ne constituent pas encore une réconciliation de tous les endpoints et widgets. La couverture de virements internes appariés, remboursements économiques et erreurs SQL généralisées reste à construire avec les lots C–F.

## Résultats de validation locale

- Tests Go unitaires exécutés ; tests PostgreSQL exécutés sur un conteneur éphémère PostgreSQL 16 sans montage de données de production.
- Tests Node : 16 réussis ; build React réussi (avertissement de taille de bundle préexistant).
- `go test ./...`, intégration PostgreSQL comprise, `go vet ./...` et build du serveur : réussis sur l’état final. Une exécution `-race` avec CGO/GCC a réussi avant les derniers ajouts ; la suivante a révélé une fixture `validated` nullable, corrigée pour reproduire la contrainte de production. Le dernier passage intégral réussi est sans `-race` ; la CI conserve ce contrôle.
- Test de réconciliation ajouté à l'intégration : total SQL des opérations importées, puis revenus/dépenses et catégories de `/api/summary` pour janvier 2020 malgré des opérations plus récentes. Ce test valide la source du dashboard, pas tous ses calculs React.
- Aucun changement de schéma : aucune migration ou reprise de données nécessaire pour ces lots.
