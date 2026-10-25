import { memo, useCallback, useId, useMemo, useState } from 'react'
import {
  ResponsiveContainer,
  ComposedChart,
  Area,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
  Brush,
} from 'recharts'
import {
  ChartTooltip,
  readColors,
  softGradient,
  axisX,
  axisY,
  gridProps,
} from './ChartPrimitives'
import { analysisMoney } from './analysisPageModel.mjs'
import { tagHistoryMonthLabel, tagHistoryModel } from './tagHistoryModel.mjs'

const historyValue = (value) =>
  value == null ? 'Non observé' : analysisMoney(value)

// Brush selection updates its caption without remounting/redrawing the curves.
const Curves = memo(function Curves({ model, colors, onRange }) {
  const gradientId = useId().replaceAll(':', '')
  const hasBrush = model.points.length > 14
  return (
    <div
      className="analysis-tag-history-chart"
      role="group"
      aria-label="Courbes des dépenses mensuelles par tag ; montants accessibles dans le tableau ci-dessous"
    >
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart
          data={model.points}
          margin={{ top: 12, right: 12, left: 0, bottom: 0 }}
        >
          <defs>
            {model.series.map((s) =>
              softGradient(`${gradientId}-${s.key}`, s.color, 0.2, 0.01),
            )}
          </defs>
          <CartesianGrid {...gridProps(colors)} />
          <XAxis
            dataKey="period"
            {...axisX(colors)}
            tickFormatter={tagHistoryMonthLabel}
          />
          <YAxis {...axisY(colors, 48)} domain={[0, 'auto']} />
          <Tooltip
            filterNull={false}
            content={
              <ChartTooltip
                colors={colors}
                formatter={historyValue}
                labelFormatter={tagHistoryMonthLabel}
              />
            }
            cursor={{
              stroke: colors.MUTED,
              strokeOpacity: 0.45,
              strokeDasharray: '4 4',
            }}
          />
          <Legend
            iconType="circle"
            wrapperStyle={{ fontSize: 12, paddingTop: 6 }}
          />
          {model.series.map((s) => (
            <Area
              key={s.key}
              dataKey={s.key}
              name={s.tag}
              type="monotone"
              stroke={s.color}
              fill={`url(#${gradientId}-${s.key})`}
              strokeWidth={2.5}
              fillOpacity={1}
              connectNulls={false}
              isAnimationActive={false}
              activeDot={{ r: 4, strokeWidth: 0 }}
              dot={({ cx, cy, payload }) =>
                payload?.[`${s.key}_isolated`] &&
                Number.isFinite(cx) &&
                Number.isFinite(cy) ? (
                  <circle
                    cx={cx}
                    cy={cy}
                    r={3.5}
                    fill={s.color}
                    stroke={colors.PAPER}
                    strokeWidth={1.5}
                  />
                ) : null
              }
            />
          ))}
          {hasBrush && (
            <Brush
              ariaLabel="Zoom de l’historique : flèches gauche et droite pour déplacer les bornes"
              dataKey="period"
              height={28}
              stroke={colors.ACCENT}
              fill={colors.PAPER}
              travellerWidth={9}
              gap={1}
              startIndex={0}
              endIndex={model.points.length - 1}
              tickFormatter={tagHistoryMonthLabel}
              onChange={onRange}
            />
          )}
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  )
})

export default function TagHistoryChart({
  data,
  window,
  tags,
  picked,
  onNavigate,
}) {
  const theme =
    typeof document !== 'undefined'
      ? document.documentElement.getAttribute('data-theme')
      : 'light'
  const colors = useMemo(() => readColors(), [theme])
  const base = useMemo(() => tagHistoryModel(data, window), [data, window])
  const model = useMemo(() => {
    const palette = [
      colors.ACCENT,
      colors.GREEN,
      '#c78238',
      colors.RED,
      '#8e72db',
      '#3b9c9a',
    ]
    return {
      ...base,
      series: base.series.map((s) => ({
        ...s,
        color:
          palette[
            Math.max(
              0,
              tags.findIndex((t) => t.name === s.tag),
            ) % palette.length
          ],
      })),
    }
  }, [base, colors, tags])
  const [range, setRange] = useState(null)
  const onRange = useCallback((next) => {
    if (
      !next ||
      !Number.isInteger(next.startIndex) ||
      !Number.isInteger(next.endIndex)
    )
      return
    setRange((previous) =>
      previous?.startIndex === next.startIndex &&
      previous?.endIndex === next.endIndex
        ? previous
        : next,
    )
  }, [])
  const start = range?.startIndex ?? 0,
    end = range?.endIndex ?? model.points.length - 1
  const missing = picked.filter(
    (name) => !model.series.some((s) => s.tag === name),
  )
  if (!model.hasObservations)
    return (
      <p>
        {picked.length
          ? 'Aucune opération observée pour ces tags dans cette fenêtre.'
          : 'Sélectionne au moins un tag.'}
      </p>
    )
  return (
    <>
      <div
        className="analysis-history-caption"
        role="status"
        aria-live="polite"
        aria-atomic="true"
      >
        <span>
          {tagHistoryMonthLabel(model.points[start].period)} →{' '}
          {tagHistoryMonthLabel(model.points[end].period)} · {end - start + 1}{' '}
          mois affichés · montants en €
        </span>
        {model.points.length > 14 && (
          <span>Déplace les poignées sous le graphique pour zoomer.</span>
        )}
      </div>
      <Curves model={model} colors={colors} onRange={onRange} />
      <p className="analysis-footnote">
        Les interruptions indiquent des mois sans observation, pas des dépenses
        nulles. Les tags peuvent se chevaucher : les courbes ne s’additionnent
        pas.
      </p>
      {missing.length > 0 && (
        <p className="analysis-footnote">
          Aucune opération observée pour : {missing.join(', ')}.
        </p>
      )}
      {onNavigate && (
        <div
          className="analysis-history-sources"
          aria-label="Sources de l’historique par tag"
        >
          {model.series.map((s) => (
            <button
              key={s.key}
              className="link"
              onClick={() =>
                onNavigate('transactions', {
                  tag: s.tag,
                  from: window.from,
                  to: window.to,
                  type: 'debit',
                })
              }
            >
              Opérations #{s.tag} →
            </button>
          ))}
        </div>
      )}
      <details>
        <summary>Consulter les montants par tag</summary>
        <div className="analysis-table-wrap">
          <table className="analysis-table">
            <caption>
              Historique mensuel du {window.from} au {window.to}. Une absence
              n’est pas un zéro documenté.
            </caption>
            <thead>
              <tr>
                <th>Période</th>
                {model.series.map((s) => (
                  <th key={s.key}>{s.tag}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {model.points.map((row) => (
                <tr key={row.period}>
                  <th scope="row">{tagHistoryMonthLabel(row.period)}</th>
                  {model.series.map((s) => (
                    <td className="amount" key={s.key}>
                      {historyValue(row[s.key])}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </>
  )
}
