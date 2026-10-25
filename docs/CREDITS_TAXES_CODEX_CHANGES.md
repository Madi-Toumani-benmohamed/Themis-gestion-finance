# Crédits & taxes — consolidation Codex

## Périmètre et provenance

Travail sur `feat/credits-taxes-hardening`, worktree `/home/localadm/projets/themis-credits-taxes-hardening`, basé sur `main` après la fusion Analyse (#29, `e628f0b`). Les modifications OpenCode Crédits & taxes ont été sauvegardées puis intégrées seules. Les rapports source et le travail Analyse/Simulation ne sont pas importés. La reprise initiale a été validée sans publication ni déploiement. Après demande explicite de l’utilisateur le 25 octobre 2026, le lot est construit et démarré avec Compose puis publié en PR ; aucune fusion dans main n’est effectuée.

## Architecture et fichiers

- `backend/internal/api/features2.go` conserve comptes/domaines/tags ; les anciens calculs et handlers d’engagements sont remplacés par les modules `engagement_money.go`, `engagement_models.go`, `engagement_report.go`, `engagement_credits.go`, `engagement_payments.go`, `engagement_taxes.go`, `engagement_history.go`. `handlers.go` conserve les routes existantes et raccorde les capacités de paiement/historique.
- `engagement_money.go` centralise validation des montants, bornes, dates, soumissions et contrôle de propriétaire authentifié. Les NUMERIC sont lus en texte ; calculs en `money.Cents`, ratios rationnels réutilisés de l’existant.
- `engagement_models.go` définit la même lecture financière pour les listes, synthèse et mutations. `engagement_report.go` charge crédits et taxes dans un instantané PostgreSQL REPEATABLE READ ; une erreur ne devient pas un total nul.
- `export.go` exporte aussi paiements, historique et métadonnées, y compris engagements retirés, en isolant le propriétaire. Vérification de `rows.Err()` et erreurs génériques ; le tag joint respecte également le propriétaire de la transaction.
- `frontend/src/pages/Credits.jsx` conserve bandeau, cartes et tableau OpenCode, avec chargement unique du rapport et états distincts. `CreditDialogs.jsx` regroupe formulaires contextuels, paiements, historique paginé et consultation des engagements retirés à la demande. `creditsPageModel.mjs` ne recalcule pas les soldes : validation exacte des saisies, formatage, tris et présentation des statuts backend.
- `frontend/src/Credits.css` est entièrement limité à cette page et ses fenêtres. Grille responsive, suppression du double encadrement de largeur, montants tabulaires, confirmations et labels associés. Fenêtres natives avec focus et Échap ; restitution du focus corrigée pour React StrictMode.
- Tests : `features2_test.go`, `engagement_migrations_test.go`, `creditsPageModel.test.mjs`, `credits.browser.cjs`, `backend/tests/credits_http_smoke.py`. Les anciens tests de calendrier qui déduisaient des paiements sont remplacés par des assertions séparées sur échéancier et paiements. Le test d’authentification/mot de passe reste inchangé dans son comportement.
- CI : le parcours navigateur Crédits est ajouté après les quatre parcours existants. Tests Go/PostgreSQL et modèles Node sont déjà découverts par les commandes existantes. Aucun contrôle retiré, aucune dépendance applicative ajoutée.

## Définitions et invariants financiers

| Indicateur | Définition et limites |
|---|---|
| Total nominal prévu du crédit | Mensualité déclarée × nombre de mois, sans capital, intérêts ou amortissement bancaire inventés. |
| Déjà remboursé | Montant déclaré par l’utilisateur **hors** paiements enregistrés dans cette page. `NULL` signifie historique inconnu ; aucune ancienneté ne le remplace par zéro. |
| Paiements enregistrés | Somme exacte des paiements non annulés du propriétaire du crédit. Deux versements similaires restent distincts si leurs soumissions sont distinctes. |
| Total payé déclaré | Historique antérieur + paiements enregistrés, uniquement si l’historique antérieur est renseigné. Ce sont des déclarations, sans rapprochement bancaire automatique. |
| Solde déclaré | Total nominal − total payé déclaré. Inconnu si l’historique est absent ; invalide si les paiements dépassent le nominal. Aucun solde n’est réduit par le seul temps écoulé. |
| Progression | Total payé déclaré / nominal × 100, arrondi rationnel à deux décimales côté backend. Absent si l’historique n’est pas documenté. Les mois payés représentent un équivalent de mensualités, sans affectation automatique de paiement à une date. |
| Échéancier | Première mensualité, durée et mois de fin déterministes. Échéance calendaire à vérifier par mois ; aucune date de prélèvement ni confirmation de paiement inférée. |
| Mensualités prévues ce mois | Mensualités des crédits non clôturés/non soldés dont l’échéancier couvre le mois courant ; estimation de charge, pas dépense observée. |
| Taxe/échéance restant à payer | Montant déclaré − total payé déclaré. Une ancienne case « payé » conserve sa sémantique de confirmation manuelle intégrale, identifiée comme ancienne confirmation. Une simple édition du nom ne change pas cette provenance. |
| Synthèse | Addition des seuls engagements actifs. Total global et total crédits `null` si un crédit n’a pas de solde documenté ; sous-total connu explicitement présenté. Données incohérentes : avertissement/erreur, jamais zéro de remplacement. |

Les montants individuels suivent NUMERIC(12,2) ; durée 1–1 200 mois, année 1–9999 avec fin d’échéancier valide. Les sommes restent dans les entiers sûrs du navigateur (9 007 199 254 740 991 centimes). Dépassement : rejet explicite. Les décimales JSON historiques sont conservées ; les champs `amounts_cents` (et `amount_cents` des paiements) sont la référence pour React. Le formatage utilise BigInt pour conserver même le dernier centime des grandes sommes, sans division flottante. Une saisie avec plus de deux décimales est refusée, jamais arrondie silencieusement.

## Paiements, concurrence et historique

- Toute mutation exige le propriétaire ; identifiants invalides : 400, ressource absente/étrangère : 404, conflit : 409, panne SQL : 500 générique. Toutes les routes d’engagement exigent un propriétaire authentifié, y compris lors de l’installation initiale que le middleware historique laisse ouverte pour d’autres routes.
- Les crédits sont verrouillés avant paiement/modification/clôture/retrait. Le verrou est pris dans une instruction séparée de l’agrégation des paiements : après attente, le snapshot READ COMMITTED voit les versements du précédent écrivain.
- Une clé de soumission obligatoire protège les paiements contre un clic double ou une reprise réseau. Même clé + contenu différent = conflit ; une annulation ne permet pas de réactiver silencieusement l’ancienne soumission.
- Annuler un paiement conserve montant, date, note, motif et date d’annulation ; son effet est exclu du solde. Crédit clôturé ou retiré : pas de paiement/correction cachée.
- Création, édition, paiements, annulations et retrait écrivent des états avant/après dans la même transaction. Une panne d’historique annule la mutation. Une édition sans changement n’ajoute pas de doublon d’historique.
- Les taxes sont verrouillées de même. Paiement partiel = delta positif daté, idempotent ; correction du total payé = action explicite et historisée. Modifier le montant nominal conserve le total payé déjà déclaré.
- Clôturer exige un solde déclaré nul, sans créer de paiement fictif. Rouvrir préserve les écritures.
- DELETE conserve la route mais retire de la synthèse par archivage ; les originaux restent consultables dans les archives/historiques et exportables. Aucune fusion ni suppression des transactions bancaires.

## Statuts et contexte temporel

Date future : « À venir » ; date du jour : « Échéance du jour » ; date passée sans preuve : « Paiement à vérifier ». Paiement partiel et paiement intégral déclarés sont distingués. « Retard confirmé » exige une action utilisateur explicite et peut être retiré explicitement ; il n’est jamais déduit uniquement du calendrier. Salaire de fin de mois et décalage des relevés ne produisent aucune alerte de déficit sur cette page, qui expose des engagements déclarés, pas une couverture bancaire mensuelle.

## Contrats API

Les listes existantes `/api/credits` et `/api/taxes`, les POST/PATCH/DELETE associés et `/api/obligations` sont réutilisés. Le rapport ajoute les listes, contexte `as_of`, source des déclarations, sous-totaux et centimes exacts. `remaining`, `months_paid`, `months_remaining`, progression et totaux globaux peuvent être nuls si inconnus : correction volontaire de l’ancienne certitude calendaire ; les clients externes doivent accepter ces valeurs nulles. Les anciens champs en euros restent présents.

Ajouts nécessaires à l’exploitation des tables OpenCode :

- `POST /api/credits/{id}/payments` : amount, payment_date, idempotency_key, note facultative.
- `DELETE /api/credits/{id}/payments/{payment_id}` : annulation conservatrice avec motif.
- `GET /api/credits/{id}/history` et `GET /api/taxes/{id}/history` : propriétaire vérifié, pagination de 50 éléments par liste (`offset`), états avant/après et paiements.
- `PATCH /api/taxes/{id}` étendu aux champs éditables, déclaration de paiement (`payment_amount`, date, clé, note), correction (`paid_amount`) et `overdue_confirmed` ; un versement ne se mélange pas avec une autre édition.
- `include_archived=1` sur les listes existantes pour la consultation à la demande des engagements retirés.

## Migrations et retour arrière

022/023 sont conservées exactement comme dans la sauvegarde OpenCode. 024 ajoute champs, contraintes de propriétaire, index, idempotence et historique des crédits. Les seules reprises de données sont des identifiants de provenance de l’ancien historique fiscal ; aucun remboursement ancien n’est inventé. Les contraintes NOT VALID conservent les anomalies préexistantes pour diagnostic tout en protégeant les nouvelles écritures. Une ancienne association paiement/propriétaire incohérente invalide le crédit au lieu d’entrer dans un solde certain.

La clé étrangère des paiements empêche la suppression SQL directe d’un crédit avec paiements ; la suppression explicite d’un utilisateur conserve sa cascade antérieure. Supprimer un ancien acteur de modification fiscale n’efface plus l’historique d’un autre propriétaire. L’API utilise l’archivage, pas ces suppressions SQL.

`backend/internal/db/rollbacks/024_engagement_integrity.sql` revient atomiquement au schéma 023 **uniquement avant utilisation des nouveaux champs/historiques**. Il refuse sinon pour préserver les preuves. Ne jamais exécuter un downgrade destructeur après saisie de paiements ; prévoir une sauvegarde et une procédure de restauration revue. Revenir à l’ancien code réintroduirait ses hypothèses financières : ce n’est pas un retour fonctionnel sûr.

## Limites assumées

Déclarations utilisateur, sans vérification par établissement bancaire ; pas d’amortissement/capital/intérêts connus, ni d’affectation automatique des paiements aux transactions. Les historiques antérieurs à 023/024 ne peuvent pas être reconstruits. Les données historiques atypiques ou propriétaires absents ne sont pas corrigés automatiquement. Une modification SQL directe peut contourner l’historisation applicative ; les contraintes protègent la propriété/les bornes mais ne remplacent pas une journalisation externe. La consultation mobile est testée sous Chromium, pas sur tous les moteurs ou lecteurs d’écran. Le découpage du bundle frontend reste une amélioration globale hors périmètre.


## Déploiement Compose autorisé le 25 octobre 2026

Construction depuis ce worktree avec le projet existant `themis-gestion-finance` et son fichier `.env` existant, sans en copier les secrets. `docker compose ... build app`, puis `up -d --no-deps --no-build --wait app` : réussite. Seul le conteneur applicatif est remplacé ; PostgreSQL, FalkorDB, Knowledge Graph et volumes existants sont conservés.

Sauvegarde PostgreSQL privée au format custom, vérifiée avec `pg_restore --list`, et ancienne image conservée avant remplacement : `/home/localadm/.codex/backups/themis-before-credits-compose-20261025T070739Z`. PostgreSQL 16.15 ; migrations 022, 023 et 024 appliquées par le mécanisme existant. Comparaison d’empreintes avant/après : colonnes historiques des crédits et taxes, transactions et utilisateurs inchangés. Aucun montant ni mot de passe existant modifié.

Contrôles opérationnels en lecture : santé HTTP 200, page et assets compilés HTTP 200, API obligations sans session HTTP 401 ; les quatre services sont démarrés. Application disponible sur `http://localhost:8080`. Ces contrôles de déploiement ne remplacent pas les tests synthétiques décrits dans TESTS.
