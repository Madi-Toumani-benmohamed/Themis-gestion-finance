import { expect, test } from '@playwright/test'

test('un projet retrouve une dépense réelle, exige une confirmation et conserve les décisions', async ({
  page,
  browser,
}) => {
  await page.goto('/')
  await page.getByRole('tab', { name: 'Analyse', exact: true }).click()
  await page.getByLabel('Mois analysé').fill('2026-08')
  await page.getByRole('button', { name: 'Pilotage', exact: true }).click()
  const projects = page.getByRole('region', {
    name: 'Projets financiers',
    exact: true,
  })
  await projects
    .getByRole('button', { name: 'Créer un projet', exact: true })
    .click()
  await page.getByLabel('Nom du projet').fill('Voyage E2E TEST')
  await page.getByLabel('Budget prévisionnel (€), facultatif').fill('30.00')
  await page.getByLabel('Date de début', { exact: true }).fill('2026-08-01')
  await page.getByLabel('Date de fin', { exact: true }).fill('2026-08-31')
  const saved = page.waitForResponse(
    (r) =>
      r.request().method() === 'POST' &&
      new URL(r.url()).pathname === '/api/projects',
  )
  await page.getByRole('button', { name: 'Enregistrer le projet' }).click()
  const { id } = await (await saved).json()
  const report = async () => {
    const response = await page.request.get(`/api/projects/${id}?month=2026-08`)
    expect(response.ok()).toBeTruthy()
    return response.json()
  }
  const detail = page.getByRole('article', { name: 'Détail du projet' })
  await expect(detail).toContainText('Aucune dépense confirmée')
  await page.getByText('Configurer les règles', { exact: true }).click()
  await page
    .getByRole('combobox', { name: 'Critère', exact: true })
    .selectOption('label')
  await page.getByLabel('Valeur du critère').fill('ACHAT TEST 001')
  await page.getByRole('button', { name: 'Ajouter et rechercher' }).click()
  await expect
    .poll(async () => (await report()).project.suggestion_count)
    .toBe(1)
  expect((await report()).project.cost_cents).toBe(0)
  await page.getByLabel('Afficher les opérations').selectOption('suggested')
  await expect(
    detail.getByRole('table', { name: /suggestions exclues/ }),
  ).toContainText('TEST')
  await detail.getByRole('checkbox').check()
  await page.getByRole('button', { name: 'Confirmer la sélection (1)' }).click()
  await expect.poll(async () => (await report()).project.cost_cents).toBe(5000)
  const confirmed = (await report()).project
  expect(confirmed.remaining_cents).toBe(-2000)
  expect(confirmed.overrun_cents).toBe(2000)
  await expect(detail).toContainText(
    'Le coût confirmé observé dépasse le budget.',
  )
  const other = await browser.newContext({
    baseURL: process.env.THEMIS_E2E_BASE_URL ?? 'http://127.0.0.1:15176',
    storageState: 'test-results/.auth/other.json',
  })
  expect((await other.request.get(`/api/projects/${id}`)).status()).toBe(404)
  expect(
    (
      await other.request.post(`/api/projects/${id}/decisions`, {
        data: { transaction_ids: [1], state: 'confirmed' },
      })
    ).status(),
  ).toBe(404)
  await other.close()
  await page.reload()
  await page.getByRole('tab', { name: 'Analyse', exact: true }).click()
  await page.getByLabel('Mois analysé').fill('2026-08')
  await page.getByRole('button', { name: 'Pilotage', exact: true }).click()
  await page
    .getByRole('button', { name: /Voyage E2E TEST.*Coût observé/ })
    .click()
  await expect(detail).toContainText('50,00')
  await page.getByRole('button', { name: 'Retirer', exact: true }).click()
  await expect.poll(async () => (await report()).project.cost_cents).toBe(0)
  await page.getByRole('button', { name: 'Rechercher les suggestions' }).click()
  await expect
    .poll(async () => (await report()).project.suggestion_count)
    .toBe(0)
  const history = await (
    await page.request.get(`/api/projects/${id}/history`)
  ).json()
  expect(
    history.history.filter(
      (row: { event: string }) => row.event === 'decision',
    ),
  ).toHaveLength(2)
  // Leave the shared fixture intact for subsequent transaction and visual tests.
  const p = (await report()).project
  const archived = await page.request.put(`/api/projects/${id}`, {
    data: {
      name: p.name,
      description: p.description,
      budget_cents: p.budget_cents,
      start_date: p.start_date,
      end_date: p.end_date,
      status: 'archived',
    },
  })
  expect(archived.ok()).toBeTruthy()
})
