import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  projectBudget,
  projectBudgetInput,
  projectCumulative,
  projectMoney,
  projectReportModel,
} from './projectModel.mjs'
test('Project inputs preserve absent versus zero and reject inexact cents', () => {
  assert.equal(projectBudget(''), null)
  assert.equal(projectBudget('0'), 0)
  assert.equal(projectBudget('12,34'), 1234)
  assert.equal(projectBudgetInput(1234), '12.34')
  assert.equal(projectBudgetInput(null), '')
  for (const invalid of ['1.001', '-1', 'NaN', '1e2', '10000000000'])
    assert.throws(() => projectBudget(invalid))
  assert.equal(projectMoney(null), 'Non disponible')
})
test('Observed cumulative series reconcile integer cents without filling missing periods', () => {
  assert.deepEqual(
    projectCumulative([
      { name: '2026-01', amount_cents: 1001 },
      { name: '2026-03', amount_cents: 1 },
    ]).map((r) => r.cumulative_cents),
    [1001, 1002],
  )
  assert.throws(() => projectCumulative([{ amount_cents: 1.5 }]))
  assert.throws(() =>
    projectCumulative([
      { amount_cents: Number.MAX_SAFE_INTEGER },
      { amount_cents: 1 },
    ]),
  )
})

test('Malformed or unreconciled reports cannot masquerade as an observed zero', () => {
  for (const data of [null, {}, { project: { cost_cents: 0 } }])
    assert.throws(() => projectReportModel(data))
  const project = {
    id: 1,
    cost_cents: 1,
    transaction_count: 1,
    suggestion_count: 0,
    budget_cents: null,
    remaining_cents: null,
    overrun_cents: null,
    consumed_pct: null,
  }
  const data = {
    project,
    by_category: [{ amount_cents: 1 }],
    by_merchant: [{ amount_cents: 1 }],
    by_month: [{ amount_cents: 1 }],
    by_tag: [],
    observations: [],
  }
  assert.equal(projectReportModel(data), data)
  assert.throws(() => projectReportModel({ ...data, by_category: [] }))
})
