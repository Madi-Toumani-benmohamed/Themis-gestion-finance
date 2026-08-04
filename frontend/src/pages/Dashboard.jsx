import { useCallback, useEffect, useState } from 'react'
import { api, eur, kindIcon } from '../api'
import { WIDGETS, DEFAULT_LAYOUT } from '../widgets'
import { Reveal } from '../Reveal'
import { Logo } from '../Logo'

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
const TONE_LABEL = {
  positive: 'Tout va bien',
  neutral: 'Rien à signaler',
  attention: 'À surveiller',
  alert: 'Demande une action',
}

function Hero() {
  const [h, setH] = useState(null)
  useEffect(() => { api.hero().then(setH).catch(() => setH({ available: false })) }, [])
  if (!h) return null
  if (!h.available) return null
  const fr = (d) => d ? new Date(d).toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' }) : null
  const icon = { positive: '✓', neutral: '·', attention: '△', alert: '✕' }
  return (
    <section className={`hero hero-${h.tone}`}>
      <span className={`hero-balance hero-balance-${h.tone}`} aria-hidden="true">
        <Logo size={104} />
      </span>
      <span className={`hero-tone tone-${h.tone}`}>{TONE_LABEL[h.tone]}</span>
      <div className="hero-headline">{h.headline}</div>
      {(h.points ?? []).length > 0 && (
        <ul className="hero-points">
          {h.points.map((p, i) => (
            <li key={i} className={`hero-point tone-${p.tone}`} style={{ '--n': i }}>
              <span className="hero-point-icon" aria-hidden="true">{icon[p.tone] || '·'}</span>
              {p.text}
            </li>
          ))}
        </ul>
      )}
      <div className="hero-meta">
        {h.coverage?.last_tx && <span>Données jusqu'au <strong>{fr(h.coverage.last_tx)}</strong></span>}
        {h.coverage?.last_import && <span className="coverage-sep">·</span>}
        {h.coverage?.last_import && <span>dernier import le {fr(h.coverage.last_import)}</span>}
        {h.coverage?.stale && <span className="hero-stale">— un relevé récent manque peut-être</span>}
      </div>
    </section>
  )
}

// Indicateurs clés : les quatre chiffres essentiels du mois de référence.
function KpiRow() {
  const [kpi, setKpi] = useState(null)
  useEffect(() => {
    Promise.all([api.summary(2), api.remaining().catch(() => null)]).then(([s2, rav]) => {
      const m = s2.monthly?.at(-1)
      if (!m) return setKpi(null)
      const prev = s2.monthly?.at(-2)
      const deltaPct = (cur, was) => {
        if (was == null || was === 0) return null
        return Math.round(((cur - was) / Math.abs(was)) * 100)
      }
      setKpi({
        income: m.income, spent: m.expenses, net: m.income - m.expenses,
        rav: rav && rav.available ? rav.remaining : null,
        spentDelta: prev ? deltaPct(m.expenses, prev.expenses) : null,
        incomeDelta: prev ? deltaPct(m.income, prev.income) : null,
      })
    }).catch(() => setKpi(null))
  }, [])
  if (!kpi) return null
  const delta = (d) => d == null ? null : (
    <span className="kpi-delta">{d > 0 ? '+' : ''}{d}%<span>vs mois préc.</span></span>
  )
  const cell = (label, value, tone, deltaEl) => (
    <div className={`kpi ${tone || ''}`}>
      <div className="kpi-label">{label}</div>
      <div className="kpi-value">{eur(value)}</div>
      {deltaEl}
    </div>
  )
  return (
    <div className="kpi-row">
      {cell('Revenus', kpi.income, '', delta(kpi.incomeDelta))}
      {cell('Dépenses', kpi.spent, '', delta(kpi.spentDelta))}
      {cell('Épargne du mois', kpi.net, kpi.net >= 0 ? 'kpi-pos' : 'kpi-neg')}
      {kpi.rav != null && cell('Reste à vivre estimé', kpi.rav, kpi.rav < 0 ? 'kpi-neg' : '')}
    </div>
  )
}

function Journal({ onNavigate }) {
  const [events, setEvents] = useState(null)
  useEffect(() => { api.journal().then((d) => setEvents(d.events || [])).catch(() => setEvents([])) }, [])
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
          {e.amount ? <span className={`jevent-amount ${e.amount > 0 ? 'pos' : 'neg'}`}>{e.amount > 0 ? '+' : ''}{Math.round(e.amount)} €</span> : null}
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
  const [data, setData] = useState(null)
  const [err, setErr] = useState(null)
  const [layout, setLayout] = useState(null)
  const [editing, setEditing] = useState(false)
  const [saveState, setSaveState] = useState('idle') // idle | saving | saved

  useEffect(() => {
    api.summary(24).then(setData).catch((e) => setErr(e.message))
    api.getSetting('dashboard')
      .then((s) => setLayout(Array.isArray(s.value) && s.value.length ? s.value : DEFAULT_LAYOUT))
      .catch(() => setLayout(DEFAULT_LAYOUT))
  }, [])

  const persist = useCallback((next) => {
    setLayout(next)
    setSaveState('saving')
    api.putSetting('dashboard', next)
      .then(() => { setSaveState('saved'); setTimeout(() => setSaveState('idle'), 1500) })
      .catch((e) => { setErr(e.message); setSaveState('idle') })
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

  if ((data.monthly ?? []).length === 0) {
    return (
      <div className="empty" style={{ marginTop: 24 }}>
        <p><b>Aucune donnée pour l'instant.</b></p>
        <p>Dépose ton premier relevé PDF dans la fente ci-dessus, ou{' '}
          <button className="link" onClick={onGoUpload}>choisis un fichier</button>.</p>
      </div>
    )
  }

  const available = Object.entries(WIDGETS)

  return (
    <div className="section">
      {err && <div className="notice err">{err}</div>}

      <Hero />
      <Reveal delay={80}><KpiRow /></Reveal>
      <Reveal delay={160}><Journal onNavigate={onNavigate} /></Reveal>

      <div className="dash-toolbar">
        <span className="muted" style={{ fontSize: 13 }}>
          {saveState === 'saving' && 'enregistrement…'}
          {saveState === 'saved' && 'disposition enregistrée ✓'}
        </span>
        <button className="btn-secondary" onClick={() => setEditing((e) => !e)}>
          {editing ? 'Terminer' : 'Personnaliser'}
        </button>
      </div>

      {editing && (
        <div className="widget-palette">
          <span className="muted">Ajouter un widget :</span>
          {available.map(([type, def]) => (
            <button key={type} className="btn-secondary" onClick={() => add(type)}>＋ {def.title}</button>
          ))}
          <button className="link" onClick={reset}>réinitialiser</button>
        </div>
      )}

      <div className="widget-grid">
        {layout.map((w, i) => {
          const def = WIDGETS[w.type]
          if (!def) return null
          const C = def.component
          return (
            <div key={w.id} className={`widget ${w.size}`} style={{ '--i': i }}>
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
              <C data={data} config={w.config} editing={editing} onConfig={(c) => setConfig(i, c)} onNavigate={onNavigate} />
            </div>
          )
        })}
      </div>
    </div>
  )
}
