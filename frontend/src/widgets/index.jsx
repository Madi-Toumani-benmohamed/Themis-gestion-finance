import { useEffect, useMemo, useState } from 'react'
import {
  ResponsiveContainer, CartesianGrid, XAxis, YAxis, Tooltip, Legend, ReferenceLine,
  ComposedChart, Area, LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, Brush,
} from 'recharts'
import { api, eur, monthLabel } from '../api'

// Couleurs : lues depuis les variables CSS pour suivre le thème clair/sombre.
// On les relit à chaque rendu de widget via readColors().
function readColors() {
  if (typeof window === 'undefined') return {}
  const s = getComputedStyle(document.documentElement)
  const v = (n, fb) => (s.getPropertyValue(n).trim() || fb)
  return {
    INK: v('--ink', '#16161D'),
    GREEN: v('--pos', '#21935B'),
    RED: v('--neg', '#D6455D'),
    MUTED: v('--muted', '#8A8A99'),
    LINE: v('--line', '#ECECF1'),
    ACCENT: v('--accent', '#5B5BD6'),
    PAPER: v('--paper', '#FFFFFF'),
  }
}

// Valeurs par défaut (clair) pour le premier rendu synchrone.
const INK = '#16161D'
const GREEN = '#21935B'
const RED = '#D6455D'
const MUTED = '#8A8A99'
const LINE = '#ECECF1'
const ACCENT = '#5B5BD6'

/* Compteur déroulant : les montants "montent" à l'affichage.
   Respecte prefers-reduced-motion. */
function useCountUp(target, duration = 750) {
  const [v, setV] = useState(0)
  useEffect(() => {
    if (typeof target !== 'number' || !isFinite(target)) { setV(target); return }
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) { setV(target); return }
    let raf
    const t0 = performance.now()
    const tick = (t) => {
      const p = Math.min((t - t0) / duration, 1)
      const eased = 1 - Math.pow(1 - p, 3)
      setV(target * eased)
      if (p < 1) raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [target, duration])
  return v
}

function Roll({ value, fmt }) {
  const v = useCountUp(value)
  return <>{fmt(v)}</>
}

// Dégradé vertical réutilisable : plein en haut, transparent en bas — l'effet
// « aire lumineuse » caractéristique des graphiques soignés.
function softGradient(id, color, top = 0.28, bottom = 0.02) {
  return (
    <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stopColor={color} stopOpacity={top} />
      <stop offset="100%" stopColor={color} stopOpacity={bottom} />
    </linearGradient>
  )
}

// Tooltip épuré, lisible en clair comme en sombre (fond = --paper).
function ChartTooltip({ active, payload, label, colors }) {
  if (!active || !payload || !payload.length) return null
  return (
    <div style={{
      background: colors.PAPER, borderRadius: 14,
      boxShadow: '0 12px 40px rgba(22,22,29,.16), 0 2px 8px rgba(22,22,29,.06)',
      padding: '10px 14px', fontFamily: 'Inter, sans-serif', fontSize: 12.5,
      border: `1px solid ${colors.LINE}`,
    }}>
      {label != null && <div style={{ color: colors.MUTED, marginBottom: 4, fontWeight: 500 }}>{label}</div>}
      {payload.map((p, i) => (
        <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, color: colors.INK, fontWeight: 600 }}>
          <span style={{ width: 8, height: 8, borderRadius: 9, background: p.color || p.stroke || p.fill }} />
          <span style={{ flex: 1 }}>{p.name}</span>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>{eur(p.value)}</span>
        </div>
      ))}
    </div>
  )
}

const fmtTick = (v) => (Math.abs(v) >= 1000 ? `${Math.round(v / 1000)}k` : v)
// Axes épurés façon Apple : pas de ligne d'axe, pas de trait de tick, texte discret.
const axisX = (c) => ({ tick: { fontSize: 11, fill: c.MUTED }, tickLine: false, axisLine: false, dy: 6, minTickGap: 24 })
const axisY = (c, w = 40) => ({ tickFormatter: fmtTick, tick: { fontSize: 11, fill: c.MUTED }, tickLine: false, axisLine: false, width: w })
const gridProps = (c) => ({ stroke: c.LINE, strokeOpacity: 0.5, vertical: false })

