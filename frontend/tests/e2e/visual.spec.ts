import { expect, test } from '@playwright/test'

test.describe('références visuelles déterministes', () => {
  // Static references are separate from motion.browser.cjs, which checks real animations.
  test.use({ reducedMotion: 'reduce' })
  test.beforeEach(async ({ page }) => {
    // Background recomputation must finish before freezing the browser clock.
    await expect.poll(async () => {
      const response = await page.request.get('/api/intel/jobs/status')
      if (!response.ok()) return false
      const job = await response.json()
      return job.status === 'idle' || job.status === 'done'
    }, { timeout: 30_000, message: 'Intelligence fixtures must settle before visual capture' }).toBe(true)
    await page.clock.install({ time: new Date('2026-10-26T12:00:00+02:00') })
    await page.goto('/')
    await page.evaluate(() => document.fonts.ready)
  })

  test('dashboard', async ({ page }) => {
    // A historical period stays historical when the server's calendar advances.
    await page.getByLabel('Sélectionner un mois').selectOption('2026-09')
    await expect(page.locator('.financial-summary h2')).toHaveText('septembre 2026')
    await expect(page.locator('.kpi-row')).toHaveAttribute('aria-busy', 'false')
    await expect(page.getByRole('heading', { name: 'Ce que Thémis a remarqué', exact: true })).toBeVisible()
    await expect(page.locator('.intel-highlights').getByRole('status').filter({ hasText: 'Chargement des découvertes' })).toHaveCount(0)
    await expect(page.locator('main')).toHaveScreenshot('dashboard.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
  })

  test('transactions', async ({ page }) => {
    await page.getByRole('tab', { name: 'Transactions' }).click()
    await expect(page.locator('main')).toHaveScreenshot('transactions.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
  })

  test('budgets', async ({ page }) => {
    await page.getByRole('tab', { name: 'Budgets' }).click()
    // Browser clock mocking does not freeze Go's clock. Use a closed month so
    // the real API's report/recurrence boundaries cannot drift each day.
    const reportResponse = page.waitForResponse(r => r.url().includes('/api/budgets/report?month=2026-09'))
    const recurringResponse = page.waitForResponse(r => r.url().includes('/api/recurring/review?month=2026-09'))
    await page.getByLabel('Mois consulté', { exact: true }).fill('2026-09')
    const [report, recurring] = await Promise.all([reportResponse, recurringResponse])
    expect(report.ok()).toBe(true)
    expect(recurring.ok()).toBe(true)
    expect((await report.json()).query_to).toBe('2026-09-30')
    expect((await recurring.json()).observation_end).toBe('2026-09-30')
    await expect(page.locator('.budget-period')).toContainText('Mois historique')
    await expect(page.locator('.budget-period')).toContainText('30 sept. 2026')
    await expect(page.locator('.budget-observation-window')).toContainText('30 sept. 2026')
    await expect(page.locator('main')).toHaveScreenshot('budgets.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
  })

  test('simulation', async ({ page }) => {
    await page.getByRole('tab', { name: 'Simulation' }).click()
    await expect(page.locator('main')).toHaveScreenshot('simulation.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
  })
})
