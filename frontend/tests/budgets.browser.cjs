// Financial UI fixtures only: every API request is intercepted. No real bank data.
const { chromium } = require('playwright')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const current = new Date().toISOString().slice(0, 7)
const shiftMonth = (month, offset) => {
  const date = new Date(`${month}-01T12:00:00Z`)
  date.setUTCMonth(date.getUTCMonth() + offset)
  return date.toISOString().slice(0, 7)
}
const stalePeriod = shiftMonth(current, -2),
  selectedPeriod = shiftMonth(current, -1)
const selectedMonthName = new Date(
  `${selectedPeriod}-01T12:00:00Z`,
).toLocaleDateString('fr-FR', { month: 'short', timeZone: 'UTC' })
const categories = [
  { id: 1, name: 'Courses', parent_id: null },
  { id: 2, name: 'Logement', parent_id: null },
  { id: 3, name: 'Loisirs', parent_id: null },
  { id: 4, name: 'Santé', parent_id: null },
  { id: 5, name: 'Épargne', is_transfer: true, parent_id: null },
]
const envelope = (id, domain_id, limit, spent, scope = 'expense') => ({
  id,
  domain_id,
  domain_name: categories.find((c) => c.id === domain_id).name,
  monthly_limit: limit,
  spent,
  remaining: (Math.round(limit * 100) - Math.round(spent * 100)) / 100,
  scope,
  ratio: spent / limit,
  transaction_count: spent ? 3 : 0,
  credit_total: id === 1 ? 40 : 0,
  debit_total: spent,
  projected: null,
  projection_available: false,
})
let rows = [
  envelope(1, 1, 200, 250),
  envelope(2, 2, 1000, 1000),
  envelope(3, 3, 100, 0),
  envelope(5, 5, 300, 150, 'transfer'),
]
const rec = (key, label, nature) => ({
  key: key.repeat(64),
  label,
  nature,
  detected_nature: nature,
  status: 'detected',
  confidence: 'insufficient',
  confidence_reason: 'Couverture documentaire non établie.',
  occurrence_count: 3,
  months_count: 3,
  last_date: `${current}-10`,
  first_date: `${current}-01`,
  last_amount: 20,
  typical_amount: 20,
  average_per_transaction: 20,
  average_denominator: 3,
  frequency: null,
  estimated_next_date: null,
  coverage_complete: false,
  coverage_start: `${current}-01`,
  coverage_end: `${current}-21`,
  direction: 'debit',
  transaction_ids: [101, 102, 103],
})
let recurrenceItems = [
  rec('a', 'NETFLIX TEST', 'subscription'),
  rec('b', 'EDF TEST', 'fixed_charge'),
  rec('c', 'DAB TEST', 'variable_recurring'),
  rec('d', 'PEL TEST', 'transfer_savings'),
]
let budgetFailure = false,
  empty = false,
  slowMonth = null,
  seen = []
