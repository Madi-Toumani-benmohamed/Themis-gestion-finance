import test from 'node:test'
import assert from 'node:assert/strict'
import { budgetWidgetModel } from './budgetWidgetModel.mjs'

test('présente la catégorie et la consommation du plafond reçues de l’API', () => {
  assert.deepEqual(budgetWidgetModel({ domain_name: 'Courses', spent: 150, monthly_limit: 200, ratio: 0.75 }), {
    name: 'Courses', spent: 150, limit: 200, percent: 75, width: 75, over: false, difference: 50,
  })
})

test('conserve le pourcentage réel au-delà du plafond et borne la barre', () => {
  const model = budgetWidgetModel({ domain_name: 'Courses', spent: 250, monthly_limit: 200, ratio: 1.25 })
  assert.equal(model.percent, 125)
  assert.equal(model.width, 100)
  assert.equal(model.over, true)
  assert.equal(model.difference, 50)
})

test('un plafond atteint ou sans dépense ne constitue pas un dépassement', () => {
  for (const spent of [0, 200]) {
    const model = budgetWidgetModel({ domain_name: 'Courses', spent, monthly_limit: 200, ratio: spent / 200 })
    assert.equal(model.over, false)
    assert.equal(model.difference, 200 - spent)
  }
})
