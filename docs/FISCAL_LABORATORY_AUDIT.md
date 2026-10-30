# Fiscalité et Laboratoire — audit et décisions

Base : `86d1b951930647cc0bf8f473121828f74a2b7e9a`. Branche `feat/fiscal-laboratory`. État initial propre ; aucune donnée bancaire réelle consultée. AGENTS et graphe consultés.

## Existant réutilisé

Simulation utilise `/api/simulate` et un modèle React en centimes/BigInt, avec référence documentaire, états indisponibles et scénarios non persistés. Crédits & taxes utilise `/api/obligations`, des déclarations/paiements exacts et leur historique : aucun calcul d'impôt sur les revenus. Ces contrats ne changent pas. `internal/money.Cents`, `money.mjs`, le dialogue natif/focus et les tokens visuels sont réutilisés.

L'authentification transmet le propriétaire par contexte. Les nouvelles routes restent derrière ce middleware. Les documents du Laboratoire n'ont aucun lien de mutation avec transactions, budgets ou obligations. L'export propriétaire inclura les documents ; aucune donnée n'est envoyée à une IA.

## Choix de bibliothèque

Versions npm vérifiées le 30 octobre 2026 : ReactGrid 4.1.17 (MIT, peers React 16/17/18), Univer core 1.3.1 (Apache-2.0), mathjs 15.2.0 (Apache-2.0). Projet Thémis sous GPLv3 ; licences compatibles, notices conservées.

| Solution | Fonctions et intégration | Décision |
|---|---|---|
| ReactGrid | Grille React 18, navigation/sélection ; aucun moteur de formules. Ajoute une UI et nécessite encore un moteur/adapter décimal. | Écarté pour la V1 bornée ; utile si la grille devient plus ambitieuse. |
| Univer | SDK tableur riche, plugins/versions synchronisés, UI propre et modules additionnels ; intégration React documentée. Coût d'intégration supérieur pour un petit outil libre. | Écarté ; pas de collaboration/Excel/import XLSX demandés dans cette V1. |
| mathjs + grille HTML bornée | Parseur/moteur maintenu, BigNumber décimal, AST inspectable, séparation nette de React. Adapter limité pour références/plages/cycles ; pas de moteur Excel réécrit. | Retenu. Bundle partiel et Worker, import à la demande. |

Sources primaires : https://github.com/silevis/reactgrid ; https://docs.univer.ai/guides/sheets/getting-started/installation ; https://github.com/dream-num/univer/blob/dev/LICENSE ; https://mathjs.org/docs/expressions/security.html ; https://mathjs.org/docs/custom_bundling.html . Aucune performance relative chiffrée prétendue avant mesure du build/du moteur local. Vérifier la liste des dépendances et notices installées dans le rapport final.

## Architecture retenue

A. Moteur Go pur `internal/fiscal`, règles identifiées par année de revenus/année d'imposition/version/statut, API POST sans persistance et UI Fiscalité séparée dans Simulation. Base salariale net fiscal avant frais ou revenu net imposable final explicitement distincts. Montants en centimes à l'entrée ; rationnels et arrondis légaux à l'euro dans le calcul.

Périmètre V1 : revenus 2024/imposition 2025 et revenus 2025/imposition 2026 vérifiés. Résidents métropolitains de moins de 65 ans, salaires ordinaires, enfants mineurs à charge exclusive, cas général ou parent isolé, frais forfaitaires ou réels qualifiés. Rejet explicite des situations particulières, veuvage avec enfants, hauts revenus et dispositifs non implémentés. Pas de conversion arbitraire du brut. Aucune application silencieuse du barème passé aux revenus 2026. Règles/sources et cas de référence dans `FISCAL_RULES.md`.

B. Calculatrice et mini-tableur partagent mathjs/BigNumber. Expressions bornées et AST autorisé (arithmétique, comparaisons, références, fonctions limitées). Aucun accès propriétés/assignation/import/JS/réseau/fichiers. Worker interruptible. Maximum 100 lignes × 26 colonnes, 1000 cellules stockées (y compris les cellules vides avec format), 512 caractères par cellule/formule, profondeur 32, budget global d'opérations.

Documents JSONB versionnés : `laboratory_documents`, propriétaire, titre, type, contenu, révision et timestamps. PUT/DELETE conditionnés à la révision ; conflits visibles sans effacer le brouillon. Sauvegarde serveur explicite ; récupération temporaire dans sessionStorage liée à l'utilisateur et effacement à la déconnexion, jamais une sauvegarde par frappe en base. Fermeture de l'onglet termine cette récupération locale. Conversion conserve expression, résultat, opérandes extractibles et historique ; aucune réduction de feuille destructive.

## Risques et validation

Règles fiscales datées, cas officiels et tests aux frontières avant qualification fiable. Conventions numériques et fonctions documentées ; erreurs isolées par cellule, cycles et limites testés. Tests PostgreSQL synthétiques à deux propriétaires et conflits de révision. Tests Node, navigateur, Axe, chargement différé, build et non-régression des pages existantes. Aucun commit/push/merge/déploiement automatique pour cette mission.
