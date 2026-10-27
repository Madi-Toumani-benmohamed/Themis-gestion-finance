import ImportReport from '../components/ImportReport'
import { AnimatedRegion } from '../motion'
import { useEffect, useRef, useState } from 'react'
import { useModal } from '../components/useModal'
import { api } from '../api'
import { useToast } from '../Toast'
import { DataReliability } from '../components/DataReliability'
import { reliabilityDate } from '../components/dataReliabilityModel.mjs'
import { euroCents } from '../components/money.mjs'
import {
  controlStatus,
  filterStatements,
  statementListModel,
  statementStatus,
  statementStatusIcon,
  statementSummary,
} from '../components/statementPageModel.mjs'
import './Statements.css'

const amount = (value) => (value == null ? 'Non disponible' : euroCents(value))
function StatementDialog({
  statement: st,
  mode,
  close,
  onDelete,
  onReview,
  onNavigate,
  onRecheck,
  aiEnabled = false,
}) {
  const ref = useRef(null)
  const [busy, setBusy] = useState(false),
    [error, setError] = useState(''),
    [note, setNote] = useState(''),
    [checkMode, setCheckMode] = useState('deterministic'),
    [checkResult, setCheckResult] = useState(null)
  const dismiss = useModal(ref, '.statements-table-header h2')
  const requestClose = () => { if (!busy) dismiss(close) }
  const act = async (callback) => {
    setBusy(true)
    setError('')
    try {
      await callback()
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <dialog
      ref={ref}
      className="statement-dialog"
      aria-labelledby="statement-dialog-title"
      onCancel={(e) => {
        e.preventDefault()
        requestClose()
      }}
    >
      <header>
        <h2 id="statement-dialog-title">
          {mode === 'delete'
            ? 'Supprimer ce relevé ?'
            : 'Actions et contrôles du relevé'}
        </h2>
        <button
          className="btn-secondary"
          onClick={requestClose}
          disabled={busy}
          aria-label="Fermer le détail"
        >
          Fermer
        </button>
      </header>
      <p className="statement-filename">{st.filename}</p>
      {error && (
        <p className="notice err" role="alert">
          {error}
        </p>
      )}
      {busy && <p role="status">Action en cours…</p>}
      {mode === 'delete' ? (
        <>
          <p>
            La suppression retirera ce relevé et ses{' '}
            <strong>{st.tx_count} opérations</strong>, ainsi que les tags et
            propositions associés à ces opérations. Les indicateurs seront
            recalculés et les analyses enregistrées devenues obsolètes seront
            retirées. Les autres relevés et les pointages de compte seront
            conservés.
          </p>
          <p>
            Les métadonnées et la justification de suppression resteront dans
            l’historique exportable. Les écritures originales supprimées ne
            seront plus consultables dans Transactions. Cette suppression est
            irréversible.
          </p>
          <div className="statement-actions">
            <button className="btn-secondary" onClick={requestClose} disabled={busy}>
              Annuler la suppression
            </button>
            <button
              className="btn-danger"
              onClick={() => act(onDelete)}
              disabled={busy}
            >
              {busy
                ? 'Suppression…'
                : `Confirmer la suppression de ${st.tx_count} opérations`}
            </button>
          </div>
        </>
      ) : (
        <>
          <section
            className="statement-action-panel"
            aria-labelledby="statement-actions-title"
          >
            <h3 id="statement-actions-title">Actions sur ce relevé</h3>
            <span className={`statement-status ${st.status}`}>
              <span aria-hidden="true">{statementStatusIcon(st.status)}</span>
              {statementStatus(st.status)}
            </span>
            <div className="statement-check-launch">
              <label>
                Mode de vérification
                <select
                  aria-label="Mode de vérification"
                  value={checkMode}
                  onChange={(e) => setCheckMode(e.target.value)}
                  disabled={busy}
                >
                  <option value="deterministic">
                    Contrôles comptables · sans IA
                  </option>
                  <option value="combined" disabled={!aiEnabled}>
                    Contrôles comptables + relecture IA
                  </option>
                </select>
              </label>
              <button
                className="btn-secondary"
                disabled={busy}
                onClick={() =>
                  act(async () => setCheckResult(await onRecheck(checkMode)))
                }
              >
                Relancer les vérifications
              </button>
            </div>
            <p className="statement-help">
              Les contrôles sont recalculés sur les données enregistrées, sans
              changer les transactions ni la validation. Le PDF original n’étant
              pas conservé, il ne s’agit pas d’une nouvelle extraction.
            </p>
            {checkMode === 'combined' && (
              <p className="statement-help">
                La relecture envoie uniquement les constats des contrôles au
                fournisseur IA configuré, sans PDF, montants ou libellés
                bancaires. Elle peut consommer des jetons API et ne constitue
                pas une preuve supplémentaire de complétude.
              </p>
            )}
            {!aiEnabled && (
              <p className="statement-help">
                IA non configurée : les contrôles sans IA restent disponibles.
              </p>
            )}
            {checkResult && (
              <div className="statement-check-result" role="status">
                <strong>
                  Contrôles recalculés le{' '}
                  {new Date(checkResult.checked_at).toLocaleString('fr-FR')}
                </strong>
                <p>{checkResult.ai.detail}</p>
                {checkResult.ai.model && (
                  <p className="statement-help">
                    Modèle : {checkResult.ai.model} · avis consultatif
                  </p>
                )}
                {checkResult.ai.evidence?.length > 0 && (
                  <ul>
                    {checkResult.ai.evidence.map((f) => (
                      <li key={f.id}>{f.text}</li>
                    ))}
                  </ul>
                )}
                <p className="statement-help">
                  Résultat de cette session, lié à l’état des données à l’heure
                  du contrôle. La validation manuelle reste une action séparée.
                </p>
              </div>
            )}
            <h3>Validation manuelle</h3>
            <label>
              Justification de la revue
              <textarea
                aria-label="Justification de la revue"
                rows={2}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                disabled={busy}
                placeholder="Contrôles effectués sur le document original…"
              />
            </label>
            <div className="statement-validation-actions">
              <button
                className="btn"
                disabled={
                  busy ||
                  !st.can_validate ||
                  (st.validated && st.validation_source === 'manual') ||
                  Array.from(note.trim()).length < 10 ||
                  Array.from(note.trim()).length > 1000
                }
                onClick={() => act(() => onReview('validate', note.trim()))}
              >
                Valider le relevé
              </button>
              {st.validated && (
                <button
                  className="btn-secondary"
                  disabled={
                    busy ||
                    Array.from(note.trim()).length < 10 ||
                    Array.from(note.trim()).length > 1000
                  }
                  onClick={() => act(() => onReview('reopen', note.trim()))}
                >
                  Rouvrir la vérification
                </button>
              )}
            </div>
            <p className="statement-help">
              Justification de 10 à 1 000 caractères. La validation atteste ta
              revue de l’original ; un écart comptable ou une rupture de solde
              confirmée ne peut pas être contourné, même avec l’IA.
            </p>
            {!st.can_validate && (
              <p className="statement-validation-blocked">
                Validation indisponible : consulter les contrôles en échec ou
                non établis ci-dessous.
              </p>
            )}
            {st.validated && st.validation_source === 'manual' && (
              <p className="statement-help">
                Validation manuelle déjà enregistrée. Tu peux relancer les
                contrôles ou rouvrir la vérification.
              </p>
            )}
          </section>
          <div className="statement-detail-grid">
            <dl>
              <div>
                <dt>Compte</dt>
                <dd>{st.account_name}</dd>
              </div>
              <div>
                <dt>Période bancaire inclusive</dt>
                <dd>
                  {reliabilityDate(st.period_start)} →{' '}
                  {reliabilityDate(st.period_end)}
                </dd>
              </div>
              <div>
                <dt>Import</dt>
                <dd>
                  {reliabilityDate(st.uploaded_at)} ·{' '}
                  {st.parser || 'Parseur non renseigné'}
                </dd>
              </div>
              <div>
                <dt>Statut</dt>
                <dd>
                  <span className={`statement-status ${st.status}`}>
                    <span aria-hidden="true">
                      {statementStatusIcon(st.status)}
                    </span>
                    {statementStatus(st.status)}
                  </span>
                </dd>
              </div>
              <div>
                <dt>Validation enregistrée</dt>
                <dd>
                  {st.validation_source === 'manual'
                    ? 'Confirmation manuelle tracée'
                    : st.validation_source === 'automatic_legacy'
                      ? 'Validation historique · rapport d’extraction indisponible'
                      : st.validated
                      ? 'Contrôles disponibles du parsing conformes'
                      : 'Non validée / rouverte'}
                </dd>
              </div>
            </dl>
            <dl>
              <div>
                <dt>Solde initial</dt>
                <dd>{amount(st.opening_balance_cents)}</dd>
              </div>
              <div>
                <dt>Crédits observés</dt>
                <dd>{amount(st.credits_cents)}</dd>
              </div>
              <div>
                <dt>Débits observés</dt>
                <dd>{amount(st.debits_cents)}</dd>
              </div>
              <div>
                <dt>Solde théorique · initial + opérations</dt>
                <dd>{amount(st.theoretical_balance_cents)}</dd>
              </div>
              <div>
                <dt>Solde final déclaré</dt>
                <dd>{amount(st.closing_balance_cents)}</dd>
              </div>
              <div>
                <dt>Écart · final − théorique</dt>
                <dd>{amount(st.difference_cents)}</dd>
              </div>
            </dl>
          </div>
          <ImportReport report={st.import_report} />
          <h3>Résultats des contrôles</h3>
          <ul className="statement-controls">
            {st.controls.map((c, i) => (
              <li key={`${c.code}-${i}`}>
                <strong className={`statement-control-${c.status}`}>
                  {controlStatus(c.status)}
                </strong>
                <p>{c.detail}</p>
                {c.related_id && <small>Relevé lié n° {c.related_id}</small>}
              </li>
            ))}
          </ul>
          <p className="statement-help">
            Un contrôle arithmétique réussi ne garantit pas l’exhaustivité du
            document. Les transactions provisoires restent utilisables ; la
            couverture et les anomalies précisent leurs limites. Aucune date de
            valeur n’est inventée.
          </p>
          <button
            className="btn-secondary"
            onClick={() => {
              close()
              onNavigate?.('transactions', { statement_id: st.id })
            }}
          >
            Consulter les {st.tx_count} opérations
          </button>
          <h3>Historique de validation</h3>
          {st.history.length ? (
            <ul className="statement-history">
              {st.history.map((h) => (
                <li key={h.id}>
                  <strong>
                    {h.action === 'validate'
                      ? 'Validation manuelle'
                      : 'Vérification rouverte'}
                  </strong>{' '}
                  · {reliabilityDate(h.at)}
                  <p>{h.note}</p>
                </li>
              ))}
            </ul>
          ) : (
            <p>
              Aucune revue manuelle enregistrée. Le résultat historique du
              parsing n’est pas une revue manuelle.
            </p>
          )}
          <p className="statement-help">
            Les 20 dernières revues sont affichées ; l’export personnel conserve
            l’historique complet, y compris après suppression.
          </p>
        </>
      )}
    </dialog>
  )
}

export default function Statements({
  onChanged,
  initialMonth,
  onMonthChange,
  onGoUpload,
  onNavigate,
  aiEnabled = false,
}) {
  const toast = useToast(),
    tableTitle = useRef(null)
  const [month, setMonth] = useState(
      initialMonth || new Date().toISOString().slice(0, 7),
    ),
    [attempt, setAttempt] = useState(0)
  const [stmts, setStmts] = useState(null),
    [error, setError] = useState(''),
    [dialog, setDialog] = useState(null)
  const [filters, setFilters] = useState({
    account: '',
    status: '',
    search: '',
    month: '',
    ids: null,
    sort: 'period',
  })
  useEffect(() => {
    const abort = new AbortController()
    setError('')
    setStmts(null)
    api
      .statements({ signal: abort.signal })
      .then((rows) => {
        if (!abort.signal.aborted) setStmts(statementListModel(rows))
      })
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message)
      })
    return () => abort.abort()
  }, [attempt])
  const chooseMonth = (value) => {
    setMonth(value)
    setFilters((f) => ({ ...f, month: f.month ? value : '', ids: null }))
    onMonthChange?.(value)
  }
  const inspect = (ids) => {
    setFilters({
      account: '',
      status: '',
      search: '',
      month: '',
      ids: new Set(ids),
      sort: 'period',
    })
    tableTitle.current?.focus()
  }
  const reset = () =>
    setFilters({
      account: '',
      status: '',
      search: '',
      month: '',
      ids: null,
      sort: 'period',
    })
  const selected = stmts?.find((s) => s.id === dialog?.id),
    summary = stmts && statementSummary(stmts)
  const visible = stmts ? filterStatements(stmts, filters) : []
  const refresh = () => {
    setDialog(null)
    setAttempt((n) => n + 1)
  }
  const accounts = [
    ...new Map(
      (stmts || []).map((s) => [String(s.account_id), s.account_name]),
    ).entries(),
  ]
  return (
    <div className="section statements-page">
      <header className="statements-heading">
        <div>
          <h1>Contrôle des relevés</h1>
          <p>
            Documents bancaires, rapprochements et limites des données
            transactionnelles.
          </p>
        </div>
        {onGoUpload && (
          <button className="btn" onClick={onGoUpload}>
            Importer un relevé
          </button>
        )}
      </header>
      {error ? (
        <div className="notice err" role="alert">
          <p>{error}</p>
          <button
            className="btn-secondary"
            onClick={() => setAttempt((n) => n + 1)}
          >
            Réessayer les relevés
          </button>
        </div>
      ) : !stmts ? (
        <p role="status">Chargement des relevés et contrôles…</p>
      ) : (
        <>
          <dl className="statement-summary">
            <div>
              <dt>Relevés importés · tout l’historique</dt>
              <dd>{summary.total}</dd>
            </div>
            <div>
              <dt>Relevés à vérifier ou valider</dt>
              <dd>{summary.review}</dd>
            </div>
            <div>
              <dt>Dernière fin de relevé validé</dt>
              <dd>
                {reliabilityDate(summary.lastValidated)}
                <small>
                  Repère documentaire, sans garantie de couverture globale.
                </small>
              </dd>
            </div>
          </dl>
          <DataReliability
            key={attempt}
            month={month}
            onMonthChange={chooseMonth}
            onInspect={inspect}
          />
          <div className="statements-table-header">
            <h2 ref={tableTitle} tabIndex={-1}>
              {filters.ids ? 'Relevés concernés' : 'Historique des relevés'}
            </h2>
            <span>
              {visible.length} / {stmts.length} relevé(s)
            </span>
          </div>
          <div className="statement-filters">
            <label>
              Recherche
              <input
                type="search"
                value={filters.search}
                onChange={(e) =>
                  setFilters({ ...filters, search: e.target.value })
                }
                placeholder="Fichier ou compte"
              />
            </label>
            <label>
              Compte
              <select
                aria-label="Compte"
                value={filters.account}
                onChange={(e) =>
                  setFilters({ ...filters, account: e.target.value })
                }
              >
                <option value="">Tous les comptes</option>
                {accounts.map(([id, name]) => (
                  <option key={id} value={id}>
                    {name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Statut
              <select
                aria-label="Statut"
                value={filters.status}
                onChange={(e) =>
                  setFilters({ ...filters, status: e.target.value })
                }
              >
                <option value="">Tous les statuts</option>
                {['validated', 'to_review', 'failed', 'analyzed'].map((s) => (
                  <option key={s} value={s}>
                    {statementStatus(s)}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Périmètre
              <select
                aria-label="Périmètre"
                value={filters.month ? 'month' : 'all'}
                onChange={(e) =>
                  setFilters({
                    ...filters,
                    month: e.target.value === 'month' ? month : '',
                  })
                }
              >
                <option value="all">Tout l’historique</option>
                <option value="month">
                  Mois examiné · périodes inconnues incluses
                </option>
              </select>
            </label>
            <label>
              Trier par
              <select
                aria-label="Trier par"
                value={filters.sort}
                onChange={(e) =>
                  setFilters({ ...filters, sort: e.target.value })
                }
              >
                <option value="period">Période récente</option>
                <option value="import">Import récent</option>
              </select>
            </label>
            <button className="btn-secondary" onClick={reset}>
              Réinitialiser les filtres
            </button>
          </div>
          <p className="statement-help">
            Périodes bancaires inclusives, distinctes des mois calendaires. Un
            relevé validé peut ne couvrir qu’une partie du mois ; les jours
            futurs ne sont jamais signalés manquants.
          </p>
          {!visible.length ? (
            <p className="empty">
              {stmts.length
                ? 'Aucun relevé pour ces filtres.'
                : 'Aucun relevé importé. Les comptes suivis par pointage restent gérés dans Comptes.'}
            </p>
          ) : (
            <AnimatedRegion revision={filters}
              className="statements-table-scroll"
              tabIndex={0}
              role="region"
              aria-label="Historique des relevés, tableau défilant"
            >
              <table className="statements-table">
                <thead>
                  <tr>
                    <th>Document / compte</th>
                    <th>Période bancaire</th>
                    <th className="num">Solde initial</th>
                    <th className="num">Solde final</th>
                    <th className="num">Opérations</th>
                    <th>État des contrôles</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {visible.map((st) => (
                    <tr key={st.id}>
                      <td>
                        <button
                          className="statement-open"
                          onClick={() =>
                            setDialog({ id: st.id, mode: 'detail' })
                          }
                          title={st.filename}
                        >
                          {st.filename}
                        </button>
                        <small>
                          {st.account_name} ·{' '}
                          {st.tracking_mode === 'manual'
                            ? 'pointage manuel'
                            : 'suivi documentaire'}
                        </small>
                      </td>
                      <td>
                        {st.period_start && st.period_end ? (
                          <>
                            <time>{st.period_start}</time>
                            <span> → {st.period_end}</span>
                          </>
                        ) : (
                          'Non établie'
                        )}
                      </td>
                      <td className="num amt">
                        {amount(st.opening_balance_cents)}
                      </td>
                      <td className="num amt">
                        {amount(st.closing_balance_cents)}
                      </td>
                      <td className="num amt">{st.tx_count}</td>
                      <td>
                        <span className={`statement-status ${st.status}`}>
                          <span aria-hidden="true">
                            {statementStatusIcon(st.status)}
                          </span>
                          {statementStatus(st.status)}
                        </span>
                        <small>
                          {
                            st.controls.filter((c) => !['passed', 'informational', 'not_applicable'].includes(c.status))
                              .length
                          }{' '}
                          contrôle(s) non confirmé(s)
                        </small>
                      </td>
                      <td>
                        <div className="statement-row-actions">
                          <button
                            className="btn-secondary"
                            onClick={() =>
                              setDialog({ id: st.id, mode: 'detail' })
                            }
                            aria-label={`Actions pour ${st.filename}`}
                          >
                            Actions
                          </button>
                          <button
                            className="link"
                            onClick={() =>
                              setDialog({ id: st.id, mode: 'delete' })
                            }
                            aria-label={`Supprimer ${st.filename}`}
                          >
                            Supprimer
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </AnimatedRegion>
          )}
        </>
      )}
      {selected && (
        <StatementDialog
          key={`${selected.id}-${dialog.mode}`}
          statement={selected}
          mode={dialog.mode}
          close={() => setDialog(null)}
          onNavigate={onNavigate}
          aiEnabled={aiEnabled}
          onRecheck={async (mode) => {
            const result = await api.recheckStatement(selected.id, mode)
            const [checked] = statementListModel([result.statement])
            setStmts((rows) =>
              rows.map((st) => (st.id === checked.id ? checked : st)),
            )
            return result
          }}
          onDelete={async () => {
            await api.deleteStatement(selected.id)
            toast.ok('Relevé et opérations supprimés ; trace conservée.')
            refresh()
            onChanged?.()
          }}
          onReview={async (action, note) => {
            await api.validateStatement(selected.id, action, note)
            toast.ok('Revue enregistrée et tracée.')
            refresh()
          }}
        />
      )}
    </div>
  )
}
