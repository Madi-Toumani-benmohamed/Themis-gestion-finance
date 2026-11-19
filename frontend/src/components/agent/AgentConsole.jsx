import { useEffect, useId, useRef, useState } from 'react'
import { KnowledgeTrace } from '../KnowledgeFacts'
import { api } from '../../api'
import {
  agentRequest,
  agentList,
  agentTimezone,
  agentSources,
} from './agentClient.mjs'
import './AgentConsole.css'

const labels = {
  categorize: 'Catégorisation',
  tag: 'Ajout de tag',
  rule: 'Règle pour les prochains imports',
  simulation: 'Simulation',
  memory: 'Préférence à mémoriser',
}
const states = {
  prepared: 'À valider',
  applied: 'Appliquée',
  dismissed: 'Ignorée',
  stale: 'À préparer de nouveau',
}
const date = (value) =>
  value
    ? new Date(value).toLocaleString('fr-FR')
    : 'Pas encore de point enregistré'
const toolLabels = {
  search_transactions: 'Recherche des opérations',
  get_financial_summary: 'Synthèse des montants observés',
  compare_periods: 'Comparaison des périodes',
  get_category_breakdown: 'Analyse par catégorie',
  get_data_quality: 'Vérification des données disponibles',
  get_project_analysis: 'Analyse du projet',
  get_verified_discoveries: 'Lecture des découvertes vérifiées',
  get_recurring_expenses: 'Examen des récurrences',
  get_categories: 'Consultation des catégories',
  prepare_action: 'Préparation d’un aperçu',
}
const pageLabels = {
  dashboard: 'Tableau de bord',
  transactions: 'Transactions',
  analyse: 'Analyse',
  budgets: 'Budgets',
  statements: 'Relevés',
  credits: 'Crédits et taxes',
  accounts: 'Comptes',
  simulate: 'Simulation',
  fiscal: 'Impôts',
  laboratory: 'Laboratoire',
  knowledge: 'Mémoire',
  advice: 'Conseils IA',
  admin: 'Administration',
}
const money = (value) =>
  value == null ? 'Non observé' : `${String(value).replace('.', ',')} €`
