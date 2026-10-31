// Icône sémantique discrète pour les opérations interprétées par le
// normaliseur (versement d'épargne, cotisation, prestation, santé).
export const kindIcon = (k) => ({ savings: '💰', tax: '🏛️', benefit: '🧾', health: '🩺' }[k] || null)
const json = (r) => {
  if (r.status === 204) return null
  return r.json().then((body) => {
    if (!r.ok) throw Object.assign(new Error(body?.error || `Erreur ${r.status}`), { status: r.status })
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

// Analysis requests remain on the existing endpoints and can be cancelled.
const scopedG = (path, params = {}, options = {}) => {
  const query = new URLSearchParams(Object.fromEntries(Object.entries(params).filter(([, value]) => value !== undefined && value !== null && value !== '')))
  return fetch(path + (query.size ? `?${query}` : ''), { credentials: 'same-origin', cache: 'no-store', signal: options.signal }).then(json)
}

export const api = {
  knowledgeOverview: (before = 0, options = {}) => scopedG('/api/knowledge/overview', {before_id:before}, options),
  knowledgeSearch: (query, options = {}) => fetch('/api/knowledge/search', {method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify({query}),signal:options.signal}).then(json),
  fiscalRules: (options = {}) => scopedG('/api/fiscal/rules', {}, options),
  fiscalSimulate: input => J('/api/fiscal/simulate', 'POST', input),
  laboratoryList: (offset = 0, options = {}) => scopedG('/api/laboratory', { offset }, options),
  laboratoryGet: (id, options = {}) => scopedG(`/api/laboratory/${id}`, {}, options),
  laboratorySave: (id, document) => J(id ? `/api/laboratory/${id}` : '/api/laboratory', id ? 'PUT' : 'POST', document),
  laboratoryDelete: (id, revision) => J(`/api/laboratory/${id}`, 'DELETE', { revision }),
  health: () => fetch('/api/health').then(json),
  tagsInCategory: (categoryId, month, options = {}) => scopedG('/api/analytics/tags-in-category', { category: categoryId, month }, options),
  tagSeries: (tags, period = 'month', params = {}, options = {}) => scopedG('/api/analytics/tag-series', { period, ...params, tags: tags?.length ? tags.join(',') : undefined }, options),
  anomalies: (month) => fetch('/api/anomalies' + (month ? `?month=${month}` : '')).then(json),
  journal: (month) => fetch('/api/journal' + (month ? `?month=${month}` : '')).then(json),
  remaining: (month) => fetch('/api/remaining' + (month ? `?month=${month}` : '')).then(json),
  reliability: (month, options = {}) => scopedG('/api/reliability', { month }, options),
  coverage: () => fetch('/api/coverage').then(json),
  intelMerchants: (months = 3, month, options = {}) => scopedG('/api/intel/merchants', { months, month }, options),
  intelNature: (month, options = {}) => scopedG('/api/intel/nature', { month }, options),
  intelComparison: (month, options = {}) => scopedG('/api/intel/comparison', { month }, options),
  intelDiscoveries: (month, options = {}) => scopedG('/api/intel/discoveries', { month }, options),
  intelFeed: (limit = 3, options = {}) => scopedG('/api/intel/feed', { limit }, options),
  intelDiscovery: (id, options = {}) => scopedG(`/api/intel/discoveries/${encodeURIComponent(id)}`, {}, options),
  intelFeedback: (id, action) => J(`/api/intel/discoveries/${encodeURIComponent(id)}/feedback`, 'POST', { action }),
  intelPreferences: (options = {}) => scopedG('/api/intel/preferences', {}, options),
  setIntelPreference: (discoveryId, action, note = '') => J('/api/intel/preferences', 'PATCH', { discovery_id: discoveryId, action, note }),
  clearIntelPreference: discoveryId => J('/api/intel/preferences', 'PATCH', { discovery_id: discoveryId, clear: true }),
  recomputeIntel: month => J('/api/intel/recompute', 'POST', { month }),
  intelJobStatus: (options = {}) => scopedG('/api/intel/jobs/status', {}, options),
  intelNarrative: (month, options = {}) => scopedG('/api/intel/narrative', { month }, options),
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
  period: month => G('/api/period?' + new URLSearchParams({ month })),
  transactionReport: (params = {}, options = {}) => fetch('/api/transactions?' + new URLSearchParams({ ...params, view: 'report' }), { credentials: 'same-origin', signal: options.signal }).then(json),
  categories: (options = {}) => scopedG('/api/categories', {}, options),
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
  statements: (options = {}) => scopedG('/api/statements', {}, options),
  validateStatement: (id, action, note) => J(`/api/statements/${id}/validation`, 'PATCH', { action, note }),
  recheckStatement: (id, mode) => J(`/api/statements/${id}/checks`, 'POST', { mode }),
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
  adminConfig: (options = {}) => scopedG('/api/admin/config', {}, options),
  adminPutConfig: (cfg) =>
    fetch('/api/admin/config', {
      method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(cfg),
    }).then(json),
  adminTestAI: (cfg, options = {}) =>
    fetch('/api/admin/ai/test', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(cfg || {}), signal: options.signal,
    }).then(json),
  adminOllamaModels: (url, options = {}) => scopedG('/api/admin/ollama/models', { url }, options),
  adminTestMemory: (options = {}) => fetch('/api/admin/ai/memory/test', { method: 'POST', credentials: 'same-origin', signal: options.signal }).then(json),
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
  exportTransactionsUrl: (params = {}) => '/api/transactions/export?' + new URLSearchParams({ ...params, view: 'report' }),
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
  updateAccountTracking: (id, tracking_mode) => J(`/api/accounts/${id}`, 'PATCH', { tracking_mode }),
  deleteAccount: (id) => J(`/api/accounts/${id}`, 'DELETE'),
  addSnapshot: (id, date, balance) => J(`/api/accounts/${id}/snapshots`, 'POST', { date, balance }),

  // Crédits, taxes, dettes
  credits: (params = {}, options = {}) => scopedG('/api/credits', params, options),
  createCredit: (c) => J('/api/credits', 'POST', c),
  updateCredit: (id, patch) => J(`/api/credits/${id}`, 'PATCH', patch),
  deleteCredit: (id) => J(`/api/credits/${id}`, 'DELETE'),
  taxes: (params = {}, options = {}) => scopedG('/api/taxes', params, options),
  createTax: (t) => J('/api/taxes', 'POST', t),
  setTaxPaid: (id, paid) => J(`/api/taxes/${id}`, 'PATCH', { paid }),
  deleteTax: (id) => J(`/api/taxes/${id}`, 'DELETE'),
  obligations: (options = {}) => scopedG('/api/obligations', {}, options),
  patchTax: (id, patch) => J(`/api/taxes/${id}`, 'PATCH', patch),
  creditPayment: (id, payment) => J(`/api/credits/${id}/payments`, 'POST', payment),
  voidCreditPayment: (id, paymentId, note) => J(`/api/credits/${id}/payments/${paymentId}`, 'DELETE', { note }),
  engagementHistory: (kind, id, offset = 0, options = {}) => scopedG(`/api/${kind}/${id}/history`, { offset }, options),

  // Manual reimbursement qualification and exact-cent allocation workflows.
  reimbursements: (params = {}, options = {}) => scopedG('/api/reimbursements', params, options),
  reimbursement: (id, options = {}) => scopedG(`/api/reimbursements/${id}`, {}, options),
  createReimbursement: (data) => J('/api/reimbursements', 'POST', data),
  eligibleReimbursementExpenses: (caseId, params = {}, options = {}) => scopedG('/api/reimbursements/eligible-expenses', { case_id: caseId, ...params }, options),
  createReimbursementAllocation: (caseId, allocation, idempotencyKey) => fetch(`/api/reimbursements/${caseId}/allocations`, {
    method: 'POST', credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey },
    body: JSON.stringify(allocation),
  }).then(json),
  voidReimbursementAllocation: (caseId, allocationId, reason = '') => J(`/api/reimbursements/${caseId}/allocations/${allocationId}`, 'DELETE', { reason }),
  voidReimbursement: (caseId, reason = '') => J(`/api/reimbursements/${caseId}/void`, 'POST', { reason }),
  reimbursementHistory: (caseId, params = {}, options = {}) => scopedG(`/api/reimbursements/${caseId}/history`, params, options),

  // Domaines & tags
  domains: () => G('/api/domains'),
  createDomain: (name, parentId, color) => J('/api/domains', 'POST', { name, parent_id: parentId ?? null, color }),
  deleteDomain: (id) => J(`/api/domains/${id}`, 'DELETE'),
  updateDomain: (id, patch) => J(`/api/domains/${id}`, 'PATCH', patch),
  tags: (options = {}) => scopedG('/api/tags', {}, options),
  renameTag: (name, newName) => J(`/api/tags/${encodeURIComponent(name)}`, 'PATCH', { name: newName }),
  deleteTag: (name) => J(`/api/tags/${encodeURIComponent(name)}`, 'DELETE'),
  projects: (month, offset = 0, options = {}) => scopedG('/api/projects', { month, offset }, options),
  project: (id, month, options = {}) => scopedG(`/api/projects/${id}`, { month }, options),
  saveProject: (id, data) => J(id ? `/api/projects/${id}` : '/api/projects', id ? 'PUT' : 'POST', data),
  projectTransactions: (id, params, options = {}) => scopedG(`/api/projects/${id}/transactions`, params, options),
  projectDecide: (id, ids, state) => J(`/api/projects/${id}/decisions`, 'POST', { transaction_ids: ids, state }),
  projectRules: (id, options = {}) => scopedG(`/api/projects/${id}/rules`, {}, options),
  projectRuleSave: (id, rule) => J(`/api/projects/${id}/rules`, 'POST', rule),
  projectRuleRemove: (id, ruleId) => J(`/api/projects/${id}/rules/${ruleId}`, 'DELETE'),
  projectSuggest: id => J(`/api/projects/${id}/suggestions`, 'POST', {}),
  projectHistory: (id, offset = 0, options = {}) => scopedG(`/api/projects/${id}/history`, { offset }, options),
  tagFolders: (month, options = {}) => scopedG('/api/tag-folders', { month }, options),
  tagFolderDetail: (name, month, options = {}) => scopedG(`/api/tag-folders/${encodeURIComponent(name)}`, { month }, options),
  setTagFolderPref: (name, state) => J(`/api/tag-folders/${encodeURIComponent(name)}/pref`, 'PUT', { state }),
  setTransactionMeta: (id, meta) => J(`/api/transactions/${id}`, 'PATCH', meta),

  // Graphe & simulation
  aiEnrich: (month) => J('/api/enrich/ai', 'POST', month ? { month } : {}),
  categorySeries: (params = {}, options = {}) => scopedG('/api/category-series', params, options),
  simulate: (scenario) => J('/api/simulate', 'POST', scenario),
}

export const eur = (v) =>
  new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR' }).format(v)

export const monthLabel = (m) => {
  const [y, mo] = m.split('-')
  const names = ['janv.', 'févr.', 'mars', 'avr.', 'mai', 'juin', 'juil.', 'août', 'sept.', 'oct.', 'nov.', 'déc.']
  return `${names[+mo - 1]} ${y}`
}
