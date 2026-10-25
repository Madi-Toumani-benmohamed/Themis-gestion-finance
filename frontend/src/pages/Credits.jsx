import { useEffect, useState } from 'react'
import { api } from '../api'
import { useToast } from '../Toast'
import {
  CreditDialog,
  CreditHistory,
  ArchivedEngagements,
} from '../components/CreditDialogs'
import {
  amountCents,
  calculateProgress,
  formatDate,
  formatMonth,
  moneyText,
  sortCredits,
  sortTaxes,
  statusLabel,
} from '../components/creditsPageModel.mjs'
import '../Credits.css'

export default function Credits() {
  const toast = useToast()
  const [report, setReport] = useState(null),
    [error, setError] = useState(''),
    [reload, setReload] = useState(0)
  const [action, setAction] = useState(null),
    [history, setHistory] = useState(null)
  const [creditSort, setCreditSort] = useState('remaining'),
    [taxSort, setTaxSort] = useState('due'),
    [filter, setFilter] = useState('all')
  useEffect(() => {
    const abort = new AbortController()
    setReport(null)
    setError('')
    api
      .obligations({ signal: abort.signal })
      .then((data) => {
        if (
          !Array.isArray(data.credits) ||
          !Array.isArray(data.taxes) ||
          !data.amounts_cents
        )
          throw new Error(
            'Rapport incomplet : les engagements ne peuvent pas être affichés de façon fiable.',
          )
        setReport(data)
      })
      .catch((e) => {
        if (e.name !== 'AbortError') setError(e.message)
      })
    return () => abort.abort()
  }, [reload])
  const refresh = () => {
    setReport(null)
    setReload((n) => n + 1)
  }
  const saved = () => {
    setAction(null)
    refresh()
    toast.ok('Déclaration enregistrée. Actualisation de la synthèse…')
  }
  const open = (kind, mode, row) => setAction({ kind, mode, row })
  const credits = sortCredits(report?.credits, creditSort).filter(
    (c) =>
      filter === 'all' ||
      (filter === 'open'
        ? !['paid', 'closed'].includes(c.status)
        : ['paid', 'closed'].includes(c.status)),
  )
  return (
    <div className="credits-page">
      <header className="credits-context">
        <h2 tabIndex={-1}>Mes engagements</h2>
        <p className="muted">
          {report
            ? `Déclarations au ${formatDate(report.as_of)}.`
            : 'Crédits, taxes et autres échéances.'}{' '}
          Ces montants décrivent vos engagements, pas les dépenses bancaires
          observées.
        </p>
      </header>
      {error && (
        <div className="notice err" role="alert">
          {error}
          <div>
            <button type="button" className="btn-secondary" onClick={refresh}>
              Réessayer
            </button>
          </div>
        </div>
      )}
      {!report && !error && (
        <p role="status">
          <span className="spinner" />
          Chargement des engagements…
        </p>
      )}
      {report && (
        <>
          <div
            className="obligations-banner"
            aria-label="Synthèse des engagements déclarés"
          >
            <div className="obligation-card">
              <span className="obligation-label">Solde nominal déclaré</span>
              <strong className="obligation-value">
                {moneyText(report, 'total_debt')}
              </strong>
              {report.total_debt == null && (
                <span className="obligation-sub">
                  Sous-total documenté : {moneyText(report, 'known_total_debt')}
                </span>
              )}
            </div>
            <div className="obligation-card">
              <span className="obligation-label">
                Crédits · restant déclaré
              </span>
              <strong className="obligation-value">
                {moneyText(report, 'credits_remaining')}
              </strong>
              <span className="obligation-sub">
                Mensualités prévues ce mois :{' '}
                {moneyText(report, 'monthly_load')}
              </span>
            </div>
            <div className="obligation-card">
              <span className="obligation-label">
                Taxes et échéances · restant
              </span>
              <strong className="obligation-value">
                {moneyText(report, 'taxes_due')}
              </strong>
              <span className="obligation-sub">
                {report.taxes_due_soon} à échéance sous 30 jours ·{' '}
                {report.taxes_to_verify} paiement(s) à vérifier
              </span>
            </div>
          </div>
          {(report.unknown_credits_count > 0 ||
            report.invalid_credits_count > 0) && (
            <p className="notice" role="status">
              {report.unknown_credits_count} crédit(s) sans historique de
              paiement complet ; {report.invalid_credits_count} à vérifier. Le
              temps écoulé ne prouve pas qu’un paiement a été effectué. Le solde
              global reste non documenté.
            </p>
          )}
          <section aria-labelledby="credits-title">
            <div className="section-header">
              <h2 id="credits-title" className="section-title">
                Crédits
              </h2>
              <button
                className="btn-secondary"
                onClick={() => open('credits', 'create')}
              >
                ＋ Crédit
              </button>
            </div>
            <div className="credits-tools">
              <label>
                Afficher{' '}
                <select
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                >
                  <option value="all">Tous les crédits</option>
                  <option value="open">À suivre</option>
                  <option value="paid">Payés et clôturés</option>
                </select>
              </label>
              <label>
                Trier les crédits{' '}
                <select
                  value={creditSort}
                  onChange={(e) => setCreditSort(e.target.value)}
                >
                  <option value="remaining">Solde décroissant</option>
                  <option value="name">Nom</option>
                  <option value="progress">Progression déclarée</option>
                  <option value="end">Fin prévue</option>
                </select>
              </label>
            </div>
            {!credits.length ? (
              <p className="empty-state">
                {report.credits.length
                  ? 'Aucun crédit pour ce filtre.'
                  : 'Aucun crédit enregistré.'}
              </p>
            ) : (
              <div className="credits-grid">
                {credits.map((c) => {
                  const pct = calculateProgress(c)
                  return (
                    <article
                      className="credit-card"
                      key={c.id}
                      aria-label={c.name}
                    >
                      <div className="credit-header">
                        <div>
                          <h3 className="credit-name">{c.name}</h3>
                          <span className={`due-chip ${c.status}`}>
                            {statusLabel(c.status)}
                          </span>
                        </div>
                        <button
                          className="btn-secondary"
                          aria-label={`Modifier ${c.name}`}
                          onClick={() => open('credits', 'edit', c)}
                        >
                          Modifier
                        </button>
                      </div>
                      {c.note && <p className="credit-note">{c.note}</p>}
                      {c.issue && <p className="notice">{c.issue}</p>}
                      <dl className="credit-details">
                        <div>
                          <dt>Mensualité prévue</dt>
                          <dd>{moneyText(c, 'monthly_amount')}</dd>
                        </div>
                        <div>
                          <dt>Solde déclaré</dt>
                          <dd>{moneyText(c, 'remaining')}</dd>
                        </div>
                        <div>
                          <dt>Total nominal prévu</dt>
                          <dd>{moneyText(c, 'total_scheduled')}</dd>
                        </div>
                        <div>
                          <dt>Total payé déclaré</dt>
                          <dd>{moneyText(c, 'paid_total')}</dd>
                        </div>
                      </dl>
                      <div className="progress-container">
                        {pct == null ? (
                          <p className="muted">Progression non documentée.</p>
                        ) : (
                          <>
                            <progress
                              max="100"
                              value={pct}
                              aria-label={`Progression déclarée de ${c.name}`}
                            />
                            <p className="progress-info">
                              {pct.toLocaleString('fr-FR')} % remboursé selon
                              vos déclarations
                            </p>
                          </>
                        )}
                      </div>
                      <p className="credit-note">
                        Échéancier : {c.months_total} mois ·{' '}
                        {formatMonth(c.start_month)} à{' '}
                        {formatMonth(c.end_month)}.
                        {c.next_scheduled_month &&
                          ` Échéance calendaire à vérifier : ${formatMonth(c.next_scheduled_month)}.`}
                      </p>
                      <p className="credit-note">
                        {moneyText(c, 'payments_recorded')} enregistrés ici (
                        {c.payment_count} paiement(s)). Historique antérieur :{' '}
                        {moneyText(c, 'opening_paid')}.
                      </p>
                      <div className="credit-actions">
                        {!['closed', 'paid', 'invalid'].includes(c.status) && (
                          <button
                            className="btn-secondary"
                            onClick={() => open('credits', 'payment', c)}
                          >
                            Déclarer un paiement
                          </button>
                        )}
                        {c.status === 'paid' && (
                          <button
                            className="btn-secondary"
                            onClick={() => open('credits', 'close', c)}
                          >
                            Clôturer
                          </button>
                        )}
                        {c.status === 'closed' && (
                          <button
                            className="btn-secondary"
                            onClick={() => open('credits', 'reopen', c)}
                          >
                            Rouvrir
                          </button>
                        )}
                        <button
                          className="btn-secondary"
                          onClick={() =>
                            setHistory({ kind: 'credits', row: c })
                          }
                        >
                          Historique
                          <span className="sr-only"> de {c.name}</span>
                        </button>
                        <button
                          className="link"
                          aria-label={`Retirer ${c.name}`}
                          onClick={() => open('credits', 'delete', c)}
                        >
                          Retirer
                        </button>
                      </div>
                    </article>
                  )
                })}
              </div>
            )}
          </section>
          <section aria-labelledby="taxes-title">
            <div className="section-header">
              <h2 id="taxes-title" className="section-title">
                Taxes & autres échéances
              </h2>
              <button
                className="btn-secondary"
                onClick={() => open('taxes', 'create')}
              >
                ＋ Échéance
              </button>
            </div>
            <div className="credits-tools">
              <label>
                Trier les échéances{' '}
                <select
                  value={taxSort}
                  onChange={(e) => setTaxSort(e.target.value)}
                >
                  <option value="due">Date d’échéance</option>
                  <option value="amount">Montant décroissant</option>
                  <option value="name">Nom</option>
                </select>
              </label>
            </div>
            <p className="muted">
              Une date dépassée invite à vérifier le paiement. Elle ne confirme
              pas un retard.
            </p>
            {!report.taxes.length ? (
              <p className="empty-state">Aucune échéance enregistrée.</p>
            ) : (
              <div className="taxes-list">
                <table className="taxes-table">
                  <caption className="sr-only">
                    Échéances et paiements déclarés
                  </caption>
                  <thead>
                    <tr>
                      <th scope="col">Engagement</th>
                      <th scope="col">Montants déclarés</th>
                      <th scope="col">Échéance et statut</th>
                      <th scope="col">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sortTaxes(report.taxes, taxSort).map((t) => (
                      <tr key={t.id}>
                        <th scope="row" className="tax-name">
                          {t.name}
                        </th>
                        <td className="tax-amount">
                          {moneyText(t, 'remaining')} restant
                          <small>
                            {moneyText(t, 'paid_amount')} payé sur{' '}
                            {moneyText(t, 'amount')}
                          </small>
                        </td>
                        <td className="tax-due">
                          {formatDate(t.due_date)}
                          <span className={`due-chip ${t.status}`}>
                            {statusLabel(t.status)}
                          </span>
                          {t.payment_source === 'legacy_confirmation' && (
                            <small>Ancienne confirmation de paiement</small>
                          )}
                          {t.issue && <small>{t.issue}</small>}
                        </td>
                        <td>
                          <div className="credit-actions">
                            {!t.paid && (
                              <button
                                className="btn-secondary"
                                aria-label={`Déclarer un paiement pour ${t.name}`}
                                onClick={() => open('taxes', 'payment', t)}
                              >
                                Paiement
                              </button>
                            )}
                            <button
                              className="btn-secondary"
                              aria-label={`Modifier ${t.name}`}
                              onClick={() => open('taxes', 'edit', t)}
                            >
                              Modifier
                            </button>
                            <button
                              className="btn-secondary"
                              aria-label={`Historique de ${t.name}`}
                              onClick={() =>
                                setHistory({ kind: 'taxes', row: t })
                              }
                            >
                              Historique
                            </button>
                            {amountCents(t, 'paid_amount') > 0 && (
                              <button
                                className="link"
                                aria-label={`Corriger le payé pour ${t.name}`}
                                onClick={() => open('taxes', 'reset', t)}
                              >
                                Corriger le payé
                              </button>
                            )}
                            {!t.paid && t.days_left < 0 && (
                              <button
                                className="link"
                                onClick={() => open('taxes', 'late', t)}
                              >
                                {t.overdue_confirmed
                                  ? 'Retirer la confirmation de retard'
                                  : 'Confirmer un retard'}
                              </button>
                            )}
                            <button
                              className="link"
                              aria-label={`Retirer ${t.name}`}
                              onClick={() => open('taxes', 'delete', t)}
                            >
                              Retirer
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
          <p className="credit-note">
            Les soldes reposent sur les montants et paiements que vous déclarez.
            Le total nominal d’un crédit correspond à mensualité × durée ; il ne
            constitue pas un capital bancaire certifié. Aucun paiement déclaré
            ici n’est ajouté aux transactions, budgets ou analyses.
          </p>
        </>
      )}
      {report && <ArchivedEngagements onHistory={setHistory} />}
      {action && (
        <CreditDialog
          key={`${action.kind}-${action.mode}-${action.row?.id || 'new'}`}
          action={action}
          onClose={() => setAction(null)}
          onSaved={saved}
        />
      )}
      {history && (
        <CreditHistory
          kind={history.kind}
          row={history.row}
          onClose={() => setHistory(null)}
          onSaved={refresh}
        />
      )}
    </div>
  )
}
