# Audit — Intelligence financière v2

Date de l’audit : 2026-10-29. Périmètre : backend, mémoire, migrations, Dashboard/Analyse/Assistant et tests existants. Les constats initiaux décrivent `feat/intelligence-v2-foundation`; les sections d’état d’implémentation ci-dessous consignent les lots ajoutés ensuite.

## Synthèse

Thémis possède déjà plusieurs briques fiables qu’il faut fédérer : rapports d’analyse avec couverture documentaire, montants en centimes, exclusions de transferts/épargne centralisées, revue des récurrences avec décisions utilisateur, découvertes de marchands/tags et narration IA contrainte à des faits vérifiés. Il n’existe toutefois pas de registre commun des observations, de score de pertinence, de feed priorisé persistant, ni de recalcul asynchrone déclenché par les mutations.

La première intégration doit adapter ces services existants vers un contrat Discovery commun. Il ne faut pas reconstruire les moteurs de couverture, récurrence, budgets, analyse ou l’extraction narrative.

## Cartographie existante

| Surface | Calculs et déclencheurs | Preuves / couverture | Limites à traiter |
|---|---|---|---|
| `GET /api/insights` (`insights.go`) | Relevé manquant, non catégorisées, abonnements en hausse, pic de catégorie, dépassement budget. | Selon le sous-système. | Mélange de domaines ; pic de catégorie basé sur le mois max disponible et moyenne historique, sans garde générale sur mois courant partiel ; seuil >1,5× et >50 € ; montants convertis en `float64` ; pas d’identité ni déduplication. |
| `GET /api/anomalies` (`anomalies.go`) | Réutilise `detectAnomalies` / récurrences legacy : hausse, nouveau récurrent, achat inhabituel, récurrent disparu. Exclut retraits DAB et transferts ; plancher 10 €, limites de fréquence et de variation. | Le détecteur associe période et marchands/récurrences, mais n’émet pas un modèle uniforme de preuve. Filtrage IA facultatif sur la liste. | Calculs et montants legacy en `float64`; règles de disparition/hausse codées dans le détecteur ; filtre IA qui classe, sans registry ni apprentissage de feedback. |
| `GET /api/intel/discoveries` (`discoveries.go`, `analysis_discoveries.go`) | Signaux de récurrence, hausse de prix à vérifier, services similaires, dérive marchand, hausse de tags, saisonnalité possible. | Récurrences et signaux portent des IDs de transaction ; analyses requièrent périodes documentées ; quatre mois complets pour dérive/tag ; saisonnalité est une hypothèse sur historique long. | Contrat JSON hétérogène en maps ; certains signaux se chevauchent avec anomalies/hero ; tags peuvent se recouvrir ; pas de registre commun ou priorité. |
| `GET /api/hero` (`hero.go`) | Résumé, budgets, catégorie en hausse, chute de revenu, récurrences. | `heroMonthCovered` gate les comparaisons ; mois courant incomplet ne produit pas ces variations. | Calculs d’agrégats et seuils en `float64`; crédits ordinaires incluent des remboursements non rapprochés et ne prouvent donc pas un revenu régulier ; pas de persistance/déduplication. |
| `GET /api/intel/narrative` (`narrative.go`) | LLM sélectionne/reformule un sous-ensemble de faits déterministes par identifiants. | Le serveur valide les identifiants, conserve les réserves de couverture et rejette les faits inconnus ; mode déterministe sans LLM. | Bonne frontière à réutiliser ; ne constitue pas lui-même un moteur de découverte ni un registre. |
| `GET /api/period*` (`period.go`) | Périodes observées, couverture, comparaisons de flux. | `reliability.go`, `getPeriodInfo`, `getComparableMonths`; périodes incomplètes et inconnues explicites. | Plusieurs montants de comparaison historiques sont encore exposés/calculés via `float64`; conserver ces API, remplacer progressivement leurs consommateurs analytiques par les rapports exacts. |
| Récurrences (`recurring_review.go`) | Regroupe les observations par clé stable propriétaire/compte/marchand/nature/sens ; cadence à partir d’occurrences et couverture suffisantes. | Centimes exacts, IDs/date d’opérations, snapshot répétable, vérification propriétaire ; retraits, transferts, crédits/remboursements exclus de l’abonnement présumé. | C’est la source à adapter. La clé de récurrence n’est pas l’identité d’une découverte. Décisions `detected/confirmed/ignored` déjà persistées en migration 020. |
| Budgets (`budgets.go`) | Débits et enveloppes, dépenses par catégorie. | Centimes exacts ; utilise `financialExpenseSQLT`; propriétaire et couverture conservés. | Les ratios sont des flottants, mais non des montants ; réutiliser les résultats plutôt que refaire les sommes dans un nouveau moteur. |
| Rapport d’analyse (`analysis_report.go`) | Flux, catégories, périodes et traçabilité. | Snapshot lecture seule, montants en centimes, sources et couverture explicites. | Fondation recommandée pour les faits et limitations. |

