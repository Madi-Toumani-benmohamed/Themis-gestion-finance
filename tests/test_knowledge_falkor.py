"""Opt-in real FalkorDB inspection; use only a disposable synthetic test server."""
import os
import unittest
from memory.inspection import inspect_graph


@unittest.skipUnless(os.getenv('THEMIS_KG_FALKOR_TEST_PORT'), 'Dedicated synthetic FalkorDB not configured')
class FalkorInspectionIntegration(unittest.IsolatedAsyncioTestCase):
    async def test_real_read_only_inspection_and_two_groups(self):
        from falkordb.asyncio import FalkorDB
        from types import SimpleNamespace
        client = FalkorDB(host='127.0.0.1', port=int(os.environ['THEMIS_KG_FALKOR_TEST_PORT']))
        driver = SimpleNamespace(client=client)
        stamp = '2026-10-31T10:00:00+00:00'
        try:
            for group in ('90001', '90002'):
                graph = client.select_graph(group)
                await graph.query('CREATE (a:Entity {uuid:$a,group_id:$gid,name:$aname,created_at:$stamp}),(b:Entity {uuid:$b,group_id:$gid,name:$bname,created_at:$stamp}),(e:Episodic {uuid:$ep,group_id:$gid,name:$ename,source_description:$src,valid_at:$stamp}),(a)-[:RELATES_TO {uuid:$edge,group_id:$gid,name:$relation,fact:$fact,created_at:$stamp,episodes:[$ep]}]->(b)', params={'gid': group,'a':f'a-{group}','b':f'b-{group}','aname':f'OWNER {group} MERCHANT TEST','bname':'HABIT TEST','stamp':stamp,'ep':f'ep-{group}','ename':'IMPORT SYNTHETIC TEST','src':'statement:101','edge':f'edge-{group}','relation':'RECURRENT','fact':f'FACT OWNER {group} TEST'})
            before = sorted(await client.list_graphs())
            one = await inspect_graph(driver,'90001')
            self.assertEqual(one['node_count'],2)
            self.assertEqual(one['edge_count'],1)
            self.assertEqual(one['episode_count'],1)
            self.assertEqual(one['edges'][0]['fact'],'FACT OWNER 90001 TEST')
            self.assertEqual(one['sources'][0]['source'],'statement:101')
            self.assertNotIn('90002',str(one))
            two = await inspect_graph(driver,'90002')
            self.assertNotIn('90001',str(two))
            empty = await inspect_graph(driver,'90003')
            self.assertEqual(empty['node_count'],0)
            self.assertEqual(sorted(await client.list_graphs()),before,'A read must not create a namespace')
        finally:
            for group in ('90001','90002'):
                if group in await client.list_graphs():
                    await client.select_graph(group).delete()
            await client.connection.aclose()