function SourceData({ data, onNavigate }) {
  if (!data || typeof data !== 'object') return null
  const amounts = {
    income: 'Revenus observés',
    expenses: 'Dépenses observées',
    net: 'Solde des flux observés',
    income_a: 'Revenus de la première période',
    expenses_a: 'Dépenses de la première période',
    income_b: 'Revenus de la seconde période',
    expenses_b: 'Dépenses de la seconde période',
    expense_change: 'Écart des dépenses',
    monthly_saved: 'Réduction simulée',
    base_spend: 'Dépenses de référence',
    simulated_spend: 'Dépenses simulées',
    simulated_net: 'Solde des flux simulés',
  }
  return (
    <div className="agent-source-data">
      {data.query_from && (
        <p>
          Du {data.query_from} au {data.query_to}.
        </p>
      )}
      {data.filters && (
        <p className="agent-read-filters">
          Périmètre :{' '}
          {data.filters.direction === 'expense'
            ? 'dépenses'
            : data.filters.direction === 'income'
              ? 'revenus'
              : 'revenus et dépenses'}
          {data.filters.merchant && ` · marchand : ${data.filters.merchant}`}
          {data.filters.category && ` · catégorie : ${data.filters.category}`}
          {data.filters.tag && ` · tag : ${data.filters.tag}`}
          {data.filters.min_amount != null &&
            ` · à partir de ${money(data.filters.min_amount)}`}
          {data.filters.max_amount != null &&
            ` · jusqu’à ${money(data.filters.max_amount)}`}
          .
        </p>
      )}
      {data.has_more && (
        <p>
          Liste partielle ({data.returned_count} opérations affichées). Le total
          complet doit être consulté avec la synthèse financière.
        </p>
      )}
      {data.reason && <p>{data.reason}</p>}
      {data.source && <p>Source : {data.source}</p>}
      <dl>
        {Object.entries(amounts)
          .filter(([key]) => Object.hasOwn(data, key))
          .map(([key, label]) => (
            <div key={key}>
              <dt>{label}</dt>
              <dd>{money(data[key])}</dd>
            </div>
          ))}
      </dl>
      {agentList(data.transactions).length > 0 && (
        <ul>
          {data.transactions.map((t) => (
            <li key={t.id}>
              {t.date} · {t.label} · {money(t.amount)}{' '}
              {onNavigate && (
                <button
                  className="link"
                  onClick={() => onNavigate('transactions', { ids: [t.id] })}
                >
                  Voir l’opération
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      {agentList(data.categories).length > 0 && (
        <ul>
          {data.categories.map((c, i) => (
            <li key={i}>
              {c.category} : {money(c.amount ?? c.before)}
              {c.after != null && ` → ${money(c.after)} dans le scénario`}
            </li>
          ))}
        </ul>
      )}
      {agentList(data.groups).length > 0 && (
        <ul>
          {data.groups.map((group, i) => (
            <li key={i}>
              {group.key} : revenus {money(group.income)}, dépenses{' '}
              {money(group.expenses)} ({group.transaction_count} opérations).
            </li>
          ))}
        </ul>
      )}
      {data.truncated && (
        <p>
          Résultat limité aux 200 premiers groupes ; ne pas additionner cette
          liste comme un total complet.
        </p>
      )}
      {agentList(data.items).length > 0 && (
        <ul>
          {data.items.map((item, i) => (
            <li key={item.key || i}>
              {item.label} · {money(item.typical_amount)} ·{' '}
              {item.occurrence_count} occurrence(s).{' '}
              {item.coverage_complete
                ? 'Période documentée.'
                : 'Couverture à vérifier.'}
            </li>
          ))}
        </ul>
      )}
      {data.transaction_count != null && (
        <p>{data.transaction_count} opération(s) observée(s).</p>
      )}
      {data.first_transaction && (
        <p>
          Première opération :{' '}
          {typeof data.first_transaction === 'string'
            ? data.first_transaction
            : data.first_transaction.String}
        </p>
      )}
      {data.last_transaction && (
        <p>
          Dernière opération :{' '}
          {typeof data.last_transaction === 'string'
            ? data.last_transaction
            : data.last_transaction.String}
        </p>
      )}
    </div>
  )
}
function Preview({ action }) {
  const p = action.preview || {}
  return (
    <div className="agent-preview">
      <p>{p.effect}</p>
      {p.category && (
        <p>
          Catégorie : <strong>{p.category}</strong>
        </p>
      )}
      {p.tag && (
        <p>
          Tag : <strong>{p.tag}</strong>
        </p>
      )}
      {p.content && <blockquote>{p.content}</blockquote>}
      {p.pattern && (
        <p>
          Motif : <strong>{p.pattern}</strong>
        </p>
      )}
      {p.count != null && <p>{p.count} opération(s) dans l’aperçu.</p>}
      {agentList(p.transactions || p.samples).length > 0 && (
        <ul>
          {agentList(p.transactions || p.samples).map((t) => (
            <li key={t.id}>
              {t.date && <time>{t.date} · </time>}
              {t.label}
              {t.amount != null && ` · ${t.amount} €`}
            </li>
          ))}
        </ul>
      )}
      {p.scenario && (
        <>
          <p>
            Référence : {p.month}. Les résultats sont des hypothèses sur les
            données observées.
          </p>
          <details>
            <summary>Résultats du scénario</summary>
            <SourceData data={p.scenario} />
          </details>
        </>
      )}
    </div>
  )
}
function ActionBuilder({
  disabled = false,
  conversation,
  mission,
  messages,
  selection,
  month,
  onPrepared,
  onBusyChange,
}) {
  const [kind, setKind] = useState('tag'),
    [tag, setTag] = useState(''),
    [domain, setDomain] = useState(''),
    [pattern, setPattern] = useState(''),
    [content, setContent] = useState(''),
    [percent, setPercent] = useState('10'),
    [reference, setReference] = useState(month || ''),
    [chosen, setChosen] = useState([]),
    [domains, setDomains] = useState([]),
    [busy, setBusy] = useState(false),
    [error, setError] = useState('')
  const id = useId(),
    sources = agentSources(messages)
  useEffect(() => {
    let active = true
    api
      .domains()
      .then((v) => {
        if (active) setDomains(agentList(v))
      })
      .catch(() => {
        if (active)
          setError('Catégories indisponibles. Réessaie en rouvrant ce panneau.')
      })
    return () => {
      active = false
    }
  }, [])
  const prepare = async (event) => {
    event.preventDefault()
    if (busy || disabled) return
    setBusy(true)
    onBusyChange(true)
    setError('')
    try {
      const action = { kind }
      if (kind === 'tag' || kind === 'categorize')
        action.transaction_ids = chosen.length ? chosen : selection
      if (kind === 'tag') action.tag = tag
      if (kind === 'categorize' || kind === 'rule')
        action.domain_id = Number(domain)
      if (kind === 'rule') action.pattern = pattern
      if (kind === 'memory') action.content = content
      if (kind === 'simulation') {
        action.month = reference
        action.simulation = {
          cuts: [
            {
              category:
                domains.find((d) => d.id === Number(domain))?.name || '',
              reduce_pct: percent,
            },
          ],
        }
      }
      const result = await agentRequest('actions', 'POST', {
        conversation_id: conversation,
        mission_id: mission || 0,
        action,
      })
      onPrepared(result)
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
      onBusyChange(false)
    }
  }
  return (
    <form className="agent-builder" onSubmit={prepare}>
      <label htmlFor={id + 'kind'}>Action à préparer</label>
      <select
        id={id + 'kind'}
        value={kind}
        onChange={(e) => {
          setKind(e.target.value)
          setError('')
        }}
      >
        {Object.entries(labels).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>
      {(kind === 'tag' || kind === 'categorize') && (
        <fieldset>
          <legend>Opérations concernées</legend>
          {selection.length > 0 && (
            <p>
              {selection.length} opération(s) sélectionnée(s) dans la page.{' '}
              {chosen.length > 0
                ? 'Les cases cochées ci-dessous remplacent cette sélection.'
                : 'Elles seront utilisées si aucune case n’est cochée.'}
            </p>
          )}
          {sources.map((t) => (
            <label className="agent-choice" key={t.id}>
              <input
                type="checkbox"
                checked={chosen.includes(t.id)}
                onChange={(e) =>
                  setChosen((v) =>
                    e.target.checked
                      ? [...v, t.id]
                      : v.filter((x) => x !== t.id),
                  )
                }
              />
              <span>
                {t.date} · {t.label} · {t.amount} €
              </span>
            </label>
          ))}
          {!selection.length && !sources.length && (
            <p>
              Demande d’abord les opérations à examiner ou utilise « Discuter de
              la sélection » dans Transactions.
            </p>
          )}
        </fieldset>
      )}
      {['categorize', 'rule', 'simulation'].includes(kind) && (
        <>
          <label htmlFor={id + 'domain'}>Catégorie</label>
          <select
            id={id + 'domain'}
            required
            value={domain}
            onChange={(e) => setDomain(e.target.value)}
          >
            <option value="">Choisir une catégorie</option>
            {domains.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </>
      )}
      {kind === 'tag' && (
        <>
          <label htmlFor={id + 'tag'}>Tag à ajouter</label>
          <input
            id={id + 'tag'}
            required
            maxLength={40}
            value={tag}
            onChange={(e) => setTag(e.target.value)}
          />
        </>
      )}
      {kind === 'rule' && (
        <>
          <label htmlFor={id + 'pattern'}>Libellé ou motif à reconnaître</label>
          <input
            id={id + 'pattern'}
            required
            maxLength={200}
            value={pattern}
            onChange={(e) => setPattern(e.target.value)}
          />
          <p>
            La règle agira sur les prochains imports. L’aperçu montre les
            correspondances actuelles.
          </p>
        </>
      )}
      {kind === 'memory' && (
        <>
          <label htmlFor={id + 'memory'}>
            Préférence ou décision à retenir
          </label>
          <textarea
            id={id + 'memory'}
            required
            maxLength={1000}
            value={content}
            onChange={(e) => setContent(e.target.value)}
          />
        </>
      )}
      {kind === 'simulation' && (
        <>
          <label htmlFor={id + 'month'}>Mois de référence</label>
          <input
            id={id + 'month'}
            type="month"
            required
            value={reference}
            onChange={(e) => setReference(e.target.value)}
          />
          <label htmlFor={id + 'percent'}>
            Réduction hypothétique de cette catégorie (%)
          </label>
          <input
            id={id + 'percent'}
            type="number"
            required
            min="0"
            max="100"
            step="0.01"
            value={percent}
            onChange={(e) => setPercent(e.target.value)}
          />
        </>
      )}
      {error && <p role="alert">{error}</p>}
      <button
        className="btn-secondary"
        disabled={busy || disabled || !conversation}
      >
        {busy ? 'Préparation…' : 'Préparer un aperçu'}
      </button>
      {!conversation && (
        <p>Commence un échange avant de préparer une action.</p>
      )}
    </form>
  )
}

export default function AgentConsole({
  compact = false,
  pageContext = {},
  refreshKey = 0,
  onNavigate,
}) {
  const [messages, setMessages] = useState([]),
    [input, setInput] = useState(''),
    [conversation, setConversation] = useState(null),
    [conversations, setConversations] = useState([]),
    [missions, setMissions] = useState([]),
    [mission, setMission] = useState(0),
    [memories, setMemories] = useState([]),
    [briefing, setBriefing] = useState(null),
    [actions, setActions] = useState([]),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(''),
    [objective, setObjective] = useState(''),
    [includeContext, setIncludeContext] = useState(true),
    [builder, setBuilder] = useState(false),
    [pending, setPending] = useState(0),
    [preparing, setPreparing] = useState(false)
  const prefix = useId(),
    generation = useRef(0),
    bottom = useRef(null),
    mounted = useRef(true),
    request = useRef(null),
    refreshGeneration = useRef(0)
  const locked = busy || pending !== 0 || preparing
  const selectedMission = missions.find((m) => m.id === mission)
  const refresh = async () => {
    const token = ++refreshGeneration.current
    const results = await Promise.allSettled(
      ['missions', 'conversations', 'memories', 'briefing'].map((p) =>
        agentRequest(p),
      ),
    )
    if (!mounted.current || token !== refreshGeneration.current) return
    const setters = [
      (v) => setMissions(agentList(v?.missions)),
      (v) => setConversations(agentList(v?.conversations)),
      (v) => setMemories(agentList(v?.memories)),
      setBriefing,
    ]
    let failed = false
    results.forEach((r, i) => {
      if (r.status === 'fulfilled') setters[i](r.value)
      else failed = true
    })
    if (failed)
      setError(
        'Une partie du suivi est indisponible. Tu peux réessayer ; aucune absence de données n’est interprétée comme zéro.',
      )
  }
  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
      generation.current++
      request.current?.abort()
    }
  }, [])
  useEffect(() => {
    refresh()
  }, [refreshKey])
  useEffect(() => {
    bottom.current?.scrollIntoView({
      behavior: matchMedia('(prefers-reduced-motion: reduce)').matches
        ? 'auto'
        : 'smooth',
      block: 'nearest',
    })
  }, [messages.length])
  const loadActions = async (id, token = generation.current) => {
    if (!id) {
      setActions([])
      return
    }
    const data = await agentRequest(`actions?conversation_id=${id}`)
    if (mounted.current && token === generation.current)
      setActions(agentList(data?.actions))
  }
  const loadConversation = async (id) => {
    if (locked) return
    const token = ++generation.current
    setBusy(true)
    setError('')
    try {
      const [data, actionData] = await Promise.all([
        agentRequest(`conversations/${id}`),
        agentRequest(`actions?conversation_id=${id}`),
      ])
      if (token !== generation.current) return false
      if (!Array.isArray(data?.messages) || data.conversation_id !== Number(id))
        throw new Error('Historique de conversation incomplet')
      setConversation(Number(id))
      setMessages(data.messages)
      setActions(agentList(actionData?.actions))
      setBuilder(false)
      setMission((previous) =>
        missions.find((m) => m.id === previous)?.conversation_id === Number(id)
          ? previous
          : 0,
      )
      return true
    } catch (e) {
      if (token === generation.current) setError(e.message)
    } finally {
      if (token === generation.current) setBusy(false)
    }
  }
  const newConversation = () => {
    if (locked) return
    generation.current++
    setConversation(null)
    setMessages([])
    setActions([])
    setError('')
    setMission(0)
    setBuilder(false)
  }
  const chooseMission = async (id) => {
    if (Number(id) === 0) {
      setMission(0)
      return
    }
    const m = missions.find((v) => v.id === Number(id))
    if (m?.conversation_id) {
      if (!(await loadConversation(m.conversation_id))) return
    } else {
      setConversation(null)
      setMessages([])
      setActions([])
    }
    setMission(Number(id))
  }
  const send = async (text = input) => {
    if (!text.trim() || locked) return
    const token = ++generation.current
    setMessages((v) => [...v, { role: 'user', content: text }])
    setInput('')
    setBusy(true)
    setError('')
    request.current = new AbortController()
    try {
      const response = await fetch('/api/chat', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        signal: request.current.signal,
        body: JSON.stringify({
          conversation_id: conversation,
          mission_id: mission,
          messages: [{ role: 'user', content: text }],
          context: {
            timezone: agentTimezone(),
            page_context: includeContext ? pageContext : {},
          },
        }),
      })
      const data = await response.json().catch(() => ({}))
      if (!response.ok)
        throw new Error(data.error || `Erreur ${response.status}`)
      if (typeof data.answer !== 'string')
        throw new Error('Réponse assistant incomplète')
      if (!mounted.current || generation.current !== token) return
      setConversation(data.conversation_id || conversation)
      setMessages((v) => [
        ...v,
        {
          role: 'assistant',
          content: data.answer,
          memory: data.memory,
          evidence: data.evidence,
          status: data.status,
          steps: data.steps,
          context: data.context,
          data_quality: data.data_quality,
        },
      ])
      setActions((v) =>
        [...agentList(data.actions), ...v].filter(
          (a, i, all) => all.findIndex((b) => b.id === a.id) === i,
        ),
      )
      refresh()
    } catch (e) {
      if (
        mounted.current &&
        token === generation.current &&
        e.name !== 'AbortError'
      ) {
        setError(e.message)
        setMessages((v) => [
          ...v,
          { role: 'assistant', content: `Erreur : ${e.message}`, error: true },
        ])
      }
    } finally {
      if (mounted.current && token === generation.current) setBusy(false)
    }
  }
  const saveMission = async (e) => {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const result = await agentRequest('missions', 'POST', {
        objective,
        conversation_id: conversation,
      })
      setObjective('')
      setConversation(result.conversation_id || conversation)
      await refresh()
      setMission(result.id)
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }
  const missionStatus = async (status) => {
    try {
      await agentRequest(`missions/${mission}`, 'PATCH', { status })
      await refresh()
    } catch (e) {
      setError(e.message)
    }
  }
  const decide = async (action, decision) => {
    setPending(action.id)
    setError('')
    try {
      await agentRequest(`actions/${action.id}/decision`, 'POST', { decision })
      await loadActions(conversation)
      await refresh()
      window.dispatchEvent(new Event('themis:agent-applied'))
    } catch (e) {
      setError(e.message)
      await loadActions(conversation).catch(() => {})
    } finally {
      setPending(0)
    }
  }
  const ignore = async (d) => {
    try {
      await api.intelFeedback(d.id, 'hide')
      await refresh()
    } catch (e) {
      setError(e.message)
    }
  }
  const currentSelection = agentList(pageContext.transaction_ids)
  const initiatives = [
    ...missions
      .filter((m) => m.status === 'active' && m.new_statements > 0)
      .map((m) => ({ mission: m })),
    ...agentList(briefing?.discoveries).map((d) => ({ discovery: d })),
  ].slice(0, 3)
  const suggestions = [
    'Donne-moi ma top dépense du mois.',
    'Aide-moi à réduire mes dépenses fixes.',
    'Préparons le budget du mois prochain.',
    'Quelles données me manquent ?',
  ]
  const msgClass = compact ? 'assistant-panel-msg' : 'assistant-msg'
  return (
    <div
      className={`agent-console${compact ? ' agent-console-compact' : ''}`}
      role="region"
      aria-label={compact ? 'Agent Thémis flottant' : 'Agent Thémis'}
    >
      <div className="agent-toolbar">
        <label htmlFor={prefix + 'conversation'}>Conversation</label>
        <select
          id={prefix + 'conversation'}
          value={conversation || ''}
          disabled={locked}
          onChange={(e) =>
            e.target.value
              ? loadConversation(e.target.value)
              : newConversation()
          }
        >
          <option value="">Nouvel échange</option>
          {conversation &&
            !conversations.some((c) => c.id === conversation) && (
              <option value={conversation}>Échange en cours</option>
            )}
          {conversations.map((c) => (
            <option key={c.id} value={c.id}>
              {c.title}
            </option>
          ))}
        </select>
        <button
          className="btn-secondary"
          disabled={locked}
          onClick={newConversation}
        >
          Nouvel échange
        </button>
      </div>
      <details className="agent-workspace">
        <summary>Missions et préférences</summary>
        <label htmlFor={prefix + 'mission'}>Mission suivie</label>
        <select
          id={prefix + 'mission'}
          value={mission}
          disabled={locked}
          onChange={(e) => chooseMission(e.target.value)}
        >
          <option value="0">Sans mission</option>
          {missions.map((m) => (
            <option key={m.id} value={m.id}>
              {m.title} ·{' '}
              {m.status === 'active'
                ? 'En cours'
                : m.status === 'paused'
                  ? 'En pause'
                  : 'Terminée'}
            </option>
          ))}
        </select>
        {selectedMission && (
          <div className="agent-mission">
            <p>{selectedMission.objective}</p>
            <p>Dernier point : {date(selectedMission.last_review_at)}</p>
            {selectedMission.new_statements > 0 && (
              <p>
                {selectedMission.new_statements} nouveau(x) relevé(s) depuis ce
                point.
              </p>
            )}
            <div className="agent-buttons">
              <button
                className="btn-secondary"
                disabled={locked}
                onClick={() =>
                  send(
                    'Reprenons cette mission : examine les nouvelles données et propose la prochaine étape concrète, avec sources et limites.',
                  )
                }
              >
                Reprendre la mission
              </button>
              <button
                className="link"
                disabled={locked}
                onClick={() =>
                  missionStatus(
                    selectedMission.status !== 'active' ? 'active' : 'paused',
                  )
                }
              >
                {selectedMission.status !== 'active'
                  ? 'Réactiver'
                  : 'Mettre en pause'}
              </button>
              <button
                className="link"
                disabled={locked}
                onClick={() => missionStatus('completed')}
              >
                Marquer terminée
              </button>
            </div>
          </div>
        )}
        <form onSubmit={saveMission}>
          <label htmlFor={prefix + 'objective'}>
            Nouvelle mission : quel objectif ?
          </label>
          <textarea
            id={prefix + 'objective'}
            value={objective}
            onChange={(e) => setObjective(e.target.value)}
            maxLength={2000}
            required
            placeholder="Ex. Préparer mon budget ou examiner mes dépenses fixes."
          />
          <button
            className="btn-secondary"
            disabled={busy || !objective.trim()}
          >
            Créer la mission
          </button>
        </form>
        <h3>Préférences déclarées</h3>
        <p>
          Tu contrôles ce qui est retenu. Une préférence ne remplace pas les
          données financières.
        </p>
        {memories.length ? (
          <ul>
            {memories.map((m) => (
              <li key={m.id}>
                {m.content}{' '}
                <button
                  className="link"
                  onClick={async () => {
                    try {
                      await agentRequest(`memories/${m.id}`, 'DELETE')
                      await refresh()
                    } catch (e) {
                      setError(e.message)
                    }
                  }}
                >
                  Oublier
                </button>
              </li>
            ))}
          </ul>
        ) : (
          <p>Aucune préférence déclarée affichée.</p>
        )}
      </details>
      {selectedMission?.status === 'active' &&
        selectedMission.new_statements > 0 && (
          <aside className="notice" role="status">
            De nouveaux relevés sont disponibles pour « {selectedMission.title}{' '}
            ».{' '}
            <button
              className="link"
              disabled={locked}
              onClick={() =>
                send(
                  'Fais le point sur les nouvelles données de cette mission, avec les sources et les limites.',
                )
              }
            >
              Faire le point
            </button>
          </aside>
        )}
      <div
        className={compact ? 'assistant-panel-messages' : 'assistant-messages'}
      >
        {messages.length === 0 && (
          <>
            {initiatives.length > 0 && (
              <section
                className="agent-initiatives"
                aria-label="Points à reprendre"
              >
                <h3>À reprendre ensemble</h3>
                {initiatives.map((item, i) =>
                  item.mission ? (
                    <article key={'m' + item.mission.id}>
                      <strong>{item.mission.title}</strong>
                      <p>
                        De nouveaux relevés sont disponibles pour cette mission.
                      </p>
                      <button
                        className="btn-secondary"
                        disabled={locked}
                        onClick={() => chooseMission(item.mission.id)}
                      >
                        Ouvrir la mission
                      </button>
                    </article>
                  ) : (
                    <article key={'d' + item.discovery.id}>
                      <strong>{item.discovery.title}</strong>
                      <p>{item.discovery.summary}</p>
                      <details>
                        <summary>Sources et limites du constat</summary>
                        <p>Période : {item.discovery.period}</p>
                        <ul>
                          {agentList(item.discovery.limitations).map(
                            (limit, j) => (
                              <li key={j}>{limit}</li>
                            ),
                          )}
                        </ul>
                        {agentList(item.discovery.evidence?.transaction_ids)
                          .length > 0 && (
                          <p>
                            Opérations sources :{' '}
                            {item.discovery.evidence.transaction_ids.join(', ')}
                          </p>
                        )}
                      </details>
                      <div className="agent-buttons">
                        <button
                          className="btn-secondary"
                          disabled={locked}
                          onClick={() =>
                            send(
                              `Examine la découverte vérifiée ${item.discovery.id} (${item.discovery.title}) et propose la prochaine étape, avec les preuves et les limites.`,
                            )
                          }
                        >
                          Examiner avec Thémis
                        </button>
                        <button
                          className="link"
                          onClick={() => ignore(item.discovery)}
                        >
                          Ignorer
                        </button>
                      </div>
                    </article>
                  ),
                )}
              </section>
            )}
            <div
              className={
                compact
                  ? 'assistant-panel-suggestions'
                  : 'assistant-suggestions'
              }
            >
              <p>Que souhaites-tu faire avancer ?</p>
              {suggestions.map((s) => (
                <button
                  className={
                    compact ? 'assistant-panel-sug' : 'assistant-suggestion'
                  }
                  key={s}
                  disabled={locked}
                  onClick={() => send(s)}
                >
                  {s}
                </button>
              ))}
            </div>
          </>
        )}
        {messages.map((m, i) => (
          <div
            key={m.id || i}
            className={`${msgClass} ${msgClass}-${m.role}${m.error ? ' error' : ''}`}
          >
            <div className={`${msgClass}-avatar`} aria-hidden="true">
              {m.role === 'user' ? 'V' : 'T'}
            </div>
            <div className={`${msgClass}-body`}>
              <div
                className={
                  compact ? 'assistant-panel-msg-text' : 'assistant-msg-text'
                }
              >
                {m.content}
              </div>
              {m.context?.target_month && (
                <p className="agent-period">
                  Objectif de préparation : {m.context.target_month}. Les
                  observations restent séparées du scénario.
                </p>
              )}
              {m.context?.period?.from && (
                <p className="agent-period">
                  Période interprétée : {m.context.period.from} →{' '}
                  {m.context.period.to} · {m.context.timezone}
                </p>
              )}
              <KnowledgeTrace trace={m.memory} />
              {agentList(m.steps).length > 0 && (
                <details className="agent-steps">
                  <summary>Travail réalisé · {m.steps.length} étape(s)</summary>
                  <ol>
                    {m.steps.map((step, j) => (
                      <li key={j}>
                        {toolLabels[step.name] || step.name} ·{' '}
                        {step.status === 'success' ? 'Réussi' : 'Échec'}
                        {step.from && ` · ${step.from} → ${step.to}`}
                      </li>
                    ))}
                  </ol>
                </details>
              )}
              {agentList(m.evidence).length > 0 && (
                <details className="assistant-evidence">
                  <summary>Voir les données sources</summary>
                  {m.evidence.map((ev, j) => (
                    <div key={j}>
                      <strong>
                        {toolLabels[ev.tool] ||
                          (ev.type === 'transaction'
                            ? 'Opération source'
                            : ev.key || 'Données consultées')}
                      </strong>
                      {ev.type === 'transaction' ? (
                        <p>
                          {ev.date} · {ev.label} · {ev.amount} €{' '}
                          {onNavigate && (
                            <button
                              className="link"
                              onClick={() =>
                                onNavigate('transactions', { ids: [ev.id] })
                              }
                            >
                              Voir l’opération
                            </button>
                          )}
                        </p>
                      ) : (
                        <SourceData
                          data={ev.data || ev}
                          onNavigate={onNavigate}
                        />
                      )}
                    </div>
                  ))}
                </details>
              )}
            </div>
          </div>
        ))}
        {busy && (
          <p role="status" className="agent-busy">
            <span className="spinner" />
            Consultation des données et préparation de la réponse…
          </p>
        )}
        <div ref={bottom} />
      </div>
      {actions.length > 0 && (
        <details
          className="agent-actions"
          open={actions.some((a) => a.status === 'prepared')}
        >
          <summary>
            Actions proposées ·{' '}
            {actions.filter((a) => a.status === 'prepared').length} à valider
          </summary>
          {actions.map((a) => (
            <article key={a.id}>
              <h3>
                {labels[a.kind] || a.kind} · {states[a.status] || a.status}
              </h3>
              <details open={a.status === 'prepared'}>
                <summary>Vérifier l’aperçu</summary>
                <Preview action={a} />
              </details>
              {a.status === 'prepared' && (
                <div className="agent-buttons">
                  <button
                    className="btn"
                    disabled={locked}
                    onClick={() => decide(a, 'apply')}
                  >
                    {pending === a.id
                      ? 'Enregistrement…'
                      : 'Valider et appliquer'}
                  </button>
                  <button
                    className="btn-secondary"
                    disabled={locked}
                    onClick={() => decide(a, 'dismiss')}
                  >
                    Ignorer cette action
                  </button>
                </div>
              )}
              {a.status === 'stale' && (
                <p>Les données ont changé. Prépare un nouvel aperçu.</p>
              )}
            </article>
          ))}
        </details>
      )}
      <details
        className="agent-builder-toggle"
        open={builder}
        onToggle={(e) => setBuilder(e.currentTarget.open)}
      >
        <summary>Préparer une action ou mémoriser une décision</summary>
        {builder && (
          <ActionBuilder
            key={conversation || 'new'}
            disabled={locked}
            conversation={conversation}
            mission={mission}
            messages={messages}
            selection={currentSelection}
            month={pageContext.month}
            onBusyChange={setPreparing}
            onPrepared={(a) => setActions((v) => [a, ...v])}
          />
        )}
      </details>
      {Object.hasOwn(briefing || {}, 'last_transaction_date') && (
        <p className="agent-period">
          Dernière opération connue :{' '}
          {briefing.last_transaction_date || 'date non établie'}. La fraîcheur
          des données dépend des relevés importés.
        </p>
      )}
      {pageContext.page && pageContext.page !== 'assistant' && (
        <label className="agent-choice agent-context">
          <input
            type="checkbox"
            checked={includeContext}
            onChange={(e) => setIncludeContext(e.target.checked)}
          />
          <span>
            Contexte de la page :{' '}
            {pageLabels[pageContext.page] || pageContext.page}
            {pageContext.month && ` · ${pageContext.month}`}
            {currentSelection.length > 0 &&
              ` · ${currentSelection.length} opération(s) sélectionnée(s)`}
            . Une période demandée explicitement reste prioritaire.{' '}
            {pageContext.filters_require_selection &&
              'Des filtres sont actifs : sélectionne les opérations pour les analyser précisément.'}
          </span>
        </label>
      )}
      {error && (
        <div className="notice err" role="alert">
          {error}{' '}
          <button
            className="link"
            onClick={() => {
              setError('')
              refresh()
            }}
          >
            Actualiser le suivi
          </button>
        </div>
      )}
      <form
        className={compact ? 'assistant-panel-input' : 'assistant-input'}
        onSubmit={(e) => {
          e.preventDefault()
          send()
        }}
      >
        <input
          className={
            compact ? 'assistant-panel-field' : 'assistant-input-field'
          }
          aria-label="Votre question"
          placeholder="Une question, un objectif, une prochaine étape…"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          maxLength={8000}
          disabled={locked}
        />
        <button
          type="submit"
          className={
            compact ? 'btn assistant-panel-send' : 'btn assistant-send'
          }
          disabled={locked || !input.trim()}
        >
          {busy ? '…' : 'Envoyer'}
        </button>
      </form>
    </div>
  )
}
