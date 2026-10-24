# Complément de validation du déploiement

Le workflow Security ajouté sur main a révélé des alertes dans des fichiers existants lors de la PR Transactions. Les seuils Trivy HIGH/CRITICAL et le mode bloquant Semgrep restent inchangés ; aucune règle globale ni aucun contrôle CI n'est désactivé.

## Corrections

- Images PostgreSQL d'initialisation, frontend autonome et mémoire : utilisateur non root explicite. Le frontend Nginx autonome écoute désormais sur8080 et place son PID dans `/tmp`, avec cache accessible à son utilisateur101. L'image applicative unique et son port8080 restent inchangés.
- Pods : identité non root explicite, seccomp RuntimeDefault, absence d'élévation de privilèges et capacités retirées. PostgreSQL16 Alpine utilise l'UID/GID70 vérifié dans l'image, avec répertoires de données, sockets et temporaires distincts. Les systèmes de fichiers racine concernés sont en lecture seule ; les volumes nécessaires restent accessibles.
- Le pod de consultation des sauvegardes monte le volume en lecture seule. Le CronJob utilise des permissions0600, vérifie les erreurs du pipeline et publie le fichier final par renommage seulement après succès. Un échec de dump ne devient pas une sauvegarde complète apparente.
- Les séries par tag utilisent des paramètres PostgreSQL pour la granularité, le format et le tableau de tags ; le filtre de propriétaire et les exclusions financières partagés sont appliqués à la recherche des tags et aux séries. La recherche du chat utilise également une requête fixe et des paramètres liés.
- L'ancien diagnostic Python conserve la bibliothèque standard : origines HTTP(S) contrôlées, certificats TLS et noms d'hôte vérifiés, cookies effectivement associés aux requêtes. Les URL `file://`, les autres origines et les identifiants intégrés dans l'URL sont refusés. Les nouveaux tests réseau sont isolés et ne contactent pas l'application configurée.

## Alertes d'audit revues

Quatre annotations `nosemgrep` sont précises, documentées dans le code et limitées à des faux positifs :

1. `analytics.go`, `chat.go`, `period.go` : le texte SQL concatène uniquement des constantes de compilation, dont le prédicat financier partagé ; toutes les données externes sont des paramètres PostgreSQL. Le contrôle d'injection par tag avec apostrophe et chaîne ressemblant à du SQL, l'isolation utilisateur et les filtres sont testés sur PostgreSQL.
2. `equipe/essais/test_navigation_top.py` : l'alerte générique HTTPSConnection demande une revue de la vérification TLS. Le contexte est explicitement `ssl.create_default_context()`. Un test vérifie `CERT_REQUIRED` et `check_hostname`, et les schémas/origines non autorisés sont refusés avant tout appel réseau.

L'ancien contexte SSL non vérifié est supprimé. Les annotations ne couvrent ni des SQL fournis par l'utilisateur ni une connexion TLS non vérifiée.

## Vérifications réalisées

- Semgrep `p/security-audit --error` : aucune alerte active après revue des quatre faux positifs ci-dessus.
- Trivy0.70.0, scanners vuln/misconfig, HIGH/CRITICAL, erreurs bloquantes : aucune alerte aux seuils retenus.
- Suite Go/PostgreSQL isolée avec `-race`, puis `go vet` : réussite ; régression des séries mensuelles/annuelles et tags liés ajoutée.
- Deux tests Python de transport isolé : réussite, repris dans la CI existante.
- Images mémoire et frontend autonome construites ; identité non root vérifiée, `nginx -t` réussi et endpoints statique/mémoire HTTP200 sur des ports locaux de test.
- PostgreSQL démarre et exécute les tests avec UID/GID70, sans capacités, sans élévation de privilèges et avec racine en lecture seule. La commande réelle du CronJob est essayée sur cette base isolée : succès avec fichier0600 ; échec forcé sans fichier final ni fichier partiel publié.
- Manifestes YAML parsés, workflow contrôlé par actionlint. Aucune application de ces manifestes sur un cluster n'est effectuée. L'image PostgreSQL historique contenant un script d'initialisation n'est pas reconstruite à partir de ce script pour les tests.

## Limites de déploiement

Les volumes Kubernetes existants doivent supporter les permissions et `fsGroup` indiqués ; leur propriétaire et le comportement du pilote de stockage nécessitent une validation avant un déploiement effectif. Les noms des PVC sont conservés et aucune donnée de volume réelle n'a été modifiée pour les tests. Les essais Docker isolés ne remplacent pas une validation sur le cluster cible.

Pour le frontend autonome, adapter un éventuel ancien mapping `80:80` à `80:8080`. La stack principale continue à servir l'application unique sur son port habituel. Les scans effectués ne constituent pas une garantie exhaustive de sécurité ni une validation de toutes les images distantes d'exécution.
