// Synthetic data only. Run against Vite with: node tests/cinematic.browser.cjs
// Optional: THEMIS_CINEMATIC_ARTIFACTS=/tmp/themis-cinematic BASE_URL=http://127.0.0.1:5173
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const { chromium } = require('playwright')
const { expect } = require('@playwright/test')
const { AxeBuilder } = require('@axe-core/playwright')

const baseURL = process.env.BASE_URL || 'http://127.0.0.1:5176'
const artifactDir = process.env.THEMIS_CINEMATIC_ARTIFACTS
const now = new Date()
const month = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
const prior = new Date(now.getFullYear(), now.getMonth() - 1, 1)
const previous = `${prior.getFullYear()}-${String(prior.getMonth() + 1).padStart(2, '0')}`
const normalize = value => value.replace(/[\u00a0\u202f]/g, ' ').trim()
const money = value => normalize(new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR' }).format(value))
const monthly = [
  { month: previous, income: 3000.11, expenses: 2600.10, is_current: false, status: 'incomplete', days_observed: 21, days_in_month: 30 },
  { month, income: 3456.78, expenses: 2222.22, is_current: true, status: 'current', days_observed: 21, days_in_month: 31 },
]
const categories = monthly.flatMap(row => [
  { month: row.month, category: 'Logement TEST', total: 1200.11, color: '#6d65d9' },
  { month: row.month, category: 'Vie quotidienne TEST', total: +(row.expenses - 1200.11).toFixed(2), color: '#a09ae8' },
])
let layout = [
  { id: 'test-flux', type: 'flux', size: 'full', config: { months: 12 } },
  { id: 'test-kpi', type: 'kpi', size: 'full', config: {} },
  { id: 'test-categories', type: 'categories', size: 'half', config: {} },
  { id: 'test-trend', type: 'trend', size: 'half', config: { category: 'Logement TEST', months: 12 } },
]
let omittedMonth = null, invalidCategories = false, failSummary = false
let layoutGate = null, releaseLayout, layoutWrites = 0
let holdPrevious = false
let releasePrevious
let previousRequested
let previousGate

async function mockAPI(context) {
  await context.route('**/api/**', async route => {
    const request = route.request()
    const url = new URL(request.url())
    const selected = url.searchParams.get('month') || month
    const row = monthly.find(item => item.month === selected)
    if (url.pathname === '/api/summary' && failSummary && url.searchParams.has('month')) return route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: 'Indicateurs TEST indisponibles' }) })
    if (holdPrevious && url.searchParams.get('month') === previous && ['/api/summary', '/api/hero', '/api/journal', '/api/remaining'].includes(url.pathname)) {
      previousRequested?.()
      await previousGate
    }
    let body
    switch (url.pathname) {
      case '/api/auth/status': body = { setup_required: false }; break
      case '/api/auth/me': body = { id: 1, username: 'CINEMATIC TEST', role: 'user' }; break
      case '/api/health': body = { status: 'ok', ai_enabled: false, ai_provider: 'none' }; break
      case '/api/accounts': body = [{ id: 1, name: 'COMPTE SYNTHETIQUE', kind: 'courant', balance: null }]; break
      case '/api/summary': body = { monthly: monthly.filter(item => item.month <= selected && item.month !== omittedMonth), by_category: invalidCategories ? categories.map(item => ({ ...item, total: 1.001 })) : categories, uncategorized: 0 }; break
      case '/api/hero':
        if (!row || selected === omittedMonth) { body = { available: false, month: selected }; break }
        body = {
        available: true, month: row.month, period: { is_current: row.is_current }, period_complete: false,
        income: row.income, expenses: row.expenses, net: +(row.income - row.expenses).toFixed(2), tone: 'neutral',
        headline: 'Les flux disponibles restent provisoires — TEST.',
        coverage: { last_tx: `${row.month}-21`, stale: false },
        points: [{ kind: 'coverage', tone: 'attention', text: 'Couverture documentaire partielle — TEST.' }],
      }; break
      case '/api/remaining': body = { available: false }; break
      case '/api/journal': body = { events: [{ title: 'Relevé synthétique reçu', detail: 'Jeu de données de test uniquement.', tone: 'neutral', importance: 65 }] }; break
      case '/api/settings/dashboard':
        if (request.method() === 'PUT') {
          layoutWrites++
          if (layoutGate) await layoutGate
          layout = request.postDataJSON()
        }
        body = { value: layout }; break
      case '/api/categories':
      case '/api/domains': body = [{ id: 1, name: 'Logement TEST', color: '#6d65d9' }]; break
      case '/api/budgets':
      case '/api/tags': body = []; break
      default:
        throw new Error(`Unexpected API call: ${request.method()} ${url.pathname}`)
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) })
  })
}

