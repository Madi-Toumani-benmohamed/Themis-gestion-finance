import { Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { api, eur, monthLabel, kindIcon } from '../api'
import { useToast } from '../Toast'
import SortStack from './SortStack'
import MerchantRenamer from './MerchantRenamer'
import MerchantList from './MerchantList'

export default function Transactions({ initialSearch = '', initialMonth = '', initialDomain = '', initialBudget = '', initialTransactionIds = [], initialFrom = '', initialTo = '' }) {
  const toast = useToast()
  const [txs, setTxs] = useState([])
  const [month, setMonth] = useState(initialMonth)
  const [from, setFrom] = useState(initialFrom)
  const [to, setTo] = useState(initialTo)
  const [showDates, setShowDates] = useState(Boolean(initialFrom || initialTo))
  const [domainFilter, setDomainFilter] = useState(initialDomain)
  const [sourceFilter, setSourceFilter] = useState({ budget: initialBudget, ids: initialTransactionIds.slice().reverse() })
  const [sourcePage, setSourcePage] = useState(0)
  const [fetching, setFetching] = useState(false)
  const requestEpoch = useRef(0)
  const [search, setSearch] = useState(initialSearch)
  const [months, setMonths] = useState([])
  const [err, setErr] = useState(null)
  const [aiEnabled, setAiEnabled] = useState(false)
  const [enrichBusy, setEnrichBusy] = useState(false)
  const [applyBusy, setApplyBusy] = useState(false)
  const [domains, setDomains] = useState([])
  const [tagFilter, setTagFilter] = useState('')
  const [allTags, setAllTags] = useState([])
  const [tagEdit, setTagEdit] = useState(null)
  const [tagDraft, setTagDraft] = useState('')
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
    api.summary(24).then((d) => {
      const ms = (d.monthly ?? []).map((m) => m.month).reverse()
      setMonths(ms)
      // On ne cale sur le mois le plus récent QUE si l'utilisateur n'arrive pas
      // avec une recherche ou un mois déjà imposé (ex. clic sur une anomalie qui
      // concerne un autre mois). Sinon on garde « tous les mois » pour ne pas
      // masquer les transactions ciblées.
      if (ms[0] && !initialSearch && !initialMonth && !initialDomain && !initialBudget && !initialTransactionIds.length && !initialFrom && !initialTo) setMonth(ms[0])
    }).catch((e) => setErr(e.message))
  }, [])

  const reload = () => {
    const params = {}
    // Une période explicite (du/au) prime sur le sélecteur de mois.
    if (from || to) {
      if (from) params.from = from
      if (to) params.to = to
    } else if (month) {
      params.month = month
    }
    if (sourceFilter.budget) { params.budget_id = sourceFilter.budget; params.offset = sourcePage * 1000 }
    if (sourceFilter.ids.length) params.ids = sourceFilter.ids.slice(sourcePage * 1000, (sourcePage + 1) * 1000).join(',')
    if (domainFilter) params.domain = domainFilter
    if (search) params.q = search
    if (tagFilter) params.tag = tagFilter
    const epoch = ++requestEpoch.current
    setFetching(true); setErr(null); setTxs([])
    api.transactions(params).then(x => { if (epoch === requestEpoch.current) setTxs(Array.isArray(x) ? x : []) }).catch(e => { if (epoch === requestEpoch.current) setErr(e.message) }).finally(() => { if (epoch === requestEpoch.current) setFetching(false) })
  }

  useEffect(() => { setSourcePage(0) }, [month, domainFilter, search, tagFilter, from, to, sourceFilter])
  useEffect(() => {
    const t = setTimeout(reload, search ? 250 : 0)
    return () => { clearTimeout(t); requestEpoch.current++ }
  }, [month, domainFilter, search, tagFilter, from, to, sourceFilter, sourcePage])

  const topDomains = useMemo(() => domains.filter((d) => !d.parent_id), [domains])
  const domainById = useMemo(() => Object.fromEntries(domains.map((d) => [d.id, d])), [domains])

  const tagSuggestions = useMemo(() => {
    if (tagEdit === null) return []
    const parts = tagDraft.split(',')
    const current = parts[parts.length - 1].trim().toLowerCase()
    const already = new Set(parts.slice(0, -1).map((p) => p.trim().toLowerCase()))
    if (current === '') return []
    return allTags.map((t) => t.name)
      .filter((n) => n.includes(current) && n !== current && !already.has(n))
      .slice(0, 6)
  }, [tagDraft, tagEdit, allTags])

  const applySuggestion = (sug) => {
    const parts = tagDraft.split(',')
    parts[parts.length - 1] = ' ' + sug
    setTagDraft(parts.join(',').replace(/^ /, '') + ', ')
  }

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
      setTxs((list) => list.map((x) => (x.id === tx.id ? { ...x, domain_id: domainId } : x)))
      if (domainId) proposeRule(tx, domainId)
    } catch (e) { toast.err(e.message) }
  }

  const saveTags = async (tx) => {
    const tags = tagDraft.split(',').map((s) => s.trim().toLowerCase()).filter(Boolean)
    try {
      await api.setTransactionMeta(tx.id, { tags })
      setTxs((list) => list.map((x) => (x.id === tx.id ? { ...x, tags } : x)))
      setTagEdit(null); setTagDraft('')
      api.tags().then(setAllTags).catch(() => {})
    } catch (e) { toast.err(e.message) }
  }

  const runAiEnrich = async () => {
    setEnrichBusy(true)
    try {
      const r = await api.aiEnrich(month || undefined)
      toast.ok(r.domains_set > 0 || r.tags_set > 0
        ? `IA : ${r.domains_set} catégorie(s) et ${r.tags_set} tag(s) attribués.`
        : "l'IA n'a rien proposé de plus.")
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
      toast.ok('Catégorie retirée.')
    } catch (e2) { toast.err(e2.message) }
  }
  const changeDomainColor = async (d, color) => {
    // maj optimiste locale, puis persistance
    setDomains((list) => list.map((x) => (x.id === d.id ? { ...x, color } : x)))
    try {
      await api.updateDomain(d.id, { color })
    } catch (e2) { toast.err(e2.message) }
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
    new Date(d).toLocaleDateString('fr-FR', { weekday: 'long', day: 'numeric', month: 'long' })

  return (
    <div className="section">
      {sorting && (
        <SortStack
          transactions={txs.filter((t) => !t.domain_id)}
          domains={domains}
          onClose={() => { setSorting(false); reload() }}
          onSorted={(id, domainId) => setTxs((list) => list.map((x) => x.id === id ? { ...x, domain_id: domainId } : x))}
        />
      )}
      <div className="domains-bar">
        <span className="domains-label">Catégories</span>
        {(showAllDomains || manageDomains ? topDomains : topDomains.slice(0, 4)).map((d) => (
          <span key={d.id} className="domain-chip" style={{ '--dc': d.color }}>
            <span className="dom-dot" />{d.name}
            {manageDomains && (
              <label className="domain-color" title="Changer la couleur">
                <input type="color" value={d.color || '#8A8A99'}
                       onChange={(e) => changeDomainColor(d, e.target.value)} />
              </label>
            )}
            {manageDomains && <button className="domain-x" onClick={() => removeDomain(d)} title="Retirer">×</button>}
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
            <input value={newDomain} onChange={(e) => setNewDomain(e.target.value)} placeholder="nouvelle catégorie" />
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
                          className="rule-input"
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
                                title="Changer la catégorie">
                          {topDomains.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
                        </select>
                      </span>
                    </td>
                    {/* Tags automatiques : saisis « a, b, c », appliqués au blur */}
                    <td>
                      <input
                        className="rule-tags-input"
                        defaultValue={(r.tags || []).join(', ')}
                        placeholder="+ tags…"
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

      {(sourceFilter.budget || sourceFilter.ids.length > 0) && <div className="notice">Vue des opérations contribuant {sourceFilter.budget ? "à l’enveloppe sélectionnée" : "à la récurrence sélectionnée"}. Les crédits sont affichés avec leur signe. <button className="link" onClick={() => setSourceFilter({ budget: '', ids: [] })}>Retirer ce filtre de provenance</button></div>}
      {!(sourceFilter.budget || sourceFilter.ids.length) && txs.length === 1000 && <p className="notice">Affichage limité à 1 000 opérations. D’autres opérations peuvent contribuer aux totaux : affine la période pour examiner les sources restantes. Cette liste n’est pas une preuve de réconciliation complète.</p>}
      {fetching && <p role="status" className="notice">Chargement des opérations sources…</p>}
      {(sourceFilter.budget || sourceFilter.ids.length > 1000) && <nav aria-label="Pagination des sources" className="notice" style={{display:'flex',alignItems:'center',gap:12,flexWrap:'wrap'}}><button className="link" disabled={fetching || sourcePage === 0} onClick={() => setSourcePage(p => p - 1)}>Sources précédentes</button><span>Page {sourcePage + 1}{sourceFilter.ids.length ? ` sur ${Math.ceil(sourceFilter.ids.length / 1000)}` : ''} · 1 000 opérations maximum par page</span><button className="link" disabled={fetching || (sourceFilter.ids.length ? (sourcePage + 1) * 1000 >= sourceFilter.ids.length : txs.length < 1000)} onClick={() => setSourcePage(p => p + 1)}>Sources suivantes</button></nav>}
      <div className="toolbar">
        {!showDates ? (
          <select value={month} onChange={(e) => setMonth(e.target.value)} aria-label="Mois">
            <option value="">Tous les mois</option>
            {months.map((m) => <option key={m} value={m}>{monthLabel(m)}</option>)}
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
          onClick={() => { setShowDates((v) => !v); if (showDates) { setFrom(''); setTo('') } else { setMonth('') } }}
          title="Basculer entre mois et période personnalisée"
        >
          {showDates ? '← par mois' : 'par période'}
        </button>
        <select value={domainFilter} onChange={(e) => setDomainFilter(e.target.value)} aria-label="Domaine">
          <option value="">Toutes les catégories</option>
          <option value="none">Sans catégorie</option>
          {topDomains.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
        </select>
        <input
          type="search"
          placeholder="Rechercher un libellé…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Recherche"
          style={{ flex: 1, minWidth: 160 }}
        />
        <span style={{ color: 'var(--muted)', fontSize: 13 }}>{txs.length} transaction{txs.length > 1 ? 's' : ''}</span>
        {allTags.length > 0 && (
          <select value={tagFilter} onChange={(e) => setTagFilter(e.target.value)} aria-label="Tag">
            <option value="">Tous les tags</option>
            <option value="__none__">— Sans tag —</option>
            {allTags.map((t) => <option key={t.name} value={t.name}>#{t.name} ({t.count})</option>)}
          </select>
        )}
        <a className="btn-secondary" href={api.exportUrl(search ? '' : month)} download>Exporter CSV</a>
        {txs.some((t) => !t.domain_id) && (
          <button className="btn-secondary" onClick={applyRules} disabled={applyBusy}
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
          <button className="btn-secondary ai" onClick={runAiEnrich} disabled={enrichBusy}>
            {enrichBusy ? 'Tri en cours…' : 'Trier catégories & tags (IA)'}
          </button>
        )}
      </div>

      {err && <div className="notice err">{err}</div>}

      {txs.length === 0 ? (
        <div className="empty">Aucune transaction pour ces filtres.</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th style={{ width: 96 }}>Date</th>
              <th>Libellé</th>
              <th style={{ width: 150 }}>Catégorie</th>
              <th style={{ width: 220 }}>Tags</th>
              <th className="num" style={{ width: 110 }}>Montant</th>
            </tr>
          </thead>
          <tbody>
            {Object.entries(byDay).map(([date, rows]) => {
              const net = rows.reduce((s, t) => s + t.amount, 0)
              return (
                <Fragment key={date}>
                  <tr className="tx-day">
                    <td colSpan={5}>
                      {dayLabel(date)}
                      <span className={'tx-day-total' + (net >= 0 ? ' pos' : '')}>
                        {net >= 0 ? '+' : ''}{eur(net)}
                      </span>
                    </td>
                  </tr>
                  {rows.map((t) => (
                    <tr key={t.id}>
                <td className="amt" style={{ color: 'var(--muted)', fontSize: 13 }}>{t.date}</td>
                <td>
                  <div className="tx-label">
                    {kindIcon(t.kind) && <span className="kicon" aria-hidden="true">{kindIcon(t.kind)}</span>}
                    {t.display_label || t.label}
                  </div>
                  {t.display_label && t.display_label !== t.label && (
                    <div className="tx-raw">{t.label}</div>
                  )}
                </td>
                <td className="dom-cell">
                  <div className={'dom-pick' + (t.domain_id ? ' set' : '')}
                       style={t.domain_id ? { '--dc': domainById[t.domain_id]?.color || 'var(--muted)' } : {}}>
                    <span className="dom-dot" />
                    <select className="dom-select" value={t.domain_id ?? ''}
                            onChange={(e) => changeDomain(t, e.target.value)}>
                      <option value="">— à ranger —</option>
                      {topDomains.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
                    </select>
                  </div>
                </td>
                <td className="tag-cell">
                  {tagEdit === t.id ? (
                    <form className="tag-edit" onSubmit={(e) => { e.preventDefault(); saveTags(t) }}>
                      <div className="tag-edit-field">
                        <input autoFocus value={tagDraft} placeholder="loyer, loyer janvier…"
                               onChange={(e) => setTagDraft(e.target.value)}
                               onKeyDown={(e) => { if (e.key === 'Escape') setTagEdit(null) }} />
                        {tagSuggestions.length > 0 && (
                          <div className="tag-suggest">
                            {tagSuggestions.map((sug) => (
                              <button type="button" key={sug} className="tag-suggest-item"
                                      onMouseDown={(e) => { e.preventDefault(); applySuggestion(sug) }}>
                                #{sug}
                              </button>
                            ))}
                          </div>
                        )}
                      </div>
                      <button type="submit" title="Enregistrer">✓</button>
                    </form>
                  ) : (
                    <div className="tag-row" onClick={() => { setTagEdit(t.id); setTagDraft((t.tags || []).join(', ')) }}
                         title="Cliquer pour éditer les tags">
                      {(t.tags || []).length > 0
                        ? t.tags.map((tag) => <span key={tag} className="tag-pill">{tag}</span>)
                        : <span className="tag-add">+ ajouter</span>}
                    </div>
                  )}
                </td>
                <td className={'num amt ' + (t.amount >= 0 ? 'credit' : 'debit')}>
                  {t.amount >= 0 ? '+' : ''}{eur(t.amount)}
                </td>
              </tr>
                  ))}
                </Fragment>
              )
            })}
          </tbody>
        </table>
      )}
    </div>
  )
}
