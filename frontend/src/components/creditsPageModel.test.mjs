import test from 'node:test'
import assert from 'node:assert/strict'
import {
  amountCents,
  calculateProgress,
  creditPayload,
  decimalInput,
  formatEuro,
  getTaxStatus,
  inputAmount,
  isCreditCompleted,
  moneyText,
  sortCredits,
  sortTaxes,
  statusLabel,
} from './creditsPageModel.mjs'

test('centimes exacts et inconnus distincts de zéro', () => {
  assert.equal(
    amountCents(
      { amounts_cents: { remaining: null }, remaining: 0 },
      'remaining',
    ),
    null,
  )
  assert.equal(
    moneyText({ amounts_cents: { remaining: 0 } }, 'remaining'),
    '0,00\u00a0€',
  )
  assert.equal(moneyText({}, 'remaining'), 'Non documenté')
  assert.equal(
    formatEuro(9007199254740991),
    '90\u202f071\u202f992\u202f547\u202f409,91\u00a0€',
  )
  assert.equal(
    inputAmount(
      { amounts_cents: { remaining: 9007199254740991 } },
      'remaining',
    ),
    '90071992547409.91',
  )
  assert.equal(amountCents({ remaining: '33.33' }, 'remaining'), 3333)
  assert.equal(amountCents({ remaining: 0.1 + 0.2 }, 'remaining'), null)
})
test('les champs décimaux ne sont jamais arrondis', () => {
  assert.equal(decimalInput('33,33'), '33.33')
  assert.equal(decimalInput('0', false), '0.00')
  for (const n of ['0', '-1', '0.001', 'NaN', '1e2', '90071992547409.92'])
    assert.throws(() => decimalInput(n))
  const form = {
    name: ' Test ',
    monthly: '33,33',
    months: '12',
    start: '2026-01',
    opening: '',
    note: '',
  }
  assert.deepEqual(creditPayload(form), {
    name: 'Test',
    monthly_amount: '33.33',
    months_total: 12,
    start_month: '2026-01',
    opening_paid: null,
    note: '',
  })
  assert.equal(creditPayload({ ...form, opening: '0' }).opening_paid, '0.00')
  for (const bad of [
    { months: '1201' },
    { months: '1.2' },
    { start: '2026-13' },
    { start: '0000-01' },
    { name: ' ' },
  ])
    assert.throws(() => creditPayload({ ...form, ...bad }))
})
test('aucune progression ou clôture dérivée du calendrier', () => {
  const old = {
    months_remaining: 0,
    months_paid: 12,
    months_total: 12,
    end_month: '2020-01',
    remaining: null,
    status: 'unknown',
  }
  assert.equal(calculateProgress(old), null)
  assert.ok(!isCreditCompleted(old))
  const partial = {
    ...old,
    payment_history_complete: true,
    status: 'active',
    progress_pct: 33.33,
    amounts_cents: { remaining: 6667 },
  }
  assert.equal(calculateProgress(partial), 33.33)
  assert.ok(!isCreditCompleted(partial))
  assert.ok(
    isCreditCompleted({
      ...partial,
      status: 'paid',
      amounts_cents: { remaining: 0 },
    }),
  )
  assert.equal(calculateProgress({ ...partial, status: 'invalid' }), null)
})
test('retard confirmé uniquement par statut backend', () => {
  assert.equal(getTaxStatus({ days_left: -100 }), 'unknown')
  assert.equal(statusLabel('to_verify'), 'Paiement à vérifier')
  assert.equal(getTaxStatus({ status: 'partial', days_left: -100 }), 'partial')
  assert.equal(statusLabel('late_confirmed'), 'Retard confirmé')
})
test('tris sans mutation, valeurs inconnues en fin', () => {
  const rows = [
    { id: 1, name: 'Z', remaining: null },
    { id: 2, name: 'A', amounts_cents: { remaining: 100 } },
    { id: 3, name: 'B', amounts_cents: { remaining: 300 } },
  ]
  assert.deepEqual(
    sortCredits(rows).map((x) => x.id),
    [3, 2, 1],
  )
  assert.deepEqual(
    sortCredits(rows, 'name').map((x) => x.id),
    [2, 3, 1],
  )
  assert.deepEqual(
    rows.map((x) => x.id),
    [1, 2, 3],
  )
  assert.deepEqual(
    sortTaxes([
      { id: 1, paid: true, due_date: '2020-01-01' },
      { id: 2, paid: false, due_date: '2026-01-01' },
    ]).map((x) => x.id),
    [2, 1],
  )
})
