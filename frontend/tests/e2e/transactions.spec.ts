import { expect, test } from '@playwright/test'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

test('les transactions réelles sont visibles et filtrables par période', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('tab', { name: 'Transactions' }).click()
  await expect(page.getByRole('heading', { name: 'Transactions' })).toBeVisible()
  await expect(page.getByRole('table', { name: 'Transactions regroupées par date, selon les filtres actifs' })).toBeVisible()
  const month = page.getByLabel('Mois')
  await month.selectOption({ label: 'Tous les mois' })
  await expect(page.getByText('ACHAT TEST 001')).toBeVisible()
  const allRows = await page.locator('.transactions-table tbody tr:not(.tx-day)').count()
  const fixture = JSON.parse(await readFile(resolve('test-results/e2e-fixture.json'), 'utf8'))
  const response = await page.request.get(`/api/transactions?view=report&statement_id=${fixture.adminStatement.statementID}`)
  expect(response.status()).toBe(200)
  const report = await response.json()
  expect(report.total).toBe(81)
  expect(allRows).toBeGreaterThan(0)
  await month.selectOption('2026-08')
  await expect(page.getByText('ACHAT TEST 001')).toBeVisible()
  await month.selectOption('2026-09')
  await expect(page.getByText('ACHAT TEST 001')).toHaveCount(0)
  await page.getByLabel('Catégorie', { exact: true }).selectOption({ label: 'Logement' })
  await expect(page.getByText('Aucune transaction pour ces filtres.')).toBeVisible()
})
