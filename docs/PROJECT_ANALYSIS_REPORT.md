# Refonte des projets financiers — rapport de livraison locale

30 octobre 2026. Branche `feat/project-financial-analysis`, base `7a28d795930b1f34201f04e5a071247c7949811a`. Rapport des validations locales avant publication. Publication et fusion autorisées ensuite par l’utilisateur ; leur état final est porté par la PR et les contrôles GitHub.

## 1. Diagnostic vérifié

Les dossiers étaient des agrégats de tags, sans projet stable, budget ni décision de rattachement. Les anciens problèmes de jointures propriétaires, multiplication des agrégats et erreurs ignorées avaient déjà été corrigés dans le code de départ. Ces corrections ont été conservées et revérifiées. La mutation de préférence d'un dossier exige désormais aussi l'existence d'un tag appartenant à l'utilisateur.

Audit détaillé : `PROJECT_ANALYSIS_AUDIT.md`.

## 2. Architecture et choix

Ajout d'un modèle minimal de projet : une préférence textuelle épinglé/masqué ne pouvait pas porter une identité financière ni des confirmations traçables. Les vues historiques restent accessibles, sans conversion automatique des hypothèses. Une règle sur le tag puis une validation explicite permettent leur reprise.

Les prédicats financiers, centimes, couverture documentaire, composants graphiques, orchestration IA et file de jobs existants sont réutilisés. Aucune dépendance, nouveau service ou fournisseur externe. Les agrégats sont calculés à la lecture ; aucune copie de total à synchroniser.

## 3. Persistance

Migration additive 031 : `financial_projects` (nom, description, budget facultatif, dates, statut, timestamps), `project_rules`, `project_transactions` (source unique des liens et décisions), `project_history` (modifications et snapshots des décisions).

Unicité d'un lien par projet/transaction, FK composites par propriétaire, index de lecture et de recherche inverse. La migration ne change aucun montant, tag, préférence ou relevé existant. Le rollback refuse une base contenant un projet ; les rollbacks se font en ordre inverse, 031 avant 027 dont l'index d'identité est partagé. Tests sur schémas PostgreSQL isolés, y compris données historiques et rejeu.

## 4. Fiabilité et sécurité

- Contrôles propriétaire à chaque lecture/mutation ; tests avec deux utilisateurs et attaques par IDs de projet, transaction, tag et catégorie. FK PostgreSQL contre les liens croisés même en SQL direct.
- Calculs en centimes/rationnels, aucune arithmétique financière `float64` ; budgets absents, nuls et dépassés distincts. Un agrégat hors précision exacte échoue.
- Débits ordinaires confirmés uniquement. Exclusion partagée des transferts descendants et de l'épargne normalisée. Aucun crédit assimilé à un remboursement, aucun coût net inventé.
- Une transaction est comptée une fois dans un projet, même avec plusieurs tags/règles. Les chevauchements entre projets et tags sont explicitement non additifs.
- Décisions par lot atomiques et réversibles. Confirmations concurrentes sérialisées ; répétitions sans événements artificiels. Les rejets/retraits ne sont pas écrasés par les recalculs.
- Sources confirmées protégées contre suppression dans l'API et PostgreSQL ; après retrait, suppression permise avec conservation de l'historique bancaire de décision. La suppression complète explicite d'un utilisateur conserve sa cascade existante, vérifiée sans impact sur l'autre propriétaire.
- SQL/réseau indisponible n'apparaît jamais comme un succès à zéro. Les réponses financières sont également validées avant affichage.

## 5. Fonctionnalités livrées

Vue globale et détail dans Analyse → Pilotage, création/édition, statuts actif/terminé/archivé, budget facultatif, coût brut, reste, dépassement, taux de consommation et nombre distinct de dépenses. Dates explicites, dernier coût observé et couverture documentaire prudente.

Règles bornées sur tag, catégorie exacte, marchand normalisé exact ou texte littéral du libellé. Suggestions justifiées, exclues du coût jusqu'à validation individuelle ou par lot ; rejet, retrait, correction et recherche manuelle sur le mois choisi. Le worker durable existant recherche les nouvelles correspondances après import confirmé, sans bloquer l'import et sans LLM.

Partitions exactes par catégorie, marchand et mois ; cumul observé avec tableau accessible ; tags présentés comme chevauchants. Observations de dépassement et de concentration (au moins 40 % et trois dépenses), avec identifiants, valeurs, période, conditions et sources exactes, y compris « Sans catégorie ».

