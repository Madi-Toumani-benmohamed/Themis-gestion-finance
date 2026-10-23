# Revue finale — feat/ollama-remote
**Branche:** `feat/ollama-remote`  
**Date:** 2026-10-23  
**Objet:** Passage de `REMOTE_OLLAMA_URL` → `REMOTE_OLLAMA_BASE_URL` avec suffixe `/v1` auto pour le Knowledge Graph

## Synthèse
Le changement demandé est implémenté de façon cohérente dans les deux fichiers compose, `.env` et `README.md`. La logique de suffixe `/v1` est correctement externalisée dans le compose pour le service `knowledge-graph` uniquement. Aucune modification de code Go/Python n’a été introduite. Les tests unitaires restent verts.

---

## 1. `docker-compose.yml` — service app & knowledge-graph

**knowledge-graph**
- Ligne 43: `OLLAMA_URL: ${REMOTE_OLLAMA_BASE_URL:-http://ollama:11434}/v1`
  - ✅ Base URL configurable, suffixe `/v1` ajouté automatiquement après la substitution.
  - Avec `REMOTE_OLLAMA_BASE_URL=http://host.docker.internal:11434` → `http://host.docker.internal:11434/v1`
  - Sans variable → fallback `http://ollama:11434/v1`

**app**
- Ligne 61: `OLLAMA_URL: ${REMOTE_OLLAMA_BASE_URL:-${OLLAMA_URL:-}}`
  - ✅ URL brute, sans `/v1`. Fallback vers variable legacy `OLLAMA_URL` puis vide.
  - `depends_on` ne référence pas `ollama` : `db` healthy + `knowledge-graph` started.

Aucun profil `local-ai` sur `knowledge-graph`/`falkordb`, seul `ollama` garde `profiles: ["local-ai"]`. Découplage confirmé.

## 2. `compose-supabase/docker-compose.yml` — cohérence

**knowledge-graph**
- Ligne 21: `OLLAMA_URL: ${REMOTE_OLLAMA_BASE_URL:-http://ollama:11434}/v1`
  - ✅ Identique au fichier principal.

**app**
- Ligne 52: `OLLAMA_URL: ${REMOTE_OLLAMA_BASE_URL:-${OLLAMA_URL:-}}`
  - ✅ Même chaîne de fallback, sans `/v1`.
- `env_file: - ../.env` utilisé pour les deux services, variable substituée correctement.

Cohérence fonctionnelle entre les deux variantes confirmée.

## 3. `.env` — variable documentée

```bash
# URL Ollama distante (Mac hôte). Laisser vide pour utiliser le conteneur local (profile local-ai).
# Exemple : http://host.docker.internal:11434
REMOTE_OLLAMA_BASE_URL=
```
- ✅ Variable présente, commentée, sans valeur par défaut.
- ✅ Variable legacy documentée :
```bash
# Variable legacy (utilisée si REMOTE_OLLAMA_BASE_URL n'est pas définie).
# OLLAMA_URL=http://host.docker.internal:11434
```
Aucun secret commité.

## 4. `README.md` — documentation à jour

Section **« Ollama distant sur le Mac »** L213-249 :
- ✅ Étape 3 indique explicitement :
```bash
REMOTE_OLLAMA_BASE_URL=http://host.docker.internal:11434
```
- ✅ Mention : *« Le suffixe `/v1` est ajouté automatiquement par le compose pour le service Knowledge Graph ; le backend Go utilise l'URL brute. Une seule variable suffit — elle est propagée aux deux services. »*
- ✅ Commande de lancement sans profile `local-ai` et vérification `curl http://localhost:8080/api/health`.

La documentation suit le nouveau nom `REMOTE_OLLAMA_BASE_URL` et explique le mécanisme de suffixe.

## 5. Tests

- `docker compose config` et `docker compose -f compose-supabase/docker-compose.yml config` → syntaxe valide.
- `go test ./...` : 
  - `themis/internal/ai` OK
  - `themis/internal/api` OK
  - `themis/internal/llm` OK
  - `themis/internal/normalize` OK
  - `themis/internal/parser` OK
  - ✅ Tous les tests passent, aucun code Go modifié.

Vérification de propagation :
- Avec `REMOTE_OLLAMA_BASE_URL=http://host.docker.internal:11434` :
  - `app` → `http://host.docker.internal:11434` sans `/v1`
  - `knowledge-graph` → `http://host.docker.internal:11434/v1` avec `/v1`
- Sans variable :
  - `app` → vide
  - `knowledge-graph` → `http://ollama:11434/v1`

Comportement attendu respecté.

---

## Points d’attention mineurs
- Le `PLAN.md` et l’ancienne `REVUE.md` mentionnent encore `REMOTE_OLLAMA_URL`. Ce sont des artefacts documentaires, pas du code. Pas d’impact fonctionnel.
- La table « Configuration (env) » du README liste encore `OLLAMA_URL` sans mentionner `REMOTE_OLLAMA_BASE_URL`. Information utile mais non bloquante : la section dédiée « Ollama distant sur le Mac » est correcte.

Aucun défaut critique, majeur ou bloquant.

---

VERDICT : VALIDÉ
