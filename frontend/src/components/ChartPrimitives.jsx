import { eur } from '../api'

// Couleurs : lues depuis les variables CSS pour suivre le thème clair/sombre.
// On les relit à chaque rendu de widget via readColors().
export function readColors() {
  if (typeof window === 'undefined') return {}
  const s = getComputedStyle(document.documentElement)
  const v = (n, fb) => s.getPropertyValue(n).trim() || fb
  return {
    INK: v('--ink', '#16161D'),
    GREEN: v('--pos', '#21935B'),
    RED: v('--neg', '#D6455D'),
    MUTED: v('--muted', '#8A8A99'),
    LINE: v('--line', '#ECECF1'),
    ACCENT: v('--accent', '#5B5BD6'),
    PAPER: v('--paper', '#FFFFFF'),
    VIZ: [
      v('--viz-1', '#5B5BD6'),
      v('--viz-2', '#7C7CE0'),
      v('--viz-3', '#9C9CEA'),
      v('--viz-4', '#B9B9F1'),
      v('--viz-5', '#D2D2F7'),
      v('--viz-6', '#E9E9FB'),
    ],
  }
}

// Dégradé vertical réutilisable : plein en haut, transparent en bas — l'effet
// « aire lumineuse » caractéristique des graphiques soignés.
export function softGradient(id, color, top = 0.28, bottom = 0.02) {
  return (
    <linearGradient key={id} id={id} x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stopColor={color} stopOpacity={top} />
      <stop offset="100%" stopColor={color} stopOpacity={bottom} />
    </linearGradient>
  )
}

// Tooltip épuré, lisible en clair comme en sombre (fond = --paper).
export function ChartTooltip({
  active,
  payload,
  label,
  colors,
  formatter = eur,
  labelFormatter = (value) => value,
}) {
  if (!active || !payload || !payload.length) return null
  return (
    <div
      className="chart-tooltip"
      role="tooltip"
      style={{
        background: colors.PAPER,
        borderRadius: 14,
        boxShadow:
          '0 12px 40px rgba(22,22,29,.16), 0 2px 8px rgba(22,22,29,.06)',
        padding: '10px 14px',
        fontFamily: 'Inter, sans-serif',
        fontSize: 12.5,
        border: `1px solid ${colors.LINE}`,
      }}
    >
      {label != null && (
        <div style={{ color: colors.MUTED, marginBottom: 4, fontWeight: 500 }}>
          {labelFormatter(label)}
        </div>
      )}
      {payload.map((p, i) => (
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
              background: p.color || p.stroke || p.fill,
            }}
          />
          <span style={{ flex: 1 }}>{p.name}</span>
          <span style={{ fontVariantNumeric: 'tabular-nums' }}>
            {formatter(p.value)}
          </span>
        </div>
      ))}
    </div>
  )
}

export const fmtTick = (v) =>
  Math.abs(v) >= 1000 ? `${Math.round(v / 1000)}k` : v
// Axes épurés façon Apple : pas de ligne d'axe, pas de trait de tick, texte discret.
export const axisX = (c) => ({
  tick: { fontSize: 11, fill: c.MUTED },
  tickLine: false,
  axisLine: false,
  dy: 6,
  minTickGap: 24,
})
export const axisY = (c, w = 40) => ({
  tickFormatter: fmtTick,
  tick: { fontSize: 11, fill: c.MUTED },
  tickLine: false,
  axisLine: false,
  width: w,
})
export const gridProps = (c) => ({
  stroke: c.LINE,
  strokeOpacity: 0.5,
  vertical: false,
})
