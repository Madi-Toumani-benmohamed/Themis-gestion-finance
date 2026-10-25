// Every API is intercepted. Synthetic sources only; no real bank data.
const { chromium } = require('playwright')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const accounts = [
  {
    id: 11,
    name: 'Courant TEST',
    kind: 'courant',
    tracking_mode: 'statements',
  },
  { id: 12, name: 'Livret TEST', kind: 'livret_a', tracking_mode: 'manual' },
  { id: 13, name: 'PEA TEST', kind: 'bourse', tracking_mode: 'manual' },
]
const st = (id, extra = {}) => ({
  id,
  filename: `cycle-${id}-TEST.pdf`,
  account_id: 11,
  account_name: 'Courant TEST',
  tracking_mode: 'statements',
  period_start: '2026-09-22',
  period_end: '2026-10-21',
  uploaded_at: '2026-10-23',
  parser: 'local',
  tx_count: 2,
  opening_balance_cents: 100000,
  closing_balance_cents: 90001,
  credits_cents: 1,
  debits_cents: 10000,
  sum_cents: -9999,
  theoretical_balance_cents: 90001,
  difference_cents: 0,
  validated: true,
  effective_validated: true,
  status: 'validated',
  can_validate: true,
  validation_source: 'automatic_legacy',
  controls: [
    {
      code: 'reconciliation',
      status: 'passed',
      severity: 'info',
      detail:
        'Égalité exacte au centime. Ce contrôle ne prouve pas à lui seul la complétude.',
    },
  ],
  history: [],
  ...extra,
})
let rows = [
  st(101),
  st(102, {
    filename: 'controle-CSV-TEST.csv',
    period_start: null,
    period_end: null,
    opening_balance_cents: null,
    closing_balance_cents: null,
    theoretical_balance_cents: null,
    difference_cents: null,
    validated: false,
    effective_validated: false,
    status: 'to_review',
    can_validate: false,
    controls: [
      {
        code: 'reconciliation',
        status: 'unknown',
        severity: 'warning',
        detail: 'Soldes absents : contrôle non exécuté.',
      },
    ],
  }),
  st(103, {
    filename: 'revue-TEST.pdf',
    validated: false,
    effective_validated: false,
    status: 'analyzed',
    validation_source: 'none',
  }),
]
rows.push(
  st(104, {
    filename: 'ecart-TEST.pdf',
    closing_balance_cents: 90101,
    difference_cents: 100,
    effective_validated: false,
    status: 'failed',
    can_validate: false,
    controls: [
      {
        code: 'reconciliation',
        status: 'failed',
        severity: 'error',
        detail: 'Écart comptable confirmé de 1 euro TEST.',
      },
    ],
  }),
)
let failLoad = true,
  failDelete = false,
  failReview = false,
  delay = 0,
  failChecks = false,
  aiUnavailable = false
const mutations = [],
  requests = []
