const states = new Set([
  'running',
  'success',
  'empty',
  'error',
  'uncertain',
  'partial',
  'disabled',
  'not_configured',
  'cancelled',
  'skipped',
  'available',
])
export const knowledgeState = (value) =>
  ({
    running: 'En cours',
    success: 'Réussi',
    empty: 'Aucun fait retrouvé',
    error: 'Échec',
    uncertain: 'Résultat non confirmé',
    partial: 'Partiellement traité',
    disabled: 'Désactivée',
    not_configured: 'Non configurée',
    cancelled: 'Annulée',
    skipped: 'Non envoyé',
    available: 'Lecture du graphe réussie',
  })[value] || 'État inconnu'
export const knowledgeError = (code) =>
  ({
    disabled: 'Mémoire désactivée côté serveur.',
    not_configured: 'Service mémoire non configuré.',
    network: 'Connexion au Knowledge Graph impossible.',
    upstream: 'Le service a signalé une erreur. Consulter ses diagnostics.',
    timeout:
      'Délai dépassé ; un apprentissage peut encore se terminer côté service.',
    cancelled: 'Requête annulée.',
    protocol:
      'Réponse mémoire incompatible ou incomplète. Vérifier que les services sont à jour.',
    scope: 'Réponse hors de votre périmètre refusée.',
    model_missing:
      'Modèle ou endpoint introuvable : vérifier les modèles de génération et d’embedding.',
    model_access: 'Accès au modèle refusé.',
    model_limited: 'Quota ou limite du modèle atteint.',
    statement_unvalidated:
      'Relevé non réconcilié : aucun fait envoyé au graphe.',
  })[code] || 'Diagnostic détaillé indisponible.'
export const knowledgeDate = (value) =>
  value && Number.isFinite(Date.parse(value))
    ? new Date(value).toLocaleString('fr-FR')
    : 'Non enregistré'
export const knowledgeNumber = (value) =>
  Number.isSafeInteger(value) && value >= 0
    ? value.toLocaleString('fr-FR')
    : 'Non mesuré'
const integer = (n) => Number.isSafeInteger(n) && n >= 0
const date = (value) =>
  typeof value === 'string' && Number.isFinite(Date.parse(value))
const group = (value) =>
  typeof value === 'string' && /^[1-9]\d{0,18}$/.test(value)
function factsValid(facts, expectedGroup, limit = 5) {
  if (!Array.isArray(facts) || facts.length > limit) return false
  const groups = new Set(facts.map((f) => f?.group_id))
  if (groups.size > 1) return false
  return facts.every(
    (f) =>
      f &&
      group(f.group_id) &&
      (!expectedGroup || f.group_id === expectedGroup) &&
      typeof f.id === 'string' &&
      f.id &&
      typeof f.fact === 'string' &&
      typeof f.source === 'string' &&
      typeof f.target === 'string' &&
      typeof f.type === 'string' &&
      Array.isArray(f.episode_ids) &&
      f.episode_ids.length <= 5 &&
      (f.sources == null ||
        (Array.isArray(f.sources) &&
          f.sources.length <= 5 &&
          f.sources.every(
            (s) =>
              s.group_id === f.group_id &&
              typeof s.name === 'string' &&
              typeof s.source === 'string',
          ))),
  )
}
function eventValid(e) {
  return (
    e &&
    integer(e.id) &&
    e.id > 0 &&
    states.has(e.status) &&
    ['learn', 'chat', 'search'].includes(e.kind) &&
    factsValid(e.facts) &&
    date(e.started_at) &&
    (e.finished_at === null || date(e.finished_at)) &&
    typeof e.context_included === 'boolean' &&
    [
      'fact_count',
      'processed_count',
      'entities_processed',
      'edges_processed',
      'result_count',
      'duration_ms',
    ].every((k) => e[k] === null || integer(e[k]))
  )
}
export function readKnowledge(data) {
  const invalid = () => {
    throw new Error(
      'Réponse du panneau mémoire invalide. Aucun compteur supposé affiché.',
    )
  }
  if (
    data?.scope !== 'current_user' ||
    !states.has(data.service?.status) ||
    !date(data.service.checked_at) ||
    typeof data.service.detail !== 'string' ||
    !Array.isArray(data.activity) ||
    data.activity.length > 50 ||
    typeof data.has_more !== 'boolean' ||
    !data.activity.every(eventValid) ||
    (data.last_learning !== null && !eventValid(data.last_learning))
  )
    invalid()
  if (data.service.status === 'available') {
    const g = data.graph
    if (
      !g ||
      !group(g.group_id) ||
      !integer(g.node_count) ||
      !integer(g.edge_count) ||
      !integer(g.episode_count) ||
      !Array.isArray(g.nodes) ||
      !Array.isArray(g.sources) ||
      g.nodes.length > 110 ||
      g.sources.length > 200 ||
      g.nodes.length > g.node_count ||
      !factsValid(g.edges, g.group_id, 40) ||
      g.edges.length > g.edge_count ||
      typeof g.truncated !== 'boolean' ||
      g.nodes.some(
        (n) =>
          n.group_id !== g.group_id ||
          typeof n.id !== 'string' ||
          !n.id ||
          typeof n.name !== 'string',
      ) ||
      g.sources.some(
        (s) =>
          s.group_id !== g.group_id ||
          typeof s.name !== 'string' ||
          typeof s.source !== 'string',
      )
    )
      invalid()
    const ids = new Set(g.nodes.map((n) => n.id))
    if (
      ids.size !== g.nodes.length ||
      g.edges.some((e) => !ids.has(e.source) || !ids.has(e.target))
    )
      invalid()
  } else if (data.graph !== null) invalid()
  return data
}
export function readKnowledgeTrace(data) {
  if (
    !data ||
    !states.has(data.status) ||
    !factsValid(data.facts) ||
    typeof data.context_included !== 'boolean' ||
    typeof data.audit_available !== 'boolean' ||
    !date(data.checked_at) ||
    !integer(data.duration_ms) ||
    (data.context_included &&
      (data.facts.length === 0 || data.status !== 'success')) ||
    (data.status === 'empty' && data.facts.length > 0) ||
    (!['success', 'empty'].includes(data.status) && data.facts.length > 0)
  )
    throw new Error('Trace mémoire indisponible.')
  return data
}
export function graphPreview(graph, selected, limit = 16) {
  const byID = new Map(graph.nodes.map((n) => [n.id, n])),
    ids = new Set()
  if (byID.has(selected)) {
    ids.add(selected)
    for (const edge of graph.edges) {
      if (ids.size < limit && edge.source === selected && byID.has(edge.target))
        ids.add(edge.target)
      if (ids.size < limit && edge.target === selected && byID.has(edge.source))
        ids.add(edge.source)
      if (ids.size >= limit) break
    }
  }
  for (const node of graph.nodes) {
    if (ids.size >= limit) break
    ids.add(node.id)
  }
  const nodes = [...ids].map((id, i) => ({
    ...byID.get(id),
    x: 110 + (i % 4) * 225,
    y: 55 + Math.floor(i / 4) * 100,
  }))
  const visible = new Set(nodes.map((n) => n.id))
  return {
    nodes,
    edges: graph.edges.filter(
      (e) => visible.has(e.source) && visible.has(e.target),
    ),
    truncated: graph.truncated || graph.nodes.length > nodes.length,
  }
}
