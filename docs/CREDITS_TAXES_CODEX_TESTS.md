# Crédits & taxes — validations Codex

Validation locale du 25 octobre 2026, branche `feat/credits-taxes-hardening`, worktree `/home/localadm/projets/themis-credits-taxes-hardening`. Aucun test ne vise une base financière réelle. Go 1.23 est exécuté via Docker ; PostgreSQL 16 de test utilise des schémas uniques isolés. `THEMIS_TEST_DATABASE_URL` est défini : les tests PostgreSQL ne sont pas ignorés.

## Résultats constatés

| Commande ou contrôle | Résultat |
|---|---|
| OpenCode importé, avant correction : `go test ./...`, `go vet ./...`, `npm ci`, `npm run build` | Réussite, malgré les hypothèses calendaires incorrectes identifiées par l’audit. |
| Backend consolidé : `go test ./...` | Réussite de tous les packages, tests PostgreSQL et migrations inclus. |
| `go vet ./...` | Réussite. |
| `CGO_ENABLED=1 go test -race ./...` | Réussite, PostgreSQL inclus. |
| `node --test src/components/*.test.mjs` | **48 tests réussis**, 0 échec, 0 ignoré ; 5 nouveaux tests du modèle Crédits. |
| `npm run build` | Réussite ; avertissement Vite existant du bundle supérieur à 500 kB, non masqué. |
| `node tests/credits.browser.cjs` | Réussite avec API simulée, cinq largeurs, parcours et Axe. |
| Scripts navigateur existants Transactions, Budgets, Simulation et Analyse | Les quatre réussissent, scénarios financiers et accessibilité inclus. |
| `docker build -f backend/Dockerfile -t themis-credits-taxes-hardening:review .` | Réussite : frontend et backend compilés dans l’image. |
| `docker compose --env-file example.env config --quiet` | Réussite. Aucun déploiement sur la stack financière habituelle. |
| Image démarrée sur un schéma PostgreSQL synthétique vide, puis `backend/tests/credits_http_smoke.py` | Réussite sur HTTP réel : cookies, routeur, propriété, paiements, historique, agrégats et export. |
| `git diff --check` | Réussite. |
| Préservation OpenCode | Les 16 empreintes SHA-256 correspondent à la sauvegarde initiale. Le `AGENTS.md` utilisateur du worktree Analyse reste hors de ce lot. |

## Backend : scénarios critiques

Dix tests dans `backend/internal/api/features2_test.go`, plus `engagement_migrations_test.go` avec variantes base vide et données anciennes :

- Crédit ancien sans historique : aucun paiement, solde ni progression certifié par le seul calendrier. Échéancier et fin sont vérifiés séparément.
- Paiements partiels exacts, crédit payé intégralement, dépassement, montants inhabituels, limites de durée et d’année, mois et dates invalides.
- Taxe future, du jour, passée à vérifier, partielle, payée, retard confirmé uniquement explicitement.
- CRUD, clôture refusée si le solde n’est pas nul, clôture/réouverture sans création de paiement.
- Deux utilisateurs : refus des modifications, retraits, paiements, annulations et lectures d’historique croisés ; listes, synthèse, export et jointure des tags isolés. Administrateur et membre vérifiés également par le test HTTP réel.
- Même clé et contenu : une seule écriture ; clé réutilisée avec montant différent : conflit. Deux soumissions distinctes de même montant sont conservées. Une annulation ne réactive pas l’ancienne clé.
- Deux paiements concurrents susceptibles de dépasser le nominal : un seul accepté. Le snapshot après attente du verrou voit le paiement précédent.
- Pannes SQL provoquées sur la colonne d’historique dans un schéma synthétique : modification, paiement, annulation et confirmation fiscale entièrement annulés. Une mutation financière ne réussit pas sans preuve historisée.
- Modification du montant fiscal : conserve le total payé antérieurement. Une édition du nom avec montant inchangé conserve la provenance d’une ancienne confirmation.
- Réconciliation : crédit de 99,99 € − paiement de 33,33 € = 66,66 € ; plus taxe de 10,01 € = 76,67 €, réconciliés dans les vues sources et champs centimes.
- Archivage conservateur, suppression SQL directe d’un crédit avec paiements refusée, édition sans changement n’ajoutant pas d’historique.
- Validation par les vrais handlers, sans recopier leurs conditions dans le test.
- Propriétaire nul/négatif refusé. Routeur complet avant configuration : les treize routes d’engagement exigent une authentification.

Le test de migrations applique le mécanisme réel des fichiers embarqués sur base vide et sur schéma arrêté à 023. Second appel sans réapplication ; montant d’origine inchangé et historique de remboursements maintenu à NULL ; ancien historique fiscal enrichi seulement de ses identifiants, sans faux snapshot financier. Rollback 024 avant utilisation puis réapplication réussis ; rollback refusé après nouvelle déclaration, sans perte. Suppression d’un ancien acteur : historique d’un autre propriétaire conservé ; suppression explicite du propriétaire : cascade antérieure préservée.

