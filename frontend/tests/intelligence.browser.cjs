// Synthetic evidence only. Exercise the real component through Vite.
const { chromium, expect } = require('@playwright/test')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const base = process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176'
const discovery = {
  id: 'synthetic-trend', kind: 'progressive_trend', period: '2026-06',
  title: 'Hausse progressive : Marché TEST', summary: 'Six mois documentés, de 100,01 € à 250,02 €.', confidence: 85,
  metrics: [{ name: 'first_period_cents', kind: 'money_cents', value: 10001 }, { name: 'last_period_cents', kind: 'money_cents', value: 25002 }],
  evidence: { transaction_ids: [11, 22, 33, 44, 55, 66], periods: ['2026-01', '2026-02', '2026-03', '2026-04', '2026-05', '2026-06'] },
  coverage: { description: 'Six mois calendaires complets.' }, limitations: ['La hausse observée ne démontre pas sa cause.'],
}
async function main() {
  const browser = await chromium.launch({ headless: true, channel: 'chromium' })
  try {
    const context = await browser.newContext()
    const page = await context.newPage()
    const errors = [], unexpected = [], mutations = []
    let hidden = false, preference = null, jobRequests = 0
    page.on('pageerror', error => { errors.push(error.message); console.error(error.message) })
    await page.route('**/api/**', async route => {
      const request = route.request(), path = new URL(request.url()).pathname
      let body
      if (path === '/api/intel/feed') body = { discoveries: hidden ? [] : [discovery] }
      else if (path === '/api/intel/jobs/status') { jobRequests++; body = { status: 'idle' } }
      else if (path === '/api/intel/discoveries/synthetic-trend' && request.method() === 'GET') body = { discovery, observations: [{ ...discovery, observation_id: 'observation-test' }] }
      else if (path === '/api/intel/discoveries/synthetic-trend/feedback') {
        const data = request.postDataJSON()
        mutations.push(data)
        hidden = data.action === 'hide'
        preference = { discovery_id: discovery.id, title: discovery.title, action: data.action, note: '' }
        body = { state: hidden ? 'hidden' : 'active' }
      } else if (path === '/api/intel/preferences') body = { preferences: preference ? [preference] : [] }
      else if (path === '/api/intel/recompute') { mutations.push(request.postDataJSON()); body = { discoveries: [discovery] } }
      else { unexpected.push(`${request.method()} ${path}`); body = { error: 'Unexpected synthetic request' } }
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) })
    })
    await page.route('**/__intelligence_test__', route => route.fulfill({ contentType: 'text/html', body: `<!doctype html><html lang="fr"><head><title>Découvertes TEST</title><meta name="viewport" content="width=device-width, initial-scale=1"></head><body><main id="root"></main><script type="module">
      import RefreshRuntime from '/@react-refresh'; RefreshRuntime.injectIntoGlobalHook(window); window.$RefreshReg$ = () => {}; window.$RefreshSig$ = () => type => type; window.__vite_plugin_react_preamble_installed__ = true;
      import React from '/node_modules/.vite/deps/react.js';
      import ReactDOM from '/node_modules/.vite/deps/react-dom_client.js';
      const { IntelligenceHighlights } = await import('/src/components/IntelligenceHighlights.jsx');
      import '/src/styles.css'; import '/src/cockpit.css'; import '/src/motion/cinematic.css';
      window.__navigation = [];
      ReactDOM.createRoot(document.getElementById('root')).render(React.createElement(IntelligenceHighlights, { month: '2026-06', onNavigate: (...args) => window.__navigation.push(args) }));
    </script></body></html>` }))
    await page.clock.install()
    await page.goto(`${base}/__intelligence_test__`)
    await expect(page.getByRole('heading', { name: discovery.title })).toBeVisible()
    await page.getByRole('button', { name: 'Voir l’historique, le calcul et les limites' }).click()
    await expect(page.locator('.intel-observation-history')).toContainText(discovery.summary)
    await expect(page.getByRole('table', { name: 'Données de la comparaison' })).toContainText('100,01')
    await expect(page.getByRole('table', { name: 'Données de la comparaison' })).toContainText('250,02')
    await page.getByRole('button', { name: 'Voir les opérations sources →' }).click()
    assert.deepEqual(await page.evaluate(() => window.__navigation), [['transactions', { ids: discovery.evidence.transaction_ids }]], 'Six-month evidence must not be intersected with the final month')
    await page.getByRole('button', { name: 'Recalculer', exact: true }).click()
    await expect.poll(() => mutations.length).toBe(1)
    assert.deepEqual(mutations[0], { month: '2026-06' })
    await page.getByRole('button', { name: 'Intéressant', exact: true }).click()
    await expect(page.getByRole('status').filter({ hasText: 'Retour enregistré' })).toBeVisible()
    assert.deepEqual(mutations[1], { action: 'interesting' })
    await page.getByRole('button', { name: 'Gérer mes retours et préférences' }).click()
    await expect(page.getByRole('combobox', { name: 'Retour', exact: true })).toHaveValue('interesting')
    for (const theme of ['light', 'dark']) {
      await page.evaluate(theme => { document.documentElement.dataset.theme = theme }, theme)
      for (const width of [1440, 768, 390, 320]) {
        await page.setViewportSize({ width, height: 1000 })
        await page.evaluate(() => document.fonts.ready)
        await expect.poll(() => page.evaluate(() => document.getAnimations().filter(animation => animation.playState === 'running').length)).toBe(0)
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), `${theme} overflow at ${width}`)
        const result = await new AxeBuilder({ page }).include('main').withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze()
        assert.deepEqual(result.violations.map(v => ({ id: v.id, nodes: v.nodes.map(n => ({ target: n.target, failure: n.failureSummary })) })), [], `${theme} Axe at ${width}`)
      }
    }
    await expect.poll(() => jobRequests).toBeGreaterThan(0)
    await page.evaluate(() => {
      Object.defineProperty(document, 'hidden', { configurable: true, get: () => true })
      document.dispatchEvent(new Event('visibilitychange'))
    })
    const beforeHidden = jobRequests
    await page.clock.fastForward(15_000)
    assert.equal(jobRequests, beforeHidden, 'Hidden tabs must not poll intelligence jobs')
    await page.evaluate(() => {
      Object.defineProperty(document, 'hidden', { configurable: true, get: () => false })
      document.dispatchEvent(new Event('visibilitychange'))
    })
    await expect.poll(() => jobRequests).toBeGreaterThan(beforeHidden)
    await page.getByRole('button', { name: 'Masquer', exact: true }).click()
    await expect(page.locator('.intel-empty')).toContainText('Aucune découverte suffisamment étayée')
    assert.deepEqual(errors, [])
    assert.deepEqual(unexpected, [])
    console.log('Intelligence browser PASS: exact metrics, history, multi-month sources, recompute, feedback, preferences, empty state, hidden-tab polling/resume and light/dark Axe at four widths.')
  } finally { await browser.close() }
}
main().catch(error => { console.error(error); process.exitCode = 1 })
