import { expect, test } from '@playwright/test'

// Synthetic E2E database only. An optional reference server produces paired
// captures for review without replacing the existing visual goldens.
const pages = [
  ['dashboard', 'Tableau de bord'], ['analyse', 'Analyse'],
  ['transactions', 'Transactions'], ['budgets', 'Budgets'],
  ['simulate', 'Simulation budgétaire'], ['statements', 'Relevés'],
  ['accounts', 'Comptes'], ['assistant', 'Assistant'], ['admin', 'Admin'],
]
for (const theme of ['light', 'dark']) {
  for (const width of [1440, 390]) {
    test(`design system ${theme} ${width}`, async ({ page, baseURL }, testInfo) => {
      test.setTimeout(180_000)
      await page.setViewportSize({ width, height: width === 390 ? 844 : 1000 })
      await page.emulateMedia({ reducedMotion: 'reduce' })
      await page.addInitScript(theme => localStorage.setItem('themis-theme', theme), theme)
      // Expose the assistant UI without configuring or contacting an AI service.
      await page.route('**/api/health', route => route.fulfill({ json: { ai_enabled: true } }))
      const errors: string[] = []
      page.on('pageerror', error => errors.push(error.message))
      page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
      const targets = process.env.THEMIS_DESIGN_BASELINE_URL
        ? [['before', process.env.THEMIS_DESIGN_BASELINE_URL], ['after', baseURL!]]
        : [['after', baseURL!]]
      for (const [version, url] of targets) {
        await page.goto(url)
        await expect(page.locator('.dashboard-heading')).toBeVisible()
        await page.evaluate(() => document.fonts.ready)
        const tokens = () => page.evaluate(() => {
          const style = getComputedStyle(document.documentElement)
          return Object.fromEntries(['--page', '--paper', '--ink', '--accent', '--r-lg', '--sh-md', '--z-toast'].map(name => [name, style.getPropertyValue(name)]))
        })
        const initial = await tokens()
        for (const [id, name] of pages) {
          const nav = id === 'admin' ? page.getByRole('button', { name, exact: true }) : page.getByRole('tab', { name: new RegExp(`^${name}(?: \\d+)?$`) })
          await nav.click()
          await expect(page.locator('main')).toHaveAttribute('data-page', id)
          await page.waitForLoadState('networkidle')
          await expect(page.locator('main .spinner')).toHaveCount(0)
          expect(await tokens(), `Lazy CSS for ${id} must not redefine global tokens`).toEqual(initial)
          await page.keyboard.press('Tab')
          await nav.focus()
          await expect(nav).toHaveCSS('outline-style', 'solid')
          await expect(nav).toHaveCSS('outline-width', '2px')
          await page.mouse.move(0, 0)
          await page.screenshot({ path: testInfo.outputPath(`${version}-${id}.png`), fullPage: true, animations: 'disabled' })
        }
        await page.getByRole('button', { name: "Ouvrir l'assistant" }).click()
        await expect(page.locator('.assistant-panel')).toBeVisible()
        if (version === 'after') {
          // Panel displays with no ambient animation under reduced-motion; send awaits input.
          await expect(page.locator('.assistant-panel')).toHaveCSS('animation-name', 'none')
          await expect(page.locator('.assistant-panel-send')).toBeDisabled()
          await page.mouse.move(0, 0)
        }
        await page.screenshot({ path: testInfo.outputPath(`${version}-assistant-panel.png`), animations: 'disabled' })
      }
      expect(errors).toEqual([])
    })
  }
}
