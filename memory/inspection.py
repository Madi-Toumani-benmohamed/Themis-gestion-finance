"""Bounded, read-only FalkorDB inspection, scoped exactly like Graphiti's group.

Kept independent of FastAPI/Graphiti so contracts can be tested without a model.
"""
import re
from datetime import datetime


def owner_group(value):
    if not isinstance(value, str) or not re.fullmatch(r"[1-9][0-9]{0,18}", value) or int(value) > 9223372036854775807:
        raise ValueError("invalid owner group")
    return value


def timestamp(value):
    return value.isoformat() if isinstance(value, datetime) else str(value)[:80] if value is not None else None


def bounded_text(value, limit=4096):
    if not isinstance(value, str):
        raise ValueError("invalid graph text")
    return value[:limit]


def serialize_edge(edge, group_id):
    if edge.group_id != owner_group(group_id):
        raise ValueError("foreign graph edge")
    # Graphiti EntityEdge exposes name, not relationship, and search() does not
    # provide a relevance score. Never invent a score from rank or a default 0.
    return {
        "id": bounded_text(edge.uuid, 128), "group_id": group_id,
        "fact": bounded_text(edge.fact), "truncated": len(edge.fact) > 4096,
        "source": bounded_text(edge.source_node_uuid, 128),
        "target": bounded_text(edge.target_node_uuid, 128),
        "type": bounded_text(edge.name, 128), "score": None,
        "created_at": timestamp(edge.created_at),
        "valid_at": timestamp(edge.valid_at), "invalid_at": timestamp(edge.invalid_at),
        "expired_at": timestamp(edge.expired_at),
        "episode_ids": [bounded_text(s, 128) for s in edge.episodes[:5]],
    }


async def inspect_graph(driver, group_id):
    group_id = owner_group(group_id)
    # Graphiti routes FalkorDB groups to a graph named after group_id. The old
    # hardcoded graph "graphiti" could therefore report an unrelated empty DB.
    # Do not call driver.clone here: FalkorDriver's constructor schedules
    # index creation, even for a read-only diagnostic of an absent group.
    client = driver.client
    out = {"group_id": group_id, "node_count": 0, "edge_count": 0,
           "episode_count": 0, "nodes": [], "edges": [], "sources": [], "truncated": False}
    if group_id not in await client.list_graphs():
        return out  # Confirmed absence, without implicitly creating a graph.
    graph = client.select_graph(group_id)

    async def read(query, **parameters):
        result = await graph.ro_query(query, params={"gid": group_id, **parameters}, timeout=2500)
        header = [h[1].decode() if isinstance(h[1], bytes) else h[1] for h in result.header]
        return [dict(zip(header, row, strict=True)) for row in result.result_set]

    for label, field in [("Entity", "node_count"), ("Episodic", "episode_count")]:
        rows = await read(f"MATCH (n:{label} {{group_id:$gid}}) RETURN count(n) AS count")
        out[field] = int(rows[0]["count"])
    rows = await read("MATCH (a:Entity {group_id:$gid})-[r:RELATES_TO {group_id:$gid}]->(b:Entity {group_id:$gid}) RETURN count(r) AS count")
    out["edge_count"] = int(rows[0]["count"])
    nodes = {}
    for row in await read("MATCH (n:Entity {group_id:$gid}) RETURN n.uuid AS id,n.name AS name,n.created_at AS created_at ORDER BY n.created_at DESC,n.uuid LIMIT 30"):
        nodes[row["id"]] = {"id": bounded_text(row["id"],128), "name": bounded_text(row["name"],256), "group_id": group_id, "created_at": timestamp(row["created_at"])}
    edge_rows = await read("MATCH (a:Entity {group_id:$gid})-[r:RELATES_TO {group_id:$gid}]->(b:Entity {group_id:$gid}) RETURN r.uuid AS id,a.uuid AS source,b.uuid AS target,a.name AS source_name,b.name AS target_name,r.name AS type,r.fact AS fact,r.created_at AS created_at,r.valid_at AS valid_at,r.invalid_at AS invalid_at,r.expired_at AS expired_at,r.episodes AS episode_ids ORDER BY r.created_at DESC,r.uuid LIMIT 40")
    episodes = set()
    for row in edge_rows:
        for side in ("source", "target"):
            nodes.setdefault(row[side], {"id": bounded_text(row[side],128), "name": bounded_text(row[side+"_name"],256), "group_id": group_id, "created_at": None})
        edge = {key: row.get(key) for key in ("id", "source", "target", "type")}
        for key in edge:
            edge[key] = bounded_text(edge[key],128)
        edge.update({"group_id": group_id, "fact": bounded_text(row["fact"]), "truncated": len(row["fact"]) > 4096, "score": None})
        for key in ("created_at", "valid_at", "invalid_at", "expired_at"):
            edge[key] = timestamp(row.get(key))
        edge["episode_ids"] = [bounded_text(s,128) for s in (row.get("episode_ids") or [])[:5]]
        episodes.update(edge["episode_ids"])
        out["edges"].append(edge)
    if episodes:
        for row in await read("MATCH (e:Episodic {group_id:$gid}) WHERE e.uuid IN $ids RETURN e.uuid AS id,e.name AS name,e.source_description AS source,e.valid_at AS created_at ORDER BY e.uuid LIMIT 200", ids=sorted(episodes)):
            out["sources"].append({"id": bounded_text(row["id"],128), "group_id": group_id, "name": bounded_text(row["name"],256), "source": bounded_text(row["source"],256), "created_at": timestamp(row["created_at"])})
    out["nodes"] = list(nodes.values())
    out["truncated"] = out["node_count"] > len(nodes) or out["edge_count"] > len(out["edges"])
    return out


async def relation_sources(driver, group_id, episode_ids):
    group_id = owner_group(group_id)
    ids = sorted(set(episode_ids))[:25]
    if not ids:
        return []
    graph = driver.client.select_graph(group_id)
    result = await graph.ro_query("MATCH (e:Episodic {group_id:$gid}) WHERE e.uuid IN $ids RETURN e.uuid AS id,e.name AS name,e.source_description AS source,e.valid_at AS created_at ORDER BY e.uuid LIMIT 25", params={"gid": group_id, "ids": ids}, timeout=2500)
    header = [h[1].decode() if isinstance(h[1], bytes) else h[1] for h in result.header]
    rows = [dict(zip(header, row, strict=True)) for row in result.result_set]
    return [{"id": bounded_text(row["id"],128), "group_id": group_id, "name": bounded_text(row["name"],256), "source": bounded_text(row["source"],256), "created_at": timestamp(row["created_at"])} for row in rows]


def failure_detail(exc):
    status = getattr(exc, "status_code", None)
    code = "upstream"
    if "timeout" in type(exc).__name__.lower():
        code = "timeout"
    elif status == 404:
        code = "model_missing"
    elif status in (401, 403):
        code = "model_access"
    elif status == 429:
        code = "model_limited"
    return {"code": code}  # No prompts, financial facts, endpoints or credentials.
