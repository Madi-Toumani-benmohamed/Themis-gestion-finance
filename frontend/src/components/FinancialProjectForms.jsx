import { useState } from 'react'
import { api } from '../api'
import { useAnalysisResource } from './analysisResources'
import { ResourceState } from './FinancialProjectUI'
import {
  projectStatus,
  projectBudget,
  projectBudgetInput,
  projectReason,
} from './projectModel.mjs'
export function ProjectForm({ project, onSaved, onCancel }) {
  const [values, setValues] = useState({
    name: project?.name || '',
    description: project?.description || '',
    budget: projectBudgetInput(project?.budget_cents),
    start_date: project?.start_date || '',
    end_date: project?.end_date || '',
    status: project?.status || 'active',
  })
  const [busy, setBusy] = useState(false),
    [error, setError] = useState('')
  const field = (name) => ({
    value: values[name],
    onChange: (e) => setValues((v) => ({ ...v, [name]: e.target.value })),
  })
  const submit = async (e) => {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const { budget, ...data } = values
      const result = await api.saveProject(project?.id, {
        ...data,
        budget_cents: projectBudget(budget),
      })
      onSaved(result.id)
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <form
      className="project-form"
      onSubmit={submit}
      aria-label={project ? 'Modifier le projet' : 'Créer un projet'}
    >
      <h3>{project ? 'Modifier le projet' : 'Nouveau projet financier'}</h3>
      <label>
        Nom du projet
        <input required maxLength={120} {...field('name')} />
      </label>
      <label>
        Description
        <textarea maxLength={2000} {...field('description')} />
      </label>
      <label>
        Budget prévisionnel (€), facultatif
        <input
          inputMode="decimal"
          placeholder="Non défini"
          {...field('budget')}
        />
      </label>
      <div className="project-fields">
        <label>
          Date de début
          <input type="date" {...field('start_date')} />
        </label>
        <label>
          Date de fin
          <input type="date" {...field('end_date')} />
        </label>
      </div>
      <label>
        Statut
        <select {...field('status')}>
          {Object.entries(projectStatus).map(([v, label]) => (
            <option key={v} value={v}>
              {label}
            </option>
          ))}
        </select>
      </label>
      <p>
        Les dates bornent le coût observé. « Terminé » est un statut manuel et
        ne garantit pas la présence de tous les relevés.
      </p>
      {error && <p role="alert">{error}</p>}
      <div className="project-actions">
        <button className="btn-primary" disabled={busy}>
          {busy ? 'Enregistrement…' : 'Enregistrer le projet'}
        </button>
        <button
          type="button"
          className="btn-secondary"
          disabled={busy}
          onClick={onCancel}
        >
          Annuler
        </button>
      </div>
    </form>
  )
}
export function ProjectRules({ id, project, run, revision }) {
  const rules = useAnalysisResource(
    (signal) => api.projectRules(id, { signal }),
    [id, revision],
  )
  const references = useAnalysisResource(
    async (signal) => {
      const [tags, categories] = await Promise.all([
        api.tags({ signal }),
        api.categories({ signal }),
      ])
      return { tags, categories }
    },
    [id],
  )
  const [kind, setKind] = useState('tag'),
    [value, setValue] = useState('')
  const [from, setFrom] = useState(project.start_date || ''),
    [to, setTo] = useState(project.end_date || '')
  const choices =
    kind === 'tag' ? references.data?.tags : references.data?.categories
  const submit = (e) => {
    e.preventDefault()
    const criterion =
      kind === 'tag'
        ? { tag_id: Number(value) }
        : kind === 'category'
          ? { domain_id: Number(value) }
          : { match_text: value }
    run(async () => {
      await api.projectRuleSave(id, {
        kind,
        ...criterion,
        from_date: from,
        to_date: to,
      })
      await api.projectSuggest(id)
    })
  }
  return (
    <section>
      <h4>Règles de suggestion</h4>
      <p>
        Ces règles proposent des dépenses ; elles ne les confirment jamais.
        Choisis une période limitée. Un marchand peut correspondre à plusieurs
        besoins.
      </p>
      <ResourceState resource={rules} label="des règles" />
      <ResourceState resource={references} label="des critères" />
      {rules.data?.length === 0 && <p>Aucune règle définie.</p>}
      {rules.data?.map((rule) => (
        <div className="project-rule" key={rule.id}>
          <span>{projectReason(rule)}</span>
          <button
            className="btn-secondary"
            disabled={project.status === 'archived'}
            onClick={() => run(() => api.projectRuleRemove(id, rule.id))}
          >
            Désactiver la règle nº{rule.id}
          </button>
        </div>
      ))}
      {project.status === 'active' && (
        <form
          className="project-form"
          aria-label="Ajouter une règle"
          onSubmit={submit}
        >
          <label>
            Critère
            <select
              value={kind}
              onChange={(e) => {
                setKind(e.target.value)
                setValue('')
              }}
            >
              <option value="tag">Tag</option>
              <option value="category">Catégorie exacte</option>
              <option value="merchant">Marchand exact</option>
              <option value="label">Texte du libellé</option>
            </select>
          </label>
          {['tag', 'category'].includes(kind) ? (
            <label>
              Valeur du critère
              <select
                required
                value={value}
                onChange={(e) => setValue(e.target.value)}
              >
                <option value="">Choisir…</option>
                {(Array.isArray(choices) ? choices : []).map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </label>
          ) : (
            <label>
              Valeur du critère
              <input
                required
                minLength={3}
                maxLength={200}
                value={value}
                onChange={(e) => setValue(e.target.value)}
              />
            </label>
          )}
          <div className="project-fields">
            <label>
              Du
              <input
                required
                type="date"
                value={from}
                onChange={(e) => setFrom(e.target.value)}
              />
            </label>
            <label>
              Au
              <input
                required
                type="date"
                value={to}
                onChange={(e) => setTo(e.target.value)}
              />
            </label>
          </div>
          <button
            className="btn-secondary"
            disabled={references.loading || !!references.error}
          >
            Ajouter et rechercher
          </button>
        </form>
      )}
    </section>
  )
}
