// All API calls are intercepted. No real account, transaction or LLM is used.
const { chromium } = require('playwright')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const month = new Date().toISOString().slice(0, 7)
const before = new Date(`${month}-01T12:00:00Z`)
before.setUTCMonth(before.getUTCMonth() - 1)
const previous = before.toISOString().slice(0, 7)
let fail = '',
  empty = false,
  delay = '',
  narrativeUnavailable = false,
  requests = [],
  preferences = []
let tagChanges = [],
  tagRows = [
    { name: 'projet test', count: 3 },
    { name: 'second test', count: 1 },
  ]
const cats = [
  {
    id: 1,
    name: 'Courses TEST',
    color: '#9c99ff',
    parent_id: null,
    is_transfer: false,
  },
]
function nature(m) {
  return {
    available: !empty,
    month: m,
    query_from: `${m}-01`,
    query_to: `${m}-21`,
    transaction_count: empty ? 0 : 5,
    period: { month: m, is_current: m === month, completeness: 'unknown' },
    coverage: {
      accounts: [
        {
          id: 1,
          name: 'COMPTE TEST',
          status: 'insufficient',
          gaps: [{ from: `${m}-22`, to: `${m}-28` }],
        },
      ],
    },
    income: 3249.47,
    expenses: 4197.14,
    net: -947.67,
    real_spend: 4100,
    mandatory: 97.14,
    savings: 100,
    transfer_out: 22,
    transfer_in: 22,
    total_out: 4319.14,
    uncategorized_count: 0,
    uncategorized_spent: 0,
    unattributed_count: 0,
    categories: [
      {
        id: 1,
        name: 'Courses TEST',
        spent: 4197.14,
        transaction_count: 3,
        share_pct: 100,
      },
    ],
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
    const req = route.request(),
      url = new URL(req.url()),
      path = url.pathname,
      q = url.searchParams,
      m = q.get('month') || month
    requests.push(path + url.search)
    let data = {},
      status = 200
    if (delay && path === delay) await new Promise((r) => setTimeout(r, 650))
    if (path === fail) {
      status = 500
      data = { error: 'ERREUR ANALYSE TEST' }
    } else if (path === '/api/auth/status') data = { setup_required: false }
    else if (path === '/api/auth/me')
      data = { id: 1, username: 'ANALYSE TEST', role: 'user' }
    else if (path === '/api/health') data = { ai_enabled: true }
    else if (path === '/api/summary')
      data = { monthly: [], by_category: [], uncategorized: 0 }
    else if (path === '/api/hero' || path === '/api/remaining')
      data = { available: false }
    else if (path === '/api/journal') data = { events: [] }
    else if (path === '/api/anomalies') data = { anomalies: [] }
    else if (path === '/api/categories' || path === '/api/domains') data = cats
    else if (path === '/api/accounts') data = [{ id: 1, name: 'COMPTE TEST' }]
    else if (path === '/api/tags') data = tagRows
    else if (path.startsWith('/api/tags/') && req.method() === 'PATCH') {
      const name = decodeURIComponent(path.split('/').at(-1)),
        next = req.postDataJSON().name
      tagChanges.push({ method: 'PATCH', name, next })
      tagRows = tagRows.map((tag) =>
        tag.name === name ? { ...tag, name: next } : tag,
      )
      data = { merged: false }
    } else if (path.startsWith('/api/tags/') && req.method() === 'DELETE') {
      const name = decodeURIComponent(path.split('/').at(-1))
      tagChanges.push({ method: 'DELETE', name })
      tagRows = tagRows.filter((tag) => tag.name !== name)
      data = { ok: true }
    } else if (path === '/api/intel/nature') data = nature(m)
    else if (path === '/api/intel/comparison')
      data = {
        month: m,
        available: true,
        comparable: false,
        reference_months: ['2020-01', '2020-02', '2020-03'],
        denominator: 0,
        reason: 'Couverture inconnue : aucune comparaison à un mois complet.',
        categories: [
          {
            id: 1,
            name: 'Courses TEST',
            current: 4197.14,
            average: null,
            delta: null,
            delta_pct: null,
          },
        ],
      }
    else if (path === '/api/intel/merchants')
      data = {
        month: m,
        query_from: `${m}-01`,
        query_to: `${m}-21`,
        total: 4197.14,
        reason: 'Panier moyen par transaction, tendance non établie.',
        comparable: m === previous,
        reference_from: '2020-01-01',
        reference_to: '2020-01-31',
        merchants: [
          {
            name: 'MARCHAND TEST',
            visits: 3,
            total: 4197.14,
            avg_basket: 1399.05,
            trend_pct: m === previous ? 10 : null,
            share_pct: 100,
            last_seen: `${m}-10`,
          },
        ],
      }
    else if (path === '/api/intel/discoveries')
      data = {
        month: m,
        query_from: '2020-01-01',
        query_to: `${m}-21`,
        discoveries: [
          {
            kind: 'recurring',
            title: 'RECURRENCE TEST',
            detail: '3 occurrences observées. Fréquence probable.',
            status: 'detected',
            confidence: 'medium',
            source: { ids: [1, 2, 3], from: '2020-01-01', to: `${m}-21` },
          },
        ],
        limitations: ['Détection à confirmer.'],
      }
    else if (path === '/api/intel/narrative')
      data = narrativeUnavailable
        ? {
            available: false,
            reason: 'IA indisponible, constats déterministes accessibles.',
          }
        : {
            available: true,
            month: m,
            model: 'LLM TEST',
            evidence: [
              {
                id: 'coverage',
                text: 'Période partielle ou inconnue, sans conclusion définitive.',
                source: { month: m },
              },
              { id: 'flows', text: '4197.14 € de débits observés.' },
            ],
          }
    else if (path === '/api/category-series')
      data = {
        series: [{ period: m, 'Courses TEST': 4197.14 }],
        categories: [{ name: 'Courses TEST', color: '#9c99ff' }],
      }
    else if (path === '/api/analytics/tags-in-category')
      data = {
        category_total: 30.03,
        category_count: 2,
        overlap_count: 1,
        slices: [
          { tag: 'projet test', spent: 30.03, count: 2 },
          { tag: 'second test', spent: 10.01, count: 1 },
        ],
      }
    else if (path === '/api/analytics/tag-series') {
      const selected = q.get('month') || month
      const end = new Date(`${selected}-01T12:00:00Z`)
      end.setUTCMonth(end.getUTCMonth() + 1, 0)
      const first = new Date(`${selected}-01T12:00:00Z`)
      first.setUTCMonth(first.getUTCMonth() + 1 - Number(q.get('months') || 6))
      const from = q.get('from') || first.toISOString().slice(0, 10)
      const to =
        q.get('to') ||
        (selected === month
          ? new Date().toISOString().slice(0, 10)
          : end.toISOString().slice(0, 10))
      const buckets = []
      const cursor = new Date(`${from.slice(0, 7)}-01T12:00:00Z`)
      while (cursor.toISOString().slice(0, 7) <= to.slice(0, 7)) {
        buckets.push(cursor.toISOString().slice(0, 7))
        cursor.setUTCMonth(cursor.getUTCMonth() + 1)
      }
      data = {
        query_from: from,
        query_to: to,
        buckets,
        series: (q.get('tags') || '')
          .split(',')
          .filter(Boolean)
          .map((tag, index) => ({
            tag,
            values: buckets.map((_, i) =>
              i % 7 === 1
                ? null
                : [30.03, 45.67, 21.05, 32.15, 39.99, 28.05][(i + index) % 6],
            ),
          })),
      }
    } else if (path === '/api/tag-folders')
      data = {
        month: m,
        query_to: `${m}-21`,
        folders: [
          {
            name: 'projet test',
            total: 180.02,
            selected_month_total: 30.03,
            first_date: '2020-01-01',
            last_date: `${m}-10`,
            count: 3,
            days: 20,
            pinned: false,
          },
        ],
        others: [{ name: 'second test', total: 10.01, count: 1 }],
      }
    else if (path.startsWith('/api/tag-folders/') && req.method() === 'PUT') {
      preferences.push(req.postDataJSON())
      data = { ok: true }
    } else if (path.startsWith('/api/tag-folders/'))
      data = {
        name: 'projet test',
        total: 180.02,
        count: 3,
        days: 20,
        first_date: '2020-01-01',
        last_date: `${m}-10`,
        by_category: [{ category: 'Courses TEST', total: 180.02 }],
        by_merchant: [{ name: 'MARCHAND TEST', total: 180.02 }],
      }
    else if (path === '/api/transactions')
      data = {
        transactions: [],
        total: 0,
        limit: 100,
        offset: 0,
        query_from: q.get('from') || `${m}-01`,
        query_to: q.get('to') || `${m}-21`,
        summary: {
          income: 0,
          expenses: 0,
          net: 0,
          transfer_debits: 0,
          transfer_credits: 0,
          uncategorized_count: 0,
        },
      }
    else if (path === '/api/period')
      data = { month: m, is_current: m === month, completeness: 'unknown' }
    else if (
      [
        '/api/rules',
        '/api/merchants/aliases',
        '/api/merchants/unrecognized',
      ].includes(path)
    )
      data = []
    await route
      .fulfill({
        status,
        contentType: 'application/json',
        body: JSON.stringify(data),
      })
      .catch(() => {})
  })
  const family = (name) =>
    page
      .getByRole('navigation', { name: 'Familles d’analyse' })
      .getByRole('button', { name, exact: true })
  const detail = (name) =>
    page
      .getByRole('navigation', { name: 'Analyses détaillées' })
      .getByRole('button', { name, exact: true })
  const settle = () =>
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
  const open = async () => {
    await page
      .locator('.sidebar-nav button')
      .filter({ hasText: 'Analyse' })
      .click()
    await page
      .getByRole('heading', { name: 'Ce qui explique les sorties' })
      .waitFor()
  }
  await page.goto(process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176')
  await open()
  assert.match(
    await page.locator('.analysis-summary-grid').innerText(),
    /4\s?197,14/,
  )
  assert.equal(
    requests.filter((x) => x.startsWith('/api/intel/narrative')).length,
    0,
  )
  assert.equal(
    requests.filter((x) => x.startsWith('/api/intel/nature')).length,
    1,
  )
  assert.equal(
    requests.filter((x) => x.startsWith('/api/intel/merchants')).length,
    0,
  )
  // Native month control and cancellation: obsolete month never replaces current.
  delay = '/api/intel/nature'
  await page.getByLabel('Mois analysé').fill(previous)
  await page
    .getByRole('status')
    .filter({ hasText: 'Chargement de l’analyse' })
    .waitFor()
  await page.getByLabel('Mois analysé').fill(month)
  await page
    .getByRole('heading', { name: 'Ce qui explique les sorties' })
    .waitFor()
  assert.ok(
    (await page.locator('.analysis-context').innerText()).includes(
      'Mois en cours',
    ),
  )
  delay = ''
  await page.getByRole('button', { name: 'Prioriser avec l’IA' }).click()
  await page
    .getByText('Période partielle ou inconnue, sans conclusion définitive.')
    .waitFor()
  narrativeUnavailable = true
  await page
    .getByRole('button', { name: 'Actualiser la lecture assistée' })
    .click()
  await page
    .getByText('IA indisponible, constats déterministes accessibles.')
    .waitFor()
  await family('Flux financiers').click()
  await page
    .getByRole('heading', { name: 'Où vont les sorties observées ?' })
    .waitFor()
  assert.match(
    await page.locator('.analysis-definition').last().innerText(),
    /947,67/,
  )
  await detail('Par catégorie').click()
  await page
    .getByText('Aucune dépense sur cette sélection.')
    .waitFor({ state: 'hidden' })
  await page.locator('.recharts-bar-rectangle').first().hover()
  await page
    .locator('.recharts-tooltip-wrapper')
    .first()
    .waitFor({ state: 'visible' })
  await page.getByRole('button', { name: 'Courbes', exact: true }).click()
  await page.getByRole('button', { name: 'Trimestre', exact: true }).click()
  await page
    .getByText('Consulter les montants du graphique', { exact: true })
    .click()
  await page.locator('.analysis-table').waitFor()
  assert.ok(
    requests
      .filter((x) => x.startsWith('/api/category-series'))
      .every((x) => x.includes('from=') && x.includes('to=')),
  )
  await detail('Par marchand').click()
  await page
    .getByRole('button', { name: 'MARCHAND TEST', exact: true })
    .waitFor()
  await page.getByRole('button', { name: '3 mois', exact: true }).click()
  await page.getByLabel('Mois analysé').fill(previous)
  await page
    .getByRole('columnheader', { name: 'Variation', exact: true })
    .waitFor()
  await page.getByRole('cell', { name: '10 %', exact: true }).waitFor()
  await page.getByLabel('Mois analysé').fill(month)
  await page
    .getByRole('button', { name: 'MARCHAND TEST', exact: true })
    .waitFor()

  await settle()
  await page.getByRole('button', { name: 'MARCHAND TEST', exact: true }).click()
  await page.getByText('Marchand exact : MARCHAND TEST').waitFor()
  assert.ok(
    requests.some(
      (x) =>
        x.startsWith('/api/transactions') &&
        x.includes('merchant=MARCHAND+TEST') &&
        x.includes('type=debit'),
    ),
  )
  await open()
  await family('Habitudes').click()
  await page.getByText('Non établie', { exact: true }).first().waitFor()
  assert.equal(await page.getByText('nouveau', { exact: true }).count(), 0)
  await detail('Repérages').click()
  await page.getByRole('heading', { name: 'RECURRENCE TEST' }).waitFor()
  await page
    .getByRole('button', { name: 'Vérifier les transactions →', exact: true })
    .click()
  await page
    .getByRole('heading', { name: 'Transactions', exact: true })
    .waitFor()
  assert.ok(
    requests.some(
      (x) => x.startsWith('/api/transactions') && x.includes('ids=3%2C2%2C1'),
    ),
  )
  await open()
  await family('Pilotage').click()
  await page.getByRole('button', { name: /projet test.*180,02/ }).click()
  await page.getByRole('button', { name: 'voir les transactions →' }).waitFor()
  await page.getByRole('button', { name: "ce n'est pas un projet" }).click()
  await settle()
  assert.deepEqual(preferences, [{ state: 'hidden' }])
  const tagSource = page.waitForRequest(
    (r) =>
      r.url().includes('/api/transactions?') &&
      r.url().includes('tag=projet+test'),
  )
  await page.getByRole('button', { name: 'voir les transactions →' }).click()
  await page
    .getByRole('heading', { name: 'Transactions', exact: true })
    .waitFor()
  await tagSource
  assert.ok(
    requests.some(
      (x) =>
        x.startsWith('/api/transactions') &&
        x.includes('tag=projet+test') &&
        x.includes('type=debit'),
    ),
  )
  await open()
  await family('Pilotage').click()
  await detail('Analyse par tag').click()
  await page
    .getByText('Une opération peut porter plusieurs tags.', { exact: false })
    .waitFor()
  assert.match(
    await page
      .locator('.analysis-panel')
      .filter({
        has: page.getByRole('heading', {
          name: 'Dans une catégorie, quels tags ?',
        }),
      })
      .innerText(),
    /30,03/,
  )
  const history = page.locator('.analysis-panel').filter({
    has: page.getByRole('heading', {
      name: 'Évolution observée par tag',
      exact: true,
    }),
  })
  const duration = page.getByRole('combobox', {
    name: 'Période de l’historique',
  })
  const historyRequest = (months) =>
    page.waitForResponse((r) => {
      const u = new URL(r.url())
      return (
        u.pathname === '/api/analytics/tag-series' &&
        u.searchParams.get('months') === String(months)
      )
    })
  let requested = historyRequest(24)
  await duration.selectOption('24')
  await requested
  await history.locator('.recharts-area').first().waitFor()
  assert.equal(await history.locator('.recharts-area').count(), 2)
  assert.equal(
    await history
      .locator('.tag-pick')
      .filter({ hasText: 'projet test' })
      .getAttribute('aria-pressed'),
    'true',
  )
  assert.equal(await history.locator('linearGradient').count(), 2)
  await history
    .locator('summary')
    .filter({ hasText: 'Consulter les montants par tag' })
    .click()
  assert.equal(await history.locator('tbody tr').count(), 24)
  assert.ok(
    (await history.locator('tbody').innerText()).includes('Non observé'),
  )
  assert.deepEqual(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
      id: v.id,
      nodes: v.nodes.map((n) => n.target),
    })),
    [],
    'two-year tag history axe',
  )
  const historyArtifacts =
    process.env.THEMIS_ANALYSIS_ARTIFACTS || '/tmp/themis-analysis-ui'
  fs.mkdirSync(historyArtifacts, { recursive: true })
  await history
    .locator('summary')
    .filter({ hasText: 'Consulter les montants par tag' })
    .click()
  await history.screenshot({
    path: `${historyArtifacts}/tags-history-24-months.png`,
  })
  const gapTick = await history
    .locator('.recharts-xAxis .recharts-cartesian-axis-tick text')
    .first()
    .boundingBox()
  const plot = await history.locator('.recharts-wrapper').boundingBox()
  await page.mouse.move(gapTick.x + gapTick.width / 2, plot.y + 100)
  await history
    .locator('.recharts-tooltip-wrapper')
    .getByText('Non observé', { exact: true })
    .first()
    .waitFor()

  const slider = history.getByRole('slider').first()
  await slider.focus()
  await page.keyboard.press('ArrowRight')
  await settle()
  assert.ok(
    (await history.locator('.analysis-history-caption').innerText()).includes(
      '23 mois affichés',
    ),
    'keyboard zoom updates visible calendar',
  )
  requested = historyRequest(120)
  await duration.selectOption('120')
  await requested
  await history.locator('.recharts-area').first().waitFor()
  assert.ok(
    (await history.locator('.analysis-history-caption').innerText()).includes(
      '120 mois affichés',
    ),
  )
  await duration.selectOption('custom')
  await page
    .getByLabel('Début de l’historique')
    .fill(`${previous.slice(0, 4)}-01`)
  await page.getByLabel('Fin de l’historique').fill(previous)
  const custom = page.waitForResponse((r) => {
    const u = new URL(r.url())
    return (
      u.pathname === '/api/analytics/tag-series' &&
      u.searchParams.has('from') &&
      u.searchParams.has('to')
    )
  })
  await page.getByRole('button', { name: 'Appliquer la période' }).click()
  await custom
  await history.locator('.recharts-area').first().waitFor()
  assert.equal(
    await history
      .locator('.tag-pick')
      .filter({ hasText: 'projet test' })
      .getAttribute('aria-pressed'),
    'true',
  )
  await page.getByLabel('Début de l’historique').fill(month)
  await page.getByRole('button', { name: 'Appliquer la période' }).click()
  await page
    .getByRole('alert')
    .filter({ hasText: 'Le début doit précéder' })
    .waitFor()
  requested = historyRequest(6)
  await duration.selectOption('6')
  await requested
  await detail('Gérer les tags').click()
  await page.getByText('Outil global :', { exact: false }).waitFor()
  await page
    .getByRole('button', { name: 'renommer', exact: true })
    .first()
    .click()
  await page
    .getByRole('textbox', { name: 'Nouveau nom de projet test' })
    .waitFor()
  await page.keyboard.press('Escape')
  assert.equal(tagChanges.length, 0, 'Escape must not mutate a tag')
  await page
    .getByRole('button', { name: 'renommer', exact: true })
    .first()
    .click()
  await page
    .getByRole('textbox', { name: 'Nouveau nom de projet test' })
    .fill('renamed test')
  await page.keyboard.press('Enter')
  await page
    .getByText('« projet test » renommé en « renamed test ».', { exact: true })
    .waitFor()
  assert.equal(
    tagChanges.filter((c) => c.method === 'PATCH').length,
    1,
    'Enter and blur must not duplicate a rename',
  )
  await page
    .getByRole('button', { name: 'supprimer', exact: true })
    .first()
    .click()
  await page.getByRole('button', { name: 'non', exact: true }).click()
  assert.equal(
    tagChanges.filter((c) => c.method === 'DELETE').length,
    0,
    'cancelled deletion',
  )
  await page
    .getByRole('button', { name: 'supprimer', exact: true })
    .first()
    .click()
  await page.getByRole('button', { name: 'oui', exact: true }).click()
  await page
    .getByText('Tag « renamed test » supprimé.', { exact: true })
    .waitFor()
  assert.equal(tagChanges.filter((c) => c.method === 'DELETE').length, 1)

  // Failures and absence are distinct, with a working retry.
  fail = '/api/intel/nature'
  await family('Synthèse').click()
  await page.getByRole('button', { name: 'Mois précédent' }).click()
  await page.getByRole('alert').waitFor()
  fail = ''
  empty = true
  await page.getByRole('button', { name: 'Réessayer', exact: true }).click()
  await page
    .getByRole('heading', { name: 'Données insuffisantes pour ce mois' })
    .waitFor()
  empty = false
  await page.getByRole('button', { name: 'Mois suivant' }).click()
  await page
    .getByRole('heading', { name: 'Ce qui explique les sorties' })
    .waitFor()
  const artifacts =
    process.env.THEMIS_ANALYSIS_ARTIFACTS || '/tmp/themis-analysis-ui'
  fs.mkdirSync(artifacts, { recursive: true })
  // Native keyboard navigation and activation.
  await family('Flux financiers').focus()
  await page.keyboard.press('Enter')
  await detail('Nature des flux').waitFor()
  await family('Synthèse').click()
  for (const [width, height] of [
    [1920, 1080],
    [1280, 800],
    [960, 540],
    [768, 1024],
    [390, 844],
    [320, 700],
  ]) {
    await page.setViewportSize({ width, height })
    await settle()
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth + 1,
      ),
      `horizontal overflow ${width}`,
    )
    const axe = await new AxeBuilder({ page }).analyze()
    assert.deepEqual(
      axe.violations.map((v) => ({
        id: v.id,
        nodes: v.nodes.map((n) => n.target),
      })),
      [],
      `axe ${width}`,
    )
    await page.screenshot({
      path: `${artifacts}/analysis-${width}.png`,
      fullPage: true,
    })
  }
  await page.setViewportSize({ width: 1920, height: 1080 })
  await family('Flux financiers').click()
  await detail('Par catégorie').click()
  await settle()
  assert.deepEqual(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => v.id),
    [],
    'category chart axe',
  )
  await family('Pilotage').click()
  await detail('Analyse par tag').click()
  await settle()
  assert.deepEqual(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
      id: v.id,
      nodes: v.nodes.map((n) => n.target),
    })),
    [],
    'tags axe',
  )
  for (const width of [1920, 320]) {
    await page.setViewportSize({ width, height: width === 320 ? 700 : 1080 })
    for (const [group, section] of [
      ['Flux financiers', 'Par catégorie'],
      ['Flux financiers', 'Par marchand'],
      ['Habitudes', 'Comparaisons'],
      ['Habitudes', 'Repérages'],
      ['Pilotage', 'Projets par tag'],
      ['Pilotage', 'Analyse par tag'],
      ['Pilotage', 'Gérer les tags'],
    ]) {
      await family(group).click()
      await detail(section).click()
      await page
        .getByRole('status')
        .filter({ hasText: /Chargement/ })
        .waitFor({ state: 'hidden' })
        .catch(() => {})
      if (section === 'Analyse par tag' && width === 320) {
        const longRequest = historyRequest(24)
        await page
          .getByRole('combobox', { name: 'Période de l’historique' })
          .selectOption('24')
        await longRequest
        await page
          .locator('.analysis-tag-history-chart .recharts-area')
          .first()
          .waitFor()
        await page
          .getByRole('combobox', { name: 'Période de l’historique' })
          .selectOption('custom')
        await page
          .locator('.analysis-tag-history-chart')
          .screenshot({ path: `${historyArtifacts}/tags-history-mobile.png` })
      }
      await settle()
      assert.ok(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
        `${section} overflow ${width}`,
      )
      assert.deepEqual(
        (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
          id: v.id,
          nodes: v.nodes.map((n) => n.target),
        })),
        [],
        `${section} axe ${width}`,
      )
    }
  }
  // Layout equivalent to 200% on a 1920px viewport (960 CSS pixels).
  const cdp = await context.newCDPSession(page)
  await cdp.send('Emulation.setDeviceMetricsOverride', {
    width: 960,
    height: 540,
    deviceScaleFactor: 2,
    mobile: false,
  })
  await family('Synthèse').click()
  await settle()
  assert.ok(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
    '200% layout overflow',
  )
  await cdp.send('Emulation.clearDeviceMetricsOverride')
  await page.setViewportSize({ width: 1920, height: 1080 })
  await page.locator('.sidebar-footer .theme-toggle').click()
  await settle()
  assert.deepEqual(
    (await new AxeBuilder({ page }).analyze()).violations.map((v) => ({
      id: v.id,
      nodes: v.nodes.map((n) => n.target),
    })),
    [],
    'light theme Axe',
  )
  await page.screenshot({
    path: `${artifacts}/analysis-light.png`,
    fullPage: true,
  })
  assert.deepEqual(errors, [], 'browser exceptions')
  console.log(
    'Analysis browser: navigation, periods, exact sources, optional AI, errors/empty, charts, six layouts and Axe PASS',
  )
  await browser.close()
}
main().catch((error) => {
  console.error(error)
  process.exit(1)
})
