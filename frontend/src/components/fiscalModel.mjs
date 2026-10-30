export function fiscalResult(data, year) {
  const fail = () => {
    throw new Error(
      'Réponse fiscale incomplète ou incohérente : aucun résultat affiché.',
    )
  }
  if (
    !data ||
    data.rules?.income_year !== year ||
    data.rules.assessment_year !== year + 1 ||
    data.rules.status !== 'official' ||
    typeof data.rules.version !== 'string' ||
    !Array.isArray(data.rules.sources) ||
    data.rules.sources.length === 0 ||
    !data.rules.sources.every((s) =>
      /^https:\/\/(www\.|simulateur-ir-ifi\.)?impots\.gouv\.fr\//.test(s),
    )
  )
    fail()
  for (const key of [
    'annual_income_cents',
    'salary_deduction_cents',
    'taxable_cents',
    'gross_tax_cents',
    'discount_cents',
    'tax_cents',
  ])
    if (!Number.isSafeInteger(data[key]) || data[key] < 0) fail()
  for (const key of [
    'observed_paid_cents',
    'projected_future_paid_cents',
    'gap_before_future_cents',
    'settlement_cents',
  ])
    if (data[key] !== null && !Number.isSafeInteger(data[key])) fail()
  if (
    data.annual_income_cents - data.salary_deduction_cents !==
      data.taxable_cents ||
    data.discount_cents > data.gross_tax_cents ||
    data.tax_cents !==
      (data.gross_tax_cents - data.discount_cents < 6100
        ? 0
        : data.gross_tax_cents - data.discount_cents)
  )
    fail()
  if (
    !Number.isInteger(data.half_parts) ||
    data.half_parts < 2 ||
    data.half_parts > 24 ||
    !['unknown', 'incomplete', 'projected', 'annual_estimate'].includes(
      data.settlement_status,
    ) ||
    !['0', '11', '30', '41', '45'].includes(String(data.marginal_rate_pct)) ||
    !/^\d+\.\d{2}$/.test(data.average_rate_pct)
  )
    fail()
  if (data.observed_paid_cents === null) {
    if (
      data.gap_before_future_cents !== null ||
      data.settlement_cents !== null ||
      data.projected_future_paid_cents !== null
    )
      fail()
  } else {
    if (
      data.observed_paid_cents < 0 ||
      data.gap_before_future_cents !== data.tax_cents - data.observed_paid_cents
    )
      fail()
    if (
      data.projected_future_paid_cents !== null &&
      data.projected_future_paid_cents < 0
    )
      fail()
    if (
      data.settlement_cents !== null &&
      data.settlement_cents !==
        data.gap_before_future_cents - (data.projected_future_paid_cents ?? 0)
    )
      fail()
  }
  if (
    !Array.isArray(data.steps) ||
    !Array.isArray(data.assumptions) ||
    !Array.isArray(data.limitations) ||
    data.steps.some(
      (s) =>
        typeof s.label !== 'string' || !Number.isSafeInteger(s.amount_cents),
    )
  )
    fail()
  return data
}
