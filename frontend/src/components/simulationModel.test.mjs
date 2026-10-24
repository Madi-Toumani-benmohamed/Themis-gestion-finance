import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  cents,
  emptyScenario,
  simulationModel,
  loanEstimate,
  euroCents,
} from './simulationModel.mjs'
const base = {
  income: 3249.47,
  observed_income: 3249.47,
  base_spend: 4197.14,
  current_net: -947.67,
  categories: [
    { category: 'Logement', before: 3000 },
    { category: 'Courses', before: 1197.14 },
  ],
}
test('réconcilie le centime signalé à partir des montants affichés', () => {
  const m = simulationModel(base, emptyScenario())
  assert.equal(m.currentNet, -94767)
  assert.equal(m.simulatedNet, -94767)
  assert.equal(m.delta, 0)
  assert.equal(
    m.baseSpend,
    m.categories.reduce((sum, c) => sum + c.before, 0),
  )
  assert.throws(
    () => simulationModel({ ...base, current_net: -947.66 }, emptyScenario()),
    /écart de réconciliation/,
  )
})
test('plusieurs coupes indépendantes, économies au centime et remise à zéro', () => {
  const s = emptyScenario()
  s.cuts = { Logement: '10', Courses: '25.25' }
  const m = simulationModel(base, s)
  assert.equal(m.categories[0].saved, 30000)
  assert.equal(m.categories[1].saved, 30228)
  assert.equal(m.saved, 60228)
  assert.equal(m.delta, m.saved)
  assert.equal(m.spend, m.baseSpend - m.saved)
  assert.equal(simulationModel(base, emptyScenario()).saved, 0)
  s.cuts.Logement = 0
  assert.equal(simulationModel(base, s).categories[1].saved, 30228)
})
test('revenu hypothétique, dépenses ajoutées et crédit ne modifient pas le net observé', () => {
  const s = emptyScenario()
  s.income = '4000'
  s.expenses = [
    { label: 'Projet', amount: '10.01' },
    { label: 'Abonnement', amount: '20' },
  ]
  s.loan = { amount: '1200', rate: '0', months: '12' }
  const m = simulationModel(base, s)
  assert.equal(m.currentNet, -94767)
  assert.equal(m.added, 3001)
  assert.equal(m.loan.payment, 10000)
  assert.equal(m.simulatedNet, 400000 - 419714 - 3001 - 10000)
  assert.equal(m.delta, 400000 - 324947 - 3001 - 10000)
})
test('arrondi des demi-centimes par catégorie puis sommation', () => {
  const b = {
    income: 1,
    base_spend: 0.02,
    current_net: 0.98,
    categories: [
      { category: 'A', before: 0.01 },
      { category: 'B', before: 0.01 },
    ],
  }
  const s = emptyScenario()
  s.cuts = { A: 50, B: 50 }
  assert.equal(simulationModel(b, s).saved, 2)
  assert.equal(simulationModel(b, s).spend, 0)
})
test('mensualités de crédit à taux fixe, sans taux et bornes de saisie', () => {
  assert.deepEqual(loanEstimate({ amount: '1200', rate: '0', months: '12' }), {
    payment: 10000,
    total: 120000,
    interest: 0,
  })
  assert.equal(
    loanEstimate({ amount: '15000', rate: '4.5', months: '48' }).payment,
    34205,
  )
  assert.throws(
    () => loanEstimate({ amount: '12', rate: '', months: '' }),
    /Complète/,
  )
  assert.throws(
    () => loanEstimate({ amount: '12', rate: '', months: '601' }),
    /durée/,
  )
})
test('absence de référence, zéro documenté et saisies invalides distingués', () => {
  assert.equal(simulationModel({ available: false }, emptyScenario()), null)
  assert.equal(
    simulationModel(
      { income: 0, current_net: 0, base_spend: 0, categories: [] },
      emptyScenario(),
    ).simulatedNet,
    0,
  )
  for (const income of ['-1', '1.001', 'NaN']) {
    const s = emptyScenario()
    s.income = income
    assert.throws(() => simulationModel(base, s))
  }
  for (const pct of ['101', '-1', '1.001']) {
    const s = emptyScenario()
    s.cuts.Logement = pct
    assert.throws(() => simulationModel(base, s))
  }
  const s = emptyScenario()
  s.expenses = [{ label: '', amount: '10' }]
  assert.throws(() => simulationModel(base, s), /intitulé/)
  assert.equal(cents('0.29'), 29)
})

test('un crédit sans intérêts ne crée pas un intérêt négatif par arrondi', () => {
  const loan = loanEstimate({ amount: '100', rate: '0', months: '3' })
  assert.equal(loan.payment, 3333)
  assert.equal(loan.total, 10000)
  assert.equal(loan.interest, 0)
  assert.equal(euroCents(-0), euroCents(0))
})
