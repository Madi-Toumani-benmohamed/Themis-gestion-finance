import test from 'node:test'
import assert from 'node:assert/strict'
import { financialSummaryModel } from './financialSummaryModel.mjs'

test('a negative current result never determines the presentation status', () => {
  const model = financialSummaryModel({ period: { is_current: true }, net: -859.28, tone: 'neutral' })
  assert.equal(model.status, 'Flux enregistrés')
  assert.equal(model.resultLabel, 'Solde des flux à date')
  assert.equal(model.complete, false)
})

test('real alerts remain visible and ordered during a provisional period', () => {
  const model = financialSummaryModel({ period: { is_current: true }, tone: 'attention', points: [
    { kind: 'anomalies', tone: 'attention' },
    { kind: 'budget_over', tone: 'alert' },
    { kind: 'deficit', tone: 'neutral' },
    { kind: 'budgets_ok', tone: 'positive' },
  ] })
  assert.equal(model.status, 'À surveiller')
  assert.deepEqual(model.attention.map(p => p.kind), ['budget_over', 'anomalies'])
  assert.deepEqual(model.observations.map(p => p.kind), ['budgets_ok'])
})

test('historical status alone cannot establish complete coverage', () => {
  const model = financialSummaryModel({ period: { status: 'complete' }, tone: 'positive' })
  assert.equal(model.status, 'Données incomplètes')
  assert.equal(model.resultLabel, 'Résultat des flux disponibles')
})

test('documented historical periods use the monthly result', () => {
  const model = financialSummaryModel({ period: { is_current: false }, period_complete: true, tone: 'alert' })
  assert.equal(model.status, 'Situation sous tension')
  assert.equal(model.resultLabel, 'Résultat des flux du mois')
})
