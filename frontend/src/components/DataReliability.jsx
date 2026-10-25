import { useEffect, useId, useState } from 'react'
import { api } from '../api'
import { reliabilityDate, reliabilityStatusLabel, reliabilitySummary } from './dataReliabilityModel.mjs'
import './DataReliability.css'

function AccountCoverage({ account, current, onInspect }) {
  return (
    <details className="data-reliability-account">
      <summary>
        <span className="data-reliability-account-name">{account.name}</span>
        <span className={`data-reliability-state ${account.status === 'documented' ? '' : 'data-reliability-state-review'}`}>
          {reliabilityStatusLabel(account, current)}
        </span>
        <span className="data-reliability-expand" aria-hidden="true">⌄</span>
      </summary>
      <div className="data-reliability-account-body">
        <dl className="data-reliability-dates">
          <div><dt>Dernière transaction sur la période</dt><dd>{reliabilityDate(account.last_transaction)}</dd></div>
          <div><dt>Dernier import pour ce compte, toutes périodes</dt><dd>{reliabilityDate(account.last_import)}</dd></div>
        </dl>
        {account.gaps.length > 0 ? (
          <div>
            <h3>Périodes sans couverture validée</h3>
            <ul className="data-reliability-gaps">
              {account.gaps.map(gap => (
                <li key={gap.from}>
                  Du <time dateTime={gap.from}>{reliabilityDate(gap.from)}</time> au <time dateTime={gap.to}>{reliabilityDate(gap.to)}</time>
                </li>
              ))}
            </ul>
            <p className="data-reliability-explanation">
              Ces intervalles ne sont pas documentés par des relevés validés.
              Cela ne signifie pas nécessairement que des opérations sont absentes.
            </p>
          </div>
        ) : <p>Les relevés validés documentent la période examinée sans interruption de dates. Cela ne prouve pas l’exhaustivité de chaque extraction.</p>}
        {account.unknown_periods > 0 && (
          <p className="data-reliability-explanation">
            {account.unknown_periods} relevé(s) ont une période absente ou invalide : ils sont présentés à titre de vérification
            et ne peuvent pas être attribués avec certitude à ce mois.
          </p>
        )}
        {account.statements.length > 0 && (
          <>
            <h3>Relevés disponibles et vérifications</h3>
            <ul className="data-reliability-statements">
              {account.statements.map(st => (
                <li key={st.id}>
                  <div>
                    <strong>{st.filename}</strong>
                    <span>{st.period_known ? `${reliabilityDate(st.from)} → ${reliabilityDate(st.to)}` : 'Période inconnue ou invalide'}</span>
                  </div>
                  <span>{st.validated ? 'Équilibre comptable validé' : 'À vérifier · équilibre non confirmé'}</span>
                </li>
              ))}
            </ul>
            <button className="btn-secondary" onClick={() => onInspect(account.statements.map(st => st.id))}>
              Examiner les relevés de {account.name}
            </button>
          </>
        )}
      </div>
    </details>
  )
}

export function DataReliability({ month, onMonthChange, onInspect, onGoUpload }) {
  const [report, setReport] = useState(null)
  const [error, setError] = useState(null)
  const [attempt, setAttempt] = useState(0)
  const titleId = useId()
  const monthId = useId()
  const now = new Date()
  const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  useEffect(() => {
    let active = true
    setReport(null)
    setError(null)
    const abort = new AbortController()
    api.reliability(month, { signal: abort.signal }).then(value => { if (active) setReport(value) })
      .catch(err => { if (active) setError(err.message) })
    return () => { active = false; abort.abort() }
  }, [month, attempt])
  const summary = report ? reliabilitySummary(report) : null

  return (
    <section className="data-reliability" aria-labelledby={titleId}>
      <header className="data-reliability-header">
        <div>
          <h2 id={titleId}>Couverture documentaire</h2>
          <p>Vérifie les périodes documentées par tes relevés, compte par compte.</p>
        </div>
        <div className="data-reliability-month">
          <label htmlFor={monthId}>Mois à examiner</label>
          <input id={monthId} type="month" value={month} max={currentMonth} onChange={e => {
            if (/^\d{4}-\d{2}$/.test(e.target.value) && e.target.value <= currentMonth) onMonthChange(e.target.value)
          }} />
        </div>
      </header>
      {error ? (
        <div role="alert">
          <p>{error}</p>
          <button className="btn-secondary" onClick={() => setAttempt(n => n + 1)}>Réessayer</button>
        </div>
      ) : !report ? <p role="status">Vérification de la couverture…</p> : (
        <>
          <div className="data-reliability-overview" aria-live="polite">
            <p><strong>{summary.documented} / {summary.total}</strong> comptes soumis au suivi documentaire couverts{report.current ? ' à date' : ' sur le mois'}</p>
            <p>
              <strong>{summary.toReview}</strong> relevés à vérifier
              {summary.unknownPeriods > 0 && <span> · {summary.unknownPeriods} à période indéterminée</span>}
            </p>
            <p>
              {report.current ? 'Mois en cours · Vérification arrêtée au ' : 'Période examinée jusqu’au '}
              <time dateTime={report.through}>{reliabilityDate(report.through)}</time>
            </p>
          </div>
          {report.accounts.some(a => a.tracking_mode === 'manual') && <p className="data-reliability-explanation">{report.accounts.filter(a => a.tracking_mode === 'manual').length} compte(s) suivis par pointage, exclus du dénominateur documentaire.</p>}
          {report.current && (
            <p className="data-reliability-explanation">
              Les jours à venir ne sont pas considérés comme manquants.
              Même documentés à date, les flux du mois restent provisoires.
            </p>
          )}
          {summary.total === 0 ? (
            <p>Aucun compte soumis au suivi documentaire. Les comptes suivis par pointage ne nécessitent pas de relevé mensuel.</p>
          ) : (
            <div className="data-reliability-accounts">
              {report.accounts.filter(account => account.tracking_mode !== 'manual').map(account => (
                <AccountCoverage key={account.id} account={account} current={report.current} onInspect={onInspect} />
              ))}
            </div>
          )}
          <footer className="data-reliability-footer">
            <p>
              La date du dernier import et la dernière transaction ne prouvent pas la complétude.
              Seules les périodes réconciliées de relevés validés sont prises en compte. Les pointages et valorisations manuels restent dans Comptes et sont exclus de ce contrôle.
            </p>
            {onGoUpload && <button className="btn-secondary" onClick={onGoUpload}>Importer un relevé</button>}
          </footer>
        </>
      )}
    </section>
  )
}
