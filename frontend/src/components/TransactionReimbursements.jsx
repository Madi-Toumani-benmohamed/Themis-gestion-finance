import { useEffect, useId, useRef, useState } from 'react'
import { api } from '../api'
import { allocationRequest, amountInputFromCents, personalCostCents, reimbursementKindLabel, reimbursementStatusLabel } from './reimbursementModel.mjs'
import { transactionCents, transactionDate, transactionMoney } from './transactionPageModel.mjs'

let fallbackRequestID = 0
const newIdempotencyKey = () => globalThis.crypto?.randomUUID?.() || `reimbursement-${Date.now()}-${++fallbackRequestID}`

const transactionName = operation => operation.display_label || operation.merchant || operation.label || 'Libellé indisponible'

function ReimbursementAllocationForm({ reimbursementCase, onSubmit, onCancel, busy }) {
  const title = useId()
  const [expenses, setExpenses] = useState(null)
  const [loadError, setLoadError] = useState(null)
  const [expenseID, setExpenseID] = useState('')
  const [amount, setAmount] = useState('')
  const [error, setError] = useState(null)
  const requestKey = useRef(null)

  useEffect(() => {
    let active = true
    api.eligibleReimbursementExpenses(reimbursementCase.id, { limit: 100 }).then(result => {
      if (active) setExpenses(result.expenses || [])
    }).catch(reason => {
      if (active) setLoadError(reason.message || 'Dépenses éligibles indisponibles.')
    })
    return () => { active = false }
  }, [reimbursementCase.id])

  const selected = expenses?.find(item => String(item.transaction_id) === expenseID)
  const maxCents = selected ? Math.min(reimbursementCase.unallocated_cents, selected.remaining_cents) : 0
  const draft = () => allocationRequest(Number(expenseID), amount, reimbursementCase.unallocated_cents, selected?.remaining_cents || 0)
  let draftError = null
  if (expenseID && amount) {
    try { draft() } catch (reason) { draftError = reason.message }
  }

  const submit = async event => {
    event.preventDefault()
    setError(null)
    let request
    try { request = draft() } catch (reason) { setError(reason.message); return }
    requestKey.current ||= newIdempotencyKey()
    try {
      const saved = await onSubmit(request, requestKey.current)
      if (saved) requestKey.current = null
      else setError('Association non confirmée. Vérifie le message ci-dessus puis réessaie.')
    } catch (reason) {
      setError(reason.message || 'Association non enregistrée. Tu peux réessayer sans créer de doublon.')
    }
  }

  return <form className="reimbursement-allocation-form" onSubmit={submit}>
    <h4 id={title}>Associer à une dépense</h4>
    <p>Crédit encore à répartir : <strong>{transactionMoney(reimbursementCase.unallocated_cents)}</strong></p>
    {expenses === null && !loadError && <p role="status">Chargement des dépenses disponibles…</p>}
    {loadError && <p role="alert" className="notice err">{loadError} <button className="link" type="button" onClick={() => { setLoadError(null); setExpenses(null); api.eligibleReimbursementExpenses(reimbursementCase.id, { limit: 100 }).then(result => setExpenses(result.expenses || [])).catch(reason => setLoadError(reason.message)) }}>Réessayer</button></p>}
    {expenses?.length === 0 && <p>Aucune dépense avec un montant remboursable n’est disponible.</p>}
    {expenses?.length > 0 && <>
      <label htmlFor={`${title}-expense`}>Dépense</label>
      <select id={`${title}-expense`} value={expenseID} disabled={busy} onChange={event => {
        const nextID = event.target.value
        setExpenseID(nextID)
        const next = expenses.find(item => String(item.transaction_id) === nextID)
        setAmount(next ? amountInputFromCents(Math.min(reimbursementCase.unallocated_cents, next.remaining_cents)) : '')
        requestKey.current = null
        setError(null)
      }}>
        <option value="">Choisir une dépense</option>
        {expenses.map(expense => <option key={expense.transaction_id} value={expense.transaction_id}>
          {transactionDate(expense.date)} · {transactionName(expense)} · {transactionMoney(expense.remaining_cents)} encore remboursable
        </option>)}
      </select>
      {selected && <>
        <p className="reimbursement-source-detail">Débit original : {transactionMoney(-selected.amount_cents)} · Reste remboursable : {transactionMoney(selected.remaining_cents)}</p>
        <label htmlFor={`${title}-amount`}>Montant à associer (€)</label>
        <input id={`${title}-amount`} type="text" inputMode="decimal" autoComplete="off" value={amount} disabled={busy}
          aria-describedby={`${title}-amount-help`} onChange={event => { setAmount(event.target.value); requestKey.current = null; setError(null) }}/>
        <small id={`${title}-amount-help`}>Maximum pour cette opération : {transactionMoney(maxCents)}. Saisis les euros avec deux décimales au plus.</small>
      </>}
    </>}
    {(error || draftError) && <p role="alert" className="notice err">{error || draftError}</p>}
    <footer>
      <button type="button" className="btn-secondary" onClick={onCancel} disabled={busy}>Annuler</button>
      <button type="submit" className="btn" disabled={busy || !selected || Boolean(draftError) || expenses === null}>
        {busy ? 'Enregistrement…' : 'Confirmer l’association'}
      </button>
    </footer>
  </form>
}

