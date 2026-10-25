import {
  analysisMonth,
  analysisWindow,
  analysisCents,
} from './analysisPageModel.mjs'

export const TAG_HISTORY_DURATIONS = [
  [6, '6 mois'],
  [12, '1 an'],
  [24, '2 ans'],
  [36, '3 ans'],
  [60, '5 ans'],
  [120, '10 ans'],
]
const monthIndex = (value) =>
  Number(value.slice(0, 4)) * 12 + Number(value.slice(5, 7)) - 1
export const tagHistoryMonthLabel = (value) =>
  new Date(`${value}-01T12:00:00Z`).toLocaleDateString('fr-FR', {
    month: 'short',
    year: '2-digit',
    timeZone: 'UTC',
  })

export function tagHistoryWindow(anchor, duration = 6, from, to) {
  const month = analysisMonth(0, anchor)
  if (month > analysisMonth() || month < '0001-01')
    throw new Error('Mois de référence invalide.')
  if (duration === 'custom') {
    if (!from || !to) throw new Error('Choisis les deux bornes de la période.')
    from = analysisMonth(0, from)
    to = analysisMonth(0, to)
    if (from > to)
      throw new Error('Le début doit précéder la fin de la période.')
    if (to > month)
      throw new Error(
        'La période doit se terminer au plus tard au mois de référence.',
      )
    const months = monthIndex(to) - monthIndex(from) + 1
    if (from < '0001-01' || months > 600)
      throw new Error('Choisis une période de 50 ans maximum.')
    const window = { from: `${from}-01`, to: analysisWindow(to, 1).to }
    return { ...window, months, params: window }
  }
  if (!Number.isInteger(duration) || duration < 1 || duration > 600)
    throw new Error('Durée invalide.')
  const window = analysisWindow(month, duration)
  if (window.from < '0001-01-01') throw new Error('Début de période invalide.')
  return { ...window, months: duration, params: { month, months: duration } }
}

// Safe, internal data keys keep tags such as "period", "a.b" or "__proto__"
// separate from chart metadata. Values remain source amounts; no interpolation.
export function tagHistoryModel(data, window) {
  if (!data || !Array.isArray(data.buckets) || !Array.isArray(data.series))
    throw new Error('Historique des tags invalide.')
  if (data.query_from !== window.from || data.query_to !== window.to)
    throw new Error(
      'L’historique reçu ne correspond pas à la période demandée.',
    )
  const calendar = []
  for (
    let m = window.from.slice(0, 7);
    m <= window.to.slice(0, 7);
    m = analysisMonth(1, m)
  )
    calendar.push(m)
  if (calendar.length > 600) throw new Error('Historique trop long.')
  const seen = new Set(),
    position = new Map()
  data.buckets.forEach((bucket, i) => {
    analysisMonth(0, bucket)
    if (
      bucket < calendar[0] ||
      bucket > calendar.at(-1) ||
      seen.has(bucket) ||
      (i > 0 && bucket <= data.buckets[i - 1])
    )
      throw new Error('Intervalles historiques incohérents.')
    seen.add(bucket)
    position.set(bucket, i)
  })
  const tagNames = new Set()
  const series = data.series.map((s, i) => {
    if (
      typeof s.tag !== 'string' ||
      !s.tag ||
      tagNames.has(s.tag) ||
      !Array.isArray(s.values) ||
      s.values.length !== data.buckets.length
    )
      throw new Error('Série de tag invalide.')
    tagNames.add(s.tag)
    s.values.forEach((value) => {
      if (value != null && analysisCents(value) < 0)
        throw new Error('Dépense historique négative.')
    })
    return { tag: s.tag, key: `tag_${i}`, values: s.values }
  })
  const points = calendar.map((period) => {
    const index = position.get(period)
    return Object.fromEntries([
      ['period', period],
      ...series.map((s) => [
        s.key,
        index === undefined || s.values[index] == null
          ? null
          : Number(s.values[index]),
      ]),
    ])
  })
  series.forEach((s) =>
    points.forEach((point, i) => {
      point[`${s.key}_isolated`] =
        point[s.key] != null &&
        points[i - 1]?.[s.key] == null &&
        points[i + 1]?.[s.key] == null
    }),
  )
  return {
    points,
    series,
    hasObservations: points.some((p) => series.some((s) => p[s.key] != null)),
  }
}
