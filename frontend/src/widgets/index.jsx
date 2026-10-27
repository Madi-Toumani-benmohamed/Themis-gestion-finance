import { memo, useCallback, useEffect, useMemo, useState } from 'react'
import {
  ResponsiveContainer, CartesianGrid, XAxis, YAxis, Tooltip, Legend, ReferenceLine,
  ComposedChart, Area, LineChart, Line, BarChart, Bar, PieChart, Pie, Cell, Brush,
} from 'recharts'
import { AnimatedNumber } from '../motion/AnimatedNumber'
import { AnimatedChart, ChartFocusDot } from '../motion/AnimatedChart'
import { ProgressFill } from '../motion'
import { api, eur, monthLabel } from '../api'
import { budgetWidgetModel } from '../components/budgetWidgetModel.mjs'
import { cents, euroCents } from '../components/money.mjs'
import { readColors, softGradient, ChartTooltip, axisX, axisY, gridProps } from '../components/ChartPrimitives'

// Valeurs par défaut (clair) pour le premier rendu synchrone.
const INK = '#16161D'
const GREEN = '#21935B'
const RED = '#D6455D'
const MUTED = '#8A8A99'
const LINE = '#ECECF1'
const ACCENT = '#5B5BD6'

/* ------------------------------------------------------------------ */
/* Widget : flux — revenus vs dépenses (aires interactives)            */
/* ------------------------------------------------------------------ */
// Graphique mémoïsé : il ne se redessine QUE si les données ou les couleurs
// changent — jamais quand la fenêtre de la brosse bouge. C'est la clé pour que
// la brosse reste où on la pose : si le graphe re-render, Recharts réinitialise
// la brosse. On isole donc le résumé (qui réagit au déplacement) du graphe.
const FluxChart = memo(function FluxChart({ all, c, initialStart, onBrush, currentMonthIdx, selectedMonthIdx }) {
  // Ligne verticale pour le mois courant
  const currentMonthLabel = currentMonthIdx >= 0 ? all[currentMonthIdx]?.label : null
  // Ligne verticale pour le mois sélectionné (si différent)
  const selectedMonthLabel = selectedMonthIdx >= 0 && selectedMonthIdx !== currentMonthIdx ? all[selectedMonthIdx]?.label : null
  return (
    <AnimatedChart revision={all}><ResponsiveContainer width="100%" height={300}>
      <ComposedChart data={all} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <defs>
          {softGradient('grad-income', c.GREEN)}
          {softGradient('grad-expenses', c.RED)}
        </defs>
        <CartesianGrid {...gridProps(c)} />
        <XAxis dataKey="label" {...axisX(c)} />
        <YAxis {...axisY(c, 40)} />
        <Tooltip content={<ChartTooltip colors={c} />} />
        <Legend iconType="circle" wrapperStyle={{ fontSize: 12, paddingTop: 6 }} />
        {currentMonthIdx >= 0 && (
          <ReferenceLine x={currentMonthLabel} stroke={c.ACCENT} strokeOpacity={0.4} strokeDasharray="4 4" label={{ position: 'top', offset: -10, fill: c.ACCENT, fontSize: 10, value: 'Mois en cours' }} />
        )}
        {selectedMonthLabel && (
          <ReferenceLine x={selectedMonthLabel} stroke={c.ACCENT} strokeOpacity={0.3} strokeDasharray="8 4" label={{ position: 'bottom', offset: 10, fill: c.ACCENT, fontSize: 9, value: 'Mois consulté' }} />
        )}
        <Area isAnimationActive={false} name="Revenus" type="monotone" dataKey="income" stroke={c.GREEN} fill="url(#grad-income)" strokeWidth={2.5} dot={false} activeDot={<ChartFocusDot />} />
        <Area isAnimationActive={false} name="Dépenses" type="monotone" dataKey="expenses" stroke={c.RED} fill="url(#grad-expenses)" strokeWidth={2.5} dot={false} activeDot={<ChartFocusDot />} />
        <Line isAnimationActive={false} name="Net" type="monotone" dataKey="net" stroke={c.ACCENT} strokeWidth={2.5} dot={false} activeDot={<ChartFocusDot />} />
        <Brush
          dataKey="label" height={28} stroke={c.ACCENT} fill={c.PAPER}
          travellerWidth={9} gap={1}
          startIndex={initialStart}
          onChange={onBrush}
        />
      </ComposedChart>
    </ResponsiveContainer></AnimatedChart>
  )
}, (prev, next) =>
  prev.all === next.all &&
  prev.initialStart === next.initialStart &&
  prev.currentMonthIdx === next.currentMonthIdx &&
  prev.selectedMonthIdx === next.selectedMonthIdx &&
  prev.c.GREEN === next.c.GREEN &&   // re-render seulement si le thème change
  prev.c.PAPER === next.c.PAPER
)

