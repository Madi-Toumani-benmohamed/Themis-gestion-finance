import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { api, eur, kindIcon, monthLabel } from '../api'
import { WIDGETS, DEFAULT_LAYOUT } from '../widgets'
import { PresencePanel, m, AnimatePresence, ease, useMotionReduced } from '../motion'
import { AnimatedNumber } from '../motion/AnimatedNumber'
import { MotionSurface } from '../motion/Motion'
import { CinematicDashboard } from '../motion/CinematicDashboard'
import { dashboardModel } from '../components/dashboardModel.mjs'
import { FinancialSummary } from '../components/FinancialSummary'

let uid = 100
const newId = () => `w${uid++}_${Date.now() % 10000}`

// Journal d'événements : le cœur de thémis. Plutôt qu'un tableau à interpréter,
// une chronologie de faits marquants — passés, présents et à venir — chacun
// éventuellement assorti d'une action. On lit son argent, on ne le déchiffre pas.
// Reste à vivre projeté : ce qu'il reste sur le compte une fois retirés les
// prélèvements récurrents encore attendus d'ici la fin du mois. Répond à la
// vraie question — « combien puis-je dépenser sans mauvaise surprise ? » — au
// lieu de la laisser flotter dans un coin de la tête.
// Période couverte : jusqu'où vont les données, et dernier import. Répond à
// « suis-je à jour ? » sans avoir à ouvrir l'onglet Relevés.
// Hero : la conclusion de l'analyse. Un ton global, une phrase de situation,
// et les faits saillants du mois — calculés depuis les données, pas générés.
// Tout le reste de la page vient expliquer cette conclusion.
// Sélecteur de mois navigable dans le dashboard.
// Permet de consulter le mois précédent/suivant, de sélectionner directement
// un mois et une année, et de revenir rapidement au mois en cours.
function MonthSelector({ selectedMonth, onMonthChange }) {
  const now = new Date()
  const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  const isCurrent = selectedMonth === currentMonth

  // Liste des mois disponibles (sur les 24 derniers mois)
  const availableMonths = useMemo(() => {
    const months = []
    const [selectedYear, selectedNumber] = selectedMonth.split('-').map(Number)
    const count = Math.max(24, (now.getFullYear() - selectedYear) * 12 + now.getMonth() - selectedNumber + 2)
    for (let i = 0; i < count; i++) {
      const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
      months.push({
        value: `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`,
        label: monthLabel(`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`),
      })
    }
    return months
  }, [selectedMonth, currentMonth])

  const goToPrev = () => {
    const [y, m] = selectedMonth.split('-').map(Number)
    const prev = new Date(y, m - 2, 1)
    onMonthChange(`${prev.getFullYear()}-${String(prev.getMonth() + 1).padStart(2, '0')}`)
  }

  const goToNext = () => {
    const [y, m] = selectedMonth.split('-').map(Number)
    // On ne permet pas de naviguer vers un mois futur
    if (selectedMonth >= currentMonth) return
    const next = new Date(y, m, 1)
    const nextStr = `${next.getFullYear()}-${String(next.getMonth() + 1).padStart(2, '0')}`
    if (nextStr <= currentMonth) {
      onMonthChange(nextStr)
    }
  }

  const goToCurrent = () => onMonthChange(currentMonth)

  return (
    <div className="month-selector">
      <button
        className="month-nav-btn"
        onClick={goToPrev}
        disabled={!selectedMonth}
        title="Mois précédent"
        aria-label="Mois précédent"
      >
        ‹
      </button>

      <div className="month-selector-content">
        <select
          className="month-selector-select"
          value={selectedMonth}
          onChange={(e) => onMonthChange(e.target.value)}
          title="Sélectionner un mois"
          aria-label="Sélectionner un mois"
        >
          {availableMonths.map((m) => (
            <option key={m.value} value={m.value}>
              {m.label}
            </option>
          ))}
        </select>
      </div>

      <button
        className="month-nav-btn"
        onClick={goToNext}
        disabled={selectedMonth >= currentMonth}
        title="Mois suivant"
        aria-label="Mois suivant"
      >
        ›
      </button>

      <button
        className={`month-now-btn ${isCurrent ? 'active' : ''}`}
        onClick={goToCurrent}
        title="Retour au mois en cours"
        aria-label="Retour au mois en cours"
      >
        {isCurrent ? '●' : '○'}
      </button>
    </div>
  )
}

