import { expect, test } from '@playwright/test'

test('un budget est créé, persiste, se modifie puis se supprime', async ({
  page,
}) => {
  const baseline = await (await page.request.get('/api/budgets')).json()
  const baselineIDs = new Set(baseline.map((row: { id: number }) => row.id))
  expect(
    baseline.some(
      (row: { domain_name: string }) => row.domain_name === 'Alimentation',
    ),
  ).toBe(false)
  try {
    await page.goto('/')
    await page.getByRole('tab', { name: 'Budgets' }).click()
    await expect(
      page.getByRole('heading', { name: /Des enveloppes/ }),
    ).toBeVisible()
    const month = page.getByLabel('Mois consulté')
    await month.fill('2026-08')
    await page.getByRole('button', { name: 'Créer un budget' }).click()
    const dialog = page.getByRole('dialog', { name: 'Créer une enveloppe' })
    await dialog.getByLabel('Catégorie').selectOption({ label: 'Alimentation' })
    await dialog.getByLabel('Plafond mensuel (€)').fill('250.00')
    const creating = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname === '/api/budgets' &&
        response.request().method() === 'POST',
    )
    await dialog.getByRole('button', { name: 'Créer l’enveloppe' }).click()
    expect((await creating).ok()).toBe(true)
    await expect(dialog).not.toBeVisible()
    const budget = await page.request.get('/api/budgets')
    let budgets = await budget.json()
    let created = budgets.find(
      (row: { domain_name: string }) => row.domain_name === 'Alimentation',
    )
    expect(created.monthly_limit).toBe(250)
    const spending = await (
      await page.request.get('/api/budgets/report?month=2026-08')
    ).json()
    const food = spending.budgets.find(
      (row: { domain_name: string }) => row.domain_name === 'Alimentation',
    )
    expect(food.spent).toBe(50)
    expect(food.remaining).toBe(200)
    await page.reload()
    await page.getByRole('tab', { name: 'Budgets' }).click()
    await page.getByLabel('Mois consulté').fill('2026-08')
    await expect(page.getByText('Alimentation').first()).toBeVisible()
    await page
      .getByRole('button', { name: /Modifier/ })
      .first()
      .click()
    const edit = page.getByRole('dialog', { name: 'Modifier le plafond' })
    await edit.getByLabel('Plafond mensuel (€)').fill('275.50')
    const saving = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname === '/api/budgets' &&
        response.request().method() === 'POST',
    )
    await edit.getByRole('button', { name: 'Enregistrer le plafond' }).click()
    expect((await saving).ok()).toBe(true)
    await expect(edit).not.toBeVisible()
    budgets = await (await page.request.get('/api/budgets')).json()
    created = budgets.find(
      (row: { domain_name: string }) => row.domain_name === 'Alimentation',
    )
    expect(created.monthly_limit).toBe(275.5)
    const updatedSpending = await (
      await page.request.get('/api/budgets/report?month=2026-08')
    ).json()
    const updatedFood = updatedSpending.budgets.find(
      (row: { domain_name: string }) => row.domain_name === 'Alimentation',
    )
    expect(updatedFood.remaining).toBe(225.5)
    await page
      .getByRole('button', { name: /Supprimer/ })
      .first()
      .click()
    const deleting = page.waitForResponse(
      (response) =>
        new URL(response.url()).pathname === `/api/budgets/${created.id}` &&
        response.request().method() === 'DELETE',
    )
    await page
      .getByRole('dialog', { name: 'Supprimer cette enveloppe ?' })
      .getByRole('button', { name: 'Confirmer la suppression' })
      .click()
    expect((await deleting).ok()).toBe(true)
    await expect(
      page.getByRole('dialog', { name: 'Supprimer cette enveloppe ?' }),
    ).not.toBeVisible()
    budgets = await (await page.request.get('/api/budgets')).json()
    expect(
      budgets.some(
        (row: { domain_name: string }) => row.domain_name === 'Alimentation',
      ),
    ).toBeFalsy()
  } finally {
    // Retries and later visual tests must never inherit a partially tested envelope.
    const remaining = await (await page.request.get('/api/budgets')).json()
    for (const row of remaining) {
      if (row.domain_name === 'Alimentation' && !baselineIDs.has(row.id)) {
        expect((await page.request.delete(`/api/budgets/${row.id}`)).ok()).toBe(
          true,
        )
      }
    }
  }
})
