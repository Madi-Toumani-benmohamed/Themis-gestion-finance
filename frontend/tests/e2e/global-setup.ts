import { request, type APIRequestContext, type APIResponse, type FullConfig } from '@playwright/test'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { spawnSync } from 'node:child_process'

const apiURL = process.env.THEMIS_E2E_API_URL ?? 'http://127.0.0.1:18080'
const admin = { username: 'e2e-owner-a', password: 'E2E-finance-2026-safe' }
const other = { username: 'e2e-owner-b', password: 'E2E-finance-2026-other' }
const output = resolve('test-results')

async function checkedJSON(response: APIResponse) {
  if (!response.ok()) throw new Error(`Préparation E2E échouée (${response.status()}).`)
  return response.json()
}

async function importSyntheticStatement(context: APIRequestContext, accountID: number) {
  const pdfPath = resolve('tests/fixtures/statement-009.pdf')
  const file = await readFile(pdfPath)
  const previewResponse = await context.post('/api/statements/previews', {
    multipart: { account_id: String(accountID), file: { name: 'releve-test-synthetique.pdf', mimeType: 'application/pdf', buffer: file } },
    timeout: 180_000,
  })
  const preview = await checkedJSON(previewResponse)
  const report = preview.import_report
  if (preview.new_count !== 81 || report?.parsed !== 81 || report?.debits_cents !== 429385 || report?.credits_cents !== 296585) {
    throw new Error('Le relevé synthétique ne respecte pas ses invariants d’extraction attendus.')
  }
  if (preview.opening_balance !== 2492.88 || preview.closing_balance !== 1164.88 || preview.balance_difference !== 0) {
    throw new Error('Le rapprochement du relevé synthétique est incohérent.')
  }
  const confirmed = await checkedJSON(await context.post(`/api/statements/previews/${preview.token}/confirm`))
  if (confirmed.inserted !== 81 || confirmed.duplicates !== 0) throw new Error('L’import synthétique n’a pas créé exactement 81 opérations.')

  const transactions = await checkedJSON(await context.get(`/api/transactions?statement_id=${confirmed.statement_id}`))
  if (transactions.length !== 81) throw new Error('Le nombre de transactions persistées ne correspond pas au relevé.')
  const cents = transactions.reduce((sum: bigint, tx: { amount: number }) => sum + BigInt(Math.round(tx.amount * 100)), 0n)
  if (cents !== -132800n) throw new Error('La somme des mouvements persistés ne correspond pas au solde net du relevé.')
  return { statementID: confirmed.statement_id as number, transactionCount: transactions.length }
}

export default async function globalSetup(_config: FullConfig) {
  const backendDir = resolve('../backend')
  const fixturePath = resolve('tests/fixtures/statement-009.pdf')
  const generated = spawnSync('go', ['run', './cmd/e2efixtures', 'statement', fixturePath], { cwd: backendDir, encoding: 'utf8' })
  if (generated.status !== 0) throw new Error(`Génération du relevé synthétique échouée: ${generated.stderr}`)

  const statusContext = await request.newContext({ baseURL: apiURL })
  const status = await checkedJSON(await statusContext.get('/api/auth/status'))
  if (!status.setup_required) {
    await statusContext.dispose()
    throw new Error('La base E2E n’est pas vierge. Utiliser npm run test:e2e:local pour la réinitialiser.')
  }
  const adminContext = await request.newContext({ baseURL: apiURL })
  await checkedJSON(await adminContext.post('/api/auth/setup', { data: admin }))
  await checkedJSON(await adminContext.post('/api/admin/users', { data: { ...other, role: 'user' } }))
  const adminAccounts = await checkedJSON(await adminContext.get('/api/accounts'))
  const adminStatement = await importSyntheticStatement(adminContext, adminAccounts[0].id)
  const adminTransactions = await checkedJSON(await adminContext.get(`/api/transactions?statement_id=${adminStatement.statementID}`))
  const categorizedSample = adminTransactions.find((tx: { label: string }) => tx.label === 'ACHAT TEST 001')
  const adminDomains = await checkedJSON(await adminContext.get('/api/categories'))
  const foodDomain = adminDomains.find((domain: { name: string }) => domain.name === 'Alimentation')
  if (!categorizedSample || !foodDomain) throw new Error('Fixture de dépense catégorisée indisponible.')
  const categorized = await adminContext.patch(`/api/transactions/${categorizedSample.id}`, { data: { domain_id: foodDomain.id } })
  if (!categorized.ok()) throw new Error('Impossible de préparer la catégorie de la dépense synthétique.')

  const otherContext = await request.newContext({ baseURL: apiURL })
  await checkedJSON(await otherContext.post('/api/auth/login', { data: other }))
  const otherAccounts = await checkedJSON(await otherContext.get('/api/accounts'))
  const otherStatement = await importSyntheticStatement(otherContext, otherAccounts[0].id)
  // Keep visual references independent of the wall-clock upload date while
  // retaining real API responses and all imported financial values.
  for (const statementID of [adminStatement.statementID, otherStatement.statementID]) {
    const dated = spawnSync('go', ['run', './cmd/e2efixtures', 'visual-import-time', String(statementID)], { cwd: backendDir, encoding: 'utf8' })
    if (dated.status !== 0) throw new Error(`Date d’import E2E non stabilisée : ${dated.stderr}`)
  }
  const domains = await checkedJSON(await otherContext.get('/api/categories'))
  const ordinaryDomain = domains.find((domain: { is_transfer?: boolean }) => !domain.is_transfer)
  if (!ordinaryDomain) throw new Error('Aucun domaine ordinaire disponible pour tester l’isolation des budgets.')
  await checkedJSON(await otherContext.post('/api/budgets', { data: { domain_id: ordinaryDomain.id, monthly_limit: 250 } }))
  const otherBudgets = await checkedJSON(await otherContext.get('/api/budgets'))

  await mkdir(resolve(output, '.auth'), { recursive: true })
  await adminContext.storageState({ path: resolve(output, '.auth/admin.json') })
  await otherContext.storageState({ path: resolve(output, '.auth/other.json') })
  await writeFile(resolve(output, 'e2e-fixture.json'), JSON.stringify({ adminStatement, otherStatement, adminFoodDomainID: foodDomain.id, otherBudgetID: otherBudgets[0].id }))
  await Promise.all([adminContext.dispose(), otherContext.dispose(), statusContext.dispose()])
}
