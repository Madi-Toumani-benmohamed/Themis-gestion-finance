# Relevés — changements, invariants et limites

Branche `feat/statements-documentary-hardening`, worktree `/home/localadm/projets/Themis-gestion-finance`. Base `86705e4` : PR IA #32 fusionnée après CI et sécurité réussies. Ce chantier Relevés reste local et non commité ; `AGENTS.md` préexistant est préservé. Aucun changement appliqué aux données financières réelles.

## Fonctionnement livré

- Le dénominateur documentaire ne compte que les comptes `tracking_mode=statements`. Les autres comptes restent identifiables comme pointages, sans fausse alerte mensuelle. Plusieurs comptes documentaires sont pris en charge. Zéro compte requis ne signifie pas « période complète ».
- Le mode est persisté, initialisé selon `kind` puis modifiable explicitement depuis Comptes. Ni le nom du compte ni une valorisation ancienne ne déterminent une anomalie. Les soldes et pointages existants restent utilisables.
- Le rapprochement est calculé sur les transactions effectivement conservées : `théorique = ouverture + somme signée`, `écart = clôture − théorique`. NUMERIC PostgreSQL et centimes entiers ; tous les débits/crédits, transferts et remboursements participent au contrôle du solde bancaire, indépendamment de leur traitement analytique. Les champs inconnus restent nuls.
- Les soldes, dates et libellés bruts ne sont pas corrigés automatiquement. Une validation historique incohérente après déduplication est conservée comme déclaration mais ne prouve plus la couverture. Une validation manuelle ne peut pas contourner un écart.
- Un import partiellement dédupliqué n’hérite plus de la validation du parsing. Un import entièrement dupliqué garde le mécanisme existant de retrait de son enregistrement vide. Un véritable relevé sans opération est désormais conservé ; son message d’import ne prétend plus à un doublon.
- L’aperçu d’import utilise lui aussi des centimes exacts et affiche la formule du même écart, sans résidu flottant pour `0,10 + 0,20 = 0,30`.

## Statuts et contrôles

`Importé` correspond à la réception et au traitement existants ; aucune ligne n’est persistée avant que le parsing soit terminé. Aucun faux statut de parsing en cours, d’erreur persistée ou d’archive n’a été ajouté.

| État affiché | Conditions |
|---|---|
| Analysé · non validé | Contrôles élémentaires exploitables, mais validation non enregistrée ou rouverte |
| Validé · vert ✓ | Validation enregistrée et contrôles bloquants actuellement cohérents. Une revue manuelle peut attester les avertissements non bloquants, qui restent détaillés |
| À vérifier · orange ! | Contrôle requis inconnu, date d’opération hors période ou suspicion de similarité/chevauchement |
| Contrôles en échec · rouge × | Écart comptable ou rupture de solde confirmé ; prioritaire sur un soupçon de chevauchement |

Les résultats des contrôles sont distincts : **Vérifié**, **Échec confirmé**, **Non établi**, **À examiner**. Un contrôle non exécuté n’est jamais affiché réussi. Une égalité arithmétique ne prouve pas la complétude de l’extraction.

Reprise du 25 octobre 2026 : le statut « À vérifier » persistait après une validation manuelle parce que la finalisation réimposait le statut orange pour les avertissements de date ou de similarité. `finalizeStatementStatus` conserve désormais « Validé » après revue manuelle lorsque les prérequis passent. Les avertissements restent visibles ; un contrôle bloquant garde la priorité.

Les périodes bancaires réelles sont inclusives et distinctes du mois calendaire. Deux relevés partageant une seule borne ne constituent pas automatiquement un chevauchement suspect. Les trous indiquent une absence de preuve documentaire, pas une absence d’opérations. Les jours futurs ne sont pas comptés. La continuité des soldes est comparée au centime entre périodes consécutives ou partageant une borne ; elle reste non établie à travers une interruption.

Les similarités sont strictes : même compte, date, montant et libellé bancaire après casse/espaces de bord. Elles sont signalées comme soupçons, sans fusion, suppression ou exclusion analytique automatique. Les opérations hors période sont également des éléments à vérifier : le modèle ne conserve pas de date de valeur indépendante permettant d’en déduire une erreur certaine.

