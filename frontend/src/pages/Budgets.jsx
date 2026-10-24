import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { api, monthLabel } from '../api'
import { useToast } from '../Toast'
import {
  budgetCents,
  budgetEuro as eur,
  budgetDate,
  budgetMonth,
  budgetPageModel,
  budgetPercent,
  recurringNatures,
  recurringNatureLabel,
  recurringStatusLabel,
  recurringFrequencyLabel,
} from '../components/budgetPageModel.mjs'
import './Budgets.css'

function BudgetDialog({ mode, row, domains, onSave, onClose }) {
  const ref = useRef(null),
    title = useId()
  const [domain, setDomain] = useState(row?.domain_id ?? '')
  const [limit, setLimit] = useState(row?.monthly_limit ?? '')
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(null)
  const deleting = mode === 'delete'
  useEffect(() => {
    const opener = document.activeElement
    ref.current?.showModal()
    return () => {
      ref.current?.close()
      opener?.focus()
    }
  }, [])
  const close = () => {
    if (!busy) {
      ref.current?.close()
      onClose()
    }
  }
  const save = async (event) => {
    event.preventDefault()
    setError(null)
    try {
      if (!deleting && (!domain || budgetCents(limit) <= 0))
        throw new Error(
          'Sélectionne une catégorie et un plafond positif, au centime près.',
        )
      setBusy(true)
      await onSave(
        deleting
          ? row
          : {
              domain_id: Number(domain),
              monthly_limit: budgetCents(limit) / 100,
            },
      )
      ref.current?.close()
      onClose()
    } catch (e) {
      setError(e.message)
      setBusy(false)
    }
  }
  return (
    <dialog
      ref={ref}
      className="budget-dialog"
      aria-labelledby={title}
      onCancel={(event) => {
        event.preventDefault()
        close()
      }}
    >
      <form onSubmit={save}>
        <header>
          <h2 id={title}>
            {deleting
              ? 'Supprimer cette enveloppe ?'
              : row
                ? 'Modifier le plafond'
                : 'Créer une enveloppe'}
          </h2>
          <button
            type="button"
            className="link"
            onClick={close}
            disabled={busy}
            aria-label="Fermer le dialogue"
          >
            ✕
          </button>
        </header>
        {error && (
          <p role="alert" className="notice err">
            {error}
          </p>
        )}
        {deleting ? (
          <p>
            Le plafond de <strong>{row.domain_name}</strong> sera supprimé. Les
            transactions et leurs catégories restent conservées.
          </p>
        ) : (
          <>
            <label>
              Catégorie
              <select
                value={domain}
                onChange={(event) => setDomain(event.target.value)}
                disabled={busy || Boolean(row)}
                required
              >
                <option value="">Choisir une catégorie</option>
                {domains.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.parent_id ? '↳ ' : ''}
                    {d.name}
                    {d.is_transfer ? ' · transfert / épargne' : ''}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Plafond mensuel (€)
              <input
                type="number"
                min="0.01"
                max="9999999999.99"
                step="0.01"
                inputMode="decimal"
                value={limit}
                onChange={(event) => setLimit(event.target.value)}
                autoFocus={Boolean(row)}
                disabled={busy}
                required
              />
            </label>
            <p className="budget-note">
              Ce plafond s’applique chaque mois. Le modifier change aussi la
              référence affichée pour les mois historiques ; il ne modifie
              aucune opération.
            </p>
          </>
        )}
        <footer>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={close}
            disabled={busy}
          >
            Annuler
          </button>
          <button
            className={deleting ? 'btn-danger' : 'btn'}
            type="submit"
            disabled={busy}
          >
            {busy
              ? 'Enregistrement…'
              : deleting
                ? 'Confirmer la suppression'
                : row
                  ? 'Enregistrer le plafond'
                  : 'Créer l’enveloppe'}
          </button>
        </footer>
      </form>
    </dialog>
  )
}
function RecurrenceDialog({ item, month, onSaved, onClose }) {
  const ref = useRef(null),
    title = useId()
  const [nature, setNature] = useState(item.nature),
    [status, setStatus] = useState(item.status),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(null)
  useEffect(() => {
    const opener = document.activeElement
    ref.current?.showModal()
    return () => {
      ref.current?.close()
      opener?.focus()
    }
  }, [])
  const close = () => {
    if (!busy) {
      ref.current?.close()
      onClose()
    }
  }
  const save = async (event) => {
    event.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await api.updateRecurringReview(item.key, { status, nature }, month)
      onSaved()
      ref.current?.close()
      onClose()
    } catch (e) {
      setError(e.message)
      setBusy(false)
    }
  }
  return (
    <dialog
      ref={ref}
      className="budget-dialog"
      aria-labelledby={title}
      onCancel={(event) => {
        event.preventDefault()
        close()
      }}
    >
      <form onSubmit={save}>
        <header>
          <h2 id={title}>Examiner la détection</h2>
          <button
            type="button"
            className="link"
            aria-label="Fermer le dialogue"
            disabled={busy}
            onClick={close}
          >
            ✕
          </button>
        </header>
        <p>
          <strong>{item.label}</strong>
        </p>
        {error && (
          <p role="alert" className="notice err">
            {error}
          </p>
        )}
        <label>
          Nature économique
          <select
            value={nature}
            disabled={busy}
            onChange={(event) => setNature(event.target.value)}
          >
            {Object.entries(recurringNatures).map(([key, label]) => (
              <option key={key} value={key}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <label>
          Décision
          <select
            value={status}
            disabled={busy}
            onChange={(event) => setStatus(event.target.value)}
          >
            <option value="detected">À vérifier</option>
            <option value="confirmed">Confirmer cette récurrence</option>
            <option value="ignored">Ignorer cette détection</option>
          </select>
        </label>
        <p className="budget-note">
          La décision porte sur cette détection, pas sur les transactions. Une
          confirmation ne prouve ni la complétude des relevés ni la certitude
          d’une prochaine échéance. Les transferts ne peuvent pas être
          transformés en abonnements.
        </p>
        <footer>
          <button
            className="btn btn-secondary"
            type="button"
            disabled={busy}
            onClick={close}
          >
            Annuler
          </button>
          <button className="btn" disabled={busy} type="submit">
            {busy ? 'Enregistrement…' : 'Enregistrer la décision'}
          </button>
        </footer>
      </form>
    </dialog>
  )
}
function BudgetMetric({ label, value, note }) {
  return (
    <div className="budget-metric">
      <dt>{label}</dt>
      <dd>
        {value}
        <small>{note}</small>
      </dd>
    </div>
  )
}
function Envelope({ row, onEdit, onDelete, onInspect }) {
  const remaining = row.remainingCents
  return (
    <article className={`budget-envelope budget-envelope-${row.status}`}>
      <div className="budget-envelope-main">
        <div>
          <h4>{row.domain_name}</h4>
          <span className={`budget-state budget-state-${row.status}`}>
            {row.status === 'over'
              ? 'Plafond dépassé'
              : row.status === 'reached'
                ? 'Plafond atteint'
                : 'Disponible dans l’enveloppe'}
          </span>
        </div>
        <div className="budget-envelope-value">
          <strong>{eur(row.spentCents)}</strong>
          <span>sur {eur(row.limit)} de plafond</span>
        </div>
      </div>
      <div
        className="budget-progress"
        role="meter"
        aria-label={`Consommation de ${row.domain_name}`}
        aria-valuemin="0"
        aria-valuemax={row.limit}
        aria-valuenow={Math.min(row.spentCents, row.limit)}
        aria-valuetext={`${budgetPercent(row.ratio)} consommés, ${eur(Math.abs(remaining))} ${remaining < 0 ? 'de dépassement' : 'disponibles'}`}
      >
        <span style={{ width: `${row.width}%` }} />
      </div>
      <div className="budget-envelope-balance">
        <span>
          {remaining < 0
            ? `Dépassement : ${eur(-remaining)}`
            : remaining === 0
              ? 'Aucun montant disponible'
              : `${eur(remaining)} disponibles`}
        </span>
        <strong>{budgetPercent(row.ratio)}</strong>
      </div>
      {row.creditCents > 0 && (
        <p className="budget-note">
          {eur(row.creditCents)} de crédits observés, présentés séparément des
          débits.
        </p>
      )}
      {row.transaction_count > 1000 && (
        <p className="budget-note">
          Cette enveloppe compte {row.transaction_count} opérations sources ; la
          liste affiche au maximum les 1 000 premières. Le total budgétaire
          inclut toutes les sources.
        </p>
      )}
      <div className="budget-envelope-actions">
        <button className="link" onClick={onInspect}>
          Voir les transactions
        </button>
        <span>
          <button className="link" onClick={onEdit}>
            Modifier le plafond
          </button>
          <button className="link" onClick={onDelete}>
            Supprimer
          </button>
        </span>
      </div>
    </article>
  )
}
function Recurrence({ item, onReview, onInspect }) {
  const ids = item.transaction_ids ?? []
  return (
    <details className="budget-recurrence">
      <summary>
        <span>
          <strong>{item.label}</strong>
          <small>
            {item.direction === 'credit'
              ? 'Entrée · '
              : item.direction === 'debit'
                ? 'Sortie · '
                : ''}
            {recurringStatusLabel(item.status)} ·{' '}
            {item.confidence === 'probable'
              ? 'Récurrence probable'
              : 'Documentation insuffisante'}
          </small>
        </span>
        <span className="budget-rec-amount">
          {item.typical_amount == null
            ? 'Montant non établi'
            : eur(budgetCents(item.typical_amount))}
          <small>médiane par occurrence</small>
        </span>
        <span className="budget-rec-chevron" aria-hidden="true">
          ⌄
        </span>
      </summary>
      <div className="budget-recurrence-body">
        <dl className="budget-rec-facts">
          <div>
            <dt>Nature économique</dt>
            <dd>{recurringNatureLabel(item.nature)}</dd>
          </div>
          <div>
            <dt>Sens du mouvement</dt>
            <dd>
              {item.direction === 'credit'
                ? 'Entrée / crédit'
                : item.direction === 'debit'
                  ? 'Sortie / débit'
                  : 'Non établi'}
            </dd>
          </div>
          <div>
            <dt>Période de couverture examinée</dt>
            <dd>
              {budgetDate(item.coverage_start)} au{' '}
              {budgetDate(item.coverage_end)}
            </dd>
          </div>
          <div>
            <dt>Dernière occurrence</dt>
            <dd>
              {budgetDate(item.last_date)} ·{' '}
              {item.last_amount == null
                ? '—'
                : eur(budgetCents(item.last_amount))}
            </dd>
          </div>
          <div>
            <dt>Fréquence</dt>
            <dd>{recurringFrequencyLabel(item.frequency)}</dd>
          </div>
          <div>
            <dt>Prochaine échéance estimée</dt>
            <dd>{budgetDate(item.estimated_next_date)}</dd>
          </div>
          <div>
            <dt>Observations</dt>
            <dd>
              {item.occurrence_count} occurrence(s) sur {item.months_count} mois
            </dd>
          </div>
          <div>
            <dt>Moyenne par opération</dt>
            <dd>
              {item.average_per_transaction == null
                ? 'Non disponible'
                : eur(budgetCents(item.average_per_transaction))}{' '}
              · diviseur : {item.average_denominator ?? item.occurrence_count}
            </dd>
          </div>
          <div>
            <dt>Couverture documentaire</dt>
            <dd>
              {item.coverage_complete
                ? 'Documentée sur la période examinée'
                : 'Partielle ou inconnue'}
            </dd>
          </div>
        </dl>
        {item.confidence_reason && (
          <p className="budget-note">{item.confidence_reason}</p>
        )}
        <p className="budget-note">
          La moyenne par opération, le montant habituel et le dernier montant
          sont distincts. Aucun total mensuel ni prélèvement futur certain n’est
          déduit de cette détection.
        </p>
        <div className="budget-recurrence-actions">
          <button className="link" onClick={onReview}>
            Confirmer, corriger ou ignorer
          </button>
          <button className="link" disabled={!ids.length} onClick={onInspect}>
            {ids.length > 1000
              ? `Voir les 1 000 premières sources sur ${ids.length}`
              : 'Voir les opérations sources'}
          </button>
        </div>
      </div>
    </details>
  )
}
export default function Budgets({ onNavigate }) {
  const toast = useToast()
  const [month, setMonth] = useState(() => budgetMonth()),
    [refresh, setRefresh] = useState(0)
  const [report, setReport] = useState(null),
    [model, setModel] = useState(null),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(null)
  const [recReport, setRecReport] = useState(null),
    [recLoading, setRecLoading] = useState(true),
    [recError, setRecError] = useState(null)
  const [domains, setDomains] = useState([]),
    [domainError, setDomainError] = useState(null),
    [dialog, setDialog] = useState(null),
    [review, setReview] = useState(null)
  const [query, setQuery] = useState(''),
    [onlyOver, setOnlyOver] = useState(false),
    [showIgnored, setShowIgnored] = useState(false)
  const current = budgetMonth()
  useEffect(() => {
    let active = true
    setLoading(true)
    setError(null)
    setReport(null)
    setModel(null)
    api
      .budgetReport(month)
      .then((data) => {
        if (data.month !== month)
          throw new Error(
            'La période reçue ne correspond pas au mois consulté.',
          )
        const next = budgetPageModel(data)
        if (active) {
          setReport(data)
          setModel(next)
        }
      })
      .catch((e) => {
        if (active) setError(e.message)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [month, refresh])
  useEffect(() => {
    let active = true
    setRecLoading(true)
    setRecError(null)
    setRecReport(null)
    api
      .recurringReview(month)
      .then((data) => {
        if (!Array.isArray(data.items) || data.month !== month)
          throw new Error(
            'Rapport de récurrences incomplet ou de période incohérente.',
          )
        for (const item of data.items) {
          for (const value of [
            item.last_amount,
            item.typical_amount,
            item.average_per_transaction,
          ])
            if (value != null) budgetCents(value)
        }
        if (active) setRecReport(data)
      })
      .catch((e) => {
        if (active) setRecError(e.message)
      })
      .finally(() => {
        if (active) setRecLoading(false)
      })
    return () => {
      active = false
    }
  }, [month, refresh])
  useEffect(() => {
    let active = true
    setDomainError(null)
    api
      .domains()
      .then((data) => {
        if (active) setDomains(Array.isArray(data) ? data : [])
      })
      .catch((e) => {
        if (active) setDomainError(e.message)
      })
    return () => {
      active = false
    }
  }, [refresh])
  const availableDomains = useMemo(
    () =>
      domains.filter((d) => !report?.budgets.some((b) => b.domain_id === d.id)),
    [domains, report],
  )
  const changeMonth = (value) => {
    if (/^\d{4}-(0[1-9]|1[0-2])$/.test(value) && value <= current) {
      setMonth(value)
      setDialog(null)
      setReview(null)
    }
  }
  const reload = () => setRefresh((n) => n + 1)
  const save = async (value) => {
    if (dialog.mode === 'delete') {
      await api.deleteBudget(value.id)
      toast.ok('Enveloppe supprimée.')
    } else {
      await api.setBudget(value.domain_id, value.monthly_limit)
      toast.ok('Plafond enregistré.')
    }
    reload()
  }
  const goBudget = (row) =>
    onNavigate?.('transactions', { month, budget_id: row.id })
  const goRecurrence = (item) =>
    onNavigate?.('transactions', {
      ids: (item.transaction_ids ?? []).slice(0, 1000),
      from: recReport.observation_start,
      to: recReport.observation_end,
    })
  const filtered =
    model?.expenses.filter(
      (b) =>
        b.domain_name
          .toLocaleLowerCase('fr')
          .includes(query.toLocaleLowerCase('fr')) &&
        (!onlyOver || b.status === 'over'),
    ) ?? []
  const recItems = (recReport?.items ?? []).filter(
    (item) => showIgnored || item.status !== 'ignored',
  )
  const exportRow = (row) => (
    <Envelope
      key={row.id}
      row={row}
      onEdit={() => setDialog({ mode: 'edit', row })}
      onDelete={() => setDialog({ mode: 'delete', row })}
      onInspect={() => goBudget(row)}
    />
  )
  return (
    <div className="section budgets-page">
      <header className="budgets-heading">
        <div>
          <span className="eyebrow">Pilotage · Dépenses importées</span>
          <h2>
            Des enveloppes <em>pour décider.</em>
          </h2>
          <p>Comprends les dépenses du mois, puis ajuste tes plafonds.</p>
        </div>
        <div className="budgets-month">
          <button
            className="btn btn-secondary"
            aria-label="Mois précédent"
            onClick={() => changeMonth(budgetMonth(-1, month))}
          >
            ‹
          </button>
          <label>
            Mois consulté
            <input
              type="month"
              value={month}
              max={current}
              onChange={(e) => changeMonth(e.target.value)}
            />
          </label>
          <button
            className="btn btn-secondary"
            aria-label="Mois suivant"
            disabled={month >= current}
            onClick={() => changeMonth(budgetMonth(1, month))}
          >
            ›
          </button>
          {month !== current && (
            <button className="link" onClick={() => changeMonth(current)}>
              Mois courant
            </button>
          )}
        </div>
      </header>
      {loading && (
        <p role="status" className="notice">
          Chargement des budgets de {monthLabel(month)}…
        </p>
      )}
      {error && (
        <div role="alert" className="notice err">
          Le rapport de {monthLabel(month)} est indisponible : {error}{' '}
          <button className="link" onClick={reload}>
            Réessayer
          </button>
        </div>
      )}
      {report && model && (
        <>
          <div className="budget-period">
            <span className="budget-period-badge">
              {model.current
                ? 'Mois courant · Provisoire'
                : model.complete
                  ? 'Mois historique · Données documentées'
                  : 'Mois historique · Couverture partielle ou inconnue'}
            </span>
            <span>
              Opérations interrogées : {budgetDate(report.query_from)} au{' '}
              {budgetDate(report.query_to)}.
            </span>
          </div>
          <p className="budget-method">
            Montants observés, sans extrapolation de fin de mois. Les débits
            sont comptés une seule fois, hors transferts internes ; les crédits
            ne sont pas assimilés automatiquement à des remboursements.
          </p>
          {!model.complete && (
            <p className="budget-caution">
              Le suivi dépend des opérations importées. Une période sans
              opération peut être mal documentée ; la date de dernière
              transaction ne prouve pas la couverture bancaire. Le salaire
              attendu en fin de mois ne déclenche aucune alerte ici.
            </p>
          )}
          <dl className="budget-summary" aria-label="Synthèse budgétaire">
            <BudgetMetric
              label="Budget total alloué"
              value={eur(model.allocated)}
              note="Enveloppes de dépenses uniquement"
            />
            <BudgetMetric
              label="Consommé observé"
              value={eur(model.spent)}
              note="Débits couverts par une enveloppe"
            />
            <BudgetMetric
              label="Montant restant"
              value={eur(model.remaining)}
              note={
                model.remaining < 0
                  ? 'Dépassement global des plafonds'
                  : 'Avant les plafonds alloués'
              }
            />
            <BudgetMetric
              label="Budgets dépassés"
              value={model.overCount}
              note="Plafond atteint ≠ dépassé"
            />
            <BudgetMetric
              label="Taux de consommation"
              value={budgetPercent(model.ratio)}
              note="Consommé / budget alloué"
            />
            <BudgetMetric
              label="Couverture budgétaire"
              value={budgetPercent(model.coverageRatio)}
              note="Part des débits observés couverte"
            />
          </dl>
          <details className="budget-provenance">
            <summary>Comprendre la période et les calculs</summary>
            <div>
              <p>
                Le restant correspond aux plafonds moins les débits observés. Ce
                n’est pas ton solde bancaire. Les plafonds sont mensuels et ne
                sont pas proratisés.
              </p>
              <p>
                Dépenses observées hors transferts :{' '}
                <strong>{eur(model.observedExpenses)}</strong>. Crédits observés
                présentés séparément : <strong>{eur(model.credits)}</strong>.
              </p>
              <p>
                Dernière transaction observée :{' '}
                {budgetDate(report.period?.last_tx_date)}. Couverture examinée
                jusqu’au {budgetDate(report.coverage?.through)} ; ces deux dates
                ont des significations différentes.
              </p>
              {report.coverage?.accounts?.map((account) => (
                <div className="budget-account-coverage" key={account.id}>
                  <strong>{account.name}</strong>
                  <span>
                    {account.status === 'documented'
                      ? 'Couverture documentée'
                      : 'Couverture à vérifier'}
                  </span>
                  {account.gaps?.length > 0 && (
                    <p>
                      Intervalles sans preuve documentaire :{' '}
                      {account.gaps
                        .map(
                          (gap) =>
                            `${budgetDate(gap.from)} → ${budgetDate(gap.to)}`,
                        )
                        .join(' ; ')}
                      .
                    </p>
                  )}
                  <p>Dernier import : {budgetDate(account.last_import)}.</p>
                </div>
              ))}
              <p>
                Un mois vide n’est un zéro établi que si sa couverture
                documentaire est connue. Les dépenses récurrentes ci-dessous
                restent distinctes des plafonds : elles ne sont jamais
                additionnées une deuxième fois à la consommation.
              </p>
            </div>
          </details>
          <section
            className="budget-envelopes"
            aria-labelledby="budget-envelopes-title"
          >
            <div className="budgets-section-heading">
              <div>
                <span className="eyebrow">01 · Gérer</span>
                <h3 id="budget-envelopes-title">Tes enveloppes de dépenses</h3>
              </div>
              <button
                className="btn"
                onClick={() => setDialog({ mode: 'create' })}
                disabled={!availableDomains.length || Boolean(domainError)}
              >
                Créer un budget
              </button>
            </div>
            {domainError && (
              <p role="alert" className="notice err">
                Catégories indisponibles : {domainError}{' '}
                <button className="link" onClick={reload}>
                  Réessayer
                </button>
              </p>
            )}
            {model.unclassifiedCount > 0 && (
              <div className="budget-unclassified">
                <p>
                  <strong>{eur(model.unclassified)}</strong> de débits sans
                  catégorie ({model.unclassifiedCount} opération(s)). Ces
                  dépenses ne sont pas couvertes par une enveloppe.
                </p>
                <button
                  className="link"
                  onClick={() =>
                    onNavigate?.('transactions', { month, domain: 'none' })
                  }
                >
                  Vérifier les opérations non catégorisées
                </button>
              </div>
            )}
            {model.expenses.length > 0 && (
              <div className="budget-filters">
                <label>
                  <span className="budget-sr-only">
                    Rechercher une enveloppe
                  </span>
                  <input
                    type="search"
                    placeholder="Rechercher une enveloppe…"
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                  />
                </label>
                <label>
                  <input
                    type="checkbox"
                    checked={onlyOver}
                    onChange={(e) => setOnlyOver(e.target.checked)}
                  />
                  Dépassées uniquement
                </label>
              </div>
            )}
            <div className="budget-envelope-list">
              {filtered.map(exportRow)}
            </div>
            {!model.expenses.length && (
              <div className="empty">
                <p>Aucune enveloppe de dépenses définie pour le moment.</p>
                <p>
                  Crée un plafond pour une catégorie ; le suivi utilisera les
                  opérations importées du mois sélectionné.
                </p>
              </div>
            )}
            {model.expenses.length > 0 && !filtered.length && (
              <p className="budget-empty-filter">
                Aucune enveloppe ne correspond aux filtres.
              </p>
            )}
          </section>
          {model.transfers.length > 0 && (
            <section
              className="budget-transfer-section"
              aria-labelledby="budget-transfer-title"
            >
              <div className="budgets-section-heading">
                <div>
                  <span className="eyebrow">Périmètre distinct</span>
                  <h3 id="budget-transfer-title">
                    Plafonds de transferts et d’épargne
                  </h3>
                </div>
              </div>
              <p className="budget-method">
                Ces mouvements ne constituent pas des dépenses de consommation.
                Leurs plafonds et leurs sorties sont exclus de la synthèse
                ci-dessus. Ils ne prouvent pas à eux seuls un objectif d’épargne
                atteint.
              </p>
              <div className="budget-envelope-list">
                {model.transfers.map(exportRow)}
              </div>
            </section>
          )}
        </>
      )}
      <section
        className="budget-recurrences"
        aria-labelledby="budget-recurrences-title"
      >
        <div className="budgets-section-heading">
          <div>
            <span className="eyebrow">02 · Comprendre</span>
            <h3 id="budget-recurrences-title">
              Dépenses et mouvements récurrents
            </h3>
          </div>
          <label className="budget-ignored-toggle">
            <input
              type="checkbox"
              checked={showIgnored}
              onChange={(e) => setShowIgnored(e.target.checked)}
            />
            Inclure les détections ignorées
          </label>
        </div>
        <p className="budget-method">
          Détections déterministes à examiner. Un montant stable n’est pas une
          preuve d’abonnement ; transferts, épargne et retraits ne sont pas
          assimilés à des abonnements.
        </p>
        {recLoading && (
          <p role="status" className="notice">
            Chargement des récurrences…
          </p>
        )}
        {recError && (
          <div role="alert" className="notice err">
            Les récurrences sont indisponibles : {recError}{' '}
            <button className="link" onClick={reload}>
              Réessayer
            </button>
          </div>
        )}
        {recReport && (
          <>
            <p className="budget-observation-window">
              Période d’observation : {budgetDate(recReport.observation_start)}{' '}
              au {budgetDate(recReport.observation_end)}. Aucune opération
              postérieure au mois sélectionné n’est utilisée.
            </p>
            {Object.entries(recurringNatures).map(([nature, label]) => {
              const items = recItems.filter((item) => item.nature === nature)
              return items.length > 0 ? (
                <section className="budget-recurrence-group" key={nature}>
                  <div className="budget-recurrence-group-heading">
                    <h4>{label}</h4>
                    <span>{items.length} détection(s)</span>
                  </div>
                  {items.map((item) => (
                    <Recurrence
                      key={item.key}
                      item={item}
                      onReview={() => setReview(item)}
                      onInspect={() => goRecurrence(item)}
                    />
                  ))}
                </section>
              ) : null
            })}
            {recItems.length === 0 && (
              <p className="empty">
                Aucune récurrence à afficher sur cette période. Cela ne prouve
                pas l’absence de charges futures.
              </p>
            )}
          </>
        )}
      </section>
      {dialog && (
        <BudgetDialog
          mode={dialog.mode}
          row={dialog.row}
          domains={dialog.row ? domains : availableDomains}
          onClose={() => setDialog(null)}
          onSave={save}
        />
      )}
      {review && (
        <RecurrenceDialog
          item={review}
          month={month}
          onClose={() => setReview(null)}
          onSaved={() => {
            toast.ok('Décision enregistrée.')
            reload()
          }}
        />
      )}
    </div>
  )
}
