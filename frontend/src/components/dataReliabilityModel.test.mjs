import test from 'node:test'
import assert from 'node:assert/strict'
import { reliabilityDate, reliabilityStatusLabel, reliabilitySummary } from './dataReliabilityModel.mjs'

test('no accounts is an empty state, not a complete report', () => {
  assert.deepEqual(reliabilitySummary({ accounts: [] }), { total: 0, documented: 0, toReview: 0, unknownPeriods: 0 })
})
test('validated coverage and review counts are independent', () => {
  assert.deepEqual(reliabilitySummary({ accounts: [
    { status: 'documented', to_review: 1, unknown_periods: 1 },
    { status: 'insufficient', to_review: 0, unknown_periods: 0 },
  ] }), { total: 2, documented: 1, toReview: 1, unknownPeriods: 1 })
})
test('current coverage never claims a complete monthly result', () => {
  assert.equal(reliabilityStatusLabel({ status: 'documented' }, true), 'Couverture documentée à date')
  assert.equal(reliabilityStatusLabel({ status: 'documented' }, false), 'Couverture documentée')
})
test('unknown dates stay unknown and date labels preserve calendar day', () => {
  assert.equal(reliabilityDate(''), 'Indisponible')
  assert.equal(reliabilityDate('2026-10-21'), '21 octobre 2026')
})