La **couverture des dates**, la **réconciliation**, les **soupçons** et la **validation déclarée** restent des dimensions distinctes. Des périodes peuvent être documentées tandis qu’un chevauchement ou une écriture similaire nécessite encore une revue ; aucun score global ne mélange ces dimensions.

## Revue manuelle et suppression

`PATCH /api/statements/{id}/validation` accepte `validate` ou `reopen`, avec justification de 10 à 1 000 caractères. Compte et relevé doivent appartenir à l’utilisateur. Une validation automatique peut être attestée manuellement une seule fois, sans réouverture artificielle ; une validation manuelle répétée ou la réouverture d’un relevé déjà rouvert est refusée. Le verrou propriétaire sérialise les imports et les transitions ; contrôle, modification et historique forment une transaction. Un échec SQL précoce ou tardif annule tout.

### Actions et nouvelles vérifications

Le bouton **Actions** ouvre le panneau existant. Validation, justification et relance sont placées en haut ; chargement et erreurs restent visibles. Les couleurs sont accompagnées de texte et de symboles, avec contrastes vérifiés dans les deux thèmes.

`POST /api/statements/{id}/checks` accepte `{"mode":"deterministic"}` ou `{"mode":"combined"}`. Il exige une session et un relevé appartenant à l’utilisateur. Une transaction en lecture seule fournit les contrôles recalculés et une date de photographie réelle ; aucun montant, transaction, flag de validation ou historique n’est écrit par ce contrôle. Son résultat est affiché pendant la session du panneau, sans historique durable des relances.

- **Sans IA** : recalcul des contrôles comptables existants, sans appel fournisseur.
- **Contrôles + relecture IA** : mêmes calculs déterministes, puis priorisation facultative des constats par le fournisseur déjà configuré. Aucune nouvelle dépendance ou nouveau modèle.

La relecture transmet uniquement les textes des constats déterministes : pas de PDF, soldes, montants bancaires, libellés, noms de comptes, noms de fichiers, identifiants de ressources, notes privées, contexte personnel ou Knowledge Graph. Au plus 20 contrôles prioritaires sont transmis et la limite est indiquée. Le modèle retourne uniquement des identifiants de constats ; le serveur reconstruit les textes et les références, impose la réserve sur l’absence d’original et réinsère tout échec confirmé omis par le modèle. Aucun texte libre ou chiffre inventé n’est accepté.

Un fournisseur absent, indisponible, en timeout ou une réponse non vérifiable laisse les résultats comptables disponibles, avec une explication. La requête entière est bornée à 30 secondes maximum. La relecture n’est pas une seconde extraction indépendante du PDF et n’augmente pas à elle seule la preuve de complétude. **L’IA ne peut ni valider un relevé ni contourner un contrôle en échec.** Une nouvelle validation reste une action explicite avec justification.

L’historique conserve propriétaire, acteur, action, état précédent, date, justification et snapshot des métadonnées/contrôles. Le détail affiche les 20 dernières revues ; l’export conserve l’historique complet. La validation automatique issue du parsing est distinguée d’une revue manuelle ; les événements historiques inexistants ne sont pas inventés.

La suppression demande une confirmation indiquant le nombre d’opérations et les conséquences : transactions/tags/propositions associés retirés, indicateurs recalculés, analyses enregistrées du propriétaire invalidées. Les autres relevés, comptes et pointages restent en place. La suppression d’un relevé ayant des transactions rattachées à un autre propriétaire est bloquée. L’historique conserve le `resource_id` et ses snapshots même après retrait de la ressource ; les écritures supprimées ne restent pas consultables dans Transactions. Rien n’est supprimé dans la base réelle par l’agent.

## Architecture et contrats

