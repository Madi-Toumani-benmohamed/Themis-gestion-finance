"""
Memory Service — interface Graphiti pour Themis.

Expose une API REST (FastAPI) qui manipule un graphe de connaissances
temporel via Graphiti. Le backend Go appelle ce service pour :
  - Search : récupérer des faits pertinents avant une réponse chat
  - AddEpisode : persister les conversations et les imports de transactions
  - Status    : vérifier la santé du graphe

Variables d'environnement :
  FALKORDB_HOST     – hôte FalkorDB    (défaut: falkordb)
  FALKORDB_PORT     – port FalkorDB    (défaut: 6379)
  FALKORDB_PASSWORD – mot de passe     (défaut: "")
  OLLAMA_URL        – endpoint Ollama  (défaut: http://ollama:11434/v1)
  OLLAMA_MODEL      – modèle LLM       (défaut: qwen2.5:14b)
  OLLAMA_EMBED_MODEL– modèle embedding (défaut: nomic-embed-text)
  MEMORY_SERVICE_PORT– port API         (défaut: 8100)
"""

import os
import asyncio
import logging
from datetime import datetime, timezone
from typing import Optional

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from graphiti_core import Graphiti
from graphiti_core.nodes import EpisodeType
from graphiti_core.llm_client.openai_generic_client import OpenAIGenericClient
from graphiti_core.llm_client.config import LLMConfig
from graphiti_core.embedder.openai import OpenAIEmbedder, OpenAIEmbedderConfig
from graphiti_core.cross_encoder.openai_reranker_client import OpenAIRerankerClient

# ────────────────────────── Config ──────────────────────────

FALKORDB_HOST = os.getenv("FALKORDB_HOST", "falkordb")
FALKORDB_PORT = int(os.getenv("FALKORDB_PORT", "6379"))
FALKORDB_PASSWORD = os.getenv("FALKORDB_PASSWORD", "")

OLLAMA_URL = os.getenv("OLLAMA_URL", "http://ollama:11434/v1")
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "qwen2.5:14b")
OLLAMA_EMBED_MODEL = os.getenv("OLLAMA_EMBED_MODEL", "nomic-embed-text")

PORT = int(os.getenv("MEMORY_SERVICE_PORT", "8100"))

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("memory")

# ────────────────────────── Models ──────────────────────────

class EpisodeInput(BaseModel):
    """Épisode à ingest dans le graphe."""
    name: str                      # court descriptif
    body: str                      # contenu textuel
    source_description: str        # source (conversation, import…)
    group_id: str                  # partition par user (owner_id)
    reference_time: Optional[str]  # ISO 8601, défaut now
    episode_type: str = "message"  # message | text | json

class SearchInput(BaseModel):
    """Requête de recherche dans le graphe."""
    query: str
    group_id: str
    num_results: int = 5

class AddEntityInput(BaseModel):
    """Ajouter directement une entité (sans passer par un épisode)."""
    name: str
    description: str
    entity_type: str = "Entity"
    group_id: str

class AddTripleInput(BaseModel):
    """Ajouter un triplet (sujet, prédicat, objet)."""
    source: str
    relation: str
    target: str
    group_id: str

class MemoryStatus(BaseModel):
    ok: bool
    db: str
    group_count: int
    uptime_seconds: float

# ────────────────────────── Graphiti init ─────────────────────

graphiti: Optional[Graphiti] = None
_start_time: Optional[datetime] = None


def _build_clients():
    """Construit les clients LLM/embedding pointant sur Ollama."""
    llm_config = LLMConfig(
        api_key="ollama",
        model=OLLAMA_MODEL,
        small_model=OLLAMA_MODEL,
        base_url=OLLAMA_URL,
        structured_output_mode="json_object",  # plus tolérant pour les petits modèles
    )
    llm_client = OpenAIGenericClient(config=llm_config)

    embedder_config = OpenAIEmbedderConfig(
        api_key="ollama",
        embedding_model=OLLAMA_EMBED_MODEL,
        embedding_dim=768,  # nomic-embed-text
        base_url=OLLAMA_URL,
    )
    embedder = OpenAIEmbedder(config=embedder_config)

    reranker = OpenAIRerankerClient(client=llm_client, config=llm_config)

    return llm_client, embedder, reranker, llm_config


def init_graphiti() -> Graphiti:
    """Initialise Graphiti avec FalkorDB + Ollama."""
    from graphiti_core.driver.falkordb_driver import FalkorDriver

    driver_kwargs = {"host": FALKORDB_HOST, "port": FALKORDB_PORT}
    if FALKORDB_PASSWORD:
        driver_kwargs["password"] = FALKORDB_PASSWORD

    driver = FalkorDriver(**driver_kwargs)

    llm_client, embedder, reranker, llm_config = _build_clients()

    g = Graphiti(
        graph_driver=driver,
        llm_client=llm_client,
        embedder=embedder,
        cross_encoder=reranker,
    )
    return g


async def _ensure_init():
    global graphiti, _start_time
    if graphiti is None:
        log.info("initialisation Graphiti (%s → %s:%d)", OLLAMA_MODEL, FALKORDB_HOST, FALKORDB_PORT)
        try:
            graphiti = init_graphiti()
            await graphiti.build_indices_and_constraints()
            _start_time = datetime.now(timezone.utc)
            log.info("Graphiti prêt")
        except Exception as e:
            log.error("échec initialisation Graphiti: %s", e)
            raise


# ────────────────────────── FastAPI ──────────────────────────

