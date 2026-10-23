"""
Knowledge Graph Service — Thémis

Expose une API REST légère qui alimente un graphe de connaissances
financières via Graphiti. Le backend Go appelle ce service pour :
  - Learn : ingérer des faits financiers (transactions, habitudes)
  - Query : récupérer les faits pertinents avant une réponse chat

Ce n'est PAS un système de mémoire conversationnelle.
On n'y stocke que des faits financiers structurés, pas les conversations.

Variables d'environnement :
  FALKORDB_HOST     – hôte FalkorDB    (défaut: falkordb)
  FALKORDB_PORT     – port FalkorDB    (défaut: 6379)
  OLLAMA_URL        – endpoint Ollama  (défaut: http://ollama:11434/v1)
  OLLAMA_MODEL      – modèle LLM       (défaut: qwen2.5:14b)
  OLLAMA_EMBED_MODEL– modèle embedding (défaut: nomic-embed-text)
  KG_SERVICE_PORT   – port API         (défaut: 8100)
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
from graphiti_core.llm_client.openai_generic_client import OpenAIGenericClient
from graphiti_core.llm_client.config import LLMConfig
from graphiti_core.embedder.openai import OpenAIEmbedder, OpenAIEmbedderConfig
from graphiti_core.cross_encoder.openai_reranker_client import OpenAIRerankerClient

# ────────────────────────── Config ──────────────────────────

FALKORDB_HOST = os.getenv("FALKORDB_HOST", "falkordb")
FALKORDB_PORT = int(os.getenv("FALKORDB_PORT", "6379"))

OLLAMA_URL = os.getenv("OLLAMA_URL", "http://ollama:11434/v1")
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "qwen2.5:14b")
OLLAMA_EMBED_MODEL = os.getenv("OLLAMA_EMBED_MODEL", "nomic-embed-text")

PORT = int(os.getenv("KG_SERVICE_PORT", "8100"))

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("knowledge-graph")

# ────────────────────────── Models ──────────────────────────

class LearnFactsInput(BaseModel):
    """Ingérer des faits financiers à partir d'un épisode structuré."""
    group_id: str                      # owner_id
    facts: list[dict]                  # [{"fact": "...", "type": "merchant"|"habit"|"preference"}]
    source: str = "transaction_import" # source du fait

class QueryInput(BaseModel):
    """Requête de recherche dans le graphe de connaissances."""
    query: str
    group_id: str
    top_k: int = 5

class KGStatus(BaseModel):
    ok: bool
    node_count: int
    error: Optional[str] = None

# ────────────────────────── Graphiti init ─────────────────────

graphiti: Optional[Graphiti] = None
_start_time: Optional[datetime] = None


def _build_clients():
    llm_config = LLMConfig(
        api_key="ollama",
        model=OLLAMA_MODEL,
        small_model=OLLAMA_MODEL,
        base_url=OLLAMA_URL,
        temperature=0.1,
        max_tokens=16384,
    )
    embedder = OpenAIEmbedder(config=OpenAIEmbedderConfig(
        api_key="ollama",
        embedding_model=OLLAMA_EMBED_MODEL,
        embedding_dim=768,
        base_url=OLLAMA_URL,
    ))
    llm_client = OpenAIGenericClient(config=llm_config)
    reranker = OpenAIRerankerClient(client=llm_client, config=llm_config)
    return llm_client, embedder, reranker


def init_graphiti() -> Graphiti:
    from graphiti_core.driver.falkordb_driver import FalkorDriver
    driver = FalkorDriver(host=FALKORDB_HOST, port=FALKORDB_PORT)
    llm, embedder, reranker = _build_clients()
    g = Graphiti(graph_driver=driver, llm_client=llm, embedder=embedder, cross_encoder=reranker)
    return g


async def _ensure_init():
    global graphiti, _start_time
    if graphiti is None:
        log.info("initialisation KG (%s)", OLLAMA_MODEL)
        try:
            graphiti = init_graphiti()
            await graphiti.build_indices_and_constraints()
            _start_time = datetime.now(timezone.utc)
            log.info("KG prêt")
        except Exception as e:
            log.error("échec init KG: %s", e)
            raise

# ────────────────────────── FastAPI ──────────────────────────

app = FastAPI(title="Themis Knowledge Graph", version="1.0.0")

app.add_middleware(CORSMiddleware, allow_origins=["*"], allow_methods=["*"], allow_headers=["*"])


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


@app.post("/kg/learn")
async def learn_facts(inp: LearnFactsInput) -> dict:
    """
    Ingère des faits financiers sous forme d'épisodes.
    Chaque fait devient une relation extraite dans le graphe.
    """
    await _ensure_init()
    try:
        # Construire un body textuel à partir des faits
        fact_lines = "\n".join(f"- {f['fact']}" for f in inp.facts)
        body = f"Faits financiers appris :\n{fact_lines}"

        result = await graphiti.add_episode(
            name=f"Financial facts from {inp.source}",
            episode_body=body,
            source_description=inp.source,
            reference_time=datetime.now(timezone.utc),
            group_id=inp.group_id,
            update_communities=True,
        )
        log.info("learned %d entities, %d edges for group %s", len(result.nodes), len(result.edges), inp.group_id)
        return {
            "entities_added": len(result.nodes),
            "edges_added": len(result.edges),
        }
    except Exception as e:
        log.error("learn error: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/kg/query")
async def query_kg(inp: QueryInput) -> dict:
    """Recherche hybride (vectoriel + texte) dans le graphe de connaissances."""
    await _ensure_init()
    try:
        edges = await graphiti.search(
            query=inp.query,
            group_ids=[inp.group_id],
            num_results=inp.top_k,
        )
        results = []
        for edge in edges:
            results.append({
                "fact": edge.fact,
                "source": edge.source_node_uuid,
                "target": edge.target_node_uuid,
                "type": edge.relationship,
                "score": edge.relevance_score,
            })
        return {"results": results, "count": len(results)}
    except Exception as e:
        log.error("query error: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/kg/status/{group_id}")
async def kg_status(group_id: str) -> dict:
    """Statut du graphe pour un utilisateur."""
    await _ensure_init()
    try:
        from falkordb import FalkorDB
        client = FalkorDB(host=FALKORDB_HOST, port=FALKORDB_PORT)
        graph = client.select_graph("graphiti")
        count_result = graph.query(f"MATCH (n {{group_id: '{group_id}'}}) RETURN count(n) AS cnt")
        rows = list(count_result.rows)
        node_count = int(rows[0][0]) if rows else 0
        return {"ok": True, "node_count": node_count, "group_id": group_id}
    except Exception as e:
        return {"ok": False, "error": str(e), "node_count": 0}


@app.post("/kg/clear/{group_id}")
async def clear_kg(group_id: str) -> dict:
    """Supprime toutes les connaissances d'un utilisateur."""
    await _ensure_init()
    try:
        from falkordb import FalkorDB
        client = FalkorDB(host=FALKORDB_HOST, port=FALKORDB_PORT)
        graph = client.select_graph("graphiti")
        graph.query(f"MATCH (n {{group_id: '{group_id}'}}) DETACH DELETE n")
        return {"cleared": True}
    except Exception as e:
        log.error("clear error: %s", e)
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    import uvicorn
    log.info("démarrage KG service sur port %d", PORT)
    uvicorn.run(app, host="0.0.0.0", port=PORT, log_level="info")
