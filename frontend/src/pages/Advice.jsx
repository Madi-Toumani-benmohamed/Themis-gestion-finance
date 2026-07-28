import { useEffect, useState } from 'react'
import { api, monthLabel } from '../api'

// rendu markdown minimal (###, listes, gras) — pas de dépendance externe
function renderMarkdown(md) {
  const lines = md.split('\n')
  const out = []
  let list = null
  const flush = () => { if (list) { out.push(<ul key={out.length}>{list}</ul>); list = null } }
  const inline = (s) =>
    s.split(/(\*\*[^*]+\*\*)/g).map((part, i) =>
      part.startsWith('**') ? <b key={i}>{part.slice(2, -2)}</b> : part)

  lines.forEach((line, i) => {
    const t = line.trim()
    if (t.startsWith('###')) { flush(); out.push(<h3 key={i}>{t.replace(/^#+\s*/, '')}</h3>) }
    else if (t.startsWith('- ') || t.startsWith('* ')) { (list ??= []).push(<li key={i}>{inline(t.slice(2))}</li>) }
    else if (t === '') { flush() }
    else { flush(); out.push(<p key={i}>{inline(t)}</p>) }
  })
  flush()
  return out
}


// Cohérence du tri : audit déterministe (règles vs transactions réelles) +
// relecture IA optionnelle. La partie déterministe fonctionne sans IA
// configurée : elle est fiable et vérifiable ; l'IA complète pour les
// incohérences plus subtiles.
function CoherenceSection({ aiEnabled, onNavigate }) {
  const [report, setReport] = useState(null)
  const [busy, setBusy] = useState(false)
  const [cErr, setCErr] = useState(null)

  const run = async (withAI) => {
    setBusy(true); setCErr(null)
    try { setReport(await api.coherence(withAI)) }
    catch (e) { setCErr(e.message) }
    finally { setBusy(false) }
  }

  return (
    <div className="coherence">
      <div className="section-head">
        <h2>Cohérence du tri</h2>
        <span className="section-sub">règles vs catégories et tags réellement posés</span>
      </div>
      <div className="toolbar">
        <button className="btn-secondary" onClick={() => run(false)} disabled={busy}>
          {busy && <span className="spinner" />} Vérifier
        </button>
        {aiEnabled && (
          <button className="btn" onClick={() => run(true)} disabled={busy}
                  title="Audit déterministe + relecture par l'IA (libellés envoyés au modèle configuré)">
            {busy && <span className="spinner" />} Vérifier avec l'IA
          </button>
        )}
      </div>
      {cErr && <div className="notice err">{cErr}</div>}
      {report && (
        <div className="coherence-report">
          {(report.findings ?? []).length === 0 ? (
            <div className="notice ok">
              Aucune incohérence détectée : les {report.rules_checked} règle(s) et les transactions sont alignées.
            </div>
          ) : (
            <ul className="coherence-list">
              {report.findings.map((f, i) => (
                <li key={i} className={`coherence-item ${f.kind}`}>
                  <div className="coherence-detail">{f.detail}</div>
                  {f.examples?.length > 0 && (
                    <div className="coherence-examples">
                      {f.examples.map((ex, j) => (
                        <button
                          key={j}
                          className="coherence-ex"
                          onClick={() => onNavigate?.('transactions', { q: ex.display })}
                          title="Voir cette transaction"
                        >
                          <span className="coherence-ex-name">{ex.display}</span>
                          <span className="coherence-ex-cat">{ex.category}</span>
                          {ex.tags?.length > 0 ? (
                            <span className="coherence-ex-tags">
                              {ex.tags.map((t) => <span key={t} className="mini-tag">{t}</span>)}
                            </span>
                          ) : (
                            <span className="coherence-ex-notags">sans tag</span>
                          )}
                        </button>
                      ))}
                    </div>
                  )}
                  {f.query && (
                    <button className="coherence-goto" onClick={() => onNavigate?.('transactions', { q: f.query })}>
                      Voir les {f.count} transaction{f.count > 1 ? 's' : ''} →
                    </button>
                  )}
                </li>
              ))}
            </ul>
          )}
          {report.ai_error && <div className="notice warn">Relecture IA indisponible : {report.ai_error}</div>}
          {report.ai && <div className="advice coherence-ai">{renderMarkdown(report.ai)}</div>}
        </div>
      )}
    </div>
  )
}

export default function Advice({ aiEnabled, onNavigate }) {
  const [months, setMonths] = useState([])
  const [month, setMonth] = useState('')
  const [analysis, setAnalysis] = useState(null)
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState(null)

  useEffect(() => {
    api.summary(24).then((d) => {
      const ms = (d.monthly ?? []).map((m) => m.month).reverse()
      setMonths(ms)
      if (ms[0]) setMonth(ms[0])
    }).catch((e) => setErr(e.message))
  }, [])

  useEffect(() => {
    if (!month) return
    setAnalysis(null)
    setErr(null)
    api.getAnalysis(month).then(setAnalysis).catch(() => {}) // 404 = pas encore générée
  }, [month])

  const generate = async () => {
    setLoading(true)
    setErr(null)
    try {
      const r = await api.createAnalysis(month)
      setAnalysis({ ...r, created_at: new Date().toISOString() })
    } catch (e) {
      setErr(e.message)
    } finally {
      setLoading(false)
    }
  }

  if (!aiEnabled) {
    return (
      <div className="section">
        <div className="empty">
          <p><b>Analyse IA désactivée.</b></p>
          <p>Côté backend, définis <code>OLLAMA_URL</code> pour une IA <b>100&nbsp;% locale</b> (tes
          données bancaires ne quittent pas ta machine), ou <code>ANTHROPIC_API_KEY</code> pour
          l'API Claude. Cela active les conseils mensuels et le parsing de secours par IA.</p>
        </div>
        <CoherenceSection aiEnabled={false} onNavigate={onNavigate} />
      </div>
    )
  }

  return (
    <div className="section">
      <div className="toolbar">
        <select value={month} onChange={(e) => setMonth(e.target.value)} aria-label="Mois à analyser">
          {months.map((m) => <option key={m} value={m}>{monthLabel(m)}</option>)}
        </select>
        <button className="btn" onClick={generate} disabled={loading || !month}>
          {loading && <span className="spinner" />}
          {analysis ? 'Régénérer l\'analyse' : 'Analyser ce mois'}
        </button>
      </div>

      {err && <div className="notice err">{err}</div>}

      {analysis ? (
        <div className="advice">
          {renderMarkdown(analysis.content)}
          <p className="meta">
            Analyse générée le {new Date(analysis.created_at).toLocaleString('fr-FR')} ·
            seuls des agrégats anonymes (totaux par catégorie) sont envoyés à l'API, jamais les libellés.
          </p>
        </div>
      ) : !loading && (
        <div className="empty">
          Pas encore d'analyse pour ce mois. Clique sur «&nbsp;Analyser ce mois&nbsp;» —
          le conseiller compare tes dépenses aux 3 mois précédents et propose des actions concrètes.
        </div>
      )}

      <CoherenceSection aiEnabled={aiEnabled} onNavigate={onNavigate} />
    </div>
  )
}
