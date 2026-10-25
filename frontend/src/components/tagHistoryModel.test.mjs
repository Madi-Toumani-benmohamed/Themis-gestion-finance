import test from 'node:test'
import assert from 'node:assert/strict'
import { tagHistoryWindow, tagHistoryModel } from './tagHistoryModel.mjs'
const window = tagHistoryWindow('2020-03', 3)
const response = () => ({
  query_from: window.from,
  query_to: window.to,
  buckets: ['2020-01', '2020-03'],
  series: [{ tag: 'food', values: [10.01, 30.03] }],
})
test('historique : six mois, deux ans, dix ans et bornes personnalisées exactes', () => {
  assert.deepEqual(tagHistoryWindow('2020-02', 24), {
    from: '2018-03-01',
    to: '2020-02-29',
    months: 24,
    params: { month: '2020-02', months: 24 },
  })
  assert.equal(tagHistoryWindow('2020-02', 6).from, '2019-09-01')
  assert.equal(tagHistoryWindow('2020-02', 120).from, '2010-03-01')
  assert.deepEqual(
    tagHistoryWindow('2020-03', 'custom', '2018-06', '2020-02'),
    {
      from: '2018-06-01',
      to: '2020-02-29',
      months: 21,
      params: { from: '2018-06-01', to: '2020-02-29' },
    },
  )
})
test('les périodes invalides et dates futures ne déclenchent pas une autre fenêtre', () => {
  for (const args of [
    ['2020-03', 0],
    ['2020-03', 601],
    ['2020-03', 'custom', '', '2020-02'],
    ['2020-03', 'custom', '2020-04', '2020-05'],
    ['2020-03', 'custom', '2020-02', '2020-01'],
    ['2020-03', 'custom', '1900-01', '2020-03'],
    ['2999-01', 6],
    ['2020-13', 6],
  ])
    assert.throws(() => tagHistoryWindow(...args))
  const current = new Date().toISOString().slice(0, 7)
  assert.equal(
    tagHistoryWindow(current, 6).to,
    new Date().toISOString().slice(0, 10),
  )
})
test('les mois totalement absents interrompent toutes les courbes, sans zéro ni interpolation', () => {
  const model = tagHistoryModel(response(), window)
  assert.deepEqual(
    model.points.map((p) => p.tag_0),
    [10.01, null, 30.03],
  )
  assert.deepEqual(
    model.points.map((p) => p.period),
    ['2020-01', '2020-02', '2020-03'],
  )
  assert.equal(model.points[0].tag_0_isolated, true)
  assert.equal(model.hasObservations, true)
  assert.equal(
    tagHistoryModel({ ...response(), buckets: [], series: [] }, window)
      .hasObservations,
    false,
  )
  const zero = tagHistoryModel(
    { ...response(), series: [{ tag: 'food', values: [0, null] }] },
    window,
  )
  assert.equal(zero.points[0].tag_0, 0)
  assert.equal(zero.hasObservations, true)
})
test('noms de tags spéciaux et données incohérentes restent séparés des métadonnées du graphique', () => {
  const payload = response()
  payload.series = [
    { tag: 'period', values: [10.01, null] },
    { tag: 'a.b', values: [1.23, null] },
    { tag: '__proto__', values: [null, 2.34] },
  ]
  const data = tagHistoryModel(payload, window)
  assert.equal(data.points[0].period, '2020-01')
  assert.equal(data.points[0].tag_1, 1.23)
  for (const change of [
    { query_to: '2020-04-01' },
    { buckets: ['2020-03', '2020-01'] },
    { buckets: ['2020-01', '2020-01'] },
    { series: [{ tag: 'food', values: [10.001, 30.03] }] },
    { series: [{ tag: 'food', values: [-1, 30.03] }] },
    { series: [{ tag: 'food', values: [10.01] }] },
  ])
    assert.throws(() => tagHistoryModel({ ...response(), ...change }, window))
})
