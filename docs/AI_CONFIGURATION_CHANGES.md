# Configuration IA — changements et exploitation

25 octobre 2026 — branche `feat/ai-configuration-hardening`, worktree courant `/home/localadm/projets/Themis-gestion-finance`. Les modifications préexistantes d’`AGENTS.md` restent intactes. Le dernier message utilisateur autorise la publication de la PR et son merge une fois les tests locaux, CI et sécurité réussis, puis la reprise de Relevés. Aucun déploiement automatique demandé pour ce chantier.

## Interface

`frontend/src/components/AIConfiguration.jsx`, `AIConfiguration.css` et `aiConfigurationModel.mjs` isolent la configuration du reste d’`Admin.jsx`. Quatre panneaux accessibles au clavier :

1. Fournisseur et connexion : les six choix existants, champs conditionnels, modèle libre et découverte Ollama explicite, clés masquées côté serveur, remplacement/retrait explicite.
2. Comportement : suggestions à l’import et contexte personnel global déclaratif, limite de 1 000 caractères Unicode. Les données transactionnelles et les calculs structurés restent prioritaires.
3. Diagnostic : configuration enregistrée, activation des fonctions, source des réglages, résultat du dernier contrôle et durée réellement mesurée. Le brouillon testé est distingué de la connexion enregistrée.
4. Mémoire : activation/configuration effectives côté serveur, contrôle de santé volontaire, service facultatif. Aucune action destructive.

L’enregistrement persiste et applique les paramètres, comme auparavant. Le test ne les enregistre pas. Une sélection ou une sauvegarde ne prouve jamais la disponibilité du fournisseur. Aucun test ni découverte n’est lancé à l’ouverture de la page. Les derniers contrôles ne garantissent pas la disponibilité future.

Les chargements peuvent être relancés ; les erreurs conservent le brouillon, notamment la nouvelle clé saisie après une sauvegarde échouée. Les contrôles obsolètes sont annulés/ignorés. Un conflit de révision propose un rechargement explicite avec avertissement de remplacement du brouillon. Le formulaire est bloqué pendant sauvegarde/rechargement. Les clés saisies disparaissent du formulaire après succès ou annulation.

`Admin.jsx` conserve statistiques, utilisateurs, maintenance, export et mot de passe personnel. Ces actions n’écrasent plus le brouillon IA. Les fonctions réservées aux administrateurs sont masquées pour les autres utilisateurs. `App.jsx` répercute l’activation IA après sauvegarde et élargit seulement cette page. Les styles restent limités à l’administration ; aucune dépendance ajoutée.

## API et stockage

- `admin.go` conserve `AIConfig`, les clients et les actions existantes.
- `ai_configuration.go` centralise garde administrateur, validation, fusion des champs, persistance et DTO masqué. `InitAI` lit exclusivement la ligne globale. Les lignes utilisateur existantes sont conservées et ne peuvent plus masquer les paramètres globaux.
- `ai_diagnostics.go` réutilise les endpoints test/découverte et ajoute uniquement `POST /api/admin/ai/memory/test`, nécessaire pour consulter explicitement la santé du service déjà configuré.
- `handlers.go` applique la garde aux cinq routes IA, même pendant le bootstrap. Les routes génériques `/api/settings/ai_config` refusent accès/écriture ; ce réglage global passe désormais par l’API administrative.

Les champs et routes existants sont conservés. Ajouts au DTO : `enabled`, `revision`, `source`, `configuration_error`, `key_sources`, `diagnostic`, `memory`. Aucune valeur secrète n’est renvoyée. Les indicateurs historiques `*_key_set` restent présents. Les erreurs de fournisseur sont classées sans texte externe : authentification, accès au modèle, quota, timeout, réseau, réponse invalide, modèle Ollama absent.

PUT conserve les champs absents et les clés vides. `clear_keys: ["openai_key", ...]` retire explicitement les clés stockées ; une clé d’environnement reste utilisable. Remplacer et retirer simultanément une même clé est refusé. `expected_revision` est facultatif pour les anciens clients ; l’interface le transmet et reçoit HTTP 409 si la configuration a changé. Une erreur SQL préserve le client et la configuration actifs. Les diagnostics ne sont pas enregistrés en base : ils redeviennent non testés au redémarrage, contrairement aux paramètres persistants.

