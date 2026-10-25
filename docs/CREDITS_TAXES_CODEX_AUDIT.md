# Crédits & taxes — audit de reprise Codex

Audit commencé le 25 octobre 2026, sans consultation de données bancaires réelles. L’intégration Analyse a d’abord été terminée : PR #29 fusionnée dans `main` (`e628f0b`), workflows CI et Security réussis. Le présent lot utilise `feat/credits-taxes-hardening` dans `/home/localadm/projets/themis-credits-taxes-hardening`.

## Provenance et préservation

- Codex : ancienne branche `feat/analysis-exploration`, fichiers applicatifs committés ; modification utilisateur locale de `AGENTS.md` conservée hors commit.
- OpenCode : `/home/localadm/projets/themis-opencode`, branche `feat/simulation-redesign`, HEAD `3746b29`, modifications Crédits & taxes non commitées.
- Sauvegarde privée : `/home/localadm/.codex/backups/themis-credits-taxes-20261025T055232Z`, patch binaire des changements suivis, copies des 16 fichiers modifiés/non suivis, manifeste SHA-256 et références Git. Aucun fichier OpenCode modifié par Codex.
- Rapports consultés dans le worktree source : `equipe/PLAN.md`, `REVUE.md`, `TESTS.md`. La revue conclut « À refaire », tandis que TESTS conclut « Validé » ; ces affirmations sont confrontées au code, pas utilisées comme preuves de non-régression.
- Le CSS annoncé est réellement `frontend/src/Credits.css`, le modèle `frontend/src/components/creditsPageModel.mjs`. Les migrations 022 et 023 existent uniquement parmi les fichiers non suivis d’OpenCode au début de la reprise.

## Constats vérifiés dans le code

| Priorité | Sujet | Constat et cause | Consolidation prévue |
|---|---|---|---|
| P0 | IDOR fiscal | Le code de base met `paid` à jour sans propriétaire. OpenCode ajoute SELECT et UPDATE avec `owner_id`, et un test PostgreSQL croisé. Correction présente, mais erreurs SQL assimilées à 404 et historique non atomique. | Verrouiller la ressource du propriétaire et modifier état/historique dans une transaction ; tester non-propriétaire, panne SQL et concurrence. |
| P0 | Paiements fictivement déduits | `computeCredit` nomme `months_paid` le nombre de mois calendaires écoulés, mois courant inclus ; l’UI conclut « Soldé » à la fin théorique. Aucun paiement n’est consulté. | Séparer échéancier estimé et paiements déclarés. Une absence de données historiques ne peut confirmer un remboursement. |
| P0 | Monnaie | Les modèles d’entrée/sortie et scans des montants restent `float64`. Des conversions `FromLegacy`/`Parse` ignorent les erreurs. La multiplication par un nombre de mois non borné peut déborder ; les sommes de taxes utilisent un parseur limité à un montant NUMERIC individuel. | Lire NUMERIC en texte, décoder les montants exacts, vérifier bornes et opérations ; JSON décimal sans calcul en float. |
| P0 | Historique fiscal | SELECT, UPDATE et INSERT sont trois opérations séparées ; l’erreur d’insertion est ignorée. Deux changements concurrents peuvent écrire un état avant incorrect. | Transaction, verrou et comparaison avant/après ; aucune modification réussie sans historique cohérent. |
| P0 | Migration paiements | 022 crée une table mais aucun endpoint, calcul ou contrôle UI ne l’utilise. Le propriétaire du paiement n’est pas lié par contrainte au propriétaire du crédit. | Exploiter la table, isoler lecture/écriture, sérialiser les paiements et rendre les soumissions idempotentes sans confondre deux paiements similaires. |
| P0 | Suppression/historique | 022 et 023 utilisent `ON DELETE CASCADE` depuis les engagements : supprimer une ressource peut supprimer ses preuves. | Préserver les historiques lors du retrait d’un engagement ; migration corrective additive, pas de réécriture des 022/023 potentiellement déjà appliquées. |
| P1 | Modification crédit | Le nouvel endpoint OpenCode filtre le propriétaire dans EXISTS/UPDATE/SELECT. Il n’existait pas dans le code de base : la protection est présente sur une fonctionnalité ajoutée, pas une faille d’un ancien endpoint PATCH. Vérification et mutation ne sont pas atomiques et certaines erreurs deviennent 404. | Réutiliser l’endpoint avec validation réelle, contrôle des lignes et verrou partagé avec les paiements. |
| P1 | Validation mois | `time.Parse` rejette 00/13 et améliore la regex historique. Année zéro, dates incohérentes, précision et limites restent à traiter. | Calendrier strict, validation des montants et du nombre de mensualités, erreurs explicites. |
| P1 | Agrégats | `obligations` ignore les erreurs de requête/scan et peut renvoyer des zéros réussis. Les crédits sont indépendants des transactions dans les modules consultés ; le `credits_observed` de Budgets désigne des crédits bancaires observés (montants positifs), et non les échéanciers de prêts. | Agrégats issus d’une même lecture cohérente, incomplet distinct de zéro ; pas de nouvelle dépense bancaire créée à partir d’un échéancier. |
| P1 | Frontend | Les erreurs de chargement deviennent des listes vides et peuvent alimenter un total de remplacement. L’agrégateur frontend additionne des floats. Les soumissions ne sont pas verrouillées ; suppressions sans confirmation ; labels non associés. | États distincts, calculs backend vérifiés en centimes, attente des mutations, confirmations et contrôles accessibles. |
| P1 | Échéance dépassée | `getTaxStatus` renvoie automatiquement `late` à partir de la date ; l’UI affiche « en retard » sans déclaration de paiement ou de retard. | « Paiement à vérifier » par défaut ; états du jour, partiel, payé et retard explicitement confirmé distingués. |
| P1 | Fonctionnalités annoncées | Les paiements partiels, consultation d’historique et clôture ne sont pas présents dans le frontend ou le routeur source. Le tri et l’édition des crédits existent. | Conserver tri/édition et terminer les mécanismes de paiement/historique utiles, sans amortissement bancaire inventé. |
| P2 | Design | Le CSS emploie des tokens existants mais définit des classes génériques (`form-row`, `btn-icon`, etc.) et impose une largeur/marges supplémentaires. La dette est colorée comme une anomalie. | Conserver cartes, bandeau et tableau ; scoper le CSS, améliorer largeur et responsive, neutraliser les alertes injustifiées. |

