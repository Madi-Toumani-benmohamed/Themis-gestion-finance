// Anonymous API fixtures only. The test never reads a real PDF or database.
const { chromium } = require('playwright')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const month = new Date().toISOString().slice(0, 7)
const sourceReport = () => ({ detected: 81, parsed: 81, inserted: 0, duplicates: 0, rejected: 0, expected_count: null,
  declared_debits_cents: 429385, declared_credits_cents: 296585, debits_cents: 429385, credits_cents: 296585,
  balance_difference_cents: 0, count_control: 'unverifiable', totals_control: 'passed', balance_control: 'passed',
  completeness: 'unknown', extraction: 'poppler_layout', issues: [], warnings: [], candidates: Array.from({ length: 81 }, (_, index) => ({ index, page: 1 + Math.floor(index / 27), line: 10 + index % 27, status: 'parsed' })) })
let mode = 'failed', confirmations = 0, cancellations = 0
function anonymousTransactions() {
 let d=0
 const credits=new Map([[8,100000],[30,94058],[60,102527]])
 return Array.from({length:81},(_,index)=>{let cents=credits.get(index);if(cents==null){cents=-([29816,269,24300][d-75] ?? 5000);d++}return {date:"2026-09-15",label:`OPERATION ANONYME TEST ${index+1}`,amount:cents/100,duplicate:mode==="duplicate"}})
}
function preview() {
  const report = sourceReport()
  if (mode === 'failed') { report.detected = 82; report.rejected = 1; report.completeness = 'failed'; report.issues = ['missing_amount_or_label']; report.candidates.push({ index: 81, page: 3, line: 45, status: 'uninterpretable', reason: 'missing_amount_or_label' }) }
  return { token: `TEST-${mode}`, expires_at: new Date(Date.now() + 600000).toISOString(), filename: 'ANONYME TEST.pdf', account_id: 11, account_name: 'COURANT TEST',
    parser: 'local', validated: mode !== 'failed', period_start: '2026-08-21', period_end: '2026-09-21', opening_balance: 2492.88, closing_balance: 1164.88, balance_difference: 0,
    new_count: mode === 'duplicate' ? 0 : 81, duplicates: mode === 'duplicate' ? 81 : 0, potential_duplicates: mode === 'similar' ? 2 : 0,
    blocked_reason: mode === 'failed' ? 'Les contrôles ont échoué. Aucune opération ne sera enregistrée.' : mode === 'similar' ? 'Ressemblance avec un autre import : doublon non prouvé.' : '', import_report: report,
    transactions: anonymousTransactions() }
}
async function main() {
 const browser = await chromium.launch({ headless: true, channel: 'chromium' })
 try {
  const context = await browser.newContext({ viewport: { width: 1920, height: 1080 } })
  const page = await context.newPage()
  page.setDefaultTimeout(10000)
  const errors = [], requests = []
  page.on('pageerror', error => errors.push(error.message))
  await page.addInitScript(() => localStorage.setItem('themis-theme', 'dark'))
  await page.route('**/api/**', async route => {
   const req = route.request(), path = new URL(req.url()).pathname
   requests.push(path)
   let data = {}
   if (path === '/api/auth/status') data = { setup_required: false }
   else if (path === '/api/auth/me') data = { id: 1, username: 'IMPORT TEST', role: 'user' }
   else if (path === '/api/health') data = { ai_enabled: false }
   else if (path === '/api/accounts') data = [{ id: 11, name: 'COURANT TEST', kind: 'courant', tracking_mode: 'statements' }]
   else if (path === '/api/summary') data = { monthly: [{ month, income: 0, expenses: 0 }], by_category: [], uncategorized: 0 }
   else if (path === '/api/reliability') data = {month,status:'insufficient',documented_accounts:0,total_accounts:1,current:true,through:`${month}-25`,accounts:[{id:11,name:'COURANT TEST',tracking_mode:'statements',status:'insufficient',gaps:[],statements:[],to_review:0,unknown_periods:0}]}
   else if (path === '/api/period') data = { month, is_current: true, completeness: 'unknown' }
   else if (path === '/api/hero' || path === '/api/remaining') data = { available: false }
   else if (path === '/api/anomalies') data = { anomalies: [] }
   else if (path === '/api/journal') data = { events: [] }
   else if (path === '/api/statements/previews') data = preview()
   else if (/\/previews\/[^/]+\/confirm$/.test(path)) { confirmations++; data = { statement_id: 100, inserted: mode === 'duplicate' ? 0 : 81, duplicates: mode === 'duplicate' ? 81 : 0, parser: 'local', validated: true, import_report: { ...sourceReport(), inserted: 81 } } }
   else if (/\/previews\/[^/]+$/.test(path) && req.method() === 'DELETE') { cancellations++ }
   else if (path === '/api/domains' || path === '/api/categories' || path === '/api/tags' || path === '/api/statements') data = []
   await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(data) }).catch(() => {})
  })
  await page.goto(process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176')
  await page.locator(".sidebar-nav").getByRole("tab", {name:"Relevés",exact:true}).click()
  // Let initial dashboard-only asynchronous requests settle before exercising
  // the upload queue, so an unrelated widget render cannot race the preview.
  await page.waitForTimeout(300)
  const open = async value => {
   mode = value
   await page.locator('input[type=file]').setInputFiles({ name: 'ANONYME TEST.pdf', mimeType: 'application/pdf', buffer: Buffer.from('%PDF-1.4\nTEST') })
   await page.getByRole('dialog', { name: 'Vérifier avant d’importer' }).waitFor().catch(async e=>{console.log({requests,errors,body:await page.locator('body').innerText()});throw e})
  }
  await open('failed')
  let dialog = page.getByRole('dialog')
  assert.equal(await dialog.getByRole('button', { name: 'Import suspendu — vérification nécessaire' }).isDisabled(), true)
  assert.match(await dialog.innerText(), /Page 3, ligne 45/)
  assert.equal(confirmations, 0)
  await page.keyboard.press('Escape')
  await dialog.waitFor({ state: 'detached' })
  assert.equal(cancellations, 1)
  await open('similar')
  assert.equal(await dialog.getByRole('button', { name: 'Import suspendu — vérification nécessaire' }).isDisabled(), true)
  assert.match(await dialog.innerText(), /doublon non prouvé/)
  await dialog.getByRole('button', { name: 'Annuler ce fichier' }).click()
  await dialog.waitFor({ state: 'detached' })
  await open('valid')
  assert.match(await dialog.innerText(), /Non vérifiable automatiquement/)
  assert.match(await dialog.innerText(), /Complétude non établie/)
  assert.match(await dialog.innerText(), /4.?293,85/)
  assert.match(await dialog.innerText(), /2.?965,85/)
  await dialog.getByText('Références des 81 opérations candidates', { exact: true }).click()
  assert.match(await dialog.innerText(), /Opération 81 · page 3/)
  for (const width of [1920, 960, 390]) {
   await page.setViewportSize({ width, height: 900 })
   assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, JSON.stringify(await page.evaluate(()=>({width:innerWidth,scroll:document.documentElement.scrollWidth,overflow:[...document.querySelectorAll("*")].filter(c=>c.getBoundingClientRect().right>innerWidth).map(c=>({tag:c.tagName,class:c.className,right:c.getBoundingClientRect().right})).slice(0,8)}))))
   assert.equal(await dialog.evaluate(el => el.scrollWidth <= el.clientWidth + 1), true, JSON.stringify(await dialog.evaluate(el => ({width:innerWidth,scroll:el.scrollWidth,client:el.clientWidth,children:[...el.querySelectorAll("*")].filter(c=>c.getBoundingClientRect().right>el.getBoundingClientRect().right).map(c=>({tag:c.tagName,class:c.className,width:c.getBoundingClientRect().width})).slice(0,8)}))))
  }
  await page.setViewportSize({ width: 1280, height: 900 })
  const axe = await new AxeBuilder({ page }).include('.import-preview-dialog').analyze()
  assert.deepEqual(axe.violations.map(v => ({ id: v.id, impact: v.impact, targets:v.nodes.map(n=>n.target) })), [])
  await dialog.getByRole('button', { name: 'Confirmer l’import (81 nouvelles)', exact: true }).click()
  await dialog.waitFor({ state: 'detached' })
  assert.equal(confirmations, 1)
  await open('duplicate')
  await dialog.getByRole('button', { name: 'Terminer — aucune nouvelle opération' }).click()
  await dialog.waitFor({ state: 'detached' })
  assert.equal(confirmations, 2)
  assert.deepEqual(errors, [])
  console.log('Import fidelity browser PASS: blocked incomplete extraction, unconfirmed similarity, independent totals/count unavailable, 81 source references, explicit confirmation/replay, Escape, 3 viewports and Axe.')
 } finally { await browser.close() }
}
main().catch(error => { console.error(error); process.exit(1) })
