import {
  knowledgeDate,
  knowledgeState,
  knowledgeError,
  readKnowledgeTrace,
} from './knowledgeModel.mjs'
import './KnowledgeActivity.css'
export function KnowledgeFacts({ facts = [], sources = [] }) {
  return (
    <ul className="kg-facts">
      {facts.map((f, index) => {
        const provenance = f.sources?.length
          ? f.sources
          : sources.filter((s) => f.episode_ids?.includes(s.id))
        return (
          <li key={f.id || index}>
            <p>
              {f.fact}
              {f.truncated && <small> (extrait tronqué)</small>}
            </p>
            <small>
              Relation : {f.type || 'Non renseignée'} · Enregistrée :{' '}
              {knowledgeDate(f.created_at)}
            </small>
            {(f.invalid_at || f.expired_at) && (
              <p className="kg-warning">
                Relation historique ou invalidée ; vérifier la période avant
                utilisation.
              </p>
            )}
            {f.valid_at && (
              <p>
                <small>
                  Valide à partir du {knowledgeDate(f.valid_at)}
                  {f.invalid_at && ` jusqu’au ${knowledgeDate(f.invalid_at)}`}
                </small>
              </p>
            )}
            {provenance.length > 0 ? (
              <ul>
                {provenance.map((s) => (
                  <li key={s.id}>
                    <small>
                      {/^statement:[1-9]\d*$/.test(s.source)
                        ? `Relevé #${s.source.split(':')[1]}`
                        : s.source || 'Source non renseignée'}{' '}
                      · {s.name} · {knowledgeDate(s.created_at)}
                    </small>
                  </li>
                ))}
              </ul>
            ) : (
              <small>
                Provenance détaillée non disponible
                {f.episode_ids?.length
                  ? ` · ${f.episode_ids.length} épisode(s) référencé(s)`
                  : ''}
                .
              </small>
            )}
          </li>
        )
      })}
    </ul>
  )
}
export function KnowledgeTrace({ trace }) {
  if (!trace) return null
  let data
  try {
    data = readKnowledgeTrace(trace)
  } catch {
    return <p className="kg-trace">Trace mémoire indisponible.</p>
  }
  return (
    <details className="kg-trace">
      <summary>
        {data.context_included
          ? `${data.facts.length} fait(s) de mémoire joints à la requête IA`
          : `Mémoire : ${knowledgeState(data.status)}`}
      </summary>
      <p>
        Ces faits sont un contexte indicatif. Leur présence dans la requête ne
        prouve pas que le modèle les a utilisés ; les montants doivent être
        vérifiés dans les données bancaires.
      </p>
      {data.error_code && <p>{knowledgeError(data.error_code)}</p>}
      <KnowledgeFacts facts={data.facts} />
      {data.audit_available === false && (
        <p>
          Journal persistant indisponible pour cette opération ; cette trace
          décrit uniquement la réponse courante.
        </p>
      )}
    </details>
  )
}