export function FluxWidget({ data, config, selectedMonth }) {
  // `all` DOIT garder une référence stable : sinon Recharts croit que les
  // données ont changé et réinitialise la brosse.
  const all = useMemo(() => (data.monthly ?? []).map((m) => ({
    ...m, net: +(m.income - m.expenses).toFixed(2), label: monthLabel(m.month),
  })), [data.monthly])

  // Identifier le mois courant
  const currentMonthIdx = all.findIndex(m => m.is_current || m.status === 'current')
  // Identifier le mois sélectionné (si différent du mois courant)
  const selectedMonthIdx = selectedMonth ? all.findIndex(m => m.month === selectedMonth) : -1

  const hasBrush = all.length > 14
  const initialStart = useMemo(
    () => (hasBrush ? Math.max(0, all.length - 12) : 0),
    [all.length, hasBrush]
  )

  // Fenêtre visible, pour le résumé. Mise à jour au déplacement de la brosse,
  // mais SANS forcer le re-render du graphe (grâce au memo ci-dessus).
  const [range, setRange] = useState(null)
  const onBrush = useCallback((r) => {
    if (!r || r.startIndex == null) return
    setRange((prev) =>
      prev && prev.startIndex === r.startIndex && prev.endIndex === r.endIndex
        ? prev : { startIndex: r.startIndex, endIndex: r.endIndex }
    )
  }, [])

  // Couleurs : mémoïsées pour ne pas donner une nouvelle référence au graphe à
  // chaque render (ce qui le ferait re-dessiner et réinitialiserait la brosse).
  const themeKey = (typeof document !== 'undefined' && document.documentElement.getAttribute('data-theme')) || 'light'
  const c = useMemo(() => readColors(), [themeKey])

  if (all.length === 0) return <Empty />

  const from = range?.startIndex ?? (hasBrush ? initialStart : 0)
  const to = range?.endIndex ?? (all.length - 1)
  const win = all.slice(from, to + 1)
  const avg = (k) => win.length ? win.reduce((s, m) => s + m[k], 0) / win.length : 0
  const span = hasBrush
    ? `${all[from]?.label} → ${all[to]?.label} · ${to - from + 1} mois`
    : `${all.length} mois`

  // Note pour la moyenne si le mois courant est dans la fenêtre
  const windowHasCurrent = currentMonthIdx >= from && currentMonthIdx <= to
  const avgNote = windowHasCurrent && currentMonthIdx === all.length - 1
    ? ' (incl. mois en cours partiel)'
    : ''

  return (
    <div className="flux-widget">
      <div className="flux-head">
        <span className="flux-span">{span}</span>
        {hasBrush && <span className="flux-hint">tire la brosse pour zoomer</span>}
      </div>
      <FluxChart all={all} c={c} initialStart={initialStart} onBrush={hasBrush ? onBrush : undefined} currentMonthIdx={currentMonthIdx} selectedMonthIdx={selectedMonthIdx} />
      <div className="flux-foot">
        Moyenne sur la fenêtre <b>{eur(avg('income'))}</b> de revenus · <b>{eur(avg('expenses'))}</b> de dépenses{avgNote}
      </div>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : solde net cumulé                                           */
/* ------------------------------------------------------------------ */
export function SoldeWidget({ data, config, selectedMonth }) {
  const monthly = (data.monthly ?? []).slice(-(config.months || 12))
  const currentMonthIdx = monthly.findIndex(m => m.is_current || m.status === 'current')
  const selectedMonthIdx = selectedMonth ? monthly.findIndex(m => m.month === selectedMonth) : -1
  const series = useMemo(() => {
    let cum = 0
    return monthly.map((m) => {
      cum += m.income - m.expenses
      return { label: monthLabel(m.month), cumul: +cum.toFixed(2) }
    })
  }, [data, config.months])
  if (series.length === 0) return <Empty />
  const c = readColors()
  const currentLabel = currentMonthIdx >= 0 ? series[currentMonthIdx]?.label : null
  const selectedLabel = selectedMonthIdx >= 0 && selectedMonthIdx !== currentMonthIdx ? series[selectedMonthIdx]?.label : null
  return (
    <AnimatedChart revision={series}><ResponsiveContainer width="100%" height={220}>
      <ComposedChart data={series} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <defs>{softGradient('grad-cumul', c.ACCENT, 0.22, 0.01)}</defs>
        <CartesianGrid {...gridProps(c)} />
        <XAxis dataKey="label" {...axisX(c)} />
        <YAxis {...axisY(c, 44)} />
        <Tooltip content={<ChartTooltip colors={c} />} />
        <ReferenceLine y={0} stroke={c.MUTED} strokeOpacity={0.4} strokeDasharray="4 4" />
        {currentMonthIdx >= 0 && currentMonthIdx === series.length - 1 && (
          <ReferenceLine x={currentLabel} stroke={c.ACCENT} strokeOpacity={0.4} strokeDasharray="4 4" label={{ position: 'top', offset: -10, fill: c.ACCENT, fontSize: 10, value: 'Mois en cours' }} />
        )}
        {selectedLabel && (
          <ReferenceLine x={selectedLabel} stroke={c.ACCENT} strokeOpacity={0.3} strokeDasharray="8 4" label={{ position: 'bottom', offset: 10, fill: c.ACCENT, fontSize: 9, value: 'Mois consulté' }} />
        )}
        <Area isAnimationActive={false} type="monotone" name="Cumul net" dataKey="cumul" stroke={c.ACCENT} fill="url(#grad-cumul)" strokeWidth={2.5} dot={false} activeDot={<ChartFocusDot />} />
      </ComposedChart>
    </ResponsiveContainer></AnimatedChart>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : donut des dépenses par catégorie (mois sélectionné)        */
/* ------------------------------------------------------------------ */
export function CategoriesWidget({ data, selectedMonth }) {
  const [active, setActive] = useState(null)
  // Si un mois est sélectionné, on l'utilise ; sinon le dernier mois disponible
  const month = selectedMonth || data.monthly?.at(-1)?.month
  const cur = month ? data.monthly?.find((m) => m.month === month) : data.monthly?.at(-1)
  const isCurrent = cur?.is_current || cur?.status === 'current'
  const curStatus = cur?.status === 'incomplete'
  const slices = useMemo(() => (data.by_category ?? []).filter((c) => c.month === month && c.total > 0), [data.by_category, month])
  const total = slices.reduce((s, c) => s + c.total, 0)
  if (slices.length === 0) return <Empty />
  const cc = readColors()
  return (
    <div className="category-overview">
    <div className="donut-wrap">
      <AnimatedChart revision={slices}><ResponsiveContainer width="100%" height={220}>
        <PieChart>
          <Pie isAnimationActive={false}
            data={slices} dataKey="total" nameKey="category"
            innerRadius="64%" outerRadius="88%" paddingAngle={3} stroke="none" cornerRadius={6}
            onMouseEnter={(_, i) => setActive(i)} onMouseLeave={() => setActive(null)}
          >
            {slices.map((c, i) => (
              <Cell key={i} aria-label={`${c.category} : ${eur(c.total)}`} fill={c.color || cc.VIZ[i % cc.VIZ.length]} opacity={active === null || active === i ? 1 : 0.3}
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
      </ResponsiveContainer></AnimatedChart>
      <div className="donut-center">
        <div className="donut-total"><AnimatedNumber value={active !== null ? slices[active].total : total} /></div>
        <div className="donut-label">
          {active !== null ? slices[active].category : monthLabel(month)}
          {isCurrent && <span className="donut-period-badge">Mois en cours</span>}
          {curStatus && <span className="donut-period-badge donut-period-incomplete">Incomplet</span>}
        </div>
      </div>
    </div>
    <ul className="category-legend" aria-label="Montants par catégorie">
      {slices.map((category, index) => (
        <li key={category.category}>
          <span className="category-legend-dot" style={{ background: category.color || cc.VIZ[index % cc.VIZ.length] }} aria-hidden="true" />
          <span>{category.category}</span>
          <strong>{eur(category.total)}</strong>
        </li>
      ))}
    </ul>
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : tendance d'une catégorie                                   */
/* ------------------------------------------------------------------ */
export function CategoryTrendWidget({ data, config, editing, onConfig, selectedMonth }) {
  const names = useMemo(
    () => [...new Set((data.by_category ?? []).map((c) => c.category))].sort(),
    [data],
  )
  const cat = config.category || names[0]
  const months = (data.monthly ?? []).slice(-(config.months || 12))
  const currentMonthIdx = months.findIndex(m => m.is_current || m.status === 'current')
  const selectedMonthIdx = selectedMonth ? months.findIndex(m => m.month === selectedMonth) : -1
  const series = useMemo(() => months.map((m) => ({
    label: monthLabel(m.month),
    total: +((data.by_category ?? []).find((c) => c.month === m.month && c.category === cat)?.total ?? 0).toFixed(2),
  })), [data.monthly, data.by_category, config.months, cat])
  const color = (data.by_category ?? []).find((c) => c.category === cat)?.color || GREEN
  const cc = readColors()
  const currentLabel = currentMonthIdx >= 0 ? series[currentMonthIdx]?.label : null
  const selectedLabel = selectedMonthIdx >= 0 && selectedMonthIdx !== currentMonthIdx ? series[selectedMonthIdx]?.label : null
  return (
    <div>
      <div className="widget-config">
        <select value={cat || ''} onChange={(e) => onConfig({ ...config, category: e.target.value })}>
          {names.map((n) => <option key={n} value={n}>{n}</option>)}
        </select>
      </div>
      {series.every((s) => s.total === 0) ? <Empty /> : (
        <AnimatedChart revision={series} growBars><ResponsiveContainer width="100%" height={190}>
          <BarChart data={series} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
            <defs>{softGradient('grad-bar', color, 0.95, 0.55)}</defs>
            <CartesianGrid {...gridProps(cc)} />
            <XAxis dataKey="label" {...axisX(cc)} />
            <YAxis {...axisY(cc, 40)} />
            <Tooltip content={<ChartTooltip colors={cc} />} cursor={{ fill: cc.LINE, opacity: 0.3, radius: 6 }} />
            {currentMonthIdx >= 0 && currentMonthIdx === series.length - 1 && (
              <ReferenceLine x={currentLabel} stroke={cc.ACCENT} strokeOpacity={0.4} strokeDasharray="4 4" label={{ position: 'top', offset: -10, fill: cc.ACCENT, fontSize: 10, value: 'Mois en cours' }} />
            )}
            {selectedLabel && (
              <ReferenceLine x={selectedLabel} stroke={cc.ACCENT} strokeOpacity={0.3} strokeDasharray="8 4" label={{ position: 'bottom', offset: 10, fill: cc.ACCENT, fontSize: 9, value: 'Mois consulté' }} />
            )}
            <Bar isAnimationActive={false} dataKey="total" name={cat} fill="url(#grad-bar)" maxBarSize={34} radius={[6, 6, 0, 0]} />
          </BarChart>
        </ResponsiveContainer></AnimatedChart>
      )}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : cartes KPI du mois sélectionné                             */
/* ------------------------------------------------------------------ */
export function KpiWidget({ data, selectedMonth }) {
  // Si un mois est sélectionné, on le cherche ; sinon le dernier mois disponible
  const month = selectedMonth || data.monthly?.at(-1)?.month
  const cur = month ? data.monthly?.find((m) => m.month === month) : data.monthly?.at(-1)
  const prevIdx = cur ? data.monthly?.indexOf(cur) - 1 : -1
  const prev = prevIdx >= 0 ? data.monthly[prevIdx] : undefined
  if (!cur) return <Empty />
  const net = cur.income - cur.expenses
  const savings = cur.income > 0 ? (net / cur.income) * 100 : null

  // Ne pas calculer de delta si le mois courant est incomplet
  // ou si le mois précédent est incomplet.
  const isCurrent = cur.is_current || cur.status === 'current'
  const prevIncomplete = prev && (prev.status === 'incomplete' || prev.is_current)

  const delta = (a, b) => (!isCurrent && !prevIncomplete && b ? ((a - b) / b) * 100 : null)

  const kpis = [
    { label: 'Revenus', num: cur.income, fmt: eur, d: delta(cur.income, prev?.income), goodUp: true },
    { label: 'Dépenses', num: cur.expenses, fmt: eur, d: delta(cur.expenses, prev?.expenses), goodUp: false },
    { label: 'Net', num: net, fmt: eur, cls: net >= 0 ? 'credit' : 'debit' },
    { label: "Taux d'épargne", num: savings, fmt: (v) => (v === null ? '—' : `${v.toFixed(0)} %`), cls: savings >= 0 ? 'credit' : 'debit' },
  ]

  const periodBadge = isCurrent ? (
    <span className="kpi-period-badge">Mois en cours — {cur.days_observed}/{cur.days_in_month} jours</span>
  ) : cur.status === 'incomplete' ? (
    <span className="kpi-period-badge kpi-period-incomplete">Incomplet — {cur.days_observed}/{cur.days_in_month} jours</span>
  ) : null

  return (
    <div className="kpi-grid">
      {periodBadge && <div className="kpi-period-row">{periodBadge}</div>}
      {kpis.map((k) => (
        <div className="kpi-card" key={k.label}>
          <div className="label">{k.label}</div>
          <div className={'value ' + (k.cls || '')}>
            {k.num === null ? '—' : <AnimatedNumber value={k.num} format={k.fmt} />}
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
/* Widget : top dépenses du mois sélectionné                           */
/* ------------------------------------------------------------------ */
export function TopWidget({ config, selectedMonth, data }) {
  const [rows, setRows] = useState(null)
  const [periodInfo, setPeriodInfo] = useState(null)
  useEffect(() => {
    let cancelled = false
    // Les transferts internes (virements entre comptes personnels) ne sont
    // pas des dépenses : on les exclut du top.
    const now = new Date()
    const currentMonthStr = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
    const monthToUse = selectedMonth || data?.monthly?.at(-1)?.month || currentMonthStr

    const cur = data?.monthly?.find((m) => m.month === monthToUse) || data?.monthly?.at(-1)
    if (!cur) {
      if (!cancelled) {
        setRows([])
        setPeriodInfo(null)
      }
      return
    }
    if (!cancelled) {
      setPeriodInfo({ isCurrent: cur.is_current || cur.status === 'current', incomplete: cur.status === 'incomplete' })
    }

    // The backend applies financialsql's owner-scoped transfer/savings predicate,
    // including child domains; the roots-only categories API cannot do that.
    api.transactions({ month: monthToUse, type: 'debit' }).then((txs) => {
      if (cancelled) return
      setRows(txs.sort((a, b) => a.amount - b.amount).slice(0, config.count || 5))
    }).catch(() => {
      if (!cancelled) setRows([])
    })
    return () => { cancelled = true }
  }, [config.count, selectedMonth, data?.monthly?.length])
  if (rows === null) return <Loading />
  if (rows.length === 0) return <Empty />
  const max = -rows[0].amount
  return (
    <div className="top-list">
      {(periodInfo?.isCurrent || periodInfo?.incomplete) && (
        <div className="top-period-badge">
          {periodInfo.isCurrent ? `Mois en cours` : `Incomplet`}
        </div>
      )}
      {rows.map((t) => (
        <div className="top-row" key={t.id}>
          <ProgressFill className="top-bar" percent={(-t.amount / max) * 100} />
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
export function BudgetsWidget({ selectedMonth }) {
  const [budgets, setBudgets] = useState(null)
  const [budgetError, setBudgetError] = useState(null)
  useEffect(() => {
    let active = true
    setBudgets(null); setBudgetError(null)
    api.budgets(selectedMonth).then(rows => {
      if (active) setBudgets(rows.filter(b => !b.scope || b.scope === 'expense'))
    }).catch(e => { if (active) setBudgetError(e.message) })
    return () => { active = false }
  }, [selectedMonth])
  if (budgetError) return <p role="alert" className="widget-empty">Budgets indisponibles : {budgetError}</p>
  if (budgets === null) return <Loading />
  if (budgets.length === 0) return <Empty text="Aucun budget — onglet Budgets pour en créer." />
  return (
    <div className="mini-budgets">
      <p className="mini-budget-explanation">Part du plafond mensuel dépensée, selon les opérations importées.</p>
      {budgets.slice(0, 4).map((b) => {
        const model = budgetWidgetModel(b)
        return (
        <div key={b.id} className="mini-budget">
          <div className="mini-budget-head">
            <span>{model.name}</span>
            <span className={model.over ? 'debit' : ''}>{model.percent} % utilisé</span>
          </div>
          <div className="mini-budget-amounts">{eur(model.spent)} dépensés / {eur(model.limit)} de plafond</div>
          <div className="gauge"><div
            className={'gauge-fill' + (model.over ? ' over' : model.percent >= 85 ? ' warn' : '')}
            style={{ width: model.width + '%' }}
          /></div>
          <div className={'mini-budget-balance' + (model.over ? ' debit' : '')}>
            {model.over ? `Plafond dépassé de ${eur(model.difference)}` : model.difference === 0 ? 'Plafond atteint' : `${eur(model.difference)} avant le plafond`}
          </div>
        </div>
        )
      })}
    </div>
  )
}

/* ------------------------------------------------------------------ */
/* Widget : abonnements                                                */
/* ------------------------------------------------------------------ */
export function SubsWidget({ selectedMonth }) {
  const [subs, setSubs] = useState(null)
  const [error, setError] = useState(null)
  useEffect(() => {
    let active = true
    setSubs(null); setError(null)
    api.recurringReview(selectedMonth).then(report => {
      if (active) setSubs(report.items.filter(item => item.nature === 'subscription' && item.status !== 'ignored'))
    }).catch(e => { if (active) setError(e.message) })
    return () => { active = false }
  }, [selectedMonth])
  if (error) return <p role="alert" className="widget-empty">Détections indisponibles : {error}</p>
  if (subs === null) return <Loading />
  if (subs.length === 0) return <Empty text="Aucun abonnement documenté ou détecté à examiner sur cette période." />
  return (
    <div>
      <p className="mini-budget-explanation">Médianes par opération observée. Ces détections ne constituent pas un total mensuel certain.</p>
      <div className="top-list">
        {subs.slice(0, 6).map(item => (
          <div className="top-row" key={item.key}>
            <span className="top-label">{item.label.slice(0, 40)}<small style={{display:'block'}}>{item.status === 'confirmed' ? 'Confirmée par vous' : item.confidence === 'probable' ? 'Probable' : 'À vérifier · données insuffisantes'}</small></span>
            <span className="top-amount">{euroCents(cents(item.typical_amount))}</span>
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
export function AttentionWidget({ onNavigate, selectedMonth }) {
  const [items, setItems] = useState(null)
  useEffect(() => {
    api.anomalies(selectedMonth).then((d) => setItems(d.anomalies || [])).catch(() => setItems([]))
  }, [selectedMonth])
  if (items === null) return <Loading />
  if (items.length === 0) return <Empty text="Rien à signaler — aucune anomalie détectée." />
  const icon = { subscription_up: '↗', unusual: '!', new_recurring: '+', subscription_gone: '×' }
  // Le clic amène aux transactions de ce marchand SANS filtre de mois : une
  // anomalie peut porter sur un mois autre que le mois courant (ex. un
  // prélèvement de mars vu en juillet). Filtrer sur le mois courant afficherait
  // une liste vide — on cherche donc par libellé sur tout l'historique.
  const go = (a) => onNavigate?.('transactions', { q: a.label, month: '' })
  return (
    <div className="attn-list">
      {items.slice(0, 6).map((a, i) => (
        <button
          type="button"
          className={`attn-item attn-${a.severity}${onNavigate ? ' attn-clickable' : ''}`}
          key={i}
          onClick={() => a.label && go(a)}
          disabled={!onNavigate || !a.label}
        >
          <span className="attn-icon">{icon[a.kind] || '•'}</span>
          <div className="attn-body">
            <div className="attn-title">{a.title}</div>
            <div className="attn-detail">{a.detail}</div>
          </div>
          {onNavigate && a.label && <span className="attn-arrow" aria-hidden="true">→</span>}
        </button>
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
