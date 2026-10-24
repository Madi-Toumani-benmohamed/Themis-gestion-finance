// Presentation contract for budgets: all monetary operations use integer cents.
import { cents, euroCents } from './money.mjs'
export const budgetCents = cents
export const budgetEuro = euroCents
export const budgetPercent = (value) =>
  value == null || !Number.isFinite(value)
    ? 'Non établie'
    : `${(value * 100).toLocaleString('fr-FR', { maximumFractionDigits: 1 })} %`
export const budgetDate = (value) =>
  value
    ? new Date(`${value.slice(0, 10)}T12:00:00Z`).toLocaleDateString('fr-FR', {
        day: 'numeric',
        month: 'short',
        year: 'numeric',
        timeZone: 'UTC',
      })
    : 'Non déterminée'
export function budgetMonth(offset = 0, from) {
  const date = from ? new Date(`${from}-01T12:00:00Z`) : new Date()
  date.setUTCDate(1)
  date.setUTCMonth(date.getUTCMonth() + offset)
  return date.toISOString().slice(0, 7)
}
export function budgetEnvelope(row) {
  const limit = budgetCents(row.monthly_limit),
    spent = budgetCents(row.spent)
  const remaining = limit - spent
  if (limit <= 0 || spent < 0)
    throw new Error('Enveloppe budgétaire incohérente.')
  if (row.remaining != null && budgetCents(row.remaining) !== remaining)
    throw new Error(
      'Le montant restant ne correspond pas aux opérations sources.',
    )
  return {
    ...row,
    limit,
    spentCents: spent,
    remainingCents: remaining,
    creditCents: budgetCents(row.credit_total ?? 0),
    debitCents: budgetCents(row.debit_total ?? row.spent),
    ratio: spent / limit,
    width: Math.min(100, (spent * 100) / limit),
    status: spent > limit ? 'over' : spent === limit ? 'reached' : 'available',
  }
}
export function budgetPageModel(report) {
  if (!report || !Array.isArray(report.budgets) || !report.summary)
    throw new Error('Le rapport budgétaire est incomplet.')
  const rows = report.budgets.map(budgetEnvelope)
  const expenses = rows.filter((b) => b.scope !== 'transfer'),
    transfers = rows.filter((b) => b.scope === 'transfer')
  const allocated = expenses.reduce((n, b) => n + b.limit, 0),
    spent = expenses.reduce((n, b) => n + b.spentCents, 0)
  const remaining = allocated - spent,
    overCount = expenses.filter((b) => b.status === 'over').length
  for (const [key, value] of Object.entries({ allocated, spent, remaining }))
    if (budgetCents(report.summary[key]) !== value)
      throw new Error(
        'Les totaux reçus ne se réconcilient pas avec les enveloppes.',
      )
  if (report.summary.over_count !== overCount)
    throw new Error('Le nombre de dépassements est incohérent.')
  return {
    expenses,
    transfers,
    allocated,
    spent,
    remaining,
    overCount,
    ratio: allocated ? spent / allocated : null,
    coverageRatio: report.summary.coverage_ratio,
    unclassified: budgetCents(report.summary.uncategorized_spent ?? 0),
    unclassifiedCount: report.summary.uncategorized_count ?? 0,
    observedExpenses: budgetCents(report.summary.observed_expenses ?? 0),
    credits: budgetCents(report.summary.credits_observed ?? 0),
    complete: report.period?.completeness === 'complete',
    current: report.period?.is_current === true,
  }
}
export const recurringNatures = {
  subscription: 'Abonnements',
  fixed_charge: 'Charges fixes récurrentes',
  variable_recurring: 'Dépenses variables récurrentes',
  transfer_savings: 'Transferts et épargne',
}
export const recurringNatureLabel = (nature) =>
  recurringNatures[nature] || 'Nature à vérifier'
export const recurringStatusLabel = (status) =>
  ({
    confirmed: 'Confirmée par vous',
    ignored: 'Ignorée',
    detected: 'Détection à vérifier',
  })[status] || 'À vérifier'
export const recurringFrequencyLabel = (frequency) =>
  ({
    monthly: 'Mensuelle probable',
    weekly: 'Hebdomadaire probable',
    quarterly: 'Trimestrielle probable',
    annual: 'Annuelle probable',
    yearly: 'Annuelle probable',
  })[frequency] ||
  (frequency ? `${frequency} · estimation` : 'Fréquence non établie')
