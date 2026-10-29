// Add explicit null rows for omitted calendar months. This is display geometry
// only: it keeps Recharts from drawing a line across a period with no record.
export function withCalendarGaps(monthly = []) {
  const rows = new Map(monthly.map(row => [row.month, row]))
  const months = [...rows.keys()].filter(value => /^\d{4}-\d{2}$/.test(value)).sort()
  if (months.length < 2) return months.map(month => rows.get(month))

  const [startYear, startMonth] = months[0].split('-').map(Number)
  const [endYear, endMonth] = months.at(-1).split('-').map(Number)
  const first = startYear * 12 + startMonth - 1
  const last = endYear * 12 + endMonth - 1
  const result = []
  for (let index = first; index <= last; index++) {
    const year = Math.floor(index / 12)
    const month = index % 12 + 1
    const key = `${year}-${String(month).padStart(2, '0')}`
    result.push(rows.get(key) || { month: key, income: null, expenses: null, net: null, status: 'missing', isMissing: true })
  }
  return result
}
