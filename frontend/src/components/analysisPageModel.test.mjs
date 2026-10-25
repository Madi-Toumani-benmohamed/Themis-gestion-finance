import test from 'node:test'
import assert from 'node:assert/strict'
import {
  analysisNatureModel,
  analysisWindow,
  analysisMonth,
  analysisPeriodLabel,
  analysisMoney,
  analysisSource,
} from './analysisPageModel.mjs'
const report = () => ({
  available: true,
  month: '2020-01',
  transaction_count: 6,
  income: 3249.47,
  expenses: 4197.14,
  net: -947.67,
  real_spend: 4100,
  mandatory: 97.14,
  savings: 100,
  transfer_out: 22,
  transfer_in: 22,
  total_out: 4319.14,
  uncategorized_spent: 0,
  categories: [{ id: 1, name: 'TEST', spent: 4197.14, share_pct: 100 }],
})
test('rapport financier exact : écart d’un centime, postes et catégories réconciliés', () => {
  const data = analysisNatureModel(report(), '2020-01')
  assert.equal(data.money.net, -94767)
  for (const key of ['net', 'total_out', 'expenses', 'real_spend']) {
    const invalid = report()
    invalid[key] += 0.01
    assert.throws(() => analysisNatureModel(invalid, '2020-01'))
  }
  const wrong = report()
  wrong.categories[0].spent = 4197.13
  assert.throws(() => analysisNatureModel(wrong, '2020-01'))
  assert.match(analysisMoney(data.net), /947,67/)
})
test('absence, mois et catégories invalides ne deviennent pas de faux zéros', () => {
  assert.equal(
    analysisNatureModel(
      { available: false, month: '2020-01', transaction_count: 0 },
      '2020-01',
    ).money,
    null,
  )
  for (const change of [
    { month: '2020-02' },
    { income: null },
    { transaction_count: -1 },
    { categories: [...report().categories, ...report().categories] },
  ])
    assert.throws(() =>
      analysisNatureModel({ ...report(), ...change }, '2020-01'),
    )
})
test('fenêtres bornées, année bissextile, aucune inférence de complétude', () => {
  assert.deepEqual(analysisWindow('2020-02', 6), {
    from: '2019-09-01',
    to: '2020-02-29',
  })
  assert.equal(analysisMonth(-1, '2020-01'), '2019-12')
  assert.throws(() => analysisMonth(0, '2020-13'))
  assert.match(
    analysisPeriodLabel({ is_current: true, completeness: 'unknown' }),
    /en cours.*inconnue/,
  )
  assert.match(
    analysisPeriodLabel({ is_current: false, completeness: 'unknown' }),
    /non établie/,
  )
})
test('traçabilité conserve les filtres exacts et rejette les identifiants invalides', () => {
  assert.deepEqual(
    analysisSource(
      { ids: [1, 2], merchant: 'TEST', type: 'debit', unsafe: 'not allowed' },
      { month: '2020-01' },
    ),
    { month: '2020-01', type: 'debit', merchant: 'TEST', ids: [1, 2] },
  )
  assert.equal(analysisSource({ ids: [-1] }).ids, undefined)
})