export default function TransactionReimbursements({ transaction, onRefresh }) {
  const amountCents = transactionCents(transaction)
  const isCredit = amountCents > 0
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)
  const [qualifyKind, setQualifyKind] = useState('')
  const [note, setNote] = useState('')
  const [allocationCase, setAllocationCase] = useState(null)
  const [mutationError, setMutationError] = useState(null)
  const requestEpoch = useRef(0)

  const load = async (notifyParent = false) => {
    const epoch = ++requestEpoch.current
    setLoading(true)
    setError(null)
    try {
      const data = await api.reimbursements({ transaction_id: transaction.id, limit: 100 })
      if (epoch !== requestEpoch.current) return
      setResult(data)
      if (notifyParent) onRefresh?.()
    } catch (reason) {
      if (epoch === requestEpoch.current) setError(reason.message || 'Rapprochements indisponibles.')
    } finally {
      if (epoch === requestEpoch.current) setLoading(false)
    }
  }

  useEffect(() => {
    let active = true
    const epoch = ++requestEpoch.current
    setLoading(true)
    setError(null)
    api.reimbursements({ transaction_id: transaction.id, limit: 100 }).then(data => {
      if (active && epoch === requestEpoch.current) setResult(data)
    }).catch(reason => {
      if (active && epoch === requestEpoch.current) setError(reason.message || 'Rapprochements indisponibles.')
    }).finally(() => {
      if (active && epoch === requestEpoch.current) setLoading(false)
    })
    return () => { active = false }
  }, [transaction.id])

  const cases = result?.cases || []
  const sourceCases = cases.filter(item => item.credit_transaction_id === transaction.id)
  const targetCases = cases.filter(item => item.allocations?.some(allocation => allocation.expense_transaction_id === transaction.id))
  const targetAllocations = targetCases.flatMap(item => item.allocations
    .filter(allocation => allocation.expense_transaction_id === transaction.id)
    .map(allocation => ({ ...allocation, case: item })))
  let personalCost = null
  if (!isCredit && targetAllocations.some(item => !item.voided_at && item.case.status !== 'voided')) {
    try { personalCost = personalCostCents(amountCents, targetCases, transaction.id) } catch { personalCost = null }
  }

  const mutate = async action => {
    setBusy(true)
    setMutationError(null)
    try {
      await action()
      setQualifyKind('')
      setAllocationCase(null)
      await load(true)
      return true
    } catch (reason) {
      setMutationError(reason.message || 'La modification n’a pas été enregistrée.')
      return false
    } finally {
      setBusy(false)
    }
  }

  const qualify = event => {
    event.preventDefault()
    if (!qualifyKind) return
    mutate(() => api.createReimbursement({ credit_transaction_id: transaction.id, kind: qualifyKind, note: note.trim() }))
  }

  const activeReimbursements = targetAllocations.filter(item => !item.voided_at && item.case.status !== 'voided')
  return <section className="transaction-reimbursements" aria-labelledby={`reimbursements-${transaction.id}`}>
    <header><div><span className="eyebrow">Rapprochements</span><h3 id={`reimbursements-${transaction.id}`}>Remboursements associés</h3></div></header>
    <p className="reimbursement-scope-note">Les associations sont conservées ici. Le coût net indicatif n’a pas encore d’effet sur le Dashboard, les budgets ou les analyses.</p>
    {loading && <p role="status">Chargement des rapprochements…</p>}
    {error && <p role="alert" className="notice err">{error} <button className="link" type="button" onClick={() => load()}>Réessayer</button></p>}
    {mutationError && <p role="alert" className="notice err">{mutationError}</p>}

    {isCredit && sourceCases.map(item => <article className="reimbursement-case" key={item.id}>
      <div className="reimbursement-case-heading"><strong>{reimbursementKindLabel(item.kind)}</strong><span className="transaction-status">{reimbursementStatusLabel(item.status)}</span></div>
      <p>Crédit bancaire : <strong>{transactionMoney(item.credit.amount_cents)}</strong></p>
      <p>Affecté : <strong>{transactionMoney(item.allocated_cents)}</strong> · Non affecté : <strong>{transactionMoney(item.unallocated_cents)}</strong></p>
      {item.allocations.length > 0 && <ul className="reimbursement-allocation-list">
        {item.allocations.map(allocation => <li key={allocation.id}>
          <span><strong>{transactionMoney(allocation.amount_cents)}</strong> vers {transactionDate(allocation.expense.date)} · {transactionName(allocation.expense)}
            {allocation.voided_at && <small>Association annulée le {transactionDate(allocation.voided_at.slice(0, 10))} : {allocation.void_reason || 'motif non renseigné'}.</small>}
            {!allocation.expense_transaction_id && <small>Transaction source indisponible; le snapshot documentaire est conservé.</small>}
          </span>
          {!allocation.voided_at && item.status !== 'voided' && <button type="button" className="link" disabled={busy} onClick={() => {
            if (window.confirm('Retirer cette association ? La décision restera dans l’historique.'))
              mutate(() => api.voidReimbursementAllocation(item.id, allocation.id, 'Retrait confirmé depuis le détail de transaction'))
          }}>Retirer</button>}
        </li>)}
      </ul>}
      {allocationCase === item.id && <ReimbursementAllocationForm reimbursementCase={item} busy={busy} onCancel={() => setAllocationCase(null)}
        onSubmit={(allocation, key) => mutate(() => api.createReimbursementAllocation(item.id, allocation, key))}/>}
      {item.status !== 'voided' && <div className="reimbursement-actions">
        {item.unallocated_cents > 0 && allocationCase !== item.id && <button type="button" className="btn-secondary" disabled={busy} onClick={() => { setAllocationCase(item.id); setMutationError(null) }}>Associer à une dépense</button>}
        <button type="button" className="link reimbursement-void" disabled={busy} onClick={() => {
          if (window.confirm('Annuler ce dossier et toutes ses associations actives ? L’historique sera conservé.'))
            mutate(() => api.voidReimbursement(item.id, 'Annulation confirmée depuis le détail de transaction'))
        }}>Annuler le dossier</button>
      </div>}
    </article>)}

    {!isCredit && targetCases.length > 0 && <article className="reimbursement-case">
      <h4>Dépense d’origine</h4>
      {activeReimbursements.length > 0 && <p>Payé : <strong>{transactionMoney(-amountCents)}</strong> · Remboursé : <strong>{transactionMoney(-amountCents - personalCost)}</strong> · Coût personnel net indicatif : <strong>{transactionMoney(personalCost)}</strong></p>}
      <ul className="reimbursement-allocation-list">
        {targetAllocations.map(allocation => <li key={`${allocation.case.id}-${allocation.id}`}>
          <span><strong>{transactionMoney(allocation.amount_cents)}</strong> depuis {transactionDate(allocation.case.credit.date)} · {transactionName(allocation.case.credit)}
            <small>{reimbursementKindLabel(allocation.case.kind)} · {allocation.voided_at || allocation.case.status === 'voided' ? 'Association annulée' : reimbursementStatusLabel(allocation.case.status)}</small>
          </span>
        </li>)}
      </ul>
    </article>}

    {isCredit && !loading && sourceCases.length === 0 && <>
      <p>Ce crédit n’est pas encore qualifié. Aucun remboursement n’est déduit automatiquement.</p>
      {!qualifyKind && <button type="button" className="btn-secondary" disabled={busy || Boolean(error)} onClick={() => setQualifyKind('personal_expense')}>Qualifier comme remboursement</button>}
      {qualifyKind && <form className="reimbursement-qualification-form" onSubmit={qualify}>
        <label htmlFor={`refund-kind-${transaction.id}`}>Nature du remboursement</label>
        <select id={`refund-kind-${transaction.id}`} value={qualifyKind} onChange={event => setQualifyKind(event.target.value)} disabled={busy}>
          <option value="personal_expense">Remboursement d’une dépense personnelle</option>
          <option value="merchant_refund">Retour ou annulation commerciale</option>
        </select>
        <label htmlFor={`refund-note-${transaction.id}`}>Note (facultative)</label>
        <textarea id={`refund-note-${transaction.id}`} value={note} maxLength={2000} onChange={event => setNote(event.target.value)} disabled={busy}/>
        <p>Cette qualification est manuelle. Le crédit bancaire reste inchangé et ses indicateurs ne sont pas encore recalculés par cette version.</p>
        <footer><button type="button" className="btn-secondary" disabled={busy} onClick={() => setQualifyKind('')}>Annuler</button><button className="btn" disabled={busy}>{busy ? 'Enregistrement…' : 'Confirmer la qualification'}</button></footer>
      </form>}
    </>}

    {!isCredit && targetCases.length === 0 && !loading && <p>Aucun remboursement confirmé n’est associé à cette dépense.</p>}
    {result?.total > cases.length && <p className="reimbursement-scope-note">Certains rapprochements supplémentaires sont masqués. Filtre l’historique pour les consulter.</p>}
  </section>
}
