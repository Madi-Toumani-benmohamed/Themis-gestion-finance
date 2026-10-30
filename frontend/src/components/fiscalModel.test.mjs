import { test } from 'node:test'
import assert from 'node:assert/strict'
import { fiscalResult } from './fiscalModel.mjs'
const fixture = () => ({
  rules: {
    income_year: 2025,
    assessment_year: 2026,
    status: 'official',
    version: 'fr-ir-2025-v1',
    sources: ['https://www.impots.gouv.fr/source'],
  },
  half_parts: 2,
  annual_income_cents: 3000000,
  salary_deduction_cents: 0,
  taxable_cents: 3000000,
  gross_tax_cents: 210400,
  discount_cents: 0,
  tax_cents: 210400,
  average_rate_pct: '7.01',
  marginal_rate_pct: 30,
  observed_paid_cents: null,
  projected_future_paid_cents: null,
  gap_before_future_cents: null,
  settlement_cents: null,
  settlement_status: 'unknown',
  steps: [],
  assumptions: [],
  limitations: [],
})
test('Reject incoherent fiscal responses and preserve unknown PAS', () => {
  const r = fixture()
  assert.equal(fiscalResult(r, 2025).settlement_cents, null)
  for (const change of [
    { tax_cents: 210400.5 },
    { tax_cents: 0 },
    { taxable_cents: 2700000 },
    { rules: { ...r.rules, income_year: 2024 } },
    { rules: { ...r.rules, sources: ['https://attacker.test/'] } },
    { settlement_cents: 0 },
    { gap_before_future_cents: 210400 },
    { discount_cents: 210401 },
    { marginal_rate_pct: 99 },
  ])
    assert.throws(() => fiscalResult({ ...r, ...change }, 2025))
  const paid = {
    ...r,
    observed_paid_cents: 220000,
    gap_before_future_cents: -9600,
    settlement_cents: -9600,
    settlement_status: 'annual_estimate',
  }
  assert.equal(fiscalResult(paid, 2025).settlement_cents, -9600)
  assert.equal(
    fiscalResult(
      { ...paid, settlement_cents: null, settlement_status: 'incomplete' },
      2025,
    ).settlement_cents,
    null,
  )
  assert.throws(() =>
    fiscalResult(
      { ...paid, projected_future_paid_cents: 100, settlement_cents: -9600 },
      2025,
    ),
  )
})
