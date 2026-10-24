import test from 'node:test'
import assert from 'node:assert/strict'
import { previewExpired, previewPage } from './importPreviewModel.mjs'
import { reviewedImport } from './reviewedImport.mjs'

test('preview filters and paginates without changing operation amounts', () => {
  const txs = Array.from({ length: 120 }, (_, i) => ({ date: '2026-10-21', amount: -i, duplicate: i % 2 === 0 }))
  const page = previewPage(txs, 'new', 1)
  assert.equal(page.pages, 2)
  assert.equal(page.total, 60)
  assert.equal(page.rows.length, 10)
  assert.equal(page.rows[0].amount, -101)
  assert.equal(previewPage(txs, 'duplicates', 99).currentPage, 1)
  assert.equal(previewPage([], 'all', 3).currentPage, 0)
})
test('expired previews are identified at the exact expiration time', () => {
  assert.equal(previewExpired('2026-10-24T00:15:00Z', Date.parse('2026-10-24T00:14:59Z')), false)
  assert.equal(previewExpired('2026-10-24T00:15:00Z', Date.parse('2026-10-24T00:15:00Z')), true)
})
test('nothing is confirmed while the user is still reviewing', async () => {
  const calls = []
  let approve
  const api = {
    previewStatement: async () => { calls.push('preview'); return { token: 'server-token' } },
    confirmStatementPreview: async token => { calls.push(token); return { inserted: 2 } },
  }
  const work = reviewedImport({ api, review: () => new Promise(resolve => { approve = resolve }) })
  await Promise.resolve()
  assert.deepEqual(calls, ['preview'])
  approve(true)
  const result = await work
  assert.equal(result.result.inserted, 2)
  assert.deepEqual(calls, ['preview', 'server-token'])
})
test('cancelled preview is deleted and never confirmed, even if cancellation request fails', async () => {
  let confirmations = 0, cancellations = 0
  const api = {
    previewStatement: async () => ({ token: 'token' }),
    cancelStatementPreview: async () => { cancellations++; throw new Error('offline') },
    confirmStatementPreview: async () => { confirmations++ },
  }
  assert.equal((await reviewedImport({ api, review: async () => false })).cancelled, true)
  assert.equal(confirmations, 0)
  assert.equal(cancellations, 1)
})
test('confirmation failure never falls back to direct upload or retries', async () => {
  let confirmations = 0
  const api = {
    previewStatement: async () => ({ token: 'expired' }),
    confirmStatementPreview: async () => { confirmations++; throw new Error('expired') },
    upload: () => { assert.fail('must not use direct import') },
  }
  await assert.rejects(reviewedImport({ api, review: async () => true }), /expired/)
  assert.equal(confirmations, 1)
})
