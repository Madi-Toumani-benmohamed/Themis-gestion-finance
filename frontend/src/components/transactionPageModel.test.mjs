import test from 'node:test'
import assert from 'node:assert/strict'
import { transactionReportModel, transactionCents, transactionDayTotal, transactionTags } from './transactionPageModel.mjs'
const report = { transactions: [{ id: 1, amount: -4197.14, amount_cents: -419714 }], total: 1, summary: { expenses: 4197.14, income: 3249.47, net: -947.67, transfer_debits: 22, transfer_credits: 22 } }
test('filtered server totals reconcile exactly including the reported one-cent regression', () => {
  assert.equal(transactionReportModel(report).money.net, -94767)
  assert.throws(() => transactionReportModel({ ...report, summary: { ...report.summary, net: -947.66 } }), /réconcilient/)
})
test('day totals retain compensating operations without floating point error', () => {
  assert.equal(transactionDayTotal([{ amount: -22 }, { amount: 22 }, { amount: -0.1 }, { amount: -0.2 }]), -30)
})
test('inconsistent or overprecise amounts must not appear as reliable', () => {
  assert.throws(() => transactionCents({ amount: 10.001 }), /précision/)
  assert.throws(() => transactionCents({ amount: 10, amount_cents: 999 }), /incohérent/)
  assert.throws(() => transactionReportModel({ transactions: [], total: 0 }), /incomplète/)
})
test('tag editing deduplicates normalized entries and supports clearing', () => {
  assert.deepEqual(transactionTags(' Taxes, taxes, Mensuel , , '), ['taxes', 'mensuel'])
  assert.deepEqual(transactionTags(''), [])
})

test('duplicated response rows and impossible counts cannot masquerade as reconciled data', () => {
  assert.throws(() => transactionReportModel({ ...report, total: 2, transactions: [...report.transactions, ...report.transactions] }), /répété/)
  assert.throws(() => transactionReportModel({ ...report, total: 0 }), /périmètre/)
})
