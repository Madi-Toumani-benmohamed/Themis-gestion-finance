import { useState } from 'react'
import TagHistoryChart from '../components/TagHistoryChart'
import {
  TAG_HISTORY_DURATIONS,
  tagHistoryWindow,
  tagHistoryModel,
} from '../components/tagHistoryModel.mjs'
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
} from 'recharts'
import { api } from '../api'
import TagPickerNavigation from '../components/TagPickerNavigation'
import '../components/TagPickerNavigation.css'
import {
  analysisMonth,
  analysisMoney,
  analysisDate,
} from '../components/analysisPageModel.mjs'
import { useAnalysisResource } from '../components/analysisResources'
const PALETTE = [
  '#9c99ff',
  '#55cda0',
  '#e6ac63',
  '#ee8d9c',
  '#b697ed',
  '#6fc7ba',
]
function State({ resource, children }) {
  if (resource.loading) return <p role="status">Chargement des tags…</p>
  if (resource.error)
    return (
      <div role="alert">
        <p>{resource.error}</p>
        <button className="btn-secondary" onClick={resource.retry}>
          Réessayer
        </button>
      </div>
    )
  return children(resource.data)
}
function CategoryTags({ month, catId, onNavigate }) {
  const resource = useAnalysisResource(
    (signal) => api.tagsInCategory(catId, month, { signal }),
    [catId, month],
  )
  return (
    <State resource={resource}>
      {(data) => (
        <>
          <p>
            Total unique de la catégorie :{' '}
            <strong>{analysisMoney(data.category_total)}</strong> ·{' '}
            {data.category_count} opération(s).
          </p>
          <p className="analysis-footnote">
            Une opération peut porter plusieurs tags. Les lignes se chevauchent
            et ne s’additionnent pas. {data.overlap_count} opération(s)
            multitag.
          </p>
          {data.slices?.length ? (
            <>
              <div
                className="analysis-chart"
                role="img"
                aria-label="Dépenses par tag, non additives ; données dans le tableau"
              >
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart
                    data={data.slices}
                    layout="vertical"
                    margin={{ left: 10, right: 16 }}
                  >
                    <CartesianGrid stroke="var(--line)" horizontal={false} />
                    <XAxis
                      type="number"
                      unit=" €"
                      tick={{ fill: 'var(--ink)', fontSize: 11 }}
                    />
                    <YAxis
                      type="category"
                      dataKey="tag"
                      width={95}
                      tick={{ fill: 'var(--ink)', fontSize: 11 }}
                    />
                    <Tooltip formatter={(v) => analysisMoney(v)} />
                    <Bar
                      dataKey="spent"
                      name="Débits observés"
                      fill={PALETTE[0]}
                    />
                  </BarChart>
                </ResponsiveContainer>
              </div>
              <div className="analysis-table-wrap">
                <table className="analysis-table">
                  <caption>
                    Dépenses par tag · {month} · hors transferts et épargne.
                  </caption>
                  <thead>
                    <tr>
                      <th>Tag</th>
                      <th className="amount">Observé</th>
                      <th className="amount">Opérations</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.slices.map((s) => (
                      <tr key={s.tag}>
                        <td>
                          <button
                            className="link"
                            onClick={() =>
                              onNavigate?.('transactions', {
                                month,
                                domain: catId,
                                type: 'debit',
                                tag:
                                  s.tag === '(sans tag)' ? '__none__' : s.tag,
                              })
                            }
                          >
                            {s.tag}
                          </button>
                        </td>
                        <td className="amount">{analysisMoney(s.spent)}</td>
                        <td className="amount">{s.count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          ) : (
            <p>Aucune dépense ordinaire observée dans cette catégorie.</p>
          )}
        </>
      )}
    </State>
  )
}
function TagHistory({ month, tags, onNavigate }) {
  const [picked, setPicked] = useState(tags.slice(0, 4).map((t) => t.name))
  const [duration, setDuration] = useState('6')
  const [window, setWindow] = useState(() => tagHistoryWindow(month, 6))
  const [draftFrom, setDraftFrom] = useState(window.from.slice(0, 7)),
    [draftTo, setDraftTo] = useState(month)
  const [rangeError, setRangeError] = useState(null)
  const resource = useAnalysisResource(
    async (signal) => {
      const data = picked.length
        ? await api.tagSeries(picked, 'month', window.params, { signal })
        : {
            buckets: [],
            series: [],
            query_from: window.from,
            query_to: window.to,
          }
      tagHistoryModel(data, window)
      return data
    },
    [JSON.stringify(window.params), JSON.stringify(picked)],
  )
  const chooseDuration = (value) => {
    setDuration(value)
    setRangeError(null)
    if (value !== 'custom') setWindow(tagHistoryWindow(month, Number(value)))
  }
  const applyRange = (event) => {
    event.preventDefault()
    try {
      setWindow(tagHistoryWindow(month, 'custom', draftFrom, draftTo))
      setRangeError(null)
    } catch (error) {
      setRangeError(error.message)
    }
  }
  return (
    <>
      <div className="analysis-history-toolbar">
        <label>
          Période de l’historique
          <select
            value={duration}
            onChange={(event) => chooseDuration(event.target.value)}
          >
            {TAG_HISTORY_DURATIONS.map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
            <option value="custom">Personnalisée</option>
          </select>
        </label>
        <p>
          Du {analysisDate(window.from)} au {analysisDate(window.to)} ·{' '}
          {window.months} mois. Les durées prédéfinies se terminent au mois de
          référence {month}.
        </p>
      </div>
      {duration === 'custom' && (
        <form className="analysis-history-range" onSubmit={applyRange}>
          <label>
            Début de l’historique
            <input
              type="month"
              required
              min="0001-01"
              max={month}
              value={draftFrom}
              onChange={(e) => setDraftFrom(e.target.value)}
            />
          </label>
          <label>
            Fin de l’historique
            <input
              type="month"
              required
              min="0001-01"
              max={month}
              value={draftTo}
              onChange={(e) => setDraftTo(e.target.value)}
            />
          </label>
          <button className="btn-secondary" type="submit">
            Appliquer la période
          </button>
          <p>
            Jusqu’à 50 ans par requête. Les bornes ne sont appliquées qu’après
            validation.
          </p>
        </form>
      )}
      {rangeError && (
        <p role="alert" className="notice err">
          {rangeError}
        </p>
      )}
      <TagPickerNavigation
        tags={tags.map((t) => t.name)}
        pickedTags={picked}
        onToggle={(name) =>
          setPicked((values) =>
            values.includes(name)
              ? values.filter((t) => t !== name)
              : [...values, name],
          )
        }
      />
      <State resource={resource}>
        {(data) => (
          <TagHistoryChart
            key={JSON.stringify(window.params) + JSON.stringify(picked)}
            data={data}
            window={window}
            tags={tags}
            picked={picked}
            onNavigate={onNavigate}
          />
        )}
      </State>
    </>
  )
}
export default function TagAnalysis({
  categories,
  month = analysisMonth(),
  onNavigate,
}) {
  const [catId, setCatId] = useState(String(categories[0]?.id || ''))
  const tags = useAnalysisResource((signal) => api.tags({ signal }), [])
  return (
    <>
      <section className="analysis-panel">
        <div className="analysis-panel-head">
          <h2>Dans une catégorie, quels tags ?</h2>
          <label>
            Catégorie{' '}
            <select value={catId} onChange={(e) => setCatId(e.target.value)}>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.parent_id ? '↳ ' : ''}
                  {c.name}
                </option>
              ))}
            </select>
          </label>
        </div>
        {catId ? (
          <CategoryTags
            key={catId}
            month={month}
            catId={catId}
            onNavigate={onNavigate}
          />
        ) : (
          <p>Aucune catégorie disponible.</p>
        )}
      </section>
      <section className="analysis-panel">
        <h2>Évolution observée par tag</h2>
        <p>
          Explore tes habitudes sur la durée de ton choix. Couverture historique
          non établie : les différences ne prouvent pas une variation à
          périmètre constant.
        </p>
        <State resource={tags}>
          {(values) =>
            values.length ? (
              <TagHistory month={month} tags={values} onNavigate={onNavigate} />
            ) : (
              <p>Aucun tag disponible.</p>
            )
          }
        </State>
      </section>
    </>
  )
}
