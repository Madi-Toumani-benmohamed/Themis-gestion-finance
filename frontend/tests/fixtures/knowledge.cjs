const stamp = '2026-10-31T10:00:00Z'
const source = {
  id: 'episode-1',
  group_id: '1',
  name: 'IMPORT SYNTHETIQUE TEST',
  source: 'statement:101',
  created_at: stamp,
}
const fact = {
  id: 'edge-1',
  group_id: '1',
  fact: 'MARCHAND TEST revient dans les dépenses du compte TEST.',
  source: 'node-1',
  target: 'node-2',
  type: 'RECURRENT',
  score: null,
  created_at: stamp,
  valid_at: stamp,
  invalid_at: null,
  expired_at: null,
  episode_ids: ['episode-1'],
  sources: [source],
  truncated: false,
}
function activity(overrides = {}) {
  return {
    id: 3,
    kind: 'learn',
    source: 'statement',
    status: 'success',
    statement_id: 101,
    conversation_id: null,
    statement_name: 'RELEVE SYNTHETIQUE TEST',
    fact_count: 5,
    processed_count: 5,
    entities_processed: 2,
    edges_processed: 1,
    result_count: null,
    context_included: false,
    facts: [],
    error_code: '',
    duration_ms: 42,
    started_at: stamp,
    finished_at: stamp,
    possibly_interrupted: false,
    ...overrides,
  }
}
function overview() {
  return {
    scope: 'current_user',
    service: {
      status: 'available',
      detail: 'Lecture effective de FalkorDB réussie TEST.',
      error_code: '',
      checked_at: stamp,
    },
    graph: {
      group_id: '1',
      node_count: 2,
      edge_count: 1,
      episode_count: 1,
      nodes: [
        {
          id: 'node-1',
          group_id: '1',
          name: 'MARCHAND TEST',
          created_at: stamp,
        },
        {
          id: 'node-2',
          group_id: '1',
          name: 'HABITUDE TEST',
          created_at: stamp,
        },
      ],
      edges: [structuredClone(fact)],
      sources: [source],
      truncated: false,
    },
    activity: [
      activity(),
      activity({
        id: 2,
        kind: 'search',
        source: 'diagnostic',
        status: 'error',
        statement_id: null,
        statement_name: null,
        fact_count: null,
        processed_count: null,
        entities_processed: null,
        edges_processed: null,
        error_code: 'model_missing',
      }),
      activity({
        id: 1,
        kind: 'chat',
        source: 'assistant',
        statement_id: null,
        statement_name: null,
        conversation_id: 1,
        fact_count: null,
        processed_count: null,
        entities_processed: null,
        edges_processed: null,
        result_count: 1,
        context_included: true,
        facts: [structuredClone(fact)],
      }),
    ],
    last_learning: activity(),
    has_more: false,
  }
}
function trace(overrides = {}) {
  return {
    status: 'success',
    activity_id: 10,
    audit_available: true,
    facts: [structuredClone(fact)],
    context_included: false,
    error_code: '',
    checked_at: stamp,
    duration_ms: 12,
    ...overrides,
  }
}
module.exports = { stamp, source, fact, activity, overview, trace }
