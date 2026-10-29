import assert from 'node:assert/strict'
import test from 'node:test'
import { withCalendarGaps } from './chartCalendar.mjs'

test('calendar gaps are explicit null observations and original rows are preserved', () => {
  const first = { month: '2026-01', income: 100, expenses: 40, status: 'complete' }
  const last = { month: '2026-03', income: 90, expenses: 50, status: 'incomplete' }
  const result = withCalendarGaps([first, last])
  assert.deepEqual(result, [first, {
    month: '2026-02', income: null, expenses: null, net: null,
    status: 'missing', isMissing: true,
  }, last])
})

test('calendar rows are chronological and empty input stays empty', () => {
  assert.deepEqual(withCalendarGaps([]), [])
  assert.deepEqual(withCalendarGaps([
    { month: '2026-03', income: 1 }, { month: '2026-01', income: 2 },
  ]).map(row => row.month), ['2026-01', '2026-02', '2026-03'])
})
