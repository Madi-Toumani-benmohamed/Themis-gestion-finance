"""Real FastAPI/Graphiti models, fake model/driver transport. No live user graph."""
import asyncio
import importlib.util
import sys
import unittest
from pathlib import Path
from unittest.mock import patch
from types import SimpleNamespace
from test_knowledge_inspection import FakeDriver, edge

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'memory'))
spec = importlib.util.spec_from_file_location('knowledge_service_test', ROOT / 'memory/main.py')
service = importlib.util.module_from_spec(spec)
spec.loader.exec_module(service)
from fastapi.testclient import TestClient


class FakeGraphiti:
    def __init__(self):
        self.driver, self.received = FakeDriver(), []

    async def search(self, **kwargs):
        self.received.append(kwargs)
        return [edge(kwargs['group_ids'][0])]

    async def add_episode(self, **kwargs):
        self.received.append(kwargs)
        return SimpleNamespace(nodes=[1, 2], edges=[1])


class KnowledgeRuntimeTest(unittest.TestCase):
    def setUp(self):
        self.graph = FakeGraphiti()
        self.patch = patch.object(service, 'graphiti', self.graph)
        self.patch.start()
        self.client = TestClient(service.app)

    def tearDown(self):
        self.client.close()
        self.patch.stop()

    def test_routes_inspection_search_and_sources(self):
        response = self.client.get('/kg/inspect/1')
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()['node_count'], 2)
        response = self.client.post('/kg/query', json={'group_id': '1', 'query': 'Marchand TEST', 'top_k': 5})
        self.assertEqual(response.status_code, 200)
        data = response.json()
        self.assertEqual(data['results'][0]['type'], 'RECURRENT')
        self.assertIsNone(data['results'][0]['score'])
        self.assertEqual(data['results'][0]['sources'][0]['source'], 'statement:101')
        self.assertEqual(self.graph.received[-1]['group_ids'], ['1'])

    def test_reranker_uses_supported_openai_transport(self):
        llm, embedder, reranker = service._build_clients()
        self.assertTrue(hasattr(reranker.client, 'chat'))
        self.assertIsNot(reranker.client, llm)

    def test_real_entity_edge_contract(self):
        from graphiti_core.edges import EntityEdge
        from datetime import datetime, timezone
        real = EntityEdge(group_id='1', source_node_uuid='a', target_node_uuid='b', name='RECURRENT', fact='CONTRACT TEST', created_at=datetime.now(timezone.utc))
        data = service.serialize_edge(real, '1')
        self.assertEqual(data['type'], 'RECURRENT')
        self.assertIsNone(data['score'])

    def test_owner_and_query_limits(self):
        for group in ['0', "1' OR 1=1", '9223372036854775808']:
            self.assertEqual(self.client.post('/kg/query', json={'group_id': group, 'query': 'test'}).status_code, 422)
        self.assertEqual(self.client.post('/kg/query', json={'group_id': '1', 'query': 'x' * 501}).status_code, 422)
        self.assertEqual(self.client.post('/kg/query', json={'group_id': '1', 'query': 'test', 'top_k': 100}).status_code, 422)

    def test_learning_is_real_episode_output_and_explicit_source(self):
        response = self.client.post('/kg/learn', json={'group_id': '1', 'facts': [{'fact': 'FAIT TEST'}], 'source_ref': 'statement:101'})
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json(), {'entities_added': 2, 'edges_added': 1})
        self.assertEqual(self.graph.received[-1]['source_description'], 'statement:101')

    def test_search_failure_never_becomes_empty_success(self):
        async def failed(**kwargs):
            error = RuntimeError('SECRET PROVIDER DETAIL')
            error.status_code = 404
            raise error
        self.graph.search = failed
        response = self.client.post('/kg/query', json={'group_id': '1', 'query': 'test'})
        self.assertEqual(response.status_code, 500)
        self.assertEqual(response.json()['detail']['code'], 'model_missing')
        self.assertNotIn('SECRET', response.text)

    def test_foreign_relation_is_never_serialized(self):
        async def foreign(**kwargs):
            return [edge('2')]
        self.graph.search = foreign
        response = self.client.post('/kg/query', json={'group_id': '1', 'query': 'test'})
        self.assertEqual(response.status_code, 500)
        self.assertNotIn('FAIT TEST 2', response.text)


class InitializationTest(unittest.IsolatedAsyncioTestCase):
    async def test_initialization_only_published_after_indices_ready(self):
        entered, release = asyncio.Event(), asyncio.Event()
        async def build():
            entered.set()
            await release.wait()
        candidate = SimpleNamespace(build_indices_and_constraints=build)
        with patch.object(service, 'graphiti', None), patch.object(service, '_init_lock', asyncio.Lock()), patch.object(service, 'init_graphiti', return_value=candidate):
            first = asyncio.create_task(service._ensure_init())
            await entered.wait()
            second = asyncio.create_task(service._ensure_init())
            self.assertIsNone(service.graphiti)
            release.set()
            await asyncio.gather(first, second)
            self.assertIs(service.graphiti, candidate)
