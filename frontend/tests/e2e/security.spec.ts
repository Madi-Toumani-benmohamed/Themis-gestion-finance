import { expect, test } from '@playwright/test'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

test('les identifiants de relevé et budget d’un autre propriétaire sont inaccessibles', async ({ page }) => {
  const fixture = JSON.parse(await readFile(resolve('test-results/e2e-fixture.json'), 'utf8'))
  const statementAccess = await page.request.get(`/api/transactions?statement_id=${fixture.otherStatement.statementID}`)
  expect(statementAccess.ok()).toBeTruthy()
  expect(await statementAccess.json()).toEqual([])
  const deleteStatement = await page.request.delete(`/api/statements/${fixture.otherStatement.statementID}`)
  expect(deleteStatement.status()).toBe(404)
  const deleteBudget = await page.request.delete(`/api/budgets/${fixture.otherBudgetID}`)
  expect(deleteBudget.status()).toBe(404)

  const browser = page.context().browser()
  expect(browser).toBeTruthy()
  const otherContext = await browser!.newContext({ baseURL: process.env.THEMIS_E2E_BASE_URL ?? 'http://127.0.0.1:15176' })
  const login = await otherContext.request.post('/api/auth/login', { data: { username: 'e2e-owner-b', password: 'E2E-finance-2026-other' } })
  expect(login.ok()).toBeTruthy()
  const otherPage = await otherContext.newPage()
  const stillExists = await otherPage.request.get('/api/statements')
  expect((await stillExists.json()).some((statement: { id: number }) => statement.id === fixture.otherStatement.statementID)).toBeTruthy()
  const otherBudgets = await otherPage.request.get('/api/budgets')
  expect((await otherBudgets.json()).some((budget: { id: number }) => budget.id === fixture.otherBudgetID)).toBeTruthy()
  await otherContext.close()
})