// Indicateurs clés : les quatre chiffres essentiels du mois de référence.
function KpiRow({ selectedMonth }) {
  const [kpi, setKpi] = useState(null)
  const [loadedMonth, setLoadedMonth] = useState('')
  const [error, setError] = useState(null)
  useEffect(() => {
    let active = true
    setError(null)
    Promise.all([api.summary(24, selectedMonth), api.remaining(selectedMonth).catch(() => null)]).then(([s2, rav]) => {
      if (!active) return
      setLoadedMonth(selectedMonth)
      const model = dashboardModel(s2.monthly, selectedMonth)
      setKpi(model ? { ...model, rav: rav?.available && Number.isFinite(rav.remaining) ? rav.remaining : null } : null)
    }).catch(e => { if (active) { setKpi(null); setError(e.message); setLoadedMonth(selectedMonth) } })
    return () => { active = false }
  }, [selectedMonth])
  if (error) return <p className="notice err" role="alert">Indicateurs indisponibles : {error}</p>
  if (!kpi) return <p className="motion-period-state" role="status">{loadedMonth === selectedMonth ? 'Aucun indicateur observé pour cette période.' : 'Chargement des indicateurs…'}</p>

  const variation = (value, inverse = false) => {
    if (value == null) return { value: '—', context: 'Comparaison indisponible', tone: '' }
    const tone = value === 0 ? '' : (value > 0) !== inverse ? 'kpi-pos' : 'kpi-neg'
    return { value: `${value > 0 ? '+' : ''}${value} %`, context: 'vs mois précédent complet', tone }
  }
  const incomeChange = variation(kpi.incomeDelta)
  const expenseChange = variation(kpi.spentDelta, true)
  const savings = kpi.savings == null
    ? { value: '—', context: 'Indisponible sur cette période', tone: '' }
    : { value: `${new Intl.NumberFormat('fr-FR', { maximumFractionDigits: 1 }).format(kpi.savings)} %`, context: 'résultat / revenus observés', tone: kpi.savings < 0 ? 'kpi-neg' : '' }
  const cell = (label, value, tone, context, money = false) => (
    <div className={`kpi ${tone || ''}`}>
      <div className="kpi-label">{label}</div>
      <div className="kpi-value">{money ? <AnimatedNumber value={value} missing="Indisponible" /> : value}</div>
      <span className="kpi-period-badge">{context}</span>
    </div>
  )

  return (
    <>{loadedMonth !== selectedMonth && <p className="motion-period-state" role="status">Actualisation des indicateurs ; données affichées : {monthLabel(loadedMonth)}.</p>}<div className="kpi-row" data-period={loadedMonth} data-pending={loadedMonth !== selectedMonth} aria-busy={loadedMonth !== selectedMonth}>
      {cell('Variation des revenus', incomeChange.value, incomeChange.tone, incomeChange.context)}
      {cell('Variation des dépenses', expenseChange.value, expenseChange.tone, expenseChange.context)}
      {cell('Taux d’épargne des flux', savings.value, savings.tone, savings.context)}
      {kpi.rav != null && cell('Reste à vivre estimé', kpi.rav, kpi.rav < 0 ? 'kpi-neg' : '', kpi.periodLabel || 'Estimation disponible', true)}
    </div></>
  )
}

