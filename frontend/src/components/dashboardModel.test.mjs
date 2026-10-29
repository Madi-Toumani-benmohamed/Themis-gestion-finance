import test from 'node:test'
import assert from 'node:assert/strict'
import { dashboardModel } from './dashboardModel.mjs'
const complete = (month, income = 100.3, expenses = 100.1) => ({ month, income, expenses, status: 'complete' })

test('un mois absent ne reprend pas les chiffres du dernier mois disponible', () => {
  assert.equal(dashboardModel([complete('2026-08'), complete('2026-10')], '2026-09'), null)
})
test('le net est calculé en centimes et une donnée absente ne devient pas zéro', () => {
  assert.equal(dashboardModel([complete('2026-09')], '2026-09').net, .2)
  const model = dashboardModel([{ ...complete('2026-09'), income: null }], '2026-09')
  assert.equal(model.income, null); assert.equal(model.net, null); assert.equal(model.savings, null)
  assert.equal(dashboardModel([complete('2026-09', 0, 0)], '2026-09').net, 0)
  assert.throws(() => dashboardModel([complete('2026-09', 1.001)], '2026-09'), /précision/)
})
test('les écarts exigent deux mois civils consécutifs et complets', () => {
  assert.equal(dashboardModel([complete('2026-08'), complete('2026-10', 200)], '2026-10').incomeDelta, null)
  for (const status of ['current', 'incomplete', undefined]) {
    assert.equal(dashboardModel([complete('2026-08'), { ...complete('2026-09'), status }], '2026-09').incomeDelta, null)
    assert.equal(dashboardModel([{ ...complete('2026-08'), status }, complete('2026-09')], '2026-09').incomeDelta, null)
  }
  assert.equal(dashboardModel([complete('2025-12', 100), complete('2026-01', 125)], '2026-01').incomeDelta, 25)
})
test('les métadonnées absentes ne produisent pas un faux nombre de jours', () => {
  assert.equal(dashboardModel([{ ...complete('2026-09'), status: 'current' }], '2026-09').periodLabel, 'Mois en cours')
})
