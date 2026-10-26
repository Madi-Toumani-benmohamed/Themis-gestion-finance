import { expect, test } from '@playwright/test'

test('le dashboard affiche la période et distingue le mois courant provisoire', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'La vue d’ensemble.' })).toBeVisible()
  const period = await page.request.get('/api/period')
  expect(period.ok()).toBeTruthy()
  const periodData = await period.json()
  expect(periodData.month).toMatch(/^\d{4}-\d{2}$/)
  await expect(page.getByLabel('Sélectionner un mois')).toBeVisible()
  const monthly = await page.request.get('/api/summary?months=12')
  expect(monthly.ok()).toBeTruthy()
  const summary = await monthly.json()
  const current = summary.monthly.find((entry: { is_current: boolean }) => entry.is_current)
  if (current) expect(current.status).toBe('current')
})