- `backend/internal/api/statement_controls.go` : DTO enrichi, contrôles exacts, lecture cohérente dans un snapshot, révisions manuelles et suppression atomiques, modification du mode.
- `statement_recheck.go` : contrôle explicite en lecture seule, session/propriétaire, relecture IA via `llm.Client` et sélection de preuves réutilisée de `narrative.go`, erreurs sûres et délai borné.
- `reliability.go`, `hero_diagnosis.go`, `period.go`, `intelligence.go` : même périmètre documentaire, comptes manuels exclus, prédicat financier commun. Pas de bilan négatif déduit d’un mois courant ou d’un salaire non encore versé.
- `transactions.go` : `statement_id` partagé entre rapport, agrégats et export ; provenance `statement_validated` fondée sur les contrôles actuels, et non le seul flag historique. Les opérations provisoires restent observables.
- `handlers.go` : routes et préservation des imports sans opération ; validation désactivée sur nouvel import partiellement dédupliqué. `import_preview.go` : écart exact. `features2.go` : mode des comptes et rattachements propriétaire. `export.go` : mode de suivi et historique documentaire exportés.
- `Statements.jsx/.css` + `statementPageModel.mjs` : synthèse globale, détail de couverture mensuelle, filtres compte/statut/recherche/périmètre et tri, tableau dense large, détail clavier, erreurs/reprise et confirmation secondaire de suppression. Le frontend contrôle aussi la réconciliation des centimes reçus.
- `DataReliability.jsx` et son modèle : couverture réutilisée, exclusion des pointages, requêtes annulables. `Accounts.jsx` : seul le mode de suivi est ajouté. `Analyse.jsx` : texte adapté pour ne pas marquer les comptes manuels à vérifier. `App.jsx`, `api.js`, `Transactions.jsx` : rebond vers toutes les opérations de la source, sans restriction mensuelle implicite.
- `useModal.js` : extraction du gestionnaire natif de Transactions, réutilisé sans nouvelle dépendance ; maintien du focus, Échap et restauration du focus. Les tests de Transactions restent inchangés.
- Tests d’intégration existants : fixtures de compte explicites et soldes de contrôle synthétiques cohérents, y compris après ajout d’opérations. Aucune assertion financière affaiblie. Le nouveau parcours navigateur est ajouté aux contrôles CI existants.

`GET /api/statements` reste un tableau et conserve les anciens champs ; compte, contrôles, centimes, statut effectif et historique sont additifs. Le booléen historique `validated` reste accessible pour traçabilité ; l’interface utilise `effective_validated` et les contrôles. Nouvelle route de revue et `PATCH /api/accounts/{id}` limités au mode ; aucun endpoint de valorisation remplacé.

La migration **025** ajoute seulement mode de suivi, historique et index `(statement_id,owner_id)`. Les migrations antérieures ne changent pas. Le rollback est atomique et refuse de perdre une revue ou un mode personnalisé. Les copies de test vérifient installation, reprise de données anciennes, répétition et rollback. Aucune migration exécutée en production.

## Limites et suites possibles

- Les PDF binaires originaux ne sont pas conservés par le modèle existant : la revue nécessite le document détenu par l’utilisateur. Aucun téléchargement original fictif proposé.
- Pas de modification de soldes/libellés pour masquer un écart. Une extraction erronée doit être reprise depuis la source ; une attribution historique perdue après déduplication n’est pas reconstruite automatiquement.
- Les soupçons ne sont pas une preuve de doublon et peuvent rester affichés après une confirmation manuelle. Aucun statut de résolution par anomalie ni rapprochement bancaire externe ajouté.
- Les dates de valeur indépendantes, le nombre déclaré d’opérations bancaires et des preuves d’extraction exhaustive sont absents du modèle actuel. Le contrôle arithmétique ne leur sert pas de substitut.
- La fraîcheur des valorisations reste la date connue dans Comptes, sans seuil comptable imposé. Les faits déjà appris dans le Knowledge Graph ne sont pas purgés automatiquement lors d’une suppression ; ils restent indicatifs, les données transactionnelles font référence.
- L’ancien endpoint `/api/coverage`, non utilisé par cette page, conserve ses repères historiques et son heuristique de fraîcheur ; il ne doit pas remplacer `/api/reliability` pour prouver une couverture. Le périmètre de cette intervention reste le contrôle documentaire et ses consommateurs actifs.
- Aucun archivage ni restauration des écritures supprimées ajouté. L’historique de suppression préserve les métadonnées, pas une copie intégrale des opérations.

Prochaines itérations envisageables, sans implémentation automatique : conservation sécurisée des originaux, source de date de valeur, résolution tracée de chaque soupçon et reprise contrôlée des métadonnées erronées. Aucun nouveau fournisseur ou score arbitraire ajouté.
