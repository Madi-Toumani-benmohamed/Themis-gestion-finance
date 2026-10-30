export const LIMITS = {
  rows: 100,
  columns: 26,
  cells: 1000,
  input: 512,
  bytes: 200000,
  history: 50,
}
export const address = (row, column) =>
  `${String.fromCharCode(65 + column)}${row + 1}`
export function coordinates(ref) {
  const match = /^\$?([A-Z])\$?([1-9]\d{0,2})$/.exec(ref)
  if (!match) throw new Error('Référence de cellule invalide.')
  return [Number(match[2]) - 1, match[1].charCodeAt(0) - 65]
}
export const emptyContent = () => ({
  rows: 20,
  columns: 8,
  cells: {},
  expression: '',
  history: [],
  source_expression: '',
  source_result: '',
  source_operands: [],
})
export function validateContent(value) {
  if (
    !value ||
    !Number.isInteger(value.rows) ||
    value.rows < 2 ||
    value.rows > LIMITS.rows ||
    !Number.isInteger(value.columns) ||
    value.columns < 2 ||
    value.columns > LIMITS.columns ||
    !value.cells ||
    Array.isArray(value.cells) ||
    Object.keys(value.cells).length > LIMITS.cells ||
    !Array.isArray(value.history) ||
    value.history.length > LIMITS.history
  )
    throw new Error('Feuille invalide ou trop grande.')
  const text = (input) =>
    typeof input === 'string' && [...input].length <= LIMITS.input
  for (const [ref, cell] of Object.entries(value.cells)) {
    const [row, col] = coordinates(ref)
    if (
      ref.includes('$') ||
      row >= value.rows ||
      col >= value.columns ||
      !cell ||
      !text(cell.input) ||
      !['number', 'currency', 'percent'].includes(cell.format)
    )
      throw new Error('Cellule invalide ou hors feuille.')
  }
  for (const key of ['expression', 'source_expression', 'source_result'])
    if (!text(value[key])) throw new Error('Expression ou métadonnée invalide.')
  if (
    !Array.isArray(value.source_operands) ||
    value.source_operands.length > 128 ||
    value.source_operands.some(
      (s) => typeof s !== 'string' || [...s].length > 128,
    )
  )
    throw new Error('Opérandes invalides.')
  for (const item of value.history)
    if (
      !item ||
      !text(item.expression) ||
      !text(item.result) ||
      typeof item.created_at !== 'string' ||
      !Number.isFinite(Date.parse(item.created_at))
    )
      throw new Error('Historique invalide.')
  if (new TextEncoder().encode(JSON.stringify(value)).length > LIMITS.bytes)
    throw new Error('Document trop volumineux (200 ko maximum).')
  return value
}
export function migrateDocument(doc) {
  if (!doc || !Number.isSafeInteger(doc.schema_version))
    throw new Error('Version du document absente.')
  if (doc.schema_version === 0 && Array.isArray(doc.content?.grid)) {
    const content = emptyContent(),
      grid = doc.content.grid
    content.rows = Math.max(2, grid.length)
    content.columns = Math.max(2, ...grid.map((row) => row.length))
    grid.forEach((row, r) =>
      row.forEach((input, c) => {
        if (input !== '')
          content.cells[address(r, c)] = {
            input: String(input),
            format: 'number',
          }
      }),
    )
    return {
      ...doc,
      schema_version: 1,
      content: validateContent(content),
      dirty: true,
    }
  }
  if (doc.schema_version !== 1)
    throw new Error(
      'Version de document non prise en charge ; original conservé.',
    )
  return { ...doc, content: validateContent(doc.content) }
}
export function newWorkbook(
  content = emptyContent(),
  title = 'Nouvelle feuille',
) {
  return {
    client_key: crypto.randomUUID(),
    title,
    type: 'workbook',
    schema_version: 1,
    revision: 0,
    content: validateContent(content),
    dirty: true,
  }
}
export function convertCalculation(expression, result, history) {
  if (!expression.trim())
    throw new Error('Saisis un calcul avant la conversion.')
  const content = emptyContent()
  content.source_expression = expression
  content.source_result = result
  content.expression = expression
  content.history = structuredClone(history).slice(0, 50)
  const operands = /^[\d\s().+*/\-]+$/.test(expression)
    ? expression.match(/(?:\d+(?:\.\d+)?)/g) || []
    : []
  content.source_operands = operands.length <= 96 ? operands : []
  content.cells = {
    A1: { input: 'Expression originale', format: 'number' },
    B1: {
      input: 'Expression conservée dans les métadonnées',
      format: 'number',
    },
    A2: { input: 'Résultat recalculé', format: 'number' },
    B2: { input: `=${expression.replace(/^=/, '')}`, format: 'number' },
  }
  content.source_operands.forEach((operand, i) => {
    content.cells[address(i + 3, 0)] = {
      input: `Opérande ${i + 1}`,
      format: 'number',
    }
    content.cells[address(i + 3, 1)] = { input: operand, format: 'number' }
  })
  content.rows = Math.max(20, content.source_operands.length + 4)

  return newWorkbook(content, 'Calcul converti')
}
// Quoted strings are never rewritten as references.
export function rewriteReferences(input, transform) {
  if (!input.startsWith('=')) return input
  return input.replace(
    /"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|(?<![A-Za-z0-9_$])\$?[A-Z]\$?[1-9]\d{0,2}(?![A-Za-z0-9_$])/g,
    (token) =>
      token.startsWith('"') || token.startsWith("'") ? token : transform(token),
  )
}
export function shiftedFormula(input, deltaRow, deltaColumn) {
  return rewriteReferences(input, (ref) => {
    const [row, col] = coordinates(ref)
    const r = ref.match(/\$\d/) ? row : row + deltaRow
    const c = ref.startsWith('$') ? col : col + deltaColumn
    if (r < 0 || r >= 100 || c < 0 || c >= 26) return 'REF_DELETED'
    return `${ref.startsWith('$') ? '$' : ''}${String.fromCharCode(65 + c)}${/\$\d/.test(ref) ? '$' : ''}${r + 1}`
  })
}
export function resizeContent(content, rows, columns) {
  const next = structuredClone(content)
  next.rows = rows
  next.columns = columns
  for (const [ref, cell] of Object.entries(next.cells)) {
    const [r, c] = coordinates(ref)
    if (r >= rows || c >= columns) delete next.cells[ref]
    else
      cell.input = rewriteReferences(cell.input, (token) => {
        const [r, c] = coordinates(token)
        return r >= rows || c >= columns ? 'REF_DELETED' : token
      })
  }
  return validateContent(next)
}
export function safeClipboardText(value) {
  const text = String(value)
  return !/^-?\d+(?:\.\d+)?$/.test(text) && /^[=+\-@\t\r]/.test(text)
    ? `'${text}`
    : text
}
