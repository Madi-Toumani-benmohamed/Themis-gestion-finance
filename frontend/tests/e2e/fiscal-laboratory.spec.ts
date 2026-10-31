import { expect, test } from '@playwright/test'
import { resolve } from 'node:path'

test('Fiscalité et Laboratoire fonctionnent sans IA et isolent les documents', async ({
  page,
  browser,
}) => {
  await page.goto('/')
  await page.getByRole('tab', { name: 'Impôts', exact: true }).click()
  await expect(
    page.getByRole('heading', {
      name: 'Fiscalité · impôt sur le revenu',
      exact: true,
    }),
  ).toBeVisible()
  await page.getByRole('tab', { name: 'Simulation', exact: true }).click()
  await expect(page.locator('.scenario-heading')).toBeVisible()
  await page.getByLabel('Mode de simulation').selectOption('fiscal')
  await page.getByLabel('Base de revenu').selectOption('final_taxable')
  await page
    .getByRole('textbox', {
      name: 'Revenu net imposable final (€)',
      exact: true,
    })
    .fill('30000')
  await page.getByRole('checkbox', { name: /Je confirme/ }).check()
  await page.getByRole('button', { name: 'Estimer mon impôt' }).click()
  await expect(
    page.getByRole('region', { name: 'Résultat fiscal' }),
  ).toContainText(/2.104,00/)
  await expect(page.getByText(/Pas de solde final estimable/)).toBeVisible()
  await page.getByRole('tab', { name: 'Laboratoire', exact: true }).click()
  await page.getByRole('button', { name: 'Nouvelle feuille' }).click()
  const a = page.getByLabel('A1', { exact: true }),
    b = page.getByLabel('B1', { exact: true })
  await a.dblclick()
  await a.fill('0.1')
  await a.press('Enter')
  await b.dblclick()
  await b.fill('=A1+0.2')
  await b.press('Enter')
  await expect(b).toHaveValue('0.3')
  await page.getByLabel('Titre du document').fill('LABORATOIRE E2E SYNTHETIQUE')
  const saved = page.waitForResponse(
    (r) =>
      r.request().method() === 'POST' &&
      new URL(r.url()).pathname === '/api/laboratory',
  )
  await page
    .getByRole('button', { name: 'Enregistrer le document', exact: true })
    .click()
  const doc = await (await saved).json()
  expect(doc.revision).toBe(1)
  const other = await browser.newContext({
    storageState: resolve('test-results/.auth/other.json'),
  })
  const payload = {
    client_key: doc.client_key,
    title: doc.title,
    type: doc.type,
    schema_version: 1,
    revision: 1,
    content: doc.content,
  }
  expect(
    (
      await other.request.put(`/api/laboratory/${doc.id}`, { data: payload })
    ).status(),
  ).toBe(404)
  expect(
    (
      await other.request.delete(`/api/laboratory/${doc.id}`, {
        data: { revision: 1 },
      })
    ).status(),
  ).toBe(404)
  const foreign = await other.request.get(`/api/laboratory/${doc.id}`)
  expect(foreign.status()).toBe(404)
  const listing = await (await other.request.get('/api/laboratory')).json()
  expect(listing.documents.some((d: { id: number }) => d.id === doc.id)).toBe(
    false,
  )
  await other.close()
  await page
    .getByRole('button', {
      name: 'Fermer LABORATOIRE E2E SYNTHETIQUE',
      exact: true,
    })
    .click()
  await page
    .getByText('Ouvrir ou supprimer un document serveur', { exact: true })
    .click()
  await page
    .getByRole('button', { name: /LABORATOIRE E2E SYNTHETIQUE · révision 1/ })
    .click()
  await expect(page.getByLabel('B1', { exact: true })).toHaveValue('0.3')
})
