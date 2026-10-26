import { defineConfig, devices } from '@playwright/test'
import { resolve } from 'node:path'

const baseURL = process.env.THEMIS_E2E_BASE_URL ?? 'http://127.0.0.1:15176'
const apiURL = process.env.THEMIS_E2E_API_URL ?? 'http://127.0.0.1:18080'
const databaseURL = process.env.THEMIS_E2E_DATABASE_URL ?? 'postgres://themis_e2e:e2e_only_password@127.0.0.1:55432/themis_e2e?sslmode=disable'
let databaseName = ''
try {
  databaseName = new URL(databaseURL).pathname.replace(/^\//, '')
} catch {
  throw new Error('THEMIS_E2E_DATABASE_URL doit être une URL PostgreSQL dédiée.')
}
if (databaseName !== 'themis_e2e') {
  throw new Error('La configuration E2E refuse toute base dont le nom n’est pas themis_e2e.')
}

export default defineConfig({
  testDir: './tests/e2e',
  globalSetup: './tests/e2e/global-setup.ts',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  outputDir: './test-results',
  reporter: [
    ['list'],
    ['html', { outputFolder: './playwright-report', open: 'never' }],
    ['junit', { outputFile: './test-results/junit.xml' }],
  ],
  use: {
    baseURL,
    storageState: resolve('test-results/.auth/admin.json'),
    ...devices['Desktop Chrome'],
    viewport: { width: 1440, height: 1000 },
    actionTimeout: 10_000,
    navigationTimeout: 20_000,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
    colorScheme: 'light',
    locale: 'fr-FR',
    timezoneId: 'Europe/Paris',
    launchOptions: { args: ['--disable-lcd-text'] },
  },
  webServer: [
    {
      command: 'go run ./cmd/server',
      cwd: '../backend',
      url: `${apiURL}/api/health`,
      reuseExistingServer: false,
      timeout: 120_000,
      env: {
        DATABASE_URL: databaseURL,
        PORT: '18080',
        COOKIE_SECURE: 'false',
        AI_PROVIDER: 'none',
        OPENAI_API_KEY: '',
        ANTHROPIC_API_KEY: '',
        MISTRAL_API_KEY: '',
        FREELLM_API_KEY: '',
        OLLAMA_URL: '',
        MEMORY_ENABLED: 'false',
      },
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 15176 --strictPort',
      url: baseURL,
      reuseExistingServer: false,
      timeout: 60_000,
      env: { THEMIS_API_PROXY_TARGET: apiURL },
    },
  ],
})
