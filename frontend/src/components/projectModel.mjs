import { cents, euroCents } from './money.mjs'
export const projectMoney = (value) =>
  Number.isSafeInteger(value) ? euroCents(value) : 'Non disponible'
export const projectStatus = {
  active: 'Actif',
  completed: 'Terminé',
  archived: 'Archivé',
}
export function projectBudgetInput(value) {
  if (value === null || value === undefined) return ''
  if (!Number.isSafeInteger(value) || value < 0)
    throw new Error('Budget invalide.')
  return `${Math.trunc(value / 100)}.${String(value % 100).padStart(2, '0')}`
}
export function projectBudget(value) {
  if (!String(value).trim()) return null
  const result = cents(String(value).trim().replace(',', '.'))
  if (result < 0 || result > 999999999999) throw new Error('Budget invalide.')
  return result
}
export function projectCumulative(rows) {
  let total = 0
  return rows.map((row) => {
    if (
      !Number.isSafeInteger(row.amount_cents) ||
      row.amount_cents < 0 ||
      !Number.isSafeInteger(total + row.amount_cents)
    )
      throw new Error('Cumul hors limites exactes.')
    total += row.amount_cents
    return { ...row, cumulative_cents: total }
  })
}
export function projectReason(reason) {
  if (reason?.kind === 'manual') return 'Association manuelle'
  const kinds = {
    tag: 'Tag sélectionné',
    category: 'Catégorie sélectionnée',
    merchant: 'Marchand exact',
    label: 'Texte présent dans le libellé',
  }
  return `${kinds[reason?.kind] || 'Règle enregistrée'}${reason?.match_text ? ` : ${reason.match_text}` : ''}${reason?.tag_id ? ` ${reason.tag_name || `nº${reason.tag_id}`}` : ''}${reason?.domain_id ? ` ${reason.domain_name || `nº${reason.domain_id}`}` : ''}${reason?.from_date ? ` · ${reason.from_date} → ${reason.to_date}` : ''}`
}

export function projectListModel(data) {
  if (!data || !Array.isArray(data.projects))
    throw new Error('Liste de projets incomplète.')
  for (const p of data.projects) {
    if (
      !p ||
      !Number.isSafeInteger(p.id) ||
      p.id <= 0 ||
      !Number.isSafeInteger(p.cost_cents) ||
      p.cost_cents < 0 ||
      !Number.isSafeInteger(p.transaction_count) ||
      p.transaction_count < 0 ||
      !Number.isSafeInteger(p.suggestion_count) ||
      p.suggestion_count < 0 ||
      (p.budget_cents !== null &&
        (!Number.isSafeInteger(p.budget_cents) || p.budget_cents < 0))
    )
      throw new Error('Données financières du projet invalides.')
    if (
      p.budget_cents === null
        ? p.remaining_cents !== null ||
          p.overrun_cents !== null ||
          p.consumed_pct !== null
        : p.remaining_cents !== p.budget_cents - p.cost_cents ||
          p.overrun_cents !== Math.max(p.cost_cents - p.budget_cents, 0)
    )
      throw new Error('Les indicateurs du projet ne se réconcilient pas.')
  }
  return data
}
export function projectReportModel(data) {
  projectListModel({ projects: [data?.project] })
  for (const field of [
    'by_category',
    'by_merchant',
    'by_month',
    'by_tag',
    'observations',
  ])
    if (!Array.isArray(data[field]))
      throw new Error('Rapport du projet incomplet.')
  for (const rows of [data.by_category, data.by_merchant, data.by_month]) {
    const sum = projectCumulative(rows).at(-1)?.cumulative_cents || 0
    if (sum !== data.project.cost_cents)
      throw new Error('La répartition des coûts ne se réconcilie pas.')
  }
  return data
}
