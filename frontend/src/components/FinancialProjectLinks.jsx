import { useState } from 'react'
import { api } from '../api'
import { useAnalysisResource } from './analysisResources'
import { ResourceState, Pagination } from './FinancialProjectUI'
import { projectMoney as money, projectReason } from './projectModel.mjs'
export function ProjectHistory({ id, revision }) {
  const [offset, setOffset] = useState(0)
  const resource = useAnalysisResource(
    (signal) => api.projectHistory(id, offset, { signal }),
    [id, offset, revision],
  )
  return (
    <section>
      <h4>Historique des décisions</h4>
      <ResourceState resource={resource} label="de l’historique" />
      <ul>
        {resource.data?.history.map((h) => (
          <li key={h.id}>
            {h.changed_at} ·{' '}
            {{
              decision: 'Rattachement modifié',
              rule_saved: 'Règle enregistrée',
              project_saved: 'Projet enregistré',
            }[h.event] || 'Modification enregistrée'}
            {h.transaction_id ? ` · opération nº${h.transaction_id}` : ''} ·{' '}
            {{ confirmed: 'Confirmée', rejected: 'Rejetée ou retirée' }[
              typeof h.after === 'string' ? h.after : h.after?.state
            ] || 'Modification enregistrée'}
          </li>
        ))}
      </ul>
      <Pagination
        offset={offset}
        step={50}
        more={resource.data?.has_more}
        setOffset={setOffset}
      />
    </section>
  )
}
export function ProjectLinks({
  id,
  month,
  state,
  setState,
  category,
  clearCategory,
  run,
  revision,
  archived,
  onNavigate,
}) {
  const [offset, setOffset] = useState(0),
    [checked, setChecked] = useState([])
  const [search, setSearch] = useState(''),
    [submitted, setSubmitted] = useState('')
  const resource = useAnalysisResource(
    (signal) =>
      api.projectTransactions(
        id,
        { month, state, offset, category },
        { signal },
      ),
    [id, month, state, offset, category, revision],
  )
  const candidates = useAnalysisResource(
    (signal) =>
      submitted
        ? api.transactionReport(
            { q: submitted, type: 'debit', month, limit: 20 },
            { signal },
          )
        : Promise.resolve(null),
    [submitted, month, revision],
  )
  const rows = resource.data?.transactions || []
  const decide = (ids, next) =>
    run(async () => {
      await api.projectDecide(id, ids, next)
      setChecked([])
    })
  return (
    <section>
      <h4>Rattachements</h4>
      <label>
        Afficher les opérations
        <select
          value={state}
          onChange={(e) => {
            setState(e.target.value)
            setOffset(0)
            setChecked([])
          }}
        >
          <option value="confirmed">Confirmées</option>
          <option value="suggested">Suggestions à valider</option>
          <option value="rejected">Rejetées ou retirées</option>
        </select>
      </label>
      {category && (
        <p>
          Poste : {category}{' '}
          <button className="link" onClick={clearCategory}>
            Tous les postes
          </button>
        </p>
      )}
      <ResourceState resource={resource} label="des opérations" />
      {!resource.loading && !resource.error && rows.length === 0 && (
        <p>Aucune opération dans cette vue à la date consultée.</p>
      )}
      {state === 'suggested' && !archived && rows.length > 0 && (
        <div className="project-actions">
          <button
            className="btn-primary"
            disabled={!checked.length}
            onClick={() => decide(checked, 'confirmed')}
          >
            Confirmer la sélection ({checked.length})
          </button>
          <button
            className="btn-secondary"
            disabled={!checked.length}
            onClick={() => decide(checked, 'rejected')}
          >
            Rejeter la sélection
          </button>
        </div>
      )}
      {rows.length > 0 && (
        <div
          className="project-table"
          tabIndex={0}
          role="region"
          aria-label="Tableau des rattachements"
        >
          <table>
            <caption>
              Opérations du projet —{' '}
              {state === 'suggested'
                ? 'suggestions exclues du coût'
                : state === 'confirmed'
                  ? 'confirmations et éligibilité au coût'
                  : 'décisions conservées'}
            </caption>
            <thead>
              <tr>
                {state === 'suggested' && !archived && <th>Sélection</th>}
                <th>Date et opération</th>
                <th>Montant bancaire</th>
                <th>Justification</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((t) => (
                <tr key={t.id}>
                  {state === 'suggested' && !archived && (
                    <td>
                      <input
                        aria-label={`Sélectionner ${t.label}`}
                        type="checkbox"
                        disabled={!t.eligible}
                        checked={checked.includes(t.id)}
                        onChange={(e) =>
                          setChecked((ids) =>
                            e.target.checked
                              ? [...ids, t.id]
                              : ids.filter((id) => id !== t.id),
                          )
                        }
                      />
                    </td>
                  )}
                  <td>
                    {t.date}
                    <br />
                    {t.label}
                    <br />
                    {t.category}
                  </td>
                  <td>
                    {money(t.amount_cents)}
                    {!t.eligible && (
                      <p>Exclue : hors dates ou flux non éligible.</p>
                    )}
                  </td>
                  <td>{projectReason(t.reason)}</td>
                  <td>
                    <div className="project-actions">
                      <button
                        className="link"
                        onClick={() =>
                          onNavigate?.('transactions', { ids: [t.id] })
                        }
                      >
                        Voir l’opération nº{t.id}
                      </button>
                      {!archived &&
                        (state !== 'confirmed' ? (
                          <button
                            className="btn-secondary"
                            disabled={!t.eligible}
                            onClick={() => decide([t.id], 'confirmed')}
                          >
                            Confirmer
                          </button>
                        ) : (
                          <button
                            className="btn-secondary"
                            onClick={() => decide([t.id], 'rejected')}
                          >
                            Retirer
                          </button>
                        ))}
                      {state === 'suggested' && !archived && (
                        <button
                          className="btn-secondary"
                          onClick={() => decide([t.id], 'rejected')}
                        >
                          Rejeter
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <Pagination
        offset={offset}
        step={50}
        more={resource.data?.has_more}
        setOffset={(value) => {
          setOffset(value)
          setChecked([])
        }}
      />
      {!archived && (
        <details>
          <summary>Associer manuellement une dépense</summary>
          <p>
            Recherche sur le mois {month}. Seuls les débits ordinaires peuvent
            être confirmés ; les dates du projet restent vérifiées par le
            serveur.
          </p>
          <form
            className="project-actions"
            onSubmit={(e) => {
              e.preventDefault()
              setSubmitted(search.trim())
            }}
          >
            <label>
              Marchand ou libellé
              <input
                required
                minLength={3}
                maxLength={200}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </label>
            <button className="btn-secondary">Rechercher</button>
          </form>
          <ResourceState resource={candidates} label="des résultats" />
          {submitted && candidates.data?.transactions?.length === 0 && (
            <p>Aucune dépense correspondante.</p>
          )}
          <ul>
            {candidates.data?.transactions?.map((t) => (
              <li key={t.id}>
                {t.date || t.tx_date} ·{' '}
                {t.merchant || t.display_label || t.label} ·{' '}
                {money(t.amount_cents)}{' '}
                <button
                  className="btn-secondary"
                  onClick={() => decide([t.id], 'confirmed')}
                >
                  Associer l’opération nº{t.id}
                </button>
              </li>
            ))}
          </ul>
          {candidates.data?.total > candidates.data?.transactions.length && (
            <p>Affiner la recherche pour voir les autres résultats.</p>
          )}
        </details>
      )}
    </section>
  )
}
