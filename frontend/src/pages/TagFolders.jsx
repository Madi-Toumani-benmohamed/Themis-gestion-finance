import { useState, useEffect } from 'react'
import { api } from '../api'
import {
  analysisMoney,
  analysisMonth,
  analysisDate,
} from '../components/analysisPageModel.mjs'
import { useAnalysisResource } from '../components/analysisResources'

const fr0 = analysisMoney
const frD = analysisDate

// Détail dépliable d'un dossier : total, durée, répartition catégorie, marchands.
function FolderDetail({ name, month }) {
  const resource = useAnalysisResource(
    (signal) => api.tagFolderDetail(name, month, { signal }),
    [name, month],
  )
  const d = resource.data
  if (resource.loading) return <p role="status">Chargement du détail…</p>
  if (resource.error)
    return (
      <p role="alert">
        {resource.error}{' '}
        <button className="link" onClick={resource.retry}>
          Réessayer
        </button>
      </p>
    )
  if (!d) return null
  const max = Math.max(...(d.by_category || []).map((c) => c.total), 1)
  return (
    <div className="folder-detail">
      <div className="folder-stats">
        <div>
          <span className="fs-val">{fr0(d.total)}</span>
          <span className="fs-lbl">total</span>
        </div>
        <div>
          <span className="fs-val">{d.count}</span>
          <span className="fs-lbl">dépenses</span>
        </div>
        <div>
          <span className="fs-val">{d.days} j</span>
          <span className="fs-lbl">
            {frD(d.first_date)} → {frD(d.last_date)}
          </span>
        </div>
      </div>
      {d.by_category?.length > 0 && (
        <div className="folder-block">
          <div className="folder-block-title">Répartition</div>
          {d.by_category.map((c) => (
            <div key={c.category} className="fcat-row">
              <span className="fcat-name">{c.category}</span>
              <div className="fcat-track">
                <div
                  className="fcat-bar"
                  style={{ width: `${(c.total / max) * 100}%` }}
                />
              </div>
              <span className="fcat-val">{fr0(c.total)}</span>
            </div>
          ))}
        </div>
      )}
      {d.by_merchant?.length > 0 && (
        <div className="folder-block">
          <div className="folder-block-title">Principaux marchands</div>
          <div className="fmerch-list">
            {d.by_merchant.map((m) => (
              <span key={m.name} className="fmerch">
                {m.name} <b>{fr0(m.total)}</b>
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default function TagFolders({ onNavigate, month = analysisMonth() }) {
  const resource = useAnalysisResource(
    (signal) => api.tagFolders(month, { signal }),
    [month],
  )
  const data = resource.data
  const [saving, setSaving] = useState(false),
    [saveError, setSaveError] = useState(null)
  const [open, setOpen] = useState(null)
  const [showOthers, setShowOthers] = useState(false)

  const setPref = async (name, state, e) => {
    e.stopPropagation()
    setSaving(true)
    setSaveError(null)
    try {
      await api.setTagFolderPref(name, state)
      resource.retry()
    } catch (error) {
      setSaveError(error.message)
    } finally {
      setSaving(false)
    }
  }

  if (resource.loading)
    return <p role="status">Chargement des projets par tag…</p>
  if (resource.error)
    return (
      <div role="alert">
        {resource.error}{' '}
        <button className="btn-secondary" onClick={resource.retry}>
          Réessayer
        </button>
      </div>
    )
  if (!data) return null
  const { folders = [], others = [] } = data

  return (
    <div className="tagfolders">
      <div className="section-head">
        <h2>Projets par tag</h2>
        <span className="section-sub">
          historique observé jusqu’au {frD(data.query_to)}
        </span>
      </div>

      {folders.length === 0 && (
        <p className="intel-note">
          Aucun projet détecté pour l'instant. Un tag concentré dans le temps
          avec un montant notable apparaîtra ici automatiquement.
        </p>
      )}

      <p className="analysis-footnote">
        Ces regroupements ne sont pas des enveloppes budgétaires. Les totaux
        couvrent l’historique importé jusqu’à la date indiquée, pas seulement{' '}
        {month}. Un projet détecté reste une hypothèse ; les tags peuvent se
        chevaucher.
      </p>
      {saveError && <p role="alert">{saveError}</p>}
      <div className="folder-grid" aria-busy={saving}>
        {folders.map((f) => (
          <div
            key={f.name}
            className={`folder-card ${open === f.name ? 'open' : ''}`}
          >
            <button
              className="folder-head"
              aria-expanded={open === f.name}
              onClick={() => setOpen(open === f.name ? null : f.name)}
            >
              <span className="folder-id">
                <span className="folder-name">
                  {f.name}
                  {f.pinned && (
                    <span className="folder-pin" title="épinglé">
                      📌
                    </span>
                  )}
                </span>
                <span className="folder-sub">
                  {f.count} dépenses ·{' '}
                  {f.pinned ? 'Épinglé' : 'Projet possible'} ·{' '}
                  {fr0(f.selected_month_total)} sur le mois
                </span>
              </span>
              <span className="folder-total">{fr0(f.total)}</span>
            </button>
            {open === f.name && (
              <>
                <FolderDetail name={f.name} month={month} />
                <div className="folder-actions">
                  <button
                    className="link"
                    onClick={(e) => {
                      e.stopPropagation()
                      onNavigate?.('transactions', {
                        tag: f.name,
                        from: f.first_date,
                        to: data.query_to,
                        type: 'debit',
                      })
                    }}
                  >
                    voir les transactions →
                  </button>
                  {f.pinned ? (
                    <button
                      disabled={saving}
                      className="link"
                      onClick={(e) => setPref(f.name, 'auto', e)}
                    >
                      ne plus épingler
                    </button>
                  ) : (
                    <button
                      disabled={saving}
                      className="link"
                      onClick={(e) => setPref(f.name, 'hidden', e)}
                    >
                      ce n'est pas un projet
                    </button>
                  )}
                </div>
              </>
            )}
          </div>
        ))}
      </div>

      {others.length > 0 && (
        <div className="folder-others">
          <button className="link" onClick={() => setShowOthers((v) => !v)}>
            {showOthers ? 'masquer' : `voir les ${others.length} autres tags`}
          </button>
          {showOthers && (
            <ul className="folder-others-list">
              {others.map((o) => (
                <li key={o.name} className="folder-other">
                  <span>
                    {o.name}
                    <span className="fo-count">
                      {o.count} · {fr0(o.total)}
                    </span>
                  </span>
                  <button
                    disabled={saving}
                    className="link"
                    onClick={(e) => setPref(o.name, 'pinned', e)}
                  >
                    suivre comme un projet
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
