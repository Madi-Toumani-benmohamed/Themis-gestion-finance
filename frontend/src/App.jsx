import { Fragment, useCallback, useEffect, useRef, useState } from 'react'
import { api } from './api'
import { useToast } from './Toast'
import Dashboard from './pages/Dashboard'
import Transactions from './pages/Transactions'
import Statements from './pages/Statements'
import Advice from './pages/Advice'
import Budgets from './pages/Budgets'
import Admin from './pages/Admin'
import Auth from './pages/Auth'
import Accounts from './pages/Accounts'
import Credits from './pages/Credits'
import Analyse from './pages/Analyse'
import Simulate from './pages/Simulate'
import Assistant from './pages/Assistant'
import AssistantFloating from './components/AssistantFloating'
import CommandPalette from './CommandPalette'
import { ICONS } from './navIcons'
import { Logo } from './Logo'
import ScrollProgress from './ScrollProgress'

// Familles de navigation. `admin` est rendu séparément, dans le pied.
const NAV_BASE = [
  ['Pilotage',   [['dashboard', 'Tableau de bord']]],
  ['Argent',     [['transactions', 'Transactions'], ['analyse', 'Analyse'], ['budgets', 'Budgets']]],
  ['Patrimoine', [['accounts', 'Comptes'], ['credits', 'Crédits & taxes'], ['simulate', 'Simulation']]],
  ['Sources',    [['statements', 'Relevés'], ['advice', 'Conseils IA']]],
]
// Liste à plat (titre de page, recherche) — inclut admin.
const TABS_BASE = [
  ...NAV_BASE.flatMap(([, items]) => items),
  ['admin', 'Admin'],
]

// Bouton « remonter en haut » : apparaît après un défilement, ramène en haut
// en douceur. Évite de scroller longtemps pour atteindre les filtres/menu.
function ScrollTop() {
  const [show, setShow] = useState(false)
  useEffect(() => {
    const onScroll = () => setShow(window.scrollY > 500)
    window.addEventListener('scroll', onScroll, { passive: true })
    onScroll()
    return () => window.removeEventListener('scroll', onScroll)
  }, [])
  if (!show) return null
  return (
    <button
      className="scroll-top"
      onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
      title="Remonter en haut"
      aria-label="Remonter en haut de la page"
    >
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 19V5M5 12l7-7 7 7" /></svg>
    </button>
  )
}

