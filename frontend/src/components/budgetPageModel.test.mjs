import test from 'node:test'
import assert from 'node:assert/strict'
import {
  budgetCents,
  budgetEnvelope,
  budgetPageModel,
  budgetMonth,
  recurringFrequencyLabel,
} from './budgetPageModel.mjs'
const row = (spent = 150, scope = 'expense') => ({
  id: 1,
  domain_name: 'Courses',
  monthly_limit: 200,
  spent,
  remaining: 200 - spent,
  scope,
  debit_total: spent,
  credit_total: 10,
})
const report = (rows = [row()]) => ({
  budgets: rows,
  summary: {
    allocated: 200,
    spent: 150,
    remaining: 50,
    over_count: 0,
    coverage_ratio: 0.75,
    uncategorized_spent: 50,
    uncategorized_count: 1,
    observed_expenses: 200,
    covered_expenses: 150,
    credits_observed: 0,
  },
  period: { completeness: 'partial', is_current: true },
})
test('enveloppe vide, plafond atteint et dépassement ont des statuts distincts', () => {
  assert.equal(budgetEnvelope(row(0)).status, 'available')
  assert.equal(budgetEnvelope(row(200)).status, 'reached')
  const exceeded = budgetEnvelope(row(250))
  assert.equal(exceeded.status, 'over')
  assert.equal(exceeded.width, 100)
  assert.equal(exceeded.ratio, 1.25)
  assert.equal(exceeded.remainingCents, -5000)
})
test('synthèse exacte, crédits séparés, transferts exclus et période provisoire', () => {
  const model = budgetPageModel(
    report([row(), { ...row(50, 'transfer'), id: 2 }]),
  )
  assert.equal(model.allocated, 20000)
  assert.equal(model.spent, 15000)
  assert.equal(model.remaining, 5000)
  assert.equal(model.expenses[0].creditCents, 1000)
  assert.equal(model.transfers.length, 1)
  assert.equal(model.complete, false)
  assert.equal(model.current, true)
})
test('absence de budget et couverture inconnue ne deviennent pas un taux nul', () => {
  const m = budgetPageModel({
    budgets: [],
    summary: {
      allocated: 0,
      spent: 0,
      remaining: 0,
      over_count: 0,
      coverage_ratio: null,
      observed_expenses: 0,
      covered_expenses: 0,
      credits_observed: 0,
      uncategorized_spent: 0,
      uncategorized_count: 0,
    },
    period: { completeness: 'unknown' },
  })
  assert.equal(m.ratio, null)
  assert.equal(m.coverageRatio, null)
})
test('une incohérence de source ou un centime perdu interrompt la présentation', () => {
  assert.throws(
    () =>
      budgetPageModel({
        ...report(),
        summary: { ...report().summary, spent: 149.99 },
      }),
    /réconcilient/,
  )
  assert.throws(() => budgetEnvelope({ ...row(), remaining: 49.99 }), /restant/)
  assert.throws(() => budgetCents('1.001'), /précision/)
  assert.equal(budgetCents('0.29'), 29)
})
test('la navigation mensuelle franchit les années et la fréquence manquante reste inconnue', () => {
  assert.equal(budgetMonth(-1, '2026-01'), '2025-12')
  assert.equal(budgetMonth(1, '2025-12'), '2026-01')
  assert.equal(recurringFrequencyLabel(null), 'Fréquence non établie')
})

test('une donnée absente ou un périmètre inconnu ne devient pas un zéro établi', () => {
  const source = report()
  delete source.summary.observed_expenses
  assert.throws(() => budgetPageModel(source), /Montant/)
  assert.throws(
    () => budgetEnvelope({ ...row(), scope: 'unknown' }),
    /Périmètre/,
  )
  assert.throws(
    () =>
      budgetPageModel({
        ...report(),
        summary: { ...report().summary, coverage_ratio: 1 },
      }),
    /couverture/,
  )
})