## Sécurité réseau et Ollama distant

`llm/network.go` protège les destinations configurables Ollama/FreeLLM. HTTP(S) uniquement, sans identifiants, query ou fragment ; HTTPS requis par défaut. Le DNS est vérifié au moment de chaque nouvelle connexion, puis la connexion utilise l’IP vérifiée. Les réponses DNS mixtes public/privé non autorisées sont refusées. Les redirections et proxys ambiants sont désactivés pour ces endpoints.

Un opérateur peut autoriser une base précise privée/HTTP :

```dotenv
# Exemple de configuration opérateur, sans secret ni accès automatique
AI_TRUSTED_ENDPOINTS=http://192.168.1.50:11434,http://passerelle-interne:3001/v1
```

Les bases effectives `OLLAMA_URL` et `FREELLM_URL` côté serveur sont également autorisées. L’autorisation porte sur le schéma, hôte, port et chemin de base exacts ; elle ne donne pas accès à tout le réseau. Les destinations de métadonnées, link-local, multicast et non spécifiées restent interdites, même dans cette liste. Les adresses privées/loopback et CGNAT nécessitent l’autorisation. Les certificats TLS restent vérifiés.

Une ancienne base privée stockée seulement en DB et absente de l’environnement/liste reste conservée, mais la connexion est bloquée tant que l’opérateur ne l’a pas autorisée. Le panneau affiche l’erreur ; aucun remplacement silencieux du serveur. Les deux Compose transmettent désormais les paramètres nécessaires, y compris Mistral/FreeLLM et `AI_TRUSTED_ENDPOINTS`. `example.env` explique ce réglage ; le `.env` réel n’a pas été modifié.

Les tests cloud utilisent un message synthétique sans contexte ni information bancaire, plafonné à 30 secondes ; ils peuvent consommer des jetons. Ollama teste l’API des modèles et leur présence en 5 secondes, sans garantir la génération : la réserve est affichée. La découverte est limitée à 10 secondes. Les délais longs existants de génération Ollama sont préservés. Le contrôle mémoire est limité à 5 secondes et vérifie HTTP 200 + JSON de santé, sans suivre une redirection.

## Autres fichiers partagés

`config/config.go` charge la liste réseau. Les cinq clients LLM conservent leur interface, utilisent des erreurs sûres et acceptent un transport injecté pour les tests. `prompt_safety.go`, `ai/analyze.go` et `api/anomalies.go` bornent le contexte Unicode et échappent les balises susceptibles de fermer son bloc. Ce sont des protections de prompt, sans modification des calculs financiers. `memory/memory.go` expose l’activation/configuration sans IO et fiabilise le contrôle de santé, y compris les URL avec slash final.

`api.js` transmet les signaux d’annulation et évite le cache des lectures de configuration. `.github/workflows/ci.yml` ajoute le parcours navigateur IA aux cinq parcours existants, sans supprimer/assouplir de contrôle. `CODEBASE_GRAPH.md` documente les nouvelles responsabilités.

## Compatibilité et limites

Aucune migration, aucun nouveau fournisseur, aucun service/framework supplémentaire, aucune modification de transaction, catégorie ou formule financière. L’interface `llm.Client` et le format persistant sont conservés. Les clés restent stockées en clair en base comme auparavant : chiffrement au repos, sauvegardes et accès DBA restent à traiter dans une conception distincte.

La révision et les diagnostics sont propres au processus : coordination distribuée entre plusieurs instances non ajoutée. La mémoire demeure configurée par l’environnement. Son contrôle valide seulement le service `/health`, pas la qualité des faits ni chaque dépendance du graphe. Le cloisonnement/échappement du contexte réduit les risques de prompt injection sans constituer une garantie absolue. Le bootstrap d’authentification des autres fonctionnalités n’a pas été refondu.

Prochaines itérations possibles, sans implémentation automatique : coffre/chiffrement des secrets avec rotation, coordination multi-instance et politique de sortie réseau opérateur plus restrictive. La découverte des modèles reste Ollama uniquement ; les autres modèles restent saisissables.