export default function App() {
  const toast = useToast()
  const [tab, setTab] = useState('dashboard')
  const [cmdOpen, setCmdOpen] = useState(false)
  const [theme, setTheme] = useState(() => {
    try { return localStorage.getItem('themis-theme') || 'light' } catch { return 'light' }
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    document.documentElement.style.background = theme === 'dark' ? '#0F0F14' : '#FBFBFD'
    try { localStorage.setItem('themis-theme', theme) } catch { /* stockage indisponible */ }
  }, [theme])
  const [txSearch, setTxSearch] = useState('')
  const [txMonth, setTxMonth] = useState('')
  const [statementMonth, setStatementMonth] = useState('')
  const navigateTo = (id, opts) => { if (opts?.q != null) setTxSearch(opts.q); setTxMonth(opts?.month || ''); if (id === 'statements') setStatementMonth(opts?.month || ''); setTab(id); setRefreshKey((k) => k + 1) }
  const [drag, setDrag] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadProg, setUploadProg] = useState(null)
  const [refreshKey, setRefreshKey] = useState(0)
  const [aiEnabled, setAiEnabled] = useState(false)
  const [stamped, setStamped] = useState(false)
  const [me, setMe] = useState(undefined)      // undefined = en cours, null = déconnecté
  const [unsorted, setUnsorted] = useState(0)  // transactions sans catégorie (badge nav)
  const [setupRequired, setSetupRequired] = useState(false)
  const [accounts, setAccounts] = useState([])
  const [uploadAccount, setUploadAccount] = useState(1)
  const fileRef = useRef()

  useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
        e.preventDefault()
        setCmdOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  useEffect(() => {
    api.authStatus()
      .then((st) => {
        setSetupRequired(st.setup_required)
        if (st.setup_required) { setMe(null); return null }
        return api.authMe().then(setMe).catch(() => setMe(null))
      })
      .catch(() => setMe(null))
  }, [])

  useEffect(() => {
    if (!me) return
    api.health().then((h) => setAiEnabled(h.ai_enabled)).catch(() => {})
    api.accounts().then((a) => {
      setAccounts(a)
      if (a.length && !a.some((x) => x.id === uploadAccount)) setUploadAccount(a[0].id)
    }).catch(() => {})
    api.summary(1).then((s) => setUnsorted(s.uncategorized || 0)).catch(() => {})
  }, [me, refreshKey])

  const logout = async () => {
    try { await api.authLogout() } catch { /* ignore */ }
    setMe(null); setSetupRequired(false)
    api.authStatus().then((st) => setSetupRequired(st.setup_required)).catch(() => {})
  }

  const doUpload = useCallback(async (file, opts = {}) => {
    const { silent = false, index = 0, total = 0 } = opts
    if (!file) return { ok: false, inserted: 0 }
    const name = file.name.toLowerCase()
    if (!name.endsWith('.pdf') && !name.endsWith('.csv')) {
      if (!silent) toast.err('Formats acceptés : PDF (relevé) ou CSV (export bancaire).')
      return { ok: false, inserted: 0 }
    }
    setUploading(true)
    const prefix = total > 1 ? `Relevé ${index}/${total} · ` : ''
    // Progression réelle : un jeton accompagne l'upload, et on interroge le
    // serveur pendant le traitement (conversion, OCR page i/N, IA…).
    const token = (crypto.randomUUID ? crypto.randomUUID() : String(Date.now() + Math.random()))
    setUploadProg({ stage: prefix + 'Envoi du fichier', pct: 1 })
    const poll = setInterval(async () => {
      try {
        const p = await api.uploadProgress(token)
        if (p && p.stage) setUploadProg({ stage: prefix + p.stage, detail: p.detail, pct: p.pct })
      } catch { /* transitoire : on retentera au tick suivant */ }
    }, 800)
    try {
      const r = await api.upload(file, uploadAccount, token)
      if (!silent) {
        if (r.inserted === 0) {
          toast.info('Relevé déjà importé — aucune nouvelle transaction.')
        } else {
          const via = r.parser === 'local' ? 'parseur local' : 'IA'
          const check = r.validated ? '· validation comptable OK' : '· validation comptable impossible, vérifie les montants'
          const aiCat = r.ai_categorized > 0 ? ` · ${r.ai_categorized} classées par l'IA` : ''
          const acct = accounts.find((a) => a.id === uploadAccount)
          const where = acct && acct.id !== 1 ? ` → ${acct.name}` : ''
          toast.ok(`${r.inserted} transactions importées${where} (${via}) ${check}${aiCat}`)
          if (r.validated) {
            setStamped(true)
            setTimeout(() => setStamped(false), 1900)
          }
        }
        setRefreshKey((k) => k + 1)
      }
      return { ok: true, inserted: r.inserted || 0 }
    } catch (e) {
      if (!silent) toast.err(e.message)
      return { ok: false, inserted: 0 }
    } finally {
      clearInterval(poll)
      setUploadProg(null)
      setUploading(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }, [uploadAccount, accounts])

  // Traite plusieurs relevés à la suite (séquentiellement : l'OCR est lourd et
  // le backend traite un relevé à la fois). Un récapitulatif global à la fin,
  // plutôt qu'un toast par fichier.
  const doUploadMany = useCallback(async (fileList) => {
    const files = Array.from(fileList || []).filter((f) => {
      const n = f.name.toLowerCase()
      return n.endsWith('.pdf') || n.endsWith('.csv')
    })
    if (files.length === 0) {
      toast.err('Formats acceptés : PDF (relevé) ou CSV (export bancaire).')
      return
    }
    if (files.length === 1) { await doUpload(files[0]); return }

    let okCount = 0, totalInserted = 0, failed = 0
    for (let i = 0; i < files.length; i++) {
      const res = await doUpload(files[i], { silent: true, index: i + 1, total: files.length })
      if (res && res.ok) { okCount++; totalInserted += res.inserted } else { failed++ }
    }
    // Récapitulatif global
    if (failed === 0) {
      toast.ok(`${files.length} relevés importés · ${totalInserted} transactions au total`)
    } else if (okCount > 0) {
      toast.info(`${okCount}/${files.length} relevés importés (${totalInserted} transactions) · ${failed} en échec`)
    } else {
      toast.err(`Aucun relevé importé (${failed} en échec)`)
    }
    setRefreshKey((k) => k + 1)
  }, [doUpload])

  const onDrop = (e) => {
    e.preventDefault()
    setDrag(false)
    doUploadMany(e.dataTransfer.files)
  }

  if (me === undefined) {
    return <div className="app boot"><span className="spinner" /></div>
  }
  if (!me) {
    return <Auth setupRequired={setupRequired} onAuthed={(u) => { setSetupRequired(false); setMe(u) }} />
  }

  return (
    <div
      className="app"
      onDragOver={(e) => { e.preventDefault(); setDrag(true) }}
      onDragLeave={(e) => { if (e.target === e.currentTarget) setDrag(false) }}
      onDrop={onDrop}
    >
      <div className="app-shell">
      <ScrollProgress />
      <aside className="sidebar">
        <div className="sidebar-brand brand">
          <Logo size={24} className="brand-logo" animate />
          <h1>thémis<span className="tick">.</span></h1>
        </div>
        <nav className="sidebar-nav" role="tablist">
          {[...NAV_BASE, aiEnabled && ['Intelligence', [['assistant', 'Assistant']]]].filter(Boolean).map(([group, items]) => (
            <Fragment key={group}>
              <div className="nav-group-label">{group}</div>
              {items.map(([id, label]) => (
                <button
                  key={id}
                  role="tab"
                  aria-selected={tab === id}
                  className={tab === id ? 'active' : ''}
                  onClick={() => setTab(id)}
                >
                  <span className="nav-icon" aria-hidden="true">{ICONS[id]}</span>
                  <span className="nav-label">{label}</span>
                  {id === 'transactions' && unsorted > 0 && (
                    <span className="nav-count" title={`${unsorted} transaction${unsorted > 1 ? 's' : ''} sans catégorie`}>{unsorted}</span>
                  )}
                </button>
              ))}
            </Fragment>
          ))}
        </nav>
        <div className="sidebar-footer">
          {me?.role === 'admin' && (
            <button
              role="tab"
              aria-selected={tab === 'admin'}
              className={tab === 'admin' ? 'active' : ''}
              onClick={() => setTab('admin')}
            >
              <span className="nav-icon" aria-hidden="true">{ICONS.admin}</span>
              <span className="nav-label">Admin</span>
            </button>
          )}
          <button
            className="theme-toggle"
            onClick={() => setTheme((t) => (t === 'dark' ? 'light' : 'dark'))}
            title={theme === 'dark' ? 'Passer en clair' : 'Passer en sombre'}
            aria-label="Basculer le thème clair/sombre"
          >
            {theme === 'dark' ? (
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" /></svg>
            ) : (
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" /></svg>
            )}
            <span>{theme === 'dark' ? 'Clair' : 'Sombre'}</span>
          </button>
          <div className="session">
            <span className="session-user">{me.username}{me.role === 'admin' && <span className="session-role">admin</span>}</span>
            <button className="link" onClick={logout}>déconnexion</button>
          </div>
        </div>
      </aside>

      <div className="app-main">
      <header className="topbar">
        <span className="topbar-title">{(TABS_BASE.find(([id]) => id === tab) || [])[1]}</span>
        <div className="topbar-right">
          <div className="slot-wrap">
          {accounts.length > 1 && (
            <select
              className="slot-account"
              value={uploadAccount}
              onChange={(e) => setUploadAccount(Number(e.target.value))}
              aria-label="Compte de destination du relevé"
              title="Sur quel compte importer ce relevé ?"
            >
              {accounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            </select>
          )}
          <label
          className={'slot' + (drag ? ' drag' : '') + (uploading ? ' busy' : '')}
          tabIndex={0}
          onKeyDown={(e) => { if (e.key === 'Enter') fileRef.current?.click() }}
        >
          <span className="slot-text">
            {uploading ? <><span className="spinner" /> Lecture…</> : 'Importer un ou plusieurs relevés'}
          </span>
          <input
            ref={fileRef}
            type="file"
            multiple
            accept=".pdf,.csv,application/pdf,text/csv"
            onChange={(e) => doUploadMany(e.target.files)}
          />
        </label>
        </div>
        </div>
      </header>

      {stamped && (
        <div className="stamp-overlay" aria-hidden="true">
          <span className="stamp ok slam">Validé</span>
        </div>
      )}
      {drag && <div className="drop-veil" aria-hidden="true"><span>Dépose ton relevé</span></div>}

      <main className="page" key={tab + ':' + refreshKey}>
        {tab === 'dashboard' && <Dashboard onGoUpload={() => fileRef.current?.click()} onNavigate={navigateTo} />}
        {tab === 'transactions' && <Transactions initialSearch={txSearch} initialMonth={txMonth} />}
        {tab === 'analyse' && <Analyse onNavigate={navigateTo} />}
        {tab === 'budgets' && <Budgets />}
        {tab === 'credits' && <Credits />}
        {tab === 'simulate' && <Simulate />}
        {tab === 'accounts' && <Accounts />}
        {tab === 'statements' && <Statements initialMonth={statementMonth} onMonthChange={setStatementMonth} onGoUpload={() => fileRef.current?.click()} onChanged={() => setRefreshKey((k) => k + 1)} />}
        {tab === 'admin' && <Admin isAdmin={me.role === 'admin'} />}
        {tab === "advice" && <Advice aiEnabled={aiEnabled} onNavigate={navigateTo} />}
        {tab === 'assistant' && <Assistant />}
      </main>

      <footer className="colophon">
        <span>thémis — comptes personnels</span>
        <span className="colophon-rule" aria-hidden="true" />
        <span>traitement en local · tes relevés ne quittent pas ta machine</span>
      </footer>
      </div>{/* /app-main */}
      </div>{/* /app-shell */}

      <CommandPalette
        open={cmdOpen}
        onClose={() => setCmdOpen(false)}
        tabs={TABS_BASE}
        onNavigate={navigateTo}
        onUpload={() => fileRef.current?.click()}
      />

      <ScrollTop />
      {aiEnabled && <AssistantFloating aiEnabled={aiEnabled} />}

      {uploadProg && (
        <div className="upcard" role="status" aria-live="polite">
          <div className="upcard-head">
            <span className="spinner" aria-hidden="true" />
            <div className="upcard-text">
              <div className="upcard-stage">{uploadProg.stage}</div>
              {uploadProg.detail && <div className="upcard-detail">{uploadProg.detail}</div>}
            </div>
            <span className="upcard-pct">{Math.max(1, uploadProg.pct || 0)} %</span>
          </div>
          <div className="upcard-track">
            <div className="upcard-bar" style={{ width: `${Math.max(2, uploadProg.pct || 0)}%` }} />
          </div>
        </div>
      )}
    </div>
  )
}
