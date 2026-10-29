import { cents } from './money.mjs'

const optionalCents = value => value == null ? null : cents(value)
export function dashboardModel(monthly, month) {
  const row = monthly?.find(item => item.month === month)
  if (!row) return null
  const [year, number] = month.split('-').map(Number)
  const previousMonth = new Date(Date.UTC(year, number - 2, 1)).toISOString().slice(0, 7)
  const previous = monthly.find(item => item.month === previousMonth)
  const complete = item => item?.status === 'complete' && !item.is_current
  const comparable = complete(row) && complete(previous)
  const income = optionalCents(row.income), spent = optionalCents(row.expenses)
  const net = income == null || spent == null ? null : income - spent
  if (net != null && !Number.isSafeInteger(net)) throw new Error('Résultat hors limites.')
  const delta = (current, before) => {
    if (!comparable || current == null || before == null) return null
    const prior = cents(before)
    return prior === 0 ? null : Math.round(((current - prior) / Math.abs(prior)) * 100)
  }
  const days = Number.isInteger(row.days_observed) && Number.isInteger(row.days_in_month)
    ? ` — ${row.days_observed}/${row.days_in_month} jours` : ''
  const isCurrent = row.is_current || row.status === 'current'
  return {
    income: income == null ? null : income / 100,
    spent: spent == null ? null : spent / 100,
    net: net == null ? null : net / 100,
    savings: income > 0 && net != null ? (net / income) * 100 : null,
    isCurrent, periodStatus: row.status,
    periodLabel: isCurrent ? `Mois en cours${days}` : !complete(row) ? `Données incomplètes${days}` : null,
    incomeDelta: delta(income, previous?.income), spentDelta: delta(spent, previous?.expenses),
  }
}
