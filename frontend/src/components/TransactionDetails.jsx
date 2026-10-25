import { useModal } from './useModal'
import { useId, useRef, useState } from 'react'
import { transactionCents, transactionDate, transactionMoney, transactionTags, transactionTypeLabel } from './transactionPageModel.mjs'
import TransactionReimbursements from './TransactionReimbursements'

export function TransactionDrawer({ transaction: tx, domains, allTags = [], onSave, onClose, onRefresh }) {
  const ref = useRef(null), title = useId()
  const [domain, setDomain] = useState(tx.domain_id ?? ''), [tags, setTags] = useState((tx.tags || []).join(', '))
  const [categoryDirty, setCategoryDirty] = useState(false)
  const [busy, setBusy] = useState(false), [error, setError] = useState(null), [saved, setSaved] = useState(false)
  useModal(ref)
  const currentTag = tags.split(',').at(-1).trim().toLowerCase()
  const tagSuggestions = currentTag ? allTags.map(tag => tag.name).filter(name => name.includes(currentTag) && !transactionTags(tags).includes(name)).slice(0, 6) : []
  const categoryChanged = String(domain) !== String(tx.domain_id ?? '')
  const close = () => { if (!busy) onClose() }
  const save = async event => {
    event.preventDefault(); setBusy(true); setError(null); setSaved(false)
    try { const patch = { tags: transactionTags(tags) }; if (categoryDirty || categoryChanged) patch.domain_id = domain === '' ? null : Number(domain); await onSave(tx, patch); setCategoryDirty(false); setSaved(true) }
    catch (error) { setError(error.message) }
    finally { setBusy(false) }
  }
  return <dialog ref={ref} className="transaction-drawer" aria-labelledby={title} onCancel={event => { event.preventDefault(); close() }}>
    <header><div><span className="eyebrow">Opération #{tx.id}</span><h2 id={title}>Détail de la transaction</h2></div><button type="button" className="btn-secondary" onClick={close} disabled={busy} aria-label="Fermer le détail">✕</button></header>
    <div className="transaction-drawer-body">
      <p className={'transaction-detail-amount' + (transactionCents(tx) > 0 ? ' credit' : '')}>{transactionCents(tx) > 0 ? '+' : ''}{transactionMoney(transactionCents(tx))}</p>
      <h3>{tx.display_label || tx.merchant || tx.label}</h3>
      <dl className="transaction-facts">
        <div><dt>Date de l’opération</dt><dd>{transactionDate(tx.date)}</dd></div>
        {tx.value_date && <div><dt>Date de valeur</dt><dd>{transactionDate(tx.value_date)}</dd></div>}
        {tx.source_page && <div><dt>Référence dans le document extrait</dt><dd>Page {tx.source_page}, ligne {tx.source_line || "non disponible"}</dd></div>}
        <div><dt>Compte</dt><dd>{tx.account_name || 'Compte non documenté'}</dd></div>
        <div><dt>Marchand normalisé</dt><dd>{tx.merchant || 'Non renseigné'}</dd></div>
        <div><dt>Libellé bancaire original</dt><dd className="transaction-raw-full">{tx.label}</dd></div>
        <div><dt>Type analytique</dt><dd>{transactionTypeLabel(tx.financial_type)}</dd></div>
        <div><dt>Relevé source</dt><dd>{tx.statement_filename || (tx.statement_id ? `Relevé #${tx.statement_id}` : 'Import non documenté')}</dd></div>
        {tx.statement_id && <div><dt>Période déclarée du relevé</dt><dd>{transactionDate(tx.statement_period_start)} — {transactionDate(tx.statement_period_end)}<small>{tx.statement_validated ? 'Équilibre du relevé confirmé ; la couverture et les autres contrôles restent à vérifier.' : 'Validation du relevé non établie.'}</small></dd></div>}
        <div><dt>Rapprochement</dt><dd>Aucun rapprochement confirmé documenté.</dd></div>
        <div><dt>Origine de la catégorisation</dt><dd>{({ manual: 'Validée manuellement', rule: 'Règle déterministe', ai: 'Attribution IA historique', unknown: 'Origine non documentée' })[tx.classification_source] || 'Origine non documentée'}</dd></div>
      </dl>
      <p className="transaction-note">Les écritures originales sont conservées. Un crédit peut correspondre à un revenu, un remboursement ou une annulation ; son signe seul ne prouve pas sa nature.</p>
      <TransactionReimbursements transaction={tx} onRefresh={onRefresh}/>
      {tx.suggestion && <aside className="transaction-ai-suggestion"><h3>Proposition IA à vérifier</h3><p>Catégorie suggérée : {domains.find(domain => domain.id === tx.suggestion.domain_id)?.name || 'Non proposée'}. Tags suggérés : {tx.suggestion.tags?.length ? tx.suggestion.tags.map(tag => `#${tag}`).join(', ') : 'aucun'}.</p><p>Cette proposition n’est pas un fait vérifié. Les données sources restent prioritaires.</p><button type="button" className="btn-secondary" disabled={busy} onClick={() => { if (tx.suggestion.domain_id) setDomain(tx.suggestion.domain_id); setTags([...new Set([...(tx.tags || []), ...(tx.suggestion.tags || [])])].join(', ')); setCategoryDirty(Boolean(tx.suggestion.domain_id)); setSaved(false) }}>Reprendre dans le formulaire</button><small>Vérifie les champs puis clique sur Enregistrer pour appliquer explicitement la correction.</small><button type="button" className="link" disabled={busy} onClick={async () => { setBusy(true); setError(null); try { await onSave(tx, { discard_suggestion: true }) } catch (error) { setError(error.message) } finally { setBusy(false) } }}>Ignorer la suggestion</button></aside>}
      <form onSubmit={save}>
        <h3>Corriger la classification</h3>
        <label>Catégorie<span className={'dom-pick transaction-category-picker' + (domain ? ' set' : '')} style={{ '--dc': domains.find(d => String(d.id) === String(domain))?.color || 'var(--faint)' }}><span className="dom-dot" aria-hidden="true"/><select value={domain} onChange={event => { setDomain(event.target.value); setCategoryDirty(true); setSaved(false) }} disabled={busy}><option value="">Sans catégorie</option>{domains.map(d => <option key={d.id} value={d.id}>{d.parent_id ? '↳ ' : ''}{d.name}</option>)}</select></span></label>
        <label className="transaction-manual-validation"><input type="checkbox" checked={categoryDirty || categoryChanged} onChange={event => { setCategoryDirty(event.target.checked); setSaved(false) }} disabled={busy || categoryChanged}/>Valider explicitement la catégorie sélectionnée</label>
        <label>Tags<input value={tags} onChange={event => { setTags(event.target.value); setSaved(false) }} placeholder="taxes, mensuel…" aria-describedby={`${title}-tag-help`} disabled={busy}/></label><small id={`${title}-tag-help`}>Sépare les tags par des virgules. Un champ vide retire les tags.</small>
        {tagSuggestions.length > 0 && <div className="transaction-tag-suggestions" aria-label="Suggestions de tags">{tagSuggestions.map(tag => <button type="button" className="btn-secondary" key={tag} disabled={busy} onClick={() => { const parts = tags.split(','); parts[parts.length - 1] = tag; setTags(parts.join(', ') + ', '); setSaved(false) }}>#{tag}</button>)}</div>}
        {error && <p className="notice err" role="alert">{error}</p>}
        {saved && <p className="transaction-success" role="status">Changements enregistrés.</p>}
        <footer><button className="btn-secondary" type="button" onClick={close} disabled={busy}>Fermer</button><button className="btn" disabled={busy}>{busy ? 'Enregistrement…' : 'Enregistrer'}</button></footer>
      </form>
    </div>
  </dialog>
}
export function TransactionBulkDialog({ rows, domains, onApply, onClose }) {
  const ref = useRef(null), title = useId()
  const [action, setAction] = useState('category'), [domain, setDomain] = useState(''), [tags, setTags] = useState('')
  const [busy, setBusy] = useState(false), [result, setResult] = useState(null), [error, setError] = useState(null)
  useModal(ref)
  const close = () => { if (!busy) onClose() }
  const apply = async event => {
    event.preventDefault(); setError(null)
    if (action !== 'category' && !transactionTags(tags).length) { setError('Renseigne au moins un tag.'); return }
    setBusy(true)
    try { setResult(await onApply(rows, { action, domain_id: domain ? Number(domain) : null, tags: transactionTags(tags) })) }
    catch (error) { setError(error.message) }
    finally { setBusy(false) }
  }
  return <dialog ref={ref} className="transaction-bulk-dialog" aria-labelledby={title} onCancel={event => { event.preventDefault(); close() }}><form onSubmit={apply}>
    <header><h2 id={title}>Modifier {rows.length} opérations</h2><button className="link" type="button" disabled={busy} onClick={close} aria-label="Fermer la modification multiple">✕</button></header>
    <p>Cette action modifie explicitement les opérations sélectionnées sur cette page. Vérifie le périmètre avant de confirmer.</p>
    {!result && <><label>Action<select value={action} onChange={event => setAction(event.target.value)} disabled={busy}><option value="category">Changer la catégorie</option><option value="add_tags">Ajouter des tags</option><option value="remove_tags">Retirer des tags</option></select></label>{action === 'category' ? <label>Nouvelle catégorie<select value={domain} onChange={event => setDomain(event.target.value)} disabled={busy}><option value="">Sans catégorie</option>{domains.map(d => <option key={d.id} value={d.id}>{d.parent_id ? '↳ ' : ''}{d.name}</option>)}</select></label> : <label>Tags concernés<input value={tags} onChange={event => setTags(event.target.value)} disabled={busy} placeholder="taxes, mensuel…"/></label>}</>}
    {error && <p role="alert" className="notice err">{error}</p>}
    {busy && <p role="status">Enregistrement des opérations sélectionnées…</p>}
    {result && <div role="status"><p>{result.succeeded} opération(s) enregistrée(s), {result.failed.length} en échec.</p>{result.failed.length > 0 && <><p>Les opérations en échec sont listées ici ; elles restent sélectionnées si elles sont encore sur cette page. Aucune réussite n’est annulée automatiquement.</p><ul>{result.failed.map(item => <li key={item.id}>Opération #{item.id} : {item.error}</li>)}</ul></>}</div>}
    <footer><button className="btn-secondary" type="button" disabled={busy} onClick={close}>{result ? 'Terminer' : 'Annuler'}</button>{!result && <button className="btn" disabled={busy}>{busy ? 'Enregistrement…' : `Confirmer pour ${rows.length} opérations`}</button>}</footer>
  </form></dialog>
}
