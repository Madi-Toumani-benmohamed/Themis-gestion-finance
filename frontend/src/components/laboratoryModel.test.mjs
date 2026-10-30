import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  evaluateQuick,
  evaluateWorkbook,
} from '../lib/laboratory/calculationEngine.mjs'
import {
  emptyContent,
  convertCalculation,
  resizeContent,
  shiftedFormula,
  migrateDocument,
  safeClipboardText,
  validateContent,
} from '../lib/laboratory/workbookModel.mjs'
test('Decimal expressions, precedence, percentages and explicit errors', () => {
  for (const [expr, value] of [
    ['1250+350-89', '1511'],
    ['(1500-275)/3', '408.3333333333333333333333333333333333333'],
    ['199.99*12', '2399.88'],
    ['0.1+0.2', '0.3'],
    ['100+10%', '110'],
    ['ROUND(-1.005;2)', '-1.01'],
  ])
    assert.equal(evaluateQuick(expr).value, value, expr)
  for (const expr of [
    '1/0',
    '(2+',
    'import("x")',
    'a=1',
    '[1;2]',
    '2^100000000',
    'fetch("https://x")',
    'constructor.constructor("x")()',
    '1e999999999',
    '1e-999999999',
  ])
    assert.throws(() => evaluateQuick(expr), undefined, expr)
})
test('References, ranges, lazy IF, errors and recalculation', () => {
  const c = emptyContent()
  c.cells = {
    A1: { input: '0.1', format: 'number' },
    A2: { input: '0.2', format: 'number' },
    A3: { input: 'texte', format: 'number' },
    B1: { input: '=SUM(A1:A4)', format: 'currency' },
    B2: { input: '=AVERAGE(A1:A4)', format: 'number' },
    B3: { input: '=COUNT(A1:A4)', format: 'number' },
    B4: { input: '=IF(A1>0;ROUND(B1;2);1/0)', format: 'number' },
    C1: { input: '=MIN(A1:A2)+MAX(A1:A2)', format: 'number' },
    C2: { input: '=D2', format: 'number' },
    D2: { input: '=C2', format: 'number' },
    C3: { input: '=A99', format: 'number' },
    D3: { input: '=C2+1', format: 'number' },
  }
  let r = evaluateWorkbook(c)
  assert.equal(r.B1.money_cents, 30)
  assert.equal(r.B2.value, '0.15')
  assert.equal(r.B3.value, '2')
  assert.equal(r.B4.value, '0.3')
  assert.equal(r.C1.value, '0.3')
  assert.equal(r.C2.code, '#CYCLE!')
  assert.equal(r.D3.code, '#CYCLE!')
  assert.equal(r.C3.code, '#REF!')
  c.cells.A1.input = '0.4'
  r = evaluateWorkbook(c)
  assert.equal(r.B1.money_cents, 60)
})
test('Conversion, references and versioned documents preserve information', () => {
  const item = {
    expression: '1250+350-89',
    result: '1511',
    created_at: '2026-10-30T12:00:00.000Z',
  }
  const d = convertCalculation(item.expression, item.result, [item])
  assert.equal(d.content.source_expression, item.expression)
  assert.equal(evaluateWorkbook(d.content).B2.value, '1511')
  assert.deepEqual(d.content.source_operands, ['1250', '350', '89'])
  assert.deepEqual(d.content.history, [item])
  assert.equal(shiftedFormula('=$A1+B$2+"A1"', 1, 1), '=$A2+C$2+"A1"')
  d.content.cells.A1.input = '=H20'
  const resized = resizeContent(d.content, 19, 8)
  assert.match(resized.cells.A1.input, /REF_DELETED/)
  assert.equal(evaluateWorkbook(resized).A1.code, '#REF!')
  assert.throws(() =>
    migrateDocument({ schema_version: 9, content: d.content }),
  )
  assert.equal(
    migrateDocument({ schema_version: 0, content: { grid: [['1', '=A1+1']] } })
      .content.cells.B1.input,
    '=A1+1',
  )
  assert.equal(safeClipboardText('=HYPERLINK("x")'), '\'=HYPERLINK("x")')
  const bad = emptyContent()
  bad.cells.A101 = { input: '1', format: 'number' }
  assert.throws(() => validateContent(bad))
})

test('Security limits, isolation of errors and monetary rounding', () => {
  for (const expression of [
    'A1+1',
    'A1:A2',
    'IF(true;1;import("fs"))',
    'this.constructor',
    'SUM([1;2])',
    'f(x)=x',
    '1:1000000000',
    '2 cm',
    'document.cookie',
    '"x".repeat(100000000)',
    '1+'.repeat(260) + '1',
  ])
    assert.throws(() => evaluateQuick(expression), undefined, expression)
  assert.equal(evaluateQuick('IF(false;1/0;ROUND(2.675;2))').value, '2.68')
  const c = emptyContent()
  c.cells = {
    A1: { input: '=1/0', format: 'number' },
    B1: { input: '=A1+1', format: 'number' },
    C1: { input: '=1.005', format: 'currency' },
    D1: { input: '=-1.005', format: 'currency' },
    E1: { input: '=0.1+0.2', format: 'number' },
    F1: { input: '<img src=x onerror=alert(1)>', format: 'number' },
  }
  const result = evaluateWorkbook(c)
  assert.equal(result.A1.code, '#DIV/0!')
  assert.equal(result.B1.code, '#DIV/0!')
  assert.equal(result.C1.money_cents, 101)
  assert.equal(result.D1.money_cents, -101)
  assert.equal(result.E1.value, '0.3')
  assert.equal(result.F1.type, 'text')
  assert.equal(safeClipboardText('-1.25'), '-1.25')
  assert.equal(safeClipboardText('@HYPERLINK("x")'), '\'@HYPERLINK("x")')
  assert.equal(shiftedFormula('=A1000+"A1"', 1, 1), '=A1000+"A1"')
  assert.deepEqual(
    migrateDocument(
      JSON.parse(JSON.stringify({ schema_version: 1, content: c })),
    ).content,
    c,
  )
  const tooBig = emptyContent()
  tooBig.rows = 100
  tooBig.columns = 26
  for (let i = 0; i < 1001; i++)
    tooBig.cells[
      `${String.fromCharCode(65 + (i % 26))}${1 + Math.floor(i / 26)}`
    ] = { input: '1', format: 'number' }
  assert.throws(() => validateContent(tooBig))
})
