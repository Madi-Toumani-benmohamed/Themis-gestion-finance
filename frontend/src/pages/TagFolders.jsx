import { useState, useEffect } from 'react'
import { api } from '../api'

const fr0 = (n) => new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n || 0)
const frD = (d) => d ? new Date(d).toLocaleDateString('fr-FR', { day: 'numeric', month: 'short', year: 'numeric' }) : ''

// Détail dépliable d'un dossier : total, durée, répartition catégorie, marchands.
function FolderDetail({ name }) {
  const [d, setD] = useState(null)
  useEffect(() => { api.tagFolderDetail(name).then(setD).catch(() => setD(null)) }, [name])
  if (!d) return <div className="folder-detail-loading">…</div>
  const max = Math.max(...(d.by_category || []).map((c) => c.total), 1)
  return (
    <div className="folder-detail">
      <div className="folder-stats">
        <div><span className="fs-val">{fr0(d.total)}</span><span className="fs-lbl">total</span></div>
        <div><span className="fs-val">{d.count}</span><span className="fs-lbl">dépenses</span></div>
        <div><span className="fs-val">{d.days} j</span><span className="fs-lbl">{frD(d.first_date)} → {frD(d.last_date)}</span></div>
      </div>
      {d.by_category?.length > 0 && (
        <div className="folder-block">
          <div className="folder-block-title">Répartition</div>
          {d.by_category.map((c) => (
            <div key={c.category} className="fcat-row">
              <span className="fcat-name">{c.category}</span>
              <div className="fcat-track"><div className="fcat-bar" style={{ width: `${(c.total / max) * 100}%` }} /></div>
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
              <span key={m.name} className="fmerch">{m.name} <b>{fr0(m.total)}</b></span>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default function TagFolders({ onNavigate }) {
  const [data, setData] = useState(null)
  const [open, setOpen] = useState(null)
  const [showOthers, setShowOthers] = useState(false)

  const load = () => api.tagFolders().then(setData).catch(() => setData(null))
  useEffect(() => { load() }, [])

  const setPref = async (name, state, e) => {
    e.stopPropagation()
    await api.setTagFolderPref(name, state).catch(() => {})
    load()
  }

  if (!data) return null
  const { folders = [], others = [] } = data
  if (folders.length === 0 && others.length === 0) return null

  return (
    <div className="tagfolders">
      <div className="section-head">
        <h3>Budgets projet</h3>
        <span className="section-sub">tes tags qui ressemblent à des projets — total, durée, détail</span>
      </div>

      {folders.length === 0 && (
        <p className="intel-note">Aucun projet détecté pour l'instant. Un tag concentré dans le temps avec un montant notable apparaîtra ici automatiquement.</p>
      )}

      <div className="folder-grid">
        {folders.map((f) => (
          <div key={f.name} className={`folder-card ${open === f.name ? 'open' : ''}`}>
            <div className="folder-head" onClick={() => setOpen(open === f.name ? null : f.name)}>
              <div className="folder-id">
                <span className="folder-name">{f.name}{f.pinned && <span className="folder-pin" title="épinglé">📌</span>}</span>
                <span className="folder-sub">{f.count} dépenses · {f.days} jours</span>
              </div>
              <span className="folder-total">{fr0(f.total)}</span>
            </div>
            {open === f.name && (
              <>
                <FolderDetail name={f.name} />
                <div className="folder-actions">
                  <button className="link" onClick={(e) => { e.stopPropagation(); onNavigate?.('transactions', { q: f.name }) }}>voir les transactions →</button>
                  {f.pinned
                    ? <button className="link" onClick={(e) => setPref(f.name, 'auto', e)}>ne plus épingler</button>
                    : <button className="link" onClick={(e) => setPref(f.name, 'hidden', e)}>ce n'est pas un projet</button>}
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
                  <span>{o.name}<span className="fo-count">{o.count} · {fr0(o.total)}</span></span>
                  <button className="link" onClick={(e) => setPref(o.name, 'pinned', e)}>suivre comme un projet</button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
