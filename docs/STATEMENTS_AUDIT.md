# Relevés — audit et conception

25 octobre 2026, branche `feat/statements-documentary-hardening`, worktree courant. La configuration IA a été fusionnée par PR #32 (`86705e4`), après CI et sécurité réussies. `AGENTS.md` préexistant reste intact. Aucun merge automatique de cette mission.

## Constats démontrés

| Priorité | Constat | Cause / traitement retenu |
|---|---|---|
| P0 | Le dénominateur documentaire inclut livrets et placements sans relevés. | `reliability.go` et `hero_diagnosis.go` incluent tous les comptes. Ajouter un mode explicite par compte, initialisé selon le type métier existant (`courant` documentaire, autres manuels), modifiable sans changer les soldes. |
| P0 | `validated` représente la réconciliation du parsing, pas nécessairement celle des opérations persistées. | `persist` déduplique après validation ; un import partiellement dédupliqué peut conserver un badge positif incohérent. Réconciliation exacte SQL/centimes sur les sources conservées ; aucune correction des montants pour équilibrer. |
| P1 | Le tableau n’indique ni compte ni causes des contrôles ; suppression visuellement dominante. | Étendre le tableau existant, contrôles déterministes, détail accessible, filtres et confirmation explicite. |
| P1 | Pas de trace de validation manuelle, pas de transition disponible. | Ajouter un historique atomique propriétaire, autoriser une confirmation documentée seulement si les prérequis déterministes passent ; réouverture possible. Aucun statut d’erreur de parsing ou d’archive inventé : ces ressources ne sont pas persistées actuellement. |
| P1 | Les périodes ne contrôlent pas chevauchements/continuité des soldes. | Réutiliser les intervalles inclusifs existants, distinguer borne partagée, chevauchement et trou ; continuité financière seulement entre périodes consécutives ou partageant une borne. |
| P1 | Suppression en cascade réelle, sans contrôle des liens incohérents ni invalidation des anciennes analyses. | Vérifier propriétaire et dépendances, tracer les métadonnées supprimées, préserver l’historique, supprimer les analyses mises en cache devenues obsolètes atomiquement. |
| P2 | Requêtes de liste non annulables, erreur bloquante sans reprise, absence de contrôles détaillés. | États séparés, relance et rejet des réponses obsolètes ; réutiliser les composants et tokens. |

## Mécanismes conservés

- `GET/DELETE /api/statements`, import direct et aperçu/confirmation existants ; déduplication `(owner_id, hash)` et refus des collisions entre comptes.
- Un import entièrement dupliqué est déjà retiré par `finishStatementImport` : pas de nouvelle écriture vide à exposer comme document fiable. Un relevé réellement sans opération doit pouvoir être conservé.
- `parser.Validate` utilise déjà des centimes exacts. L’aperçu (`describeImport`) contient encore une somme flottante : à remplacer au même périmètre.
- `uncoveredStatementDates` couvre déjà cycles au 21, intervalles superposés et borne actuelle ; pas de remplacement par une heuristique de calendrier ou de dernière transaction.
- Transactions provisoires restent consultables et incluses comme observations ; la couverture explique leur niveau de preuve. Les pointages manuels restent dans Comptes.

## Invariants et conception

1. Le mode `statements` est soumis à couverture ; `manual` ne l’est pas. Aucun compte nommé en dur, plusieurs comptes documentaires possibles. Aucun compte soumis = couverture inconnue/non applicable, jamais « complète » par défaut.
2. Les dates de relevé sont inclusives. Deux relevés partageant seulement une borne ne créent ni trou ni preuve de doublon. Un trou indique absence de document, pas de transactions. Les jours futurs ne sont pas comptés.
3. `écart = clôture − (ouverture + somme signée des opérations)` en NUMERIC/centimes. Valeur inconnue reste nulle, aucun zéro de remplacement pour un solde absent.
4. Réconciliation, validation déclarée, couverture et soupçons sont distincts. Une égalité ne prouve pas l’extraction exhaustive ; aucune similarité ne déclenche fusion/suppression.
5. La validation manuelle exige une note et des contrôles préalables réussis. Modification et preuve sont atomiques et limitées au propriétaire. Aucun contournement d’un écart comptable.
6. Les montants/libellés historiques ne changent pas. Une ancienne validation devenue incohérente reste traçable mais ne prouve plus la couverture.

Migration additive implémentée : `025_statement_controls.sql`, mode de suivi, historique de validation et index de rattachement des opérations ; rollback gardé pour ne perdre aucune nouvelle déclaration. Les fixtures de tests sont adaptées au schéma, sans toucher aux données réelles. Les changements hors Relevés sont limités aux modes de Comptes, aux services de couverture/provenance partagés, à l’export de l’historique, à l’aperçu exact et au gestionnaire de dialogue réutilisé.

## Bilan

Les correctifs sont implémentés. Les anciennes données financières et anciens flags `validated` ne sont pas réécrits. Un indicateur effectif recalculé distingue la preuve documentaire actuelle de la déclaration historique. Les opérations provisoires restent consultables. Aucun appel LLM, aucune transaction de valorisation fictive et aucune validation déduite du salaire ou du signe des flux.

La page réutilise les détails repliables de couverture plutôt qu’une nouvelle frise graphique : les intervalles datés, les trous et les relevés liés restent lisibles et accessibles, sans dépendance ni score de qualité arbitraire. Les fonctionnalités utiles de l’import et les pointages restent disponibles.

Limites et résultats détaillés : `STATEMENTS_CHANGES.md` et `STATEMENTS_TESTS.md`. Aucun commit, PR, merge ou déploiement automatique de ce chantier ; la migration n’a été appliquée qu’aux schémas de test isolés.