## Cohérence des faits et de la couverture

- `internal/financialsql/predicates.go` définit les exclusions communes : catégories de transfert et leurs enfants, ainsi que `label_kind=savings`, sont exclus des dépenses ordinaires. Les prédicats corrélés vérifient `owner_id` des domaines. Les crédits restent des crédits ; un signe positif seul ne prouve pas un salaire.
- Les allocations de remboursement sont exactes et auditées (migration 027), mais le coût net n’est pas encore injecté dans les agrégats analytiques. Ne pas soustraire automatiquement un crédit sans rapprochement explicite.
- `analysisReportFrom`, `reliabilityWindow`, `heroMonthCovered`, `getComparableMonths` distinguent mois complets/incomplets/inconnus. Une date calendaire ou le nombre de jours écoulés n’établit pas la complétude du relevé. Les règles métier doivent continuer à exclure une comparaison mensuelle incomplète.
- L’import fidèle (migration 026) garde provenance et preuve de parsing, mais l’exhaustivité documentaire peut rester inconnue faute de total ou de compteur indépendant. Toute découverte doit hériter cette limite.
- Historique variable selon le détecteur : 4 mois documentés pour certaines évolutions, 36 mois et au moins deux années pour hypothèse saisonnière, historique nécessaire à la récurrence. Le registre devra stocker `coverage` et `limitations` avec la version du détecteur.

## Doublons et mutualisation

Les endpoints ne sont pas tous des moteurs indépendants : anomalies et hero appellent le détecteur commun de récurrence ; `discoveries` réutilise le rapport récurrent mais dérive ensuite des signaux additionnels ; insights recalculent séparément plusieurs agrégats ; la narration sélectionne des faits de rapport. Des phénomènes similaires peuvent donc apparaître sous plusieurs titres/types entre ces surfaces. Le frontend Analyse consomme `/api/intel/discoveries`; le Dashboard ne consomme actuellement pas de feed intelligence.

À mutualiser : rapports exacts/couverture, revue de récurrence, exclusions `financialsql`, rapport budgets, faits de narration et sources de transactions. À ne pas reconstruire : classifications transferts, registre des récurrences confirmées/ignorées, contrôles documentaires et pipeline de preuves IA.

## Précision numérique

- Sources modernes : `money.Cents` (`int64`) et conversions explicites depuis `NUMERIC`; transactions rapportées en centimes, budgets, remboursements, revue récurrente et nouvelles découvertes analytiques utilisent des montants exacts.
- Sources legacy encore en `float64` monétaire : `insights.go`, `anomalies.go`/adaptateur de `features.go`, `hero.go`, `period.go` et modèle historique `models.Transaction`. Les pourcentages/ratios peuvent rester flottants s’ils sont typés comme ratios et ne servent pas de montant.
- Les premiers adaptateurs v2 doivent préférer les valeurs exactes des rapports modernes. Toute frontière legacy doit expliciter conversion et provenance, et ne pas devenir la source du feed prioritaire.

## Mémoire et assistant

- Go `internal/memory` rend Graphiti facultatif : échec ou désactivation donne une réponse sans mémoire. `SearchFacts` ajoute des faits au contexte Assistant ; les textes récupérés sont des données non fiables et doivent rester sans pouvoir d’instruction.
- L’apprentissage actuel transforme chaque transaction importée en phrase avec libellé, marchand et montant formaté en float ; c’est plus détaillé que la mémoire contextuelle souhaitée et peut persister des informations transactionnelles brutes. La mémoire ne doit pas être utilisée comme source financière ni alimenter le score comme préférence vérifiée.
- Le service Python expose `/kg/learn`, `/kg/query`, `/kg/status/{group_id}`, `/kg/clear/{group_id}`. `status`/`clear` interpolent `group_id` dans Cypher et les routes internes ne montrent pas d’authentification applicative. Le réseau compose est la frontière actuelle ; avant exposition, remplacer l’interpolation par paramètres et protéger l’accès. Ne pas fonder les préférences explicites sur Graphiti.
- L’Assistant possède des outils de recherche/synthèse propriétaire. Aucune fonction existante ne lui fournit un registre de découvertes vérifiées.

## Persistance, événements et interface

