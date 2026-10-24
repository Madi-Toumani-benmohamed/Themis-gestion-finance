// Isolated UI fixtures only: every /api request is intercepted, no bank data.
// NODE_PATH=/tmp/themis-browser/node_modules node frontend/tests/simulation.browser.cjs
const { chromium } = require('playwright')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const names = [
  'Logement',
  'Courses',
  'Transport',
  'Énergie',
  'Restaurants',
  'Loisirs',
  'Santé',
  'Assurances',
  'Téléphone',
  'Internet',
  'Vêtements',
  'Voyages',
  'Éducation',
  'Sport',
  'Culture',
  'Services',
  'Animaux',
  'Cadeaux',
  'Frais bancaires',
  'Autres dépenses',
]
const categories = names.map((category, i) => ({
  category,
  before:
    i === 0 ? 1500 : i === 1 ? 800 : i === 2 ? 400 : i === 19 ? 217.14 : 80,
}))
const reference = {
  available: true,
  income: 3249.47,
  observed_income: 3249.47,
  base_spend: 4197.14,
  current_net: -947.67,
  reference_complete: false,
  categories,
  reference_periods: ['2026-07', '2026-08', '2026-09'].map((month) => ({
    month,
    completeness: 'unknown',
  })),
}
let simulationResponse = reference
async function main() {
  const browser = await chromium.launch({ headless: true, channel: 'chromium' })
  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
  })
  const page = await context.newPage()
  page.setDefaultTimeout(10000)
  const errors = []
  page.on('pageerror', (e) => errors.push(e.message))
  await page.addInitScript(() => localStorage.setItem('themis-theme', 'dark'))
  await page.route('**/api/**', (route) => {
    const path = new URL(route.request().url()).pathname
    let data = {}
    if (path === '/api/auth/status') data = { setup_required: false }
    else if (path === '/api/auth/me')
      data = { id: 1, username: 'TEST UI', role: 'user' }
    else if (path === '/api/accounts') data = []
    else if (path === '/api/summary')
      data = { monthly: [], by_category: [], uncategorized: 0 }
    else if (path === '/api/simulate') data = simulationResponse
    else if (path === '/api/hero') data = { available: false }
    else if (path === '/api/remaining') data = { available: false }
    else if (path === '/api/journal') data = { events: [] }
    else if (path === '/api/anomalies') data = { anomalies: [] }
    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(data),
    })
  })
  await page.goto(process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176')
  await page
    .locator('.sidebar-nav button')
    .filter({ hasText: 'Simulation' })
    .click()
  await page.locator('.scenario-category').first().waitFor()
  await page.evaluate(async () => {
    await document.fonts.ready
    await Promise.all(
      document
        .getAnimations()
        .filter((a) => Number.isFinite(a.effect.getComputedTiming().endTime))
        .map((a) => a.finished.catch(() => {})),
    )
  })
  assert.equal(await page.locator('.scenario-metric').count(), 5)
  assert.match(
    await page.locator('.scenario-verdict strong').innerText(),
    /-947,67/,
  )
  const list = await page.locator('.scenario-list').boundingBox()
  const result = await page.locator('.scenario-results').boundingBox()
  assert.ok(
    list.height <= 420 && result.y < 700,
    'desktop controls/results must fit the first screen',
  )
  assert.ok(
    result.y + result.height <= 1080,
    `desktop result should be visible without page scroll: ${JSON.stringify(result)}`,
  )
  // Keyboard opens an accordion; controls retain independent values.
  const first = page.locator('.scenario-category').first().locator('summary')
  await first.focus()
  await page.keyboard.press('Enter')
  await page
    .getByRole('spinbutton', {
      name: 'Réduction de Logement en pourcentage',
      exact: true,
    })
    .fill('10')
  await page.locator('.scenario-category').nth(1).locator('summary').click()
  await page
    .getByRole('spinbutton', {
      name: 'Réduction de Courses en pourcentage',
      exact: true,
    })
    .fill('25')
  assert.match(
    await page.locator('.scenario-variation dd').innerText(),
    /350,00/,
  )
  await page.getByRole('checkbox', { name: 'Modifiées uniquement' }).check()
  assert.equal(await page.locator('.scenario-category').count(), 2)
  await page
    .getByRole('searchbox', { name: 'Rechercher une catégorie' })
    .fill('Courses')
  assert.equal(await page.locator('.scenario-category').count(), 1)
  await page
    .getByRole('searchbox', { name: 'Rechercher une catégorie' })
    .fill('')
  assert.equal(
    await page
      .getByRole('spinbutton', {
        name: 'Réduction de Logement en pourcentage',
        exact: true,
      })
      .inputValue(),
    '10',
  )
  await page
    .getByRole('button', { name: 'Réinitialiser le scénario', exact: true })
    .click()
  assert.equal(await page.locator('.scenario-category').count(), 20)
  assert.match(await page.locator('.scenario-variation dd').innerText(), /0,00/)
  // Optional controls, invalid input and recovery, without API round trips.
  await page.getByText('Revenu mensuel du scénario', { exact: false }).click()
  await page
    .getByRole('spinbutton', { name: 'Revenu simulé (€/mois)', exact: true })
    .fill('4000')
  assert.match(
    await page.locator('.scenario-metric').nth(2).innerText(),
    /-947,67/,
  )

  await page
    .locator('.scenario-options > details')
    .nth(1)
    .locator('summary')
    .click()
  await page
    .getByRole('textbox', { name: 'Intitulé', exact: true })
    .fill('Projet test')
  await page
    .getByRole('spinbutton', { name: 'Montant (€/mois)', exact: true })
    .fill('25')
  assert.match(
    await page.locator('.scenario-verdict strong').innerText(),
    /-222,14/,
  )
  await page
    .locator('.scenario-options > details')
    .nth(2)
    .locator('summary')
    .click()
  await page
    .getByRole('spinbutton', { name: 'Capital emprunté (€)', exact: true })
    .fill('1200')
  assert.match(await page.getByRole('alert').innerText(), /Complète/)
  await page
    .getByRole('spinbutton', { name: 'Durée (mois)', exact: true })
    .fill('12')
  assert.match(
    await page.locator('.scenario-verdict strong').innerText(),
    /-322,14/,
  )
  await page
    .getByRole('button', { name: 'Effacer l’emprunt', exact: true })
    .click()
  await page
    .getByRole('button', {
      name: 'Retirer la dépense Projet test',
      exact: true,
    })
    .click()
  assert.match(
    await page.locator('.scenario-verdict strong').innerText(),
    /-197,14/,
  )

  // Audit the simulator in isolation from unrelated global navigation.
  const axe = await new AxeBuilder({ page })
    .include('.scenario-page')
    .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
    .analyze()
  assert.deepEqual(
    axe.violations.map((v) => ({
      id: v.id,
      impact: v.impact,
      nodes: v.nodes.length,
    })),
    [],
  )
  await page
    .getByRole('button', { name: 'Réinitialiser le scénario', exact: true })
    .click()
  await page.locator('.scenario-page details[open]').evaluateAll((elements) =>
    elements.forEach((el) => {
      el.open = false
    }),
  )
  await page.evaluate(() => window.scrollTo(0, 0))
  const artifacts =
    process.env.THEMIS_UI_ARTIFACTS || '/tmp/themis-simulation-ui'
  fs.mkdirSync(artifacts, { recursive: true })
  await page.screenshot({ path: `${artifacts}/desktop.png`, fullPage: true })
  const sizes = [
    [1280, 800, 'laptop'],
    [960, 540, 'zoom-200-layout'],
    [768, 1024, 'tablet'],
    [390, 844, 'mobile'],
    [320, 700, 'narrow'],
  ]
  for (const [width, height, name] of sizes) {
    await page.setViewportSize({ width, height })
    await page.evaluate(() => window.scrollTo(0, 0))
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
      `horizontal overflow at ${width}`,
    )
    await page.locator('.scenario-category').first().locator('summary').focus()
    if (
      !(await page
        .locator('.scenario-category')
        .first()
        .evaluate((el) => el.open))
    )
      await page.keyboard.press('Enter')
    await page
      .getByRole('spinbutton', {
        name: 'Réduction de Logement en pourcentage',
        exact: true,
      })
      .fill('15')
    assert.match(
      await page.locator('.scenario-variation dd').innerText(),
      /225,00/,
    )
    if (width <= 1150)
      assert.ok(await page.locator('.scenario-mobile-result').isVisible())
    if (width === 390) {
      assert.ok(await page.locator('.scenario-mobile-result').isVisible())
      assert.match(
        await page.locator('.scenario-mobile-result').innerText(),
        /225,00/,
      )
      const mobileAxe = await new AxeBuilder({ page })
        .include('.scenario-page')
        .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
        .analyze()
      assert.deepEqual(
        mobileAxe.violations.map((v) => v.id),
        [],
      )
    }
    await page.screenshot({ path: `${artifacts}/${name}.png`, fullPage: true })
    if (width <= 600)
      await page.screenshot({ path: `${artifacts}/${name}-viewport.png` })
    await page
      .getByRole('button', { name: 'Réinitialiser le scénario', exact: true })
      .click()
  }
  await page.setViewportSize({ width: 1920, height: 1080 })
  await page.locator('.sidebar-footer .theme-toggle').click()
  await page.evaluate(async () => {
    await new Promise(requestAnimationFrame)
    await Promise.all(
      document
        .getAnimations()
        .filter((a) => Number.isFinite(a.effect.getComputedTiming().endTime))
        .map((a) => a.finished.catch(() => {})),
    )
  })
  const lightAxe = await new AxeBuilder({ page })
    .include('.scenario-page')
    .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
    .analyze()
  assert.deepEqual(
    lightAxe.violations.map((v) => ({
      id: v.id,
      nodes: v.nodes.map((n) => ({
        target: n.target,
        summary: n.failureSummary,
      })),
    })),
    [],
  )
  simulationResponse = {
    available: false,
    reason: 'Aucune référence de test disponible.',
  }
  await page.reload()
  await page
    .locator('.sidebar-nav button')
    .filter({ hasText: 'Simulation' })
    .click()
  await page
    .getByText('Aucune référence de test disponible.', { exact: true })
    .waitFor()
  assert.equal(await page.locator('.scenario-summary').count(), 0)
  simulationResponse = { ...reference, current_net: -947.66 }
  await page.reload()
  await page
    .locator('.sidebar-nav button')
    .filter({ hasText: 'Simulation' })
    .click()
  assert.match(
    await page.getByRole('alert').innerText(),
    /écart de réconciliation/,
  )
  assert.equal(await page.locator('.scenario-summary').count(), 0)
  assert.deepEqual(errors, [])
  console.log(
    JSON.stringify(
      {
        passed: true,
        viewports: 6,
        axeViolations: axe.violations.length,
        artifacts,
      },
      null,
      2,
    ),
  )
  await browser.close()
}
main().catch((err) => {
  console.error(err)
  process.exit(1)
})
