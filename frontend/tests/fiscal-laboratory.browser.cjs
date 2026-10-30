// Synthetic fixtures. Optional THEMIS_LAB_REAL_API=1 uses a disposable local API.
const { chromium, expect } = require('@playwright/test')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const base = process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176'
const real = process.env.THEMIS_LAB_REAL_API === '1'
async function main() {
  const browser = await chromium.launch({ headless: true, channel: 'chromium' })
  try {
    const context = await browser.newContext({ reducedMotion: 'reduce' }),
      page = await context.newPage(),
      errors = []
    page.on('pageerror', (e) => {
      errors.push(e.message)
      console.error('PAGE ERROR', e.message)
    })
    page.on('dialog', (d) => d.accept())
    let documents = [],
      next = 1,
      writes = 0,
      failSave = false
    if (real) {
      const st = await (
        await context.request.get(`${base}/api/auth/status`)
      ).json()
      const r = await context.request.post(
        `${base}/api/auth/${st.setup_required ? 'setup' : 'login'}`,
        {
          data: {
            username: 'LAB SYNTHETIC TEST',
            password: 'Synthetic-Lab-Only-42!',
          },
        },
      )
      assert.ok(r.ok(), await r.text())
    }
    const rule = {
      income_year: 2025,
      assessment_year: 2026,
      status: 'official',
      version: 'fr-ir-2025-v1',
      verified_on: '2026-10-30',
      sources: [
        'https://www.impots.gouv.fr/www2/fichiers/documentation/brochure/ir_2026/pdf_som/21-calcul_impot_369a382.pdf',
      ],
    }
    await page.route('**/api/**', async (route) => {
      const req = route.request(),
        path = new URL(req.url()).pathname,
        method = req.method()
      if (['POST', 'PUT', 'DELETE'].includes(method)) writes++
      if (failSave && path.startsWith('/api/laboratory') && method === 'PUT')
        return route.fulfill({
          status: 409,
          json: { error: 'Révision modifiée TEST' },
        })
      if (real) return route.continue()
      let body,
        status = 200
      if (path === '/api/fiscal/rules') body = [rule]
      else if (path === '/api/fiscal/simulate') {
        const d = req.postDataJSON()
        assert.equal(d.income_cents, 3000000)
        const tax = 210400,
          paid = d.withholding_observed_cents,
          future =
            d.future_monthly_withholding_cents === null
              ? null
              : d.future_monthly_withholding_cents * (12 - d.withholding_months)
        if (paid !== null && d.withholding_year !== 2025) {
          status = 400
          body = {
            error:
              'Les prélèvements doivent concerner la même année de revenus.',
          }
        } else
          body = {
            rules: rule,
            half_parts: 2,
            annual_income_cents: 3000000,
            salary_deduction_cents: 0,
            taxable_cents: 3000000,
            gross_tax_cents: tax,
            discount_cents: 0,
            tax_cents: tax,
            average_rate_pct: '7.01',
            marginal_rate_pct: 30,
            observed_paid_cents: paid,
            projected_future_paid_cents: future,
            gap_before_future_cents: paid === null ? null : tax - paid,
            settlement_cents:
              paid === null || (d.withholding_months < 12 && future === null)
                ? null
                : tax - paid - (future || 0),
            settlement_status:
              paid === null
                ? 'unknown'
                : future !== null
                  ? 'projected'
                  : d.withholding_months === 12
                    ? 'annual_estimate'
                    : 'incomplete',
            steps: [{ label: 'Impôt estimé', amount_cents: tax }],
            assumptions: [],
            limitations: ['Estimation non officielle.'],
          }
      } else if (path === '/api/laboratory' && method === 'GET')
        body = { documents, has_more: false }
      else if (path === '/api/laboratory' && method === 'POST') {
        body = { ...req.postDataJSON(), id: next++, revision: 1 }
        documents.push(body)
        status = 201
      } else {
        const id = Number(path.split('/').at(-1)),
          old = documents.find((d) => d.id === id)
        if (!old) {
          status = 404
          body = { error: 'Absent TEST' }
        } else if (method === 'GET') body = old
        else if (method === 'PUT') {
          body = { ...req.postDataJSON(), id, revision: old.revision + 1 }
          documents = documents.map((d) => (d.id === id ? body : d))
        } else if (method === 'DELETE') {
          documents = documents.filter((d) => d.id !== id)
          body = { ok: true }
        } else throw new Error(`Unexpected ${method} ${path}`)
      }
      await route.fulfill({ status, json: body })
    })
    await page.route('**/__fiscal_lab_test__', (r) =>
      r.fulfill({
        contentType: 'text/html',
        body: `<!doctype html><html lang="fr"><head><title>Fiscalité Laboratoire TEST</title><meta name="viewport" content="width=device-width,initial-scale=1"></head><body><main id="root" style="padding:16px"></main><script type="module">
 import RefreshRuntime from '/@react-refresh';RefreshRuntime.injectIntoGlobalHook(window);window.$RefreshReg$=()=>{};window.$RefreshSig$=()=>type=>type;window.__vite_plugin_react_preamble_installed__=true;
 import React from '/node_modules/.vite/deps/react.js';import ReactDOM from '/node_modules/.vite/deps/react-dom_client.js';const {default:Test}=await import('/tests/fixtures/FiscalLaboratoryHarness.jsx');import '/src/styles.css';import '/src/cockpit.css';
 ReactDOM.createRoot(document.getElementById('root')).render(React.createElement(React.StrictMode,null,React.createElement(Test)));
 </script></body></html>`,
      }),
    )
    await page.goto(`${base}/__fiscal_lab_test__`)
    await expect(
      page.getByRole('heading', { name: 'Laboratoire', exact: true }),
    ).toBeVisible()
    await page.keyboard.press('Control+Shift+Space')
    const dialog = page.getByRole('dialog', { name: 'Calcul rapide' })
    await expect(dialog).toBeVisible()
    await dialog.getByLabel('Expression', { exact: true }).fill('0.1+0.2')
    await dialog.getByRole('button', { name: 'Calculer', exact: true }).click()
    await expect(dialog.getByLabel('Résultat du calcul')).toHaveText('0.3')
    assert.equal(writes, 0)
    await dialog.getByRole('button', { name: 'Ouvrir en feuille' }).click()
    await expect(dialog).not.toBeVisible()
    await expect(page.getByLabel('B2', { exact: true })).toHaveValue('0.3')
    const cell = page.getByLabel('C1', { exact: true })
    await cell.dblclick()
    await cell.fill('=B2+0.7')
    await cell.press('Enter')
    await expect(cell).toHaveValue('1')
    await cell.focus()
    await cell.press('Control+z')
    await expect(cell).toHaveValue('')
    await cell.press('Control+Shift+z')
    await expect(cell).toHaveValue('1')
    await cell.evaluate((el) => {
      const data = new DataTransfer()
      data.setData('text/plain', '=IF(B2>0;ROUND(1.005;2);1/0)')
      el.dispatchEvent(
        new ClipboardEvent('paste', { bubbles: true, clipboardData: data }),
      )
    })
    await expect(cell).toHaveValue('1.01')
    await page.getByLabel('Titre du document').fill('FEUILLE SYNTHÉTIQUE TEST')
    await page
      .getByRole('button', { name: 'Enregistrer le document', exact: true })
      .click()
    await expect(
      page.getByText('Document enregistré sur le serveur.', { exact: true }),
    ).toBeVisible()
    assert.equal(writes, 1)
    failSave = true
    await cell.dblclick()
    await cell.fill('=B2+2')
    await cell.press('Enter')
    await expect(cell).toHaveValue('2.3')
    await page
      .getByRole('button', { name: 'Enregistrer le document', exact: true })
      .click()
    await expect(page.getByRole('alert')).toContainText(
      'Révision modifiée TEST',
    )
    await expect(cell).toHaveValue('2.3')
    failSave = false
    await page
      .getByRole('button', { name: 'Enregistrer le document', exact: true })
      .click()
    await expect(
      page.getByText('Document enregistré sur le serveur.', { exact: true }),
    ).toBeVisible()
    await page.getByRole('button', { name: 'Changer de parcours' }).click()
    await page.getByLabel('Base de revenu').selectOption('final_taxable')
    await page
      .getByRole('textbox', {
        name: 'Revenu net imposable final (€)',
        exact: true,
      })
      .fill('30000')
    await page.getByRole('checkbox', { name: /Je confirme/ }).check()
    await page.getByRole('button', { name: 'Estimer mon impôt' }).click()
    await expect(
      page.getByRole('region', { name: 'Résultat fiscal' }),
    ).toContainText(/2.104,00/)
    await expect(page.getByText(/Pas de solde final estimable/)).toBeVisible()
    await page.getByLabel('PAS déjà prélevé').fill('2200')
    await expect(
      page.getByRole('region', { name: 'Résultat fiscal' }),
    ).not.toBeVisible()
    await page.getByRole('button', { name: 'Estimer mon impôt' }).click()
    await expect(page.getByText(/Remboursement potentiel/)).toContainText(
      /96,00/,
    )
    await page.getByLabel('Mois de PAS déjà couverts').fill('6')
    await page.getByRole('button', { name: 'Estimer mon impôt' }).click()
    await expect(page.getByText(/Pas de solde final estimable/)).toBeVisible()
    await page.getByLabel('Année de revenus du PAS').fill('2024')
    await page.getByRole('button', { name: 'Estimer mon impôt' }).click()
    await expect(page.getByRole('alert')).toContainText('même année')
    await page.getByLabel('Année de revenus du PAS').fill('2025')
    await page.getByRole('button', { name: 'Estimer mon impôt' }).click()
    for (const view of ['Fiscal', 'Lab']) {
      if (view === 'Lab') {
        await page.getByRole('button', { name: 'Changer de parcours' }).click()
        await expect(page.getByLabel('C1', { exact: true })).toHaveValue('2.3')
      }
      for (const width of [360, 768, 1280])
        for (const theme of ['light', 'dark']) {
          await page.setViewportSize({ width, height: 900 })
          await page.evaluate(async (t) => {
            document.documentElement.setAttribute('data-theme', t)
            await document.fonts.ready
            // Axe must inspect a painted, settled theme, not cached pixels from
            // the previous one. This also waits for finite CSS transitions.
            await new Promise(requestAnimationFrame)
            await Promise.all(
              document
                .getAnimations()
                .filter(
                  (animation) =>
                    animation.effect?.getComputedTiming().iterations !==
                    Infinity,
                )
                .map((animation) => animation.finished.catch(() => {})),
            )
            await new Promise(requestAnimationFrame)
          }, theme)
          assert.ok(
            await page.evaluate(
              () => document.documentElement.scrollWidth <= innerWidth + 1,
            ),
            `${view} overflow`,
          )
          const axe = await new AxeBuilder({ page })
            .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
            .analyze()
          assert.deepEqual(axe.violations, [], `${view} Axe ${width} ${theme}`)
          if (process.env.THEMIS_LAB_ARTIFACTS) {
            const fs = require('node:fs'),
              path = require('node:path')
            fs.mkdirSync(process.env.THEMIS_LAB_ARTIFACTS, { recursive: true })
            await page.screenshot({
              path: path.join(
                process.env.THEMIS_LAB_ARTIFACTS,
                `${view}-${width}-${theme}.png`,
              ),
              fullPage: true,
            })
          }
        }
    }
    await page.getByLabel('Titre du document').focus()
    await page.keyboard.press('Control+Shift+Space')
    await expect(dialog).not.toBeVisible()
    await page
      .locator('summary')
      .filter({ hasText: 'Ouvrir ou supprimer' })
      .click()
    await expect(
      page
        .getByRole('button', { name: /FEUILLE SYNTHÉTIQUE TEST · révision/ })
        .first(),
    ).toBeVisible()
    // Close and fetch the saved document again: local navigation alone is not persistence.
    await page
      .getByRole('button', {
        name: 'Fermer FEUILLE SYNTHÉTIQUE TEST',
        exact: true,
      })
      .click()
    await page
      .getByRole('button', { name: /FEUILLE SYNTHÉTIQUE TEST · révision/ })
      .first()
      .click()
    await expect(page.getByLabel('C1', { exact: true })).toHaveValue('2.3')
    await expect
      .poll(() =>
        page.evaluate(
          () => sessionStorage.getItem('themis-laboratory:987654321') || '',
        ),
      )
      .toContain('FEUILLE SYNTHÉTIQUE TEST')
    await page.reload()
    await expect(page.getByLabel('C1', { exact: true })).toHaveValue('2.3')
    const beforeReloadWrites = writes
    const recoveredCell = page.getByLabel('C1', { exact: true })
    await recoveredCell.dblclick()
    await recoveredCell.fill('=9+0.3')
    await recoveredCell.press('Enter')
    await page.reload()
    await expect(page.getByLabel('C1', { exact: true })).toHaveValue('9.3')
    assert.equal(
      writes,
      beforeReloadWrites,
      'Recovery must not write PostgreSQL',
    )
    assert.deepEqual(errors, [])
    console.log(
      `Fiscal/Laboratory ${real ? 'REAL API' : 'mock API'} PASS: decimals, Worker, conversion, refs, undo/redo, clipboard, save/conflict preservation, PAS, keyboard, six layouts/themes and Axe.`,
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
