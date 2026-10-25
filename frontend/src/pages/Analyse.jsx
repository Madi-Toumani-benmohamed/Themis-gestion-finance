import { useState } from 'react'
import { api } from '../api'
import {
  Narrative,
  Discoveries,
  NatureBreakdown,
  CategoryComparison,
  MerchantsView,
} from './Intelligence'
import TagFolders from './TagFolders'
import CategoryChart from './CategoryChart'
import TagAnalysis from './TagAnalysis'
import TagManager from './TagManager'
import { useAnalysisResource } from '../components/analysisResources'
import {
  analysisNatureModel,
  analysisMonth,
  analysisMoney,
  analysisDate,
  analysisPeriodLabel,
  analysisPercent,
} from '../components/analysisPageModel.mjs'
import './Analyse.css'

const FAMILIES = [
  ['synthese', 'Synthèse'],
  ['flux', 'Flux financiers'],
  ['habitudes', 'Habitudes'],
  ['pilotage', 'Pilotage'],
]
const DETAILS = {
  flux: [
    ['nature', 'Nature des flux'],
    ['categories', 'Par catégorie'],
    ['marchands', 'Par marchand'],
  ],
  habitudes: [
    ['comparaison', 'Comparaisons'],
    ['reperage', 'Repérages'],
  ],
  pilotage: [
    ['projets', 'Projets par tag'],
    ['tags', 'Analyse par tag'],
    ['gerer-tags', 'Gérer les tags'],
  ],
}
export function AnalysisState({ resource, children }) {
  if (resource.loading)
    return (
      <p className="analysis-state" role="status">
        Chargement de l’analyse…
      </p>
    )
  if (resource.error)
    return (
      <div className="analysis-state notice err" role="alert">
        <p>{resource.error}</p>
        <button className="btn-secondary" onClick={resource.retry}>
          Réessayer
        </button>
      </div>
    )
  return children(resource.data)
}
function Summary({ report, onNavigate, onExplore }) {
  if (!report.available)
    return (
      <div className="analysis-state">
        <h2>Données insuffisantes pour ce mois</h2>
        <p>
          Aucune opération observée et aucune couverture complète établie. Cette
          absence ne constitue pas un montant nul.
        </p>
      </div>
    )
  const source = (opts) =>
    onNavigate?.('transactions', {
      month: report.month,
      from: report.query_from,
      to: report.query_to,
      ...opts,
    })
  return (
    <>
      <div className="analysis-summary-grid">
        <section className="analysis-panel">
          <div className="analysis-panel-head">
            <h2>Ce qui explique les sorties</h2>
            <button
              className="link"
              onClick={() => onExplore('flux', 'nature')}
            >
              Décomposer les flux →
            </button>
          </div>
          <p className="analysis-lead">
            <strong>{analysisMoney(report.expenses)}</strong> de débits
            ordinaires observés
          </p>
          <p>
            Hors épargne et transferts internes. Les crédits ordinaires, y
            compris les remboursements non rapprochés, sont présentés
            séparément.
          </p>
          {report.categories.length ? (
            <ol className="analysis-ranking">
              {report.categories.slice(0, 5).map((c) => (
                <li key={c.id ?? 'none'}>
                  <div>
                    <button
                      className="link"
                      onClick={() =>
                        source({ domain: c.id ?? 'none', type: 'debit' })
                      }
                    >
                      {c.name}
                    </button>
                    <small>
                      {c.transaction_count} opération(s) ·{' '}
                      {analysisPercent(c.share_pct)} des débits ordinaires
                    </small>
                  </div>
                  <strong>{analysisMoney(c.spent)}</strong>
                </li>
              ))}
            </ol>
          ) : (
            <p>Aucun débit ordinaire observé sur cette période.</p>
          )}
          <button
            className="btn-secondary"
            onClick={() => source({ type: 'debit' })}
          >
            Consulter les dépenses sources
          </button>
        </section>
        <section className="analysis-panel">
          <h2>Avant d’interpréter</h2>
          <p>
            <strong>{analysisPeriodLabel(report.period)}</strong>
          </p>
          <p>
            Le salaire arrive généralement en fin de mois ; les relevés sont
            disponibles autour du 21. La dernière transaction ne prouve pas la
            couverture du mois.
          </p>
          <dl className="analysis-definition">
            <div>
              <dt>Crédits ordinaires observés</dt>
              <dd>{analysisMoney(report.income)}</dd>
            </div>
            <div>
              <dt>Épargne normalisée sortante</dt>
              <dd>{analysisMoney(report.savings)}</dd>
            </div>
            <div>
              <dt>Autres transferts sortants</dt>
              <dd>{analysisMoney(report.transfer_out)}</dd>
            </div>
          </dl>
          {report.uncategorized_count > 0 && (
            <p>
              <button
                className="link"
                onClick={() => source({ domain: 'none', type: 'debit' })}
              >
                {report.uncategorized_count} dépense(s) sans catégorie ·{' '}
                {analysisMoney(report.uncategorized_spent)}
              </button>{' '}
              : leur affectation peut améliorer l’explication des sorties.
            </p>
          )}
          {report.unattributed_count > 0 && (
            <p>
              {report.unattributed_count} opération(s) sans compte et relevé
              d’origine vérifiables.
            </p>
          )}
          <button
            className="btn-secondary"
            onClick={() => onExplore('habitudes', 'comparaison')}
          >
            Examiner les évolutions →
          </button>
        </section>
      </div>
      <Narrative month={report.month} onNavigate={onNavigate} />
    </>
  )
}
function CategoryDetail({ month, kind, onNavigate }) {
  const categories = useAnalysisResource(
    (signal) => api.categories({ signal }),
    [],
  )
  return (
    <AnalysisState resource={categories}>
      {(cats) =>
        kind === 'categories' ? (
          <CategoryChart month={month} categories={cats} />
        ) : (
          <TagAnalysis
            month={month}
            categories={cats}
            onNavigate={onNavigate}
          />
        )
      }
    </AnalysisState>
  )
}
function MonthContent({ month, family, section, onNavigate, onExplore }) {
  const resource = useAnalysisResource(
    async (signal) =>
      analysisNatureModel(await api.intelNature(month, { signal }), month),
    [month],
  )
  return (
    <>
      <AnalysisState resource={resource}>
        {(report) => (
          <>
            <div className="analysis-context">
              <span className="analysis-status">
                {analysisPeriodLabel(report.period)}
              </span>
              <span>
                Interrogé du {analysisDate(report.query_from)} au{' '}
                {analysisDate(report.query_to)} · {report.transaction_count}{' '}
                opération(s)
              </span>
            </div>
            <details className="analysis-coverage">
              <summary>Comprendre la couverture des données</summary>
              <p>
                La complétude nécessite des relevés validés couvrant tous les
                comptes connus. Les montants observés ne sont ni des prévisions
                ni un solde bancaire.
              </p>
              {report.coverage?.accounts?.length ? (
                <ul>
                  {report.coverage.accounts.map((a) => (
                    <li key={a.id}>
                      {a.name} :{' '}
                      {a.status === 'documented'
                        ? 'couverture documentée'
                        : 'couverture à vérifier'}
                      {a.gaps?.map((g) => (
                        <span key={g.from}>
                          {' '}
                          · sans preuve du {analysisDate(g.from)} au{' '}
                          {analysisDate(g.to)}
                        </span>
                      ))}
                    </li>
                  ))}
                </ul>
              ) : (
                <p>Aucune couverture documentaire établie.</p>
              )}
            </details>
            {family === 'synthese' && (
              <Summary
                report={report}
                onNavigate={onNavigate}
                onExplore={onExplore}
              />
            )}
            {family === 'flux' && section === 'nature' && (
              <NatureBreakdown report={report} onNavigate={onNavigate} />
            )}
          </>
        )}
      </AnalysisState>
      {family === 'flux' && section === 'categories' && (
        <CategoryDetail month={month} kind="categories" />
      )}
      {family === 'flux' && section === 'marchands' && (
        <MerchantsView month={month} onNavigate={onNavigate} />
      )}
      {family === 'habitudes' && section === 'comparaison' && (
        <CategoryComparison month={month} onNavigate={onNavigate} />
      )}
      {family === 'habitudes' && section === 'reperage' && (
        <Discoveries month={month} onNavigate={onNavigate} />
      )}
      {family === 'pilotage' && section === 'projets' && (
        <TagFolders month={month} onNavigate={onNavigate} />
      )}
      {family === 'pilotage' && section === 'tags' && (
        <CategoryDetail month={month} kind="tags" onNavigate={onNavigate} />
      )}
      {family === 'pilotage' && section === 'gerer-tags' && (
        <section className="analysis-panel">
          <p>
            Outil global : le renommage, la fusion et la suppression
            s’appliquent à tout l’historique, indépendamment du mois
            sélectionné.
          </p>
          <TagManager />
        </section>
      )}
    </>
  )
}
export default function Analyse({ onNavigate, initialMonth }) {
  const [month, setMonth] = useState(initialMonth || analysisMonth())
  const [family, setFamily] = useState('synthese')
  const [section, setSection] = useState('nature')
  const explore = (next, detail) => {
    setFamily(next)
    setSection(detail || DETAILS[next]?.[0]?.[0] || 'nature')
  }
  const changeMonth = (value) => {
    try {
      const valid = analysisMonth(0, value)
      if (valid <= analysisMonth() && valid >= '0001-01') setMonth(valid)
    } catch {
      /* native month entry may be incomplete */
    }
  }
  return (
    <div className="analysis-page">
      <header className="analysis-header">
        <div>
          <p className="analysis-eyebrow">COMPRENDRE SES FINANCES</p>
          <h1>Analyse</h1>
          <p>
            Explorer les causes, retrouver les sources, interpréter avec le bon
            contexte.
          </p>
        </div>
        <div className="analysis-month">
          <button
            className="btn-secondary"
            aria-label="Mois précédent"
            onClick={() => changeMonth(analysisMonth(-1, month))}
          >
            ←
          </button>
          <label>
            Mois analysé
            <input
              type="month"
              value={month}
              min="0001-01"
              max={analysisMonth()}
              onChange={(e) => changeMonth(e.target.value)}
            />
          </label>
          <button
            className="btn-secondary"
            aria-label="Mois suivant"
            disabled={month >= analysisMonth()}
            onClick={() => changeMonth(analysisMonth(1, month))}
          >
            →
          </button>
        </div>
      </header>
      <nav className="analysis-families" aria-label="Familles d’analyse">
        {FAMILIES.map(([id, label]) => (
          <button
            key={id}
            aria-current={family === id ? 'page' : undefined}
            onClick={() => explore(id)}
          >
            {label}
          </button>
        ))}
      </nav>
      {DETAILS[family] && (
        <nav className="analysis-details-nav" aria-label="Analyses détaillées">
          {DETAILS[family].map(([id, label]) => (
            <button
              key={id}
              aria-current={section === id ? 'page' : undefined}
              onClick={() => setSection(id)}
            >
              {label}
            </button>
          ))}
        </nav>
      )}
      <MonthContent
        key={month}
        month={month}
        family={family}
        section={section}
        onNavigate={onNavigate}
        onExplore={explore}
      />
    </div>
  )
}