function report(month) {
  const budgets = empty ? [] : rows
  const expenses = budgets.filter((r) => r.scope === 'expense')
  const allocated = expenses.reduce(
      (s, r) => s + Math.round(r.monthly_limit * 100),
      0,
    ),
    spent = expenses.reduce((s, r) => s + Math.round(r.spent * 100), 0)
  const historical = month !== current
  return {
    month,
    query_from: `${month}-01`,
    query_to: historical
      ? `${month}-28`
      : new Date().toISOString().slice(0, 10),
    period: {
      month,
      is_current: !historical,
      completeness: 'unknown',
      last_tx_date: `${month}-10`,
    },
    coverage: {
      month,
      through: `${month}-21`,
      accounts: [
        {
          id: 1,
          name: 'COMPTE TEST',
          status: 'insufficient',
          gaps: [{ from: `${month}-22`, to: `${month}-28` }],
          last_import: `${current}-21`,
        },
      ],
    },
    budgets,
    summary: {
      allocated: allocated / 100,
      spent: spent / 100,
      remaining: (allocated - spent) / 100,
      over_count: expenses.filter((r) => r.spent > r.monthly_limit).length,
      consumption_ratio: allocated ? spent / allocated : null,
      observed_expenses: 1500,
      covered_expenses: spent / 100,
      coverage_ratio: spent / 150000,
      uncategorized_spent: 250,
      uncategorized_count: 2,
      credits_observed: 40,
      savings_outflows: 150,
      transfer_outflows: 0,
    },
  }
}
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
  await page.route('**/api/**', async (route) => {
    const request = route.request(),
      url = new URL(request.url()),
      path = url.pathname
    seen.push(url.pathname + url.search)
    let data = {},
      status = 200
    if (path === '/api/auth/status') data = { setup_required: false }
    else if (path === '/api/auth/me')
      data = { id: 1, username: 'TEST BUDGETS', role: 'user' }
    else if (path === '/api/accounts' || path === '/api/tags') data = []
    else if (path === '/api/domains' || path === '/api/categories')
      data = categories
    else if (path === '/api/summary')
      data = { monthly: [], by_category: [], uncategorized: 0 }
    else if (path === '/api/hero' || path === '/api/remaining')
      data = { available: false }
    else if (path === '/api/journal') data = { events: [] }
    else if (path === '/api/anomalies') data = { anomalies: [] }
    else if (path === '/api/budgets/report') {
      const month = url.searchParams.get('month') || current
      data = report(month)
      if (slowMonth === month)
        await new Promise((resolve) => setTimeout(resolve, 700))
      if (budgetFailure) {
        status = 500
        data = { error: 'ERREUR TEST CONTRÔLÉE' }
      }
    } else if (path === '/api/budgets' && request.method() === 'POST') {
      const input = request.postDataJSON()
      const existing = rows.find((r) => r.domain_id === input.domain_id)
      if (existing)
        rows = rows.map((r) =>
          r.id === existing.id
            ? envelope(r.id, r.domain_id, input.monthly_limit, r.spent, r.scope)
            : r,
        )
      else rows.push(envelope(10, input.domain_id, input.monthly_limit, 0))
      data = { status: 'ok' }
    } else if (
      /^\/api\/budgets\/\d+$/.test(path) &&
      request.method() === 'DELETE'
    ) {
      rows = rows.filter((r) => r.id !== Number(path.split('/').at(-1)))
      data = { status: 'ok' }
    } else if (path === '/api/budgets') data = rows
    else if (path === '/api/recurring/review')
      data = {
        month: url.searchParams.get('month'),
        observation_start: '2026-01-01',
        observation_end: `${url.searchParams.get('month')}-21`,
        items: empty ? [] : recurrenceItems,
      }
    else if (
      path.startsWith('/api/recurring/review/') &&
      request.method() === 'PUT'
    ) {
      const patch = request.postDataJSON(),
        key = path.split('/').at(-1)
      recurrenceItems = recurrenceItems.map((r) =>
        r.key === key ? { ...r, ...patch } : r,
      )
      data = { status: 'ok' }
    } else if (path === '/api/transactions') data = []
    return route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(data),
    })
  })
  const open = async () => {
    await page.goto(process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176')
    await page
      .locator('.sidebar-nav button')
      .filter({ hasText: 'Budgets' })
      .click()
    await page
      .getByRole('heading', { name: 'Tes enveloppes de dépenses' })
      .waitFor()
  }
  const settle = async () =>
    page.evaluate(async () => {
      await document.fonts.ready
      await new Promise(requestAnimationFrame)
      await Promise.all(
        document
          .getAnimations()
          .filter((a) => Number.isFinite(a.effect.getComputedTiming().endTime))
          .map((a) => a.finished.catch(() => {})),
      )
    })
  await open()
  await settle()
  assert.equal(await page.locator('.budget-summary .budget-metric').count(), 6)
  assert.match(await page.locator('.budget-summary').innerText(), /1.?300,00/)
  assert.equal(
    await page.getByText('Plafond dépassé', { exact: true }).count(),
    1,
  )
  assert.equal(
    await page.getByText('Plafond atteint', { exact: true }).count(),
    1,
  )
  const expenseSection = page.locator('.budget-envelopes')
  const food = expenseSection.locator('.budget-envelope').filter({
    has: page.getByRole('heading', { name: 'Courses', exact: true }),
  })
  // Modify and persist a ceiling; the observed spending remains unchanged.
  await food.getByRole('button', { name: 'Modifier le plafond' }).click()
  await page
    .getByRole('spinbutton', { name: 'Plafond mensuel (€)' })
    .fill('250')
  await page
    .getByRole('button', { name: 'Enregistrer le plafond', exact: true })
    .click()
  await page.getByRole('dialog').waitFor({ state: 'detached' })
  await food.getByText('Plafond atteint', { exact: true }).waitFor()
  // Create from a contextual form, then cancel and confirm deletion.
  await page
    .getByRole('button', { name: 'Créer un budget', exact: true })
    .click()
  await page
    .getByRole('combobox', { name: 'Catégorie', exact: true })
    .selectOption('4')
  await page
    .getByRole('spinbutton', { name: 'Plafond mensuel (€)' })
    .fill('99.99')
  await page
    .getByRole('button', { name: 'Créer l’enveloppe', exact: true })
    .click()
  const health = expenseSection
    .locator('.budget-envelope')
    .filter({ has: page.getByRole('heading', { name: 'Santé', exact: true }) })
  await health.waitFor()
  await health.getByRole('button', { name: 'Supprimer', exact: true }).click()
  await page.getByRole('button', { name: 'Annuler', exact: true }).click()
  assert.equal(
    rows.some((r) => r.domain_id === 4),
    true,
  )
  await health.getByRole('button', { name: 'Supprimer', exact: true }).click()
  await page
    .getByRole('button', { name: 'Confirmer la suppression', exact: true })
    .click()
  await health.waitFor({ state: 'detached' })
  assert.equal(
    rows.some((r) => r.domain_id === 4),
    false,
  )
  // Month switch and stale response must never change the selected period.
  const monthInput = page.getByLabel('Mois consulté', { exact: true })
  slowMonth = stalePeriod
  await monthInput.fill(stalePeriod)
  await page
    .getByRole('status')
    .filter({ hasText: 'Chargement des budgets' })
    .waitFor()
  await monthInput.fill(selectedPeriod)
  await page
    .locator('.budget-period')
    .filter({ hasText: selectedMonthName })
    .waitFor()
  await page.waitForTimeout(800)
  assert.equal(await monthInput.inputValue(), selectedPeriod)
  assert.ok(
    (await page.locator('.budget-period').innerText()).includes(
      selectedMonthName,
    ),
  )
  assert.ok(
    seen.some((u) =>
      u.includes(`/api/recurring/review?month=${selectedPeriod}`),
    ),
  )
  // Deterministic economic groups and user review actions.
  for (const label of [
    'Abonnements',
    'Charges fixes récurrentes',
    'Dépenses variables récurrentes',
    'Transferts et épargne',
  ])
    assert.equal(
      await page.getByRole('heading', { name: label, exact: true }).count(),
      1,
    )
  const netflix = page.locator('.budget-recurrence').filter({
    has: page.locator('summary strong').filter({ hasText: 'NETFLIX TEST' }),
  })
  await netflix.locator('summary').focus()
  await page.keyboard.press('Enter')
  if (!(await netflix.evaluate((el) => el.open)))
    await netflix.locator('summary').click()
  await netflix
    .getByRole('button', { name: 'Confirmer, corriger ou ignorer' })
    .click()
  await page
    .getByRole('combobox', { name: 'Décision', exact: true })
    .selectOption('confirmed')
  await page
    .getByRole('button', { name: 'Enregistrer la décision', exact: true })
    .click()
  await netflix.getByText('Confirmée par vous', { exact: false }).waitFor()
  if (!(await netflix.evaluate((el) => el.open)))
    await netflix.locator('summary').click()
  await netflix
    .getByRole('button', { name: 'Confirmer, corriger ou ignorer' })
    .click()
  await page
    .getByRole('combobox', { name: 'Décision', exact: true })
    .selectOption('ignored')
  await page
    .getByRole('button', { name: 'Enregistrer la décision', exact: true })
    .click()
  await netflix.waitFor({ state: 'detached' })
  await page
    .getByRole('checkbox', { name: 'Inclure les détections ignorées' })
    .check()
  await netflix.waitFor()
  const edf = page.locator('.budget-recurrence').filter({
    has: page.locator('summary strong').filter({ hasText: 'EDF TEST' }),
  })
  await edf.locator('summary').click()
  await edf
    .getByRole('button', { name: 'Confirmer, corriger ou ignorer' })
    .click()
  await page
    .getByRole('combobox', { name: 'Nature économique', exact: true })
    .selectOption('variable_recurring')
  await page
    .getByRole('button', { name: 'Enregistrer la décision', exact: true })
    .click()
  await page
    .locator('.budget-recurrence-group')
    .filter({
      has: page.getByRole('heading', {
        name: 'Dépenses variables récurrentes',
        exact: true,
      }),
    })
    .getByText('EDF TEST', { exact: true })
    .waitFor()
  // Exact budget source drilldown, not a label or overbroad category search.
  const sourceRequest = page.waitForRequest(
    (request) =>
      request.url().includes('/api/transactions?') &&
      request.url().includes('budget_id=1'),
  )
  await food
    .getByRole('button', { name: 'Voir les transactions', exact: true })
    .click()
  await sourceRequest
  await page
    .locator('.notice')
    .filter({ hasText: 'Vue des opérations contribuant' })
    .waitFor()
  assert.ok(
    seen.some(
      (u) =>
        u.includes('/api/transactions?') &&
        u.includes('budget_id=1') &&
        u.includes(`month=${selectedPeriod}`),
    ),
  )
  await page
    .locator('.sidebar-nav button')
    .filter({ hasText: 'Budgets' })
    .click()
  await expenseSection.waitFor()
  await settle()
  const edfSource = page.locator('.budget-recurrence').filter({
    has: page.locator('summary strong').filter({ hasText: 'EDF TEST' }),
  })
  await edfSource.locator('summary').click()
  const sourceIDsRequest = page.waitForRequest(
    (request) =>
      request.url().includes('/api/transactions?') &&
      request.url().includes('ids='),
  )
  await edfSource
    .getByRole('button', { name: 'Voir les opérations sources', exact: true })
    .click()
  const exactRequest = new URL((await sourceIDsRequest).url())
  assert.deepEqual(
    exactRequest.searchParams
      .get('ids')
      .split(',')
      .map(Number)
      .sort((a, b) => a - b),
    [101, 102, 103],
  )
  assert.ok(
    exactRequest.searchParams.get('from') &&
      exactRequest.searchParams.get('to'),
  )
  await page
    .locator('.sidebar-nav button')
    .filter({ hasText: 'Budgets' })
    .click()
  await expenseSection.waitFor()
  await settle()
  // Source groups beyond the historical cap remain fully consultable.
  recurrenceItems = recurrenceItems.map((item) =>
    item.label === 'EDF TEST'
      ? {
          ...item,
          transaction_ids: Array.from({ length: 1001 }, (_, i) => i + 1),
          occurrence_count: 1001,
          average_denominator: 1001,
        }
      : item,
  )
  await page.getByLabel('Mois consulté', { exact: true }).fill(selectedPeriod)
  const largeSource = page
    .locator('.budget-recurrence')
    .filter({
      has: page.locator('summary strong').filter({ hasText: 'EDF TEST' }),
    })
  await largeSource.locator('summary').click()
  const firstSourcePage = page.waitForRequest(
    (request) =>
      request.url().includes('/api/transactions?') &&
      new URL(request.url()).searchParams.has('ids'),
  )
  await largeSource
    .getByRole('button', {
      name: 'Voir les 1001 opérations sources',
      exact: true,
    })
    .click()
  assert.equal(
    new URL((await firstSourcePage).url()).searchParams.get('ids').split(',')
      .length,
    1000,
  )
  const nextSourcePage = page.waitForRequest(
    (request) =>
      request.url().includes('/api/transactions?') &&
      new URL(request.url()).searchParams.get('ids') === '1',
  )
  await page
    .getByRole('button', { name: 'Sources suivantes', exact: true })
    .click()
  await nextSourcePage
  recurrenceItems = recurrenceItems.map((item) =>
    item.label === 'EDF TEST'
      ? {
          ...item,
          transaction_ids: [101, 102, 103],
          occurrence_count: 3,
          average_denominator: 3,
        }
      : item,
  )
  await page
    .locator('.sidebar-nav button')
    .filter({ hasText: 'Budgets' })
    .click()
  await expenseSection.waitFor()
  await settle()
  const artifacts = process.env.THEMIS_UI_ARTIFACTS || '/tmp/themis-budgets-ui'
  fs.mkdirSync(artifacts, { recursive: true })
  for (const [width, height, name] of [
    [1920, 1080, 'desktop'],
    [1280, 800, 'laptop'],
    [960, 540, 'zoom-200-layout'],
    [768, 1024, 'tablet'],
    [390, 844, 'mobile'],
    [320, 700, 'narrow'],
  ]) {
    await page.setViewportSize({ width, height })
    await page.evaluate(() => window.scrollTo(0, 0))
    await settle()
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
      `Overflow at ${width}`,
    )
    await page.screenshot({ path: `${artifacts}/${name}.png`, fullPage: true })
    if (width === 1920 || width === 390) {
      const axe = await new AxeBuilder({ page })
        .include('.budgets-page')
        .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
        .analyze()
      assert.deepEqual(
        axe.violations.map((v) => ({
          id: v.id,
          nodes: v.nodes.map((n) => n.target),
        })),
        [],
      )
    }
  }
  await page.setViewportSize({ width: 1920, height: 1080 })
  await page.locator('.sidebar-footer .theme-toggle').click()
  await settle()
  const lightAxe = await new AxeBuilder({ page })
    .include('.budgets-page')
    .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
    .analyze()
  assert.deepEqual(
    lightAxe.violations.map((v) => ({
      id: v.id,
      nodes: v.nodes.map((n) => n.target),
    })),
    [],
  )
  // Loading, failure and empty are distinct; retry must clear the error.
  budgetFailure = true
  await page
    .getByRole('button', { name: 'Mois précédent', exact: true })
    .click()
  await page.getByRole('alert').filter({ hasText: 'ERREUR TEST' }).waitFor()
  assert.equal(await page.locator('.budget-summary').count(), 0)
  budgetFailure = false
  empty = true
  await page
    .getByRole('button', { name: 'Réessayer', exact: true })
    .first()
    .click()
  await page
    .getByText('Aucune enveloppe de dépenses définie pour le moment.', {
      exact: true,
    })
    .waitFor()
  assert.equal(await page.getByRole('alert').count(), 0)
  assert.deepEqual(errors, [])
  console.log(
    JSON.stringify({ passed: true, viewports: 6, artifacts }, null, 2),
  )
  await browser.close()
}
main().catch((e) => {
  console.error(e)
  process.exit(1)
})
