// Production bundle and synthetic API only.
const { chromium, expect } = require('@playwright/test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const { spawn } = require('node:child_process')
const base = 'http://127.0.0.1:5190'
const manifest = JSON.parse(fs.readFileSync(path.resolve('dist/.vite/manifest.json'), 'utf8'))
const asset = name => manifest[`src/pages/${name}.jsx`].file
async function main() {
  const preview = spawn(process.execPath, ['node_modules/vite/bin/vite.js', 'preview', '--host', '127.0.0.1', '--port', '5190', '--strictPort'], { stdio: 'pipe' })
  let diagnostics = '', browser
  preview.stderr.on('data', data => { diagnostics += data })
  try {
    await expect.poll(async () => {
      if (preview.exitCode != null) throw new Error(diagnostics || 'Preview stopped')
      try { return (await fetch(base)).ok } catch { return false }
    }, { timeout: 15000 }).toBe(true)
    browser = await chromium.launch({ headless: true, channel: 'chromium' })
    const context = await browser.newContext({ reducedMotion: 'reduce' })
    const page = await context.newPage(), scripts = [], errors = []
    let authenticated = false
    page.on('pageerror', error => errors.push(error.message))
    page.on('response', response => { if (new URL(response.url()).pathname.endsWith('.js')) scripts.push(new URL(response.url()).pathname.slice(1)) })
    await page.route('**/api/**', async route => {
      const pathname = new URL(route.request().url()).pathname
      let body = {}, status = 200
      if (pathname === '/api/auth/status') body = { setup_required: false }
      else if (pathname === '/api/auth/login') { authenticated = true; body = { id: 1, username: 'PERFORMANCE TEST', role: 'admin' } }
      else if (pathname === '/api/auth/me') { status = authenticated ? 200 : 401; body = authenticated ? { id: 1, username: 'PERFORMANCE TEST', role: 'admin' } : { error: 'Connexion requise' } }
      else if (pathname === '/api/health') body = { ai_enabled: false }
      else if (pathname === '/api/summary') body = { monthly: [], by_category: [], uncategorized: 0 }
      else if (['/api/accounts', '/api/tags', '/api/admin/users'].includes(pathname)) body = []
      await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
    })
    await page.goto(base)
    await expect(page.getByLabel('Identifiant', { exact: true })).toBeVisible()
    for (const name of ['Dashboard', 'Admin', 'Transactions', 'Analyse']) assert.ok(!scripts.includes(asset(name)), `${name} must not load before login`)
    assert.ok(!scripts.some(file => file.includes('ChartPrimitives-')), 'Charts must not load on the login screen')
    const loginScripts = [...new Set(scripts)]
    await page.getByLabel('Identifiant', { exact: true }).fill('PERFORMANCE TEST')
    await page.getByLabel('Mot de passe', { exact: true }).fill('SYNTHETIC ONLY')
    await page.getByRole('button', { name: /Se connecter/ }).click()
    await expect(page.getByText("Aucune donnée pour l'instant.", { exact: true })).toBeVisible()
    assert.ok(scripts.includes(asset('Dashboard')))
    assert.ok(!scripts.includes(asset('Admin')), 'Admin must remain deferred on the dashboard')
    await page.getByRole('tab', { name: 'Admin', exact: true }).focus()
    await expect.poll(() => scripts.includes(asset('Admin'))).toBe(true)
    await page.getByRole('tab', { name: 'Admin', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Statistiques', exact: true })).toBeVisible()
    await page.getByRole('tab', { name: 'Tableau de bord', exact: true }).click()
    await expect(page.getByText("Aucune donnée pour l'instant.", { exact: true })).toBeVisible()
    assert.equal(scripts.filter(file => file === asset('Dashboard')).length, 1, 'Revisiting must reuse the loaded module')
    await page.route(`**/${asset('Accounts')}`, route => route.abort('failed'))
    await page.getByRole('tab', { name: 'Comptes', exact: true }).click()
    await expect(page.getByRole('alert').filter({ hasText: 'Cette page n’a pas pu être chargée' })).toBeVisible()
    await page.getByRole('button', { name: 'Recharger la page', exact: true }).click()
    await expect(page.getByText("Aucune donnée pour l'instant.", { exact: true })).toBeVisible()
    assert.deepEqual(errors, [])
    const bytes = loginScripts.reduce((sum, file) => sum + fs.statSync(path.join('dist', file)).size, 0)
    console.log(JSON.stringify({ result: 'PASS', login_js_bytes: bytes, login_scripts: loginScripts, checks: 'deferred routes/charts, focus prefetch, module reuse, failed chunk and recovery' }, null, 2))
  } finally { await browser?.close(); preview.kill('SIGTERM') }
}
main().catch(error => { console.error(error); process.exitCode = 1 })
