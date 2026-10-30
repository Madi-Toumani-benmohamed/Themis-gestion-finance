import { useState } from 'react'
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
} from 'recharts'
import { api } from '../api'
import { useAnalysisResource } from './analysisResources'
import {
  projectMoney as money,
  projectStatus,
  projectCumulative,
  projectListModel,
  projectReportModel,
} from './projectModel.mjs'
import './financialProjects.css'

import { ResourceState, Pagination } from './FinancialProjectUI'
import { ProjectForm, ProjectRules } from './FinancialProjectForms'
import { ProjectLinks, ProjectHistory } from './FinancialProjectLinks'
function CostBreakdown({ title, rows, overlap = false, onSelect }) {
  const max = Math.max(...rows.map((r) => r.amount_cents), 1)
  if (!rows.length) return null
  return (
    <section>
      <h4>{title}</h4>
      {overlap && (
        <p>
          Une opération peut porter plusieurs tags : ces montants se chevauchent
          et ne s’additionnent pas.
        </p>
      )}
      <ul className="project-breakdown">
        {rows.map((row) => (
          <li key={row.name}>
            <div>
              {onSelect ? (
                <button className="link" onClick={() => onSelect(row.name)}>
                  {row.name}
                </button>
              ) : (
                row.name
              )}
              <strong>{money(row.amount_cents)}</strong>
            </div>
            <span className="project-bar" aria-hidden="true">
              <span style={{ width: `${(row.amount_cents / max) * 100}%` }} />
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
function ProjectDetail({ id, month, onChanged, onEdit, onNavigate, revision }) {
  const [busy, setBusy] = useState(false),
    [error, setError] = useState('')
  const [state, setState] = useState('confirmed'),
    [category, setCategory] = useState(''),
    [history, setHistory] = useState(false),
    [rulesOpen, setRulesOpen] = useState(false)
  const resource = useAnalysisResource(
    (signal) => api.project(id, month, { signal }).then(projectReportModel),
    [id, month, revision],
  )
  const run = async (action) => {
    setBusy(true)
    setError('')
    try {
      await action()
      onChanged()
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }
  const d = resource.data,
    p = d?.project
  const source = (name) => {
    setCategory(name)
    setState('confirmed')
  }
  return (
    <article
      className="project-detail"
      aria-label="Détail du projet"
      aria-busy={busy}
    >
      <ResourceState resource={resource} label="du projet" />
      {error && <p role="alert">{error}</p>}
      {p && (
        <>
          <div className="project-actions">
            <h3>
              {p.name} · {projectStatus[p.status]}
            </h3>
            <button className="btn-secondary" onClick={() => onEdit(p)}>
              Modifier le projet
            </button>
          </div>
          <p>{p.description}</p>
          <p>
            Du {d.query_from || 'début de l’historique associé'} au {d.query_to}
            . Dernière dépense observée : {d.last_observed || 'aucune'}.
          </p>
          <p className="project-coverage">
            {d.coverage === 'documented'
              ? 'Période documentée'
              : 'Coût provisoire'}{' '}
            — {d.coverage_reason}
          </p>
          <dl className="project-metrics">
            <div>
              <dt>Budget prévisionnel</dt>
              <dd>
                {p.budget_cents === null ? 'Non défini' : money(p.budget_cents)}
              </dd>
            </div>
            <div>
              <dt>Coût brut confirmé</dt>
              <dd>{money(p.cost_cents)}</dd>
            </div>
            <div>
              <dt>Budget restant</dt>
              <dd>
                {p.budget_cents === null
                  ? 'Sans budget défini'
                  : money(p.remaining_cents)}
              </dd>
            </div>
            <div>
              <dt>Dépassement</dt>
              <dd>
                {p.budget_cents === null
                  ? 'Sans budget défini'
                  : money(p.overrun_cents)}
              </dd>
            </div>
            <div>
              <dt>Dépenses distinctes</dt>
              <dd>{p.transaction_count}</dd>
            </div>
          </dl>
          {p.consumed_pct !== null && (
            <label>
              Budget consommé : {p.consumed_pct} %
              <progress max="100" value={Math.min(p.consumed_pct, 100)} />
            </label>
          )}
          {p.transaction_count === 0 && (
            <p>
              Aucune dépense confirmée dans la période. Les suggestions
              éventuelles ne sont pas incluses dans le coût.
            </p>
          )}
          {d.excluded_confirmed_count > 0 && (
            <p>
              {d.excluded_confirmed_count} confirmation(s) hors dates ou flux
              éligibles : exclues du coût, conservées dans les rattachements.
            </p>
          )}
          <section>
            <h4>Observations vérifiables</h4>
            {d.observations.length === 0 && (
              <p>Pas de constat suffisamment étayé pour ce projet.</p>
            )}
            {d.observations.map((o) => (
              <div className="project-observation" key={o.id}>
                <p>{o.text}</p>
                <p>
                  {o.type === 'budget_overrun'
                    ? `Dépassement : ${money(o.values.overrun_cents)} sur un coût de ${money(o.values.cost_cents)}.`
                    : `${o.values.category} : ${money(o.values.amount_cents)} / ${money(o.values.total_cents)} (${o.values.share_pct} %).`}
                </p>
                <p>{o.validity}</p>
                <button
                  className="link"
                  onClick={() => source(o.source.category || '')}
                >
                  Examiner les sources de ce constat
                </button>
              </div>
            ))}
            <p>
              Ces constats fonctionnent sans IA. Pour une explication
              facultative, ouvre l’assistant et demande l’analyse du projet nº
              {id} à {month}.
            </p>
          </section>
          <div className="project-cost-grid">
            <CostBreakdown
              title="Coût par catégorie"
              rows={d.by_category}
              onSelect={source}
            />
            <CostBreakdown title="Coût par marchand" rows={d.by_merchant} />
          </div>
          {d.by_month.length > 0 && (
            <section>
              <h4>Coût cumulé enregistré</h4>
              <p>
                Les mois affichés contiennent des dépenses confirmées. La courbe
                ne prouve pas la présence de tous les relevés.
              </p>
              <div className="project-chart" aria-hidden="true">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={projectCumulative(d.by_month)}>
                    <XAxis dataKey="name" />
                    <YAxis tickFormatter={(v) => money(v)} width={90} />
                    <Tooltip formatter={(v) => money(v)} />
                    <Line
                      type="stepAfter"
                      dataKey="cumulative_cents"
                      name="Coût cumulé"
                      stroke="var(--accent)"
                      isAnimationActive={false}
                    />
                  </LineChart>
                </ResponsiveContainer>
              </div>
              <div
                className="project-table"
                tabIndex={0}
                role="region"
                aria-label="Tableau du cumul"
              >
                <table>
                  <caption>Montants exacts par mois observé</caption>
                  <thead>
                    <tr>
                      <th>Mois</th>
                      <th>Dépenses</th>
                      <th>Cumul</th>
                    </tr>
                  </thead>
                  <tbody>
                    {projectCumulative(d.by_month).map((row) => (
                      <tr key={row.name}>
                        <th scope="row">{row.name}</th>
                        <td>{money(row.amount_cents)}</td>
                        <td>{money(row.cumulative_cents)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          )}
          <details>
            <summary>Répartition par tag (non additive)</summary>
            <CostBreakdown
              title="Tags des opérations"
              rows={d.by_tag}
              overlap
            />
          </details>
          <fieldset disabled={busy}>
            <legend>Gérer les rattachements du projet</legend>
            {p.status === 'active' && (
              <button
                className="btn-secondary"
                onClick={() => run(() => api.projectSuggest(id))}
              >
                Rechercher les suggestions
              </button>
            )}
            <ProjectLinks
              key={`${id}:${month}:${category}`}
              id={id}
              month={month}
              state={state}
              setState={setState}
              category={category}
              clearCategory={() => setCategory('')}
              run={run}
              revision={revision}
              archived={p.status === 'archived'}
              onNavigate={onNavigate}
            />
            <details
              open={rulesOpen}
              onToggle={(e) => setRulesOpen(e.currentTarget.open)}
            >
              <summary>Configurer les règles</summary>
              {rulesOpen && (
                <ProjectRules
                  id={id}
                  project={p}
                  revision={revision}
                  run={run}
                />
              )}
            </details>
          </fieldset>
          <button className="link" onClick={() => setHistory((v) => !v)}>
            {history
              ? 'Masquer l’historique'
              : 'Voir l’historique des décisions'}
          </button>
          {history && <ProjectHistory id={id} revision={revision} />}
        </>
      )}
    </article>
  )
}
export default function FinancialProjects({ month, onNavigate }) {
  const [revision, setRevision] = useState(0),
    [offset, setOffset] = useState(0),
    [selected, setSelected] = useState(null),
    [form, setForm] = useState(null)
  const resource = useAnalysisResource(
    (signal) => api.projects(month, offset, { signal }).then(projectListModel),
    [month, offset, revision],
  )
  const changed = () => setRevision((v) => v + 1)
  return (
    <section className="financial-projects" aria-label="Projets financiers">
      <div className="section-head">
        <h2>Projets financiers</h2>
        <button className="btn-primary" onClick={() => setForm({})}>
          Créer un projet
        </button>
      </div>
      <p>
        Coûts confirmés jusqu’au {resource.data?.query_to || month}. Une même
        opération peut appartenir à plusieurs projets : leurs coûts ne
        s’additionnent pas. Les coûts restent provisoires tant que les données
        ne sont pas documentées.
      </p>
      {form && (
        <ProjectForm
          key={form.id || 'new'}
          project={form.id ? form : null}
          onCancel={() => setForm(null)}
          onSaved={(id) => {
            setForm(null)
            setSelected(id)
            changed()
          }}
        />
      )}
      <ResourceState resource={resource} label="des projets" />
      {resource.data?.projects.length === 0 && (
        <p>
          Aucun projet créé. Crée un projet et ajoute un tag ou un critère pour
          retrouver ses dépenses.
        </p>
      )}
      <div className="project-cards">
        {resource.data?.projects.map((p) => (
          <button
            className="project-card"
            key={p.id}
            aria-expanded={selected === p.id}
            onClick={() => {
              setSelected((v) => (v === p.id ? null : p.id))
              setForm(null)
            }}
          >
            <strong>{p.name}</strong>
            <span>{projectStatus[p.status]}</span>
            <span>Coût observé : {money(p.cost_cents)}</span>
            <span>
              Budget :{' '}
              {p.budget_cents === null ? 'non défini' : money(p.budget_cents)}
            </span>
            <span>
              Reste :{' '}
              {p.budget_cents === null
                ? 'Sans budget défini'
                : money(p.remaining_cents)}
            </span>
            <span>{p.suggestion_count} suggestion(s) en attente</span>
            {p.consumed_pct !== null && (
              <span>Budget consommé : {p.consumed_pct} %</span>
            )}
          </button>
        ))}
      </div>
      <Pagination
        offset={offset}
        step={100}
        more={resource.data?.has_more}
        setOffset={setOffset}
      />
      {selected && (
        <ProjectDetail
          key={`${selected}:${month}`}
          id={selected}
          month={month}
          revision={revision}
          onChanged={changed}
          onEdit={setForm}
          onNavigate={onNavigate}
        />
      )}
    </section>
  )
}
