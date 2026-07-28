import { useEffect, useState } from 'react'
import { api, eur } from '../api'

const fr0 = (n) => new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n || 0)
const frDate = (d) => d ? new Date(d).toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' }) : ''

// ── Vue 1 : nature des sorties (dépenses réelles / obligatoire / épargne) ──
export function NatureBreakdown() {
  const [n, setN] = useState(null)
  useEffect(() => { api.intelNature().then(setN).catch(() => setN({ available: false })) }, [])
  if (!n || !n.available) return null
  const parts = [
    { key: 'real', label: 'Dépenses réelles', value: n.real_spend, cls: 'nat-real' },
    { key: 'mand', label: 'Impôts & cotisations', value: n.mandatory, cls: 'nat-mand' },
    { key: 'save', label: 'Épargne', value: n.savings, cls: 'nat-save' },
  ].filter((p) => p.value > 0)
  const total = n.total_out || 1
  return (
    <div className="intel-card">
      <div className="section-head">
        <h3>Où va ton argent ce mois-ci</h3>
        <span className="section-sub">tout ce qui sort n'est pas une dépense</span>
      </div>
      <div className="nature-bar">
        {parts.map((p) => (
          <div key={p.key} className={`nature-seg ${p.cls}`} style={{ width: `${(p.value / total) * 100}%` }} title={`${p.label} : ${fr0(p.value)}`} />
        ))}
      </div>
      <div className="nature-legend">
        {parts.map((p) => (
          <div key={p.key} className="nature-item">
            <span className={`nature-dot ${p.cls}`} />
            <span className="nature-label">{p.label}</span>
            <span className="nature-val">{fr0(p.value)}</span>
          </div>
        ))}
      </div>
      {n.savings > 0 && (
        <p className="intel-note">
          Sur {fr0(n.total_out)} sortis, {fr0(n.savings)} sont de l'épargne — de l'argent qui reste à toi.
        </p>
      )}
    </div>
  )
}

// ── Vue 2 : par marchand (chez qui, combien, tendance) ──
export function MerchantsView() {
  const [data, setData] = useState(null)
  const [months, setMonths] = useState(3)
  useEffect(() => { api.intelMerchants(months).then(setData).catch(() => setData(null)) }, [months])
  if (!data || !data.merchants?.length) return null
  const max = Math.max(...data.merchants.map((m) => m.total), 1)
  return (
    <div className="intel-card">
      <div className="section-head">
        <h3>Chez qui tu dépenses</h3>
        <div className="chart-controls">
          {[1, 3, 6].map((m) => (
            <button key={m} className={`seg-btn ${months === m ? 'active' : ''}`} onClick={() => setMonths(m)}>
              {m} mois
            </button>
          ))}
        </div>
      </div>
      <ul className="merchant-list">
        {data.merchants.slice(0, 12).map((m) => (
          <li key={m.name} className="merchant-row">
            <div className="merchant-head">
              <span className="merchant-name">{m.name}</span>
              <span className="merchant-total">{fr0(m.total)}</span>
            </div>
            <div className="merchant-bar-track">
              <div className="merchant-bar" style={{ width: `${(m.total / max) * 100}%` }} />
            </div>
            <div className="merchant-meta">
              <span>{m.visits} fois · panier {fr0(m.avg_basket)}</span>
              {m.trend_pct != null && (
                <span className={`merchant-trend ${m.trend_pct > 0 ? 'up' : m.trend_pct < 0 ? 'down' : ''}`}>
                  {m.trend_pct > 0 ? '▲' : m.trend_pct < 0 ? '▼' : ''} {Math.abs(m.trend_pct)}% vs période précédente
                </span>
              )}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

// ── Vue 3 : comparaison par catégorie (ce mois vs habitude) ──
export function CategoryComparison() {
  const [data, setData] = useState(null)
  useEffect(() => { api.intelComparison().then(setData).catch(() => setData(null)) }, [])
  if (!data || !data.categories?.length) return null
  const rows = data.categories.filter((c) => c.current > 0 || c.average > 0)
  return (
    <div className="intel-card">
      <div className="section-head">
        <h3>Ce mois-ci vs d'habitude</h3>
        <span className="section-sub">par catégorie, comparé à ta moyenne</span>
      </div>
      <ul className="cmp-list">
        {rows.map((c) => (
          <li key={c.name} className="cmp-row">
            <span className="cmp-name">{c.name}</span>
            <span className="cmp-vals">{fr0(c.current)} <span className="cmp-avg">vs {fr0(c.average)}</span></span>
            {c.delta_pct != null ? (
              <span className={`cmp-delta ${c.delta_pct > 15 ? 'up' : c.delta_pct < -15 ? 'down' : 'flat'}`}>
                {c.delta_pct > 0 ? '+' : ''}{c.delta_pct}%
              </span>
            ) : (
              <span className="cmp-delta new">nouveau</span>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}

// ── Vague 3 : synthèse IA (raisonne sur les faits des vagues 1 et 2) ──
export function Narrative() {
  const [n, setN] = useState(null)
  const [loading, setLoading] = useState(false)
  const run = () => {
    setLoading(true)
    api.intelNarrative().then(setN).catch(() => setN({ available: false, reason: 'Erreur' })).finally(() => setLoading(false))
  }
  return (
    <div className="intel-card intel-narrative">
      <div className="section-head">
        <h3>Synthèse</h3>
        <button className="btn-secondary" onClick={run} disabled={loading}>
          {loading && <span className="spinner" />} {n ? 'Rafraîchir' : 'Analyser ma situation'}
        </button>
      </div>
      {!n && <p className="intel-note">Une lecture d'ensemble de ton mois, à partir des faits ci-dessous. L'IA commente, elle n'invente pas de chiffres.</p>}
      {n && n.available && <p className="narrative-text">{n.narrative}</p>}
      {n && !n.available && <p className="intel-note">{n.reason}</p>}
    </div>
  )
}

// ── Vague 2 : découvertes (motifs repérés automatiquement) ──
export function Discoveries({ onNavigate }) {
  const [data, setData] = useState(null)
  useEffect(() => { api.intelDiscoveries().then(setData).catch(() => setData(null)) }, [])
  if (!data || !data.discoveries?.length) return null
  const icon = { duplicate_subs: '⧉', price_creep: '▲', seasonal: '↻', merchant_drift: '↗', tag_surge: '#', tag_project: '◱' }
  return (
    <div className="intel-card">
      <div className="section-head">
        <h3>Ce que thémis a repéré</h3>
        <span className="section-sub">signaux possibles — à toi de juger</span>
      </div>
      <ul className="disc-list">
        {data.discoveries.map((d, i) => (
          <li key={i} className={`disc-item disc-${d.tone}`}>
            <span className="disc-icon" aria-hidden="true">{icon[d.kind] || '•'}</span>
            <div className="disc-body">
              <div className="disc-title">{d.title}</div>
              <p className="disc-detail">{d.detail}</p>
              {d.query && (
                <button className="disc-link" onClick={() => onNavigate?.('transactions', d.scope === 'month' && d.month ? { q: d.query, month: d.month } : { q: d.query })}>
                  {d.scope === 'month' ? 'Voir les transactions du mois →' : 'Voir les transactions →'}
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

export default function Intelligence({ onNavigate }) {
  return (
    <div className="intel-grid">
      <Narrative />
      <Discoveries onNavigate={onNavigate} />
      <NatureBreakdown />
      <CategoryComparison />
      <MerchantsView />
    </div>
  )
}