async function main() {
  const browser = await chromium.launch({ headless: true, channel: 'chromium' })
  try {
    const context = await browser.newContext({
      viewport: { width: 1920, height: 1080 },
    })
    const page = await context.newPage()
    page.setDefaultTimeout(10000)
    const errors = []
    page.on('pageerror', (e) => errors.push(e.message))
    await page.addInitScript(() => localStorage.setItem('themis-theme', 'dark'))
    await page.route('**/api/**', async (route) => {
      const r = route.request(),
        url = new URL(r.url()),
        path = url.pathname,
        method = r.method()
      requests.push({ path, query: url.search })
      let data = {},
        status = 200
      if (path === '/api/auth/status') data = { setup_required: false }
      else if (path === '/api/auth/me')
        data = { id: 1, username: 'DOCUMENT TEST', role: 'user' }
      else if (path === '/api/health') data = { ai_enabled: true }
      else if (path === '/api/accounts') data = accounts
      else if (path === '/api/summary')
        data = { monthly: [], by_category: [], uncategorized: 0 }
      else if (path === '/api/hero' || path === '/api/remaining')
        data = { available: false }
      else if (path === '/api/journal') data = { events: [] }
      else if (path === '/api/anomalies') data = { anomalies: [] }
      else if (path === '/api/statements') {
        if (delay) await new Promise((resolve) => setTimeout(resolve, delay))
        if (failLoad) {
          status = 500
          data = { error: 'ERREUR RELEVES TEST' }
        } else data = rows
      } else if (path === '/api/reliability') {
        const month = url.searchParams.get('month')
        data = {
          month,
          current: month === '2026-10',
          through: month === '2026-09' ? '2026-09-30' : '2026-10-25',
          accounts: accounts.map((a) => ({
            ...a,
            status:
              a.tracking_mode === 'manual'
                ? 'not_required'
                : month === '2026-09'
                  ? 'documented'
                  : 'insufficient',
            gaps:
              a.tracking_mode === 'manual' || month === '2026-09'
                ? []
                : [{ from: '2026-10-22', to: '2026-10-25' }],
            statements:
              a.tracking_mode === 'manual'
                ? []
                : [
                    {
                      id: 101,
                      filename: 'cycle-101-TEST.pdf',
                      from: '2026-09-22',
                      to: '2026-10-21',
                      period_known: true,
                      validated: true,
                    },
                  ],
            to_review: 0,
            unknown_periods: 0,
            last_import: '2026-10-23',
            last_transaction: '2026-10-20',
          })),
        }
      } else if (/^\/api\/statements\/\d+\/checks$/.test(path)) {
        const id = Number(path.split('/')[3]),
          mode = r.postDataJSON().mode
        if (failChecks) {
          status = 500
          data = { error: 'CONTROLES INDISPONIBLES TEST' }
        } else
          data = {
            statement: rows.find((st) => st.id === id),
            mode,
            checked_at: '2026-10-25T12:00:00Z',
            ai: {
              status:
                mode === 'deterministic'
                  ? 'not_requested'
                  : aiUnavailable
                    ? 'unavailable'
                    : 'available',
              detail:
                mode === 'deterministic'
                  ? 'CONTROLES SANS IA TEST'
                  : aiUnavailable
                    ? 'IA INDISPONIBLE TEST · contrôles comptables conservés'
                    : 'RELECTURE IA CONSULTATIVE TEST',
              evidence:
                mode === 'combined' && !aiUnavailable
                  ? [
                      {
                        id: 'coverage',
                        text: 'Pas de nouvelle extraction du PDF original.',
                      },
                    ]
                  : [],
              model:
                mode === 'combined' && !aiUnavailable
                  ? 'TEST MODEL'
                  : undefined,
            },
          }
      } else if (/^\/api\/statements\/\d+/.test(path)) {
        const id = Number(path.split('/')[3])
        mutations.push({
          id,
          method,
          body: method === 'PATCH' ? r.postDataJSON() : null,
        })
        if (method === 'DELETE') {
          if (failDelete) {
            status = 500
            data = { error: 'ERREUR SUPPRESSION TEST' }
          } else {
            rows = rows.filter((st) => st.id !== id)
            status = 204
          }
        } else if (failReview) {
          status = 409
          data = { error: 'REVUE REFUSEE TEST' }
        } else {
          const body = r.postDataJSON()
          rows = rows.map((st) =>
            st.id === id
              ? {
                  ...st,
                  validated: body.action === 'validate',
                  effective_validated: body.action === 'validate',
                  validation_source:
                    body.action === 'validate' ? 'manual' : 'reopened',
                  status: body.action === 'validate' ? 'validated' : 'analyzed',
                  history: [
                    {
                      id: mutations.length,
                      action: body.action,
                      note: body.note,
                      at: '2026-10-25',
                    },
                    ...st.history,
                  ],
                }
              : st,
          )
          status = 204
        }
      } else if (path === '/api/transactions')
        data = {
          transactions: [],
          total: 0,
          offset: 0,
          limit: 100,
          summary: {
            expenses: 0,
            income: 0,
            net: 0,
            transfer_debits: 0,
            transfer_credits: 0,
          },
        }
      else if (
        ['/api/domains', '/api/categories', '/api/tags', '/api/rules'].includes(
          path,
        )
      )
        data = []
      await route
        .fulfill({
          status,
          contentType: 'application/json',
          body: status === 204 ? undefined : JSON.stringify(data),
        })
        .catch(() => {})
    })
    await page.goto(process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176')
    const nav = () =>
      page
        .locator('.sidebar-nav')
        .getByRole('tab', { name: 'Relevés', exact: true })
    await nav().click()
    await page
      .getByRole('alert')
      .filter({ hasText: 'ERREUR RELEVES TEST' })
      .waitFor()
    failLoad = false
    delay = 150
    await page.getByRole('button', { name: 'Réessayer les relevés' }).click()
    await page
      .getByRole('status')
      .filter({ hasText: 'Chargement des relevés' })
      .waitFor()
    delay = 0
    await page
      .getByRole('button', { name: 'Actions pour cycle-101-TEST.pdf' })
      .waitFor()
    const root = page.locator('.statements-page'),
      dialog = () => page.getByRole('dialog')
    assert.ok(
      (await root.locator('.statements-table').boundingBox()).width > 1200,
      'Large-screen table must use the available width',
    )
    assert.match(await root.locator('.statement-summary').innerText(), /4/)
    assert.match(
      await root.locator('.data-reliability-overview').innerText(),
      /1 comptes soumis/,
    )
    assert.match(await root.innerText(), /2 compte\(s\) suivis par pointage/)
    const statusColors = async () => {
      const values = await Promise.all(
        ['validated', 'to_review', 'failed'].map((status) =>
          root
            .locator(`.statement-status.${status}`)
            .first()
            .evaluate((el) => getComputedStyle(el).color),
        ),
      )
      assert.equal(
        new Set(values).size,
        3,
        'Statuses must have distinct colors',
      )
      assert.match(
        await root.locator('.statement-status.failed').first().innerText(),
        /Contrôles en échec/,
      )
    }
    await statusColors()
    await page
      .getByRole('combobox', { name: 'Statut', exact: true })
      .selectOption('failed')
    assert.equal(await root.locator('tbody tr').count(), 1)
    await root
      .getByRole('button', { name: 'Réinitialiser les filtres' })
      .click()
    await page
      .getByRole('button', { name: 'Actions pour ecart-TEST.pdf' })
      .click()
    assert.equal(
      await dialog()
        .getByRole('button', { name: 'Valider le relevé', exact: true })
        .isDisabled(),
      true,
    )
    assert.ok(
      await dialog()
        .getByRole('heading', { name: 'Actions sur ce relevé' })
        .isVisible(),
    )
    const beforeChecks = mutations.length
    await dialog()
      .getByRole('button', { name: 'Relancer les vérifications' })
      .click()
    await dialog()
      .getByText('CONTROLES SANS IA TEST', { exact: true })
      .waitFor()
    assert.equal(
      mutations.length,
      beforeChecks,
      'Rechecking must not save a validation',
    )
    await dialog()
      .getByRole('combobox', { name: 'Mode de vérification' })
      .selectOption('combined')
    await dialog()
      .getByRole('button', { name: 'Relancer les vérifications' })
      .click()
    await dialog()
      .getByText('RELECTURE IA CONSULTATIVE TEST', { exact: true })
      .waitFor()
    assert.equal(
      await dialog()
        .getByRole('button', { name: 'Valider le relevé', exact: true })
        .isDisabled(),
      true,
      'AI cannot override confirmed failure',
    )
    aiUnavailable = true
    await dialog()
      .getByRole('button', { name: 'Relancer les vérifications' })
      .click()
    await dialog()
      .getByText('IA INDISPONIBLE TEST · contrôles comptables conservés', {
        exact: true,
      })
      .waitFor()
    aiUnavailable = false
    failChecks = true
    await dialog()
      .getByRole('button', { name: 'Relancer les vérifications' })
      .click()
    await dialog()
      .getByRole('alert')
      .filter({ hasText: 'CONTROLES INDISPONIBLES TEST' })
      .waitFor()
    failChecks = false
    await page.keyboard.press('Escape')

    await page.getByLabel('Mois à examiner').fill('2026-09')
    await root.getByText('sur le mois', { exact: false }).waitFor()
    await page
      .getByRole('searchbox', { name: 'Recherche', exact: true })
      .fill('CSV')
    assert.equal(await root.locator('tbody tr').count(), 1)
    await root
      .getByRole('button', { name: 'Réinitialiser les filtres' })
      .click()
    await page.getByLabel('Statut', { exact: true }).selectOption('analyzed')
    assert.equal(await root.locator('tbody tr').count(), 1)
    await root
      .getByRole('button', { name: 'Réinitialiser les filtres' })
      .click()
    await page
      .getByRole('button', { name: 'Actions pour controle-CSV-TEST.csv' })
      .click()
    await dialog().waitFor()
    assert.match(await dialog().innerText(), /contrôle non exécuté/)
    await dialog()
      .getByLabel('Justification de la revue')
      .fill('Examen original TEST')
    assert.equal(
      await dialog()
        .getByRole('button', { name: 'Valider le relevé' })
        .isDisabled(),
      true,
    )
    await page.keyboard.press('Escape')
    await page
      .getByRole('button', { name: 'Actions pour revue-TEST.pdf' })
      .click()
    await dialog()
      .getByLabel('Justification de la revue')
      .fill('Examen du document original TEST')
    failReview = true
    await dialog().getByRole('button', { name: 'Valider le relevé' }).click()
    await dialog().getByRole('alert').waitFor()
    assert.equal(rows.find((st) => st.id === 103).validated, false)
    failReview = false
    await dialog().getByRole('button', { name: 'Valider le relevé' }).click()
    await dialog().waitFor({ state: 'detached' })
    await page
      .getByRole('button', { name: 'Actions pour revue-TEST.pdf' })
      .click()
    assert.match(await dialog().innerText(), /Confirmation manuelle tracée/)
    await dialog()
      .getByRole('button', { name: 'Consulter les 2 opérations' })
      .click()
    await page
      .getByRole('button', { name: 'Retirer le filtre de provenance' })
      .waitFor()
    assert.ok(
      requests.some(
        (r) =>
          r.path === '/api/transactions' &&
          new URLSearchParams(r.query).get('statement_id') === '103' &&
          !new URLSearchParams(r.query).has('month'),
      ),
    )
    await nav().click()
    await page
      .getByRole('button', { name: 'Supprimer cycle-101-TEST.pdf' })
      .click()
    const before = mutations.length
    await dialog()
      .getByRole('button', { name: 'Annuler la suppression' })
      .click()
    assert.equal(mutations.length, before)
    await page
      .getByRole('button', { name: 'Supprimer cycle-101-TEST.pdf' })
      .click()
    assert.match(await dialog().innerText(), /irréversible/)
    failDelete = true
    await dialog()
      .getByRole('button', { name: 'Confirmer la suppression de 2 opérations' })
      .click()
    await dialog().getByRole('alert').waitFor()
    assert.equal(rows.length, 4)
    failDelete = false
    await dialog()
      .getByRole('button', { name: 'Confirmer la suppression de 2 opérations' })
      .click()
    await dialog().waitFor({ state: 'detached' })
    await page
      .getByRole('button', { name: 'Actions pour revue-TEST.pdf' })
      .waitFor()
    const artifacts =
      process.env.THEMIS_STATEMENTS_ARTIFACTS || '/tmp/themis-statements-ui'
    fs.mkdirSync(artifacts, { recursive: true })
    for (const width of [1920, 960, 768, 390, 320]) {
      await page.setViewportSize({ width, height: 1080 })
      await page.evaluate(async () => {
        await document.fonts.ready
      })
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
        true,
        `overflow ${width}`,
      )
      const axe = await new AxeBuilder({ page })
        .include('.statements-page')
        .analyze()
      assert.deepEqual(
        axe.violations.map((v) => ({
          id: v.id,
          nodes: v.nodes.map((n) => n.target),
        })),
        [],
        `axe ${width}`,
      )
      await page
        .getByRole('button', { name: 'Actions pour revue-TEST.pdf' })
        .click()
      await dialog().waitFor()
      assert.equal(
        await dialog().evaluate((el) => el.scrollWidth <= el.clientWidth),
        true,
      )
      const daxe = await new AxeBuilder({ page })
        .include('.statement-dialog')
        .analyze()
      assert.deepEqual(
        daxe.violations.map((v) => ({
          id: v.id,
          nodes: v.nodes.map((n) => n.target),
        })),
        [],
      )
      for (let i = 0; i < 8; i++) {
        await page.keyboard.press('Tab')
        assert.equal(
          await page.evaluate(() =>
            document.querySelector('dialog').contains(document.activeElement),
          ),
          true,
        )
      }
      await page.keyboard.press('Escape')
      await page.evaluate(() => scrollTo(0, 0))
      await page.screenshot({
        path: `${artifacts}/statements-${width}.png`,
        fullPage: false,
      })
    }
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.locator('.sidebar-footer .theme-toggle').click()
    await page.evaluate(async () => {
      await Promise.all(
        document
          .getAnimations()
          .filter((a) => Number.isFinite(a.effect.getComputedTiming().endTime))
          .map((a) => a.finished.catch(() => {})),
      )
    })
    await statusColors()
    const light = await new AxeBuilder({ page })
      .include('.statements-page')
      .analyze()
    assert.deepEqual(
      light.violations.map((v) => ({
        id: v.id,
        nodes: v.nodes.map((n) => ({
          target: n.target,
          failure: n.failureSummary,
        })),
      })),
      [],
    )
    rows = []
    await page
      .locator('.sidebar-nav')
      .getByRole('tab', { name: 'Transactions', exact: true })
      .click()
    await nav().click()
    await page
      .getByText(
        'Aucun relevé importé. Les comptes suivis par pointage restent gérés dans Comptes.',
        { exact: true },
      )
      .waitFor()
    assert.deepEqual(errors, [])
    console.log(
      'Relevés navigateur PASS : modes, périodes, filtres, contrôles, validation, erreurs, suppression, provenance, clavier, cinq largeurs et Axe clair/sombre.',
    )
  } finally {
    await browser.close()
  }
}
main().catch((e) => {
  console.error(e)
  process.exit(1)
})
