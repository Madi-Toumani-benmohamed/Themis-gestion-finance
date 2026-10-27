// Synthetic, fully intercepted API. No account or banking service is contacted.
const { chromium, expect } = require('@playwright/test')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const baseline = process.env.THEMIS_MOTION_BASELINE === '1'
const output = process.env.THEMIS_MOTION_OUTPUT || '/tmp/themis-motion'
const monthly = Array.from({ length: 18 }, (_, i) => ({ month: new Date(Date.UTC(2025, 4 + i, 1)).toISOString().slice(0, 7), income: 3249.47, expenses: i === 17 ? 2367.18 : 2200 + (i % 5) * 87.14, is_current: i === 17, status: i === 17 ? 'current' : 'complete', days_observed: i === 17 ? 21 : 30, days_in_month: 31 }))
const by_category = monthly.flatMap(m => [{ month: m.month, category: 'Logement', color: '#7867c6', total: 1100 }, { month: m.month, category: 'Alimentation', color: '#408b7c', total: 400 }, { month: m.month, category: 'Autres dépenses', color: '#ba874b', total: m.expenses - 1500 }])
const summary = month => ({ monthly: monthly.filter(m => !month || m.month <= month), by_category, uncategorized: 0 })
const hero = month => {
  const row = monthly.find(m => m.month === month)
  return { available: !!row, month, income: row?.income, expenses: row?.expenses, net: row ? +(row.income - row.expenses).toFixed(2) : null, tone: 'neutral', headline: 'Les flux observés restent positifs sur la période.', period: { is_current: month === '2026-10' }, period_complete: month !== '2026-10', coverage: { last_tx: `${month}-21`, last_import: `${month}-22`, stale: false }, points: [{ kind: 'coverage', tone: 'attention', text: 'Le mois en cours est encore partiel.' }, { kind: 'category', tone: 'neutral', text: 'Le logement représente le premier poste observé.' }] }
}
const simulation = { available: true, income: 3249.47, observed_income: 3249.47, base_spend: 2367.18, current_net: 882.29, reference_complete: false, reference_periods: [{ month: '2026-09', completeness: 'unknown' }], categories: [{ category: 'Logement', before: 1100 }, { category: 'Autres dépenses', before: 1267.18 }] }
async function settle(page) {
  await page.bringToFront()
  await page.evaluate(() => document.fonts.ready)
  try {
    await expect.poll(() => page.evaluate(() => document.getAnimations().filter(a => a.playState === 'running' && Number.isFinite(a.effect?.getComputedTiming().endTime)).length), { timeout: 10000 }).toBe(0)
  } catch (error) {
    console.error(await page.evaluate(() => ({ visibility: document.visibilityState, now: performance.now(), timeline: document.timeline.currentTime, animations: document.getAnimations().slice(0, 6).map(a => ({ target: a.effect?.target?.className, current: a.currentTime, start: a.startTime, timing: a.effect?.getComputedTiming() })) })))
    throw error
  }
}
async function measure(page, action) {
  await page.evaluate(() => {
    window.__motionSample = { frames: [], tasks: [] }
    const sample = window.__motionSample
    sample.observer = new PerformanceObserver(list => list.getEntries().forEach(e => sample.tasks.push(e.duration)))
    sample.observer.observe({ type: 'longtask', buffered: false })
    window.__motionDone = new Promise(resolve => {
      let previous
      const tick = time => {
        if (previous) sample.frames.push(time - previous)
        previous = time
        if (sample.frames.length < 180) requestAnimationFrame(tick)
        else { sample.observer.disconnect(); resolve() }
      }
      requestAnimationFrame(tick)
    })
  })
  await action()
  return page.evaluate(async () => {
    await window.__motionDone
    const { frames, tasks } = window.__motionSample
    const sorted = [...frames].sort((a, b) => a - b)
    return { frames: frames.length, median_ms: sorted[90], p95_ms: sorted[171], max_ms: sorted.at(-1), over_33ms: frames.filter(n => n > 33.5).length, long_tasks: tasks.length, longest_task_ms: Math.max(0, ...tasks) }
  })
}
async function main() {
  fs.mkdirSync(output, { recursive: true })
  const browser = await chromium.launch({ headless: true, channel: 'chromium' })
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: 'fr-FR' })
    const page = await context.newPage()
    const errors = [], warnings = []
    page.on('pageerror', e => errors.push(e.message))
    page.on('console', e => { if (e.type() === 'error') warnings.push(e.text()) })
    // Freeze calendar dates only. Playwright's clock retains a performance.now()
    // origin across reloads, unlike document.timeline, which delays native animations.
    await page.addInitScript(() => {
      const OriginalDate = Date, fixed = Date.parse('2026-10-26T12:00:00Z')
      window.Date = class extends OriginalDate {
        constructor(...args) { super(...(args.length ? args : [fixed])) }
        static now() { return fixed }
      }
      if (!localStorage.getItem('themis-theme')) localStorage.setItem('themis-theme', 'light')
    })
    let held = null, heldSummary = null, holdMonth = null, overrideHero = null, omittedMonth = null, failHero = false, empty = false
    await page.route('**/api/**', async route => {
      const url = new URL(route.request().url()), path = url.pathname, month = url.searchParams.get('month') || '2026-10'
      let data = {}
      if (path === '/api/auth/status') data = { setup_required: false }
      else if (path === '/api/auth/me') data = { id: 1, username: 'DÉMO SYNTHÉTIQUE', role: 'admin' }
      else if (path === '/api/summary') {
        if (url.searchParams.get('month') === holdMonth && holdMonth) { heldSummary = route; return }
        data = summary(url.searchParams.get('month'))
        data.monthly = empty ? [] : data.monthly.filter(row => row.month !== omittedMonth)
      }
      else if (path === '/api/hero') {
        if (month === holdMonth) { held = route; return }
        if (failHero) return route.fulfill({ status: 503, contentType: 'application/json', body: JSON.stringify({ error: 'Synthèse indisponible TEST' }) })
        data = omittedMonth === month ? { available: false, month } : overrideHero ? { ...overrideHero, month } : hero(month)
      }
      else if (path === '/api/settings/dashboard') data = { value: [ { id: 'flux', type: 'flux', size: 'full', config: {} }, { id: 'categories', type: 'categories', size: 'half', config: {} }, { id: 'solde', type: 'solde', size: 'half', config: { months: 12 } } ] }
      else if (path === '/api/remaining') data = { available: false }
      else if (path === '/api/journal') data = { events: [] }
      else if (path === '/api/anomalies') data = { anomalies: [] }
      else if (path === '/api/simulate') data = simulation
      else if (['/api/accounts', '/api/transactions', '/api/categories', '/api/credits', '/api/statements', '/api/budgets', '/api/tags', '/api/subscriptions'].includes(path)) data = []
      return route.fulfill({ contentType: 'application/json', body: JSON.stringify(data) })
    })
    const metrics = {}
    await page.goto(process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176')
    await expect(page.locator('.financial-summary-amount')).toContainText('882,29')
    await settle(page)
    await expect(page.locator('.donut-wrap .recharts-pie-sector')).toHaveCount(3)
    await page.screenshot({ path: `${output}/dashboard-light.png` })
    const target = page.getByLabel('Sélectionner un mois')
    metrics.period = await measure(page, async () => {
      await target.selectOption('2026-09')
      await expect(page.locator('.financial-summary h2')).toHaveText('septembre 2026')
    })
    const flux = page.locator('.flux-widget')
    await flux.scrollIntoViewIfNeeded()
    await expect(flux.locator('.recharts-line-curve')).toHaveCount(1)
    await settle(page)
    if (!baseline) {
      // The SVG is kept through brush interactions; actual data geometry remains untouched.
      await flux.locator('.recharts-line-curve').evaluate(el => { el.dataset.identity = 'original' })
      const bounds = await flux.locator('.recharts-brush-traveller').first().boundingBox()
      await page.mouse.move(bounds.x + 4, bounds.y + 8)
      await page.mouse.down(); await page.mouse.move(bounds.x + 40, bounds.y + 8, { steps: 6 }); await page.mouse.up()
      await expect(flux.locator('.recharts-line-curve')).toHaveAttribute('data-identity', 'original')
      assert.equal(await flux.locator('.recharts-line-curve').evaluate(el => el.getAnimations().length), 0, 'brush must not replay drawing')
      holdMonth = '2026-08'
      await target.selectOption(holdMonth)
      await expect.poll(() => !!held).toBe(true)
      await expect.poll(() => !!heldSummary).toBe(true)
      await expect(page.locator('.financial-summary')).toHaveAttribute('data-period', '2026-09')
      await expect(page.locator('.motion-period-state').first()).toContainText('septembre 2026')
      await expect(page.locator('.kpi-row')).toHaveAttribute('data-period', '2026-09')
      await expect(page.locator('.kpi-row')).toHaveAttribute('aria-busy', 'true')
      await target.selectOption('2026-10')
      await expect(page.locator('.financial-summary')).toHaveAttribute('data-period', '2026-10')
      await held.fulfill({ contentType: 'application/json', body: JSON.stringify(hero('2026-08')) }); held = null; holdMonth = null
      await heldSummary.fulfill({ contentType: 'application/json', body: JSON.stringify(summary('2026-08')) }); heldSummary = null
      await settle(page)
      await expect(page.locator('.financial-summary')).toHaveAttribute('data-period', '2026-10')
      await expect(page.locator('.kpi-row')).toHaveAttribute('data-period', '2026-10')
      // Exact text during the animation: absent, zero, signed, decimal amounts.
      for (const [net, text] of [[null, 'Indisponible'], [0, '0,00'], [-1234.56, '-1 234,56'], [882.29, '882,29']]) {
        overrideHero = { ...hero('2026-09'), net }
        await target.selectOption('2026-09')
        await expect(page.locator('.financial-summary-amount')).toContainText(text)
        await target.selectOption('2026-10')
        await expect(page.locator('.financial-summary')).toHaveAttribute('data-period', '2026-10')
      }
      overrideHero = null
      // Transport errors and absent periods never keep an old amount under a new date.
      const warningStart = warnings.length
      failHero = true
      await target.selectOption('2026-09')
      await expect(page.locator('.financial-summary')).toContainText('indisponible')
      await expect(page.locator('.financial-summary-amount')).toHaveCount(0)
      failHero = false
      await target.selectOption('2026-10')
      await expect(page.locator('.financial-summary-amount')).toContainText('882,29')
      const expectedWarnings = warnings.splice(warningStart)
      assert.ok(expectedWarnings.every(w => w.includes('503 (Service Unavailable)')), 'only the requested synthetic network error is expected')
    }
    metrics.navigation = await measure(page, async () => {
      await page.getByRole('tab', { name: 'Simulation', exact: true }).click()
      await expect(page.locator('.scenario-verdict strong')).toContainText('882,29')
    })
    await settle(page)
    await page.screenshot({ path: `${output}/simulation-light.png` })
    if (!baseline) {
      await page.locator('.scenario-category summary').first().click()
      await page.getByRole('spinbutton', { name: 'Réduction de Logement en pourcentage', exact: true }).fill('20')
      await expect(page.locator('.scenario-verdict strong')).toContainText('1 102,29')
      await page.evaluate(() => { const tab = name => [...document.querySelectorAll('.sidebar-nav button')].find(b => b.textContent.includes(name)); tab('Tableau de bord').click(); tab('Simulation').click(); tab('Tableau de bord').click() })
      await expect(page.locator('main')).toHaveAttribute('data-page', 'dashboard')
      await expect(page.locator('main')).toHaveCount(1)
      await expect(page.locator('.financial-summary-amount')).toContainText('882,29')
      await page.emulateMedia({ reducedMotion: 'reduce' })
      await expect(page.locator('.motion-number-reel')).toHaveCount(0)
      await target.selectOption('2026-09')
      await expect(page.locator('.financial-summary h2')).toHaveText('septembre 2026')
      await settle(page)
      assert.equal(await page.locator('.motion-chart').evaluateAll(els => els.flatMap(el => el.getAnimations({ subtree: true })).filter(a => a.playState === 'running').length), 0)
      const audit = await new AxeBuilder({ page }).include('main').analyze()
      assert.deepEqual(audit.violations, [], 'Dashboard accessibility')
      await page.screenshot({ path: `${output}/dashboard-reduced.png` })
      // Reduced motion also applies on the very first render, and the navigation fallback stays usable.
      await page.reload()
      await expect(page.locator('.financial-summary-amount')).toContainText('882,29')
      await expect(page.locator('.motion-number-reel')).toHaveCount(0)
      await page.evaluate(() => { document.startViewTransition = undefined })
      await page.getByRole('tab', { name: 'Simulation', exact: true }).click()
      await expect(page.locator('main')).toHaveAttribute('data-page', 'simulate')
      await page.getByRole('tab', { name: 'Tableau de bord', exact: true }).click()
      await expect(page.locator('.financial-summary-amount')).toContainText('882,29')
      omittedMonth = '2026-05'
      await page.reload()
      await expect(page.locator('.motion-hero-flow table tbody tr').filter({ hasText: 'mai 2026' })).toContainText('Non observé')
      assert.ok(await page.locator('.motion-hero-flow .recharts-area-curve').first().evaluate(el => (el.getAttribute('d').match(/M/g) || []).length > 1), 'unobserved calendar month breaks the curve')
      await target.selectOption('2026-05')
      await expect(page.locator('.financial-summary-amount')).toHaveCount(0)
      await expect(page.locator('.kpi-row')).toHaveCount(0)
      await expect(page.getByText('Aucun indicateur observé pour cette période.')).toBeVisible()
      omittedMonth = null
      await page.emulateMedia({ reducedMotion: 'no-preference' })
      await page.evaluate(() => { document.documentElement.dataset.theme = 'dark'; localStorage.setItem('themis-theme', 'dark') })
      await page.reload()
      await expect(page.locator('.financial-summary-amount')).toContainText('882,29')
      await settle(page)
      await page.screenshot({ path: `${output}/dashboard-dark.png` })
      const darkAudit = await new AxeBuilder({ page }).include('main').analyze()
      assert.deepEqual(darkAudit.violations, [], 'Dark dashboard accessibility')
      await page.setViewportSize({ width: 390, height: 844 })
      for (const card of await page.locator('.widget').all()) {
        await card.scrollIntoViewIfNeeded()
        await expect(card).toHaveCSS('opacity', '1')
      }
      await expect(page.locator('.donut-wrap .recharts-pie-sector')).toHaveCount(3)
      await expect(page.locator('.donut-wrap .recharts-pie-sector').first()).toBeVisible()
      await page.evaluate(() => window.scrollTo(0, 0))
      await settle(page)
      await page.screenshot({ path: `${output}/dashboard-mobile.png`, fullPage: true })
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'no mobile page overflow')
      metrics.idle_running_animations = await page.evaluate(() => document.getAnimations().filter(a => a.playState === 'running').length)
      assert.equal(metrics.idle_running_animations, 0, 'no perpetual motion after settling')
      empty = true
      await page.reload()
      await expect(page.getByText("Aucune donnée pour l'instant.")).toBeVisible()
      await expect(page.locator('.animated-number')).toHaveCount(0)
    }
    assert.deepEqual(errors, [], 'no page exceptions')
    assert.deepEqual(warnings, [], 'no console errors')
    fs.writeFileSync(`${output}/performance.json`, JSON.stringify(metrics, null, 2))
    console.log(JSON.stringify({ baseline, metrics, screenshots: output, result: 'PASS' }, null, 2))
  } finally { await browser.close() }
}
main().catch(error => { console.error(error); process.exitCode = 1 })
