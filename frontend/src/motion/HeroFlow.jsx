import { useId, useMemo } from 'react'
import { Area, AreaChart, ResponsiveContainer, Tooltip } from 'recharts'
import { eur, monthLabel } from '../api'
import { ChartTooltip, readColors, softGradient } from '../components/ChartPrimitives'
import { AnimatedChart, ChartFocusDot } from './AnimatedChart'
import { m } from 'motion/react'
import { useMotionReduced } from './preference'

export function HeroAtmosphere() {
  const reduced = useMotionReduced()
  return <><span className="motion-hero-light" aria-hidden="true" /><svg className="motion-hero-orbit" viewBox="0 0 280 280" aria-hidden="true">
    {[85, 110, 135].map((radius, index) => <m.circle key={radius} cx="140" cy="140" r={radius} initial={reduced ? false : { pathLength: 0, rotate: -90 }} animate={{ pathLength: 1, rotate: 0 }} transition={{ duration: reduced ? 0 : 1.1, delay: reduced ? 0 : index * .07, ease: 'easeOut' }} />)}
  </svg></>
}

export function HeroFlow({ history = [], month, surface = 'dark' }) {
  const series = useMemo(() => {
    const available = history.filter(item => item.month <= month)
    if (!available.length) return []
    const rows = new Map(available.map(item => [item.month, item]))
    const end = new Date(`${month}-01T12:00:00Z`)
    return Array.from({ length: 12 }, (_, index) => {
      const date = new Date(end)
      date.setUTCMonth(date.getUTCMonth() - 11 + index)
      const key = date.toISOString().slice(0, 7), item = rows.get(key)
      return { ...item, month: key, label: monthLabel(key), income: item?.income ?? null, expenses: item?.expenses ?? null }
    }).filter(item => item.month >= available[0].month)
  }, [history, month])
  const id = useId().replaceAll(':', ''), colors = readColors()
  const incomeColor = surface === 'paper' ? colors.GREEN : '#9de0d0'
  const expenseColor = surface === 'paper' ? colors.ACCENT : '#c7b6ff'
  if (series.filter(row => Number.isFinite(row.income) || Number.isFinite(row.expenses)).length < 2) return null
  return <div className="motion-hero-flow">
    <div className="motion-hero-flow-head"><span>Flux mensuels · {series.length} mois</span><span className="motion-hero-flow-key"><span><i style={{ background: incomeColor }} />Revenus</span><span><i style={{ background: expenseColor }} />Dépenses</span></span></div>
    <AnimatedChart revision={series} aria-hidden="true">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={series} margin={{ top: 8, bottom: 3, right: 5, left: 5 }}>
          <defs>{softGradient(`${id}-income`, incomeColor, .18, 0)}{softGradient(`${id}-expenses`, expenseColor, .12, 0)}</defs>
          <Tooltip content={({ payload, ...props }) => <ChartTooltip {...props} payload={payload} label={payload?.[0]?.payload.label} colors={colors} />} />
          <Area isAnimationActive={false} connectNulls={false} type="linear" dataKey="income" name="Revenus" stroke={incomeColor} fill={`url(#${id}-income)`} strokeWidth={1.8} dot={{ r: 1.8, strokeWidth: 0, fill: incomeColor }} activeDot={<ChartFocusDot />} />
          <Area isAnimationActive={false} connectNulls={false} type="linear" dataKey="expenses" name="Dépenses" stroke={expenseColor} fill={`url(#${id}-expenses)`} strokeWidth={1.8} dot={{ r: 1.8, strokeWidth: 0, fill: expenseColor }} activeDot={<ChartFocusDot />} />
        </AreaChart>
      </ResponsiveContainer>
    </AnimatedChart>
    <p className="motion-hero-flow-period">{series[0].label} → {series.at(-1).label} · périodes partielles incluses</p>
    <table className="motion-sr-only"><caption>Flux mensuels observés, périodes partielles incluses</caption><thead><tr><th scope="col">Mois</th><th scope="col">Revenus</th><th scope="col">Dépenses</th></tr></thead><tbody>{series.map(row => <tr key={row.month}><th scope="row">{row.label}</th><td>{Number.isFinite(row.income) ? eur(row.income) : 'Non observé'}</td><td>{Number.isFinite(row.expenses) ? eur(row.expenses) : 'Non observé'}</td></tr>)}</tbody></table>
  </div>
}