- PostgreSQL et migrations embarquées lexicalement dans `internal/db/migrations` ; migration 027 est la plus récente. Les schémas récents utilisent contraintes composées `(id, owner_id)` pour empêcher les références inter-propriétaires.
- Il existe un registre de décisions explicites pour les récurrences (020). Les migrations 028–030 ajoutent le registre versionné, le feedback générique, les préférences et une file coalescée de recalcul avec worker PostgreSQL.
- Les mutations pertinentes sont raccordées au middleware de file après un retour HTTP réussi : confirmation/suppression d’import, modification de transaction, règles, alias marchand, tags/domaines, comptes, remboursements, récurrences et réapplication des règles. Le worker utilise des baux, `SKIP LOCKED`, coalescence par propriétaire, reprise bornée et endpoint de statut. L’enqueue est best effort pour ne jamais annuler une mutation financière validée.
- Dashboard affiche synthèse, KPI et journal ; pas de section “Ce que Thémis a remarqué”. Analyse affiche les découvertes existantes. Assistant ne peut pas explorer une découverte par ID.

## Plan d’intégration recommandé

1. **Lot 0 (présent)** : conserver cet inventaire et la décision de réutilisation ; préciser les contrats communs avant migration.
2. **Lot 1** : modèle Discovery typé (métriques en centimes/fréquences, évidence et couverture), identité phénomène distincte de l’observation, score de pertinence séparé de confiance, déduplication et registry PostgreSQL owner-scoped avec historique/état. Adapter d’abord un détecteur moderne éprouvé, puis exposer un feed borné sans casser `/api/intel/discoveries`.
3. **Lot 2** : adapter les découvertes de tendance/habitudes/contributions après validation des fenêtres et preuves ; tests synthétiques multi-périodes.
4. **Lot 3** : feedback générique et préférences PostgreSQL; reprendre les décisions de récurrence comme contexte explicite, sans élargir un feedback à tout un domaine. Graphiti demeure facultatif.
5. **Lot 4** : section Dashboard ≤3 items, détails/provenances/actions, puis Assistant via faits vérifiés.
6. Ajouter une file PostgreSQL idempotente et worker après définition des événements transactionnels et des règles de stale/retire. Publier une tâche après commit des mutations; l’échec d’intelligence ne doit jamais annuler l’import. Le feed Dashboard lira le registre, jamais n’exécutera les analyses lourdes.

## Risques à vérifier durant l’implémentation

- La compatibilité de migrations et suppression de comptes avec les preuves historiques référencées.
- Les transactions historiques modifiées (alias, tags, catégories) peuvent changer l’identité calculée ; les snapshots d’observation doivent préserver l’audit et permettre une nouvelle version.
- Des listes de preuves peuvent être volumineuses : borner payloads et inclure une justification si la source est tronquée.
- Les signaux semblables doivent être dédupliqués sans fusionner des périodes/sujets différents ; aucune pertinence élevée ne compense une preuve insuffisante.
- Évaluer les scores sur des scénarios synthétiques documentés ; l’objectif 80 % n’a pas été mesuré et ne doit pas être annoncé comme atteint.

## État d’implémentation — Lot 1 (2026-10-29)

- Le contrat commun, le score déterministe et les helpers de sélection sont dans `backend/internal/intelligence/`.
- La migration additive `028_intelligence_registry.sql` ajoute les identités de phénomène, observations versionnées et historique de feedback. Son rollback refuse de supprimer des données existantes.
- `Registry.Record` est idempotent pour un même contenu/version, conserve l’identité lors d’une nouvelle observation et vérifie que chaque ID de transaction référencé appartient au propriétaire indiqué. Les observations disparues peuvent être résolues sans réactiver celles que l’utilisateur a masquées.
- `GET /api/intel/feed` lit uniquement le registre, par utilisateur, 3 éléments par défaut et 50 maximum. `GET /api/intel/discoveries/{id}` applique le même cloisonnement et fournit jusqu’à 100 observations historiques.
- `POST /api/intel/recompute` mutualise la revue des récurrences, les signaux d’analyse existants et comportementaux. Il reste disponible comme action immédiate et est aussi appelé par le worker; le travail durable est coalescé par propriétaire et n’est pas exécuté au chargement du Dashboard. Le calcul est sérialisé par propriétaire et borné à 500 découvertes.
- `POST /api/intel/discoveries/{id}/feedback` journalise les sept actions prévues. `hide` masque immédiatement; `unhide` met en état résolu et une preuve retrouvée par recalcul réactive le phénomène. Les retours non lifecycle alimentent maintenant les préférences du Lot 3.
- Le lot a des tests unitaires et d’intégration PostgreSQL pour identité/version, idempotence, historique, feedback, réconciliation, verrou concurrent et isolation propriétaire. Le déclenchement automatique, la file et l’historique Dashboard ont été ajoutés dans les lots suivants.

