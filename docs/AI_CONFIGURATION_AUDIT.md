# Configuration IA — audit et plan de consolidation

Reprise le 25 octobre 2026 dans le worktree courant `/home/localadm/projets/Themis-gestion-finance`, branche `feat/ai-configuration-hardening`. `AGENTS.md` utilisateur modifié localement est conservé. Instruction initiale : aucune publication/fusion automatique. Instruction ultérieure explicite : publier une PR puis fusionner seulement après tests locaux, CI et sécurité réussis, avant d’entamer la mission Relevés. Aucun autre worktree n’est modifié.

## Existant vérifié

- `Admin.jsx` : configuration IA et maintenance/utilisateurs/mot de passe dans le même composant. Six choix : désactivé, Ollama, OpenAI, Anthropic, Mistral, FreeLLM/OpenRouter.
- `GET/PUT /api/admin/config`, `POST /api/admin/ai/test`, `GET /api/admin/ollama/models` existent. La découverte Ollama et le test avant enregistrement sont réutilisables.
- `admin.go` : `AIConfig`, client activé à chaud, configuration globale en JSON `settings.ai_config`. Les clés sont stockées en clair dans la base et masquées dans le DTO de lecture, avec repli environnement. L’abstraction `llm.Client` est conservée.
- `memory.Client` : activation par environnement, contrôle `/health`, service Graphiti/FalkorDB facultatif. Aucun contrôle destructeur à ajouter. Le test cloud actuel envoie déjà uniquement un prompt synthétique ; OpenAI ignore les corps d’erreur sensibles.
- Tests Go avec transport/stockage simulés, tests de contexte et clients LLM ; Node natif et Playwright/Axe existent pour le frontend.

## Anomalies confirmées

| Priorité | Cause | Correction implémentée |
|---|---|---|
| P0 | Le middleware autorise l’API avant configuration initiale ; les routes IA administratives n’ont pas de garde supplémentaire. | Garde administrative explicite sur les routes de configuration/diagnostic IA, tests routeur/utilisateur/admin/bootstrap. |
| P0 | `InitAI` lit `ai_config` sans `owner_id IS NULL`, alors que les réglages génériques permettent une clé utilisateur identique. | Lecture globale explicite ; réserver cette clé aux endpoints administratifs ; conserver les anciennes lignes sans les modifier. |
| P0 | URL Ollama/FreeLLM et découverte sans contrôle SSRF, clients suivant les redirections et proxys ambiants. | Validation centralisée, refus des métadonnées/réseaux non publics, DNS vérifié et connexion à l’IP vérifiée ; autorisations de bases réseau précises fournies par l’environnement pour les serveurs locaux/distants légitimes. Aucun secret dans une URL. |
| P0 | Anthropic/Mistral/FreeLLM/Ollama remontent des erreurs externes brutes ; modèles et URL peuvent aussi entrer dans les logs. | Erreurs structurées et messages bornés sans corps fournisseur ; logs limités au fournisseur ; tests de secrets et redirections. Préserver les replis Ollama sur thinking. |
| P1 | Conservation des clés Mistral/FreeLLM faite après validation ; test ne restaure pas toutes les clés ; seules deux clés sont effacées du formulaire après sauvegarde. | Fusion des champs avant validation ; conservation/suppression explicite des clés et indication de leur origine, sans retour du secret. |
| P1 | PUT partiel réinitialise d’autres paramètres ; erreur JSON de test retombe silencieusement sur le client actif ; découverte remplace automatiquement le modèle choisi. | Paramètres absents conservés, décodage strict, test sans mutation, saisie manuelle conservée après découverte. |
| P1 | Pas de validation réelle des modèles/URL/contexte, sauvegardes concurrentes et tests obsolètes possibles. | Révision opaque, contrôle de concurrence, résultats datés et liés à la connexion testée, rejet des entrées invalides. |
| P1 | Statut « actif » assimilable à connectivité ; aucun diagnostic mémoire configurée/désactivée/non testée. | Activation distincte du succès d’un contrôle explicite ; latence mesurée seulement lors d’un test ; mémoire indépendante et facultative. |
| P2 | Bloc long, labels incomplets, chargement initial bloqué si GET échoue, états de formulaire non distincts. | Composant IA séparé, quatre panneaux, boutons explicites, brouillon/sauvegarde/test distincts, erreurs et responsive accessibles. |

## Lots retenus

1. Sécurité et invariants : configuration globale, autorisations, fusion/validation, transport réseau contrôlé, messages sûrs. Aucun nouveau fournisseur ni nouvelle dépendance.
2. Diagnostics sur les endpoints existants, plus contrôle mémoire administratif ciblé si nécessaire. Configuration persistante ; résultats de contrôle en mémoire du processus, redevenant non testés au redémarrage. Enregistrer applique la configuration comme auparavant, sans déclarer la connexion vérifiée.
3. Interface Fournisseur/connexion, Comportement, Diagnostic et Mémoire. Autres fonctions d’administration conservées, sans réinitialiser un brouillon lors d’une action de maintenance.
4. Tests unitaires/PostgreSQL sur données synthétiques, Node, Playwright/Axe, Go/vet, build frontend. Documentation finale avec limites de stockage des clés, politique réseau et absence de garantie absolue contre les injections de prompt.

## Bilan de consolidation

Tous les correctifs du tableau sont implémentés et couverts par les tests décrits dans `AI_CONFIGURATION_TESTS.md`. Le chargement initial, les conflits entre administrateurs, les erreurs réseau, les brouillons et les états de mémoire sont explicités. L’abstraction LLM, les six choix et les actions administratives existantes sont conservés. Aucune migration ou donnée financière modifiée.

### Risques résiduels et limites vérifiées

- Les secrets restent en clair dans `settings` et les sauvegardes : aucun coffre/chiffrement applicatif ajouté. Le DTO, les logs et les erreurs des fournisseurs ne les exposent pas.
- Une base privée héritée uniquement de la DB doit être autorisée côté serveur avant de redevenir utilisable. Les bases d’environnement déjà configurées restent compatibles. Aucun accès aux métadonnées/link-local, même autorisé explicitement.
- Diagnostics et révisions propres au processus, sans historique durable ni coordination multi-instance. Le redémarrage conserve les paramètres mais remet les contrôles à « Non testé ».
- Le contrôle Ollama prouve l’accès au service et la présence du modèle, sans lancer une génération. Le contrôle KG prouve uniquement la réponse de `/health`, pas la qualité du graphe ou le bon fonctionnement de chaque dépendance.
- Aucun fournisseur réel ni graphe réel contacté lors des tests. Les contrats sont vérifiés avec mocks et serveurs synthétiques. L’accessibilité automatisée ne remplace pas une revue manuelle complète ; elle cible les quatre panneaux IA dans les thèmes clair/sombre.
- Le contexte reste une note globale déclarative et non une source financière. L’échappement et la séparation des rôles ne garantissent pas une résistance absolue aux injections de prompt.
- Les autres routes conservent le fonctionnement bootstrap existant. Cette intervention durcit les cinq routes IA, sans refonte globale de l’authentification.
- Avertissement de taille de bundle Vite existant ; la configuration Supabase n’est validée que syntaxiquement sans les variables de déploiement de cet environnement.