function Journal({ onNavigate, selectedMonth }) {
  const [events, setEvents] = useState(null)
  useEffect(() => {
    let active = true
    setEvents(null)
    api.journal(selectedMonth).then(d => { if (active) setEvents(d.events || []) }).catch(() => { if (active) setEvents([]) })
    return () => { active = false }
  }, [selectedMonth])
  if (!events || events.length === 0) return null

  const icon = { positive: '✓', attention: '⚠', neutral: '·' }
  const runAction = (a) => {
    if (!a) return
    if (a.kind === 'navigate') onNavigate?.(a.arg)
  }

  // Hiérarchie : le backend trie déjà par importance ; on affiche les
  // événements majeurs en plein, et on replie les secondaires pour que le
  // journal raconte l'analyse au lieu de tout mettre au même niveau.
  const major = events.filter((e) => (e.importance ?? 50) >= 60)
  const minor = events.filter((e) => (e.importance ?? 50) < 60)
  const primary = major.length > 0 ? major : events.slice(0, 3)
  const secondary = major.length > 0 ? minor : events.slice(3)

  const renderEvent = (e, i) => (
    <li key={i} className={`jevent jevent-${e.tone}`}>
      <span className="jevent-mark" aria-hidden="true">{icon[e.tone] || '·'}</span>
      <div className="jevent-body">
        <div className="jevent-line">
          <span className="jevent-title">{kindIcon(e.merchant_kind) && <span className="kicon" aria-hidden="true">{kindIcon(e.merchant_kind)}</span>}{e.title}</span>
          {Number.isFinite(e.amount) ? <span className={`jevent-amount ${e.amount > 0 ? 'pos' : 'neg'}`}>{e.amount > 0 ? '+' : ''}{eur(e.amount)}</span> : null}
        </div>
        <p className="jevent-detail">{e.detail}</p>
        {e.action && (
          <button className="jevent-action" onClick={() => runAction(e.action)}>
            {e.action.label} →
          </button>
        )}
      </div>
    </li>
  )

  return (
    <section className="journal">
      <div className="journal-head">
        <h2>Ton journal</h2>
        <span className="journal-sub">ce qui compte, en ce moment</span>
      </div>
      <ol className="journal-feed">
        {primary.map(renderEvent)}
      </ol>
      {secondary.length > 0 && (
        <details className="journal-more">
          <summary>Et aussi… ({secondary.length})</summary>
          <ol className="journal-feed journal-feed-minor">
            {secondary.map(renderEvent)}
          </ol>
        </details>
      )}
    </section>
  )
}

