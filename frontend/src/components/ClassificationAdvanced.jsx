import { useEffect, useRef, useState } from 'react'
import { api } from '../api'

const descriptions = {
  initial_classification: 'Classification lors de l’import',
  domain_changed: 'Catégorie ou origine modifiée',
  tag_added: 'Tag ajouté', tag_removed: 'Tag retiré', tag_decision: 'Choix explicite sur un tag',
  tag_renamed: 'Tag renommé, protection conservée', locks_changed: 'Protections modifiées',
  suggestion_rejected: 'Suggestion IA refusée', reevaluation: 'Réévaluation demandée',
}

export default function ClassificationAdvanced({ transaction, onSave, onRefresh }) {
  const [domainLocked, setDomainLocked] = useState(Boolean(transaction.domain_locked))
  const [tagsLocked, setTagsLocked] = useState(Boolean(transaction.tags_locked))
  const [busy, setBusy] = useState(false), [error, setError] = useState(''), [notice, setNotice] = useState('')
  const [history, setHistory] = useState(null), [events, setEvents] = useState([])
  const alive = useRef(true)
  useEffect(() => { alive.current = true; return () => { alive.current = false } }, [])
  const run = async action => {
    if (busy) return
    setBusy(true); setError(''); setNotice('')
    try { await action() } catch (e) { if (alive.current) setError(e.message) }
    finally { if (alive.current) setBusy(false) }
  }
  const loadHistory = async more => {
    const data = await api.classificationHistory(transaction.id, more ? events.at(-1)?.id : '')
    if (!Array.isArray(data.events)) throw new Error('Historique indisponible')
    if (alive.current) { setHistory(data); setEvents(current => more ? [...current, ...data.events] : data.events) }
  }
  return <details className="transaction-classification-advanced">
    <summary>Classification avancée</summary>
    <p>{transaction.classification_configured ? 'Les décisions sont suivies pour cette opération.' : 'Les décisions antérieures à l’activation ne sont pas reconstituées.'}</p>
    <label><input type="checkbox" checked={domainLocked} disabled={busy} onChange={e => setDomainLocked(e.target.checked)} /> Verrouiller le domaine</label>
    <label><input type="checkbox" checked={tagsLocked} disabled={busy} onChange={e => setTagsLocked(e.target.checked)} /> Verrouiller les tags</label>
    <p>Ces protections bloquent les automatismes. Tu peux toujours faire une correction explicite. Déverrouiller conserve les choix manuels.</p>
    <button type="button" className="btn-secondary" disabled={busy} onClick={() => run(async () => {
      await onSave(transaction, { domain_locked: domainLocked, tags_locked: tagsLocked })
      if (alive.current) { setNotice('Protections enregistrées.'); setHistory(null) }
    })}>Enregistrer les protections</button>{' '}
    <button type="button" className="btn-secondary" disabled={busy} onClick={() => run(async () => {
      await onSave(transaction, { reevaluate: true })
      await api.aiEnrich(undefined, transaction.id)
      await onRefresh?.()
      if (alive.current) { setNotice('Réévaluation terminée. Les propositions restent à valider.'); setHistory(null) }
    })}>Demander une réévaluation IA</button>
    <p><button type="button" className="link" disabled={busy} onClick={() => run(() => loadHistory(false))}>Voir les dernières décisions</button></p>
    {history && <>
      {events.length ? <ol>{events.map(event => <li key={event.id}>{descriptions[event.event_type] || 'Décision de classification'} · <time dateTime={event.at}>{new Date(event.at).toLocaleString('fr-FR')}</time>{event.origin === 'manual' || event.origin === 'agent' ? ' · validée par toi' : ''}</li>)}</ol> : <p>Aucune décision tracée. L’origine des classifications anciennes reste inconnue.</p>}
      {history.has_more && <button type="button" className="link" disabled={busy} onClick={() => run(() => loadHistory(true))}>Voir les décisions précédentes</button>}
    </>}
    {notice && <p role="status">{notice}</p>}
    {error && <div role="alert" className="notice err">{error}{' '}<button type="button" className="link" disabled={busy} onClick={() => run(async () => { await onRefresh?.(); if (alive.current) setNotice('État rechargé. Ton brouillon est conservé ; vérifie-le avant de réessayer.') })}>Recharger l’état</button></div>}
  </details>
}