L'Assistant dispose de `get_project_analysis`, lecture du même rapport propriétaire. Les montants sont déjà formatés en euros par le backend ; aucune mutation financière ou invocation IA automatique. Les constats restent disponibles sans IA.

Contrats et seuils : `PROJECT_ANALYSIS_API.md`.

## 6. Validations réellement exécutées

| Contrôle | Résultat |
|---|---|
| Go `test -race ./...`, PostgreSQL synthétique | Réussi ; revalidations ciblées API/DB après index, couverture et sources sans catégorie également réussies. |
| Go `vet ./...` | Réussi. |
| Migration 031, historique, rejeu et rollback gardé | Réussi sur PostgreSQL isolé. |
| Modèles frontend `node --test src/components/*.test.mjs` | 16 fichiers de tests réussis. |
| 13 suites de `test:browser` | Toutes exécutées avec succès ; chaîne reprise après correction du débordement du nouveau module. Transactions, Budgets, Simulation, Analyse, Projets, Crédits, Configuration IA, Relevés, Import, Motion, Cinématique, Intelligence, chargement de production. |
| Projets responsive / Axe | Clair et sombre, 1440/768/390/320 px ; création, édition, budget absent, batch, retrait/correction, association manuelle, sources, historique et erreurs. Réussi. |
| E2E avec API, authentification et PDF réels synthétiques | 9 scénarios fonctionnels réussis, dont le nouveau parcours projet avec vérification d'isolation. |
| Références visuelles E2E | 4 échecs locaux, également reproduits sur `main` de départ : les quatre PNG produits sont strictement identiques entre refonte et baseline (SHA-256 identiques, zéro pixel différent). Aucune référence ni tolérance modifiée. |
| Build frontend et test des chunks de production | Réussis. Chargement avant connexion toujours différé ; récupération après échec de chunk vérifiée. |
| Tests Python / syntaxe Knowledge Graph | 2 tests et compilation réussis. |
| Docker Compose `config -q` | Réussi. |
| `git diff --check` | Réussi. |

Les tests ne lisent aucune donnée bancaire réelle. PostgreSQL utilise des bases/conteneurs dédiés, les tests Go des schémas privés détruits après usage. Chromium système pour les parcours synthétiques ; Chrome 156 pour les E2E. Les E2E ont utilisé une copie isolée du code pour préserver des rapports préexistants appartenant à nobody ; seuls le cache Vite et l'autorisation locale des modules/polices ont été adaptés dans cette copie.

Preuve de comparaison : `PROJECT_ANALYSIS_VISUAL_COMPARISON.md`. Les tests visuels ne sont donc pas déclarés verts localement. La CI GitHub n'a pas été déclenchée puisque la consigne interdit de pousser sans instruction. Aucun build/publication de l'image Docker n'a été exécuté pour cette mission.

## 7. Impacts sur l'existant

Calculs Transactions/Budgets/Simulation et import bancaire inchangés. Le worker ajoute une recherche de suggestions, l'export inclut les projets et leurs preuves, la suppression d'un relevé vérifie les confirmations de projet. Les vues de tags restent disponibles. Les chargements des règles/détails/historique sont à la demande ; listes et opérations paginées, agrégats SQL restreints aux projets de la page. Les tests d'export et de rollback ont été adaptés aux vraies dépendances de schéma.

## 8. Limites et risques résiduels

Pas d'allocation partielle, de coût net, de prévision future ou de confirmation automatique. Deux constats déterministes prioritaires ; aucune anomalie inattendue, causalité ou tendance ne repose sur une comparaison non établie. Pas de notification hors application : les observations sont affichées au détail.

Une période documentée ne prouve pas l'exhaustivité des rattachements. Sans début explicite ou avec des relevés incomplets, le coût reste provisoire. La migration n'a été appliquée qu'aux bases de test, jamais à une instance utilisateur.

L'explication libre d'un fournisseur LLM n'a pas été testée avec un service réel : son adaptateur de lecture, ses erreurs, ses montants et son isolation l'ont été sans appel externe. Les contrôles CI distants et le build de l'image restent à réaliser après publication autorisée.

## 9. Suites recommandées

Après revue, commit/PR et CI avec le navigateur de référence. Tester ensuite le volume réel sur une base de staging synthétique représentative. Des tendances comparables ou une allocation partielle pourront être ajoutées séparément lorsqu'un besoin et des preuves suffisantes le justifient. Aucun besoin de nouveau framework ni service de mémoire.

État de reprise et prochaine action : `PROJECT_ANALYSIS_PROGRESS.md`.
