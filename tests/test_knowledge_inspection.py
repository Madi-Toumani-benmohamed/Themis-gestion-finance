"""Read-only, tenant-bound graph contracts; synthetic facts only."""
import unittest
from types import SimpleNamespace
from datetime import datetime, timezone
from memory.inspection import inspect_graph, owner_group, serialize_edge, failure_detail


class FakeGraph:
    def __init__(self, group, calls):
        self.group, self.calls = group, calls

    async def ro_query(self, query, params=None, timeout=None):
        self.calls.append((self.group, query, params, timeout))
        assert params['gid'] == self.group
        assert 'group_id:$gid' in query
        assert query.startswith('MATCH ')
        group = self.group
        if 'count(n)' in query:
            rows = [{'count': 1 if ':Episodic' in query else 2}]
        elif 'count(r)' in query:
            rows = [{'count': 1}]
        elif 'source_name' in query:
            rows = [{'id': f'edge-{group}', 'source': f'a-{group}', 'target': f'b-{group}', 'source_name': f'Marchand TEST {group}', 'target_name': f'Habitude TEST {group}', 'type': 'RECURRENT', 'fact': f'FAIT SYNTHETIQUE OWNER {group}', 'created_at': '2026-10-31T10:00:00Z', 'valid_at': None, 'invalid_at': None, 'expired_at': None, 'episode_ids': [f'ep-{group}']}]
        elif ':Episodic' in query:
            rows = [{'id': f'ep-{group}', 'name': 'Import TEST', 'source': 'statement:101', 'created_at': '2026-10-31T10:00:00Z'}]
        else:
            rows = [{'id': f'a-{group}', 'name': f'Marchand TEST {group}', 'created_at': None}, {'id': f'b-{group}', 'name': f'Habitude TEST {group}', 'created_at': None}]
        keys = list(rows[0]) if rows else []
        return SimpleNamespace(header=[[1, k] for k in keys], result_set=[[row[k] for k in keys] for row in rows])

    def query(self, *args, **kwargs):
        raise AssertionError('A diagnostic must never issue GRAPH.QUERY')


class FakeClient:
    def __init__(self):
        self.calls, self.selected = [], []

    async def list_graphs(self):
        return ['1', '2']

    def select_graph(self, group):
        self.selected.append(group)
        return FakeGraph(group, self.calls)


class FakeDriver:
    def __init__(self):
        self.client, self.clones = FakeClient(), []

    def clone(self, database):
        raise AssertionError('Inspection must not clone a driver and implicitly create indexes')


def edge(group='1'):
    return SimpleNamespace(uuid=f'edge-{group}', group_id=group, fact=f'FAIT TEST {group}', source_node_uuid=f'a-{group}', target_node_uuid=f'b-{group}', name='RECURRENT', created_at=datetime(2026, 10, 31, tzinfo=timezone.utc), valid_at=None, invalid_at=None, expired_at=None, episodes=[f'ep-{group}'])


class KnowledgeInspectionTest(unittest.IsolatedAsyncioTestCase):
    async def test_actual_group_and_parameterized_read_only_queries(self):
        driver = FakeDriver()
        data = await inspect_graph(driver, '1')
        self.assertEqual(driver.clones, [])
        self.assertEqual(data['node_count'], 2)
        self.assertEqual(data['edge_count'], 1)
        self.assertEqual(data['episode_count'], 1)
        self.assertEqual(data['edges'][0]['fact'], 'FAIT SYNTHETIQUE OWNER 1')
        self.assertNotIn('OWNER 2', str(data))
        self.assertEqual(data['sources'][0]['group_id'], '1')
        self.assertTrue(all(call[3] == 2500 for call in driver.client.calls))
        second = await inspect_graph(driver, '2')
        self.assertNotIn('OWNER 1', str(second))

    async def test_empty_namespace_does_not_create_a_graph(self):
        driver = FakeDriver()
        data = await inspect_graph(driver, '3')
        self.assertEqual(data['node_count'], 0)
        self.assertEqual(data['edges'], [])
        self.assertEqual(driver.client.selected, [])

    async def test_invalid_owner_rejected_before_graph_access(self):
        for value in ["1' OR 1=1", '0', '-1', '01', 'other', '9223372036854775808', 1]:
            with self.subTest(value=value):
                driver = FakeDriver()
                with self.assertRaises(ValueError):
                    await inspect_graph(driver, value)
                self.assertEqual(driver.clones, [])

    def test_graphiti_entity_edge_fields_and_unknown_score(self):
        value = serialize_edge(edge(), '1')
        self.assertEqual(value['type'], 'RECURRENT')
        self.assertIsNone(value['score'])
        self.assertEqual(value['episode_ids'], ['ep-1'])
        with self.assertRaises(ValueError):
            serialize_edge(edge('2'), '1')

    def test_sanitized_provider_errors(self):
        error = RuntimeError('SECRET KEY AND FINANCIAL FACT MUST NOT APPEAR')
        error.status_code = 404
        self.assertEqual(failure_detail(error), {'code': 'model_missing'})
        self.assertNotIn('SECRET', str(failure_detail(error)))
