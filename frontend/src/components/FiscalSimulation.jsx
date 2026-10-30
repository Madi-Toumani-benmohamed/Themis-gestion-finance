import { useEffect, useRef, useState } from 'react'
import { api } from '../api'
import { cents, euroCents } from './money.mjs'
import { useAnalysisResource } from './analysisResources'
import './laboratory/laboratory.css'
import { fiscalResult } from './fiscalModel.mjs'
const optional = (value) =>
  value.trim() === '' ? null : cents(value.replace(',', '.'))
const money = (value) =>
  value === null || value === undefined ? 'Non renseigné' : euroCents(value)
export default function FiscalSimulation() {
  const rules = useAnalysisResource((signal) => api.fiscalRules({ signal }), [])
  const [v, setV] = useState({
    year: '2025',
    family: 'single',
    children: '0',
    isolated: false,
    basis: 'salary',
    period: 'annual',
    months: '12',
    income: '',
    spouse: '',
    deduction: 'standard',
    actual: '',
    spouseActual: '',
    scope: false,
    pas: '',
    pasYear: '2025',
    pasMonths: '12',
    futurePas: '',
    project: false,
    scenario: '',
  })
  const [result, setResult] = useState(null),
    [comparison, setComparison] = useState(null),
    [error, setError] = useState(''),
    [busy, setBusy] = useState(false)
  const sequence = useRef(0)
  useEffect(
    () => () => {
      sequence.current++
    },
    [],
  )
  const change = (key, value) => {
    sequence.current++
    setBusy(false)
    setResult(null)
    setComparison(null)
    setError('')
    setV((s) => ({
      ...s,
      [key]: value,
      ...(key === 'family' ? { isolated: false } : {}),
    }))
  }
  const field = (key) => ({
    value: v[key],
    onChange: (e) => change(key, e.target.value),
  })
  const joint = ['married', 'pacs'].includes(v.family)
  const payload = () => {
    const paid = optional(v.pas)
    return {
      income_year: Number(v.year),
      family: v.family,
      children: Number(v.children),
      isolated_parent: v.isolated,
      scope_confirmed: v.scope,
      basis: v.basis,
      income_period: v.period,
      observed_months: v.period === 'annual' ? 12 : Number(v.months),
      project_income: v.project,
      income_cents: cents(v.income.replace(',', '.')),
      spouse_income_cents:
        joint && v.basis === 'salary' ? cents(v.spouse.replace(',', '.')) : 0,
      deduction: v.basis === 'final_taxable' ? 'standard' : v.deduction,
      actual_costs_cents:
        v.basis === 'salary' && v.deduction === 'actual'
          ? (optional(v.actual) ?? 0)
          : 0,
      spouse_actual_costs_cents:
        joint && v.basis === 'salary' && v.deduction === 'actual'
          ? (optional(v.spouseActual) ?? 0)
          : 0,
      withholding_year: paid === null ? 0 : Number(v.pasYear),
      withholding_observed_cents: paid,
      withholding_months: paid === null ? 0 : Number(v.pasMonths),
      future_monthly_withholding_cents:
        paid === null || v.pasMonths === '12' ? null : optional(v.futurePas),
    }
  }
  const submit = async (e) => {
    e.preventDefault()
    const id = ++sequence.current
    setBusy(true)
    setError('')
    setResult(null)
    setComparison(null)
    try {
      const input = payload()
      if (v.scenario.trim() && joint && v.period !== 'annual')
        throw new Error(
          'Pour comparer les revenus d’un couple, utilise une saisie annuelle.',
        )
      const r = fiscalResult(await api.fiscalSimulate(input), input.income_year)
      if (id !== sequence.current) return
      setResult(r)
      if (v.scenario.trim()) {
        const alt = fiscalResult(
          await api.fiscalSimulate({
            ...input,
            income_period: 'annual',
            observed_months: 12,
            project_income: false,
            income_cents: cents(v.scenario.replace(',', '.')),
          }),
          input.income_year,
        )
        if (id === sequence.current) setComparison(alt)
      }
    } catch (e) {
      if (id === sequence.current) setError(e.message)
    } finally {
      if (id === sequence.current) setBusy(false)
    }
  }
  return (
    <section
      className="laboratory fiscal-simulation"
      aria-label="Simulation fiscale"
    >
      <h2>Fiscalité · impôt sur le revenu</h2>
      <p>
        Simulation déterministe, sans IA et sans écriture dans vos obligations.
        Le net fiscal de la fiche de paie n’est ni le brut, ni le net social, ni
        le net payé.
      </p>
      {rules.loading && <p role="status">Chargement des règles fiscales…</p>}
      {rules.error && (
        <p role="alert">
          {rules.error} <button onClick={rules.retry}>Réessayer</button>
        </p>
      )}
      <form onSubmit={submit}>
        <div className="lab-fields">
          <label>
            Année des revenus
            <select {...field('year')}>
              {(rules.data || []).map((r) => (
                <option value={r.income_year} key={r.version}>
                  {r.income_year} · imposition {r.assessment_year}
                </option>
              ))}
            </select>
          </label>
          <label>
            Situation familiale
            <select {...field('family')}>
              <option value="single">Célibataire</option>
              <option value="married">Marié · déclaration commune</option>
              <option value="pacs">Pacsé · déclaration commune</option>
              <option value="divorced">Divorcé</option>
              <option value="widowed">Veuf sans enfant à charge</option>
            </select>
          </label>
          <label>
            Enfants mineurs à charge exclusive
            <input
              required
              type="number"
              min="0"
              max="10"
              step="1"
              {...field('children')}
            />
          </label>
          <label>
            Base de revenu
            <select {...field('basis')}>
              <option value="salary">
                Salaire net fiscal avant frais professionnels
              </option>
              <option value="final_taxable">
                Revenu net imposable final déjà après déductions
              </option>
            </select>
          </label>
          <label>
            Période de saisie
            <select {...field('period')}>
              <option value="annual">Année entière</option>
              <option value="monthly">Revenu mensuel représentatif</option>
              <option value="cumulative">Cumul partiel observé</option>
            </select>
          </label>
          <label>
            {v.basis === 'salary'
              ? 'Salaire net fiscal'
              : 'Revenu net imposable final'}{' '}
            (€)
            <input required inputMode="decimal" {...field('income')} />
          </label>
          {v.basis === 'final_taxable' && (
            <p>
              Base finale du foyer entier, conjoint compris, après toutes les
              déductions prises en charge.
            </p>
          )}
          {joint && v.basis === 'salary' && (
            <label>
              Revenu du conjoint, même base et période (€)
              <input required inputMode="decimal" {...field('spouse')} />
            </label>
          )}
          {v.period === 'cumulative' && (
            <label>
              Mois de revenus observés
              <input
                required
                type="number"
                min="1"
                max="12"
                {...field('months')}
              />
            </label>
          )}
        </div>
        {['single', 'divorced'].includes(v.family) &&
          Number(v.children) > 0 && (
            <label>
              <span>
                <input
                  type="checkbox"
                  checked={v.isolated}
                  onChange={(e) => change('isolated', e.target.checked)}
                />{' '}
                Je vis seul avec mes enfants et remplis la condition de parent
                isolé (case T)
              </span>
            </label>
          )}
        {v.period !== 'annual' && (
          <label>
            <span>
              <input
                required
                type="checkbox"
                checked={v.project}
                onChange={(e) => change('project', e.target.checked)}
              />{' '}
              J’autorise une annualisation à rythme constant ; ce revenu annuel
              est projeté, pas observé.
            </span>
          </label>
        )}
        <details>
          <summary>Frais professionnels et comparaison de revenus</summary>
          {v.basis === 'salary' && (
            <>
              <label>
                Déduction
                <select {...field('deduction')}>
                  <option value="standard">Forfait légal de 10 %</option>
                  <option value="actual">Frais réels qualifiés</option>
                </select>
              </label>
              {v.deduction === 'actual' && (
                <div className="lab-fields">
                  <label>
                    Frais réels annuels nets (€)
                    <input inputMode="decimal" required {...field('actual')} />
                  </label>
                  {joint && (
                    <label>
                      Frais réels annuels du conjoint (€)
                      <input
                        inputMode="decimal"
                        required
                        {...field('spouseActual')}
                      />
                    </label>
                  )}
                </div>
              )}
              <p>
                Les frais réels doivent déjà tenir compte des remboursements
                employeur imposables et être justifiables. Aucun cumul avec le
                forfait ; déficits exclus.
              </p>
            </>
          )}
          <label>
            Autre revenu annuel du déclarant à comparer (€), facultatif
            <input inputMode="decimal" {...field('scenario')} />
          </label>
          <p>
            Le scénario réutilise ce barème et les autres paramètres ; ce n’est
            pas une prévision de législation future. Le conjoint conserve la
            période de base : utilise une saisie annuelle pour comparer un
            couple.
          </p>
        </details>
        <fieldset>
          <legend>Prélèvement à la source de la même année</legend>
          <p>
            Saisis un cumul réellement prélevé, pas une mensualité. Laisse vide
            si inconnu : aucun solde ne sera inventé.
          </p>
          <div className="lab-fields">
            <label>
              PAS déjà prélevé, cumul (€)
              <input inputMode="decimal" {...field('pas')} />
            </label>
            {v.pas.trim() && (
              <>
                <label>
                  Année de revenus du PAS
                  <input type="number" required {...field('pasYear')} />
                </label>
                <label>
                  Mois de PAS déjà couverts
                  <input
                    type="number"
                    min="1"
                    max="12"
                    required
                    {...field('pasMonths')}
                  />
                </label>
                {v.pasMonths !== '12' && (
                  <label>
                    PAS mensuel futur supposé (€), facultatif
                    <input inputMode="decimal" {...field('futurePas')} />
                  </label>
                )}
              </>
            )}
          </div>
        </fieldset>
        <p className="lab-notice">
          Périmètre V1 : résident de métropole, moins de 65 ans, situation
          stable, salaires ordinaires ou base finale déjà déterminée, enfants
          mineurs à charge exclusive. Hors garde alternée, invalidité,
          décès/changement familial dans l’année, DOM/non-résidence, hauts
          revenus, réductions/crédits et autres régimes spéciaux. Années non
          disponibles refusées.
        </p>
        <label>
          <span>
            <input
              type="checkbox"
              required
              checked={v.scope}
              onChange={(e) => change('scope', e.target.checked)}
            />{' '}
            Je confirme que ma situation entre dans ce périmètre, sans
            dispositif applicable non pris en charge.
          </span>
        </label>
        <button
          className="btn"
          disabled={busy || rules.loading || !!rules.error}
        >
          {busy ? 'Calcul en cours…' : 'Estimer mon impôt'}
        </button>
      </form>
      {error && <p role="alert">{error}</p>}
      {result && (
        <section aria-label="Résultat fiscal">
          <h3>Estimation sur les revenus {result.rules.income_year}</h3>
          <p>
            {result.rules.version} · imposition {result.rules.assessment_year} ·
            règles officielles vérifiées au {result.rules.verified_on}.
            Simulation non officielle.
          </p>
          <dl className="lab-metrics">
            <div>
              <dt>Impôt annuel estimé</dt>
              <dd>{money(result.tax_cents)}</dd>
            </div>
            <div>
              <dt>Taux moyen sur la base imposable</dt>
              <dd>{result.average_rate_pct} %</dd>
            </div>
            <div>
              <dt>Tranche marginale retenue</dt>
              <dd>{result.marginal_rate_pct} %</dd>
            </div>
            <div>
              <dt>PAS déjà observé</dt>
              <dd>{money(result.observed_paid_cents)}</dd>
            </div>
            <div>
              <dt>PAS futurs projetés</dt>
              <dd>{money(result.projected_future_paid_cents)}</dd>
            </div>
          </dl>
          {result.settlement_cents !== null ? (
            <p className="lab-notice">
              {result.settlement_cents > 0
                ? 'Complément potentiel'
                : result.settlement_cents < 0
                  ? 'Remboursement potentiel'
                  : 'Équilibre estimé'}{' '}
              : {money(Math.abs(result.settlement_cents))}.{' '}
              {result.settlement_status === 'projected'
                ? 'Projection sur hypothèses.'
                : 'Estimation annuelle, sous réserve des autres éléments fiscaux.'}
            </p>
          ) : (
            <p>
              Pas de solde final estimable : PAS inconnu ou incomplet.
              {result.gap_before_future_cents !== null &&
                ` Écart à l’impôt annuel avant futurs PAS : ${money(result.gap_before_future_cents)}.`}
            </p>
          )}
          {comparison && (
            <p>
              Scénario comparé : {money(comparison.tax_cents)} d’impôt ;
              variation de {money(comparison.tax_cents - result.tax_cents)}, à
              règles et périmètre identiques.
            </p>
          )}
          <details>
            <summary>
              Comprendre le calcul, les hypothèses et les sources
            </summary>
            <p>
              {result.half_parts / 2} part(s).{' '}
              {result.family_cap_applied
                ? 'Avantage familial plafonné.'
                : 'Plafonnement familial non limitant.'}
            </p>
            <table className="lab-plain-table">
              <caption>Étapes du calcul fiscal</caption>
              <thead>
                <tr>
                  <th>Étape</th>
                  <th>Montant</th>
                </tr>
              </thead>
              <tbody>
                {result.steps.map((step) => (
                  <tr key={step.label}>
                    <th scope="row">{step.label}</th>
                    <td>{money(step.amount_cents)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <ul>
              {[...result.assumptions, ...result.limitations].map((text) => (
                <li key={text}>{text}</li>
              ))}
            </ul>
            <ul>
              {result.rules.sources.map((url) => (
                <li key={url}>
                  <a href={url} target="_blank" rel="noreferrer">
                    Source officielle du millésime{' '}
                    {result.rules.assessment_year}
                  </a>
                </li>
              ))}
            </ul>
          </details>
        </section>
      )}
    </section>
  )
}
