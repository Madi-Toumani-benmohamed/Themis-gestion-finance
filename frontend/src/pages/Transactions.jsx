import { Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { api, monthLabel, kindIcon } from '../api'
import { useToast } from '../Toast'
import SortStack from './SortStack'
import MerchantRenamer from './MerchantRenamer'
import MerchantList from './MerchantList'
import { transactionReportModel, transactionDayTotal, transactionMoney, transactionCents } from '../components/transactionPageModel.mjs'
import { TransactionDrawer, TransactionBulkDialog } from '../components/TransactionDetails'
import './Transactions.css'

export default function Transactions({ initialTag = '', initialType = '', initialMerchant = '', initialSearch = '', initialMonth = '', initialDomain = '', initialBudget = '', initialTransactionIds = [], initialFrom = '', initialTo = '' }) {
  const toast = useToast()
  const [compactTable, setCompactTable] = useState(() => window.matchMedia('(max-width: 760px)').matches)
  useEffect(() => {
    const media = window.matchMedia('(max-width: 760px)')
    const update = () => setCompactTable(media.matches)
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])
  const [txs, setTxs] = useState([])
  const [month, setMonth] = useState(initialMonth || (initialTag || initialType || initialMerchant || initialSearch || initialDomain || initialBudget || initialTransactionIds.length || initialFrom || initialTo ? '' : new Date().toISOString().slice(0, 7)))
  const [from, setFrom] = useState(initialFrom)
  const [to, setTo] = useState(initialTo)
  const [showDates, setShowDates] = useState(Boolean(initialFrom || initialTo))
  const [domainFilter, setDomainFilter] = useState(initialDomain)
  const [sourceFilter, setSourceFilter] = useState({ budget: initialBudget, ids: initialTransactionIds.slice().reverse() })
  const [sourcePage, setSourcePage] = useState(0)
  const [page, setPage] = useState(0), [report, setReport] = useState(null)
  const [accountFilter, setAccountFilter] = useState(''), [accounts, setAccounts] = useState([]), [typeFilter, setTypeFilter] = useState(initialType)
  const [reliability, setReliability] = useState(null), [selected, setSelected] = useState(new Set()), [detail, setDetail] = useState(null), [bulk, setBulk] = useState(null)
  const [showTools, setShowTools] = useState(false)
  const abortRequest = useRef(null), periodTouched = useRef(false)
  const [fetching, setFetching] = useState(false)
  const requestEpoch = useRef(0)
  const [search, setSearch] = useState(initialSearch)
  const [months, setMonths] = useState([])
  const [err, setErr] = useState(null)
  const [aiEnabled, setAiEnabled] = useState(false)
  const [enrichBusy, setEnrichBusy] = useState(false)
  const [applyBusy, setApplyBusy] = useState(false)
  const [domains, setDomains] = useState([])
  const [tagFilter, setTagFilter] = useState(initialTag)
  const [merchantFilter, setMerchantFilter] = useState(initialMerchant)
  const [allTags, setAllTags] = useState([])
  const [manageDomains, setManageDomains] = useState(false)
  const [showAllDomains, setShowAllDomains] = useState(false)
  const [showRenamer, setShowRenamer] = useState(false)
  const [renamerView, setRenamerView] = useState('todo')
  const [newDomain, setNewDomain] = useState('')
  const [sorting, setSorting] = useState(false)
  const [showRules, setShowRules] = useState(false)
  const [rules, setRules] = useState([])
  const [editingPattern, setEditingPattern] = useState(null) // id de la règle dont on édite le motif

  useEffect(() => {
    api.health().then((h) => setAiEnabled(h.ai_enabled)).catch(() => {})
    api.domains().then((x) => setDomains(Array.isArray(x) ? x : [])).catch((e) => setErr(e.message))
    api.tags().then(setAllTags).catch(() => {})
    api.accounts().then(x => setAccounts(Array.isArray(x) ? x : [])).catch(e => setErr(e.message))
    api.summary(24).then((d) => {
      const ms = (d.monthly ?? []).map((m) => m.month).reverse()
      setMonths(ms)
      // On ne cale sur le mois le plus récent QUE si l'utilisateur n'arrive pas
      // avec une recherche ou un mois déjà imposé (ex. clic sur une anomalie qui
      // concerne un autre mois). Sinon on garde « tous les mois » pour ne pas
      // masquer les transactions ciblées.
      if (ms[0] && !periodTouched.current && !initialTag && !initialType && !initialMerchant && !initialSearch && !initialMonth && !initialDomain && !initialBudget && !initialTransactionIds.length && !initialFrom && !initialTo) setMonth(ms[0])
    }).catch((e) => setErr(e.message))
  }, [])

  const queryParams = () => {
    const params = {}
    if (from || to) { params.from = from; params.to = to }
    else if (month) params.month = month
    else params.all = 1
    if (sourceFilter.budget) params.budget_id = sourceFilter.budget
    if (sourceFilter.ids.length) params.ids = sourceFilter.ids.slice(sourcePage * 1000, (sourcePage + 1) * 1000).join(',')
    if (domainFilter) params.domain = domainFilter
    if (accountFilter) params.account = accountFilter
    if (merchantFilter) params.merchant = merchantFilter
    if (typeFilter) params.type = typeFilter
    if (search) params.q = search
    if (tagFilter) params.tag = tagFilter
    return params
  }
  const reload = async () => {
    abortRequest.current?.abort()
    const controller = new AbortController()
    abortRequest.current = controller
    const epoch = ++requestEpoch.current
    const params = { ...queryParams(), limit: 100, offset: page * 100 }
    setFetching(true); setErr(null); setReport(null); setTxs([])
    if (showDates && (!from || !to || from > to)) {
      setErr('Choisis une date de début et de fin dans l’ordre chronologique.'); setFetching(false); return
    }
    try {
      const data = transactionReportModel(await api.transactionReport(params, { signal: controller.signal }))
      if (epoch !== requestEpoch.current) return
      if (page > 0 && page * 100 >= data.total) { setPage(Math.max(0, Math.ceil(data.total / 100) - 1)); return }
      setReport(data); setTxs(data.transactions); setDetail(current => current ? data.transactions.find(row => row.id === current.id) || current : null)
    } catch (error) { if (epoch === requestEpoch.current && error.name !== 'AbortError') setErr(error.message) }
    finally { if (epoch === requestEpoch.current) setFetching(false) }
  }

  useEffect(() => { setPage(0); setSelected(new Set()) }, [month, merchantFilter, domainFilter, accountFilter, typeFilter, search, tagFilter, from, to, sourceFilter, sourcePage])
  useEffect(() => { setSelected(new Set()) }, [page])
  useEffect(() => {
    const timer = setTimeout(reload, search ? 250 : 0)
    return () => { clearTimeout(timer); requestEpoch.current++; abortRequest.current?.abort() }
  }, [month, merchantFilter, domainFilter, accountFilter, typeFilter, search, tagFilter, from, to, sourceFilter, sourcePage, page])
  useEffect(() => {
    let active = true
    setReliability(null)
    if (month && !from && !to) api.period(month).then(data => { if (active) setReliability(data) }).catch(() => {})
    return () => { active = false }
  }, [month, from, to])

  const resetFilters = () => {
    periodTouched.current = true
    setMonth(months[0] || new Date().toISOString().slice(0, 7)); setFrom(''); setTo(''); setShowDates(false)
    setSearch(''); setDomainFilter(''); setAccountFilter(''); setTypeFilter(''); setTagFilter(''); setMerchantFilter(''); setSourceFilter({ budget: '', ids: [] }); setSourcePage(0); setPage(0)
  }
  const saveTransaction = async (tx, patch) => {
    await api.setTransactionMeta(tx.id, patch)
    setDetail(current => current?.id === tx.id ? { ...current, ...patch, classification_source: Object.hasOwn(patch, 'domain_id') ? 'manual' : current.classification_source, financial_type: Object.hasOwn(patch, 'domain_id') ? undefined : current.financial_type, suggestion: Object.hasOwn(patch, 'domain_id') || patch.discard_suggestion ? null : current.suggestion } : current)
    await reload()
    api.tags().then(setAllTags).catch(() => {})
  }
  const applyBulk = async (rows, change) => {
    const failed = []; let succeeded = 0
    for (const tx of rows) {
      const patch = change.action === 'category' ? { domain_id: change.domain_id } : { [change.action]: change.tags }
      try { await api.setTransactionMeta(tx.id, patch); succeeded++ }
      catch (error) { failed.push({ id: tx.id, error: error.message }) }
    }
    setSelected(new Set(failed.map(item => item.id)))
    await reload(); api.tags().then(setAllTags).catch(() => {})
    return { succeeded, failed }
  }
  const topDomains = useMemo(() => domains.filter((d) => !d.parent_id), [domains])
  const domainById = useMemo(() => Object.fromEntries(domains.map((d) => [d.id, d])), [domains])

  // motif suggéré : le mot le plus distinctif du libellé (le plus long,
  // hors préfixes bancaires et nombres) — bien plus fiable que "le 1er mot".
  const suggestPattern = (label) => {
    const stop = new Set(['CB', 'PRLV', 'VIR', 'VIREMENT', 'INST', 'DE', 'DU', 'LA', 'LE', 'CARTE', 'PAIEMENT', 'ACHAT'])
    const words = label.toUpperCase().replace(/[0-9]/g, ' ').split(/[^A-ZÀ-Ÿ]+/).filter(Boolean)
    const candidates = words.filter((w) => w.length >= 3 && !stop.has(w))
    return (candidates.sort((a, b) => b.length - a.length)[0] || words[0] || label).trim()
  }

  const proposeRule = (tx, domainId) => {
    const domName = domainById[domainId]?.name || 'ce domaine'
    toast.action(
      `Ranger automatiquement les transactions « ${suggestPattern(tx.label)} » dans ${domName} ?`,
      'Créer la règle',
      () => createRule(suggestPattern(tx.label), domainId),
    )
  }

  const changeDomain = async (tx, value) => {
    const domainId = value === '' ? null : +value
    try {
      await api.setTransactionMeta(tx.id, { domain_id: domainId })
      await reload()
      if (domainId) proposeRule(tx, domainId)
    } catch (e) { toast.err(e.message) }
  }

  const runAiEnrich = async () => {
    setEnrichBusy(true)
    try {
      const r = await api.aiEnrich(month || undefined)
      toast.ok(r.suggested > 0 ? `${r.suggested} proposition(s) IA à vérifier dans le détail des opérations. Aucune catégorie ni aucun tag appliqué automatiquement.` : 'Aucune nouvelle proposition IA.')
      reload()
      api.tags().then(setAllTags).catch(() => {})
    } catch (e) { toast.err(e.message) } finally { setEnrichBusy(false) }
  }

  const PALETTE = ['#175C46', '#3D6B99', '#8C6D2F', '#A63A28', '#6E4E8C', '#4E8578', '#96652F', '#5E7052']
  const addDomain = async (e) => {
    e.preventDefault()
    const name = newDomain.trim()
    if (!name) return
    try {
      const color = PALETTE[domains.length % PALETTE.length]
      const d = await api.createDomain(name, null, color)
      setDomains((list) => [...list, d])
      setNewDomain('')
      toast.ok(`Catégorie « ${name} » créé.`)
    } catch (e2) { toast.err(e2.message) }
  }
  const removeDomain = async (d) => {
    if (!confirm(`Retirer la catégorie « ${d.name} » ? Les transactions concernées la perdront.`)) return
    try {
      await api.deleteDomain(d.id)
      setDomains((list) => list.filter((x) => x.id !== d.id))
      if (String(domainFilter) === String(d.id)) setDomainFilter('')
      reload()
      toast.ok('Catégorie retirée.')
    } catch (e2) { toast.err(e2.message) }
  }
  const changeDomainColor = async (d, color) => {
    // maj optimiste locale, puis persistance
    setDomains((list) => list.map((x) => (x.id === d.id ? { ...x, color } : x)))
    try {
      await api.updateDomain(d.id, { color })
    } catch (e2) { setDomains(list => list.map(x => x.id === d.id ? { ...x, color: d.color } : x)); toast.err(e2.message) }
  }

  const createRule = async (pattern, domainId) => {
    if (!pattern) return
    try {
      const r = await api.createRule(pattern, domainId, true)
      const verb = r.replaced ? 'Règle mise à jour' : 'Règle créée'
      toast.ok(r.applied > 0
        ? `${verb} · ${r.applied} transaction(s) rangée(s) automatiquement.`
        : `${verb}.`)
      reload()
      loadRules()
    } catch (e) { toast.err(e.message) }
  }

  const loadRules = () => api.rules().then(setRules).catch(() => {})

  // Édition inline : on clique directement sur un champ. Le motif s'édite au
  // clic (état editingPattern = id de la règle), la catégorie via son menu.
  // Chaque changement est sauvegardé immédiatement.
  const saveRulePattern = async (r, newPattern) => {
    setEditingPattern(null)
    const pattern = (newPattern || '').trim()
    if (!pattern || pattern === r.pattern) return
    try {
      const res = await api.updateRule(r.id, pattern, r.domain_id, true, r.tags || [])
      toast.ok(res.applied > 0 ? `Règle modifiée · ${res.applied} transaction(s) rangée(s).` : 'Règle modifiée.')
      loadRules(); reload()
    } catch (e) { toast.err(e.message) }
  }
  const saveRuleDomain = async (r, newDomainId) => {
    const domainId = +newDomainId
    if (!domainId || domainId === r.domain_id) return
    try {
      const res = await api.updateRule(r.id, r.pattern, domainId, true, r.tags || [])
      toast.ok(res.applied > 0 ? `Règle modifiée · ${res.applied} transaction(s) rangée(s).` : 'Règle modifiée.')
      loadRules(); reload()
    } catch (e) { toast.err(e.message) }
  }
  // Tags de règle : saisis en texte « a, b, c », sauvés au blur. On ne
  // sauvegarde que si la liste a changé (évite les appels inutiles).
  const saveRuleTags = async (r, raw) => {
    const tags = (raw || '').split(',').map((t) => t.trim()).filter(Boolean)
    const before = (r.tags || []).join('|').toLowerCase()
    const after = tags.join('|').toLowerCase()
    if (before === after) return
    try {
      const res = await api.updateRule(r.id, r.pattern, r.domain_id, true, tags)
      toast.ok(res.applied > 0 ? `Tags appliqués à ${res.applied} transaction(s).` : 'Tags de la règle mis à jour.')
      loadRules(); reload()
    } catch (e) { toast.err(e.message) }
  }
  const removeRule = async (r) => {
    if (!confirm(`Supprimer la règle « ${r.pattern} » → ${r.domain_name} ? Les transactions déjà rangées ne changent pas.`)) return
    try {
      await api.deleteRule(r.id)
      setRules((list) => list.filter((x) => x.id !== r.id))
      toast.ok('Règle supprimée.')
    } catch (e) { toast.err(e.message) }
  }

  const toggleRules = () => {
    const next = !showRules
    setShowRules(next)
    if (next) loadRules()
  }

  const applyRules = async () => {
    setApplyBusy(true)
    try {
      const r = await api.applyRules()
      toast.ok(r.applied > 0
        ? `${r.applied} transaction(s) rangée(s) par les règles.`
        : 'Aucune transaction à ranger : les règles sont déjà à jour.')
      reload()
    } catch (e) { toast.err(e.message) } finally { setApplyBusy(false) }
  }

  // Regroupement par jour : une ligne d'en-tête par journée, avec son solde net.
  // Les transactions arrivent déjà triées par date décroissante.
  const byDay = txs.reduce((acc, t) => {
    (acc[t.date] ??= []).push(t)
    return acc
  }, {})
  const dayLabel = (d) =>
    new Date(`${d}T12:00:00Z`).toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long', timeZone: 'UTC' })

  return (
    <div className="section transactions-page">
      <header className="transactions-heading"><div><span className="eyebrow">Gestion des opérations</span><h1>Transactions</h1><p>{from || to ? `${from || '…'} — ${to || '…'}` : month ? monthLabel(month) : 'Tous les mois'} · mouvements observés</p></div><button className="btn-secondary" onClick={() => setShowTools(value => !value)} aria-expanded={showTools} aria-controls="transaction-tools">{showTools ? 'Fermer les outils' : 'Outils de classement'}</button></header>
      <div className="transactions-period"><span className="transaction-status">{reliability?.completeness === 'complete' ? 'Période complète documentée' : reliability?.completeness === 'partial' ? 'Période partielle' : month === new Date().toISOString().slice(0, 7) && !from && !to ? 'Mois en cours · couverture inconnue' : 'Complétude non établie'}</span><span>Les indicateurs portent sur toutes les opérations correspondant aux filtres, au-delà de cette page.</span></div>
      <dl className="transactions-summary" aria-label="Synthèse des opérations filtrées">{[['Dépenses observées', 'expenses'], ['Revenus et crédits', 'income'], ['Flux net hors transferts', 'net']].map(([label, key]) => <div key={key}><dt>{label}</dt><dd>{report ? transactionMoney(report.money[key]) : '—'}</dd></div>)}</dl>
      <p className="transaction-note">Débits et crédits observés, transferts et épargne exclus. Les crédits peuvent inclure des remboursements ou annulations non rapprochés. Un solde temporaire ne préjuge pas de la situation du mois ; les revenus peuvent arriver en fin de mois.</p>
      {report && (report.money.transfer_debits > 0 || report.money.transfer_credits > 0) && <p className="transaction-note">Mouvements de transfert et d’épargne exclus de ces indicateurs : sorties {transactionMoney(report.money.transfer_debits)} · entrées {transactionMoney(report.money.transfer_credits)}. Ils restent consultables dans la liste.</p>}
      {report && <p className="transaction-query-period">Périmètre interrogé : {report.query_from || 'début de l’historique'} → {report.query_to || 'fin de l’historique'}.{accountFilter && ' La couverture documentaire indiquée reste celle de l’ensemble des comptes.'}</p>}

      {sorting && (
        <SortStack
          transactions={txs.filter((t) => !t.domain_id)}
          domains={domains}
          onClose={() => { setSorting(false); reload() }}
          onSorted={(id, domainId) => setTxs((list) => list.map((x) => x.id === id ? { ...x, domain_id: domainId } : x))}
        />
      )}
      <div id="transaction-tools" className="transaction-tools" hidden={!showTools}>
      <div className="domains-bar">
        <span className="domains-label">Catégories</span>
        {(showAllDomains || manageDomains ? topDomains : topDomains.slice(0, 4)).map((d) => (
          <span key={d.id} className="domain-chip" style={{ '--dc': d.color }}>
            <span className="dom-dot" />{d.name}
            {manageDomains && (
              <label className="domain-color" title="Changer la couleur">
                <input type="color" value={d.color || '#8A8A99'} aria-label={`Couleur de ${d.name}`}
                       onChange={(e) => changeDomainColor(d, e.target.value)} />
              </label>
            )}
            {manageDomains && <button className="domain-x" onClick={() => removeDomain(d)} title="Retirer" aria-label={`Retirer ${d.name}`}>×</button>}
          </span>
        ))}
        {!manageDomains && !showAllDomains && topDomains.length > 4 && (
          <button className="link domains-more" onClick={() => setShowAllDomains(true)}>
            + {topDomains.length - 4} autres
          </button>
        )}
        {!manageDomains && showAllDomains && topDomains.length > 4 && (
          <button className="link domains-more" onClick={() => setShowAllDomains(false)}>
            réduire
          </button>
        )}
        {manageDomains && (
          <form className="domain-add" onSubmit={addDomain}>
            <input value={newDomain} onChange={(e) => setNewDomain(e.target.value)} placeholder="nouvelle catégorie" aria-label="Nom de la nouvelle catégorie" />
            <button type="submit" title="Ajouter">＋</button>
          </form>
        )}
        <button className="link" onClick={() => setManageDomains((v) => !v)}>
          {manageDomains ? 'terminer' : 'gérer'}
        </button>
        <button className="link" onClick={toggleRules} style={{ marginLeft: 'auto' }}>
          {showRules ? 'masquer les règles' : 'règles automatiques'}
        </button>
        <button className="link" onClick={() => setShowRenamer((v) => !v)}>
          {showRenamer ? 'masquer les noms' : 'noms des marchands'}
        </button>
      </div>

      {showRules && (
        <div className="rules-panel">
          <div className="rules-head">
            <b>Règles automatiques</b>
            <span className="muted">Chaque relevé importé est rangé selon ces motifs. Le nombre indique les transactions dont le libellé correspond.</span>
          </div>
          {rules.length === 0 ? (
            <p className="muted" style={{ fontSize: 13, margin: 0 }}>
              Aucune règle pour l'instant. Range une transaction dans une catégorie : l'app te proposera d'en créer une.
            </p>
          ) : (
            <table className="rules-table">
              <thead>
                <tr><th>Motif (contenu dans le libellé)</th><th>Catégorie</th><th>Tags automatiques</th><th className="num">Transactions</th><th></th></tr>
              </thead>
              <tbody>
                {rules.map((r) => (
                  <tr key={r.id}>
                    {/* Motif : cliquer dessus pour l'éditer (sauvegarde au blur / Entrée) */}
                    <td>
                      {editingPattern === r.id ? (
                        <input
                          className="rule-input" aria-label="Motif de la règle"
                          defaultValue={r.pattern}
                          autoFocus
                          onBlur={(e) => saveRulePattern(r, e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') e.target.blur()
                            if (e.key === 'Escape') { e.target.value = r.pattern; setEditingPattern(null) }
                          }}
                        />
                      ) : (
                        <button className="rule-pattern editable" onClick={() => setEditingPattern(r.id)}
                                title="Cliquer pour modifier le motif">
                          {r.pattern}
                        </button>
                      )}
                    </td>
                    {/* Catégorie : menu déroulant directement modifiable */}
                    <td>
                      <span className="dom-pick set" style={{ '--dc': r.color }}>
                        <span className="dom-dot" />
                        <select className="dom-select" value={r.domain_id}
                                onChange={(e) => saveRuleDomain(r, e.target.value)}
                                title="Changer la catégorie" aria-label={`Catégorie de la règle ${r.pattern}`}>
                          {topDomains.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
                        </select>
                      </span>
                    </td>
                    {/* Tags automatiques : saisis « a, b, c », appliqués au blur */}
                    <td>
                      <input
                        className="rule-tags-input"
                        defaultValue={(r.tags || []).join(', ')}
                        placeholder="+ tags…" aria-label={`Tags de la règle ${r.pattern}`}
                        title="Tags posés automatiquement, séparés par des virgules"
                        onBlur={(e) => saveRuleTags(r, e.target.value)}
                        onKeyDown={(e) => { if (e.key === 'Enter') e.target.blur() }}
                      />
                    </td>
                    <td className="num muted">{r.matches}</td>
                    <td className="rule-actions">
                      <button className="rule-del" onClick={() => removeRule(r)} title="Supprimer la règle" aria-label="Supprimer la règle">
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M3 6h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6" /></svg>
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {showRenamer && (
        <div className="merchant-panel">
          <div className="merchant-tabs">
            <button
              className={`merchant-tab ${renamerView === 'todo' ? 'active' : ''}`}
              onClick={() => setRenamerView('todo')}
            >À nommer</button>
            <button
              className={`merchant-tab ${renamerView === 'named' ? 'active' : ''}`}
              onClick={() => setRenamerView('named')}
            >Déjà nommés</button>
          </div>
          {renamerView === 'todo'
            ? <MerchantRenamer aiEnabled={aiEnabled} onRenamed={reload} />
            : <MerchantList />}
        </div>
      )}

      </div>
      {merchantFilter && <p className="transaction-source">Marchand exact : {merchantFilter} <button className="link" onClick={() => setMerchantFilter('')}>Retirer ce filtre</button></p>}
      {(sourceFilter.budget || sourceFilter.ids.length > 0) && <div className="notice">Opérations contribuant {sourceFilter.budget ? 'à l’enveloppe sélectionnée' : 'à la récurrence sélectionnée'}. <button className="link" onClick={() => { setSourceFilter({ budget: '', ids: [] }); setSourcePage(0) }}>Retirer le filtre de provenance</button></div>}
      {sourceFilter.ids.length > 1000 && <nav className="transaction-source-nav" aria-label="Lots de provenance"><button className="btn-secondary" disabled={fetching || sourcePage === 0} onClick={() => { setSourcePage(value => value - 1); setPage(0) }}>Lot précédent</button><span>Lot de sources {sourcePage + 1} sur {Math.ceil(sourceFilter.ids.length / 1000)}. Synthèse limitée aux 1 000 identifiants de ce lot et aux filtres actifs ; elle ne représente pas toutes les sources.</span><button className="btn-secondary" disabled={fetching || (sourcePage + 1) * 1000 >= sourceFilter.ids.length} onClick={() => { setSourcePage(value => value + 1); setPage(0) }}>Lot suivant</button></nav>}
      <div className="toolbar transaction-filters" aria-label="Filtres des transactions">
        {!showDates ? (
          <select value={month} onChange={(e) => { periodTouched.current = true; setMonth(e.target.value) }} aria-label="Mois">
            <option value="">Tous les mois</option>
            {[...new Set([month, ...months].filter(Boolean))].sort().reverse().map((m) => <option key={m} value={m}>{monthLabel(m)}</option>)}
          </select>
        ) : (
          <span className="date-range">
            <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} aria-label="Du" />
            <span className="date-range-sep">au</span>
            <input type="date" value={to} onChange={(e) => setTo(e.target.value)} aria-label="Au" />
          </span>
        )}
        <button
          className={`btn-period ${showDates ? 'active' : ''}`}
          onClick={() => { periodTouched.current = true; setShowDates((v) => !v); if (showDates) { setFrom(''); setTo('') } else { setMonth('') } }}
          title="Basculer entre mois et période personnalisée"
        >
          {showDates ? '← par mois' : 'par période'}
        </button>
        <select value={domainFilter} onChange={(e) => setDomainFilter(e.target.value)} aria-label="Catégorie">
          <option value="">Toutes les catégories</option>
          <option value="none">Sans catégorie</option>
          {domains.map((d) => <option key={d.id} value={d.id}>{d.parent_id ? '↳ ' : ''}{d.name}</option>)}
        </select>
        <input
          type="search"
          placeholder="Marchand, libellé, montant…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Recherche"
          style={{ flex: 1, minWidth: 160 }}
        />
        <select value={accountFilter} onChange={event => setAccountFilter(event.target.value)} aria-label="Compte bancaire"><option value="">Tous les comptes</option><option value="none">Compte non documenté</option>{accounts.map(account => <option key={account.id} value={account.id}>{account.name}</option>)}</select>
        <select value={typeFilter} onChange={event => setTypeFilter(event.target.value)} aria-label="Type d’opération"><option value="">Tous les types</option><option value="debit">Débits hors transferts</option><option value="credit">Crédits hors transferts</option><option value="transfer">Transferts et épargne</option><option value="neutral">Montants nuls</option></select>
        {allTags.length > 0 && (
          <select value={tagFilter} onChange={(e) => setTagFilter(e.target.value)} aria-label="Tag">
            <option value="">Tous les tags</option>
            <option value="__none__">— Sans tag —</option>
            {allTags.map((t) => <option key={t.name} value={t.name}>#{t.name}</option>)}
          </select>
        )}
        <button className="btn-secondary" onClick={resetFilters}>Réinitialiser les filtres</button>
        {report && !fetching ? <a className="btn-secondary" href={api.exportTransactionsUrl(queryParams())} download>Exporter ces résultats (CSV)</a> : <button className="btn-secondary" disabled>Exporter ces résultats (CSV)</button>}
        {txs.some((t) => !t.domain_id) && (
          <button className="btn-secondary" onClick={() => { if (confirm('Appliquer les règles déterministes existantes à toutes les opérations non catégorisées de l’historique, indépendamment des filtres ?')) applyRules() }} disabled={applyBusy}
                  title="Ranger les transactions sans catégorie selon les règles existantes">
            {applyBusy ? 'Application…' : 'Appliquer les règles'}
          </button>
        )}
        {txs.some((t) => !t.domain_id) && (
          <button className="btn" onClick={() => setSorting(true)}>
            Trier en pile ({txs.filter((t) => !t.domain_id).length})
          </button>
        )}
        {aiEnabled && txs.some((t) => !t.domain_id) && (
          <button className="btn-secondary ai" onClick={runAiEnrich} disabled={enrichBusy || !month || Boolean(from || to)} title="Propositions pour le mois sélectionné, à vérifier et appliquer individuellement. Les autres filtres ne limitent pas la recherche IA.">
            {enrichBusy ? 'Suggestions en cours…' : 'Suggérer catégories et tags (IA)'}
          </button>
        )}
      </div>

      <div className="transaction-list-heading"><p aria-live="polite">{report ? `${report.total} opération${report.total > 1 ? 's' : ''} correspondant aux filtres · ${txs.length} sur cette page` : 'Nombre d’opérations indisponible'}</p>{selected.size > 0 && <div className="transaction-selection"><strong>{selected.size} sélectionnée(s)</strong><button className="btn-secondary" disabled={!txs.some(tx => selected.has(tx.id))} onClick={() => setBulk(txs.filter(tx => selected.has(tx.id)))}>Modifier la sélection</button><button className="link" onClick={() => setSelected(new Set())}>Désélectionner</button></div>}</div>
      {err && <div className="notice err" role="alert">{err} <button className="link" onClick={reload}>Réessayer</button></div>}
      {fetching && <p role="status" className="notice">Chargement des transactions…</p>}
      {!fetching && !err && txs.length === 0 ? <div className="empty">Aucune transaction pour ces filtres.</div> : txs.length > 0 && <div className="transactions-table-wrap"><table className="transactions-table"><caption className="sr-only">Transactions regroupées par date, selon les filtres actifs</caption>
        <thead><tr><th className="transaction-check"><input type="checkbox" aria-label="Sélectionner toutes les opérations de cette page" checked={txs.length > 0 && txs.every(tx => selected.has(tx.id))} onChange={event => setSelected(event.target.checked ? new Set(txs.map(tx => tx.id)) : new Set())}/></th><th>Libellé et compte</th><th className="transaction-category-column">Catégorie</th><th className="transaction-tags-column">Tags</th><th className="num">Montant</th></tr></thead>
        <tbody>{Object.entries(byDay).map(([date, rows]) => <Fragment key={date}><tr className="tx-day"><td colSpan={compactTable ? 3 : 5}><time dateTime={date}>{dayLabel(date)}</time><span className="tx-day-total" title="Somme signée des écritures de ce jour présentes sur cette page, transferts inclus">{transactionMoney(transactionDayTotal(rows))} <small>sur cette page</small></span></td></tr>{rows.map(tx => <tr key={tx.id} className={selected.has(tx.id) ? 'transaction-selected' : ''}>
          <td className="transaction-check"><input type="checkbox" aria-label={`Sélectionner l’opération ${tx.id}`} checked={selected.has(tx.id)} onChange={event => setSelected(current => { const next = new Set(current); event.target.checked ? next.add(tx.id) : next.delete(tx.id); return next })}/></td>
          <td className="transaction-description"><button id={`transaction-open-${tx.id}`} className="transaction-open" onClick={() => setDetail(tx)} aria-label={`Détail de ${tx.display_label || tx.merchant || tx.label}`}><span className="tx-label">{kindIcon(tx.kind) && <span aria-hidden="true">{kindIcon(tx.kind)} </span>}{tx.display_label || tx.merchant || tx.label}</span></button><div className="tx-raw">{tx.label}</div><div className="transaction-row-meta"><span>{tx.account_name || 'Compte non documenté'}</span>{tx.financial_type === 'transfer' && <span className="transaction-status">Transfert / épargne</span>}{!tx.domain_id && <span>Sans catégorie</span>}{tx.suggestion && <span>Suggestion IA à vérifier</span>}</div><div className="transaction-mobile-meta"><span className="transaction-category-label" style={{ '--dc': domainById[tx.domain_id]?.color || 'var(--faint)' }}><span className="dom-dot" aria-hidden="true"/>{domainById[tx.domain_id]?.name || 'Sans catégorie'}</span>{tx.tags?.length ? ` · ${tx.tags.map(tag => `#${tag}`).join(' ')}` : ''}</div></td>
          <td className="transaction-category-column"><span className={'dom-pick transaction-category-picker' + (domainById[tx.domain_id] ? ' set' : '')} style={{ '--dc': domainById[tx.domain_id]?.color || 'var(--faint)' }}><span className="dom-dot" aria-hidden="true"/><select aria-label={`Catégorie de l’opération ${tx.id}`} value={tx.domain_id ?? ''} onChange={event => changeDomain(tx, event.target.value)}><option value="">Sans catégorie</option>{domains.map(domain => <option key={domain.id} value={domain.id}>{domain.parent_id ? '↳ ' : ''}{domain.name}</option>)}</select></span></td>
          <td className="transaction-tags-column"><button className="transaction-tags" onClick={() => setDetail(tx)} aria-label={`Modifier les tags de l’opération ${tx.id} : ${tx.tags?.length ? tx.tags.join(' ') : 'Ajouter des tags'}`}>{tx.tags?.length ? tx.tags.map(tag => <Fragment key={tag}><span className="tag-pill">{tag}</span>{' '}</Fragment>) : <span>Ajouter des tags</span>}</button></td>
          <td className={'num amt' + (transactionCents(tx) > 0 ? ' credit' : '')}>{transactionCents(tx) > 0 ? '+' : ''}{transactionMoney(transactionCents(tx))}</td>
        </tr>)}</Fragment>)}</tbody></table></div>}
      {report && <nav className="transaction-pagination" aria-label="Pagination des transactions"><button className="btn-secondary" disabled={fetching || page === 0} onClick={() => setPage(value => value - 1)}>Précédent</button><span>Page {page + 1} sur {Math.max(1, Math.ceil(report.total / 100))} · 100 opérations maximum</span><button className="btn-secondary" disabled={fetching || (page + 1) * 100 >= report.total} onClick={() => setPage(value => value + 1)}>Suivant</button></nav>}
      {detail && <TransactionDrawer key={detail.id} transaction={detail} domains={domains} allTags={allTags} onSave={saveTransaction} onClose={() => setDetail(null)}/>}
      {bulk && <TransactionBulkDialog rows={bulk} domains={domains} onApply={applyBulk} onClose={() => setBulk(null)}/>}
    </div>
  )
}
