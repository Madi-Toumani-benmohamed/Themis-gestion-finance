// Synthetic fixtures only. Every API request is intercepted; no banking data.
const { chromium } = require('playwright')
const { AxeBuilder } = require('@axe-core/playwright')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const month = new Date().toISOString().slice(0, 7)
const previous = new Date(`${month}-01T12:00:00Z`)
previous.setUTCMonth(previous.getUTCMonth() - 1)
const previousMonth = previous.toISOString().slice(0, 7)
const categories = [{ id: 1, name: 'Courses TEST', parent_id: null }, { id: 2, name: 'Logement TEST', parent_id: null }, { id: 3, name: 'Transfert TEST', parent_id: null, is_transfer: true }]
const accounts = [{ id: 11, name: 'COURANT TEST' }, { id: 12, name: 'LIVRET TEST' }]
const row = (id, day, amount_cents, display_label, domain_id = 1, financial_type) => ({
  id, date: `${month}-${day}`, amount: amount_cents / 100, amount_cents, display_label, merchant: display_label,
  label: `LIBELLE BANCAIRE ORIGINAL ${display_label} REFERENCE ${id}`, domain_id, tags: [],
  account_id: 11, account_name: 'COURANT TEST', statement_id: 1, statement_filename: 'RELEVE TEST.csv', statement_validated: true,
  statement_period_start: `${month}-01`, statement_period_end: `${month}-21`, classification_source: 'unknown', suggestion: null,
  financial_type: financial_type || (amount_cents < 0 ? 'debit' : amount_cents > 0 ? 'credit' : 'neutral'),
})
let rows = [row(1, '09', -2200, 'TRANSFERT SORTANT TEST', 3, 'transfer'), row(2, '09', 2200, 'TRANSFERT ENTRANT TEST', 3, 'transfer'), row(3, '08', 324947, 'SALAIRE TEST'), row(4, '07', -418480, 'DEPENSE TEST'), row(5, '06', 0, 'A CLASSER TEST', null), row(7, '07', -1234, 'CAFE TEST')]
rows[1].account_id = 12; rows[1].account_name = 'LIVRET TEST'
rows[5].tags = ['taxes']
rows[4].suggestion = { domain_id: 2, tags: ['facture'], source: 'ai' }
for (let i = 0; i < 101; i++) rows.push(row(1000 + i, '01', 0, `OPERATION TEST ${i}`))
let failPatch = new Set(), failReport = false, requests = [], patches = [], empty = false, slow = false
function report(params) {
  let found = empty ? [] : rows.slice()
  const selectedMonth = params.get('month')
  if (selectedMonth === previousMonth) found = [row(2000, '02', -100, 'HISTORIQUE TEST')].map(t => ({ ...t, date: `${previousMonth}-02` }))
  const ids = params.get('ids')?.split(',').map(Number)
  if (ids) found = found.filter(t => ids.includes(t.id))
  if (params.get('account')) found = found.filter(t => String(t.account_id) === params.get('account'))
  if (params.get('domain')) found = found.filter(t => params.get('domain') === 'none' ? t.domain_id == null : String(t.domain_id) === params.get('domain'))
  if (params.get('type')) found = found.filter(t => t.financial_type === params.get('type'))
  if (params.get('tag')) found = found.filter(t => params.get('tag') === '__none__' ? !t.tags.length : t.tags.includes(params.get('tag')))
  if (params.get('q')) { const q = params.get('q').toLowerCase(); found = found.filter(t => `${t.label} ${t.merchant} ${t.display_label} ${t.amount} ${t.tags.join(' ')}`.toLowerCase().includes(q)) }
  if (params.get('from')) found = found.filter(t => t.date >= params.get('from'))
  if (params.get('to')) found = found.filter(t => t.date <= params.get('to'))
  found.sort((a, b) => b.date.localeCompare(a.date) || b.id - a.id)
  const sum = predicate => found.filter(predicate).reduce((s, t) => s + Math.abs(t.amount_cents), 0)
  const expenses = sum(t => t.financial_type === 'debit'), income = sum(t => t.financial_type === 'credit')
  const offset = Number(params.get('offset') || 0), limit = Number(params.get('limit') || 100)
  return { transactions: found.slice(offset, offset + limit), total: found.length, limit, offset,
    query_from: params.get('from') || (selectedMonth ? `${selectedMonth}-01` : ''), query_to: params.get('to') || (selectedMonth ? `${selectedMonth}-21` : ''),
    summary: { expenses: expenses / 100, income: income / 100, net: (income - expenses) / 100, transfer_debits: sum(t => t.financial_type === 'transfer' && t.amount_cents < 0) / 100, transfer_credits: sum(t => t.financial_type === 'transfer' && t.amount_cents > 0) / 100, uncategorized_count: found.filter(t => t.domain_id == null).length } }
}
async function main() {
  const browser = await chromium.launch({ headless: true, channel: 'chromium' })
  const context = await browser.newContext({ viewport: { width: 1920, height: 1080 } })
  const page = await context.newPage(); page.setDefaultTimeout(10000)
  const errors = []; page.on('pageerror', e => errors.push(e.message))
  await page.addInitScript(() => localStorage.setItem('themis-theme', 'dark'))
  await page.route('**/api/**', async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname
    requests.push(path + url.search)
    let data = {}, status = 200
    if (path === '/api/auth/status') data = { setup_required: false }
    else if (path === '/api/auth/me') data = { id: 1, username: 'TRANSACTIONS TEST', role: 'user' }
    else if (path === '/api/health') data = { ai_enabled: true }
    else if (path === '/api/accounts') data = accounts
    else if (path === '/api/domains' || path === '/api/categories') data = categories
    else if (path === '/api/tags') data = [{ name: 'taxes', count: 1 }, { name: 'facture', count: 1 }]
    else if (path === '/api/summary') data = { monthly: [{ month: previousMonth, income: 0, expenses: 1 }, { month, income: 3249.47, expenses: 4197.14 }], by_category: [], uncategorized: 1 }
    else if (path === '/api/period') data = { month: url.searchParams.get('month'), is_current: url.searchParams.get('month') === month, completeness: 'unknown' }
    else if (path === '/api/hero' || path === '/api/remaining') data = { available: false }
    else if (path === '/api/journal') data = { events: [] }
    else if (path === '/api/anomalies') data = { anomalies: [] }
    else if (path === '/api/transactions') {
      if (slow) await new Promise(resolve => setTimeout(resolve, 500))
      data = report(url.searchParams)
      if (failReport) { status = 500; data = { error: 'ERREUR RESEAU TEST' } }
    } else if (/^\/api\/transactions\/\d+$/.test(path) && request.method() === 'PATCH') {
      const id = Number(path.split('/').at(-1)), patch = request.postDataJSON()
      patches.push({ id, patch })
      if (failPatch.has(id)) { status = 500; data = { error: 'ERREUR SAUVEGARDE TEST' } }
      else { rows = rows.map(t => t.id === id ? { ...t, ...patch, ...(Object.hasOwn(patch, 'domain_id') ? { classification_source: 'manual', financial_type: patch.domain_id === 3 ? 'transfer' : t.amount_cents < 0 ? 'debit' : t.amount_cents > 0 ? 'credit' : 'neutral', suggestion: null } : {}), ...(patch.discard_suggestion ? { suggestion: null } : {}), ...(patch.add_tags ? {tags:[...new Set([...t.tags,...patch.add_tags])]} : {}), ...(patch.remove_tags ? {tags:t.tags.filter(tag=>!patch.remove_tags.includes(tag))} : {}) } : t); status = 204 }
    } else if (path === '/api/enrich/ai') data = { suggested: 1, domains_set: 0, tags_set: 0 }
    else if (path === '/api/rules' || path === '/api/merchants/aliases' || path === '/api/merchants/unrecognized') data = []
    return route.fulfill({ status, contentType: 'application/json', body: status === 204 ? undefined : JSON.stringify(data) }).catch(() => {})
  })
  const settle = () => page.evaluate(async () => { await document.fonts.ready; await new Promise(requestAnimationFrame); await Promise.all(document.getAnimations().filter(a => Number.isFinite(a.effect.getComputedTiming().endTime)).map(a => a.finished.catch(() => {}))) })
  const count = text => page.locator('.transaction-list-heading').filter({ hasText: text }).waitFor()
  const search = page.getByRole('searchbox', { name: 'Recherche', exact: true })
  const reset = async () => { await page.getByRole('button', { name: 'Réinitialiser les filtres' }).click(); await count('107 opérations'); await settle() }
  const detail = () => page.getByRole('dialog', { name: 'Détail de la transaction' })
  await page.goto(process.env.THEMIS_UI_URL || 'http://127.0.0.1:5176')
  await page.locator('.sidebar-nav button').filter({ hasText: 'Transactions' }).click()
  await count('107 opérations'); await settle()
  assert.equal(await page.locator('.transactions-table tbody tr:not(.tx-day)').count(), 100)
  assert.match(await page.locator('.transactions-summary').innerText(), /947,67/)
  assert.match(await page.locator('.transactions-summary').innerText(), /4.?197,14/)
  assert.match(await page.locator('.transactions-period').innerText(), /couverture inconnue/)
  assert.ok((await page.locator('.transactions-table').boundingBox()).width > 1200)
  // Full-filter totals remain identical on page 2, with exact cent arithmetic.
  await page.getByRole('button', { name: 'Suivant', exact: true }).click()
  await page.locator('.transaction-pagination').filter({ hasText: 'Page 2 sur 2' }).waitFor()
  assert.match(await page.locator('.transactions-summary').innerText(), /947,67/)
  await reset()
  for (const [label, value, expected] of [['Compte bancaire', '12', '1 opération'], ['Type d’opération', 'transfer', '2 opérations'], ['Catégorie', 'none', '1 opération'], ['Tag', 'taxes', '1 opération']]) {
    await page.getByRole('combobox', { name: label, exact: true }).selectOption(value); await count(expected)
    await reset()
  }
  await search.fill('REFERENCE 7'); await count('1 opération')
  assert.equal(await page.getByRole('button', { name: 'Détail de CAFE TEST', exact: true }).count(), 1)
  const exported = new URL(await page.getByRole('link', { name: 'Exporter ces résultats (CSV)' }).getAttribute('href'), 'http://test')
  assert.equal(exported.searchParams.get('q'), 'REFERENCE 7'); assert.equal(exported.searchParams.has('offset'), false)
  // Keyboard opening, focus containment, complete raw label, and ESC restoration.
  const opener = page.getByRole('button', { name: 'Détail de CAFE TEST', exact: true })
  await opener.focus(); await page.keyboard.press('Enter'); await detail().waitFor()
  assert.ok((await detail().innerText()).includes('LIBELLE BANCAIRE ORIGINAL CAFE TEST REFERENCE 7'))
  for (let i = 0; i < 20; i++) { await page.keyboard.press('Tab'); assert.equal(await page.evaluate(() => document.querySelector('dialog').contains(document.activeElement)), true) }
  await page.keyboard.press('Escape'); await detail().waitFor({ state: 'detached' }); assert.equal(await opener.evaluate(el => el === document.activeElement), true)
  await opener.click()
  await detail().getByRole('textbox', { name: 'Tags', exact: true }).fill('taxes, nouveau')
  failPatch.add(7); await detail().getByRole('button', { name: 'Enregistrer', exact: true }).click()
  await detail().getByRole('alert').filter({ hasText: 'ERREUR SAUVEGARDE TEST' }).waitFor()
  assert.equal(rows.find(t => t.id === 7).tags.includes('nouveau'), false)
  failPatch.clear(); await detail().getByRole('button', { name: 'Enregistrer', exact: true }).click()
  await detail().getByRole('status').filter({ hasText: 'enregistrés' }).waitFor()
  assert.deepEqual(patches.at(-1).patch, { tags: ['taxes', 'nouveau'] })
  await detail().getByRole('button', { name: 'Fermer le détail', exact: true }).click()
  await reset()
  // Manual category change must update whole-filter aggregates immediately.
  await page.getByRole('button', { name: 'Détail de CAFE TEST', exact: true }).click()
  await detail().getByRole('combobox', { name: 'Catégorie', exact: true }).selectOption('3')
  await detail().getByRole('button', { name: 'Enregistrer', exact: true }).click()
  await detail().getByRole('status').filter({ hasText: 'enregistrés' }).waitFor()
  assert.match(await page.locator('.transactions-summary').innerText(), /935,33/)
  await detail().getByRole('button', { name: 'Fermer le détail', exact: true }).click()
  // Suggestions are not assignments; rejecting one is an explicit metadata edit.
  await page.getByRole('button', { name: 'Détail de A CLASSER TEST', exact: true }).click()
  await detail().getByRole('heading', { name: 'Proposition IA à vérifier' }).waitFor()
  assert.equal(rows.find(t => t.id === 5).domain_id, null)
  await detail().getByRole('button', { name: 'Ignorer la suggestion', exact: true }).click()
  await detail().getByRole('heading', { name: 'Proposition IA à vérifier' }).waitFor({ state: 'detached' })
  assert.equal(rows.find(t => t.id === 5).domain_id, null)
  await detail().getByRole('button', { name: 'Fermer le détail', exact: true }).click()
  // Reusing a suggestion only changes the draft; saving is an explicit edit.
  rows = rows.map(t => t.id === 5 ? {...t,suggestion:{domain_id:2,tags:['facture'],source:'ai'}} : t)
  await reset()
  await page.getByRole('button', {name:'Détail de A CLASSER TEST',exact:true}).click()
  const beforeSuggestion = patches.length
  await detail().getByRole('button',{name:'Reprendre dans le formulaire',exact:true}).click()
  assert.equal(patches.length,beforeSuggestion); assert.equal(rows.find(t=>t.id===5).domain_id,null)
  await detail().getByRole('button',{name:'Enregistrer',exact:true}).click()
  await detail().getByRole('status').filter({hasText:'enregistrés'}).waitFor()
  assert.equal(rows.find(t=>t.id===5).domain_id,2); assert.equal(rows.find(t=>t.id===5).classification_source,'manual')
  assert.deepEqual(rows.find(t=>t.id===5).tags,['facture'])
  await detail().getByRole('button',{name:'Fermer le détail',exact:true}).click()
  // Multi-edit confirmation, cancellation and explicit partial failures.
  await page.getByRole('checkbox', { name: 'Sélectionner l’opération 4', exact: true }).check()
  await page.getByRole('checkbox', { name: 'Sélectionner l’opération 7', exact: true }).check()
  await page.getByRole('button', { name: 'Modifier la sélection' }).click()
  let bulk = page.getByRole('dialog', { name: 'Modifier 2 opérations' })
  const before = patches.length
  await bulk.getByRole('button', { name: 'Annuler', exact: true }).click(); assert.equal(patches.length, before)
  await page.getByRole('button', { name: 'Modifier la sélection' }).click()
  await bulk.getByRole('combobox', { name: 'Nouvelle catégorie' }).selectOption('2')
  failPatch.add(7)
  await bulk.getByRole('button', { name: 'Confirmer pour 2 opérations' }).click()
  await bulk.getByRole('status').filter({ hasText: '1 opération(s) enregistrée(s), 1 en échec' }).waitFor()
  assert.equal(await page.getByRole('checkbox', { name: 'Sélectionner l’opération 7', exact: true }).isChecked(), true)
  assert.equal(await page.getByRole('checkbox', { name: 'Sélectionner l’opération 4', exact: true }).isChecked(), false)
  await bulk.getByRole('button', { name: 'Terminer', exact: true }).click(); failPatch.clear()
  await page.getByRole('button',{name:'Désélectionner',exact:true}).click()
  for (const action of ['add_tags','remove_tags']) {
    for (const id of [4,7]) await page.getByRole('checkbox',{name:`Sélectionner l’opération ${id}`,exact:true}).check()
    await page.getByRole('button',{name:'Modifier la sélection'}).click()
    await bulk.getByRole('combobox',{name:'Action',exact:true}).selectOption(action)
    await bulk.getByRole('textbox',{name:'Tags concernés',exact:true}).fill('commun')
    await bulk.getByRole('button',{name:'Confirmer pour 2 opérations'}).click()
    await bulk.getByRole('status').filter({hasText:'2 opération(s) enregistrée(s), 0 en échec'}).waitFor()
    assert.equal(rows.find(t=>t.id===7).tags.includes('commun'),action==='add_tags')
    assert.equal(rows.find(t=>t.id===7).tags.includes('taxes'),true)
    await bulk.getByRole('button',{name:'Terminer',exact:true}).click()
  }
  // Period changes, loading/error/empty and stale response rejection.
  await page.getByRole('combobox', { name: 'Mois', exact: true }).selectOption(previousMonth); await count('1 opération')
  assert.match(await page.locator('.transactions-summary').innerText(), /1,00/)
  await reset()
  failReport = true; await search.fill('FAIL'); await page.getByRole('alert').filter({ hasText: 'ERREUR RESEAU TEST' }).waitFor()
  assert.equal(await page.getByText('Aucune transaction pour ces filtres.', { exact: true }).count(), 0)
  failReport = false; await page.getByRole('button', { name: 'Réessayer', exact: true }).click(); await count('0 opération')
  await page.getByText('Aucune transaction pour ces filtres.', { exact: true }).waitFor()
  slow = true; await search.fill('DEPENSE'); await page.getByRole('status').filter({ hasText: 'Chargement des transactions' }).waitFor()
  slow = false; await search.fill('CAFE'); await count('1 opération'); await page.waitForTimeout(600)
  assert.equal(await page.getByRole('button', { name: 'Détail de CAFE TEST', exact: true }).count(), 1)
  await reset()
  const output = process.env.THEMIS_TRANSACTIONS_ARTIFACTS || '/tmp/themis-transactions-ui'
  fs.mkdirSync(output, { recursive: true })
  const viewports = [{width:1920,height:1080},{width:1280,height:800},{width:960,height:540},{width:768,height:1024},{width:390,height:844},{width:320,height:700}]
  for (const viewport of viewports) {
    await page.setViewportSize(viewport); await settle()
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true, `overflow ${viewport.width}`)
    await page.evaluate(() => scrollTo(0,0))
    await page.screenshot({ path: `${output}/transactions-${viewport.width}.png`, fullPage: false })
    if (viewport.width<=760) {
      const tableWidth=await page.locator('.transactions-table').evaluate(el=>el.clientWidth)
      const labelWidth=await page.locator('.transaction-description').first().evaluate(el=>el.clientWidth)
      assert.ok(labelWidth>=tableWidth*.4,`Unreadable mobile label column: ${labelWidth}/${tableWidth}`)
      await page.locator('.transactions-table').scrollIntoViewIfNeeded()
      await page.screenshot({path:`${output}/transactions-table-${viewport.width}.png`})
    }
    const results = await new AxeBuilder({ page }).analyze()
    assert.deepEqual(results.violations.map(v => ({id:v.id,impact:v.impact,nodes:v.nodes.map(n=>n.target)})), [], `axe ${viewport.width}`)
    await page.getByRole('button', { name: 'Détail de CAFE TEST', exact: true }).click(); await settle()
    assert.equal(await page.evaluate(() => document.querySelector('dialog').scrollWidth <= document.querySelector('dialog').clientWidth), true, `drawer overflow ${viewport.width}`)
    const drawerAxe = await new AxeBuilder({ page }).analyze()
    assert.deepEqual(drawerAxe.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})), [], `drawer axe ${viewport.width}`)
    await page.keyboard.press('Escape')
  }
  await page.setViewportSize({ width: 1920, height: 1080 })
  await page.locator('.sidebar-footer .theme-toggle').click(); await settle()
  const lightAxe = await new AxeBuilder({ page }).analyze(); assert.deepEqual(lightAxe.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})), [], 'light axe')
  const cdp = await context.newCDPSession(page)
  await cdp.send('Emulation.setDeviceMetricsOverride',{width:960,height:540,deviceScaleFactor:2,mobile:false})
  await settle(); assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
  await page.evaluate(()=>scrollTo(0,0)); await page.screenshot({path:`${output}/transactions-200-layout.png`})
  await cdp.send('Emulation.clearDeviceMetricsOverride')
  const chooser = page.waitForEvent('filechooser')
  await page.getByRole('button',{name:'Importer un ou plusieurs relevés',exact:true}).click()
  await (await chooser).setFiles([])
  assert.deepEqual(errors, [])
  console.log('Transactions browser PASS: filters, exact totals, pagination, drawer, errors, edits, suggestions, bulk partial failures, period/stale requests, 6 viewports, keyboard and dark/light Axe.')
  await browser.close()
}
main().catch(error => { console.error(error); process.exit(1) })
