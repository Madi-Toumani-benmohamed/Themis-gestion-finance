import { useCallback, useEffect, useState } from 'react'
import { api } from '../api'

const feedback = [
  ['interesting', 'Intéressant'],
  ['already_knew', 'Je le savais'],
  ['irrelevant', 'Peu pertinent'],
  ['normal', 'C’est normal'],
]

function euros(cents) {
  return new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR' }).format(cents / 100)
}

function metricValue(metric) {
  if (metric.kind === 'money_cents') return euros(metric.value)
  if (metric.kind === 'percent') return `${(metric.value / 100).toLocaleString('fr-FR')} %`
  if (metric.kind === 'count') return new Intl.NumberFormat('fr-FR').format(metric.value)
  return metric.value
}

function metricPairs(item) {
  const values = Object.fromEntries((item.metrics || []).map(metric => [metric.name, metric]))
  const pairs = {
    progressive_trend: [['first_period_cents', 'Premier mois observé'], ['last_period_cents', 'Dernier mois observé']],
    spend_variation_contributors: [['baseline_three_months_cents', 'Période de référence'], ['current_three_months_cents', 'Période récente']],
    weekend_frequency_change: [['previous_weekend_transactions', 'Avant'], ['current_weekend_transactions', 'Récemment']],
    purchase_frequency_change: [['previous_transactions', 'Avant'], ['current_transactions', 'Récemment']],
  }[item.kind]
  if (!pairs) return null
  const result = pairs.map(([key, label]) => values[key] ? { ...values[key], label } : null).filter(Boolean)
  return result.length === 2 ? result : null
}

function MetricVisualization({ item }) {
  const data = metricPairs(item)
  if (!data) return null
  const max = Math.max(...data.map(value => Math.abs(value.value)), 1)
  return (
    <figure className="intel-metric-chart">
      <figcaption>Comparaison des valeurs observées</figcaption>
      <div aria-hidden="true" className="intel-chart-bars">
        {data.map(value => <div key={value.label} className="intel-chart-row"><span>{value.label}</span><i><b style={{ width: `${Math.max(2, Math.abs(value.value) / max * 100)}%` }} /></i><strong>{metricValue(value)}</strong></div>)}
      </div>
      <table className="intel-chart-table"><caption>Données de la comparaison</caption><thead><tr><th scope="col">Période</th><th scope="col">Valeur</th></tr></thead><tbody>{data.map(value => <tr key={value.label}><th scope="row">{value.label}</th><td>{metricValue(value)}</td></tr>)}</tbody></table>
    </figure>
  )
}

