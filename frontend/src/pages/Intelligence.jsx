import { useState } from 'react'
import { api } from '../api'
import { useAnalysisResource } from '../components/analysisResources'
import {
  analysisMoney,
  analysisPercent,
  analysisDate,
  analysisSource,
  analysisMonth,
} from '../components/analysisPageModel.mjs'

function Resource({ resource, children }) {
  if (resource.loading)
    return (
      <p role="status" className="analysis-state">
        Chargement de l’analyse…
      </p>
    )
  if (resource.error)
    return (
      <div role="alert" className="notice err">
        <p>{resource.error}</p>
        <button className="btn-secondary" onClick={resource.retry}>
          Réessayer
        </button>
      </div>
    )
  return children(resource.data)
}
export function NatureBreakdown({ report, onNavigate }) {
  if (!report?.available)
    return (
      <p className="analysis-state">
        Aucune donnée financière suffisamment documentée pour ce mois.
      </p>
    )
  const parts = [
    ['real_spend', 'Débits ordinaires hors impôts identifiés'],
    ['mandatory', 'Impôts et cotisations identifiés'],
    ['savings', 'Épargne normalisée sortante'],
    ['transfer_out', 'Autres transferts internes sortants'],
  ]
  return (
    <section className="analysis-panel">
      <h2>Où vont les sorties observées ?</h2>
      <p>
        Les quatre postes sont disjoints et se réconcilient avec les débits
        sources. Ils décrivent des opérations importées, sans estimation.
      </p>
      <dl className="analysis-definition">
        {parts.map(([key, label]) => (
          <div key={key}>
            <dt>{label}</dt>
            <dd>{analysisMoney(report[key])}</dd>
          </div>
        ))}
        <div>
          <dt>Total des sorties</dt>
          <dd>{analysisMoney(report.total_out)}</dd>
        </div>
      </dl>
      <h3>Lire les crédits séparément</h3>
      <dl className="analysis-definition">
        <div>
          <dt>Crédits ordinaires</dt>
          <dd>{analysisMoney(report.income)}</dd>
        </div>
        <div>
          <dt>Transferts et épargne entrants</dt>
          <dd>{analysisMoney(report.transfer_in)}</dd>
        </div>
        <div>
          <dt>Flux ordinaire net observé</dt>
          <dd>{analysisMoney(report.net)}</dd>
        </div>
      </dl>
      <p className="analysis-footnote">
        Le flux net correspond aux crédits ordinaires moins les débits
        ordinaires. Ce n’est pas un solde de compte. Les remboursements et
        annulations non rapprochés restent des crédits ; aucun appariement n’est
        supposé.
      </p>
      <button
        className="btn-secondary"
        onClick={() =>
          onNavigate?.('transactions', {
            month: report.month,
            from: report.query_from,
            to: report.query_to,
          })
        }
      >
        Vérifier les opérations sources
      </button>
    </section>
  )
}
export function MerchantsView({ month = analysisMonth(), onNavigate }) {
  const [months, setMonths] = useState(1)
  const resource = useAnalysisResource(
    (signal) => api.intelMerchants(months, month, { signal }),
    [month, months],
  )
  return (
    <section className="analysis-panel">
      <div className="analysis-panel-head">
        <h2>Chez qui les dépenses se concentrent-elles ?</h2>
        <div className="chart-controls">
          {[1, 3, 6].map((n) => (
            <button
              className="btn-secondary"
              key={n}
              aria-pressed={months === n}
              onClick={() => setMonths(n)}
            >
              {n} mois
            </button>
          ))}
        </div>
      </div>
      <Resource resource={resource}>
        {(data) => (
          <>
            <p>
              Du {analysisDate(data.query_from)} au{' '}
              {analysisDate(data.query_to)} · débits ordinaires observés :{' '}
              <strong>{analysisMoney(data.total)}</strong>.
            </p>
            <p className="analysis-footnote">
              {data.reason}
              {data.truncated &&
                ' Les 40 premiers marchands sont affichés ; les parts utilisent toutes les dépenses de la fenêtre.'}
            </p>
            {data.comparable && (
              <p className="analysis-footnote">
                Référence comparable : du {analysisDate(data.reference_from)} au{' '}
                {analysisDate(data.reference_to)}. Aucun pourcentage si le
                montant de référence est nul.
              </p>
            )}
            {data.merchants?.length ? (
              <div className="analysis-table-wrap">
                <table className="analysis-table">
                  <caption>
                    Montant et part sur la fenêtre ; panier moyen par
                    transaction, arrondi au centime.
                  </caption>
                  <thead>
                    <tr>
                      <th>Marchand</th>
                      <th className="amount">Dépensé</th>
                      <th className="amount">Part</th>
                      <th className="amount">Opérations</th>
                      <th className="amount">Panier moyen</th>
                      <th>Dernière opération</th>
                      {data.comparable && <th className="amount">Variation</th>}
                    </tr>
                  </thead>
                  <tbody>
                    {data.merchants.map((m) => (
                      <tr key={m.name}>
                        <td>
                          <button
                            className="link"
                            onClick={() =>
                              onNavigate?.('transactions', {
                                merchant: m.name,
                                from: data.query_from,
                                to: data.query_to,
                                type: 'debit',
                              })
                            }
                          >
                            {m.name}
                          </button>
                        </td>
                        <td className="amount">{analysisMoney(m.total)}</td>
                        <td className="amount">
                          {analysisPercent(m.share_pct)}
                        </td>
                        <td className="amount">{m.visits}</td>
                        <td className="amount">
                          {analysisMoney(m.avg_basket)}
                        </td>
                        <td>{analysisDate(m.last_seen)}</td>
                        {data.comparable && (
                          <td className="amount">
                            {analysisPercent(m.trend_pct)}
                          </td>
                        )}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p>Aucune dépense ordinaire observée sur cette fenêtre.</p>
            )}
          </>
        )}
      </Resource>
    </section>
  )
}
export function CategoryComparison({ month = analysisMonth(), onNavigate }) {
  const resource = useAnalysisResource(
    (signal) => api.intelComparison(month, { signal }),
    [month],
  )
  return (
    <section className="analysis-panel">
      <h2>Quelles catégories expliquent les écarts ?</h2>
      <Resource resource={resource}>
        {(data) => (
          <>
            <p className="analysis-state">{data.reason}</p>
            <p>
              Référence prévue : {data.reference_months?.join(' · ')}.{' '}
              {data.comparable
                ? `Diviseur commun : ${data.denominator} mois civils, y compris les mois documentés sans opération.`
                : 'Les moyennes et variations restent indisponibles.'}
            </p>
            {data.categories?.length ? (
              <div className="analysis-table-wrap">
                <table className="analysis-table">
                  <caption>
                    Débits ordinaires. Moyennes réparties au centime pour
                    conserver leur total.
                  </caption>
                  <thead>
                    <tr>
                      <th>Catégorie</th>
                      <th className="amount">Observé · {month}</th>
                      <th className="amount">Moyenne de référence</th>
                      <th className="amount">Écart</th>
                      <th className="amount">Variation</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.categories.map((c) => (
                      <tr key={c.id ?? 'none'}>
                        <td>
                          <button
                            className="link"
                            onClick={() =>
                              onNavigate?.('transactions', {
                                month,
                                domain: c.id ?? 'none',
                                type: 'debit',
                              })
                            }
                          >
                            {c.name}
                          </button>
                        </td>
                        <td className="amount">{analysisMoney(c.current)}</td>
                        <td className="amount">
                          {c.average == null
                            ? 'Non établie'
                            : analysisMoney(c.average)}
                        </td>
                        <td className="amount">
                          {c.delta == null
                            ? 'Non établi'
                            : analysisMoney(c.delta)}
                        </td>
                        <td className="amount">
                          {analysisPercent(c.delta_pct)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p>Aucune catégorie observée sur la période.</p>
            )}
          </>
        )}
      </Resource>
    </section>
  )
}
function NarrativeResult({ month, onNavigate }) {
  const resource = useAnalysisResource(
    (signal) => api.intelNarrative(month, { signal }),
    [month],
  )
  return (
    <Resource resource={resource}>
      {(data) =>
        data.available ? (
          <>
            <p>
              Le modèle priorise des constats calculés par Thémis. Le texte et
              les chiffres affichés proviennent exclusivement de ces constats.
            </p>
            {data.evidence?.map((f) => (
              <div className="analysis-signal" key={f.id}>
                <p>{f.text}</p>
                {f.source && (
                  <button
                    className="link"
                    onClick={() =>
                      onNavigate?.(
                        'transactions',
                        analysisSource(f.source, { month }),
                      )
                    }
                  >
                    Vérifier les sources →
                  </button>
                )}
              </div>
            ))}
            <p className="analysis-footnote">
              {data.model} · lecture facultative, sans modification des
              transactions.
            </p>
            <button className="link" onClick={resource.retry}>
              Actualiser la lecture assistée
            </button>
          </>
        ) : (
          <p role="status">{data.reason}</p>
        )
      }
    </Resource>
  )
}
export function Narrative({ month = analysisMonth(), onNavigate }) {
  const [run, setRun] = useState(false)
  return (
    <section className="analysis-panel">
      <div className="analysis-panel-head">
        <h2>Lecture assistée des constats</h2>
        {!run && (
          <button className="btn-secondary" onClick={() => setRun(true)}>
            Prioriser avec l’IA
          </button>
        )}
      </div>
      {run ? (
        <NarrativeResult month={month} onNavigate={onNavigate} />
      ) : (
        <p>
          Facultatif : utiliser le service IA existant pour prioriser les
          observations vérifiables de ce mois. Les analyses ci-dessus
          fonctionnent sans IA.
        </p>
      )}
    </section>
  )
}
export function Discoveries({ month = analysisMonth(), onNavigate }) {
  const resource = useAnalysisResource(
    (signal) => api.intelDiscoveries(month, { signal }),
    [month],
  )
  return (
    <section className="analysis-panel">
      <h2>Ce que Thémis a repéré</h2>
      <Resource resource={resource}>
        {(data) => (
          <>
            <p>
              Fenêtre d’observation : {analysisDate(data.query_from)} au{' '}
              {analysisDate(data.query_to)}. Les détections sont des pistes de
              vérification.
            </p>
            {data.discoveries?.length ? (
              data.discoveries.map((d, i) => (
                <article className="analysis-signal" key={d.id ?? i}>
                  <h3>{d.title}</h3>
                  <p>{d.detail}</p>
                  {d.status && (
                    <p>
                      Statut :{' '}
                      {d.status === 'confirmed' ? 'confirmé' : 'détecté'} ·
                      confiance{' '}
                      {d.confidence === 'probable'
                        ? 'probable'
                        : 'insuffisante'}
                    </p>
                  )}
                  {d.source?.from && (
                    <p className="analysis-footnote">
                      Sources du {analysisDate(d.source.from)} au{' '}
                      {analysisDate(d.source.to)}
                    </p>
                  )}
                  {d.source && (
                    <button
                      className="link"
                      onClick={() =>
                        onNavigate?.('transactions', analysisSource(d.source))
                      }
                    >
                      Vérifier les transactions →
                    </button>
                  )}
                </article>
              ))
            ) : (
              <p>
                Aucun motif suffisamment documenté détecté. Cela ne prouve pas
                l’absence de récurrences ou d’anomalies.
              </p>
            )}
            {data.limitations?.map((text) => (
              <p className="analysis-footnote" key={text}>
                {text}
              </p>
            ))}
          </>
        )}
      </Resource>
    </section>
  )
}
// Kept for consumers of the former combined view.
export default function Intelligence(props) {
  return (
    <>
      <Narrative {...props} />
      <Discoveries {...props} />
      <CategoryComparison {...props} />
      <MerchantsView {...props} />
    </>
  )
}
