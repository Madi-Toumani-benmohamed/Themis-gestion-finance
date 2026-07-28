import { useEffect, useMemo, useState } from 'react'
import {
  ResponsiveContainer, BarChart, Bar, LineChart, Line, PieChart, Pie, Cell,
  CartesianGrid, XAxis, YAxis, Tooltip, Legend,
} from 'recharts'
import { api, eur } from '../api'

const PALETTE = ['#5B5BD6', '#21935B', '#C77D2E', '#D6455D', '#8A5CD1', '#3F8F7B', '#B5548A', '#4C6FBF']
function readColors() {
  if (typeof window === 'undefined') return { MUTED: '#8A8A99', LINE: '#ECECF1', PAPER: '#fff', INK: '#16161D' }
  const st = getComputedStyle(document.documentElement)
  const v = (n, fb) => (st.getPropertyValue(n).trim() || fb)
  return { MUTED: v('--muted', '#8A8A99'), LINE: v('--line', '#ECECF1'), PAPER: v('--paper', '#fff'), INK: v('--ink', '#16161D') }
}
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

// TagAnalysis — la vue croisée catégorie × tag. Deux angles :
//  (1) répartition par tag DANS une catégorie choisie,
//  (2) évolution de tags choisis au fil du temps, toutes catégories confondues.
export default function TagAnalysis({ categories }) {
  const [catId, setCatId] = useState('')
  const [slices, setSlices] = useState(null)
  const [allTags, setAllTags] = useState([])
  const [pickedTags, setPickedTags] = useState([])
  const [series, setSeries] = useState(null)

  // catégorie par défaut = la première disponible
  useEffect(() => {
    if (!catId && categories.length) setCatId(String(categories[0].id))
  }, [categories, catId])

  // (1) répartition par tag dans la catégorie
  useEffect(() => {
    if (!catId) return
    setSlices(null)
    api.tagsInCategory(catId).then((d) => setSlices(d.slices || [])).catch(() => setSlices([]))
  }, [catId])

  // liste des tags existants (pour la sélection de la vue 2)
  useEffect(() => {
    api.tags().then((tags) => {
      const names = tags.map((t) => t.name)
      setAllTags(names)
      setPickedTags(names.slice(0, 4)) // pré-sélection des premiers
    }).catch(() => {})
  }, [])

  // (2) évolution temporelle des tags choisis
  useEffect(() => {
    if (pickedTags.length === 0) { setSeries({ buckets: [], series: [] }); return }
    setSeries(null)
    api.tagSeries(pickedTags, 'month').then(setSeries).catch(() => setSeries({ buckets: [], series: [] }))
  }, [pickedTags])

  const toggleTag = (name) => {
    setPickedTags((p) => (p.includes(name) ? p.filter((x) => x !== name) : [...p, name]))
  }

  const catName = categories.find((c) => String(c.id) === String(catId))?.name || ''
  const totalInCat = useMemo(() => (slices || []).reduce((s, x) => s + x.spent, 0), [slices])

  // données du line chart : [{ mois, tagA, tagB, ... }]
  const lineData = useMemo(() => {
    if (!series || !series.buckets) return []
    return series.buckets.map((b, i) => {
      const row = { mois: b }
      series.series.forEach((s) => { row[s.tag] = s.values[i] })
      return row
    })
  }, [series])

  const cc = readColors()

  return (
    <div className="tag-analysis">
      <div className="section-head" style={{ marginTop: 40 }}>
        <h2>Analyse par tag</h2>
      </div>
      <p className="muted" style={{ marginTop: -12, marginBottom: 24, fontSize: 13.5 }}>
        Les catégories disent <em>où</em> part ton argent ; les tags disent <em>quoi</em> exactement.
        Croise les deux ici.
      </p>

      <div className="tag-analysis-grid">
        {/* Vue 1 : répartition par tag dans une catégorie */}
        <div className="analysis-card">
          <div className="analysis-card-head">
            <h3>Répartition par tag</h3>
            <select value={catId} onChange={(e) => setCatId(e.target.value)}>
              {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </div>
          {slices === null ? (
            <p className="muted" style={{ padding: 20 }}><span className="spinner" />Chargement…</p>
          ) : slices.length === 0 ? (
            <p className="widget-empty">Aucune dépense taguée dans « {catName} ». Ajoute des tags à tes transactions pour voir la répartition.</p>
          ) : (
            <>
              <ResponsiveContainer width="100%" height={230}>
                <PieChart>
                  <Pie data={slices} dataKey="spent" nameKey="tag" cx="50%" cy="50%"
                       innerRadius={62} outerRadius={90} paddingAngle={3} cornerRadius={6} stroke="none">
                    {slices.map((s, i) => (
                      <Cell key={i} fill={s.tag === '(sans tag)' ? '#D0D0DA' : PALETTE[i % PALETTE.length]} />
                    ))}
                  </Pie>
                  <Tooltip content={<ChartTooltip colors={cc} />} />
                </PieChart>
              </ResponsiveContainer>
              <div className="tag-legend">
                {slices.map((s, i) => (
                  <div className="tag-legend-row" key={i}>
                    <span className="tag-legend-dot" style={{ background: s.tag === '(sans tag)' ? '#D0D0DA' : PALETTE[i % PALETTE.length] }} />
                    <span className="tag-legend-name">{s.tag}</span>
                    <span className="tag-legend-val">{eur(s.spent)}</span>
                    <span className="tag-legend-pct">{totalInCat > 0 ? Math.round(s.spent / totalInCat * 100) : 0}%</span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>

        {/* Vue 2 : évolution temporelle des tags */}
        <div className="analysis-card">
          <div className="analysis-card-head">
            <h3>Évolution dans le temps</h3>
          </div>
          {allTags.length === 0 ? (
            <p className="widget-empty">Aucun tag pour l'instant. Tague quelques transactions pour suivre leur évolution.</p>
          ) : (
            <>
              <div className="tag-picker">
                {allTags.map((t) => (
                  <button key={t}
                          className={'tag-pick' + (pickedTags.includes(t) ? ' on' : '')}
                          onClick={() => toggleTag(t)}>
                    {t}
                  </button>
                ))}
              </div>
              {series === null ? (
                <p className="muted" style={{ padding: 20 }}><span className="spinner" />Chargement…</p>
              ) : lineData.length === 0 ? (
                <p className="widget-empty">Sélectionne au moins un tag pour voir son évolution.</p>
              ) : (
                <ResponsiveContainer width="100%" height={260}>
                  <LineChart data={lineData} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
                    <CartesianGrid stroke={cc.LINE} strokeOpacity={0.5} vertical={false} />
                    <XAxis dataKey="mois" tick={{ fontSize: 11, fill: cc.MUTED }} tickLine={false} axisLine={false} dy={6} minTickGap={24} />
                    <YAxis tickFormatter={fmtTick} tick={{ fontSize: 11, fill: cc.MUTED }} tickLine={false} axisLine={false} />
                    <Tooltip content={<ChartTooltip colors={cc} />} />
                    <Legend iconType="circle" wrapperStyle={{ fontSize: 12, paddingTop: 6 }} />
                    {(series.series || []).map((s, i) => (
                      <Line key={s.tag} type="monotone" dataKey={s.tag}
                            stroke={PALETTE[i % PALETTE.length]} strokeWidth={2.5} dot={false} activeDot={{ r: 4, strokeWidth: 0 }} />
                    ))}
                  </LineChart>
                </ResponsiveContainer>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