## Tests OpenCode examinés

Des tests PostgreSQL existent bien pour `patchTax` et `updateCredit`, en plus des tests de calcul. En revanche `TestUpdateCreditValidation` reproduit les conditions dans le test sans appeler le handler : ce n’est pas une preuve du comportement HTTP. Les tests de crédit considèrent toujours les mois écoulés comme payés ; cette attente doit être remplacée par la distinction métier demandée, avec maintien d’un test de calendrier. Aucun test frontend du modèle ni parcours navigateur Crédits n’est livré. Les essais shell sont sauvegardés et lus si utile, pas exécutés contre la stack réelle.

## Décisions et ordre des lots

1. Intégrer uniquement les fichiers Crédits & taxes sauvegardés dans le worktree dédié, sans importer de changement Analyse/Simulation ni modifier OpenCode ; exécuter l’état importé sur PostgreSQL de test.
2. Définir les montants d’échéancier, le suivi des paiements et les informations historiques inconnues ; fiabiliser propriété, sommes, mutations et historique.
3. Ajouter une migration corrective après 022/023 et ses vérifications sur schéma isolé. Aucun changement sur une base financière réelle.
4. Consolider l’interface existante (états, confirmation, détail/paiements/historique, espace, clavier/mobile) et ses modèles testables.
5. Exécuter tests unitaires/PostgreSQL, migrations, Go/vet, Node, navigateur et build ; documenter résultats réels et limites dans CHANGES et TESTS.

Verdict de l’audit initial : la reprise OpenCode seule n’était pas apte au merge. Le bilan après consolidation et les preuves sont précisés ci-dessous et dans TESTS.

## Compléments confirmés pendant la consolidation

- **P0 — Installation initiale** : le test HTTP Docker a constaté que `authMiddleware` autorise les routes financières avant création du premier utilisateur. Les treize routes Crédits/taxes/synthèse/paiements/historiques exigent désormais un propriétaire authentifié positif, avec test du routeur complet. Le comportement des autres routes d’installation n’est pas modifié.
- **P1 — Conservation de l’acteur historique** : 023 supprimait aussi l’historique fiscal en cascade lors de la suppression de `changed_by`. 024 remplace cette relation par SET NULL, indépendamment du propriétaire de l’historique. Test de suppression de l’acteur puis du propriétaire sur schéma réel migré.
- **P1 — Exactitude navigateur aux grandes valeurs** : un montant en euros JSON peut perdre un centime au décodage JavaScript, même si le calcul Go est exact. Les compagnons entiers en centimes et le formatage BigInt garantissent cette frontière ; test à la borne maximale exacte.
- **P1 — Provenance d’une ancienne confirmation fiscale** : un formulaire d’édition complet renvoyant un montant inchangé ne doit pas promouvoir une confirmation historique en nouveau paiement. Le backend conserve cette provenance ; augmenter le montant nominal conserve le montant payé antérieurement, sans inventer le complément.
- **P2 — Focus des fenêtres** : les parcours clavier ont constaté une restitution du focus incorrecte sous le cycle de montage/nettoyage React StrictMode. Le déclencheur est conservé dans une référence stable ; fermeture Échap/Annuler et focus de retour sont testés.

Les corrections de propriété et de validation annoncées par OpenCode étaient donc présentes mais insuffisantes pour conclure à la fiabilité financière. Le calendrier et les tables inutilisées constituaient les principaux écarts métier. Les fonctionnalités utiles et la base esthétique sont conservées ; les travaux de consolidation et les limites sont détaillés dans CHANGES, les preuves dans TESTS.

- **P2 — Coupure des actions mobiles** : la revue des captures a montré qu’une hauteur de cellule globale héritée pouvait tronquer les actions, sans débordement horizontal du document. Les hauteurs sont corrigées localement ; une assertion vérifie les limites de chaque nom et bouton dans le tableau aux cinq largeurs testées.

## Bilan après consolidation

Les risques P0/P1 identifiés sont corrigés dans le lot isolé et couverts par les validations de TESTS. Les limites restantes concernent les déclarations non rapprochées avec la banque, l’absence de capital/intérêts connus, les historiques antérieurs non reconstructibles et l’éventuelle incohérence historique à examiner sans réécriture automatique. Les contraintes NOT VALID ne prétendent pas certifier rétrospectivement toutes les données.

Verdict : apte à une PR de revue ; aucune anomalie bloquante connue après validations locales. La CI GitHub de ce lot doit encore s’exécuter après publication avant toute fusion. Aucune fusion automatique de Crédits & taxes n’est effectuée. Le déploiement Compose demandé ensuite par l’utilisateur, sa sauvegarde et ses contrôles d’intégrité sont consignés dans CHANGES et TESTS.
