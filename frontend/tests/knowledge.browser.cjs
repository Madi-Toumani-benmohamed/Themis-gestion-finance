// Actual application UI, synthetic scoped API fixtures. Real Falkor/PG contracts
// are exercised separately; rendering this panel never learns artificial facts.
const { chromium, expect } = require('@playwright/test')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fixtures = require('./fixtures/knowledge.cjs')
const base = process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176'
async function settle(page, theme) {
  await page.evaluate(async (t) => {
    if (t) document.documentElement.setAttribute('data-theme', t)
    await document.fonts.ready
    await new Promise(requestAnimationFrame)
    await Promise.all(
      document
        .getAnimations()
        .filter((a) => a.effect?.getComputedTiming().iterations !== Infinity)
        .map((a) => a.finished.catch(() => {})),
    )
    await new Promise(requestAnimationFrame)
  }, theme)
}
async function main() {
  const browser = await chromium.launch({ headless: true, channel: 'chromium' })
  try {
    const context = await browser.newContext({ reducedMotion: 'reduce' }),
      page = await context.newPage(),
      errors = [],
      calls = []
    let data = fixtures.overview(),
      searchMode = 'success',
      slowRelease = null,
      overviewFailure = false
    page.on('pageerror', (e) => errors.push(e.message))
    await page.route('**/api/**', async (route) => {
      const req = route.request(),
        url = new URL(req.url()),
        path = url.pathname
      calls.push({ path, method: req.method() })
      let body = {},
        status = 200
      if (path === '/api/auth/status') body = { setup_required: false }
      else if (path === '/api/auth/me')
        body = { id: 1, username: 'MEMORY SYNTHETIC TEST', role: 'admin' }
      else if (path === '/api/health') body = { ai_enabled: true }
      else if (path === '/api/accounts' || path === '/api/tags') body = []
      else if (path === '/api/summary')
        body = { monthly: [], by_category: [], uncategorized: 0 }
      else if (['/api/hero', '/api/remaining'].includes(path))
        body = { available: false }
      else if (path === '/api/anomalies') body = { anomalies: [] }
      else if (path === '/api/journal') body = { events: [] }
      else if (path === '/api/knowledge/overview') {
        if (overviewFailure) {
          status = 500
          body = { error: 'JOURNAL INDISPONIBLE TEST' }
        } else {
          body = structuredClone(data)
          if (url.searchParams.get('before_id') !== '0') {
            body.activity = []
            body.has_more = false
          }
        }
      } else if (path === '/api/knowledge/search') {
        const query = req.postDataJSON()
        assert.deepEqual(Object.keys(query), ['query'])
        if (searchMode === 'slow')
          await new Promise((r) => {
            slowRelease = r
          })
        body = fixtures.trace()
        if (searchMode === 'empty')
          body = fixtures.trace({ status: 'empty', facts: [] })
        if (searchMode === 'error')
          body = fixtures.trace({
            status: 'error',
            facts: [],
            error_code: 'model_missing',
          })
      } else if (path === '/api/chat')
        body = {
          conversation_id: 1,
          answer: 'REPONSE SYNTHETIQUE TEST',
          evidence: [],
          status: 'ok',
          memory: fixtures.trace({ context_included: true }),
        }
      await route.fulfill({ status, json: body }).catch((e) => {
        if (!req.failure()) throw e
      })
    })
    await page.goto(base)
    await page.getByRole('tab', { name: 'Mémoire', exact: true }).click()
    const panel = page.getByRole('region', {
      name: 'Activité du Knowledge Graph',
      exact: true,
    })
    await expect(
      panel.getByRole('heading', { name: 'Activité du Knowledge Graph' }),
    ).toBeVisible()
    await expect(
      panel.getByText('Lecture du graphe réussie', { exact: true }),
    ).toBeVisible()
    await expect(panel.getByText('2', { exact: true }).first()).toBeVisible()
    assert.equal(
      calls.filter((c) => c.method === 'POST').length,
      0,
      'Opening the panel must never learn facts',
    )
    await panel.getByLabel('Entité à explorer').selectOption('node-1')
    await expect(
      panel.getByRole('button', { name: 'Explorer MARCHAND TEST' }),
    ).toHaveAttribute('aria-pressed', 'true')
    await panel.getByLabel('Relation à consulter').selectOption('edge-1')
    await expect(panel.locator('.kg-graph')).toContainText(
      'MARCHAND TEST revient',
    )
    await expect(panel.locator('.kg-graph')).toContainText('Relevé #101')
    await panel
      .getByRole('textbox', { name: 'Question de vérification' })
      .fill('MARCHAND TEST')
    await panel
      .getByRole('button', { name: 'Tester la recherche', exact: true })
      .click()
    await expect(panel.locator('.kg-search-result')).toContainText(
      '1 fait(s) retrouvé(s)',
    )
    assert.equal(
      calls.filter((c) => c.path === '/api/knowledge/learn').length,
      0,
    )
    searchMode = 'empty'
    await panel
      .getByRole('button', { name: 'Tester la recherche', exact: true })
      .click()
    await expect(panel.locator('.kg-search-result')).toContainText(
      'Aucun fait retrouvé',
    )
    searchMode = 'error'
    await panel
      .getByRole('button', { name: 'Tester la recherche', exact: true })
      .click()
    await expect(panel.locator('.kg-search-result')).toContainText(
      'Modèle ou endpoint introuvable',
    )
    await expect(panel.locator('.kg-search-result')).not.toContainText('0 fait')
    // Both counters unknown when Falkor is inaccessible, with retained SQL history.
    data = {
      ...data,
      service: {
        ...data.service,
        status: 'error',
        detail: 'GRAPH INDISPONIBLE TEST',
        error_code: 'network',
      },
      graph: null,
    }
    await panel.getByRole('button', { name: 'Actualiser l’activité' }).click()
    await expect(
      panel.getByText('GRAPH INDISPONIBLE TEST', { exact: true }),
    ).toBeVisible()
    await expect(panel.locator('.kg-metrics')).toContainText('Non mesuré')
    await expect(
      panel.getByRole('region', { name: 'Journal mémoire' }),
    ).toBeVisible()
    overviewFailure = true
    await panel.getByRole('button', { name: 'Actualiser l’activité' }).click()
    await expect(panel.getByRole('alert')).toContainText(
      'JOURNAL INDISPONIBLE TEST',
    )
    overviewFailure = false
    data = fixtures.overview()
    data.has_more = true
    await panel.getByRole('button', { name: 'Réessayer' }).click()
    await expect(
      panel.getByRole('button', { name: 'Opérations plus anciennes' }),
    ).toBeVisible()
    await panel
      .getByRole('button', { name: 'Opérations plus anciennes' })
      .click()
    await expect(
      panel.getByText(
        'Aucune opération journalisée dans cette page pour ce filtre.',
      ),
    ).toBeVisible()
    await panel
      .getByRole('button', { name: 'Opérations plus récentes' })
      .click()
    await expect(
      panel.getByText('Lecture du graphe réussie', { exact: true }),
    ).toBeVisible()
    await panel.getByLabel('Opérations affichées').selectOption('chat')
    await expect(panel.locator('.kg-history')).toContainText(
      'Contexte de l’assistant',
    )
    await panel.getByLabel('Opérations affichées').selectOption('all')
    searchMode = 'slow'
    await panel
      .getByRole('textbox', { name: 'Question de vérification' })
      .fill('ANCIEN TEST')
    await panel
      .getByRole('button', { name: 'Tester la recherche', exact: true })
      .click()
    await expect.poll(() => !!slowRelease).toBe(true)
    await panel
      .getByRole('textbox', { name: 'Question de vérification' })
      .fill('NOUVEAU TEST')
    searchMode = 'success'
    slowRelease()
    await expect(panel.locator('.kg-search-result')).toHaveCount(0)
    data = fixtures.overview()
    data.graph = {
      ...data.graph,
      node_count: 0,
      edge_count: 0,
      episode_count: 0,
      nodes: [],
      edges: [],
      sources: [],
    }
    data.last_learning = null
    data.activity = []
    await panel.getByRole('button', { name: 'Actualiser l’activité' }).click()
    await expect(
      panel.locator('.kg-metrics').getByText('0', { exact: true }),
    ).toHaveCount(3)
    await expect(
      panel.getByText('Aucune entité enregistrée pour ce compte.', {
        exact: true,
      }),
    ).toBeVisible()
    data = fixtures.overview()
    data.graph.edges[0].fact = '<img src=x onerror="window.__kgInjected=true">'
    await panel.getByRole('button', { name: 'Actualiser l’activité' }).click()
    await panel.getByLabel('Relation à consulter').selectOption('edge-1')
    await expect(panel.locator('.kg-graph')).toContainText('<img src=x')
    await expect(panel.locator('.kg-graph img')).toHaveCount(0)
    assert.notEqual(await page.evaluate(() => window.__kgInjected), true)
    data = fixtures.overview()
    await panel.getByRole('button', { name: 'Actualiser l’activité' }).click()
    await panel.getByLabel('Relation à consulter').selectOption('edge-1')
    for (const width of [360, 768, 1280])
      for (const theme of ['light', 'dark']) {
        await page.setViewportSize({ width, height: 1000 })
        await settle(page, theme)
        assert.ok(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth + 1,
          ),
          `Overflow ${width}`,
        )
        const axe = await new AxeBuilder({ page })
          .include('.knowledge-activity')
          .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
          .analyze()
        assert.deepEqual(axe.violations, [], `Knowledge Axe ${width} ${theme}`)
        if (process.env.THEMIS_KG_ARTIFACTS) {
          const fs = require('node:fs'),
            path = require('node:path')
          fs.mkdirSync(process.env.THEMIS_KG_ARTIFACTS, { recursive: true })
          await page.screenshot({
            path: path.join(
              process.env.THEMIS_KG_ARTIFACTS,
              `knowledge-${width}-${theme}.png`,
            ),
            fullPage: true,
          })
        }
      }
    await page.getByRole('tab', { name: 'Assistant', exact: true }).click()
    await page
      .locator('.assistant-input')
      .getByRole('textbox', { name: 'Votre question' })
      .fill('QUESTION TEST')
    await page
      .locator('.assistant-input')
      .getByRole('textbox', { name: 'Votre question' })
      .press('Enter')
    await expect(page.locator('main .kg-trace')).toContainText(
      '1 fait(s) de mémoire joints à la requête IA',
    )
    await page.locator('main .kg-trace summary').click()
    await expect(page.locator('main .kg-trace')).toContainText(
      'ne prouve pas que le modèle les a utilisés',
    )
    await page.getByRole('button', { name: "Ouvrir l'assistant" }).click()
    await page
      .locator('.assistant-panel-input')
      .getByRole('textbox', { name: 'Votre question' })
      .fill('QUESTION FLOTTANTE TEST')
    await page
      .locator('.assistant-panel-input')
      .getByRole('textbox', { name: 'Votre question' })
      .press('Enter')
    await expect(page.locator('.assistant-panel .kg-trace')).toContainText(
      '1 fait(s) de mémoire joints',
    )
    await settle(page)
    const traceAxe = await new AxeBuilder({ page })
      .include('.assistant-panel')
      .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
      .analyze()
    assert.deepEqual(
      traceAxe.violations,
      [],
      'Floating memory trace accessibility',
    )
    assert.deepEqual(errors, [])
    console.log(
      'Knowledge browser PASS: actual panel, scoped graph/facts/sources, unknown vs zero, journal, failed models, stale responses, pagination, main/floating assistant trace, responsive and light/dark Axe.',
    )
    await context.close()
  } finally {
    await browser.close()
  }
}
main().catch((e) => {
  console.error(e)
  process.exitCode = 1
})
