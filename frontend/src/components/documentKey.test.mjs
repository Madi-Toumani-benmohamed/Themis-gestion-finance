import { test } from 'node:test'
import assert from 'node:assert/strict'
import { documentKey } from '../lib/laboratory/documentKey.mjs'
test('Document keys use native secure-context UUIDs when available', () => {
  const uuid = 'e6e00df3-dc1b-4b38-9674-41488d59ff78'
  assert.equal(
    documentKey({
      randomUUID: () => uuid,
      getRandomValues: () => assert.fail('unnecessary fallback'),
    }),
    uuid,
  )
})
test('HTTP fallback preserves RFC4122 version4 and variant with secure random bytes', () => {
  let calls = 0
  const key = documentKey({
    getRandomValues(bytes) {
      assert.ok(bytes instanceof Uint8Array)
      assert.equal(bytes.length, 16)
      calls++
      return bytes.map((_, i) => i)
    },
  })
  assert.equal(key, '00010203-0405-4607-8809-0a0b0c0d0e0f')
  assert.equal(calls, 1)
  const source = {
    getRandomValues: (bytes) => globalThis.crypto.getRandomValues(bytes),
  }
  const keys = new Set(Array.from({ length: 64 }, () => documentKey(source)))
  assert.equal(keys.size, 64)
  for (const key of keys)
    assert.match(
      key,
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
    )
})
test('Missing secure entropy is explicit, never replaced with Math.random', () => {
  assert.throws(() => documentKey(null), /identifiant indisponible/)
  assert.throws(() => documentKey({}), /identifiant indisponible/)
})
