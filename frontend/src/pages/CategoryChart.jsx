import { AnimatedChart, ChartFocusDot } from '../motion/AnimatedChart'
import { useMemo, useState } from 'react'
import {
  ResponsiveContainer,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
  BarChart,
  Bar,
  LineChart,
  Line,
} from 'recharts'
import { api, eur } from '../api'
import {
  analysisMonth,
  analysisWindow,
  analysisMoney,
} from '../components/analysisPageModel.mjs'
import { useAnalysisResource } from '../components/analysisResources'

function readColors() {
  if (typeof window === 'undefined')
    return {
      MUTED: '#8A8A99',
      LINE: '#ECECF1',
      PAPER: '#fff',
      INK: '#16161D',
      ACCENT: '#5B5BD6',
    }
  const s = getComputedStyle(document.documentElement)
  const v = (n, fb) => s.getPropertyValue(n).trim() || fb
  return {
    MUTED: v('--muted', '#8A8A99'),
    LINE: v('--line', '#ECECF1'),
    PAPER: v('--paper', '#fff'),
    INK: v('--ink', '#16161D'),
    ACCENT: v('--accent', '#5B5BD6'),
  }
}
const MUTED = '#8A8A99'
const LINE = '#ECECF1'
function ChartTooltip({ active, payload, label, colors }) {
  if (!active || !payload || !payload.length) return null
  const rows = payload.filter((p) => p.value != null)
  if (!rows.length) return null
  return (
    <div
      style={{
        background: colors.PAPER,
        borderRadius: 14,
        border: `1px solid ${colors.LINE}`,
        boxShadow: '0 12px 40px rgba(22,22,29,.16)',
        padding: '10px 14px',
        fontFamily: 'Inter, sans-serif',
        fontSize: 12.5,
      }}
    >
      {label != null && (
        <div style={{ color: colors.MUTED, marginBottom: 4, fontWeight: 500 }}>
          {label}
        </div>
      )}
      {rows.map((p, i) => (
        <div
          key={i}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            color: colors.INK,
            fontWeight: 600,
          }}
        >
          <span
            style={{
              width: 8,
              height: 8,
              borderRadius: 9,
              background: p.color || p.fill || p.stroke,
            }}
          />
          <span style={{ flex: 1 }}>{p.name}</span>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>
            {eur(p.value)}
          </span>
        </div>
      ))}
    </div>
  )
}
const fmtTick = (v) => (Math.abs(v) >= 1000 ? `${Math.round(v / 1000)}k` : v)