async function settleMotion(page) {
  await page.evaluate(async () => {
    await document.fonts.ready
    await new Promise(requestAnimationFrame)
  })
  await expect.poll(() => page.evaluate(() => document.getAnimations().filter(animation => animation.playState === 'running').length)).toBe(0)
}

async function screenshot(page, name) {
  if (!artifactDir) return
  fs.mkdirSync(artifactDir, { recursive: true })
  await settleMotion(page)
  await page.screenshot({ path: path.join(artifactDir, name + '.png'), fullPage: true })
}

async function noHorizontalOverflow(page) {
  assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), 'Page must not overflow horizontally')
}

async function main() {
  const executablePath = process.env.THEMIS_CHROMIUM_PATH || (fs.existsSync('/usr/bin/chromium') ? '/usr/bin/chromium' : null)
  const browser = await chromium.launch({ headless: true, ...(executablePath ? { executablePath } : { channel: 'chromium' }) })
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'fr-FR', reducedMotion: 'no-preference' })
    await mockAPI(context)
    await context.addInitScript(() => {
      Object.defineProperty(navigator, 'hardwareConcurrency', { configurable: true, get: () => 4 })
      window.__drawnPaths = []
      const animate = Element.prototype.animate
      Element.prototype.animate = function (frames, options) {
        const animation = animate.call(this, frames, options)
        if (this.matches('[data-draw], .recharts-line-curve, .recharts-area-curve')) {
          animation.finished.then(() => window.__drawnPaths.push({ className: this.getAttribute('class'), length: frames[0]?.strokeDashoffset, end: frames.at(-1)?.strokeDashoffset })).catch(() => {})
        }
        return animation
      }
      window.__displayedMoney = []
      // Record every committed amount, including short-lived intermediate values.
      new MutationObserver(() => {
        document.querySelectorAll('.financial-summary-amount, .kpi-value, .widget-kpi .value').forEach(node => {
          if (node.textContent.includes('€')) window.__displayedMoney.push(node.textContent)
        })
      }).observe(document, { subtree: true, childList: true, characterData: true })
    })
    const page = await context.newPage()
    const errors = []
    page.on('pageerror', error => { errors.push(error.message); console.error('Desktop page error:', error.message) })
    await page.goto(baseURL)
    if (process.env.THEMIS_CINEMATIC_BASELINE === '1') {
      await expect(page.locator('.financial-summary-amount')).toHaveText(/1[\s\u202f]234,56/)
      await expect(page.locator('.widget-kpi .value').first()).toHaveText(/3[\s\u202f]456,78/)
      await screenshot(page, 'dashboard-baseline-light')
      await page.getByRole('button', { name: /Passer en mode sombre/ }).click()
      await screenshot(page, 'dashboard-baseline-dark')
      console.log('PASS baseline screenshots saved; cinematic assertions intentionally not run.')
      await context.close()
      return
    }
    await expect(page.locator('.cinematic-dashboard')).toBeVisible()
    await expect(page.locator('html')).toHaveAttribute('data-motion', 'full')
    const hero = page.locator('.cinematic-header')
    await hero.hover({ position: { x: 50, y: 80 } })
    await expect.poll(() => hero.locator('.scene-orbit-system').evaluate(node => node.parentElement.parentElement.style.getPropertyValue('--scene-shift-x'))).not.toBe('0px')
    const frameSamplePromise = page.evaluate(() => new Promise(resolve => {
      const deltas = []
      let last
      const sample = time => {
        if (last !== undefined) deltas.push(time - last)
        last = time
        if (deltas.length < 90) requestAnimationFrame(sample)
        else {
          const sorted = [...deltas].sort((a, b) => a - b)
          resolve({ frames: deltas.length, medianMs: sorted[Math.floor(sorted.length / 2)], p95Ms: sorted[Math.floor(sorted.length * .95)], maximumMs: sorted.at(-1) })
        }
      }
      requestAnimationFrame(sample)
    }))
    await expect(page.locator('.cinematic-scene')).toHaveAttribute('aria-hidden', 'true')
    await expect(page.locator('.financial-summary-amount')).toHaveText(/1[\s\u202f]234,56/)
    await expect(page.getByText('Couverture documentaire partielle — TEST.', { exact: true })).toBeVisible()
    await expect(page.locator('.financial-summary-context')).toContainText('Données provisoires')
    assert.ok(await page.locator('.motion-surface').count(), 'Reusable depth surfaces must be present')
    const frameSample = await frameSamplePromise
    await expect.poll(() => page.evaluate(() => window.__drawnPaths.filter(path => path.length > 0 && path.end === 0).length)).toBeGreaterThan(0)
    await page.locator('.widget-kpi').scrollIntoViewIfNeeded()
    await expect(page.locator('.widget-kpi .value').first()).toHaveText(/3[\s\u202f]456,78/)
    const incomeSeries = page.locator('.widget-flux .cinematic-chart-legend').getByRole('button', { name: 'Revenus', exact: true })
    await incomeSeries.click()
    await expect(incomeSeries).toHaveAttribute('aria-pressed', 'false')
    await incomeSeries.click()
    await expect(incomeSeries).toHaveAttribute('aria-pressed', 'true')
    await screenshot(page, 'dashboard-cinematic-light')
    await page.locator('.dashboard-heading').scrollIntoViewIfNeeded()
    if (artifactDir) await page.screenshot({ path: path.join(artifactDir, 'dashboard-cinematic-light-viewport.png') })
    console.log('Headless Chromium frame sample (environment-specific, not a device FPS guarantee):', JSON.stringify(frameSample))
    if (artifactDir) fs.writeFileSync(path.join(artifactDir, 'frame-sample.json'), JSON.stringify(frameSample, null, 2))
    // Accessibility must not depend on optional screenshot capture settling motion.
    await settleMotion(page)
    const lightResults = await new AxeBuilder({ page }).include('main').withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
    if (artifactDir) fs.writeFileSync(path.join(artifactDir, 'axe-light.json'), JSON.stringify(lightResults, null, 2))
    assert.deepEqual(lightResults.violations.map(item => ({ id: item.id, impact: item.impact, nodes: item.nodes.map(node => node.target) })), [], 'Light dashboard must pass targeted Axe checks')

    // Controls remain usable during motion; customization must still persist.
    const customize = page.getByRole('button', { name: 'Personnaliser', exact: true })
    await customize.focus()
    await page.keyboard.press('Enter')
    await expect(page.getByText('Ajouter un widget :', { exact: true })).toBeVisible()
    const resizeFlux = page.locator('.widget-flux').getByTitle('Demi / pleine largeur', { exact: true })
    layoutGate = new Promise(resolve => { releaseLayout = resolve })
    const writesBefore = layoutWrites
    await resizeFlux.click()
    await expect.poll(() => layoutWrites).toBe(writesBefore + 1)
    await expect(page.locator('.widget-flux')).toHaveClass(/\bhalf\b/)
    await resizeFlux.click()
    await expect(page.locator('.widget-flux')).toHaveClass(/\bfull\b/)
    assert.equal(layoutWrites, writesBefore + 1, 'Concurrent layout writes must be serialized')
    layoutGate = null
    releaseLayout()
    await expect.poll(() => layoutWrites).toBe(writesBefore + 2)
    await expect(page.getByRole('status').filter({ hasText: 'Disposition enregistrée' })).toBeVisible()
    assert.equal(layout.find(widget => widget.id === 'test-flux').size, 'full')
    await resizeFlux.click()
    await expect(page.locator('.widget-flux')).toHaveClass(/\bhalf\b/)
    await expect(page.getByRole('status').filter({ hasText: 'Disposition enregistrée' })).toBeVisible()
    assert.equal(layout.find(widget => widget.id === 'test-flux').size, 'half')
    await resizeFlux.click()
    await expect(page.locator('.widget-flux')).toHaveClass(/\bfull\b/)
    await expect(page.getByRole('status').filter({ hasText: 'Disposition enregistrée' })).toBeVisible()
    assert.equal(layout.find(widget => widget.id === 'test-flux').size, 'full')
    await page.getByRole('button', { name: 'Terminer', exact: true }).click()
    const plane = page.getByRole('button', { name: /vue (plane|2d)/i })
    await plane.scrollIntoViewIfNeeded()
    await plane.focus()
    await page.keyboard.press('Enter')
    await expect(plane).toHaveAttribute('aria-pressed', 'true')

    await page.getByRole('button', { name: 'Mois précédent', exact: true }).click()
    await expect(page.locator('.financial-summary-amount')).toHaveText(/400,01/)
    await expect(page.locator('.kpi-value').first()).toHaveText('—')
    await expect(page.locator('.kpi-label').first()).toHaveText('Variation des revenus')
    await expect(page.locator('.widget-kpi .value').first()).toHaveText(/3[\s\u202f]000,11/)
    await expect(page.locator('.financial-summary-context')).toContainText('Couverture insuffisante')
    await expect(page.locator('.financial-summary-status')).toContainText('Données incomplètes')
    assert.equal(await page.locator('.month-selector-select').inputValue(), previous)

    // A late previous-month response must not overwrite the newly selected month.
    await page.getByRole('button', { name: 'Retour au mois en cours', exact: true }).click()
    await expect(page.locator('.financial-summary-amount')).toHaveText(/1[\s\u202f]234,56/)
    previousGate = new Promise(resolve => { releasePrevious = resolve })
    const requested = new Promise(resolve => { previousRequested = resolve })
    holdPrevious = true
    await page.getByRole('button', { name: 'Mois précédent', exact: true }).click()
    await requested
    await page.getByRole('button', { name: 'Retour au mois en cours', exact: true }).click()
    await expect(page.locator('.financial-summary-amount')).toHaveText(/1[\s\u202f]234,56/)
    const staleResponses = Promise.all(['/api/summary', '/api/hero', '/api/journal', '/api/remaining'].map(endpoint => page.waitForResponse(response => new URL(response.url()).pathname === endpoint && new URL(response.url()).searchParams.get('month') === previous)))
    holdPrevious = false
    releasePrevious()
    await staleResponses
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
    await expect(page.locator('.financial-summary-amount')).toHaveText(/1[\s\u202f]234,56/)
    await expect(page.locator('.kpi-value').first()).toHaveText('—')
    assert.equal(await page.locator('.month-selector-select').inputValue(), month)

    const allowedMoney = new Set(monthly.flatMap(row => [row.income, row.expenses, +(row.income - row.expenses).toFixed(2)]).map(money))
    const recorded = await page.evaluate(() => window.__displayedMoney)
    assert.ok(recorded.length > 0, 'Amount observer must actually record rendered values')
    assert.deepEqual([...new Set(recorded.map(normalize))].filter(value => !allowedMoney.has(value)), [], 'Financial figures must never show interpolated amounts')

    // Missing or invalid data must not borrow another month's figures or crash the page.
    omittedMonth = month
    await page.reload()
    await expect(page.getByText('Aucun indicateur observé pour cette période.', { exact: true })).toBeVisible()
    await expect(page.locator('.financial-summary-amount')).toHaveCount(0)
    await expect(page.locator('.kpi-row')).toHaveCount(0)
    omittedMonth = null
    invalidCategories = true
    await page.reload()
    await expect(page.getByRole('alert').filter({ hasText: 'Répartition indisponible' })).toBeVisible()
    await expect(page.locator('.financial-summary-amount')).toContainText('234,56')
    invalidCategories = false
    failSummary = true
    await page.reload()
    await expect(page.getByRole('alert').filter({ hasText: 'Indicateurs indisponibles' })).toBeVisible()
    failSummary = false
    await page.reload()
    await expect(page.locator('.financial-summary-amount')).toContainText('234,56')
    await expect.poll(() => page.evaluate(() => document.getAnimations().filter(animation => animation.playState === 'running').length)).toBe(0)
    await page.locator('.widget-kpi').scrollIntoViewIfNeeded()
    await expect.poll(() => page.locator('.widget-kpi').evaluate(node => node.getAnimations({ subtree: true }).filter(animation => animation.playState === 'running').length)).toBe(0)

    // Switching tabs preserves focus and does not replay the full introduction.
    const accounts = page.getByRole('tab', { name: 'Comptes', exact: true })
    await accounts.focus()
    await page.keyboard.press('Enter')
    await expect(page.getByRole('heading', { name: 'Mes comptes', exact: true })).toBeVisible()
    await expect(accounts).toBeFocused()
    await page.getByRole('tab', { name: 'Tableau de bord', exact: true }).click()
    await expect(page.locator('.cinematic-dashboard')).toHaveAttribute('data-entrance', 'settled')
    await expect(page.locator('.financial-summary-amount')).toHaveText(/1[\s\u202f]234,56/)
    await expect(page.locator('.cinematic-summary')).toHaveAttribute('data-revealed', 'false')
    await expect(page.locator('.widget-kpi')).toHaveAttribute('data-revealed', 'false')

    const motion = page.getByLabel('Animations de l’interface', { exact: true })
    await motion.selectOption('reduced')
    await expect(page.locator('html')).toHaveAttribute('data-motion', 'reduced')
    await motion.selectOption('full')
    await expect(page.locator('html')).toHaveAttribute('data-motion', 'full')
    await motion.selectOption('reduced')
    await expect(page.locator('html')).toHaveAttribute('data-motion', 'reduced')
    await motion.selectOption('off')
    await expect(page.locator('html')).toHaveAttribute('data-motion', 'off')
    assert.equal(await page.locator('.cinematic-scene').evaluate(node => node.getAnimations({ subtree: true }).filter(animation => animation.playState === 'running').length), 0, 'Off mode must stop ambient animations')
    await page.reload()
    await expect(page.locator('html')).toHaveAttribute('data-motion', 'off')
    await expect(page.locator('.financial-summary-amount')).toHaveText(/1[\s\u202f]234,56/)
    await motion.selectOption('auto')
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await expect(page.locator('html')).toHaveAttribute('data-motion', 'reduced')
    assert.equal(await page.locator('.cinematic-scene').evaluate(node => node.getAnimations({ subtree: true }).filter(animation => animation.playState === 'running').length), 0, 'Reduced motion must stop continuous decoration')
    await page.getByRole('button', { name: /Passer en mode sombre/ }).click()
    await screenshot(page, 'dashboard-cinematic-dark-reduced')
    await noHorizontalOverflow(page)

    await settleMotion(page)
    const results = await new AxeBuilder({ page }).include('main').withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
    if (artifactDir) fs.writeFileSync(path.join(artifactDir, 'axe.json'), JSON.stringify(results, null, 2))
    assert.deepEqual(results.violations.map(item => ({ id: item.id, impact: item.impact, nodes: item.nodes.map(node => node.target) })), [], 'Dark dashboard must pass targeted Axe checks')

    const mobileContext = await browser.newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, locale: 'fr-FR' })
    await mockAPI(mobileContext)
    const mobile = await mobileContext.newPage()
    mobile.on('pageerror', error => { errors.push(error.message); console.error('Mobile page error:', error.message) })
    await mobile.goto(baseURL)
    await expect(mobile.locator('.financial-summary-amount')).toHaveText(/1[\s\u202f]234,56/)
    await noHorizontalOverflow(mobile)
    const touchSurface = mobile.locator('.motion-surface').first()
    await touchSurface.dispatchEvent('pointerenter', { pointerType: 'touch', clientX: 10, clientY: 10 })
    await touchSurface.dispatchEvent('pointermove', { pointerType: 'touch', clientX: 100, clientY: 100 })
    await mobile.evaluate(() => new Promise(resolve => requestAnimationFrame(resolve)))
    assert.equal(await touchSurface.evaluate(node => node.style.getPropertyValue('--tilt-x')), '', 'Touch pointers must not activate tilt')
    assert.equal(await touchSurface.evaluate(node => node.style.getPropertyValue('--tilt-y')), '', 'Touch pointers must not activate tilt')
    await mobile.locator('.widget-categories').scrollIntoViewIfNeeded()
    await noHorizontalOverflow(mobile)
    await screenshot(mobile, 'dashboard-cinematic-mobile')
    await mobile.locator('.dashboard-heading').scrollIntoViewIfNeeded()
    await mobile.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
    if (artifactDir) await mobile.screenshot({ path: path.join(artifactDir, 'dashboard-cinematic-mobile-viewport.png') })
    await settleMotion(mobile)
    const mobileResults = await new AxeBuilder({ page: mobile }).include('main').withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
    if (artifactDir) fs.writeFileSync(path.join(artifactDir, 'axe-mobile.json'), JSON.stringify(mobileResults, null, 2))
    assert.deepEqual(mobileResults.violations.map(item => ({ id: item.id, impact: item.impact, nodes: item.nodes.map(node => node.target) })), [], 'Mobile dashboard must pass targeted Axe checks')
    for (const width of [320, 768, 1024]) {
      await mobile.setViewportSize({ width, height: 900 })
      await noHorizontalOverflow(mobile)
      assert.ok(await mobile.locator('.cinematic-header').evaluate(node => {
        const bounds = node.getBoundingClientRect()
        return [...node.querySelectorAll('h1,.month-selector')].every(child => { const box = child.getBoundingClientRect(); return box.top >= bounds.top && box.bottom <= bounds.bottom })
      }), 'Header content must remain on its background at every width')
    }
    // An intentionally empty layout must survive reloads instead of restoring defaults.
    layout = []
    await page.reload()
    await expect(page.locator('.financial-summary-amount')).toContainText('234,56')
    await expect(page.locator('.widget-grid > .widget')).toHaveCount(0)
    assert.deepEqual(errors, [], 'No uncaught browser errors')
    console.log('PASS cinematic dashboard: exact amounts, month, navigation, preferences, keyboard, mobile and Axe.')
    await mobileContext.close()
    await context.close()
  } finally {
    await browser.close()
  }
}

main().catch(error => { console.error(error); process.exitCode = 1 })
