import { useEffect, useId, useState } from 'react'
import { api } from '../api'
import { financialSummaryModel } from './financialSummaryModel.mjs'
import './FinancialSummary.css'
import { Logo } from '../Logo'
import { m, useMotionReduced, ease } from '../motion'
import { AnimatedNumber } from '../motion/AnimatedNumber'
import { HeroAtmosphere, HeroFlow } from '../motion/HeroFlow'

const dateLabel = value => value ? new Date(value.slice(0, 10) + 'T12:00:00').toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' }) : null
const periodLabel = month => new Date(`${month}-01T12:00:00`).toLocaleDateString('fr-FR', { month: 'long', year: 'numeric' })

export function FinancialSummary({ selectedMonth, onInspectCoverage, history = [] }) {
  const [snapshot, setSnapshot] = useState(null)
  const reduced = useMotionReduced()
  const response = snapshot?.value
  const shownMonth = snapshot?.month || selectedMonth
  const pending = shownMonth !== selectedMonth
  const titleId = useId()
  useEffect(() => {
    let active = true
    api.hero(selectedMonth).then(h => { if (active) setSnapshot({ month: selectedMonth, value: h }) })
      .catch(() => { if (active) setSnapshot({ month: selectedMonth, value: { available: false } }) })
    return () => { active = false }
  }, [selectedMonth])

  if (!response) return <section className="financial-summary financial-summary-loading" aria-label="Synthèse financière" aria-busy="true"><p role="status">Chargement de la synthèse…</p><span className="motion-skeleton" aria-hidden="true" /><span className="motion-skeleton" aria-hidden="true" /></section>
  if (!response.available) return <section className="financial-summary" aria-label="Synthèse financière" data-period={shownMonth} aria-busy={pending}><p role="status">{pending ? 'Chargement de la synthèse…' : 'La synthèse financière est indisponible pour cette période.'}</p></section>
  const h = response
  const model = financialSummaryModel(h)
  const renderPoint = (point, index) => (
    <m.li initial={reduced ? false : { opacity: 0, x: 12 }} animate={{ opacity: 1, x: 0 }} transition={{ duration: reduced ? 0 : .42, ease, delay: reduced ? 0 : .2 + Math.min(index, 4) * .06 }} className={`financial-summary-point financial-summary-point-${point.tone}`} key={`${point.kind}-${index}`}>
      <span className="financial-summary-mark" aria-hidden="true">{point.tone === 'alert' ? '!' : point.tone === 'attention' ? '↗' : point.tone === 'positive' ? '✓' : '·'}</span>
      <span>{point.text}</span>
    </m.li>
  )

  return (
    <section className="financial-summary" aria-labelledby={titleId} data-period={shownMonth} data-pending={pending} aria-busy={pending}>
      <header className="financial-summary-header">
        <div><h2 id={titleId}>{periodLabel(h.month || shownMonth)}</h2><p className={`financial-summary-context${pending ? ' motion-period-state motion-period-inline' : ''}`} role={pending ? 'status' : undefined}>{pending ? `Actualisation… Données affichées : ${periodLabel(shownMonth)}.` : model.context}</p></div>
        <span className={`financial-summary-status financial-summary-status-${model.tone}`}><span aria-hidden="true">●</span>{model.status}</span>
      </header>
      <div className="financial-summary-body">
        <m.div className="financial-summary-result" initial={reduced ? false : { opacity: 0, y: 14 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: reduced ? 0 : .6, ease }}><HeroAtmosphere />
          <Logo size={176} className="financial-summary-watermark" />
          <div className="financial-summary-label">{model.resultLabel}</div>
          <div className="financial-summary-amount"><AnimatedNumber value={h.net} missing="Indisponible" delay={80} /></div>
          <p className="financial-summary-diagnosis">{h.headline}</p>
          <dl className="financial-summary-flows">
            <div><dt><span className="financial-flow-icon" aria-hidden="true">↙</span> Revenus</dt><dd><AnimatedNumber value={h.income} missing="Indisponible" delay={140} /></dd></div>
            <div><dt><span className="financial-flow-icon" aria-hidden="true">↗</span> Dépenses</dt><dd><AnimatedNumber value={h.expenses} missing="Indisponible" delay={180} /></dd></div>
          </dl>
          <p className="financial-summary-note">Flux enregistrés sur la période · Ce résultat ne représente pas le solde bancaire.</p>
        </m.div>
        <div className="financial-summary-attention">
          <div className="financial-summary-attention-heading"><h3>Points d’attention</h3><span className="financial-summary-count">{model.attention.length}</span></div>
          {model.attention.length ? <ul className="financial-summary-points">{model.attention.map(renderPoint)}</ul> : <p className="financial-summary-empty">{model.complete ? 'Aucune alerte détectée sur les données disponibles.' : 'Aucune alerte établie à ce stade. Les flux de la période restent incomplets.'}</p>}
          {model.observations.length > 0 && <div className="financial-summary-observations"><h4>Repères</h4><ul className="financial-summary-points">{model.observations.map(renderPoint)}</ul></div>}
          <HeroFlow history={history} month={shownMonth} surface="paper" />
        </div>
      </div>
      <footer className="financial-summary-footer">
        <span>{h.coverage?.last_tx ? <>Dernière transaction disponible le <time dateTime={h.coverage.last_tx}>{dateLabel(h.coverage.last_tx)}</time></> : 'Date de couverture indisponible'}</span>
        {h.coverage?.last_import && <span>Dernier import le <time dateTime={h.coverage.last_import}>{dateLabel(h.coverage.last_import)}</time></span>}
        {h.coverage?.stale && <span>Un relevé récent manque peut-être.</span>}
        {!model.current && !model.complete && <span>Les données disponibles ne permettent pas de conclure un bilan mensuel définitif.</span>}
        {onInspectCoverage && <button className="link" onClick={() => onInspectCoverage(shownMonth)}>Examiner la fiabilité des données →</button>}
      </footer>
    </section>
  )
}