/* ------------------------------------------------------------------ */
/* Widget : flux — revenus vs dépenses (aires interactives)            */
/* ------------------------------------------------------------------ */
export function FluxWidget({ data, config }) {
  const [hidden, setHidden] = useState({})
  const series = (data.monthly ?? []).slice(-(config.months || 12)).map((m) => ({
    ...m, net: +(m.income - m.expenses).toFixed(2), label: monthLabel(m.month),
  }))
  const toggle = (e) => setHidden((h) => ({ ...h, [e.dataKey]: !h[e.dataKey] }))
  if (series.length === 0) return <Empty />
  const c = readColors()
  return (
    <ResponsiveContainer width="100%" height={260}>
      <ComposedChart data={series} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <defs>
          {softGradient('grad-income', c.GREEN)}
          {softGradient('grad-expenses', c.RED)}
        </defs>
        <CartesianGrid {...gridProps(c)} />
        <XAxis dataKey="label" {...axisX(c)} />
        <YAxis {...axisY(c, 40)} />
        <Tooltip content={<ChartTooltip colors={c} />} />
        <Legend onClick={toggle} iconType="circle" wrapperStyle={{ fontSize: 12, cursor: 'pointer', paddingTop: 6 }} />
        <Area animationDuration={700} name="Revenus" type="monotone" dataKey="income" hide={hidden.income} stroke={c.GREEN} fill="url(#grad-income)" strokeWidth={2.5} dot={false} activeDot={{ r: 4, strokeWidth: 0 }} />
        <Area animationDuration={700} name="Dépenses" type="monotone" dataKey="expenses" hide={hidden.expenses} stroke={c.RED} fill="url(#grad-expenses)" strokeWidth={2.5} dot={false} activeDot={{ r: 4, strokeWidth: 0 }} />
        <Line animationDuration={700} name="Net" type="monotone" dataKey="net" hide={hidden.net} stroke={c.ACCENT} strokeWidth={2.5} dot={false} activeDot={{ r: 4, strokeWidth: 0 }} />
        {series.length > 14 && <Brush height={18} stroke={c.MUTED} travellerWidth={8} />}
      </ComposedChart>
    </ResponsiveContainer>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : solde net cumulé                                           */
/* ------------------------------------------------------------------ */
export function SoldeWidget({ data, config }) {
  const series = useMemo(() => {
    let cum = 0
    return (data.monthly ?? []).slice(-(config.months || 12)).map((m) => {
      cum += m.income - m.expenses
      return { label: monthLabel(m.month), cumul: +cum.toFixed(2) }
    })
  }, [data, config.months])
  if (series.length === 0) return <Empty />
  const c = readColors()
  return (
    <ResponsiveContainer width="100%" height={220}>
      <ComposedChart data={series} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <defs>{softGradient('grad-cumul', c.ACCENT, 0.22, 0.01)}</defs>
        <CartesianGrid {...gridProps(c)} />
        <XAxis dataKey="label" {...axisX(c)} />
        <YAxis {...axisY(c, 44)} />
        <Tooltip content={<ChartTooltip colors={c} />} />
        <ReferenceLine y={0} stroke={c.MUTED} strokeOpacity={0.4} strokeDasharray="4 4" />
        <Area animationDuration={700} type="monotone" name="Cumul net" dataKey="cumul" stroke={c.ACCENT} fill="url(#grad-cumul)" strokeWidth={2.5} dot={false} activeDot={{ r: 4, strokeWidth: 0 }} />
      </ComposedChart>
    </ResponsiveContainer>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : donut des dépenses par catégorie (mois le plus récent)     */
/* ------------------------------------------------------------------ */
export function CategoriesWidget({ data }) {
  const [active, setActive] = useState(null)
  const month = data.monthly?.at(-1)?.month
  const slices = (data.by_category ?? []).filter((c) => c.month === month && c.total > 0)
  const total = slices.reduce((s, c) => s + c.total, 0)
  if (slices.length === 0) return <Empty />
  const cc = readColors()
  return (
    <div className="donut-wrap">
      <ResponsiveContainer width="100%" height={220}>
        <PieChart>
          <Pie animationDuration={650}
            data={slices} dataKey="total" nameKey="category"
            innerRadius="64%" outerRadius="88%" paddingAngle={3} stroke="none" cornerRadius={6}
            onMouseEnter={(_, i) => setActive(i)} onMouseLeave={() => setActive(null)}
          >
            {slices.map((c, i) => (
              <Cell key={i} fill={c.color || cc.MUTED} opacity={active === null || active === i ? 1 : 0.3}
                style={{ transition: 'opacity .2s ease' }} />
            ))}
          </Pie>
          <Tooltip content={({ active: a, payload }) => {
            if (!a || !payload?.length) return null
            const p = payload[0]
            return (
              <div style={{ background: cc.PAPER, borderRadius: 14, border: `1px solid ${cc.LINE}`,
                boxShadow: '0 12px 40px rgba(22,22,29,.16)', padding: '10px 14px', fontFamily: 'Inter, sans-serif', fontSize: 12.5 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: cc.INK, fontWeight: 600 }}>
                  <span style={{ width: 8, height: 8, borderRadius: 9, background: p.payload.color || cc.MUTED }} />
                  <span style={{ flex: 1 }}>{p.name}</span>
                  <span>{eur(p.value)} · {Math.round((p.value / total) * 100)} %</span>
                </div>
              </div>
            )
          }} />
        </PieChart>
      </ResponsiveContainer>
      <div className="donut-center">
        <div className="donut-total">{eur(active !== null ? slices[active].total : total)}</div>
        <div className="donut-label">{active !== null ? slices[active].category : monthLabel(month)}</div>
      </div>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : tendance d'une catégorie                                   */
/* ------------------------------------------------------------------ */
export function CategoryTrendWidget({ data, config, editing, onConfig }) {
  const names = useMemo(
    () => [...new Set((data.by_category ?? []).map((c) => c.category))].sort(),
    [data],
  )
  const cat = config.category || names[0]
  const months = (data.monthly ?? []).slice(-(config.months || 12)).map((m) => m.month)
  const series = months.map((m) => ({
    label: monthLabel(m),
    total: +((data.by_category ?? []).find((c) => c.month === m && c.category === cat)?.total ?? 0).toFixed(2),
  }))
  const color = (data.by_category ?? []).find((c) => c.category === cat)?.color || GREEN
  const cc = readColors()
  return (
    <div>
      <div className="widget-config">
        <select value={cat || ''} onChange={(e) => onConfig({ ...config, category: e.target.value })}>
          {names.map((n) => <option key={n} value={n}>{n}</option>)}
        </select>
      </div>
      {series.every((s) => s.total === 0) ? <Empty /> : (
        <ResponsiveContainer width="100%" height={190}>
          <BarChart data={series} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
            <defs>{softGradient('grad-bar', color, 0.95, 0.55)}</defs>
            <CartesianGrid {...gridProps(cc)} />
            <XAxis dataKey="label" {...axisX(cc)} />
            <YAxis {...axisY(cc, 40)} />
            <Tooltip content={<ChartTooltip colors={cc} />} cursor={{ fill: cc.LINE, opacity: 0.3, radius: 6 }} />
            <Bar animationDuration={600} dataKey="total" name={cat} fill="url(#grad-bar)" maxBarSize={34} radius={[6, 6, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      )}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : cartes KPI du mois                                         */
/* ------------------------------------------------------------------ */
export function KpiWidget({ data }) {
  const cur = data.monthly?.at(-1)
  const prev = data.monthly?.at(-2)
  if (!cur) return <Empty />
  const net = cur.income - cur.expenses
  const savings = cur.income > 0 ? (net / cur.income) * 100 : null
  const delta = (a, b) => (b ? ((a - b) / b) * 100 : null)
  const kpis = [
    { label: 'Revenus', num: cur.income, fmt: eur, d: delta(cur.income, prev?.income), goodUp: true },
    { label: 'Dépenses', num: cur.expenses, fmt: eur, d: delta(cur.expenses, prev?.expenses), goodUp: false },
    { label: 'Net', num: net, fmt: eur, cls: net >= 0 ? 'credit' : 'debit' },
    { label: "Taux d'épargne", num: savings, fmt: (v) => (v === null ? '—' : `${v.toFixed(0)} %`), cls: savings >= 0 ? 'credit' : 'debit' },
  ]
  return (
    <div className="kpi-grid">
      {kpis.map((k) => (
        <div className="kpi-card" key={k.label}>
          <div className="label">{k.label}</div>
          <div className={'value ' + (k.cls || '')}>
            {k.num === null ? '—' : <Roll value={k.num} fmt={k.fmt} />}
          </div>
          {k.d !== null && k.d !== undefined && (
            <div className={'delta ' + ((k.d >= 0) === k.goodUp ? 'credit' : 'debit')}>
              {k.d >= 0 ? '▲' : '▼'} {Math.abs(k.d).toFixed(0)} % vs mois préc.
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : top dépenses du mois                                       */
/* ------------------------------------------------------------------ */
export function TopWidget({ config }) {
  const [rows, setRows] = useState(null)
  useEffect(() => {
    // Les transferts internes (virements entre comptes personnels) ne sont
    // pas des dépenses : on les exclut du top.
    Promise.all([api.summary(1), api.categories()]).then(([d, cats]) => {
      const month = d.monthly?.at(-1)?.month
      if (!month) return setRows([])
      const transferIds = new Set((cats || []).filter((c) => c.is_transfer).map((c) => c.id))
      api.transactions({ month }).then((txs) => {
        setRows(txs.filter((t) => t.amount < 0 && !transferIds.has(t.domain_id))
          .sort((a, b) => a.amount - b.amount)
          .slice(0, config.count || 5))
      })
    }).catch(() => setRows([]))
  }, [config.count])
  if (rows === null) return <Loading />
  if (rows.length === 0) return <Empty />
  const max = -rows[0].amount
  return (
    <div className="top-list">
      {rows.map((t) => (
        <div className="top-row" key={t.id}>
          <div className="top-bar" style={{ width: `${(-t.amount / max) * 100}%` }} />
          <span className="top-label">{(t.display_label || t.label).slice(0, 46)}</span>
          <span className="top-amount">{eur(-t.amount)}</span>
        </div>
      ))}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : budgets (mini-jauges)                                      */
/* ------------------------------------------------------------------ */
export function BudgetsWidget() {
  const [budgets, setBudgets] = useState(null)
  useEffect(() => { api.budgets().then(setBudgets).catch(() => setBudgets([])) }, [])
  if (budgets === null) return <Loading />
  if (budgets.length === 0) return <Empty text="Aucun budget — onglet Budgets pour en créer." />
  return (
    <div className="mini-budgets">
      {budgets.slice(0, 4).map((b) => (
        <div key={b.id} className="mini-budget">
          <div className="mini-budget-head">
            <span>{b.category_name}</span>
            <span className={b.ratio >= 1 ? 'debit' : ''}>{Math.round(b.ratio * 100)} %</span>
          </div>
          <div className="gauge"><div
            className={'gauge-fill' + (b.ratio >= 1 ? ' over' : b.ratio >= 0.85 ? ' warn' : '')}
            style={{ width: Math.min(b.ratio * 100, 100) + '%' }}
          /></div>
        </div>
      ))}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : abonnements                                                */
/* ------------------------------------------------------------------ */
export function SubsWidget() {
  const [subs, setSubs] = useState(null)
  useEffect(() => {
    api.recurring().then((rs) => setSubs(rs.filter((r) => r.regular))).catch(() => setSubs([]))
  }, [])
  if (subs === null) return <Loading />
  if (subs.length === 0) return <Empty text="Aucun abonnement détecté (2 mois de relevés minimum)." />
  const total = subs.reduce((s, r) => s + r.avg_amount, 0)
  return (
    <div>
      <div className="subs-total"><Roll value={total} fmt={eur} /> <span className="muted">/ mois</span></div>
      <div className="top-list">
        {subs.slice(0, 6).map((r, i) => (
          <div className="top-row" key={i}>
            <div className="top-bar" style={{ width: `${(r.avg_amount / subs[0].avg_amount) * 100}%` }} />
            <span className="top-label">{r.label.slice(0, 40)}</span>
            <span className="top-amount">{eur(r.avg_amount)}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

const Empty = ({ text = 'Pas encore de données — importe un relevé.' }) =>
  <div className="widget-empty">{text}</div>
const Loading = () => <div className="widget-empty"><span className="spinner" />Chargement…</div>

/* ------------------------------------------------------------------ */
/* Registre                                                            */
/* ------------------------------------------------------------------ */
export function AttentionWidget() {
  const [items, setItems] = useState(null)
  useEffect(() => {
    api.anomalies().then((d) => setItems(d.anomalies || [])).catch(() => setItems([]))
  }, [])
  if (items === null) return <Loading />
  if (items.length === 0) return <Empty text="Rien à signaler — aucune anomalie détectée." />
  const icon = { subscription_up: '↗', unusual: '!', new_recurring: '+', subscription_gone: '×' }
  return (
    <div className="attn-list">
      {items.slice(0, 6).map((a, i) => (
        <div className={`attn-item attn-${a.severity}`} key={i}>
          <span className="attn-icon">{icon[a.kind] || '•'}</span>
          <div className="attn-body">
            <div className="attn-title">{a.title}</div>
            <div className="attn-detail">{a.detail}</div>
          </div>
        </div>
      ))}
    </div>
  )
}

export const WIDGETS = {
  kpi:        { title: 'Indicateurs du mois',        component: KpiWidget,           defaultSize: 'full', config: {} },
  flux:       { title: 'Revenus & dépenses',         component: FluxWidget,          defaultSize: 'full', config: { months: 12 } },
  solde:      { title: 'Cumul net',                  component: SoldeWidget,         defaultSize: 'half', config: { months: 12 } },
  categories: { title: 'Dépenses par catégorie',     component: CategoriesWidget,    defaultSize: 'half', config: {} },
  trend:      { title: 'Tendance d\u2019une catégorie', component: CategoryTrendWidget, defaultSize: 'half', config: { months: 12 } },
  top:        { title: 'Top dépenses du mois',       component: TopWidget,           defaultSize: 'half', config: { count: 5 } },
  budgets:    { title: 'Budgets',                    component: BudgetsWidget,       defaultSize: 'half', config: {} },
  subs:       { title: 'Abonnements',                component: SubsWidget,          defaultSize: 'half', config: {} },
  attention:  { title: 'Ça mérite ton attention',    component: AttentionWidget,     defaultSize: 'half', config: {} },
}

export const DEFAULT_LAYOUT = [
  { id: 'w1', type: 'kpi', size: 'full', config: {} },
  { id: 'w2', type: 'flux', size: 'full', config: { months: 12 } },
  { id: 'wa', type: 'attention', size: 'half', config: {} },
  { id: 'w3', type: 'categories', size: 'half', config: {} },
  { id: 'w4', type: 'solde', size: 'half', config: { months: 12 } },
  { id: 'w5', type: 'top', size: 'half', config: { count: 5 } },
  { id: 'w6', type: 'subs', size: 'half', config: {} },
]
