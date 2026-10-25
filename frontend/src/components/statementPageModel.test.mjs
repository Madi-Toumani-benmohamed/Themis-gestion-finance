import test from 'node:test'
import assert from 'node:assert/strict'
import {
  statementListModel,
  statementSummary,
  filterStatements,
  statementStatus,
  statementStatusIcon,
} from './statementPageModel.mjs'
const row = {
  id: 1,
  filename: 'cycle TEST.pdf',
  account_id: 11,
  account_name: 'Courant TEST',
  period_start: '2026-09-21',
  period_end: '2026-10-21',
  uploaded_at: '2026-10-23',
  opening_balance_cents: 10000,
  closing_balance_cents: 9001,
  sum_cents: -999,
  credits_cents: 0,
  debits_cents: 999,
  theoretical_balance_cents: 9001,
  difference_cents: 0,
  status: 'validated',
  effective_validated: true,
  controls: [],
  history: [],
}
test('statement reconciliation is exact and never substitutes missing balances', () => {
  assert.deepEqual(statementListModel([row]), [row])
  assert.throws(() => statementListModel([{ ...row, difference_cents: 1 }]))
  assert.throws(() => statementListModel([{ ...row, credits_cents: 1 }]))
  assert.throws(() => statementListModel([row, row]))
  assert.deepEqual(
    statementListModel([
      {
        ...row,
        opening_balance_cents: null,
        closing_balance_cents: null,
        theoretical_balance_cents: null,
        difference_cents: null,
      },
    ])[0].difference_cents,
    null,
  )
})
test('summary distinguishes validated sources from unreviewed arithmetic', () => {
  assert.deepEqual(
    statementSummary([
      row,
      {
        ...row,
        id: 2,
        status: 'to_review',
        effective_validated: false,
        period_end: '2026-11-21',
      },
    ]),
    { total: 2, review: 1, lastValidated: '2026-10-21' },
  )
  assert.equal(statementSummary([]).lastValidated, null)
  assert.equal(statementStatus('unexecuted'), 'Contrôle non exécuté')
  assert.equal(statementStatus('failed'), 'Contrôles en échec')
  assert.equal(statementStatusIcon('validated'), '✓')
  assert.equal(statementStatusIcon('to_review'), '!')
  assert.equal(statementStatusIcon('failed'), '×')
})
test('bank periods spanning months and unknown dates remain inspectable', () => {
  const unknown = {
    ...row,
    id: 2,
    period_start: null,
    period_end: null,
    status: 'to_review',
  }
  assert.equal(filterStatements([row, unknown], { month: '2026-09' }).length, 2)
  assert.equal(filterStatements([row, unknown], { account: '12' }).length, 0)
  assert.equal(
    filterStatements([row, unknown], {
      search: 'COURANT',
      status: 'to_review',
    })[0].id,
    2,
  )
  assert.equal(
    filterStatements([row, unknown], { ids: new Set([1]) }).length,
    1,
  )
})
