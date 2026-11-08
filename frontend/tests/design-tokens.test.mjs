import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync, readdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { join, relative } from 'node:path'

const root = fileURLToPath(new URL('../src/', import.meta.url))
const walk = dir => readdirSync(dir, { withFileTypes: true }).flatMap(entry => entry.isDirectory() ? walk(join(dir, entry.name)) : [join(dir, entry.name)])
const files = walk(root).filter(path => /\.(css|jsx|js)$/.test(path))
const css = readFileSync(join(root, 'design-tokens.css'), 'utf8')
const declarations = block => Object.fromEntries([...block.matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)].map(([, name, value]) => [name, value.trim()]))
const light = declarations(css.match(/:root\s*\{([^}]+)\}/)[1])
const dark = { ...light, ...declarations(css.match(/\[data-theme="dark"\]\s*\{([^}]+)\}/)[1]) }

test('global visual tokens have one owner; local variants cannot redefine them', () => {
  for (const path of files.filter(path => path.endsWith('.css') && !path.endsWith('/design-tokens.css'))) {
    const source = readFileSync(path, 'utf8')
    for (const name of Object.keys(declarations(source))) {
      assert.ok(!(name in light), `${relative(root, path)} redefines global token ${name}`)
    }
  }
})

test('CSS variables without fallbacks have a declaration in CSS or React', () => {
  const sources = files.map(path => readFileSync(path, 'utf8')).join('\n')
  const defined = new Set([...sources.matchAll(/(--[\w-]+)\s*['"]?\s*:/g)].map(match => match[1]))
  for (const path of files.filter(path => path.endsWith('.css'))) {
    for (const [, name] of readFileSync(path, 'utf8').matchAll(/var\((--[\w-]+)\)/g)) {
      assert.ok(defined.has(name), `${relative(root, path)} uses undefined ${name}`)
    }
  }
})

const resolve = (tokens, name) => tokens[name].replace(/var\((--[\w-]+)\)/g, (_, next) => resolve(tokens, next))
const luminance = hex => {
  assert.match(hex, /^#[a-f\d]{6}$/i)
  return hex.slice(1).match(/../g).map(value => parseInt(value, 16) / 255)
    .map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4)
    .reduce((sum, value, i) => sum + value * [.2126, .7152, .0722][i], 0)
}
for (const [theme, tokens] of Object.entries({ light, dark })) {
  test(`shared text/surface pairs meet contrast requirements in ${theme}`, () => {
    for (const [foreground, background, minimum] of [
      ['--ink', '--paper', 4.5], ['--body', '--page', 4.5], ['--muted', '--paper', 4.5],
      ['--on-action', '--action', 4.5], ['--on-action', '--action-hover', 4.5],
      ['--nav-text', '--nav-bg', 4.5], ['--nav-focus', '--nav-bg', 3],
      ['--pos-ink', '--pos-soft', 4.5], ['--neg-ink', '--neg-soft', 4.5], ['--warn-ink', '--warn-soft', 4.5],
    ]) {
      const values = [foreground, background].map(name => luminance(resolve(tokens, name)))
      const ratio = (Math.max(...values) + .05) / (Math.min(...values) + .05)
      assert.ok(ratio >= minimum, `${foreground}/${background}: ${ratio.toFixed(2)} < ${minimum}`)
    }
  })
}
