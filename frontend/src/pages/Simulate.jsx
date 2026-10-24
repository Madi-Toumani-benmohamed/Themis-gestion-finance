import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { api } from '../api'
import {
  cents,
  emptyScenario,
  euroCents as eur,
  signedEuroCents,
  simulationModel,
} from '../components/simulationModel.mjs'
import './Simulate.css'

function Metric({ label, value, hint, accent }) {
  return (
    <div className={`scenario-metric${accent ? ' scenario-highlight' : ''}`}>
      <dt>{label}</dt>
      <dd>
        {value == null ? '—' : eur(value)}
        <small>{hint}</small>
      </dd>
    </div>
  )
}
function MoneyInput({ label, value, onChange, placeholder }) {
  const id = useId()
  return (
    <label className="scenario-field" htmlFor={id}>
      <span>{label}</span>
      <div className="scenario-input-unit">
        <input
          id={id}
          type="number"
          min="0"
          max="9999999999.99"
          step="0.01"
          inputMode="decimal"
          value={value}
          placeholder={placeholder}
          onChange={(e) => onChange(e.target.value)}
        />
        <span aria-hidden="true">€</span>
      </div>
    </label>
  )
}
export default function Simulate() {
  const [reference, setReference] = useState(null)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(true)
  const [retry, setRetry] = useState(0)
  const [scenario, setScenario] = useState(emptyScenario)
  const [query, setQuery] = useState('')
  const [modifiedOnly, setModifiedOnly] = useState(false)
  const [openCategories, setOpenCategories] = useState({})
  const nextId = useRef(1)
  const pageRef = useRef(null)
  useEffect(() => {
    let active = true
    setLoading(true)
    setError(null)
    api
      .simulate({})
      .then((data) => {
        simulationModel(data, emptyScenario())
        if (active) setReference(data)
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
  }, [retry])
  // The shared header can wrap at zoom or when account names are long.
  useEffect(() => {
    const header = document.querySelector('.topbar')
    if (!header) return
    const observer = new ResizeObserver(() => {
      const page = pageRef.current
      if (!page) return
      const bounds = page.getBoundingClientRect()
      page.style.setProperty(
        '--scenario-top',
        `${header.getBoundingClientRect().height + 16}px`,
      )
      page.style.setProperty('--scenario-left', `${bounds.left}px`)
      page.style.setProperty(
        '--scenario-right',
        `${Math.max(12, window.innerWidth - bounds.right)}px`,
      )
    })
    observer.observe(header)
    if (pageRef.current) observer.observe(pageRef.current)
    return () => observer.disconnect()
  }, [])
  const computed = useMemo(() => {
    try {
      return { value: simulationModel(reference, scenario), error: null }
    } catch (e) {
      return { value: null, error: e.message }
    }
  }, [reference, scenario])
  const model = computed.value
  const usable = reference && reference.available !== false
  const patch = (values) =>
    setScenario((previous) => ({ ...previous, ...values }))
  const setCut = (category, value) =>
    setScenario((previous) => ({
      ...previous,
      cuts: { ...previous.cuts, [category]: value },
    }))
  const setExpense = (id, key, value) =>
    setScenario((previous) => ({
      ...previous,
      expenses: previous.expenses.map((e) =>
        e.id === id ? { ...e, [key]: value } : e,
      ),
    }))
  const reset = () => {
    setScenario(emptyScenario())
    setModifiedOnly(false)
  }
  const visible = (reference?.categories ?? []).filter(
    (c) =>
      c.category
        .toLocaleLowerCase('fr')
        .includes(query.toLocaleLowerCase('fr')) &&
      (!modifiedOnly || Number(scenario.cuts[c.category]) > 0),
  )
  const categoryModels = new Map(
    (model?.categories ?? []).map((c) => [c.name, c]),
  )
  const periods = reference?.reference_periods ?? []
  const periodLabel = periods
    .map((p) =>
      new Date(`${p.month}-01T12:00:00Z`).toLocaleDateString('fr-FR', {
        month: 'short',
        year: 'numeric',
        timeZone: 'UTC',
      }),
    )
    .join(' · ')
  return (
    <div className="section scenario-page" ref={pageRef}>
      <header className="scenario-heading">
        <div>
          <span className="eyebrow">Planifier · Scénario mensuel</span>
          <h2>
            Et si tu ajustais <em>ton budget ?</em>
          </h2>
          <p className="muted">
            Explore tes marges de manœuvre. Chaque réglage actualise
            l’estimation.
          </p>
        </div>
        <button
          className="btn btn-secondary"
          onClick={reset}
          disabled={!usable}
        >
          Réinitialiser le scénario
        </button>
      </header>
      {loading && (
        <p role="status" className="notice">
          Chargement des moyennes de référence…
        </p>
      )}
      {error && (
        <div role="alert" className="notice err">
          {error}{' '}
          <button className="link" onClick={() => setRetry((n) => n + 1)}>
            Réessayer
          </button>
        </div>
      )}
      {!loading && reference?.available === false && (
        <p className="empty">{reference.reason}</p>
      )}
      {usable && (
        <>
          <div className="scenario-source">
            <span
              className={`scenario-badge${reference.reference_complete ? '' : ' provisional'}`}
            >
              {reference.reference_complete
                ? 'Référence documentée'
                : 'Référence provisoire'}
            </span>
            <p>
              Moyennes sur 3 mois terminés{periodLabel && <> : {periodLabel}</>}
              . Le mois en cours est exclu.
            </p>
          </div>
          {!reference.reference_complete && (
            <p className="scenario-caution">
              La couverture de ces trois mois est partielle ou inconnue. Les
              moyennes portent sur les opérations importées et peuvent
              sous-estimer revenus et dépenses. L’absence de salaire importé ne
              prouve pas un revenu nul.
            </p>
          )}
          {computed.error && (
            <p role="alert" className="notice err">
              {computed.error} Les résultats sont suspendus jusqu’à correction.
            </p>
          )}
          <dl className="scenario-summary" aria-label="Synthèse mensuelle">
            <Metric
              label="Revenu de référence"
              value={model?.observedIncome}
              hint="Moyenne des crédits importés"
            />
            <Metric
              label="Dépenses observées"
              value={model?.baseSpend}
              hint="Moyenne hors transferts internes"
            />
            <Metric
              label="Reste à vivre actuel"
              value={model?.currentNet}
              hint="Revenu de référence − dépenses"
            />
            <Metric
              label="Reste à vivre simulé"
              value={model?.simulatedNet}
              hint="Estimation selon tes hypothèses"
              accent
            />
            <div className="scenario-metric scenario-variation">
              <dt>Variation estimée</dt>
              <dd>
                {model ? signedEuroCents(model.delta) : '—'}
                <small>
                  {!model
                    ? 'Saisie à vérifier'
                    : model.delta > 0
                      ? 'De marge en plus par mois'
                      : model.delta < 0
                        ? 'De marge en moins par mois'
                        : 'Scénario inchangé'}
                </small>
              </dd>
            </div>
          </dl>
          <div className="scenario-layout">
            <div className="scenario-settings">
              <section
                className="scenario-card"
                aria-labelledby="scenario-categories-title"
              >
                <div className="scenario-card-head">
                  <div>
                    <span className="eyebrow">01 · Ajuster</span>
                    <h3 id="scenario-categories-title">
                      Tes postes de dépenses
                    </h3>
                  </div>
                  <span className="scenario-count">
                    {
                      Object.values(scenario.cuts).filter((p) => Number(p) > 0)
                        .length
                    }{' '}
                    modifié(s)
                  </span>
                </div>
                <p className="scenario-help">
                  Ouvre un poste pour régler sa réduction. Tes autres réglages
                  restent conservés.
                </p>
                <div className="scenario-filters">
                  <label className="scenario-search">
                    <span className="sr-only">Rechercher une catégorie</span>
                    <input
                      type="search"
                      placeholder="Rechercher une catégorie…"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                    />
                  </label>
                  <label className="scenario-check">
                    <input
                      type="checkbox"
                      checked={modifiedOnly}
                      onChange={(e) => setModifiedOnly(e.target.checked)}
                    />
                    Modifiées uniquement
                  </label>
                </div>
                <div
                  className="scenario-list"
                  tabIndex="0"
                  role="region"
                  aria-label="Catégories ajustables, liste défilante"
                >
                  {visible.map((c, i) => {
                    const values = categoryModels.get(c.category),
                      pct = scenario.cuts[c.category] ?? 0
                    return (
                      <details
                        className={`scenario-category${Number(pct) > 0 ? ' is-modified' : ''}`}
                        key={c.category}
                        open={openCategories[c.category] ?? false}
                        onToggle={(e) => {
                          const open = e.currentTarget.open
                          setOpenCategories((previous) =>
                            previous[c.category] === open
                              ? previous
                              : { ...previous, [c.category]: open },
                          )
                        }}
                      >
                        <summary>
                          <span className="scenario-category-name">
                            {c.category}
                          </span>
                          <span className="scenario-category-base">
                            {eur(values?.before ?? cents(c.before))}
                            <small>référence / mois</small>
                          </span>
                          <span className="scenario-category-cut">
                            −{pct || 0} %
                            <small>
                              {values
                                ? `${eur(values.saved)} économisés`
                                : 'saisie à vérifier'}
                            </small>
                          </span>
                          <span className="scenario-chevron" aria-hidden="true">
                            ⌄
                          </span>
                        </summary>
                        <div className="scenario-category-controls">
                          <label htmlFor={`scenario-range-${i}`}>
                            Réduction de {c.category}
                          </label>
                          <input
                            id={`scenario-range-${i}`}
                            aria-label={`Réduction de ${c.category}, curseur`}
                            type="range"
                            min="0"
                            max="100"
                            step="1"
                            value={pct || 0}
                            onChange={(e) => setCut(c.category, e.target.value)}
                          />
                          <div className="scenario-control-foot">
                            <label>
                              Réduction (%)
                              <input
                                aria-label={`Réduction de ${c.category} en pourcentage`}
                                type="number"
                                min="0"
                                max="100"
                                step="0.01"
                                inputMode="decimal"
                                value={pct}
                                onChange={(e) =>
                                  setCut(c.category, e.target.value)
                                }
                              />
                            </label>
                            <span>
                              Après ajustement{' '}
                              <strong>
                                {values ? eur(values.after) : '—'}
                              </strong>
                            </span>
                            <button
                              className="link"
                              disabled={!Number(pct)}
                              onClick={() => setCut(c.category, 0)}
                            >
                              Réinitialiser {c.category}
                            </button>
                          </div>
                        </div>
                      </details>
                    )
                  })}
                  {!visible.length && (
                    <p className="scenario-help">
                      {reference.categories.length
                        ? 'Aucune catégorie ne correspond aux filtres.'
                        : 'Aucune dépense dans les opérations de référence.'}
                    </p>
                  )}
                </div>
              </section>
              <section
                className="scenario-card scenario-options"
                aria-labelledby="scenario-options-title"
              >
                <div className="scenario-card-head">
                  <div>
                    <span className="eyebrow">02 · Compléter</span>
                    <h3 id="scenario-options-title">Les autres hypothèses</h3>
                  </div>
                </div>
                <details>
                  <summary>
                    Revenu mensuel du scénario{' '}
                    <small>
                      {scenario.income === ''
                        ? 'Référence conservée'
                        : 'Revenu personnalisé'}
                    </small>
                  </summary>
                  <div className="scenario-option-body">
                    <MoneyInput
                      label="Revenu simulé (€/mois)"
                      value={scenario.income}
                      placeholder="Moyenne de référence"
                      onChange={(income) => patch({ income })}
                    />
                    <p className="scenario-help">
                      Laisse vide pour conserver le revenu observé. Cette
                      hypothèse ne change pas la situation actuelle.
                    </p>
                  </div>
                </details>
                <details>
                  <summary>
                    Nouvelles dépenses{' '}
                    <small>
                      {scenario.expenses.filter((e) => e.amount !== '').length}{' '}
                      ligne(s) renseignée(s)
                    </small>
                  </summary>
                  <div className="scenario-option-body">
                    {scenario.expenses.map((e, i) => (
                      <div className="scenario-expense" key={e.id}>
                        <label className="scenario-field">
                          Intitulé
                          <input
                            value={e.label}
                            placeholder="Nom de la dépense"
                            onChange={(ev) =>
                              setExpense(e.id, 'label', ev.target.value)
                            }
                          />
                        </label>
                        <MoneyInput
                          label="Montant (€/mois)"
                          value={e.amount}
                          onChange={(amount) =>
                            setExpense(e.id, 'amount', amount)
                          }
                        />
                        <button
                          className="link"
                          aria-label={`Retirer la dépense ${e.label || i + 1}`}
                          onClick={() =>
                            patch({
                              expenses: scenario.expenses.filter(
                                (row) => row.id !== e.id,
                              ),
                            })
                          }
                        >
                          Retirer
                        </button>
                      </div>
                    ))}
                    <button
                      className="link"
                      onClick={() =>
                        patch({
                          expenses: [
                            ...scenario.expenses,
                            { id: nextId.current++, label: '', amount: '' },
                          ],
                        })
                      }
                    >
                      ＋ Ajouter une dépense
                    </button>
                  </div>
                </details>
                <details>
                  <summary>
                    Simuler un emprunt{' '}
                    <small>
                      {scenario.loan.amount
                        ? 'Hypothèse de crédit'
                        : 'Facultatif'}
                    </small>
                  </summary>
                  <div className="scenario-option-body">
                    <div className="scenario-loan">
                      <MoneyInput
                        label="Capital emprunté (€)"
                        value={scenario.loan.amount}
                        onChange={(amount) =>
                          patch({ loan: { ...scenario.loan, amount } })
                        }
                      />
                      <label className="scenario-field">
                        Taux annuel (%)
                        <input
                          type="number"
                          min="0"
                          max="100"
                          step="0.01"
                          value={scenario.loan.rate}
                          onChange={(e) =>
                            patch({
                              loan: { ...scenario.loan, rate: e.target.value },
                            })
                          }
                        />
                      </label>
                      <label className="scenario-field">
                        Durée (mois)
                        <input
                          type="number"
                          min="1"
                          max="600"
                          step="1"
                          value={scenario.loan.months}
                          onChange={(e) =>
                            patch({
                              loan: {
                                ...scenario.loan,
                                months: e.target.value,
                              },
                            })
                          }
                        />
                      </label>
                    </div>
                    <p className="scenario-help">
                      Mensualité estimée à taux fixe, hors assurance et frais.
                      Le capital reçu n’est pas ajouté au revenu.
                    </p>
                    <button
                      className="link"
                      onClick={() => patch({ loan: emptyScenario().loan })}
                    >
                      Effacer l’emprunt
                    </button>
                  </div>
                </details>
              </section>
            </div>
            <aside
              className="scenario-results scenario-card"
              aria-labelledby="scenario-results-title"
            >
              <span className="eyebrow">03 · Comprendre</span>
              <h3 id="scenario-results-title">L’impact de ton scénario</h3>
              <div
                className="scenario-verdict"
                role="status"
                aria-live="polite"
                aria-atomic="true"
              >
                <span>Reste à vivre simulé</span>
                <strong>{model ? eur(model.simulatedNet) : '—'}</strong>
                <span>
                  {model
                    ? `${signedEuroCents(model.delta)} / mois par rapport à la référence`
                    : 'Vérifie les valeurs saisies'}
                </span>
              </div>
              <table className="scenario-comparison">
                <caption className="sr-only">
                  Comparaison mensuelle de la référence et du scénario estimé
                </caption>
                <thead>
                  <tr>
                    <th scope="col">Par mois</th>
                    <th scope="col">Actuel</th>
                    <th scope="col">Simulé</th>
                  </tr>
                </thead>
                <tbody>
                  {[
                    ['Revenu', model?.observedIncome, model?.income],
                    ['Dépenses', model?.baseSpend, model?.spend],
                    ['Reste à vivre', model?.currentNet, model?.simulatedNet],
                  ].map(([label, a, b]) => (
                    <tr key={label}>
                      <th scope="row">{label}</th>
                      <td>{a == null ? '—' : eur(a)}</td>
                      <td>{b == null ? '—' : eur(b)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {model && (
                <>
                  <dl className="scenario-breakdown">
                    <div>
                      <dt>Économies sur les catégories</dt>
                      <dd>{signedEuroCents(model.saved)}</dd>
                    </div>
                    <div>
                      <dt>Nouvelles dépenses</dt>
                      <dd>{eur(-model.added)}</dd>
                    </div>
                    {model.loan.payment > 0 && (
                      <div>
                        <dt>Mensualité d’emprunt</dt>
                        <dd>{eur(-model.loan.payment)}</dd>
                      </div>
                    )}
                    {model.income !== model.observedIncome && (
                      <div>
                        <dt>Variation du revenu</dt>
                        <dd>
                          {signedEuroCents(model.income - model.observedIncome)}
                        </dd>
                      </div>
                    )}
                    <div className="scenario-total">
                      <dt>Variation du reste à vivre</dt>
                      <dd>{signedEuroCents(model.delta)}</dd>
                    </div>
                  </dl>
                  {model.loan.payment > 0 && (
                    <p className="scenario-help">
                      Total remboursé estimé : {eur(model.loan.total)}, dont{' '}
                      {eur(model.loan.interest)} d’intérêts estimés. La dernière
                      échéance peut différer pour ajuster les arrondis.
                    </p>
                  )}
                </>
              )}
              <p className="scenario-footnote">
                Une projection, pas un solde bancaire garanti. Moyennes et
                économies arrondies au centime par catégorie, puis additionnées.
              </p>
            </aside>
          </div>
          <div className="scenario-mobile-result" aria-hidden="true">
            <span>
              Reste à vivre estimé
              <strong>{model ? eur(model.simulatedNet) : '—'}</strong>
            </span>
            <span>
              Variation / mois
              <strong>{model ? signedEuroCents(model.delta) : '—'}</strong>
            </span>
          </div>
        </>
      )}
    </div>
  )
}
