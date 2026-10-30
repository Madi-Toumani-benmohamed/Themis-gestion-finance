# Livraison locale — moteur fiscal et Laboratoire

Branche : `feat/fiscal-laboratory`, issue de `86d1b951930647cc0bf8f473121828f74a2b7e9a`. Lors de la livraison locale initiale, aucun commit, push, merge ou déploiement n’avait été effectué. La publication et la fusion ont ensuite été autorisées explicitement par l’utilisateur. Aucun conflit initial ; les modifications du chantier restent disponibles dans le répertoire de travail.

## 1. Audit initial

Le moteur de simulation budgétaire existant réconcilie ses références en centimes ; les crédits/taxes suivent des engagements déclarés. Aucune de ces données ne suffit à déduire une base fiscale ou un cumul de PAS. Le chantier ajoute donc deux modules indépendants, réutilisant authentification/propriétaire, API, money.mjs, modal natif, tokens et chargement différé, sans modifier les calculs bancaires existants. Audit détaillé : `FISCAL_LABORATORY_AUDIT.md`.

## 2. Architecture retenue

Fiscalité : moteur Go pur → API authentifiée sans écriture → formulaire Fiscalité séparé du budget dans Simulation. Calculs rationnels exacts, centimes aux frontières, arrondis déclaratifs à l’euro. Résultat traçable par version/source/année ; aucun LLM.

Laboratoire : contexte léger de brouillons → calculatrice/grille différées → Worker mathjs borné → sauvegarde explicite JSONB. Sources/formules conservées, résultats recalculés. Authentification même avant configuration initiale ; isolation propriétaire à chaque requête. PUT et DELETE conditionnés à une révision, création rejouable par UUID, conflits visibles sans perte locale.

## 3. Bibliothèques et justification

ReactGrid4.1.17/MIT et Univer1.3.1/Apache-2.0 comparés : le premier ne fournit pas de moteur, le second apporte une infrastructure plus riche que le petit tableur demandé. Choix : **mathjs15.2.0/Apache-2.0**, sélection de factories et BigNumber40chiffres, grille HTML native bornée. Le code spécifique adapte références/plages/dépendances, sans réécrire un parseur de mathématiques ou prétendre implémenter Excel.

Une seule dépendance runtime ajoutée, version exacte et lockfile mis à jour. Dépendances transitives MIT ; licences et NOTICE conservées dans le build public (`frontend/public/licenses/laboratory/`). Inventaire : `FISCAL_LABORATORY_LICENSES.md`. `npm audit --omit=dev` : aucune vulnérabilité connue lors du contrôle.

## 4. Références fiscales

Revenus2024/imposition2025 : `FR-IR-2024-2025.1`. Revenus2025/imposition2026 : `FR-IR-2025-2026.1`. Vérification au30octobre2026. Toute autre année est refusée.

Brochures DGFiP millésimées, BOFiP et Service-Public, liens et paramètres dans `FISCAL_RULES.md`. Références indépendantes : célibataire RNI30000 → impôt2104€ en2026 ; couple2enfants RNI54000 → brut2112€ ; exemple de décote couple, brut2140 → décote515 → net1625. Une FAQ officielle incohérente n’a pas servi d’oracle.

Base finale = revenu imposable du foyer **déjà après déductions, conjoint compris** ; second revenu du conjoint interdit sur cette base. En mode salaire, chaque déclarant fournit son net fiscal avant frais. Aucun convertisseur de brut approximatif.

## 5. Fichiers et impacts partagés

- Backend : `internal/fiscal/{rules,engine,engine_test}.go` ; `api/{fiscal,laboratory,fiscal_laboratory_test}.go` ; routes `handlers.go` ; export `export.go` et test d’export existant adapté au nouveau schéma.
- Frontend : `FiscalSimulation.jsx`, `fiscalModel.mjs` et test ; `components/laboratory/` ; `lib/laboratory/` ; `pages/Laboratory.jsx` ; tests Node/navigateur/E2E et banc React synthétique.
- Fichiers partagés : `App.jsx`, `PageLoader.jsx`, `navIcons.jsx`, `api.js`, `cockpit.css`, `pages/Simulate.jsx`, manifeste/lock npm, `vite.config.js`, test de chargement, `CODEBASE_GRAPH.md`. Impacts : nouvelle entrée de navigation, bouton calculatrice, mode dans l’en-tête, attribut status des erreurs API, purge des seules récupérations Laboratoire à la déconnexion/401, préoptimisation de mathjs en développement pour éviter un rechargement au premier Worker.
- Aucune modification des workflows, des seuils Axe, des références visuelles existantes, de l’authentification centrale ou des définitions des soldes/budgets.

Inventaire exhaustif et patch de revue fournis dans `/workspace/output` à la fin des contrôles.

## 6. Migration

`032_laboratory.sql`, additive, sans données initiales : documents JSONB versionnés, propriétaire, UUID, titre/type, révision, timestamps, index et contraintes de taille. Pas de transformation des transactions/obligations. Fresh install, rejeu et rollback à vide testés ; rollback avec documents refusé. Tests vérifient que les montants bancaires restent identiques et qu’un conflit concurrent produit un seul succès.

Aucun service externe, secret ou conteneur additionnel requis pour utiliser les fonctionnalités. Le Worker tourne dans le navigateur. Les conteneurs PostgreSQL de validation sont jetables et contiennent uniquement des fixtures synthétiques.

## 7. Validation

