import { cents, euroCents } from './money.mjs'

export const transactionMoney = euroCents
export const transactionCents = transaction => {
  const exact = cents(transaction.amount)
  if (transaction.amount_cents !== undefined && (!Number.isSafeInteger(transaction.amount_cents) || transaction.amount_cents !== exact))
    throw new Error('Le montant transmis est incohérent avec sa valeur en centimes.')
  return exact
}
export const transactionTags = raw => [...new Set(raw.split(',').map(tag => tag.trim().toLowerCase()).filter(Boolean))]
export function transactionReportModel(report) {
  if (!report || !Array.isArray(report.transactions) || !Number.isSafeInteger(report.total) || report.total < 0 || !report.summary)
    throw new Error('La réponse des transactions est incomplète.')
  const summary = Object.fromEntries(['expenses', 'income', 'net', 'transfer_debits', 'transfer_credits'].map(key => [key, cents(report.summary[key])]))
  if (summary.income - summary.expenses !== summary.net)
    throw new Error('Les totaux transmis ne se réconcilient pas au centime près.')
  if (report.total < report.transactions.length || ['expenses', 'income', 'transfer_debits', 'transfer_credits'].some(key => summary[key] < 0))
    throw new Error('Le périmètre ou les signes des agrégats sont incohérents.')
  const ids = new Set()
  report.transactions.forEach(transaction => {
    transactionCents(transaction)
    if (!Number.isSafeInteger(transaction.id) || transaction.id <= 0 || ids.has(transaction.id))
      throw new Error('Identifiant de transaction invalide ou répété dans la réponse.')
    ids.add(transaction.id)
  })
  return { ...report, money: summary }
}
export function transactionDayTotal(rows) {
  return rows.reduce((sum, transaction) => {
    const result = sum + transactionCents(transaction)
    if (!Number.isSafeInteger(result)) throw new Error('Total monétaire hors limites.')
    return result
  }, 0)
}
export const transactionTypeLabel = type => ({ debit: 'Débit', credit: 'Crédit', transfer: 'Transfert / épargne', neutral: 'Montant nul' }[type] || 'Nature non documentée')
export const transactionDate = value => value ? new Date(`${value}T12:00:00Z`).toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' }) : 'Non renseignée'
