// Fully intercepted APIs: synthetic fixtures only, never real bank data.
const { chromium } = require('playwright')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const base = process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176'
let failure = '',
  empty = false,
  delay = 0,
  payments = [],
  mutations = []
const credit = {
  id: 1,
  name: 'AUTO TEST',
  monthly_amount: 33.33,
  months_total: 12,
  start_month: '2020-01',
  end_month: '2020-12',
  next_scheduled_month: null,
  note: '',
  payment_count: 0,
  payment_history_complete: true,
  progress_pct: 0,
  status: 'active',
  amounts_cents: {
    monthly_amount: 3333,
    total_scheduled: 39996,
    opening_paid: 0,
    payments_recorded: 0,
    paid_total: 0,
    remaining: 39996,
  },
}
const unknown = {
  ...structuredClone(credit),
  id: 2,
  name: 'HISTORIQUE TEST',
  payment_history_complete: false,
  progress_pct: null,
  status: 'unknown',
  remaining: null,
  amounts_cents: {
    ...credit.amounts_cents,
    opening_paid: null,
    remaining: null,
    paid_total: null,
  },
}
const tax = {
  id: 1,
  name: 'IMPOT TEST',
  due_date: '2020-01-01',
  paid: false,
  status: 'to_verify',
  days_left: -100,
  overdue_confirmed: false,
  amounts_cents: { amount: 10001, paid_amount: 0, remaining: 10001 },
}
let archivedCredits = []
let credits = [credit, unknown],
  taxes = [tax]
