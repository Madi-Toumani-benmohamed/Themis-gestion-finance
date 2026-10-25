import test from 'node:test'
import assert from 'node:assert/strict'
import { activeAllocatedCents, allocationRequest, amountInputFromCents, personalCostCents, reimbursementKindLabel, reimbursementStatusLabel } from './reimbursementModel.mjs'

const caseView = (status, allocations) => ({ status, allocations })

test('allocation amounts are parsed as exact cents and bounded by both sides', () => {
  assert.deepEqual(allocationRequest(12, '22,05', 3000, 2500), { expense_transaction_id: 12, amount_cents: 2205 })
  assert.equal(amountInputFromCents(2205), '22.05')
  assert.throws(() => allocationRequest(12, '25.01', 3000, 2500), /dépasse/)
  assert.throws(() => allocationRequest(12, '0', 3000, 2500), /positif/)
  assert.throws(() => allocationRequest(12, '1.001', 3000, 2500), /précision/)
})

test('partial and full allocations calculate personal cost without floating point', () => {
  const partial = caseView('partial', [{ expense_transaction_id: 7, amount_cents: 3000 }])
  assert.equal(activeAllocatedCents(partial, 7), 3000)
  assert.equal(personalCostCents(-6000, [partial], 7), 3000)
  const full = caseView('allocated', [{ expense_transaction_id: 7, amount_cents: 6000 }])
  assert.equal(personalCostCents(-6000, [full], 7), 0)
})

test('revoked allocations and voided cases do not reduce the expense cost', () => {
  const cases = [
    caseView('partial', [{ expense_transaction_id: 7, amount_cents: 2500, voided_at: '2026-10-25T12:00:00Z' }]),
    caseView('voided', [{ expense_transaction_id: 7, amount_cents: 2000 }]),
  ]
  assert.equal(activeAllocatedCents(cases[0], 7), 0)
  assert.equal(personalCostCents(-6000, cases, 7), 6000)
})

test('UI labels distinguish qualification from matching state', () => {
  assert.equal(reimbursementKindLabel('merchant_refund'), 'Retour ou annulation commerciale')
  assert.equal(reimbursementStatusLabel('unmatched'), 'À rapprocher')
  assert.equal(reimbursementStatusLabel('allocated'), 'Rapproché')
})