## État d’implémentation — Lot 2 (signaux comportementaux intégrés)

- `internal/intelligence/behavior.go` fournit trois détecteurs purs : hausse progressive sur exactement six mois consécutifs (quatre pas haussiers minimum, +100 € et +30 % si la base est positive), contributions entre deux fenêtres disjointes de trois mois avec rapprochement exact, et hausse de fréquence nécessitant au moins trois opérations supplémentaires, +50 % ou plus et une activité sur deux mois récents au minimum.
- `internal/api/behavior_discoveries.go` calcule des tendances par marchand/catégorie, décompose une variation des débits ordinaires par marchand, et mesure les fréquences d’achat par marchand et le week-end par catégorie. Le service travaille dans un snapshot PostgreSQL répétable, owner-scoped; il suspend tous ces signaux si l’un des six mois n’est pas entièrement documenté. Les transferts sont exclus via `financialExpenseSQLT`; les sommes sont analysées en centimes.
- Les contributions positives et négatives sont rapprochées aux sommes des six mois. La carte garde trois contributeurs positifs au plus et une contribution résiduelle nette, avec les preuves transactionnelles des marchands contributeurs référencées (limite de registre : 500 IDs par observation).
- Limites : la fréquence par catégorie mesure le week-end; six mois ne suffisent pas à distinguer une saisonnalité; les remboursements non rapprochés restent séparés des débits observés. Les nouveaux signaux sont consultables via le Dashboard.
- Tests unitaires couvrent tendances, variation réconciliée, fréquence durable ou ponctuelle et déduplication. `TestBehaviorDiscoveriesPostgres` valide six relevés raccordés, le refus d’un mois non validé et l’exclusion d’un transfert interne.

## État d’implémentation — Lot 3 (préférences explicites)

- La migration additive `029_intelligence_preferences.sql` conserve le dernier retour révisable par découverte avec une note explicite bornée; l’historique complet reste dans `intelligence_feedback`. Le rollback refuse de supprimer des préférences existantes.
- `GET/PATCH /api/intel/preferences` expose les préférences du propriétaire, permet de corriger une action et une note, ou de supprimer le choix sans effacer l’historique du feedback.
- Au recalcul, `interesting`, `already_knew`, `irrelevant`, `normal` et `remind_me` ajustent uniquement la pertinence; la confiance, les métriques et les preuves ne changent pas. Un feedback `irrelevant` atténue aussi les signaux de même type avec une pénalité bornée, sans désactiver le détecteur. Graphiti n’est pas requis.
- Les tests d’intégration PostgreSQL vérifient la persistance, l’édition et la suppression d’une préférence, ainsi que la pénalité spécifique et celle, bornée, appliquée à un phénomène similaire.

## État d’implémentation — Lot 4 (Dashboard et Assistant)

- `IntelligenceHighlights` présente au plus trois découvertes du feed sur le Dashboard, avec période, qualité des preuves, métriques, couverture, limites, liens vers les transactions et actions de feedback. Le recalcul automatique est suivi via `GET /api/intel/jobs/status`; le bouton explicite continue de déclencher un recalcul immédiat.
- Le détail à la demande affiche l’historique versionné des observations. Les tendances, variations de dépenses et fréquences montrent une comparaison graphique accompagnée d’un tableau accessible contenant les valeurs exactes.
- Le panneau permet de corriger ou supprimer les préférences apprises. Ses contrôles restent utilisables au clavier et les styles respectent le mouvement réduit.
- L’outil `get_verified_discoveries` permet à l’Assistant de lire le même feed vérifié, avec métriques, preuves, couverture et limites. Le prompt précise que les textes de relevés, marchands, notes et mémoire sont des données non fiables.
- Limites restantes : le déclenchement dépend d’une liste explicite de routes mutation; les modifications introduites par un nouveau endpoint devront être ajoutées à cette liste. Le traitement est asynchrone dans le même processus API (plusieurs instances restent protégées par les verrous PostgreSQL); une file dédiée séparée n’est pas prévue. Les graphiques couvrent les comparaisons comportementales courantes, les autres métriques restent lisibles sous forme de valeurs typées et de tableau.
- Vérification finale de cette étape : `go test ./... -count=1` et `go vet ./...` passent avec PostgreSQL 16 isolé; `npm run build` passe. Vite conserve un avertissement de bundle JavaScript supérieur à 500 kB.
