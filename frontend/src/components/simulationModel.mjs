// Exact cent arithmetic shared by every number displayed by the simulator.
// Percentages use basis points; division rounds halves away from zero.
export function cents(value) {
  const text = String(value)
  if (!/^-?\d+(?:\.\d{1,2})?$/.test(text))
    throw new Error('Montant attendu au centime près.')
  const [units, fraction = ''] = text.replace('-', '').split('.')
  const amount =
    Number(BigInt(units) * 100n + BigInt((fraction + '00').slice(0, 2))) *
    (text.startsWith('-') ? -1 : 1)
  if (!Number.isSafeInteger(amount) || Math.abs(amount) > 999999999999)
    throw new Error('Montant hors limites.')
  return amount
}
const rounded = (n, d) => Number((n + d / 2n) / d)
const positive = (value) => {
  const n = cents(value)
  if (n < 0)
    throw new Error('Les montants saisis doivent être positifs ou nuls.')
  return n
}
function percent(value) {
  const n = positive(value === '' ? 0 : value)
  if (n > 10000)
    throw new Error('Le pourcentage doit être compris entre 0 et 100.')
  return n
}
export function emptyScenario() {
  return {
    income: '',
    cuts: {},
    expenses: [{ id: 0, label: '', amount: '' }],
    loan: { amount: '', rate: '', months: '' },
  }
}
export function loanEstimate(loan) {
  if (!Object.values(loan).some((v) => v !== ''))
    return { payment: 0, total: 0, interest: 0 }
  if (loan.amount === '' || loan.months === '')
    throw new Error('Complète le montant et la durée de l’emprunt.')
  const principal = BigInt(positive(loan.amount)),
    months = Number(loan.months)
  if (!Number.isInteger(months) || months < 1 || months > 600)
    throw new Error('La durée doit être comprise entre 1 et 600 mois.')
  const rate = BigInt(percent(loan.rate))
  let payment
  if (rate === 0n) payment = rounded(principal, BigInt(months))
  else {
    const denominator = 120000n,
      numerator = denominator + rate
    const powerN = numerator ** BigInt(months),
      powerD = denominator ** BigInt(months)
    payment = rounded(
      principal * rate * powerN,
      denominator * (powerN - powerD),
    )
  }
  const total =
    rate === 0n
      ? Number(principal)
      : Math.max(payment * months, Number(principal))
  return { payment, total, interest: total - Number(principal) }
}
export function simulationModel(reference, scenario) {
  if (!reference || reference.available === false) return null
  const observedIncome = cents(reference.observed_income ?? reference.income)
  if (!Array.isArray(reference.categories))
    throw new Error('Référence de catégories indisponible.')
  const names = new Set()
  const categories = reference.categories.map((c) => {
    if (
      typeof c.category !== 'string' ||
      names.has(c.category) ||
      cents(c.before) < 0
    )
      throw new Error('Catégories de référence incohérentes.')
    names.add(c.category)
    const before = cents(c.before),
      pct = percent(scenario.cuts[c.category] ?? 0)
    const saved = rounded(BigInt(before) * BigInt(pct), 10000n)
    return {
      name: c.category,
      before,
      pct: pct / 100,
      saved,
      after: before - saved,
    }
  })
  const baseSpend = categories.reduce((sum, c) => sum + c.before, 0)
  // Never repair an incoherent API baseline silently.
  if (
    baseSpend !== cents(reference.base_spend) ||
    observedIncome - baseSpend !== cents(reference.current_net)
  )
    throw new Error(
      'La référence reçue présente un écart de réconciliation. Recharge la page après vérification des données.',
    )
  const income =
    scenario.income === '' ? observedIncome : positive(scenario.income)
  const saved = categories.reduce((sum, c) => sum + c.saved, 0)
  const added = scenario.expenses.reduce((sum, e) => {
    if (e.label.trim() === '' && e.amount === '') return sum
    if (!e.label.trim() || e.amount === '')
      throw new Error(
        'Chaque nouvelle dépense doit avoir un intitulé et un montant.',
      )
    return sum + positive(e.amount)
  }, 0)
  const loan = loanEstimate(scenario.loan)
  const spend = baseSpend - saved + added + loan.payment
  const currentNet = observedIncome - baseSpend,
    simulatedNet = income - spend
  return {
    categories,
    observedIncome,
    income,
    baseSpend,
    saved,
    added,
    loan,
    spend,
    currentNet,
    simulatedNet,
    delta: simulatedNet - currentNet,
    modified: categories.filter((c) => c.pct > 0).length,
  }
}
export const euroCents = (value) =>
  ((value === 0 ? 0 : value) / 100).toLocaleString('fr-FR', {
    style: 'currency',
    currency: 'EUR',
  })
export const signedEuroCents = (value) =>
  (value > 0 ? '+' : '') + euroCents(value)