## Frontend : scénarios critiques

Le modèle teste la borne exacte de 90 071 992 547 409,91 €, les informations inconnues distinctes de zéro, la virgule de saisie, le rejet des décimales excessives, les tris sans mutation et l’absence de paiement déduit de l’ancienneté.

Le navigateur intercepte **toutes** les API, sans compte réel ni LLM. Il couvre : rapport unique sans chargements de listes redondants ; cycle React StrictMode et annulation ; erreurs conservant les saisies et reprise ; double clic ; paiement partiel et rafraîchissement ; détail avant/après ; annulation motivée ; création/édition/retrait avec confirmation ; correction fiscale ; confirmation/retrait d’un retard ; clôture/réouverture sans requête de paiement ; historique retiré consultable sans contrôle de modification actif ; erreurs d’historique et reprise ; tris/filtres ; attente, erreur de lecture sans fausse synthèse vide, reprise et véritables états vides.

Largeurs : **1920, 960, 768, 390 et 320 px**. La largeur 960 reproduit la largeur utile d’un zoom de 200 % sur écran 1920. Contrôle du débordement du document et des limites du tableau autour de chaque nom et bouton. Tabulation, labels, Échap et restitution du focus vérifiés. Axe WCAG 2 A/AA et 2.1 AA : aucune violation constatée sur page et fenêtres.

Captures synthétiques : `/tmp/themis-credits-ui/`. La revue visuelle a détecté une coupure des cellules mobiles malgré l’absence de débordement horizontal ; hauteur locale corrigée et assertion géométrique ajoutée. Un échec réel a aussi permis de corriger le focus des fenêtres sous StrictMode.

## Test HTTP de l’image

Image démarrée dans le réseau du PostgreSQL **de test**, `DATABASE_URL` limité à un schéma neuf, `AI_PROVIDER=none`, `MEMORY_ENABLED=false`. Aucun secret ou paramètre de la stack habituelle repris.

```bash
THEMIS_SYNTHETIC_TEST_INSTANCE=1 \
THEMIS_CREDITS_SMOKE_URL=http://IP_DU_CONTENEUR_DE_TEST:8080 \
python3 backend/tests/credits_http_smoke.py
```

Le script refuse toute mutation si l’instance est déjà configurée. Deux utilisateurs synthétiques, authentification réelle par cookie, vérification de propriété dans les deux sens. Exemple : 99,99 € − 22,22 € = 77,77 € de crédit, plus 66,68 € d’échéance après paiement partiel = 144,45 €. Répétition d’une soumission : une seule écriture. Export d’un autre utilisateur : aucun de ces paiements. Retrait du crédit : historique conservé.

## CI et limites

Le parcours Crédits rejoint le job navigateur après les quatre scripts existants ; les tests Go/PostgreSQL et Node sont découverts par les commandes habituelles. Aucun test désactivé, aucune assertion financière assouplie. Les anciennes attentes calendaires fausses ont été remplacées par la distinction échéancier/paiements.

Les résultats ci-dessus concernent les validations locales avant publication du lot Crédits. La CI GitHub est déclenchée par l’ouverture de la PR ; ses résultats doivent être contrôlés avant toute fusion. La PR Analyse #29 avait été fusionnée après CI/Security distantes vertes.

Pas d’audit des données bancaires réelles ni de certification d’une déclaration auprès de la banque. Pas d’amortissement/capital/intérêts inventés, ni d’historiques anciens reconstruits. Pas de campagne Safari/Firefox, lecteur d’écran ou zoom natif manuel ; largeur utile équivalente testée sous Chromium. Les tests synthétiques restent isolés. Le déploiement Compose ultérieur, explicitement demandé par l’utilisateur, est décrit ci-dessous.

**Verdict : apte à une PR de revue ; pas de fusion directe avant CI de cette PR.** Aucune anomalie bloquante connue après ces validations locales ; les limites financières restent explicitement présentées.


## Vérification du déploiement demandé ensuite

`docker compose -p themis-gestion-finance --env-file <fichier .env existant> build app`, puis `up -d --no-deps --no-build --wait app` : réussite. Sauvegarde PostgreSQL complète vérifiée et ancienne image conservée avant remplacement. Migrations 022–024 présentes dans le suivi PostgreSQL. Contrôle d’intégrité avant/après par empreintes, sans afficher de données sensibles : crédits/taxes historiques, transactions et utilisateurs inchangés. Santé/page/assets HTTP 200, obligations sans session HTTP 401, quatre services démarrés. Aucun scénario de test n’est exécuté sur les données financières réelles.
