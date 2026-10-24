// Icône sémantique discrète pour les opérations interprétées par le
// normaliseur (versement d'épargne, cotisation, prestation, santé).
export const kindIcon = (k) => ({ savings: '💰', tax: '🏛️', benefit: '🧾', health: '🩺' }[k] || null)
const json = (r) => {
  if (r.status === 204) return null
  return r.json().then((body) => {
    if (!r.ok) throw new Error(body?.error || `Erreur ${r.status}`)
    return body
  })
}

// toutes les requêtes portent le cookie de session
const G = (u) => fetch(u, { credentials: 'same-origin' }).then(json)
const J = (u, method, body) =>
  fetch(u, {
    method,
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  }).then(json)

export const api = {
  health: () => fetch('/api/health').then(json),
  tagsInCategory: (categoryId, month) => {
    const p = new URLSearchParams({ category: categoryId })
    if (month) p.set('month', month)
    return fetch(`/api/analytics/tags-in-category?${p}`).then(json)
  },
  tagSeries: (tags, period = 'month') => {
    const p = new URLSearchParams({ period })
    if (tags && tags.length) p.set('tags', tags.join(','))
    return fetch(`/api/analytics/tag-series?${p}`).then(json)
  },
  anomalies: (month) => fetch('/api/anomalies' + (month ? `?month=${month}` : '')).then(json),
  journal: (month) => fetch('/api/journal' + (month ? `?month=${month}` : '')).then(json),
  remaining: (month) => fetch('/api/remaining' + (month ? `?month=${month}` : '')).then(json),
  reliability: (month) => G('/api/reliability?' + new URLSearchParams({ month })),
  coverage: () => fetch('/api/coverage').then(json),
  intelMerchants: (months) => fetch(`/api/intel/merchants?months=${months || 3}`).then(json),
  intelNature: () => fetch('/api/intel/nature').then(json),
  intelComparison: () => fetch('/api/intel/comparison').then(json),
  intelDiscoveries: () => fetch('/api/intel/discoveries').then(json),
  intelNarrative: () => fetch('/api/intel/narrative').then(json),
  hero: (month) => fetch('/api/hero' + (month ? `?month=${month}` : '')).then(json),
  coherence: (ai) => fetch('/api/coherence' + (ai ? '?ai=1' : '')).then(json),
  unrecognizedMerchants: () => fetch('/api/merchants/unrecognized').then(json),
  merchantAliases: () => fetch('/api/merchants/aliases').then(json),
  setMerchantAlias: (matchKey, display, isPattern = false) => J('/api/merchants/aliases', 'POST', { match_key: matchKey, display, is_pattern: isPattern }),
  deleteMerchantAlias: (id) => J(`/api/merchants/aliases/${id}`, 'DELETE'),
  suggestMerchantNames: (keys) => J('/api/merchants/suggest', 'POST', { keys }),
  pulse: (month) => fetch('/api/pulse' + (month ? `?month=${month}` : '')).then(json),
  summary: (months = 12, month) => fetch(`/api/summary?months=${months}` + (month ? `&month=${month}` : '')).then(json),
  transactions: (params = {}) =>
    fetch('/api/transactions?' + new URLSearchParams(params)).then(json),
  categories: () => fetch('/api/categories').then(json),
  rules: () => fetch('/api/rules').then(json),
  deleteRule: (id) => fetch(`/api/rules/${id}`, { method: 'DELETE', credentials: 'same-origin' }).then((r) => r.ok),
  updateRule: (id, pattern, domainId, apply = true, tags = []) =>
    fetch(`/api/rules/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'same-origin',
      body: JSON.stringify({ pattern, domain_id: domainId, tags, apply }),
    }).then(json),
  createRule: (pattern, domainId, apply = true, tags = []) =>
    fetch('/api/rules', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pattern, domain_id: domainId, tags, apply }),
    }).then(json),
  statements: () => fetch('/api/statements').then(json),
  deleteStatement: (id) => fetch(`/api/statements/${id}`, { method: 'DELETE' }).then(json),
  upload: (file, accountId, progressToken) => {
    const fd = new FormData()
    fd.append('file', file)
    if (accountId) fd.append('account_id', String(accountId))
    if (progressToken) fd.append('progress_token', progressToken)
    return fetch('/api/statements', { method: 'POST', body: fd, credentials: 'same-origin' }).then(json)
  },
  previewStatement: (file, accountId, progressToken) => {
    const fd = new FormData()
    fd.append('file', file)
    if (accountId) fd.append('account_id', String(accountId))
    if (progressToken) fd.append('progress_token', progressToken)
    return fetch('/api/statements/previews', { method: 'POST', body: fd, credentials: 'same-origin' }).then(json)
  },
  confirmStatementPreview: token => J(`/api/statements/previews/${encodeURIComponent(token)}/confirm`, 'POST'),
  cancelStatementPreview: token => J(`/api/statements/previews/${encodeURIComponent(token)}`, 'DELETE'),
  uploadProgress: (token) => fetch(`/api/upload/progress?token=${encodeURIComponent(token)}`).then(json),
  budgetReport: month => G('/api/budgets/report?' + new URLSearchParams({ month })),
  recurringReview: month => G('/api/recurring/review?' + new URLSearchParams({ month })),
  updateRecurringReview: (key, patch, month) => J(`/api/recurring/review/${encodeURIComponent(key)}${month ? '?' + new URLSearchParams({ month }) : ''}`, 'PUT', patch),
  budgets: (month) => fetch('/api/budgets' + (month ? `?month=${month}` : '')).then(json),
  setBudget: (domainId, monthlyLimit) =>
    fetch('/api/budgets', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain_id: domainId, monthly_limit: monthlyLimit }),
    }).then(json),
  deleteBudget: (id) => fetch(`/api/budgets/${id}`, { method: 'DELETE' }).then(json),
  recurring: (minMonths = 2) => fetch(`/api/recurring?min_months=${minMonths}`).then(json),
  adminConfig: () => fetch('/api/admin/config').then(json),
  adminPutConfig: (cfg) =>
    fetch('/api/admin/config', {
      method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(cfg),
    }).then(json),
  adminTestAI: (cfg) =>
    fetch('/api/admin/ai/test', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(cfg || {}),
    }).then(json),
  adminOllamaModels: (url) =>
    fetch('/api/admin/ollama/models' + (url ? `?url=${encodeURIComponent(url)}` : '')).then(json),
  adminStats: () => fetch('/api/admin/stats').then(json),
  adminReapplyRules: () => fetch('/api/admin/reapply-rules', { method: 'POST' }).then(json),
  applyRules: () => fetch('/api/rules/apply', { method: 'POST', credentials: 'same-origin' }).then(json),
  adminPurge: (confirm) =>
    fetch('/api/admin/purge', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ confirm }),
    }).then(json),
  getSetting: (key) => fetch(`/api/settings/${key}`).then(json),
  putSetting: (key, value) =>
    fetch(`/api/settings/${key}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(value),
    }).then(json),
  exportUrl: (month) => '/api/transactions/export' + (month ? `?month=${month}` : ''),
  getAnalysis: (month) => fetch(`/api/analyses/${month}`).then(json),
  createAnalysis: (month) => fetch(`/api/analyses/${month}`, { method: 'POST' }).then(json),

  // Authentification
  authStatus: () => G('/api/auth/status'),
  authMe: () => G('/api/auth/me'),
  authSetup: (username, password) => J('/api/auth/setup', 'POST', { username, password }),
  authLogin: (username, password) => J('/api/auth/login', 'POST', { username, password }),
  authLogout: () => J('/api/auth/logout', 'POST'),
  changePassword: (current, next) => J('/api/auth/password', 'POST', { current, new: next }),
  users: () => G('/api/admin/users'),
  createUser: (username, password, role) => J('/api/admin/users', 'POST', { username, password, role }),
  deleteUser: (id) => J(`/api/admin/users/${id}`, 'DELETE'),

  // Comptes
  accounts: () => G('/api/accounts'),
  createAccount: (a) => J('/api/accounts', 'POST', a),
  deleteAccount: (id) => J(`/api/accounts/${id}`, 'DELETE'),
  addSnapshot: (id, date, balance) => J(`/api/accounts/${id}/snapshots`, 'POST', { date, balance }),

  // Crédits, taxes, dettes
  credits: () => G('/api/credits'),
  createCredit: (c) => J('/api/credits', 'POST', c),
  deleteCredit: (id) => J(`/api/credits/${id}`, 'DELETE'),
  taxes: () => G('/api/taxes'),
  createTax: (t) => J('/api/taxes', 'POST', t),
  setTaxPaid: (id, paid) => J(`/api/taxes/${id}`, 'PATCH', { paid }),
  deleteTax: (id) => J(`/api/taxes/${id}`, 'DELETE'),
  obligations: () => G('/api/obligations'),

  // Domaines & tags
  domains: () => G('/api/domains'),
  createDomain: (name, parentId, color) => J('/api/domains', 'POST', { name, parent_id: parentId ?? null, color }),
  deleteDomain: (id) => J(`/api/domains/${id}`, 'DELETE'),
  updateDomain: (id, patch) => J(`/api/domains/${id}`, 'PATCH', patch),
  tags: () => G('/api/tags'),
  renameTag: (name, newName) => J(`/api/tags/${encodeURIComponent(name)}`, 'PATCH', { name: newName }),
  deleteTag: (name) => J(`/api/tags/${encodeURIComponent(name)}`, 'DELETE'),
  tagFolders: () => G('/api/tag-folders'),
  tagFolderDetail: (name) => G(`/api/tag-folders/${encodeURIComponent(name)}`),
  setTagFolderPref: (name, state) => J(`/api/tag-folders/${encodeURIComponent(name)}/pref`, 'PUT', { state }),
  setTransactionMeta: (id, meta) => J(`/api/transactions/${id}`, 'PATCH', meta),

  // Graphe & simulation
  aiEnrich: (month) => J('/api/enrich/ai', 'POST', month ? { month } : {}),
  categorySeries: (params) => G('/api/category-series?' + new URLSearchParams(params)),
  simulate: (scenario) => J('/api/simulate', 'POST', scenario),
}

export const eur = (v) =>
  new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR' }).format(v)

export const monthLabel = (m) => {
  const [y, mo] = m.split('-')
  const names = ['janv.', 'févr.', 'mars', 'avr.', 'mai', 'juin', 'juil.', 'août', 'sept.', 'oct.', 'nov.', 'déc.']
  return `${names[+mo - 1]} ${y}`
}