export default function Dashboard({ onGoUpload, onNavigate }) {
  const reduced = useMotionReduced()
  const [data, setData] = useState(null)
  const [err, setErr] = useState(null)
  const [layout, setLayout] = useState(null)
  const [editing, setEditing] = useState(false)
  const saveQueue = useRef(Promise.resolve()), saveRevision = useRef(0), saveTimer = useRef(null)
  const [saveState, setSaveState] = useState('idle') // idle | saving | saved
  const [selectedMonth, setSelectedMonth] = useState(() => {
    const now = new Date()
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  })

  useEffect(() => {
    api.summary(24).then(setData).catch((e) => setErr(e.message))
    api.getSetting('dashboard')
      .then((s) => setLayout(Array.isArray(s.value) ? s.value : DEFAULT_LAYOUT))
      .catch(() => setLayout(DEFAULT_LAYOUT))
  }, [])

  useEffect(() => () => clearTimeout(saveTimer.current), [])

  const persist = useCallback((next) => {
    const revision = ++saveRevision.current
    clearTimeout(saveTimer.current)
    setErr(null)
    setLayout(next)
    setSaveState('saving')
    saveQueue.current = saveQueue.current.catch(() => {}).then(() => api.putSetting('dashboard', next))
    saveQueue.current.then(() => {
      if (revision !== saveRevision.current) return
      setSaveState('saved')
      saveTimer.current = setTimeout(() => setSaveState('idle'), 1500)
    }).catch(e => { if (revision === saveRevision.current) { setErr(`Disposition non enregistrée : ${e.message}`); setSaveState('idle') } })
  }, [])

  const move = (i, dir) => {
    const next = [...layout]
    const j = i + dir
    if (j < 0 || j >= next.length) return
    ;[next[i], next[j]] = [next[j], next[i]]
    persist(next)
  }
  const remove = (i) => persist(layout.filter((_, k) => k !== i))
  const resize = (i) => persist(layout.map((w, k) => (k === i ? { ...w, size: w.size === 'full' ? 'half' : 'full' } : w)))
  const setConfig = (i, config) => persist(layout.map((w, k) => (k === i ? { ...w, config } : w)))
  const add = (type) => {
    const def = WIDGETS[type]
    persist([...layout, { id: newId(), type, size: def.defaultSize, config: { ...def.config } }])
  }
  const reset = () => persist(DEFAULT_LAYOUT)

  if (err && !data) return <div className="notice err">{err}</div>
  if (!data || !layout) return <p style={{ color: 'var(--muted)' }}><span className="spinner" />Chargement…</p>

  const header = (
      <header className="dashboard-heading">
        <div>
          <p className="eyebrow">Ton espace de pilotage</p>
          <h1><span className="motion-title-line"><m.span initial={reduced ? false : { y: '110%' }} animate={{ y: 0 }} transition={{ duration: reduced ? 0 : .72, ease }}>La vue <em>d’ensemble.</em></m.span></span></h1>
          <p className="dashboard-intro">Les chiffres essentiels. Le recul pour décider.</p>
        </div>
        <MonthSelector selectedMonth={selectedMonth} onMonthChange={setSelectedMonth} />
      </header>
  )

  if ((data.monthly ?? []).length === 0) {
    return (
      <CinematicDashboard header={header}><div className="empty" style={{ marginTop: 24 }}>
        <p><b>Aucune donnée pour l'instant.</b></p>
        <p>Dépose ton premier relevé PDF dans la fente ci-dessus, ou{' '}
          <button className="link" onClick={onGoUpload}>choisis un fichier</button>.</p>
      </div></CinematicDashboard>
    )
  }

  const available = Object.entries(WIDGETS)

  return (
    <CinematicDashboard header={header}>
      {err && <div className="notice err" role="alert">{err}</div>}

      <MotionSurface className="cinematic-summary panel-level-hero" revealKey="summary"><FinancialSummary history={data.monthly} selectedMonth={selectedMonth} onInspectCoverage={month => onNavigate?.('statements', { month })} /></MotionSurface>
      <MotionSurface delay={80} revealKey="kpis"><KpiRow selectedMonth={selectedMonth} /></MotionSurface>
      <MotionSurface className="panel-level-editorial" delay={160} revealKey="journal"><Journal onNavigate={onNavigate} selectedMonth={selectedMonth} /></MotionSurface>

      <div className="dash-toolbar">
        <div className="dashboard-section-title"><span className="eyebrow">Explorer tes finances</span><h2>Les détails font la différence.</h2><span className="dashboard-save-state" role="status">{saveState === 'saving' ? 'Enregistrement…' : saveState === 'saved' ? 'Disposition enregistrée ✓' : ''}</span></div>
        <button className="btn-secondary" onClick={() => setEditing((e) => !e)}>
          {editing ? 'Terminer' : 'Personnaliser'}
        </button>
      </div>

      <AnimatePresence>{editing && (
        <PresencePanel className="widget-palette" initial={reduced ? false : { opacity: 0, height: 0 }} animate={{ opacity: 1, height: 'auto' }} exit={{ opacity: 0, height: 0 }} transition={{ duration: reduced ? 0 : .2, ease }} style={{ overflow: 'hidden' }}>
          <span className="muted">Ajouter un widget :</span>
          {available.map(([type, def]) => (
            <button key={type} className="btn-secondary" onClick={() => add(type)}>＋ {def.title}</button>
          ))}
          <button className="link" onClick={reset}>réinitialiser</button>
        </PresencePanel>
      )}</AnimatePresence>

      <div className="widget-grid">
        {layout.map((w, i) => {
          const def = WIDGETS[w.type]
          if (!def) return null
          const C = def.component
          const panelLevel = w.type === 'flux' || w.type === 'attention' ? 'focus' : ['top', 'subs', 'budgets'].includes(w.type) ? 'quiet' : 'standard'
          return (
            <MotionSurface key={w.id} revealKey={`widget-${w.id}`} layout="position" delay={Math.min(i, 4) * 70} className={`widget ${w.size} widget-${w.type} panel-level-${panelLevel}`} style={{ '--i': i }}>
              <div className="widget-head">
                <h3>{def.title}</h3>
                {editing && (
                  <div className="widget-actions">
                    <button onClick={() => move(i, -1)} disabled={i === 0} title="Monter">↑</button>
                    <button onClick={() => move(i, 1)} disabled={i === layout.length - 1} title="Descendre">↓</button>
                    <button onClick={() => resize(i)} title="Demi / pleine largeur">
                      {w.size === 'full' ? '◫' : '⬜'}
                    </button>
                    <button onClick={() => remove(i)} title="Retirer" className="danger">✕</button>
                  </div>
                )}
              </div>
              <C data={data} config={w.config} editing={editing} onConfig={(c) => setConfig(i, c)} onNavigate={onNavigate} selectedMonth={selectedMonth} />
            </MotionSurface>
          )
        })}
      </div>
    </CinematicDashboard>
  )
}
