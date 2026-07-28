import { useEffect, useMemo, useState } from 'react'
import {
  ResponsiveContainer, CartesianGrid, XAxis, YAxis, Tooltip, Legend,
  BarChart, Bar, LineChart, Line,
} from 'recharts'
import { api, eur } from '../api'

function readColors() {
  if (typeof window === 'undefined') return { MUTED: '#8A8A99', LINE: '#ECECF1', PAPER: '#fff', INK: '#16161D', ACCENT: '#5B5BD6' }
  const s = getComputedStyle(document.documentElement)
  const v = (n, fb) => (s.getPropertyValue(n).trim() || fb)
  return {
    MUTED: v('--muted', '#8A8A99'), LINE: v('--line', '#ECECF1'),
    PAPER: v('--paper', '#fff'), INK: v('--ink', '#16161D'), ACCENT: v('--accent', '#5B5BD6'),
  }
}
const MUTED = '#8A8A99'
const LINE = '#ECECF1'
function ChartTooltip({ active, payload, label, colors }) {
  if (!active || !payload || !payload.length) return null
  const rows = payload.filter((p) => p.value)
  if (!rows.length) return null
  return (
    <div style={{ background: colors.PAPER, borderRadius: 14, border: `1px solid ${colors.LINE}`,
      boxShadow: '0 12px 40px rgba(22,22,29,.16)', padding: '10px 14px', fontFamily: 'Inter, sans-serif', fontSize: 12.5 }}>
      {label != null && <div style={{ color: colors.MUTED, marginBottom: 4, fontWeight: 500 }}>{label}</div>}
      {rows.map((p, i) => (
        <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, color: colors.INK, fontWeight: 600 }}>
          <span style={{ width: 8, height: 8, borderRadius: 9, background: p.color || p.fill || p.stroke }} />
          <span style={{ flex: 1 }}>{p.name}</span>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>{eur(p.value)}</span>
        </div>
      ))}
    </div>
  )
}
const fmtTick = (v) => (Math.abs(v) >= 1000 ? `${Math.round(v / 1000)}k` : v)

export default function CategoryChart() {
  const [cats, setCats] = useState([])
  const [selected, setSelected] = useState([])
  const [period, setPeriod] = useState('month')
  const [mode, setMode] = useState('bar')
  const [data, setData] = useState(null)
  const [err, setErr] = useState(null)

  useEffect(() => { api.categories().then(setCats).catch((e) => setErr(e.message)) }, [])

  useEffect(() => {
    const params = { period }
    if (selected.length) params.categories = selected.join(',')
    api.categorySeries(params).then(setData).catch((e) => setErr(e.message))
  }, [selected, period])

  const toggle = (id) =>
    setSelected((s) => (s.includes(id) ? s.filter((x) => x !== id) : [...s, id]))

  const seriesNames = useMemo(() => (data?.categories ?? []).map((c) => c.name), [data])
  const colorOf = (name) => data?.categories.find((c) => c.name === name)?.color || MUTED

  const Chart = mode === 'bar' ? BarChart : LineChart
  const cc = readColors()

  return (
    <div className="section">
      {err && <div className="notice err">{err}</div>}

      <div className="section-head">
        <h2>Dépenses par catégorie</h2>
        <div className="chart-controls">
          <div className="seg">
            {[['month', 'Mois'], ['quarter', 'Trimestre'], ['year', 'Année']].map(([v, l]) => (
              <button key={v} className={period === v ? 'on' : ''} onClick={() => setPeriod(v)}>{l}</button>
            ))}
          </div>
          <div className="seg">
            {[['bar', 'Barres'], ['line', 'Courbes']].map(([v, l]) => (
              <button key={v} className={mode === v ? 'on' : ''} onClick={() => setMode(v)}>{l}</button>
            ))}
          </div>
        </div>
      </div>

      <div className="cat-chips">
        <button className={'cat-chip' + (selected.length === 0 ? ' on' : '')} onClick={() => setSelected([])}>
          Toutes
        </button>
        {cats.map((c) => (
          <button key={c.id}
                  className={'cat-chip' + (selected.includes(c.id) ? ' on' : '')}
                  style={selected.includes(c.id) ? { borderColor: c.color, color: c.color } : {}}
                  onClick={() => toggle(c.id)}>
            <span className="chip-dot" style={{ background: c.color }} />{c.name}
          </button>
        ))}
      </div>

      {!data || data.series.length === 0 ? (
        <div className="empty">Aucune dépense sur cette sélection.</div>
      ) : (
        <div className="widget">
          <ResponsiveContainer width="100%" height={360}>
            <Chart data={data.series} margin={{ top: 10, right: 12, left: 0, bottom: 0 }}>
              <defs>
                {seriesNames.map((name) => (
                  <linearGradient key={name} id={`cg-${name.replace(/[^a-zA-Z0-9]/g, '')}`} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={colorOf(name)} stopOpacity={0.95} />
                    <stop offset="100%" stopColor={colorOf(name)} stopOpacity={0.55} />
                  </linearGradient>
                ))}
              </defs>
              <CartesianGrid stroke={cc.LINE} strokeOpacity={0.5} vertical={false} />
              <XAxis dataKey="period" tick={{ fontSize: 11, fill: cc.MUTED }} tickLine={false} axisLine={false} dy={6} minTickGap={24} />
              <YAxis tickFormatter={fmtTick} tick={{ fontSize: 11, fill: cc.MUTED }} tickLine={false} axisLine={false} width={44} />
              <Tooltip content={<ChartTooltip colors={cc} />} cursor={{ fill: cc.LINE, opacity: 0.3, radius: 6 }} />
              <Legend iconType="circle" wrapperStyle={{ fontSize: 12, paddingTop: 6 }} />
              {seriesNames.map((name) =>
                mode === 'bar'
                  ? <Bar key={name} dataKey={name} stackId="a" fill={`url(#cg-${name.replace(/[^a-zA-Z0-9]/g, '')})`} maxBarSize={46} radius={[4, 4, 0, 0]} />
                  : <Line key={name} type="monotone" dataKey={name} stroke={colorOf(name)} strokeWidth={2.5} dot={false} activeDot={{ r: 4, strokeWidth: 0 }} />,
              )}
            </Chart>
          </ResponsiveContainer>
        </div>
      )}
      <p className="muted" style={{ fontSize: 13, marginTop: 12 }}>
        {mode === 'bar' ? 'Barres empilées : la hauteur totale est la dépense de la période.' : 'Une courbe par catégorie.'}
        {' '}Clique les étiquettes pour filtrer.
      </p>

    </div>
  )
}
