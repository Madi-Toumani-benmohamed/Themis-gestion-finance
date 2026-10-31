import { test } from 'node:test'
import assert from 'node:assert/strict'
import fixture from '../../tests/fixtures/knowledge.cjs'
import {
  readKnowledge,
  readKnowledgeTrace,
  knowledgeNumber,
  graphPreview,
} from './knowledgeModel.mjs'
test('Knowledge counts are observed, zero remains zero and unknown remains unknown', () => {
  const data = fixture.overview()
  assert.equal(readKnowledge(data).graph.node_count, 2)
  const empty = structuredClone(data)
  empty.graph = {
    ...empty.graph,
    node_count: 0,
    edge_count: 0,
    episode_count: 0,
    nodes: [],
    edges: [],
    sources: [],
  }
  assert.equal(readKnowledge(empty).graph.node_count, 0)
  const offline = {
    ...data,
    service: { ...data.service, status: 'error' },
    graph: null,
  }
  assert.equal(readKnowledge(offline).activity.length, 3)
  assert.equal(knowledgeNumber(null), 'Non mesuré')
  assert.equal(knowledgeNumber(undefined), 'Non mesuré')
  assert.equal(knowledgeNumber(0), '0')
  for (const patch of [
    { scope: 'other_users' },
    { graph: { ...data.graph, node_count: null } },
    { graph: { ...data.graph, node_count: 1 } },
    { graph: { ...data.graph, edges: [{ ...fixture.fact, group_id: '2' }] } },
    {
      graph: {
        ...data.graph,
        nodes: [{ ...data.graph.nodes[0], group_id: '2' }],
      },
    },
    { last_learning: {} },
    { activity: [{ ...data.activity[0], result_count: undefined }] },
  ])
    assert.throws(() => readKnowledge({ ...data, ...patch }))
})
test('Trace means facts included in a request, never proof of actual model usage', () => {
  assert.equal(readKnowledgeTrace(fixture.trace()).context_included, false)
  assert.equal(
    readKnowledgeTrace(fixture.trace({ context_included: true })).facts.length,
    1,
  )
  assert.equal(
    readKnowledgeTrace(fixture.trace({ status: 'empty', facts: [] })).facts
      .length,
    0,
  )
  for (const change of [
    { status: 'empty' },
    { status: 'error' },
    { context_included: true, facts: [] },
    { audit_available: null },
    {
      facts: [
        { ...fixture.fact, sources: [{ ...fixture.source, group_id: '2' }] },
      ],
    },
  ])
    assert.throws(() => readKnowledgeTrace(fixture.trace(change)))
})
test('Graph preview stays bounded and preserves only real node relations', () => {
  const graph = fixture.overview().graph
  const copy = structuredClone(graph)
  for (let i = 3; i <= 30; i++) {
    graph.nodes.push({
      id: `node-${i}`,
      group_id: '1',
      name: `ENTITY TEST ${i}`,
    })
    graph.edges.push({
      ...fixture.fact,
      id: `edge-${i}`,
      source: 'node-30',
      target: `node-${i}`,
    })
  }
  const preview = graphPreview(graph, 'node-30')
  assert.ok(preview.nodes.length <= 16)
  assert.ok(preview.nodes.some((n) => n.id === 'node-30'))
  assert.equal(preview.truncated, true)
  const ids = new Set(preview.nodes.map((n) => n.id))
  for (const edge of preview.edges)
    assert.ok(ids.has(edge.source) && ids.has(edge.target))
  assert.equal(copy.nodes.length, 2)
  assert.equal(
    graph.nodes.some((n) => 'x' in n),
    false,
  )
})
