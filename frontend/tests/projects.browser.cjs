// Synthetic API for the real component; financial/security contracts are also tested in PostgreSQL.
const { chromium, expect } = require('@playwright/test')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const base = process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176'
async function main() {
  const browser = await chromium.launch({
    headless: true,
    channel: 'chromium',
  })
  try {
    const context = await browser.newContext({ reducedMotion: 'reduce' }),
      page = await context.newPage()
    const unexpected = [],
      errors = [],
      decisions = []
    let project = null,
      links = [],
      rules = [],
      listError = false,
      mutationError = false,
      reportError = false
    const transactions = [
      {
        id: 11,
        date: '2026-02-11',
        label: 'HOTEL TEST',
        category: 'Voyage TEST',
        amount_cents: -10001,
      },
      {
        id: 12,
        date: '2026-02-12',
        label: 'TRAIN TEST',
        category: 'Voyage TEST',
        amount_cents: -5002,
      },
    ]
    const current = () => {
      const confirmed = links.filter((t) => t.state === 'confirmed'),
        cost = confirmed.reduce((sum, t) => sum - t.amount_cents, 0),
        budget = project.budget_cents
      return {
        ...project,
        cost_cents: cost,
        transaction_count: confirmed.length,
        suggestion_count: links.filter((t) => t.state === 'suggested').length,
        remaining_cents: budget === null ? null : budget - cost,
        overrun_cents: budget === null ? null : Math.max(cost - budget, 0),
        consumed_pct:
          budget > 0 ? Number(((cost / budget) * 100).toFixed(2)) : null,
      }
    }
    page.on('pageerror', (e) => errors.push(e.message))
    await page.route('**/api/**', async (route) => {
      const request = route.request(),
        url = new URL(request.url()),
        path = url.pathname,
        method = request.method()
      let body,
        status = 200
      if (path === '/api/projects' && method === 'GET') {
        status = listError ? 500 : 200
        body = listError
          ? { error: 'Projets indisponibles TEST' }
          : {
              projects: project ? [current()] : [],
              query_to: '2026-02-28',
              has_more: false,
            }
      } else if (path === '/api/projects' && method === 'POST') {
        project = { id: 1, ...request.postDataJSON() }
        body = { id: 1 }
      } else if (path === '/api/projects/1' && method === 'PUT') {
        project = { ...project, ...request.postDataJSON() }
        body = { id: 1 }
      } else if (path === '/api/projects/1') {
        if (reportError) {
          status = 500
          body = { error: 'Rapport indisponible TEST' }
        } else {
          const p = current(),
            slices = p.transaction_count
              ? [
                  {
                    name: 'Voyage TEST',
                    amount_cents: p.cost_cents,
                    count: p.transaction_count,
                  },
                ]
              : []
          body = {
            project: p,
            query_from: project.start_date,
            query_to: '2026-02-28',
            last_observed: p.transaction_count ? '2026-02-12' : '',
            coverage: 'provisional',
            coverage_reason: 'Relevés incomplets TEST.',
            by_category: slices,
            by_merchant: slices,
            by_month: p.transaction_count
              ? [{ ...slices[0], name: '2026-02' }]
              : [],
            by_tag: slices,
            excluded_confirmed_count: 0,
            observations:
              p.overrun_cents > 0
                ? [
                    {
                      id: 'project:1:budget_overrun:2026-02-28',
                      type: 'budget_overrun',
                      text: 'Le coût confirmé observé dépasse le budget.',
                      values: {
                        overrun_cents: p.overrun_cents,
                        cost_cents: p.cost_cents,
                      },
                      source: { project_id: 1 },
                      validity:
                        'Dépenses confirmées seulement ; couverture incomplète.',
                    },
                  ]
                : [],
          }
        }
      } else if (path === '/api/tags') body = [{ id: 41, name: 'Voyage TEST' }]
      else if (path === '/api/categories')
        body = [{ id: 51, name: 'Transport TEST' }]
      else if (path === '/api/projects/1/rules' && method === 'GET')
        body = rules
      else if (path === '/api/projects/1/rules') {
        const rule = { id: 1, ...request.postDataJSON(), active: true }
        rules = [rule]
        body = rule
      } else if (path === '/api/projects/1/rules/1') {
        rules = []
        body = { ok: true }
      } else if (path === '/api/projects/1/suggestions') {
        if (!links.length)
          links = transactions.map((t) => ({
            ...t,
            state: 'suggested',
            eligible: true,
            included: false,
            reason: {
              kind: 'tag',
              tag_id: 41,
              from_date: '2026-02-01',
              to_date: '2026-02-28',
            },
          }))
        body = { inserted: 2 }
      } else if (path === '/api/projects/1/transactions')
        body = {
          transactions: links.filter(
            (t) => t.state === (url.searchParams.get('state') || 'confirmed'),
          ),
          has_more: false,
        }
      else if (path === '/api/projects/1/decisions') {
        const data = request.postDataJSON()
        decisions.push(data)
        if (mutationError) {
          status = 500
          body = { error: 'Décision annulée TEST' }
        } else {
          for (const id of data.transaction_ids) {
            let link = links.find((t) => t.id === id)
            if (!link) {
              link = {
                ...transactions.find((t) => t.id === id),
                reason: { kind: 'manual' },
                eligible: true,
              }
              links.push(link)
            }
            link.state = data.state
            link.included = data.state === 'confirmed'
          }
          body = { ok: true }
        }
      } else if (path === '/api/projects/1/history')
        body = {
          history: [
            {
              id: 1,
              transaction_id: 11,
              event: 'decision',
              after: 'confirmed',
              changed_at: '2026-02-28 TEST',
            },
          ],
          has_more: false,
        }
      else if (path === '/api/transactions') body = { transactions, total: 2 }
      else {
        unexpected.push(`${method} ${path}`)
        status = 500
        body = { error: 'Unexpected TEST request' }
      }
      await route.fulfill({
        status,
        contentType: 'application/json',
        body: JSON.stringify(body),
      })
    })
    await page.route('**/__project_test__', (route) =>
      route.fulfill({
        contentType: 'text/html',
        body: `<!doctype html><html lang="fr"><head><title>Projets TEST</title><meta name="viewport" content="width=device-width,initial-scale=1"></head><body><main id="root" style="padding:16px"></main><script type="module">
   import RefreshRuntime from '/@react-refresh';RefreshRuntime.injectIntoGlobalHook(window);window.$RefreshReg$=()=>{};window.$RefreshSig$=()=>type=>type;window.__vite_plugin_react_preamble_installed__=true;
   import React from '/node_modules/.vite/deps/react.js';import ReactDOM from '/node_modules/.vite/deps/react-dom_client.js';
   const {default:FinancialProjects}=await import('/src/components/FinancialProjects.jsx');
   import '/src/styles.css';import '/src/cockpit.css';window.__navigation=[];
   ReactDOM.createRoot(document.getElementById('root')).render(React.createElement(FinancialProjects,{month:'2026-02',onNavigate:(...args)=>window.__navigation.push(args)}));
   </script></body></html>`,
      }),
    )
    await page.goto(`${base}/__project_test__`)
    await expect(
      page.getByText('Aucun projet créé.', { exact: false }),
    ).toBeVisible()
    await page
      .getByRole('button', { name: 'Créer un projet', exact: true })
      .click()
    await page.getByLabel('Nom du projet').fill('Voyage TEST')
    await page.getByLabel('Budget prévisionnel (€), facultatif').fill('100,00')
    await page.getByLabel('Date de début', { exact: true }).fill('2026-02-01')
    await page.getByLabel('Date de fin', { exact: true }).fill('2026-02-28')
    await page.getByRole('button', { name: 'Enregistrer le projet' }).click()
    const detail = page.getByRole('article', { name: 'Détail du projet' })
    await expect(detail).toContainText('Aucune dépense confirmée')
    assert.equal(project.budget_cents, 10000)
    await page.getByText('Configurer les règles', { exact: true }).click()
    await page.getByLabel('Valeur du critère').selectOption('41')
    await page.getByRole('button', { name: 'Ajouter et rechercher' }).click()
    await expect(page.locator('.project-card')).toContainText('2 suggestion(s)')
    await expect(detail).toContainText('0,00')
    await page.getByLabel('Afficher les opérations').selectOption('suggested')
    await expect(
      page.getByRole('table', { name: /suggestions exclues/ }),
    ).toContainText('HOTEL TEST')
    await page.getByLabel('Sélectionner HOTEL TEST').check()
    await page.getByLabel('Sélectionner TRAIN TEST').check()
    await page
      .getByRole('button', { name: 'Confirmer la sélection (2)' })
      .click()
    await expect(detail).toContainText('150,03')
    await expect(detail).toContainText('50,03')
    assert.deepEqual(decisions[0], {
      transaction_ids: [11, 12],
      state: 'confirmed',
    })
    await expect(detail).toContainText(
      'Le coût confirmé observé dépasse le budget.',
    )
    await page
      .getByRole('button', { name: 'Examiner les sources de ce constat' })
      .click()
    await expect(
      page.getByRole('table', { name: /confirmations et éligibilité/ }),
    ).toContainText('HOTEL TEST')
    await page.getByRole('button', { name: 'Voir l’opération nº11' }).click()
    assert.deepEqual(await page.evaluate(() => window.__navigation), [
      ['transactions', { ids: [11] }],
    ])
    mutationError = true
    await page
      .getByRole('row')
      .filter({ hasText: 'HOTEL TEST' })
      .getByRole('button', { name: 'Retirer', exact: true })
      .click()
    await expect(page.getByRole('alert')).toContainText('Décision annulée TEST')
    await expect(detail).toContainText('150,03')
    mutationError = false
    await page
      .getByRole('row')
      .filter({ hasText: 'HOTEL TEST' })
      .getByRole('button', { name: 'Retirer', exact: true })
      .click()
    await expect(detail).toContainText('50,02')
    await page.getByLabel('Afficher les opérations').selectOption('rejected')
    await expect(
      page.getByRole('table', { name: /décisions conservées/ }),
    ).toContainText('HOTEL TEST')
    await page
      .getByRole('row')
      .filter({ hasText: 'HOTEL TEST' })
      .getByRole('button', { name: 'Confirmer', exact: true })
      .click()
    await expect(detail).toContainText('150,03')
    await page.getByLabel('Afficher les opérations').selectOption('confirmed')
    await page
      .getByRole('row')
      .filter({ hasText: 'TRAIN TEST' })
      .getByRole('button', { name: 'Retirer', exact: true })
      .click()
    await expect(detail).toContainText('100,01')
    await page
      .getByText('Associer manuellement une dépense', { exact: true })
      .click()
    await page.getByLabel('Marchand ou libellé').fill('TRAIN TEST')
    await page.getByRole('button', { name: 'Rechercher', exact: true }).click()
    await page
      .getByRole('button', { name: 'Associer l’opération nº12', exact: true })
      .click()
    await expect(detail).toContainText('150,03')
    await page
      .getByRole('button', { name: 'Modifier le projet', exact: true })
      .click()
    await page.getByLabel('Budget prévisionnel (€), facultatif').fill('')
    await page.getByRole('button', { name: 'Enregistrer le projet' }).click()
    await expect(detail).toContainText('Non défini')
    assert.equal(project.budget_cents, null)
    await page
      .getByRole('button', { name: 'Voir l’historique des décisions' })
      .click()
    await expect(detail).toContainText('opération nº11')
    for (const theme of ['light', 'dark']) {
      await page.evaluate(
        (theme) => (document.documentElement.dataset.theme = theme),
        theme,
      )
      for (const width of [1440, 768, 390, 320]) {
        await page.setViewportSize({ width, height: 1000 })
        await page.evaluate(() => document.fonts.ready)
        await expect
          .poll(() =>
            page.evaluate(
              () =>
                document
                  .getAnimations()
                  .filter((a) => a.playState === 'running').length,
            ),
          )
          .toBe(0)
        assert.ok(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth + 1,
          ),
          `${theme} overflow ${width}: ${JSON.stringify(
            await page.evaluate(() =>
              [...document.querySelectorAll('main *')]
                .filter(
                  (el) => el.getBoundingClientRect().right > innerWidth + 1,
                )
                .slice(0, 15)
                .map((el) => ({
                  tag: el.tagName,
                  cls: el.className,
                  width: el.getBoundingClientRect().width,
                  text: el.textContent.slice(0, 70),
                })),
            ),
          )}`,
        )
        const result = await new AxeBuilder({ page })
          .include('main')
          .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
          .analyze()
        assert.deepEqual(
          result.violations.map((v) => ({
            id: v.id,
            nodes: v.nodes.map((n) => ({
              target: n.target,
              failure: n.failureSummary,
            })),
          })),
          [],
          `${theme} Axe ${width}`,
        )
      }
    }
    reportError = true
    await page
      .getByRole('button', { name: 'Rechercher les suggestions' })
      .click()
    await expect(page.getByRole('alert')).toContainText(
      'Rapport indisponible TEST',
    )
    await expect(detail).not.toContainText('Coût brut confirmé')
    reportError = false
    await detail.getByRole('button', { name: 'Réessayer' }).click()
    await expect(detail).toContainText('Coût brut confirmé')
    listError = true
    await page.reload()
    await expect(page.getByRole('alert')).toContainText(
      'Projets indisponibles TEST',
    )
    await expect(
      page.getByText('Aucun projet créé.', { exact: false }),
    ).toHaveCount(0)
    listError = false
    await page.getByRole('button', { name: 'Réessayer' }).click()
    await expect(page.locator('.project-card')).toContainText('Voyage TEST')
    assert.deepEqual(errors, [])
    assert.deepEqual(unexpected, [])
    console.log(
      'Projects browser PASS: create/edit, optional budget, suggestions, reversible batch confirmation, exact cents, source IDs, history, errors, responsive and light/dark Axe at four widths.',
    )
  } finally {
    await browser.close()
  }
}
main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
