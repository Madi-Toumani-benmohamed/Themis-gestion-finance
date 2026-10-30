# Fiscalité / Laboratoire — validation et publication

Branche `feat/fiscal-laboratory`, base `86d1b951930647cc0bf8f473121828f74a2b7e9a`. Publication et fusion autorisées explicitement par l’utilisateur après la livraison locale. Vérifier la CI sur la PR avant fusion. Toutes les modifications initiales de ce chantier ont été conservées.

Implémentation, architecture, références, guide et limites documentés dans `FISCAL_LABORATORY_REPORT.md`, `FISCAL_LABORATORY_AUDIT.md`, `FISCAL_RULES.md`, `FISCAL_LABORATORY_GUIDE.md`, `FISCAL_LABORATORY_LICENSES.md` et `CODEBASE_GRAPH.md`.

Validations locales réussies : Go `-race ./...` avec PostgreSQL synthétique ; `go vet ./...` ;18 fichiers de tests Node ; build production et build Node22 ; suite navigateur complète14 modules ; nouveau parcours API/PostgreSQL réel avec récupération immédiate/rechargement sans écriture serveur ;10E2E fonctionnels à2comptes ; Axe zéro violation sur les nouvelles interfaces en360/768/1280px, clair/sombre ; Worker exécuté dans le build de production et absent du login/dashboard ; Compose ; audit npm production0vulnérabilité.

Corrections issues des tests : classe du bouton calculatrice distincte de celle du thème ; contraste stable des champs lors du changement de thème ; préoptimisation mathjs évitant un rechargement Vite au premier calcul ; récupération locale vidée à la déconnexion/401 et écrite à la sortie/rechargement ; interception du chunk de test défaillant installée avant toute prélecture ; base finale fiscale du foyer refusant un revenu conjoint ajouté une seconde fois ; styles de boutons existants réutilisés.

Réserve ouverte :4 captures E2E historiques échouent avec le Chrome local ; Chromium exact Playwright bloqué par politique réseau403. Dashboard/transactions/budgets identiques pixel à pixel au témoin local précédent. Simulation ajoute1554pixels du contrôle Budget/Fiscalité dans l’en-tête fixe capturé. Ni référence ni seuil abaissés. Rejouer en Chromium CI et revoir/régénérer la référence Simulation avant fusion. Aucune CI distante déclenchée et aucune CI globale déclarée verte.

Livrables de revue dans `/workspace/output` : rapport, inventaire, patch incluant les fichiers nouveaux, journaux sélectionnés et12captures des nouveaux écrans. Les bases et serveurs de test de cette mission sont arrêtés après les derniers contrôles ; seules des données synthétiques y ont été utilisées.
