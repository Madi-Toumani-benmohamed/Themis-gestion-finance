import { useEffect, useId, useRef, useState } from 'react'
import { api } from '../api'
import { useAnalysisResource } from './analysisResources'
import { KnowledgeFacts } from './KnowledgeFacts'
import {
  knowledgeDate,
  knowledgeNumber as count,
  knowledgeState,
  knowledgeError,
  readKnowledge,
  readKnowledgeTrace,
  graphPreview,
} from './knowledgeModel.mjs'
import './KnowledgeActivity.css'
function GraphView({ graph }) {
  const [selected, setSelected] = useState(''),
    [fact, setFact] = useState('')
  const unique = useId().replaceAll(':', ''),
    view = graphPreview(graph, selected),
    nodes = new Map(view.nodes.map((n) => [n.id, n]))
  const selectedEdge = graph.edges.find((e) => e.id === fact)
  const related = selected
    ? graph.edges.filter((e) => e.source === selected || e.target === selected)
    : graph.edges
  const choose = (id) => {
    setSelected(id)
    setFact('')
  }
  return (
    <section className="kg-graph" aria-label="Relations enregistrées">
      <h3>Explorer les relations réelles</h3>
      <p>
        Aperçu de {view.nodes.length} entités et {view.edges.length} relations.{' '}
        {view.truncated &&
          'Échantillon limité : les compteurs ci-dessus portent sur le graphe complet du compte.'}
      </p>
      <label>
        Entité à explorer
        <select value={selected} onChange={(e) => choose(e.target.value)}>
          <option value="">Toutes les entités de l’échantillon</option>
          {graph.nodes.map((n) => (
            <option key={n.id} value={n.id}>
              {n.name}
            </option>
          ))}
        </select>
      </label>
      {view.nodes.length > 0 ? (
        <div
          className="kg-network-scroll"
          role="region"
          aria-label="Aperçu du graphe défilant"
          tabIndex={0}
        >
          <svg
            viewBox="0 0 900 425"
            className="kg-network"
            role="group"
            aria-label="Graphe des relations de votre compte"
          >
            <defs>
              <marker
                id={`kg-arrow-${unique}`}
                markerWidth="8"
                markerHeight="8"
                refX="15"
                refY="4"
                orient="auto"
              >
                <path d="M0 0L8 4L0 8" fill="currentColor" />
              </marker>
            </defs>
            {view.edges.map((e) => {
              const a = nodes.get(e.source),
                b = nodes.get(e.target)
              return (
                <line
                  key={e.id}
                  x1={a.x}
                  y1={a.y}
                  x2={b.x}
                  y2={b.y}
                  markerEnd={`url(#kg-arrow-${unique})`}
                  className={
                    selected && e.source !== selected && e.target !== selected
                      ? 'kg-edge dim'
                      : 'kg-edge'
                  }
                />
              )
            })}
            {view.nodes.map((n) => (
              <g
                key={n.id}
                role="button"
                tabIndex={0}
                aria-label={`Explorer ${n.name}`}
                aria-pressed={selected === n.id}
                className={selected === n.id ? 'kg-node selected' : 'kg-node'}
                onClick={() => choose(n.id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    choose(n.id)
                  }
                }}
              >
                <title>{n.name}</title>
                <circle cx={n.x} cy={n.y} r="21" />
                <text x={n.x} y={n.y + 44} textAnchor="middle">
                  {n.name.length > 24 ? n.name.slice(0, 23) + '…' : n.name}
                </text>
              </g>
            ))}
          </svg>
        </div>
      ) : (
        <p>Aucune entité enregistrée pour ce compte.</p>
      )}
      {related.length > 0 ? (
        <>
          <label>
            Relation à consulter
            <select value={fact} onChange={(e) => setFact(e.target.value)}>
              <option value="">Choisir une relation</option>
              {related.map((e) => (
                <option key={e.id} value={e.id}>
                  {e.type} · {e.fact.slice(0, 90)}
                </option>
              ))}
            </select>
          </label>
          {selectedEdge && (
            <KnowledgeFacts facts={[selectedEdge]} sources={graph.sources} />
          )}
        </>
      ) : (
        <p>Aucune relation dans cet échantillon pour l’entité sélectionnée.</p>
      )}
    </section>
  )
}
function Event({ event: e }) {
  return (
    <li>
      <details>
        <summary>
          <strong>
            {e.kind === 'learn'
              ? 'Apprentissage'
              : e.kind === 'chat'
                ? 'Contexte de l’assistant'
                : 'Recherche de vérification'}
          </strong>{' '}
          · {knowledgeState(e.status)} · {knowledgeDate(e.started_at)}
        </summary>
        {e.statement_id && (
          <p>
            Relevé #{e.statement_id} · {e.statement_name || 'Nom indisponible'}
          </p>
        )}
        {e.conversation_id && <p>Conversation #{e.conversation_id}</p>}
        {e.possibly_interrupted && (
          <p className="kg-warning">
            Aucun résultat enregistré depuis plus de trois minutes. Une
            interruption est possible ; le traitement distant peut encore se
            terminer.
          </p>
        )}
        {e.status === 'uncertain' && (
          <p className="kg-warning">
            L’écriture distante peut encore se terminer ; actualiser les
            compteurs du graphe pour constater son état.
          </p>
        )}
        {e.error_code && (
          <p className="kg-warning">{knowledgeError(e.error_code)}</p>
        )}
        {e.kind === 'learn' ? (
          <p>
            {count(e.processed_count)} / {count(e.fact_count)} faits traités ·{' '}
            {count(e.entities_processed)} entités et {count(e.edges_processed)}{' '}
            relations renvoyées par Graphiti. Ces nombres ne sont pas des
            créations nettes.
          </p>
        ) : (
          <p>
            {count(e.result_count)} faits retrouvés.{' '}
            {e.context_included
              ? 'Joints au contexte de la requête IA.'
              : 'Non joints à une requête IA.'}
          </p>
        )}
        <p>
          Terminé : {knowledgeDate(e.finished_at)} · Durée :{' '}
          {e.duration_ms === null ? 'Non mesurée' : `${e.duration_ms} ms`}
        </p>
        <KnowledgeFacts facts={e.facts} />
      </details>
    </li>
  )
}
export default function KnowledgeActivity() {
  const [revision, setRevision] = useState(0),
    [cursors, setCursors] = useState([0]),
    [filter, setFilter] = useState('all')
  const resource = useAnalysisResource(
    (signal) =>
      api.knowledgeOverview(cursors.at(-1), { signal }).then(readKnowledge),
    [cursors.at(-1), revision],
  )
  const [query, setQuery] = useState(''),
    [result, setResult] = useState(null),
    [searching, setSearching] = useState(false),
    [error, setError] = useState('')
  const pending = useRef(null),
    generation = useRef(0)
  useEffect(
    () => () => {
      generation.current++
      pending.current?.abort()
    },
    [],
  )
  const search = async (e) => {
    e.preventDefault()
    pending.current?.abort()
    const controller = new AbortController()
    pending.current = controller
    const id = ++generation.current
    setSearching(true)
    setError('')
    setResult(null)
    try {
      const data = readKnowledgeTrace(
        await api.knowledgeSearch(query, { signal: controller.signal }),
      )
      if (id === generation.current) {
        setResult(data)
        setRevision((v) => v + 1)
      }
    } catch (e) {
      if (id === generation.current && e.name !== 'AbortError')
        setError(e.message)
    } finally {
      if (id === generation.current) setSearching(false)
    }
  }
  const data = resource.data,
    graph = data?.graph,
    last = data?.last_learning
  return (
    <section
      className="knowledge-activity"
      aria-label="Activité du Knowledge Graph"
    >
      <div className="kg-heading">
        <div>
          <h2>Activité du Knowledge Graph</h2>
          <p>
            Vos connaissances réellement enregistrées et les opérations de votre
            compte, y compris pour un administrateur.
          </p>
        </div>
        <button
          className="btn-secondary"
          disabled={resource.loading}
          onClick={() => setRevision((v) => v + 1)}
        >
          Actualiser l’activité
        </button>
      </div>
      {resource.loading && (
        <p role="status">Lecture du graphe et du journal…</p>
      )}
      {resource.error && (
        <p role="alert">
          {resource.error}{' '}
          <button className="btn-secondary" onClick={resource.retry}>
            Réessayer
          </button>
        </p>
      )}
      {data && (
        <>
          <div className="kg-health">
            <strong>{knowledgeState(data.service.status)}</strong>
            <p>{data.service.detail}</p>
            <small>
              Contrôle : {knowledgeDate(data.service.checked_at)}. Une lecture
              réussie ne garantit pas que les modèles de recherche ou
              d’apprentissage fonctionnent.
            </small>
          </div>
          <dl className="kg-metrics">
            <div>
              <dt>Entités enregistrées</dt>
              <dd>{count(graph?.node_count)}</dd>
            </div>
            <div>
              <dt>Relations enregistrées</dt>
              <dd>{count(graph?.edge_count)}</dd>
            </div>
            <div>
              <dt>Épisodes enregistrés</dt>
              <dd>{count(graph?.episode_count)}</dd>
            </div>
            <div>
              <dt>Dernier apprentissage journalisé</dt>
              <dd>
                {last ? knowledgeState(last.status) : 'Aucun enregistré'}
                <small>
                  {last
                    ? knowledgeDate(last.started_at)
                    : 'L’historique antérieur à cette version n’est pas reconstitué.'}
                </small>
              </dd>
            </div>
          </dl>
          {last?.error_code && (
            <p className="kg-warning">{knowledgeError(last.error_code)}</p>
          )}
        </>
      )}
      <section className="kg-search" aria-label="Recherche dans la mémoire">
        <h3>Vérifier ce qui est retrouvé</h3>
        <p>
          Rechercher un fait déjà appris. Ce test ne crée aucun fait ; seule la
          trace de l’opération est journalisée.
        </p>
        <form onSubmit={search}>
          <label>
            Question de vérification
            <input
              required
              maxLength={500}
              value={query}
              onChange={(e) => {
                generation.current++
                pending.current?.abort()
                setSearching(false)
                setResult(null)
                setError('')
                setQuery(e.target.value)
              }}
              placeholder="Quel marchand revient dans mes dépenses ?"
            />
          </label>
          <button className="btn" disabled={searching || !query.trim()}>
            {searching ? 'Recherche en cours…' : 'Tester la recherche'}
          </button>
        </form>
        {error && <p role="alert">{error}</p>}
        {result && (
          <div className="kg-search-result" aria-live="polite">
            <strong>{knowledgeState(result.status)}</strong>
            {result.error_code ? (
              <p>{knowledgeError(result.error_code)}</p>
            ) : (
              <p>
                {result.facts.length} fait(s) retrouvé(s) en{' '}
                {result.duration_ms} ms.
              </p>
            )}
            <KnowledgeFacts facts={result.facts} />
            {!result.audit_available && (
              <p>Le journal persistant n’a pas pu être enregistré.</p>
            )}
          </div>
        )}
      </section>
      {graph && <GraphView graph={graph} />}
      {data && (
        <section className="kg-history" aria-label="Journal mémoire">
          <h3>Journal des opérations</h3>
          <p>
            Historique collecté depuis cette version. Aucun apprentissage ancien
            n’est inventé ; un relevé doit être confirmé et réconcilié pour
            alimenter automatiquement la mémoire.
          </p>
          <label>
            Opérations affichées
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="all">Toutes</option>
              <option value="learn">Apprentissages</option>
              <option value="search">Recherches de vérification</option>
              <option value="chat">Contexte de l’assistant</option>
            </select>
          </label>
          {data.activity.filter((e) => filter === 'all' || e.kind === filter)
            .length === 0 ? (
            <p>Aucune opération journalisée dans cette page pour ce filtre.</p>
          ) : (
            <ol>
              {data.activity
                .filter((e) => filter === 'all' || e.kind === filter)
                .map((event) => (
                  <Event key={event.id} event={event} />
                ))}
            </ol>
          )}
          <div className="kg-actions">
            {cursors.length > 1 && (
              <button
                className="btn-secondary"
                onClick={() => setCursors((c) => c.slice(0, -1))}
              >
                Opérations plus récentes
              </button>
            )}
            {data.has_more && (
              <button
                className="btn-secondary"
                onClick={() =>
                  setCursors((c) => [...c, data.activity.at(-1).id])
                }
              >
                Opérations plus anciennes
              </button>
            )}
          </div>
        </section>
      )}
      <p className="kg-note">
        La mémoire peut être partielle ou périmée. Les relations sont un
        contexte indicatif, jamais une preuve des montants ou de la complétude
        des relevés.
      </p>
    </section>
  )
}
