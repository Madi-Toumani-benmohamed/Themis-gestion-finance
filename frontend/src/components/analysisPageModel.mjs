import { cents, euroCents } from './money.mjs'

export const analysisMoney = (value) => euroCents(cents(value))
export const analysisCents = cents
export const analysisEuro = euroCents
export const analysisDate = (value) =>
  value
    ? new Date(`${value.slice(0, 10)}T12:00:00Z`).toLocaleDateString('fr-FR', {
        day: 'numeric',
        month: 'short',
        year: 'numeric',
        timeZone: 'UTC',
      })
    : 'Non documentée'
export const analysisPercent = (value) =>
  value == null || !Number.isFinite(value)
    ? 'Non établie'
    : `${value.toLocaleString('fr-FR', { maximumFractionDigits: 1 })} %`
export const analysisMonth = (offset = 0, reference) => {
  const value = reference || new Date().toISOString().slice(0, 7)
  if (!/^\d{4}-\d{2}$/.test(value)) throw new Error('Mois invalide.')
  const date = new Date(`${value}-01T12:00:00Z`)
  if (!Number.isFinite(+date) || date.toISOString().slice(0, 7) !== value)
    throw new Error('Mois invalide.')
  date.setUTCMonth(date.getUTCMonth() + offset)
  return date.toISOString().slice(0, 7)
}
export const analysisWindow = (month, months = 6, through) => {
  const selected = analysisMonth(0, month)
  const current = analysisMonth()
  const end = new Date(`${selected}-01T12:00:00Z`)
  end.setUTCMonth(end.getUTCMonth() + 1, 0)
  return {
    from: `${analysisMonth(1 - months, selected)}-01`,
    to:
      through ||
      (selected === current
        ? new Date().toISOString().slice(0, 10)
        : end.toISOString().slice(0, 10)),
  }
}
export const analysisPeriodLabel = (period) => {
  if (period?.is_current)
    return period.completeness === 'unknown'
      ? 'Mois en cours · couverture inconnue'
      : 'Mois en cours · données partielles'
  return (
    {
      complete: 'Historique documenté',
      partial: 'Historique partiellement couvert',
      unknown: 'Couverture non établie',
    }[period?.completeness] || 'Couverture non établie'
  )
}
const checkedSum = (values) =>
  values.reduce((total, value) => {
    const next = total + value
    if (!Number.isSafeInteger(next))
      throw new Error('Total monétaire hors limites.')
    return next
  }, 0)
export function analysisNatureModel(report, expectedMonth) {
  if (
    !report ||
    typeof report.available !== 'boolean' ||
    report.month !== expectedMonth
  )
    throw new Error(
      'La réponse analytique ne correspond pas au mois sélectionné.',
    )
  if (
    !Number.isInteger(report.transaction_count) ||
    report.transaction_count < 0
  )
    throw new Error('Le nombre de sources est invalide.')
  if (!report.available) return { ...report, money: null, categories: [] }
  if (!Array.isArray(report.categories))
    throw new Error('La répartition des dépenses est absente.')
  const money = Object.fromEntries(
    [
      'income',
      'expenses',
      'net',
      'real_spend',
      'mandatory',
      'savings',
      'transfer_out',
      'transfer_in',
      'total_out',
      'uncategorized_spent',
    ].map((key) => [key, cents(report[key])]),
  )
  if (Object.entries(money).some(([key, value]) => key !== 'net' && value < 0))
    throw new Error('Les signes du rapport sont incohérents.')
  if (
    money.income - money.expenses !== money.net ||
    money.real_spend + money.mandatory !== money.expenses ||
    checkedSum([money.expenses, money.savings, money.transfer_out]) !==
      money.total_out
  )
    throw new Error('Les flux ne se réconcilient pas au centime près.')
  const keys = new Set()
  const categories = report.categories.map((category) => {
    const key = category.id ?? 'none',
      spentCents = cents(category.spent)
    if (
      keys.has(key) ||
      spentCents < 0 ||
      (category.id != null &&
        (!Number.isSafeInteger(category.id) || category.id <= 0))
    )
      throw new Error('Une catégorie est invalide ou comptée plusieurs fois.')
    keys.add(key)
    if (
      category.share_pct != null &&
      (!Number.isFinite(category.share_pct) ||
        category.share_pct < 0 ||
        category.share_pct > 100)
    )
      throw new Error('La part d’une catégorie est incohérente.')
    return { ...category, spentCents }
  })
  if (
    checkedSum(categories.map((category) => category.spentCents)) !==
    money.expenses
  )
    throw new Error(
      'Le total par catégorie ne correspond pas aux dépenses sources.',
    )
  return { ...report, money, categories }
}
export function analysisSource(source, fallback = {}) {
  const result = {}
  for (const key of [
    'month',
    'from',
    'to',
    'domain',
    'tag',
    'type',
    'budget_id',
    'q',
    'merchant',
  ]) {
    const value = source?.[key] ?? fallback[key]
    if (value !== undefined && value !== null && value !== '')
      result[key] = value
  }
  if (
    Array.isArray(source?.ids) &&
    source.ids.length &&
    source.ids.every((id) => Number.isSafeInteger(id) && id > 0)
  )
    result.ids = source.ids
  return result
}