export function IntelligenceHighlights({ onNavigate, month }) {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [feedbackStatus, setFeedbackStatus] = useState('')
  const [expanded, setExpanded] = useState({})
  const [managePreferences, setManagePreferences] = useState(false)
  const [preferences, setPreferences] = useState([])
  const [preferenceError, setPreferenceError] = useState('')
  const [history, setHistory] = useState({})
  const [historyLoading, setHistoryLoading] = useState({})
  const [jobStatus, setJobStatus] = useState(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const result = await api.intelFeed(3)
      setItems(result.discoveries || [])
    } catch (err) {
      setError(err.message || 'Découvertes indisponibles')
      setItems([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  useEffect(() => {
    let active = true
    let wasPending = false
    let previousRequest = ''
    const poll = async () => {
      try {
        const status = await api.intelJobStatus()
        if (!active) return
        setJobStatus(status)
        const pending = status.status === 'queued' || status.status === 'running'
        const requestChanged = previousRequest && status.requested_at && previousRequest !== status.requested_at
        if ((wasPending && !pending) || (requestChanged && !pending && status.status === 'done')) await load()
        wasPending = pending
        previousRequest = status.requested_at || previousRequest
      } catch { /* Le feed existant reste lisible si le suivi du job échoue. */ }
    }
    poll()
    const timer = window.setInterval(poll, 5000)
    return () => { active = false; window.clearInterval(timer) }
  }, [load])

  useEffect(() => {
    if (!managePreferences) return
    let active = true
    api.intelPreferences().then(result => { if (active) setPreferences(result.preferences || []) }).catch(err => { if (active) setPreferenceError(err.message || 'Préférences indisponibles') })
    return () => { active = false }
  }, [managePreferences])

  const respond = async (item, action) => {
    setBusy(true)
    setError('')
    setFeedbackStatus('')
    try {
      await api.intelFeedback(item.id, action)
      setFeedbackStatus('Retour enregistré. Il influencera la priorité au prochain recalcul.')
      await load()
    } catch (err) {
      setError(err.message || 'Retour non enregistré')
    } finally {
      setBusy(false)
    }
  }

  const recompute = async () => {
    setBusy(true)
    setError('')
    try {
      await api.recomputeIntel(month)
      await load()
    } catch (err) {
      setError(err.message || 'Recalcul impossible')
    } finally {
      setBusy(false)
    }
  }

  const savePreference = async (preference) => {
    setBusy(true)
    setPreferenceError('')
    try {
      await api.setIntelPreference(preference.discovery_id, preference.action, preference.note)
      const result = await api.intelPreferences()
      setPreferences(result.preferences || [])
    } catch (err) {
      setPreferenceError(err.message || 'Préférence non enregistrée')
    } finally {
      setBusy(false)
    }
  }

  const clearPreference = async (preference) => {
    setBusy(true)
    setPreferenceError('')
    try {
      await api.clearIntelPreference(preference.discovery_id)
      setPreferences(current => current.filter(item => item.discovery_id !== preference.discovery_id))
    } catch (err) {
      setPreferenceError(err.message || 'Préférence non supprimée')
    } finally {
      setBusy(false)
    }
  }

  const toggleDetail = async item => {
    if (expanded[item.id]) {
      setExpanded(current => ({ ...current, [item.id]: false }))
      return
    }
    setExpanded(current => ({ ...current, [item.id]: true }))
    if (history[item.id]) return
    setHistoryLoading(current => ({ ...current, [item.id]: true }))
    try {
      const detail = await api.intelDiscovery(item.id)
      setHistory(current => ({ ...current, [item.id]: detail.observations || [] }))
    } catch (err) {
      setError(err.message || 'Historique de cette découverte indisponible')
    } finally {
      setHistoryLoading(current => ({ ...current, [item.id]: false }))
    }
  }

  return (
    <section className="intel-highlights panel-level-focus" aria-labelledby="intel-highlights-title">
      <div className="intel-highlights-head">
        <div>
          <p className="eyebrow">Découvertes vérifiables</p>
          <h2 id="intel-highlights-title">Ce que Thémis a remarqué</h2>
          <p>Jusqu’à trois constats prioritaires, établis à partir des périodes documentées.</p>
        </div>
        <button className="btn-secondary" type="button" onClick={recompute} disabled={busy || loading}>
          {busy ? 'Mise à jour…' : 'Recalculer'}
        </button>
      </div>
      {error && <p className="notice err" role="alert">{error}</p>}
      {(jobStatus?.status === 'queued' || jobStatus?.status === 'running') && <p className="intel-job-status" role="status">Thémis recalcule les découvertes après une modification récente…</p>}
      {feedbackStatus && <p className="intel-feedback-status" role="status">{feedbackStatus}</p>}
      {loading ? <p role="status">Chargement des découvertes…</p> : items.length === 0 ? (
        <p className="intel-empty">Aucune découverte suffisamment étayée pour le moment. Une liste vide est préférable à un signal sans preuve.</p>
      ) : (
        <div className="intel-card-list">
          {items.slice(0, 3).map(item => {
            const isExpanded = Boolean(expanded[item.id])
            const ids = item.evidence?.transaction_ids || []
            return (
              <article className="intel-card" key={item.id}>
                <div className="intel-card-main">
                  <div>
                    <p className="intel-card-meta">{item.period || 'Période non précisée'} · Qualité des preuves : {item.confidence >= 80 ? 'élevée' : item.confidence >= 60 ? 'moyenne' : 'limitée'}</p>
                    <h3>{item.title}</h3>
                    <p>{item.summary}</p>
                  </div>
                  <button type="button" className="link intel-detail-toggle" aria-expanded={isExpanded} onClick={() => toggleDetail(item)}>
                    {isExpanded ? 'Masquer l’historique' : 'Voir l’historique, le calcul et les limites'}
                  </button>
                </div>
                {isExpanded && (
                  <div className="intel-evidence">
                    <MetricVisualization item={item} />
                    {item.metrics?.length > 0 && <dl className="intel-metrics">{item.metrics.map(metric => <div key={metric.name}><dt>{metric.name.replaceAll('_', ' ')}</dt><dd>{metricValue(metric)}</dd></div>)}</dl>}
                    <p>{item.coverage?.description || 'Couverture documentaire non précisée.'}</p>
                    {item.limitations?.map(limitation => <p className="intel-limitation" key={limitation}>{limitation}</p>)}
                    <p>{ids.length} opération{ids.length > 1 ? 's' : ''} référencée{ids.length > 1 ? 's' : ''} · {(item.evidence?.periods || []).join(', ')}</p>
                    {historyLoading[item.id] ? <p role="status">Chargement de l’historique…</p> : history[item.id]?.length > 0 && <ol className="intel-observation-history">{history[item.id].map(observation => <li key={observation.observation_id}><time>{observation.period || new Date(observation.updated_at).toLocaleDateString('fr-FR')}</time><strong>{observation.title}</strong><p>{observation.summary}</p><span>Confiance des preuves : {observation.confidence}%</span></li>)}</ol>}
                  </div>
                )}
                <div className="intel-card-actions" aria-label="Votre retour sur cette découverte">
                  {feedback.map(([action, label]) => <button key={action} type="button" className="intel-feedback" disabled={busy} onClick={() => respond(item, action)}>{label}</button>)}
                  <button type="button" className="intel-feedback intel-feedback-hide" disabled={busy} onClick={() => respond(item, 'hide')}>Masquer</button>
                  {ids.length > 0 && <button type="button" className="link intel-source-link" onClick={() => onNavigate?.('transactions', { ids, month: item.period || month })}>Voir les opérations sources →</button>}
                </div>
              </article>
            )
          })}
        </div>
      )}
      <div className="intel-preferences-manager">
        <button type="button" className="link" aria-expanded={managePreferences} onClick={() => setManagePreferences(value => !value)}>
          {managePreferences ? 'Fermer la gestion des retours' : 'Gérer mes retours et préférences'}
        </button>
        {managePreferences && (
          <div className="intel-preferences-list">
            <p>Ces choix ajustent la priorité des découvertes similaires; ils ne changent ni les calculs ni leur niveau de confiance.</p>
            {preferenceError && <p className="notice err" role="alert">{preferenceError}</p>}
            {preferences.length === 0 ? <p>Aucun retour enregistré.</p> : preferences.map(preference => (
              <div className="intel-preference-row" key={preference.discovery_id}>
                <strong>{preference.title}</strong>
                <label>Retour
                  <select value={preference.action} disabled={busy} onChange={event => setPreferences(current => current.map(item => item.discovery_id === preference.discovery_id ? { ...item, action: event.target.value } : item))}>
                    {feedback.map(([action, label]) => <option key={action} value={action}>{label}</option>)}
                    <option value="remind_me">Me le rappeler</option>
                  </select>
                </label>
                <label>Note de contexte
                  <input maxLength={500} value={preference.note || ''} disabled={busy} onChange={event => setPreferences(current => current.map(item => item.discovery_id === preference.discovery_id ? { ...item, note: event.target.value } : item))} />
                </label>
                <button className="btn-secondary" type="button" disabled={busy} onClick={() => savePreference(preference)}>Enregistrer</button>
                <button className="link" type="button" disabled={busy} onClick={() => clearPreference(preference)}>Supprimer ce retour</button>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
