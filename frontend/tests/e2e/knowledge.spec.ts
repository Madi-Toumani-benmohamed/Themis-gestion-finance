import { expect, test } from '@playwright/test'
import { resolve } from 'node:path'

test('la mémoire désactivée distingue les compteurs inconnus et conserve un journal isolé', async ({
  page,
  browser,
}) => {
  await page.goto('/')
  await page.getByRole('tab', { name: 'Mémoire', exact: true }).click()
  const panel = page.getByRole('region', {
    name: 'Activité du Knowledge Graph',
    exact: true,
  })
  await expect(panel.getByText('Désactivée', { exact: true })).toBeVisible()
  await expect(
    panel.locator('.kg-metrics').getByText('Non mesuré', { exact: true }),
  ).toHaveCount(3)
  await expect
    .poll(async () => {
      const response = await page.request.get('/api/knowledge/overview')
      if (!response.ok()) return false
      return (await response.json()).activity.some(
        (e: { kind: string }) => e.kind === 'learn',
      )
    })
    .toBe(true)
  const own = await (await page.request.get('/api/knowledge/overview')).json()
  expect(own.graph).toBeNull()
  expect(own.last_learning).not.toBeNull()
  const before = own.activity.length
  await panel
    .getByRole('textbox', { name: 'Question de vérification' })
    .fill('MEMOIRE SYNTHETIQUE E2E')
  await panel.getByRole('button', { name: 'Tester la recherche' }).click()
  await expect(panel.locator('.kg-search-result')).toContainText('Désactivée')
  await expect(panel.locator('.kg-search-result')).not.toContainText('0 fait')
  const after = await (await page.request.get('/api/knowledge/overview')).json()
  expect(after.activity.length).toBeGreaterThan(before)
  expect(after.activity[0].kind).toBe('search')
  expect(after.activity[0].status).toBe('disabled')
  expect(after.activity[0].result_count).toBeNull()
  const other = await browser.newContext({
    storageState: resolve('test-results/.auth/other.json'),
  })
  try {
    const response = await other.request.get(
      '/api/knowledge/overview?owner_id=1',
    )
    expect(response.ok()).toBe(true)
    const foreign = await response.json()
    const ids = new Set(after.activity.map((e: { id: number }) => e.id))
    expect(foreign.activity.every((e: { id: number }) => !ids.has(e.id))).toBe(
      true,
    )
    const override = await other.request.post('/api/knowledge/search', {
      data: { query: 'TEST', owner_id: 1 },
    })
    expect(override.status()).toBe(400)
  } finally {
    await other.close()
  }
  await page.reload()
  await page.getByRole('tab', { name: 'Mémoire', exact: true }).click()
  await expect(panel.getByText('Désactivée', { exact: true })).toBeVisible()
})
