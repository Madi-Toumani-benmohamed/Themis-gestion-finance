# Plan : Externalisation d'Ollama vers le Mac hôte

## 1. Besoin reformulé

Rendre l'instance Ollama locale optionnelle en permettant au backend et au service Knowledge Graph de pointer vers une instance Ollama distante hébergée nativement sur le Mac de l'utilisateur (au lieu du conteneur Docker). Supprimer la dépendance implicite au service `ollama` dans les configurations Docker, et documenter la procédure de configuration réseau pour Mac.

## 2. Décisions techniques

| Décision | Choix | Raison |
|----------|-------|---------|
| Langage de modification | Docker env vars + YAML | Aucun code Go/Python à changer — l'infrastructure lit déjà `OLLAMA_URL` depuis l'environnement |
| Paramétrage de l'URL Ollama dans knowledge-graph | Variable d'env `${REMOTE_OLLAMA_URL:-}` avec suffixe `/v1` ajouté en compose | Le service Python Graphiti attend le endpoint OpenAI (`/v1`), tandis que le backend Go attend `/api/chat`. On évite de dupliquer la logique de suffixe dans le code. |
| Réseau Docker → Mac hôte | `host.docker.internal` (Docker Desktop) | Solution standard, ne nécessite pas de connaître l'IP privée du Mac à l'avance. Compatible avec les deux compose files. |
| Suppression du service ollama local | Conserver le profile `local-ai` mais le rendre explicite | Le profile existe déjà. On garde la compatibilité descendante : sans profile, pas de conteneur Ollama. |
| Cohérence des deux compose files | Uniformiser le traitement de `OLLAMA_URL` | Actuellement, `compose-supabase/docker-compose.yml` ne définit pas `OLLAMA_URL` pour knowledge-graph (car il n'y a pas de service KG). Il faut ajouter le paramétrage dans les deux fichiers. |

## 3. Étapes

### Étape 1 — Paramétrage de `OLLAMA_URL` dans le service knowledge-graph
**Fichier modifié :** `docker-compose.yml`

Changer la ligne 43 :
```yaml
# Avant
OLLAMA_URL: http://ollama:11434/v1
# Après
OLLAMA_URL: ${REMOTE_OLLAMA_URL:-http://ollama:11434/v1}
```
Même changement dans `compose-supabase/docker-compose.yml` (ajouter le service knowledge-graph avec la même variable, ou laisser tel quel s'il n'y a pas de KG dans cette variante — à vérifier).

**Livrable :** Le service knowledge-graph lit son URL Ollama depuis une variable d'environnement, avec fallback vers l'ancien comportement containerisé.

### Étape 2 — Ajouter une entrée claire dans `.env` pour le mode distant
**Fichier modifié :** `.env`

Ajouter après la section Ollama (ligne 24) :
```bash
# URL Ollama distante (Mac hôte). Laisser vide pour utiliser le conteneur local (profile local-ai).
# Exemple : http://host.docker.internal:11434
REMOTE_OLLAMA_URL=
```
Le suffixe `/v1` est ajouté automatiquement par le compose pour le service knowledge-graph ; le backend Go utilise l'URL brute.

**Livrable :** Le `.env` contient une variable documentée pour le mode distant.

### Étape 3 — Adapter les dépendances du service app
**Fichier modifié :** `docker-compose.yml`

Vérifier que la section `depends_on` du service `app` ne mentionne pas `ollama`. Actuellement (ligne 69-73) :
```yaml
depends_on:
  db:
    condition: service_healthy
  knowledge-graph:
    condition: service_started
```
→ Déjà correct. **Aucun changement requis.** Cette étape sert de vérification et de documentation du constat.

### Étape 4 — S'assurer que les deux compose files sont cohérents
**Fichier modifié :** `compose-supabase/docker-compose.yml`

Ce fichier ne contient pas de service `knowledge-graph` ni `ollama`. Il faut :
- Soit l'ajouter avec le profile `local-ai` pour les utilisateurs Supabase qui veulent le KG
- Soit documenter que le KG n'est pas supporté dans cette variante

**Décision à prendre :** Ajouter le service knowledge-graph dans `compose-supabase/docker-compose.yml` avec le profile `local-ai`, en réutilisant la même variable `${REMOTE_OLLAMA_URL}`. Cela rend les deux variantes fonctionnellement équivalentes.

**Livrable :** Les deux fichiers compose partagent la même logique de configuration Ollama.

### Étape 5 — Documentation : guide de configuration réseau Mac
**Fichier modifié :** `README.md` (section « IA locale avec Ollama »)

Ajouter une sous-section « Ollama distant sur le Mac » expliquant :
1. Comment trouver l'IP privée du Mac : `ipconfig getifaddr en0` (WiFi) ou `en1` (Ethernet)
2. Configurer Ollama pour écouter sur toutes les interfaces : `OLLAMA_ORIGINS="*" ollama serve`
3. Définir dans `.env` : `REMOTE_OLLAMA_URL=http://192.168.x.x:11434` (ou `http://host.docker.internal:11434`)
4. Lancer sans le profile local-ai : `docker compose up -d` (le conteneur ollama n'est pas démarré)
5. Vérifier : `curl http://localhost:8080/api/health` → `ai_provider: ollama`

**Livrable :** Un guide opérationnel dans le README.

### Étape 6 — Tests
**Action :** Lancer la suite de tests existante.

```bash
cd backend && go test ./...
```

Les tests unitaires (incluant `ollama_fallback_test.go` et `thinking_test.go`) utilisent `httptest` et ne nécessitent pas d'instance Ollama réelle. Ils doivent passer tels quels.

**Vérification supplémentaire :** Démarrer l'appli avec `REMOTE_OLLAMA_URL=http://host.docker.internal:11434` et confirmer que le ping Ollama au démarrage réussit (log : pas d'alerte « ATTENTION — Ollama injoignable »).

**Livrable :** `go test ./...` vert + vérification manuelle du ping.

## 4. Règles à respecter

1. **Ne jamais committer de secrets** : `REMOTE_OLLAMA_URL` va dans `.env` (ignoré par Git), pas dans les fichiers YAML.
2. **Garder la compatibilité descendante** : si `REMOTE_OLLAMA_URL` est vide, le comportement doit être identique à avant (fallback vers `http://ollama:11434/v1` pour le KG, et le profile `local-ai` démarre le conteneur Ollama).
3. **Ne pas modifier le code Go/Python** : toute la configuration passe par des variables d'environnement, conformément à l'architecture existante.
4. **Suffixe `/v1`** : le service knowledge-graph (Python/Graphiti) utilise l'API OpenAI-compatible (`/v1/chat/completions`, `/v1/embeddings`), donc son `OLLAMA_URL` doit finir par `/v1`. Le backend Go utilise `/api/chat` directement, donc son `OLLAMA_URL` ne doit PAS avoir de suffixe. C'est pourquoi on paramètre séparément.
5. **Pas de modification des tests** : les tests existants ne doivent pas être modifiés, seulement exécutés.

## 5. Critères d'achèvement par étape

| Étape | Critère de finition |
|-------|---------------------|
| 1 | `docker-compose.yml` ligne 43 utilise `${REMOTE_OLLAMA_URL:-http://ollama:11434/v1}` au lieu de la valeur en dur |
| 2 | `.env` contient la variable `REMOTE_OLLAMA_URL` commentée avec la doc |
| 3 | Confirmation par lecture que `depends_on` du service `app` n'inclut pas `ollama` |
| 4 | `compose-supabase/docker-compose.yml` contient le service knowledge-graph avec le même pattern de variable |
| 5 | README contient la sous-section « Ollama distant sur le Mac » avec les 5 points du guide |
| 6 | `go test ./...` retourne 0 échec ; log de démarrage sans alerte Ollama quand l'URL distante est configurée |
