import { expect, test } from '@playwright/test'
import { resolve } from 'node:path'

test('les missions et aperçus de l’agent persistent sans écriture implicite et restent privés', async ({
  page,
  browser,
}) => {
  const transactions = await (
    await page.request.get('/api/transactions')
  ).json()
  const source = transactions.find(
    (t: { label: string }) => t.label === 'ACHAT TEST 001',
  )
  expect(source).toBeTruthy()
  const initialTagNames = (await (await page.request.get('/api/tags')).json())
    .map((tag: { name: string }) => tag.name)
    .sort()
  const originalTags = source.tags || []
  const originalAmount = source.amount
  const created = await page.request.post('/api/agent/missions', {
    data: { objective: 'Examiner un achat SYNTHETIQUE E2E' },
  })
  expect(created.status()).toBe(201)
  const mission = await created.json()
  expect(mission.conversation_id).toBeGreaterThan(0)
  const prepared = await page.request.post('/api/agent/actions', {
    data: {
      conversation_id: mission.conversation_id,
      mission_id: mission.id,
      action: { kind: 'tag', transaction_ids: [source.id], tag: 'agent e2e' },
    },
  })
  expect(prepared.status()).toBe(201)
  const action = await prepared.json()
  expect(action.status).toBe('prepared')
  const readSource = async () =>
    (await (await page.request.get('/api/transactions')).json()).find(
      (t: { id: number }) => t.id === source.id,
    )
  expect((await readSource()).tags || []).toEqual(originalTags)
  const other = await browser.newContext({
    storageState: resolve('test-results/.auth/other.json'),
  })
  try {
    expect(
      (
        await other.request.get(
          `/api/agent/conversations/${mission.conversation_id}`,
        )
      ).status(),
    ).toBe(404)
    expect(
      (
        await other.request.post(`/api/agent/actions/${action.id}/decision`, {
          data: { decision: 'apply' },
        })
      ).status(),
    ).toBe(404)
    const foreign = await (
      await other.request.get('/api/agent/missions')
    ).json()
    expect(
      foreign.missions.some((m: { id: number }) => m.id === mission.id),
    ).toBe(false)
    const applied = await page.request.post(
      `/api/agent/actions/${action.id}/decision`,
      { data: { decision: 'apply' } },
    )
    expect(applied.status()).toBe(200)
    expect((await readSource()).tags).toContain('agent e2e')
    expect((await readSource()).amount).toBe(originalAmount)
    const replay = await page.request.post(
      `/api/agent/actions/${action.id}/decision`,
      { data: { decision: 'apply' } },
    )
    expect(replay.status()).toBe(200)
    expect(
      (await readSource()).tags.filter((tag: string) => tag === 'agent e2e'),
    ).toHaveLength(1)
    const history = await (
      await page.request.get(
        `/api/agent/actions?conversation_id=${mission.conversation_id}`,
      )
    ).json()
    expect(
      history.actions.find((a: { id: number }) => a.id === action.id).status,
    ).toBe('applied')
    expect(
      (
        await page.request.patch(`/api/agent/missions/${mission.id}`, {
          data: { status: 'paused' },
        })
      ).status(),
    ).toBe(204)
    const resumed = await (await page.request.get('/api/agent/missions')).json()
    expect(
      resumed.missions.find((m: { id: number }) => m.id === mission.id).status,
    ).toBe('paused')
  } finally {
    await other.close()
    // Restore the synthetic source used by the unrelated visual contracts.
    expect(
      (
        await page.request.patch(`/api/transactions/${source.id}`, {
          data: { tags: originalTags },
        })
      ).ok(),
    ).toBe(true)
    // Removing a transaction tag does not delete its vocabulary entry. Keep
    // the global filter list identical for subsequent visual tests as well.
    expect(
      (
        await page.request.delete(
          '/api/tags/' + encodeURIComponent('agent e2e'),
        )
      ).ok(),
    ).toBe(true)
    const tagsAfter = await (await page.request.get('/api/tags')).json()
    expect(tagsAfter.map((tag: { name: string }) => tag.name).sort()).toEqual(
      initialTagNames,
    )
    expect(
      (
        await page.request.patch(`/api/agent/missions/${mission.id}`, {
          data: { status: 'completed' },
        })
      ).status(),
    ).toBe(204)
  }
  await page.goto('/')
  await page.getByRole('tab', { name: /^Transactions(?: \d+)?$/ }).click()
  await expect(
    page.getByRole('table', {
      name: 'Transactions regroupées par date, selon les filtres actifs',
    }),
  ).toBeVisible()
  await expect(
    page.getByRole('combobox', { name: 'Tag', exact: true }),
  ).toHaveCount(initialTagNames.length ? 1 : 0)
})