app = FastAPI(title="Themis Memory Service", version="1.0.0")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.on_event("startup")
async def startup():
    await _ensure_init()


@app.get("/health")
async def health():
    try:
        await _ensure_init()
        return {"status": "ok"}
    except Exception as e:
        raise HTTPException(status_code=503, detail=str(e))


@app.post("/memory/episodes")
async def add_episode(inp: EpisodeInput) -> dict:
    """Ajoute un épisode et extrait entités/relations."""
    await _ensure_init()
    ref_time = (
        datetime.fromisoformat(inp.reference_time)
        if inp.reference_time
        else datetime.now(timezone.utc)
    )
    try:
        result = await graphiti.add_episode(
            name=inp.name,
            episode_body=inp.body,
            source_description=inp.source_description,
            reference_time=ref_time,
            source=EpisodeType.message if inp.episode_type == "message" else EpisodeType.text,
            group_id=inp.group_id,
            update_communities=True,
        )
        return {
            "episode_uuid": result.episode.uuid,
            "entities_added": len(result.nodes),
            "edges_added": len(result.edges),
            "communities_updated": len(result.communities) if result.communities else 0,
        }
    except Exception as e:
        log.error("add_episode error: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/memory/search")
async def search_memory(inp: SearchInput) -> dict:
    """Recherche hybride (vectoriel + texte + traversal) dans le graphe."""
    await _ensure_init()
    try:
        edges = await graphiti.search(
            query=inp.query,
            group_ids=[inp.group_id],
            num_results=inp.num_results,
        )
        results = []
        for edge in edges:
            results.append({
                "fact": edge.fact,
                "source_entity": edge.source_node_uuid,
                "target_entity": edge.target_node_uuid,
                "relationship": edge.relationship,
                "relevance_score": edge.relevance_score,
            })
        return {"results": results, "count": len(results)}
    except Exception as e:
        log.error("search error: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/memory/entities")
async def add_entity(inp: AddEntityInput) -> dict:
    """Ajoute directement une entité sans passer par un épisode."""
    await _ensure_init()
    try:
        node = await graphiti.add_node(
            name=inp.name,
            description=inp.description,
            node_type=inp.entity_type,
            group_id=inp.group_id,
        )
        return {"node_uuid": node.uuid, "name": node.name}
    except Exception as e:
        log.error("add_entity error: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/memory/triplets")
async def add_triplet(inp: AddTripleInput) -> dict:
    """Ajoute un triplet sous forme de relation explicite."""
    await _ensure_init()
    try:
        # S'assurer que les nœuds existent
        src_node = await graphiti.get_node_by_name(inp.source, group_id=[inp.group_id])
        tgt_node = await graphiti.get_node_by_name(inp.target, group_id=[inp.group_id])

        if not src_node:
            src_node = await graphiti.add_node(
                name=inp.source, description=f"Merchant/source: {inp.source}",
                node_type="Merchant", group_id=inp.group_id
            )
        if not tgt_node:
            tgt_node = await graphiti.add_node(
                name=inp.target, description=f"Category/target: {inp.target}",
                node_type="Domain", group_id=inp.group_id
            )

        # Créer la relation via add_edge
        edge = await graphiti.add_edge(
            source_node_uuid=src_node.uuid,
            target_node_uuid=tgt_node.uuid,
            edge_type=inp.relation,
            fact=f"{inp.source} {inp.relation} {inp.target}",
            group_id=inp.group_id,
        )
        return {"edge_uuid": edge.uuid, "fact": edge.fact}
    except Exception as e:
        log.error("add_triplet error: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/memory/status/{group_id}")
async def memory_status(group_id: str) -> dict:
    """Statut du graphe pour un groupe donné."""
    await _ensure_init()
    try:
        # Requêter directement FalkorDB pour le comptage
        from falkordb import FalkorDB
        client = FalkorDB(host=FALKORDB_HOST, port=FALKORDB_PORT, password=FALKORDB_PASSWORD or None)
        graph = client.select_graph("graphiti")
        # Nombre de nodes avec group_id
        count_result = graph.query(f"MATCH (n {{group_id: '{group_id}'}}) RETURN count(n) AS cnt")
        rows = list(count_result.rows)
        node_count = int(rows[0][0]) if rows else 0
        return {
            "ok": True,
            "db": f"falkordb://{FALKORDB_HOST}:{FALKORDB_PORT}",
            "group_id": group_id,
            "node_count": node_count,
            "uptime_seconds": (datetime.now(timezone.utc) - _start_time).total_seconds() if _start_time else 0,
        }
    except Exception as e:
        return {"ok": False, "error": str(e)}


@app.post("/memory/clear/{group_id}")
async def clear_group(group_id: str) -> dict:
    """Supprime toutes les données d'un groupe (user)."""
    await _ensure_init()
    try:
        from falkordb import FalkorDB
        client = FalkorDB(host=FALKORDB_HOST, port=FALKORDB_PORT, password=FALKORDB_PASSWORD or None)
        graph = client.select_graph("graphiti")
        # Supprimer tous les nœuds du groupe
        graph.query(f"MATCH (n {{group_id: '{group_id}'}}) DETACH DELETE n")
        return {"cleared": True, "group_id": group_id}
    except Exception as e:
        log.error("clear error: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


# ────────────────────────── Main ────────────────────────────

if __name__ == "__main__":
    import uvicorn
    log.info("démarrage memory service sur port %d", PORT)
    uvicorn.run(app, host="0.0.0.0", port=PORT, log_level="info")