| Vérification | Résultat / portée |
|---|---|
| Go `test -race ./...`, PostgreSQL | Réussi ; moteur, API, migration, export,2propriétaires, concurrence et non-régression existante |
| Go fiscal après dernier contrôle de base du foyer | Réussi ; références, tranches, arrondis, foyers, frais, PAS, cas refusés |
| Go `vet ./...` | Réussi |
| Node `--test src/components/*.test.mjs` |18 fichiers réussis, dont nouveaux tests moteur/contrats |
| `npm run build -- --manifest` | Réussi ; build également vérifié avec Node22.23.4 |
| `npm run test:browser` | Suite complète réussie :14 modules, dont nouveau parcours et chargement de production |
| Nouveau parcours navigateur avec API/PostgreSQL réels | Réussi ; formules, conversion, undo/redo, presse-papiers, conflit409 conservateur, réouverture/récupération, périodes PAS, clavier, Axe |
| Responsive/Axe nouveaux écrans |360/768/1280px, clair/sombre : zéro violation ; pas de débordement horizontal de page |
| E2E fonctionnels |10 scénarios réussis sur base dédiée et deux comptes, dont Fiscalité/Laboratoire |
| Compose | Configuration principale validée avec `example.env` ; aucune image Docker publiée ni nouvelle image complète reconstruite |
| Audit npm production |0 vulnérabilité connue |
| Captures E2E historiques | **Non validées ici**, voir réserve ci-dessous |

Réserve visuelle : les4 captures historiques échouent sous le Chrome local. Le Chromium exact de Playwright156.0.8078.4 ne peut pas être téléchargé (`cdn.playwright.dev` interdit par la politique réseau,403). Le navigateur disponible est Chrome156.0.8078.79. Les images dashboard/transactions/budgets obtenues sont identiques pixel à pixel au témoin local précédemment exécuté, malgré leur écart aux références du dépôt. Pour Simulation,1554pixels diffèrent du témoin, uniquement dans le rectangle du nouveau sélecteur Budget/Fiscalité situé dans l’en-tête fixe capturé. Les seuils et images de référence n’ont pas été modifiés pour masquer ces différences.

**Avant fusion, exécuter les captures dans le Chromium CI et régénérer/revoir uniquement la référence Simulation correspondant au nouveau contrôle. Aucune CI distante déclenchée pour cette livraison locale ; ne pas qualifier la CI globale de verte.**

Performance mesurée sur cette machine, pas une garantie mobile :1000cellules synthétiques en100chaînes de10,40.92ms au premier calcul Node puis13.06–15.66ms. Worker de production ~325ko non compressés, chargé à la demande. Le test de production vérifie absence de modules Laboratoire/Fiscalité/Worker au login/dashboard, puis exécute réellement `0.1+0.2 = 0.3` dans le Worker. Aucun chiffre FPS générique annoncé.

## 8. Limites connues

Fiscalité : résidents métropolitains <65ans, situation stable, salaires ordinaires ou base finale déterminée, mineurs à charge exclusive, cas général/parent isolé. Pas de garde alternée, invalidité, DOM/non-résidence, changement familial/décès dans l’année, veuvage avec enfants, crédits/réductions, hauts revenus/CEHR/CDHR. Frais réels nets qualifiés ; régime commun aux deux déclarants. Année2026 des revenus non disponible ; aucun vieux barème réutilisé silencieusement. Une projection ne devient jamais un solde définitif.

Feuille :100×26,1000cellules stockées,200ko,512caractères/entrée,50calculs historiques,4documents ouverts,20undos en mémoire. Fonctions et syntaxe limitées, point décimal/arguments`;`, références relatives/absolues. Ajout/suppression en fin de grille seulement. Pas de XLSX, import de données bancaires, macro, graphique tableur, tri complexe ou collaboration. La récupération sessionStorage n’est pas une sauvegarde durable ; fermer l’onglet/déconnexion peut la supprimer. Les raccourcis réservés par le système restent tributaires du navigateur ; bouton toujours disponible.

L’AST autorise seulement les constructions documentées. Pas d’assignation, accès propriété, import, code JS, fichier ou réseau ; limites de travail et interruption Worker3secondes. Le moteur ne promet pas une précision infinie :40chiffres significatifs et arrondi monétaire explicite au centime.

## 9. Fonctionnalités réellement opérationnelles

Fiscalité guidée, famille/parts/plafonnement/frais/décote, étapes/sources, comparaison de revenus, PAS annuel/partiel/projeté sans confusion d’année ; calculatrice globale, historique/copie/reset/raccourci ; conversion conservant les données ; grille avec formules/plages/dépendances/cycles, sélection/clavier/presse-papiers, formats et undo/redo ; sauvegarde/reprise/renommage/copie/suppression propriétaire, erreurs réseau et conflits conservateurs ; récupération temporaire par utilisateur/onglet ; export propriétaire. Les deux modules fonctionnent sans IA. Guide : `FISCAL_LABORATORY_GUIDE.md`.

## 10. V2 recommandée

Après les contrôles CI visuels : élargir les foyers/dispositifs fiscaux uniquement avec sources et cas officiels indépendants, permettre un régime de frais distinct par déclarant, ajouter revenus2026 dès publication officielle. Pour le Laboratoire : quotas globaux/document utilisateur si besoin multi-utilisateurs intensifs, gestion de versions historiques, import/export JSON explicite, insertion structurelle au milieu de grille, Worker réutilisé/invalidation incrémentale si mesures réelles l’exigent, tests de performance mobiles et éventuellement ReactGrid/Univer si le périmètre dépasse un mini-tableur.
