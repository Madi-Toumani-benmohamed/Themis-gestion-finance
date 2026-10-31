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
from pydantic import BaseModel, Field, field_validator
from inspection import owner_group, serialize_edge, inspect_graph, relation_sources, failure_detail

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

class OwnedInput(BaseModel):
    group_id: str

    @field_validator("group_id")
    @classmethod
    def valid_group(cls, value):
        return owner_group(value)


class LearnFactsInput(OwnedInput):
    """Ingérer des faits financiers à partir d'un épisode structuré."""
    facts: list[dict]                  # [{"fact": "...", "type": "merchant"|"habit"|"preference"}]
    source: str = Field(default="transaction_import", max_length=80)
    source_ref: str = Field(default="manual", pattern=r"^(manual|statement:[1-9][0-9]{0,18})$")

class QueryInput(OwnedInput):
    """Requête de recherche dans le graphe de connaissances."""
    query: str = Field(min_length=1, max_length=500)
    top_k: int = Field(default=5, ge=1, le=5)

class KGStatus(BaseModel):
    ok: bool
    node_count: int
    error: Optional[str] = None

# ────────────────────────── Graphiti init ─────────────────────

graphiti: Optional[Graphiti] = None
_start_time: Optional[datetime] = None
_init_lock = asyncio.Lock()


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
    reranker = OpenAIRerankerClient(config=llm_config)
    return llm_client, embedder, reranker


def init_graphiti() -> Graphiti:
    from graphiti_core.driver.falkordb_driver import FalkorDriver
    driver = FalkorDriver(host=FALKORDB_HOST, port=FALKORDB_PORT)
    llm, embedder, reranker = _build_clients()
    g = Graphiti(graph_driver=driver, llm_client=llm, embedder=embedder, cross_encoder=reranker)
    return g


async def _ensure_init():
    global graphiti, _start_time
    if graphiti is not None:
        return
    async with _init_lock:
        if graphiti is not None:
            return
        log.info("initialisation KG (%s)", OLLAMA_MODEL)
        candidate = init_graphiti()
        try:
            await asyncio.wait_for(candidate.build_indices_and_constraints(), timeout=15)
        except Exception as exc:
            log.error("échec init KG: %s", type(exc).__name__)
            await candidate.close()
            raise
        graphiti = candidate
        _start_time = datetime.now(timezone.utc)
        log.info("KG prêt")

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
        raise HTTPException(status_code=503, detail="Knowledge Graph indisponible")


@app.post("/kg/learn")
async def learn_facts(inp: LearnFactsInput) -> dict:
    """
    Ingère des faits financiers sous forme d'épisodes.
    Chaque fait devient une relation extraite dans le graphe.
    """
    await _ensure_init()
    if not 1 <= len(inp.facts) <= 1000 or any(not isinstance(f.get("fact"), str) or not 1 <= len(f["fact"]) <= 4096 for f in inp.facts):
        raise HTTPException(status_code=422, detail="Faits invalides ou trop volumineux")
    try:
        # Construire un body textuel à partir des faits
        fact_lines = "\n".join(f"- {f['fact']}" for f in inp.facts)
        body = f"Faits financiers appris :\n{fact_lines}"

        result = await graphiti.add_episode(
            name=f"Financial facts from {inp.source}",
            episode_body=body,
            source_description=inp.source_ref if inp.source_ref != "manual" else inp.source,
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
        log.error("learn error: %s", type(e).__name__)
        raise HTTPException(status_code=500, detail=failure_detail(e))


@app.post("/kg/query")
async def query_kg(inp: QueryInput) -> dict:
    """Recherche hybride (vectoriel + texte) dans le graphe de connaissances."""
    await _ensure_init()
    try:
        if inp.group_id not in await graphiti.driver.client.list_graphs():
            return {"results": [], "count": 0}
        edges = await graphiti.search(
            query=inp.query,
            group_ids=[inp.group_id],
            num_results=inp.top_k,
        )
        results = [serialize_edge(edge, inp.group_id) for edge in edges]
        sources = await relation_sources(graphiti.driver, inp.group_id, [item for result in results for item in result["episode_ids"]])
        for result in results:
            result["sources"] = [source for source in sources if source["id"] in result["episode_ids"]]
        return {"results": results, "count": len(results)}
    except Exception as e:
        log.error("query error: %s", type(e).__name__)
        raise HTTPException(status_code=500, detail=failure_detail(e))


@app.get("/kg/inspect/{group_id}")
async def kg_inspect(group_id: str):
    try:
        owner_group(group_id)
    except ValueError:
        raise HTTPException(status_code=422, detail="Utilisateur invalide")
    await _ensure_init()
    try:
        return await asyncio.wait_for(inspect_graph(graphiti.driver, group_id), timeout=4)
    except Exception as exc:
        log.error("inspection error: %s", type(exc).__name__)
        raise HTTPException(status_code=503, detail="Lecture du graphe indisponible")


@app.get("/kg/status/{group_id}")
async def kg_status(group_id: str):
    inspected = await kg_inspect(group_id)
    return {"ok": True, "group_id": group_id, "node_count": inspected["node_count"], "edge_count": inspected["edge_count"]}


@app.post("/kg/clear/{group_id}")
async def clear_kg(group_id: str):
    try:
        owner_group(group_id)
    except ValueError:
        raise HTTPException(status_code=422, detail="Utilisateur invalide")
    await _ensure_init()
    try:
        driver = graphiti.driver.clone(database=group_id)
        await driver.execute_query("MATCH (n {group_id:$gid}) DETACH DELETE n", gid=group_id)
        return {"cleared": True}
    except Exception as exc:
        log.error("clear error: %s", type(exc).__name__)
        raise HTTPException(status_code=500, detail="Suppression indisponible")


if __name__ == "__main__":
    import uvicorn
    log.info("démarrage KG service sur port %d", PORT)
    uvicorn.run(app, host="0.0.0.0", port=PORT, log_level="info")
