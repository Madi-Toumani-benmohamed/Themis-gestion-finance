import { useEffect, useId, useState } from 'react'
import { api, eur } from '../api'
import { financialSummaryModel } from './financialSummaryModel.mjs'
import './FinancialSummary.css'

const dateLabel = value => value ? new Date(value.slice(0, 10) + 'T12:00:00').toLocaleDateString('fr-FR', { day: 'numeric', month: 'long', year: 'numeric' }) : null
const periodLabel = month => new Date(`${month}-01T12:00:00`).toLocaleDateString('fr-FR', { month: 'long', year: 'numeric' })
const amount = value => Number.isFinite(value) ? eur(value) : 'Indisponible'

export function FinancialSummary({ selectedMonth, onInspectCoverage }) {
  const [response, setResponse] = useState(null)
  const titleId = useId()
  useEffect(() => {
    let active = true
    setResponse(null)
    api.hero(selectedMonth).then(h => { if (active) setResponse(h) })
      .catch(() => { if (active) setResponse({ available: false }) })
    return () => { active = false }
  }, [selectedMonth])

  if (!response) return <section className="financial-summary" aria-label="Synthèse financière" aria-busy="true"><p>Chargement de la synthèse…</p></section>
  if (!response.available) return <section className="financial-summary" aria-label="Synthèse financière"><p>La synthèse financière est indisponible pour cette période.</p></section>
  const h = response
  const model = financialSummaryModel(h)
  const renderPoint = (point, index) => (
    <li className={`financial-summary-point financial-summary-point-${point.tone}`} key={`${point.kind}-${index}`}>
      <span className="financial-summary-mark" aria-hidden="true">{point.tone === 'alert' ? '!' : point.tone === 'attention' ? '△' : '✓'}</span>
      <span>{point.text}</span>
    </li>
  )

  return (
    <section className="financial-summary" aria-labelledby={titleId}>
      <header className="financial-summary-header">
        <div><h2 id={titleId}>{periodLabel(h.month || selectedMonth)}</h2><p className="financial-summary-context">{model.context}</p></div>
        <span className={`financial-summary-status financial-summary-status-${model.tone}`}><span aria-hidden="true">●</span>{model.status}</span>
      </header>
      <div className="financial-summary-body">
        <div className="financial-summary-result">
          <div className="financial-summary-label">{model.resultLabel}</div>
          <div className="financial-summary-amount">{amount(h.net)}</div>
          <p className="financial-summary-diagnosis">{h.headline}</p>
          <dl className="financial-summary-flows">
            <div><dt>Revenus</dt><dd>{amount(h.income)}</dd></div>
            <div><dt>Dépenses</dt><dd>{amount(h.expenses)}</dd></div>
          </dl>
          <p className="financial-summary-note">Flux enregistrés sur la période · Ce résultat ne représente pas le solde bancaire.</p>
        </div>
        <div className="financial-summary-attention">
          <h3>Points d’attention</h3>
          {model.attention.length ? <ul className="financial-summary-points">{model.attention.map(renderPoint)}</ul> : <p className="financial-summary-empty">{model.complete ? 'Aucune alerte détectée sur les données disponibles.' : 'Aucune alerte établie à ce stade. Les flux de la période restent incomplets.'}</p>}
          {model.observations.length > 0 && <div className="financial-summary-observations"><h4>Repères</h4><ul className="financial-summary-points">{model.observations.map(renderPoint)}</ul></div>}
        </div>
      </div>
      <footer className="financial-summary-footer">
        <span>{h.coverage?.last_tx ? <>Dernière transaction disponible le <time dateTime={h.coverage.last_tx}>{dateLabel(h.coverage.last_tx)}</time></> : 'Date de couverture indisponible'}</span>
        {h.coverage?.last_import && <span>Dernier import le <time dateTime={h.coverage.last_import}>{dateLabel(h.coverage.last_import)}</time></span>}
        {h.coverage?.stale && <span>Un relevé récent manque peut-être.</span>}
        {!model.current && !model.complete && <span>Les données disponibles ne permettent pas de conclure un bilan mensuel définitif.</span>}
        {onInspectCoverage && <button className="link" onClick={onInspectCoverage}>Examiner la fiabilité des données →</button>}
      </footer>
    </section>
  )
}
