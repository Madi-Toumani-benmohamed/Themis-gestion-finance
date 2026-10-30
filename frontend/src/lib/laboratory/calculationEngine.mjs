import {
  create,
  parseDependencies,
  bignumberDependencies,
  addDependencies,
  subtractDependencies,
  multiplyDependencies,
  divideDependencies,
  unaryMinusDependencies,
  unaryPlusDependencies,
  equalDependencies,
  unequalDependencies,
  largerDependencies,
  largerEqDependencies,
  smallerDependencies,
  smallerEqDependencies,
} from 'mathjs'
import { cents } from '../../components/money.mjs'
import { coordinates, address, validateContent } from './workbookModel.mjs'

const math = create(
  {
    parseDependencies,
    bignumberDependencies,
    addDependencies,
    subtractDependencies,
    multiplyDependencies,
    divideDependencies,
    unaryMinusDependencies,
    unaryPlusDependencies,
    equalDependencies,
    unequalDependencies,
    largerDependencies,
    largerEqDependencies,
    smallerDependencies,
    smallerEqDependencies,
  },
  { number: 'BigNumber', precision: 40 },
)
const operators = new Set([
  'add',
  'subtract',
  'multiply',
  'divide',
  'unaryMinus',
  'unaryPlus',
  'equal',
  'unequal',
  'larger',
  'largerEq',
  'smaller',
  'smallerEq',
])
const functions = new Set([
  'SUM',
  'AVERAGE',
  'MIN',
  'MAX',
  'COUNT',
  'IF',
  'ROUND',
])
class CalculationError extends Error {
  constructor(code, message) {
    super(message)
    this.code = code
  }
}
const fail = (code, message) => {
  throw new CalculationError(code, message)
}
const decimal = (value) => {
  const d = math.bignumber(value)
  if (
    !d.isFinite() ||
    d.abs().gt('1e18') ||
    (!d.isZero() && d.abs().lt('1e-40'))
  )
    fail('#NUM!', 'Valeur hors limites décimales.')
  return d
}
function syntax(expression) {
  if (
    typeof expression !== 'string' ||
    expression.length > 512 ||
    !expression.trim()
  )
    fail('#PARSE!', 'Expression vide ou trop longue.')
  if (expression.includes('REF_DELETED'))
    fail('#REF!', 'Une référence a été supprimée.')
  return expression
    .replace(/^=/, '')
    .replace(/"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^"']+/g, (token) => {
      if (token.startsWith('"') || token.startsWith("'")) return token
      if (token.includes(','))
        fail(
          '#PARSE!',
          'Utilise le point décimal et le point-virgule entre arguments.',
        )
      return token.replaceAll(';', ',')
    })
}
function parse(expression, tick) {
  let node
  try {
    node = math.parse(syntax(expression))
  } catch (e) {
    if (e.code) throw e
    fail('#PARSE!', 'Expression invalide.')
  }
  let count = 0
  function check(n, depth = 0) {
    if (++count > 200 || depth > 32)
      fail('#LIMIT!', 'Expression trop complexe.')
    if (n.isConstantNode) {
      if (typeof n.value === 'string') {
        if (n.value.length > 512) fail('#LIMIT!', 'Texte trop long.')
      } else if (typeof n.value !== 'boolean') decimal(n.value)
    } else if (n.isSymbolNode) {
      if (
        !/^\$?[A-Z]\$?[1-9]\d{0,2}$/.test(n.name) &&
        !['true', 'false'].includes(n.name)
      )
        fail('#NAME?', 'Nom non autorisé.')
    } else if (n.isOperatorNode) {
      if (!operators.has(n.fn) || n.implicit)
        fail('#SYNTAX!', 'Opérateur non pris en charge.')
      if (n.fn === 'divide' && n.args.length !== 2)
        fail('#PARSE!', 'Division invalide.')
    } else if (n.isFunctionNode) {
      if (!n.fn.isSymbolNode || !functions.has(n.fn.name))
        fail('#NAME?', 'Fonction non autorisée.')
      if (n.args.length > 100) fail('#LIMIT!', 'Trop d’arguments.')
      if (n.fn.name === 'IF' && n.args.length !== 3)
        fail('#PARSE!', 'IF attend trois arguments.')
      if (n.fn.name === 'ROUND' && (n.args.length < 1 || n.args.length > 2))
        fail('#PARSE!', 'ROUND attend un ou deux arguments.')
      n.args.forEach((a) => check(a, depth + 1))
      return
    } else if (n.isRangeNode) {
      if (n.step || !n.start.isSymbolNode || !n.end.isSymbolNode)
        fail('#REF!', 'Plage invalide.')
    } else if (!n.isParenthesisNode)
      fail('#SECURITY!', 'Construction non autorisée.')
    n.forEach((child) => check(child, depth + 1))
  }
  check(node)
  tick(count)
  const transformed = node.transform((n) => {
    if (n.isRangeNode)
      return new math.FunctionNode('RANGE', [
        new math.ConstantNode(n.start.name),
        new math.ConstantNode(n.end.name),
      ])
    if (n.isSymbolNode && /^\$?[A-Z]\$?[1-9]\d{0,2}$/.test(n.name))
      return new math.FunctionNode('CELL', [new math.ConstantNode(n.name)])
    if (n.isSymbolNode && ['true', 'false'].includes(n.name))
      return new math.ConstantNode(n.name === 'true')
    return n
  })
  return transformed.compile()
}
const lazyIf = (args, _, scope) => {
  const condition = args[0].compile().evaluate(scope)
  const truth =
    typeof condition === 'boolean' ? condition : !decimal(condition).isZero()
  return args[truth ? 1 : 2].compile().evaluate(scope)
}
lazyIf.rawArgs = true
math.import({ IF: lazyIf })
function normalize(value, format = 'number') {
  if (typeof value === 'string') {
    if (value.length > 512) fail('#LIMIT!', 'Texte résultant trop long.')
    return { type: 'text', value }
  }
  if (typeof value === 'boolean')
    return { type: 'boolean', value: value ? 'TRUE' : 'FALSE' }
  if (!value?.isFinite) fail('#VALUE!', 'Valeur non numérique.')
  const d = decimal(value),
    out = { type: 'number', value: d.toFixed() }
  if (format === 'currency') {
    const amount = cents(d.toDecimalPlaces(2, 4).toFixed(2))
    if (Math.abs(amount) > 999999999999)
      fail('#NUM!', 'Montant hors limites monétaires de Thémis.')
    out.money_cents = amount
  }
  if (format === 'percent')
    out.formatted = `${d.times(100).toDecimalPlaces(4, 4).toFixed()} %`
  return out
}
function session(content, references = true) {
  const cache = new Map(),
    active = new Set()
  let used = 0
  const tick = (n) => {
    used += n
    if (used > 100000) fail('#LIMIT!', 'Budget de calcul dépassé.')
  }
  const key = (ref) => {
    if (!references)
      fail('#REF!', 'Les références appartiennent au mode feuille.')
    const [r, c] = coordinates(ref)
    if (r >= content.rows || c >= content.columns)
      fail('#REF!', 'Référence hors feuille.')
    return address(r, c)
  }
  const cell = (ref) => {
    const id = key(ref)
    if (cache.has(id)) return cache.get(id)
    if (active.has(id)) fail('#CYCLE!', 'Référence circulaire.')
    tick(1)
    active.add(id)
    try {
      const input = content.cells[id]?.input || ''
      const value =
        input === ''
          ? null
          : input.startsWith('=')
            ? evaluate(input)
            : /^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?$/.test(input)
              ? decimal(input)
              : input
      // Bound intermediates before any dependent can multiply/concatenate them.
      if (value !== null) normalize(value)
      cache.set(id, value)
      return value
    } finally {
      active.delete(id)
    }
  }
  const flatten = (args) => args.flatMap((a) => (Array.isArray(a) ? a : [a]))
  const numeric = (args) =>
    flatten(args)
      .filter(
        (v) => v !== null && typeof v !== 'string' && typeof v !== 'boolean',
      )
      .map(decimal)
  const scope = {
    CELL: (ref) => cell(ref) ?? decimal(0),
    RANGE: (start, end) => {
      const [a, b] = coordinates(key(start)),
        [c, d] = coordinates(key(end))
      const values = []
      for (let r = Math.min(a, c); r <= Math.max(a, c); r++)
        for (let col = Math.min(b, d); col <= Math.max(b, d); col++) {
          tick(1)
          values.push(cell(address(r, col)))
        }
      return values
    },
    SUM: (...args) =>
      numeric(args).reduce((a, b) => math.add(a, b), decimal(0)),
    COUNT: (...args) => decimal(numeric(args).length),
    AVERAGE: (...args) => {
      const list = numeric(args)
      if (!list.length) fail('#DIV/0!', 'Aucune valeur numérique.')
      return math.divide(scope.SUM(list), decimal(list.length))
    },
    MIN: (...args) => {
      const list = numeric(args)
      return list.length ? list.reduce((a, b) => (a.lt(b) ? a : b)) : decimal(0)
    },
    MAX: (...args) => {
      const list = numeric(args)
      return list.length ? list.reduce((a, b) => (a.gt(b) ? a : b)) : decimal(0)
    },
    ROUND: (value, digits = decimal(0)) => {
      const n = decimal(digits)
      if (!n.isInteger() || n.lt(-20) || n.gt(20))
        fail('#NUM!', 'Précision ROUND : -20 à 20.')
      const places = n.toNumber(),
        v = decimal(value)
      return places >= 0
        ? v.toDecimalPlaces(places, 4)
        : v
            .div(decimal(10).pow(-places))
            .toDecimalPlaces(0, 4)
            .times(decimal(10).pow(-places))
    },
  }
  function evaluate(expression) {
    try {
      const value = parse(expression, tick).evaluate(scope)
      if (value?.isFinite && !value.isFinite())
        fail('#DIV/0!', 'Division par zéro ou résultat non fini.')
      return value
    } catch (e) {
      if (e.code) throw e
      fail('#VALUE!', 'Valeur ou opération incompatible.')
    }
  }
  return { cell, evaluate, tick }
}
export function evaluateQuick(expression) {
  const s = session({ rows: 100, columns: 26, cells: {} }, false)
  return normalize(s.evaluate(expression))
}
export function evaluateWorkbook(content) {
  validateContent(content)
  const s = session(content),
    results = {}
  for (const [ref, cell] of Object.entries(content.cells)) {
    try {
      const value = s.cell(ref)
      results[ref] =
        value === null
          ? { type: 'empty', value: '' }
          : normalize(value, cell.format)
    } catch (e) {
      results[ref] = {
        type: 'error',
        code: e.code || '#VALUE!',
        message: e.message,
      }
    }
  }
  return results
}
