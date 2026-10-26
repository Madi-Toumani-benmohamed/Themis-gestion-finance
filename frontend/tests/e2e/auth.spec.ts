import { expect, test } from '@playwright/test'
import { spawnSync } from 'node:child_process'

const credentials = { username: 'e2e-owner-a', password: 'E2E-finance-2026-safe' }

test.describe('authentification réelle', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('refuse un accès API protégé sans session et les identifiants invalides', async ({ page }) => {
    const protectedResponse = await page.request.get('/api/auth/me')
    expect(protectedResponse.status()).toBe(401)
    await page.goto('/')
    await expect(page.getByLabel('Identifiant')).toBeVisible()
    await page.getByLabel('Identifiant').fill(credentials.username)
    await page.getByLabel('Mot de passe', { exact: true }).fill('motdepasse-incorrect-123')
    await page.getByRole('button', { name: 'Se connecter' }).click()
    await expect(page.getByText('identifiants incorrects')).toBeVisible()
  })

  test('permet la connexion et invalide la session à la déconnexion', async ({ page }) => {
    await page.goto('/')
    await page.getByLabel('Identifiant').fill(credentials.username)
    await page.getByLabel('Mot de passe', { exact: true }).fill(credentials.password)
    await page.getByRole('button', { name: 'Se connecter' }).click()
    await expect(page.getByText(credentials.username).first()).toBeVisible()
    await page.getByRole('button', { name: 'déconnexion' }).click()
    await expect(page.getByLabel('Identifiant')).toBeVisible()
    expect((await page.request.get('/api/auth/me')).status()).toBe(401)
  })

  test('une session expirée est rejetée après rechargement', async ({ page }) => {
    await page.goto('/')
    await page.getByLabel('Identifiant').fill(credentials.username)
    await page.getByLabel('Mot de passe', { exact: true }).fill(credentials.password)
    await page.getByRole('button', { name: 'Se connecter' }).click()
    await expect(page.getByText(credentials.username).first()).toBeVisible()
    const cookie = (await page.context().cookies()).find((entry) => entry.name === 'themis_session')
    expect(cookie?.value).toBeTruthy()
    const expired = spawnSync('go', ['run', './cmd/e2efixtures', 'expire-session', '-'], {
      cwd: '../backend',
      input: cookie?.value,
      encoding: 'utf8',
      env: { ...process.env, THEMIS_E2E_DATABASE_URL: process.env.THEMIS_E2E_DATABASE_URL },
    })
    expect(expired.status, expired.stderr).toBe(0)
    await page.reload()
    await expect(page.getByLabel('Identifiant')).toBeVisible()
    expect((await page.request.get('/api/auth/me')).status()).toBe(401)
  })
})