function report() {
  return {
    as_of: '2026-10-25',
    source: 'user_declarations',
    credits: empty ? [] : credits,
    taxes: empty ? [] : taxes,
    unknown_credits_count: empty ? 0 : 1,
    invalid_credits_count: 0,
    total_debt: empty ? 0 : null,
    taxes_due_soon: 0,
    taxes_to_verify: empty ? 0 : 1,
    amounts_cents: {
      total_debt: empty ? 0 : null,
      known_total_debt: empty
        ? 0
        : credit.amounts_cents.remaining + tax.amounts_cents.remaining,
      credits_remaining: empty ? 0 : null,
      monthly_load: 0,
      taxes_due: empty ? 0 : tax.amounts_cents.remaining,
    },
  }
}
async function main() {
  const browser = await chromium.launch({
    headless: true,
    channel: 'chromium',
  })
  try {
    const context = await browser.newContext({
      viewport: { width: 1920, height: 1080 },
    })
    const page = await context.newPage()
    const errors = []
    page.on('pageerror', (e) => errors.push(e.message))
    await page.addInitScript(() => {
      localStorage.setItem('themis-theme', 'dark')
      // StrictMode may abort the first fetch before Chromium dispatches it.
      // Observe invocations, preserving the original arguments and AbortSignal.
      window.__creditsFetchPaths = []
      const originalFetch = window.fetch.bind(window)
      window.fetch = (input, options) => {
        const url = new URL(
          input instanceof Request ? input.url : input,
          location.href,
        )
        window.__creditsFetchPaths.push(url.pathname)
        return originalFetch(input, options)
      }
    })
    await page.route('**/api/**', async (route) => {
      const req = route.request(),
        url = new URL(req.url()),
        path = url.pathname,
        method = req.method()
      let data = {},
        status = 200
      if (delay && path === '/api/obligations')
        await new Promise((r) => setTimeout(r, delay))
      if (
        failure &&
        (failure === path || (failure === 'mutation' && method !== 'GET'))
      ) {
        data = { error: 'ERREUR CREDITS TEST' }
        status = 500
      } else if (path === '/api/auth/status') data = { setup_required: false }
      else if (path === '/api/auth/me')
        data = { id: 1, username: 'CREDITS TEST', role: 'user' }
      else if (path === '/api/health') data = { ai_enabled: false }
      else if (path === '/api/summary')
        data = { monthly: [], by_category: [], uncategorized: 0 }
      else if (path === '/api/obligations') data = report()
      else if (path === '/api/credits' && method === 'GET')
        data = [...credits, ...archivedCredits]
      else if (path === '/api/taxes' && method === 'GET') data = taxes
      else if (path.endsWith('/history'))
        data = {
          changes: [
            {
              id: 1,
              event_type: 'updated',
              changed_at: '2026-10-25T12:00:00Z',
              before_state: unknown,
              after_state: credit,
            },
          ],
          payments,
          has_more: false,
          payments_has_more: false,
        }
      else if (
        method !== 'GET' &&
        (path.startsWith('/api/credits') || path.startsWith('/api/taxes'))
      ) {
        const body = req.postDataJSON()
        mutations.push({ path, method, body })
        if (path.endsWith('/payments') && method === 'POST') {
          const cents = 3333
          credit.amounts_cents.payments_recorded += cents
          credit.amounts_cents.paid_total += cents
          credit.amounts_cents.remaining -= cents
          credit.payment_count++
          credit.progress_pct = 8.33
          payments.unshift({
            id: 1,
            amount_cents: cents,
            payment_date: body.payment_date,
            note: body.note,
            voided_at: null,
          })
          data = { payment: payments[0] }
        } else if (path.includes('/payments/') && method === 'DELETE') {
          payments[0].voided_at = '2026-10-25T12:00:00Z'
          payments[0].void_note = body.note
          credit.amounts_cents.payments_recorded = 0
          credit.amounts_cents.paid_total = 0
          credit.amounts_cents.remaining = 39996
          credit.progress_pct = 0
          credit.payment_count = 0
        } else if (path === '/api/credits' && method === 'POST') {
          credits.push({
            ...structuredClone(credit),
            id: 3,
            name: body.name,
            note: body.note,
          })
        } else if (path === '/api/credits/1' && method === 'PATCH') {
          if (body.closed !== undefined) {
            credit.status = body.closed ? 'closed' : 'paid'
            credit.closed_at = body.closed ? '2026-10-25T12:00:00Z' : null
          } else {
            credit.name = body.name
            credit.note = body.note
          }
        } else if (path === '/api/credits/3' && method === 'DELETE') {
          archivedCredits.push({
            ...credits.find((c) => c.id === 3),
            archived_at: '2026-10-25T12:00:00Z',
          })
          credits = credits.filter((c) => c.id !== 3)
        } else if (path === '/api/taxes/1' && method === 'PATCH') {
          if (body.payment_amount) {
            tax.amounts_cents.paid_amount = 3333
            tax.amounts_cents.remaining = 6668
            tax.status = 'partial'
          } else if (body.paid_amount) {
            tax.amounts_cents.paid_amount = 0
            tax.amounts_cents.remaining = 10001
            tax.status = 'to_verify'
          } else if (body.name) tax.name = body.name
          else if (body.overdue_confirmed !== undefined) {
            tax.overdue_confirmed = body.overdue_confirmed
            tax.status = body.overdue_confirmed ? 'late_confirmed' : 'to_verify'
          }
        }
      }
      await route.fulfill({
        status,
        contentType: 'application/json',
        body: JSON.stringify(data),
      })
    })
    const root = page.locator('.credits-page').first()
    async function enter() {
      await page
        .getByRole('tab', { name: 'Crédits & taxes', exact: true })
        .click()
      await root.getByRole('heading', { name: 'Mes engagements' }).waitFor()
    }
    async function settleMotion() {
      await page.evaluate(() =>
        Promise.all(
          document
            .getAnimations()
            .filter((a) =>
              Number.isFinite(a.effect.getComputedTiming().endTime),
            )
            .map((a) => a.finished.catch(() => {})),
        ),
      )
    }
    async function ready() {
      await root.locator('.obligations-banner').waitFor()
      await settleMotion()
    }
    async function axe() {
      // Dialogs and history rows animate independently of page readiness.
      // Measure their final colors, not a transient ancestor opacity.
      await settleMotion()
      const result = await new AxeBuilder({ page })
        .include('main')
        .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
        .analyze()
      assert.deepEqual(
        result.violations.map((v) => ({
          id: v.id,
          nodes: v.nodes.map((n) => ({
            target: n.target,
            summary: n.failureSummary,
          })),
        })),
        [],
      )
    }
    async function reload() {
      await page
        .getByRole('tab', { name: 'Crédits & taxes', exact: true })
        .click()
      await page.getByRole('tab', { name: 'Transactions', exact: true }).click()
      await enter()
    }
    const dialog = () => page.getByRole('dialog')
    await page.goto(base)
    await enter()
    await ready()
    // Count effect invocations even when the cleanup abort prevents a network request.
    const fetchPaths = await page.evaluate(() => window.__creditsFetchPaths)
    assert.equal(
      fetchPaths.filter((path) => path === '/api/obligations').length,
      2,
    )
    assert.equal(
      fetchPaths.filter(
        (path) => path === '/api/credits' || path === '/api/taxes',
      ).length,
      0,
    )
    assert.match(await root.innerText(), /Solde global reste non documenté/i)
    assert.match(await root.innerText(), /Paiement à vérifier/)
    assert.ok(!(await root.innerText()).includes('En retard'))
    assert.equal(await root.getByRole('progressbar').count(), 1)
    await axe()
    // Exact payment and immediate refresh, double submission guard, history and void.
    await page
      .getByRole('article', { name: 'AUTO TEST', exact: true })
      .getByRole('button', { name: 'Déclarer un paiement' })
      .click()
    await dialog()
      .getByLabel('Montant du paiement (€)', { exact: true })
      .fill('33,33')
    failure = 'mutation'
    await dialog()
      .getByRole('button', { name: 'Enregistrer', exact: true })
      .click()
    await dialog().getByRole('alert').waitFor()
    assert.equal(
      await dialog()
        .getByLabel('Montant du paiement (€)', { exact: true })
        .inputValue(),
      '33,33',
    )
    failure = ''
    await dialog()
      .getByRole('button', { name: 'Enregistrer', exact: true })
      .dblclick()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    assert.equal(
      mutations.filter((m) => m.path.endsWith('/payments')).length,
      1,
    )
    assert.equal(mutations.at(-1).body.amount, '33.33')
    assert.ok(mutations.at(-1).body.idempotency_key)
    assert.match(
      await page
        .getByRole('article', { name: 'AUTO TEST', exact: true })
        .innerText(),
      /366,63/,
    )
    await page
      .getByRole('article', { name: 'AUTO TEST', exact: true })
      .getByRole('button', { name: 'Historique de AUTO TEST' })
      .click()
    await dialog().getByText('33,33', { exact: false }).first().waitFor()
    await dialog().getByRole('button', { name: 'Annuler ce paiement' }).click()
    await dialog().getByLabel('Motif d’annulation').fill('DOUBLON SAISI TEST')
    await dialog()
      .getByRole('button', { name: 'Confirmer l’annulation' })
      .click()
    await dialog()
      .getByText('Annulé : DOUBLON SAISI TEST', { exact: false })
      .waitFor()
    await dialog().getByRole('button', { name: 'Fermer', exact: true }).click()
    await ready()
    assert.equal(credit.amounts_cents.remaining, 39996)
    // Validation never rounds overprecise inputs. Create/edit/delete confirmation.
    await root.getByRole('button', { name: '＋ Crédit', exact: true }).click()
    await dialog().getByLabel('Nom', { exact: true }).fill('NOUVEAU TEST')
    await dialog().getByLabel('Mensualité (€)', { exact: true }).fill('33.333')
    await dialog().getByLabel('Nombre de mois').fill('12')
    await dialog()
      .getByRole('button', { name: 'Enregistrer', exact: true })
      .click()
    await dialog().getByRole('alert').waitFor()
    assert.equal(
      mutations.filter((m) => m.path === '/api/credits' && m.method === 'POST')
        .length,
      0,
    )
    await dialog().getByLabel('Mensualité (€)', { exact: true }).fill('33.33')
    await dialog()
      .getByRole('button', { name: 'Enregistrer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    assert.equal(mutations.at(-1).body.opening_paid, null)
    const created = page.getByRole('article', {
      name: 'NOUVEAU TEST',
      exact: true,
    })
    await created.getByRole('button', { name: 'Retirer NOUVEAU TEST' }).click()
    await dialog().getByRole('button', { name: 'Annuler', exact: true }).click()
    assert.equal(
      mutations.filter(
        (m) => m.method === 'DELETE' && m.path === '/api/credits/3',
      ).length,
      0,
    )
    await created.getByRole('button', { name: 'Retirer NOUVEAU TEST' }).click()
    await dialog()
      .getByRole('button', { name: 'Confirmer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    assert.equal(
      await page
        .getByRole('article', { name: 'NOUVEAU TEST', exact: true })
        .count(),
      0,
    )
    await root
      .getByRole('button', { name: 'Modifier AUTO TEST', exact: true })
      .click()
    await dialog().getByLabel('Nom', { exact: true }).fill('AUTO MODIFIE TEST')
    await dialog()
      .getByRole('button', { name: 'Enregistrer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    await page.getByRole('article', { name: 'AUTO MODIFIE TEST' }).waitFor()
    // Partial tax payments retain the amount and do not claim an unverified delay.
    await root
      .getByRole('button', {
        name: 'Déclarer un paiement pour IMPOT TEST',
        exact: true,
      })
      .click()
    await dialog()
      .getByLabel('Montant du paiement (€)', { exact: true })
      .fill('33.33')
    await dialog()
      .getByRole('button', { name: 'Enregistrer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    assert.match(
      await root.locator('.taxes-table').innerText(),
      /66,68\s€ restant/,
    )
    assert.match(
      await root.locator('.taxes-table').innerText(),
      /Partiellement payé/,
    )
    await root
      .getByRole('button', {
        name: 'Corriger le payé pour IMPOT TEST',
        exact: true,
      })
      .click()
    await dialog()
      .getByLabel('Total payé corrigé (€)', { exact: true })
      .fill('0')
    await dialog().getByLabel('Motif de correction').fill('ERREUR TEST')
    await dialog()
      .getByRole('button', { name: 'Enregistrer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    // Keyboard, responsive, 200% layout equivalent, complete content without overflow.
    await root.getByLabel('Trier les crédits').selectOption('name')
    await root.getByLabel('Afficher').selectOption('paid')
    assert.match(await root.innerText(), /Aucun crédit pour ce filtre/)
    await root.getByLabel('Afficher').selectOption('all')
    for (const width of [1920, 960, 768, 390, 320]) {
      await page.setViewportSize({ width, height: 1080 })
      await ready()
      assert.ok(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
        `overflow ${width}`,
      )
      const clipped = await root.locator('.taxes-table').evaluate(table => {
        const area = table.getBoundingClientRect()
        return [...table.querySelectorAll('tbody th, tbody button')].some(el => {
          const rect = el.getBoundingClientRect()
          return rect.top < area.top - 1 || rect.bottom > area.bottom + 1 || rect.left < area.left - 1 || rect.right > area.right + 1
        })
      })
      assert.equal(clipped, false, `clipped tax name/action at ${width}px`)
      await axe()
    }
    await root.getByRole('button', { name: '＋ Crédit', exact: true }).click()
    await dialog().getByLabel('Nom', { exact: true }).focus()
    await page.keyboard.press('Tab')
    assert.equal(
      await page.evaluate(() => document.activeElement.name),
      'monthly',
    )
    await axe()
    await page.keyboard.press('Escape')
    await dialog().waitFor({ state: 'hidden' })
    assert.equal(
      await root
        .getByRole('button', { name: '＋ Crédit', exact: true })
        .evaluate((e) => e === document.activeElement),
      true,
    )
    await root
      .getByRole('button', { name: 'Historique de IMPOT TEST', exact: true })
      .click()
    await dialog()
      .getByRole('heading', { name: 'Modifications de l’engagement' })
      .waitFor()
    await axe()
    await page.keyboard.press('Escape')
    const artifacts =
      process.env.THEMIS_CREDITS_ARTIFACTS || '/tmp/themis-credits-ui'
    fs.mkdirSync(artifacts, { recursive: true })
    await page.evaluate(() => scrollTo(0, 0))
    await page.screenshot({
      path: `${artifacts}/credits-mobile.png`,
      fullPage: true,
    })
    await page.setViewportSize({ width: 1920, height: 1080 })
    await page.evaluate(() => scrollTo(0, 0))
    await page.screenshot({
      path: `${artifacts}/credits-desktop.png`,
      fullPage: true,
    })

    // Closure follows a declared zero balance, and never records a new payment.
    const paymentCount = mutations.filter((m) =>
      m.path.endsWith('/payments'),
    ).length
    credit.status = 'paid'
    credit.progress_pct = 100
    credit.amounts_cents.opening_paid = 39996
    credit.amounts_cents.paid_total = 39996
    credit.amounts_cents.remaining = 0
    await reload()
    await ready()
    await page
      .getByRole('article', { name: 'AUTO MODIFIE TEST', exact: true })
      .getByRole('button', { name: 'Clôturer', exact: true })
      .click()
    await dialog()
      .getByRole('button', { name: 'Confirmer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    await page
      .getByRole('article', { name: 'AUTO MODIFIE TEST', exact: true })
      .getByRole('button', { name: 'Rouvrir', exact: true })
      .click()
    await dialog()
      .getByRole('button', { name: 'Confirmer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    assert.equal(
      mutations.filter((m) => m.path.endsWith('/payments')).length,
      paymentCount,
    )
    // Retard is an explicit, reversible declaration.
    await root
      .getByRole('button', { name: 'Confirmer un retard', exact: true })
      .click()
    await dialog()
      .getByRole('button', { name: 'Confirmer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    assert.match(
      await root.locator('.taxes-table').innerText(),
      /Retard confirmé/,
    )
    await root
      .getByRole('button', {
        name: 'Retirer la confirmation de retard',
        exact: true,
      })
      .click()
    await dialog()
      .getByRole('button', { name: 'Confirmer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    assert.match(
      await root.locator('.taxes-table').innerText(),
      /Paiement à vérifier/,
    )
    await root
      .getByRole('button', { name: 'Modifier IMPOT TEST', exact: true })
      .click()
    await dialog().getByLabel('Nom', { exact: true }).fill('IMPOT MODIFIE TEST')
    await dialog()
      .getByRole('button', { name: 'Enregistrer', exact: true })
      .click()
    await dialog().waitFor({ state: 'hidden' })
    await ready()
    assert.match(
      await root.locator('.taxes-table').innerText(),
      /IMPOT MODIFIE TEST/,
    )
    failure = '/api/credits/1/history'
    await page
      .getByRole('article', { name: 'AUTO MODIFIE TEST', exact: true })
      .getByRole('button', { name: 'Historique de AUTO MODIFIE TEST' })
      .click()
    await dialog().getByRole('alert').waitFor()
    failure = ''
    await dialog()
      .getByRole('button', { name: 'Réessayer', exact: true })
      .click()
    await dialog()
      .getByRole('heading', { name: 'Modifications de l’engagement' })
      .waitFor()
    await page.keyboard.press('Escape')

    await root.locator('.engagement-archives summary').click()
    await root
      .getByRole('button', {
        name: 'Historique archivé de NOUVEAU TEST',
        exact: true,
      })
      .waitFor()
    await root
      .getByRole('button', {
        name: 'Historique archivé de NOUVEAU TEST',
        exact: true,
      })
      .click()
    await dialog()
      .getByRole('heading', { name: 'Modifications de l’engagement' })
      .waitFor()
    assert.equal(
      await dialog()
        .getByRole('button', { name: 'Annuler ce paiement' })
        .count(),
      0,
    )
    await page.keyboard.press('Escape')
    // Loading, read error, retry and empty state stay distinct.
    failure = '/api/obligations'
    await reload()
    await root.getByRole('alert').waitFor()
    assert.equal(await root.locator('.obligations-banner').count(), 0)
    failure = ''
    delay = 350
    await root.getByRole('button', { name: 'Réessayer', exact: true }).click()
    await root
      .getByRole('status')
      .filter({ hasText: 'Chargement des engagements' })
      .waitFor()
    await ready()
    delay = 0
    empty = true
    await reload()
    await ready()
    assert.match(await root.innerText(), /Aucun crédit enregistré/)
    assert.match(await root.innerText(), /Aucune échéance enregistrée/)
    assert.match(await root.locator('.obligations-banner').innerText(), /0,00/)
    assert.deepEqual(errors, [])
    console.log(
      'Crédits navigateur : parcours, erreurs, paiements, historique, clavier, cinq largeurs et Axe OK',
    )
  } finally {
    await browser.close()
  }
}
main().catch((e) => {
  console.error(e)
  process.exit(1)
})
