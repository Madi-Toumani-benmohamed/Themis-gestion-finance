import { readFile } from 'node:fs/promises'
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
    try {
      await expect(page.locator('main')).toHaveScreenshot('dashboard.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
    } catch (error) {
      const png = await readFile(test.info().outputPath('dashboard-actual.png'))
      console.log('PR42_VISUAL_REFERENCE_BEGIN')
      for (const chunk of png.toString('base64').match(/.{1,3000}/g) || []) console.log(chunk)
      console.log('PR42_VISUAL_REFERENCE_END')
      throw error
    }
  })

  test('transactions', async ({ page }) => {
    await page.getByRole('tab', { name: 'Transactions' }).click()
    await expect(page.locator('main')).toHaveScreenshot('transactions.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
  })

  test('budgets', async ({ page }) => {
    await page.getByRole('tab', { name: 'Budgets' }).click()
    await expect(page.locator('main')).toHaveScreenshot('budgets.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
  })

  test('simulation', async ({ page }) => {
    await page.getByRole('tab', { name: 'Simulation' }).click()
    await expect(page.locator('main')).toHaveScreenshot('simulation.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
  })
})
