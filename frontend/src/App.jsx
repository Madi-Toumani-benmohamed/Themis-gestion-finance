import { useCallback, useEffect, useRef, useState } from 'react'
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
import CommandPalette from './CommandPalette'
import { ICONS } from './navIcons'
import { Logo } from './Logo'

const TABS = [
  ['dashboard', 'Tableau de bord'],
  ['transactions', 'Transactions'],
  ['analyse', 'Analyse'],
  ['budgets', 'Budgets'],
  ['credits', 'Crédits & taxes'],
  ['simulate', 'Simulation'],
  ['accounts', 'Comptes'],
  ['statements', 'Relevés'],
  ['advice', 'Conseils IA'],
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
  const navigateTo = (id, opts) => { if (opts?.q != null) setTxSearch(opts.q); setTxMonth(opts?.month || ''); setTab(id); setRefreshKey((k) => k + 1) }
  const [drag, setDrag] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadProg, setUploadProg] = useState(null)
  const [refreshKey, setRefreshKey] = useState(0)
  const [aiEnabled, setAiEnabled] = useState(false)
  const [stamped, setStamped] = useState(false)
  const [me, setMe] = useState(undefined)      // undefined = en cours, null = déconnecté
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
  }, [me, refreshKey])

  const logout = async () => {
    try { await api.authLogout() } catch { /* ignore */ }
    setMe(null); setSetupRequired(false)
    api.authStatus().then((st) => setSetupRequired(st.setup_required)).catch(() => {})
  }

  const doUpload = useCallback(async (file) => {
    if (!file) return
    const name = file.name.toLowerCase()
    if (!name.endsWith('.pdf') && !name.endsWith('.csv')) {
      toast.err('Formats acceptés : PDF (relevé) ou CSV (export bancaire).')
      return
    }
    setUploading(true)
    // Progression réelle : un jeton accompagne l'upload, et on interroge le
    // serveur pendant le traitement (conversion, OCR page i/N, IA…).
    const token = (crypto.randomUUID ? crypto.randomUUID() : String(Date.now() + Math.random()))
    setUploadProg({ stage: 'Envoi du fichier', pct: 1 })
    const poll = setInterval(async () => {
      try {
        const p = await api.uploadProgress(token)
        if (p && p.stage) setUploadProg({ stage: p.stage, detail: p.detail, pct: p.pct })
      } catch { /* transitoire : on retentera au tick suivant */ }
    }, 800)
    try {
      const r = await api.upload(file, uploadAccount, token)
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
    } catch (e) {
      toast.err(e.message)
    } finally {
      clearInterval(poll)
      setUploadProg(null)
      setUploading(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }, [])

  const onDrop = (e) => {
    e.preventDefault()
    setDrag(false)
    doUpload(e.dataTransfer.files?.[0])
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
      <aside className="sidebar">
        <div className="sidebar-brand">
          <Logo size={24} className="brand-logo" />
          <h1>thémis<span className="tick">.</span></h1>
        </div>
        <nav className="sidebar-nav" role="tablist">
          {TABS.map(([id, label], n) => (
            <button
              key={id}
              style={{ '--n': n }}
              role="tab"
              aria-selected={tab === id}
              className={tab === id ? 'active' : ''}
              onClick={() => setTab(id)}
            >
              <span className="nav-icon" aria-hidden="true">{ICONS[id]}</span>
              <span className="nav-label">{label}</span>
            </button>
          ))}
        </nav>
        <div className="sidebar-footer">
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
        <span className="topbar-title">{(TABS.find(([id]) => id === tab) || [])[1]}</span>
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
            {uploading ? <><span className="spinner" /> Lecture…</> : 'Importer un relevé'}
          </span>
          <input
            ref={fileRef}
            type="file"
            accept=".pdf,.csv,application/pdf,text/csv"
            onChange={(e) => doUpload(e.target.files?.[0])}
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
        {tab === 'statements' && <Statements onChanged={() => setRefreshKey((k) => k + 1)} />}
        {tab === 'admin' && <Admin isAdmin={me.role === 'admin'} />}
        {tab === "advice" && <Advice aiEnabled={aiEnabled} onNavigate={navigateTo} />}
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
        tabs={TABS}
        onNavigate={navigateTo}
        onUpload={() => fileRef.current?.click()}
      />

      <ScrollTop />

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
