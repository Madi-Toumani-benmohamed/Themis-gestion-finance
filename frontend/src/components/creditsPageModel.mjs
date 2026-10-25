import { cents } from './money.mjs'

// Backend integer cents are authoritative. Missing information remains missing.
export function amountCents(row, field) {
  if (row?.amounts_cents && Object.hasOwn(row.amounts_cents, field)) {
    const n = row.amounts_cents[field]
    return Number.isSafeInteger(n) && n >= 0 ? n : null
  }
  if (row?.[field] == null) return null
  try {
    const n = cents(row[field])
    return n >= 0 ? n : null
  } catch {
    return null
  }
}
export function formatEuro(value) {
  if (!Number.isSafeInteger(value) || value < 0) return 'Non documenté'
  const n = BigInt(value)
  return `${new Intl.NumberFormat('fr-FR').format(n / 100n)},${String(n % 100n).padStart(2, '0')}\u00a0€`
}
export const moneyText = (row, field) => formatEuro(amountCents(row, field))
export function decimalInput(value, positive = true) {
  const normalized = String(value).trim().replace(',', '.')
  const n = cents(normalized)
  if (n < 0 || (positive && n === 0))
    throw new Error('Saisir un montant positif avec au maximum deux décimales.')
  return `${BigInt(n) / 100n}.${String(BigInt(n) % 100n).padStart(2, '0')}`
}
export function inputAmount(row, field) {
  const n = amountCents(row, field)
  return n == null
    ? ''
    : `${BigInt(n) / 100n}.${String(BigInt(n) % 100n).padStart(2, '0')}`
}
export function formatMonth(value) {
  if (!/^\d{4}-(0[1-9]|1[0-2])$/.test(value || '')) return 'Non documenté'
  return new Intl.DateTimeFormat('fr-FR', {
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  }).format(new Date(`${value}-01T12:00:00Z`))
}
export function formatDate(value) {
  if (!value) return 'Non documenté'
  const date = new Date(value.length === 10 ? `${value}T12:00:00Z` : value)
  return Number.isFinite(date.getTime())
    ? date.toLocaleDateString('fr-FR', { timeZone: 'UTC' })
    : 'Non documenté'
}
export function calculateProgress(credit) {
  if (
    !credit?.payment_history_complete ||
    credit.status === 'invalid' ||
    credit.progress_pct == null
  )
    return null
  const n = Number(credit.progress_pct)
  return Number.isFinite(n) && n >= 0 && n <= 100 ? n : null
}
export const isCreditCompleted = (credit) =>
  credit?.payment_history_complete &&
  amountCents(credit, 'remaining') === 0 &&
  credit.status !== 'invalid'
export const statusLabel = (status) =>
  ({
    unknown: 'Historique à compléter',
    invalid: 'Données à vérifier',
    active: 'En cours',
    future: 'À venir',
    paid: 'Payé — déclaration',
    closed: 'Clôturé',
    archived: 'Archivé',
    partial: 'Partiellement payé',
    to_verify: 'Paiement à vérifier',
    today: 'Échéance du jour',
    upcoming: 'À venir',
    late_confirmed: 'Retard confirmé',
  })[status] || 'Statut non documenté'
export const getTaxStatus = (tax) => tax?.status || 'unknown'
function compareNullable(a, b, descending = false) {
  if (a == null) return b == null ? 0 : 1
  if (b == null) return -1
  return (a < b ? -1 : a > b ? 1 : 0) * (descending ? -1 : 1)
}
export function sortCredits(rows, sort = 'remaining') {
  return [...(rows || [])].sort((a, b) =>
    sort === 'name'
      ? a.name.localeCompare(b.name, 'fr')
      : sort === 'end'
        ? compareNullable(a.end_month || null, b.end_month || null)
        : compareNullable(
            sort === 'progress'
              ? calculateProgress(a)
              : amountCents(a, 'remaining'),
            sort === 'progress'
              ? calculateProgress(b)
              : amountCents(b, 'remaining'),
            true,
          ),
  )
}
export function sortTaxes(rows, sort = 'due') {
  return [...(rows || [])].sort((a, b) =>
    sort === 'name'
      ? a.name.localeCompare(b.name, 'fr')
      : sort === 'amount'
        ? compareNullable(
            amountCents(a, 'amount'),
            amountCents(b, 'amount'),
            true,
          )
        : Number(a.paid) - Number(b.paid) ||
          a.due_date.localeCompare(b.due_date),
  )
}
export function creditPayload(form) {
  const months = Number(form.months)
  if (!Number.isInteger(months) || months < 1 || months > 1200)
    throw new Error('La durée doit être comprise entre 1 et 1 200 mois.')
  if (
    !form.name.trim() ||
    !/^\d{4}-(0[1-9]|1[0-2])$/.test(form.start) ||
    form.start.startsWith('0000')
  )
    throw new Error('Nom et mois de début valides requis.')
  return {
    name: form.name.trim(),
    monthly_amount: decimalInput(form.monthly),
    months_total: months,
    start_month: form.start,
    note: form.note.trim(),
    opening_paid:
      form.opening === '' ? null : decimalInput(form.opening, false),
  }
}
