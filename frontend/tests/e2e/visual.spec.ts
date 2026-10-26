import { expect, test } from '@playwright/test'

test.describe('références visuelles déterministes', () => {
  test.beforeEach(async ({ page }) => {
    await page.clock.install({ time: new Date('2026-10-26T12:00:00+02:00') })
    await page.goto('/')
    await page.evaluate(() => document.fonts.ready)
  })

  test('dashboard', async ({ page }) => {
    await expect(page.locator('main')).toHaveScreenshot('dashboard.png', { animations: 'disabled', caret: 'hide', maxDiffPixelRatio: 0.001 })
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
