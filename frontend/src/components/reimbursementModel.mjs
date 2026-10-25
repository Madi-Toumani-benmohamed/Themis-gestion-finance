import { cents } from './money.mjs'

const safeSum = values => values.reduce((total, value) => {
  if (!Number.isSafeInteger(value) || value < 0 || !Number.isSafeInteger(total + value))
    throw new Error('Total de remboursement hors limites.')
  return total + value
}, 0)

export function activeAllocatedCents(reimbursementCase, expenseTransactionId) {
  const active = (reimbursementCase?.allocations || []).filter(allocation =>
    !allocation.voided_at && reimbursementCase.status !== 'voided' &&
    (expenseTransactionId === undefined || allocation.expense_transaction_id === expenseTransactionId))
  return safeSum(active.map(allocation => allocation.amount_cents))
}

export function personalCostCents(transactionAmountCents, cases, transactionId) {
  if (!Number.isSafeInteger(transactionAmountCents) || transactionAmountCents >= 0)
    throw new Error('Une dépense bancaire négative est requise.')
  const refunded = safeSum((cases || []).flatMap(item =>
    (item.allocations || [])
      .filter(allocation => !allocation.voided_at && item.status !== 'voided' && allocation.expense_transaction_id === transactionId)
      .map(allocation => allocation.amount_cents)))
  const gross = -transactionAmountCents
  if (refunded > gross) throw new Error('Les remboursements dépassent la dépense source.')
  return gross - refunded
}

export function amountInputFromCents(value) {
  if (!Number.isSafeInteger(value) || value < 0) throw new Error('Montant de remboursement invalide.')
  return `${Math.trunc(value / 100)}.${String(value % 100).padStart(2, '0')}`
}

export function allocationRequest(expenseTransactionId, euroAmount, creditRemainingCents, expenseRemainingCents) {
  if (!Number.isSafeInteger(expenseTransactionId) || expenseTransactionId <= 0)
    throw new Error('Sélectionne une dépense.')
  if (!Number.isSafeInteger(creditRemainingCents) || creditRemainingCents <= 0 ||
      !Number.isSafeInteger(expenseRemainingCents) || expenseRemainingCents <= 0)
    throw new Error('Aucun montant disponible pour cette association.')
  const normalized = String(euroAmount).trim().replace(',', '.')
  const amountCents = cents(normalized)
  if (amountCents <= 0) throw new Error('Saisis un montant positif au centime.')
  if (amountCents > creditRemainingCents || amountCents > expenseRemainingCents)
    throw new Error('Le montant dépasse le crédit disponible ou le reste remboursable de la dépense.')
  return { expense_transaction_id: expenseTransactionId, amount_cents: amountCents }
}

export function reimbursementStatusLabel(status) {
  return ({
    unmatched: 'À rapprocher',
    partial: 'Partiellement rapproché',
    allocated: 'Rapproché',
    needs_review: 'À examiner',
    voided: 'Annulé',
  })[status] || 'État inconnu'
}

export function reimbursementKindLabel(kind) {
  return kind === 'merchant_refund' ? 'Retour ou annulation commerciale' : 'Remboursement de dépense personnelle'
}
