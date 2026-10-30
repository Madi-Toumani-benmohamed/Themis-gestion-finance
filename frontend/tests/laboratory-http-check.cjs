// Exercise the production app from an actual insecure origin, as on a VM/LAN IP.
// All assets are served by the local preview; no request reaches themis.invalid.
const { expect } = require('@playwright/test')
const assert = require('node:assert/strict')
const rule = {
  income_year: 2025,
  assessment_year: 2026,
  status: 'official',
  version: 'FR-IR-2025-2026.1',
  verified_on: '2026-10-30',
  sources: [
    'https://www.impots.gouv.fr/www2/fichiers/documentation/brochure/ir_2026/pdf_som/21-calcul_impot_369a382.pdf',
  ],
}
module.exports = async function verifyHttpAccess(browser, preview) {
  for (const width of [1280, 360]) {
    const context = await browser.newContext({
      viewport: { width, height: 900 },
      reducedMotion: 'no-preference',
    })
    try {
      const page = await context.newPage(),
        errors = [],
        documents = []
      page.on('pageerror', (e) => errors.push(e.message))
      page.on('dialog', (dialog) => dialog.accept())
      await context.route('http://themis.invalid/**', async (route) => {
        const req = route.request(),
          url = new URL(req.url()),
          path = url.pathname
        if (!path.startsWith('/api/')) {
          const response = await context.request.get(
            preview + path + url.search,
          )
          return route.fulfill({ response })
        }
        let body = {}
        if (path === '/api/auth/status') body = { setup_required: false }
        else if (path === '/api/auth/me')
          body = { id: 1, username: 'HTTP SYNTHETIC TEST', role: 'admin' }
        else if (path === '/api/health') body = { ai_enabled: false }
        else if (path === '/api/summary')
          body = { monthly: [], by_category: [], uncategorized: 0 }
        else if (['/api/accounts', '/api/tags'].includes(path)) body = []
        else if (path === '/api/laboratory' && req.method() === 'GET')
          body = { documents, has_more: false }
        else if (path === '/api/laboratory' && req.method() === 'POST') {
          const doc = req.postDataJSON()
          assert.match(
            doc.client_key,
            /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
          )
          body = { ...doc, id: documents.length + 1, revision: 1 }
          documents.push(body)
        } else if (path === '/api/fiscal/rules') body = [rule]
        else if (path === '/api/fiscal/simulate') {
          const input = req.postDataJSON()
          assert.equal(input.income_cents, 3000000)
          assert.equal(input.basis, 'final_taxable')
          body = {
            rules: rule,
            half_parts: 2,
            annual_income_cents: 3000000,
            salary_deduction_cents: 0,
            taxable_cents: 3000000,
            gross_tax_cents: 210400,
            discount_cents: 0,
            tax_cents: 210400,
            average_rate_pct: '7.01',
            marginal_rate_pct: 30,
            observed_paid_cents: null,
            projected_future_paid_cents: null,
            gap_before_future_cents: null,
            settlement_cents: null,
            settlement_status: 'unknown',
            steps: [{ label: 'Impôt estimé', amount_cents: 210400 }],
            assumptions: [],
            limitations: ['Saisie synthétique TEST.'],
          }
        }
        await route.fulfill({ json: body })
      })
      await page.goto('http://themis.invalid/')
      assert.deepEqual(
        await page.evaluate(() => ({
          secure: isSecureContext,
          uuid: typeof crypto.randomUUID,
          random: typeof crypto.getRandomValues,
        })),
        { secure: false, uuid: 'undefined', random: 'function' },
      )
      await page.getByRole('tab', { name: 'Laboratoire', exact: true }).click()
      await page
        .getByRole('button', { name: 'Nouvelle feuille', exact: true })
        .click()
      const cell = page.getByLabel('A1', { exact: true })
      await cell.dblclick()
      await cell.fill('=0.1+0.2')
      await cell.press('Enter')
      await expect(cell).toHaveValue('0.3')
      await page
        .getByRole('button', { name: 'Enregistrer le document', exact: true })
        .click()
      await expect(
        page.getByText('Document enregistré sur le serveur.', { exact: true }),
      ).toBeVisible()
      await page
        .getByRole('button', { name: 'Enregistrer une copie', exact: true })
        .click()
      await expect(
        page.getByText('Copie enregistrée, original local conservé.', {
          exact: true,
        }),
      ).toBeVisible()
      assert.equal(documents.length, 2)
      assert.notEqual(documents[0].client_key, documents[1].client_key)
      await page
        .getByRole('button', { name: 'Calcul rapide', exact: true })
        .first()
        .click()
      const dialog = page.getByRole('dialog', { name: 'Calcul rapide' })
      await dialog.getByLabel('Expression', { exact: true }).fill('1+2')
      await dialog
        .getByRole('button', { name: 'Calculer', exact: true })
        .click()
      await expect(dialog.getByLabel('Résultat du calcul')).toHaveText('3')
      await dialog
        .getByRole('button', {
          name: 'Enregistrer une copie du brouillon',
          exact: true,
        })
        .click()
      await expect(
        dialog.getByText('Brouillon enregistré sur le serveur.', {
          exact: true,
        }),
      ).toBeVisible()
      assert.equal(documents[2].type, 'scratchpad')
      await dialog
        .getByRole('button', { name: 'Ouvrir en feuille', exact: true })
        .click()
      await expect(dialog).not.toBeVisible()
      await expect(page.getByLabel('B2', { exact: true })).toHaveValue('3')
      // A dedicated navigation entry reaches Fiscality without discovering a header select.
      await page.getByRole('tab', { name: 'Impôts', exact: true }).click()
      await expect(
        page.getByRole('heading', {
          name: 'Fiscalité · impôt sur le revenu',
          exact: true,
        }),
      ).toBeVisible()
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
      assert.ok(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      )
      assert.deepEqual(
        errors,
        [],
        'The HTTP installation must never blank the React root',
      )
    } finally {
      await context.close()
    }
  }
  console.log(
    'Production HTTP/LAN PASS: no randomUUID, worksheet creation/save/copy, calculator/save/conversion, direct Impôts navigation, desktop/mobile and normal motion.',
  )
}