export default function CategoryChart({
  month = analysisMonth(),
  categories: cats = [],
}) {
  const [selected, setSelected] = useState([])
  const [period, setPeriod] = useState('month')
  const [mode, setMode] = useState('bar')
  const window = analysisWindow(month)
  const resource = useAnalysisResource(
    (signal) =>
      api.categorySeries(
        {
          ...window,
          period,
          ...(selected.length ? { categories: selected.join(',') } : {}),
        },
        { signal },
      ),
    [month, selected.join(','), period],
  )
  const data = resource.data,
    err = resource.error

  const toggle = (id) =>
    setSelected((s) =>
      s.includes(id) ? s.filter((x) => x !== id) : [...s, id],
    )

  const seriesNames = useMemo(
    () => (data?.categories ?? []).map((c) => c.name),
    [data],
  )
  const colorOf = (name) =>
    data?.categories.find((c) => c.name === name)?.color || MUTED

  const Chart = mode === 'bar' ? BarChart : LineChart
  const cc = readColors()

  return (
    <div className="section">
      {err && (
        <div className="notice err" role="alert">
          {err}{' '}
          <button className="btn-secondary" onClick={resource.retry}>
            Réessayer
          </button>
        </div>
      )}
      <p>
        Opérations observées du {window.from} au {window.to}. La couverture
        historique n’est pas établie pour chaque intervalle : les hauteurs ne
        prouvent pas une évolution des dépenses. Les intervalles sans opération
        restent non observés.
      </p>

      <div className="section-head">
        <h2>Dépenses par catégorie</h2>
        <div className="chart-controls">
          <div className="seg">
            {[
              ['month', 'Mois'],
              ['quarter', 'Trimestre'],
              ['year', 'Année'],
            ].map(([v, l]) => (
              <button
                key={v}
                aria-pressed={period === v}
                className={period === v ? 'on' : ''}
                onClick={() => setPeriod(v)}
              >
                {l}
              </button>
            ))}
          </div>
          <div className="seg">
            {[
              ['bar', 'Barres'],
              ['line', 'Courbes'],
            ].map(([v, l]) => (
              <button
                key={v}
                aria-pressed={mode === v}
                className={mode === v ? 'on' : ''}
                onClick={() => setMode(v)}
              >
                {l}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="cat-chips">
        <button
          aria-pressed={selected.length === 0}
          className={'cat-chip' + (selected.length === 0 ? ' on' : '')}
          onClick={() => setSelected([])}
        >
          Toutes
        </button>
        {cats.map((c) => (
          <button
            key={c.id}
            aria-pressed={selected.includes(c.id)}
            className={'cat-chip' + (selected.includes(c.id) ? ' on' : '')}
            style={
              selected.includes(c.id)
                ? { borderColor: c.color, color: c.color }
                : {}
            }
            onClick={() => toggle(c.id)}
          >
            <span className="chip-dot" style={{ background: c.color }} />
            {c.name}
          </button>
        ))}
      </div>

      {resource.loading ? (
        <p role="status">Chargement des catégories…</p>
      ) : err ? null : !data || data.series.length === 0 ? (
        <div className="empty">Aucune dépense sur cette sélection.</div>
      ) : (
        <div
          className="widget"
          role="img"
          aria-label="Dépenses observées par catégorie ; données accessibles dans le tableau ci-dessous"
        >
          <AnimatedChart revision={data} dense={data.series.length > 240}><ResponsiveContainer width="100%" height={300}>
            <Chart
              data={data.series}
              margin={{ top: 10, right: 12, left: 0, bottom: 0 }}
            >
              <defs>
                {seriesNames.map((name) => (
                  <linearGradient
                    key={name}
                    id={`cg-${name.replace(/[^a-zA-Z0-9]/g, '')}`}
                    x1="0"
                    y1="0"
                    x2="0"
                    y2="1"
                  >
                    <stop
                      offset="0%"
                      stopColor={colorOf(name)}
                      stopOpacity={0.95}
                    />
                    <stop
                      offset="100%"
                      stopColor={colorOf(name)}
                      stopOpacity={0.55}
                    />
                  </linearGradient>
                ))}
              </defs>
              <CartesianGrid
                stroke={cc.LINE}
                strokeOpacity={0.5}
                vertical={false}
              />
              <XAxis
                dataKey="period"
                tick={{ fontSize: 11, fill: cc.MUTED }}
                tickLine={false}
                axisLine={false}
                dy={6}
                minTickGap={24}
              />
              <YAxis
                tickFormatter={fmtTick}
                tick={{ fontSize: 11, fill: cc.MUTED }}
                tickLine={false}
                axisLine={false}
                width={44}
              />
              <Tooltip
                content={<ChartTooltip colors={cc} />}
                cursor={{ fill: cc.LINE, opacity: 0.3, radius: 6 }}
              />
              <Legend
                iconType="circle"
                wrapperStyle={{ fontSize: 12, paddingTop: 6 }}
              />
              {seriesNames.map((name) =>
                mode === 'bar' ? (
                  <Bar isAnimationActive={false}
                    key={name}
                    dataKey={name}
                    stackId="a"
                    fill={`url(#cg-${name.replace(/[^a-zA-Z0-9]/g, '')})`}
                    maxBarSize={46}
                    radius={[4, 4, 0, 0]}
                  />
                ) : (
                  <Line isAnimationActive={false}
                    key={name}
                    type="monotone"
                    dataKey={name}
                    stroke={colorOf(name)}
                    strokeWidth={2.5}
                    dot={false}
                    activeDot={<ChartFocusDot />}
                  />
                ),
              )}
            </Chart>
          </ResponsiveContainer></AnimatedChart>
        </div>
      )}
      <p className="muted" style={{ fontSize: 13, marginTop: 12 }}>
        {mode === 'bar'
          ? 'Barres empilées : la hauteur totale est la dépense de la période.'
          : 'Une courbe par catégorie.'}{' '}
        Filtre les catégories avec les boutons ci-dessus.
      </p>

      {data?.series?.length > 0 && (
        <details>
          <summary>Consulter les montants du graphique</summary>
          <div className="analysis-table-wrap">
            <table className="analysis-table">
              <caption>
                Montants observés en euros, sans extrapolation des intervalles
                manquants.
              </caption>
              <thead>
                <tr>
                  <th>Période</th>
                  {seriesNames.map((name) => (
                    <th className="amount" key={name}>
                      {name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {data.series.map((row) => (
                  <tr key={row.period}>
                    <th scope="row">{row.period}</th>
                    {seriesNames.map((name) => (
                      <td className="amount" key={name}>
                        {row[name] == null
                          ? 'Non observé'
                          : analysisMoney(row[name])}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </details>
      )}
    </div>
  )
}
