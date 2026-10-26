import { expect, test } from '@playwright/test'
import { resolve } from 'node:path'

test('le relevé importé conserve ses invariants et un réimport ne double rien', async ({ page }) => {
  const statementsBefore = await page.request.get('/api/statements')
  expect(statementsBefore.ok()).toBeTruthy()
  const existingStatements = await statementsBefore.json()
  const existing = existingStatements.find((statement: { filename: string }) => statement.filename === 'releve-test-synthetique.pdf')
  expect(existing).toBeTruthy()
  expect(existing.debits_cents).toBe(429385)
  expect(existing.credits_cents).toBe(296585)
  expect(existing.sum_cents).toBe(-132800)
  expect(existing.opening_balance_cents).toBe(249288)
  expect(existing.closing_balance_cents).toBe(116488)

  await page.goto('/')
  await page.getByRole('button', { name: 'Importer un ou plusieurs relevés' }).click()
  await page.locator('input[type=file]').setInputFiles(resolve('tests/fixtures/statement-009.pdf'))
  const duplicatePreviewButton = page.getByRole('button', { name: 'Terminer — aucune nouvelle opération' })
  await expect(duplicatePreviewButton).toBeVisible()
  await expect(page.getByText(/81/).first()).toBeVisible()
  await duplicatePreviewButton.click()
  await expect(page.getByText(/déjà importé|aucune nouvelle opération/i)).toBeVisible()

  const rows = await page.request.get(`/api/transactions?statement_id=${existing.id}`)
  const transactions = await rows.json()
  expect(rows.ok()).toBeTruthy()
  expect(transactions).toHaveLength(81)
  const cents = transactions.reduce((sum: number, tx: { amount: number }) => sum + Math.round(tx.amount * 100), 0)
  expect(cents).toBe(-132800)

  await page.locator('input[type=file]').setInputFiles({ name: 'document-invalide.pdf', mimeType: 'application/pdf', buffer: Buffer.from('not a PDF') })
  await expect(page.getByText(/parsing|impossible|erreur/i).first()).toBeVisible({ timeout: 30_000 })
})
