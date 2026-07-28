import { useState, useEffect } from 'react'
import { api } from '../api'

// TagManager — renommer ou supprimer les tags existants. Le renommage fusionne
// automatiquement si le nom cible existe déjà (les transactions se regroupent).
export default function TagManager() {
  const [tags, setTags] = useState(null)
  const [editing, setEditing] = useState(null)   // nom en cours d'édition
  const [draft, setDraft] = useState('')
  const [confirmDel, setConfirmDel] = useState(null)
  const [msg, setMsg] = useState(null)

  const load = () => api.tags().then(setTags).catch(() => setTags([]))
  useEffect(() => { load() }, [])

  const startEdit = (t) => { setEditing(t.name); setDraft(t.name); setMsg(null) }

  const save = async (oldName) => {
    if (editing !== oldName) return // déjà traité (évite le double Enter+blur)
    const neo = draft.trim()
    setEditing(null)
    if (!neo || neo === oldName) return
    try {
      const r = await api.renameTag(oldName, neo)
      setMsg(r.merged
        ? `« ${oldName} » fusionné avec « ${neo} ».`
        : `« ${oldName} » renommé en « ${neo} ».`)
      load()
    } catch (e) { setMsg(e.message) }
  }

  const remove = async (name) => {
    try {
      await api.deleteTag(name)
      setMsg(`Tag « ${name} » supprimé.`)
      setConfirmDel(null)
      load()
    } catch (e) { setMsg(e.message) }
  }

  if (!tags) return null
  if (tags.length === 0) {
    return (
      <div className="tagmgr">
        <div className="section-head"><h3>Gérer les tags</h3></div>
        <p className="intel-note">Aucun tag pour l'instant. Ajoute des tags à tes transactions pour les retrouver ici.</p>
      </div>
    )
  }

  return (
    <div className="tagmgr">
      <div className="section-head">
        <h3>Gérer les tags</h3>
        <span className="section-sub">renommer fusionne les doublons ; supprimer retire le tag des transactions</span>
      </div>
      {msg && <div className="notice">{msg}</div>}
      <ul className="tagmgr-list">
        {tags.map((t) => (
          <li key={t.name} className="tagmgr-item">
            {editing === t.name ? (
              <input
                className="tagmgr-input"
                value={draft}
                autoFocus
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') save(t.name); if (e.key === 'Escape') setEditing(null) }}
                onBlur={() => save(t.name)}
              />
            ) : (
              <span className="tagmgr-name">{t.name}<span className="tagmgr-count">{t.count}</span></span>
            )}
            {confirmDel === t.name ? (
              <span className="tagmgr-confirm">
                Supprimer ?
                <button className="link danger" onClick={() => remove(t.name)}>oui</button>
                <button className="link" onClick={() => setConfirmDel(null)}>non</button>
              </span>
            ) : editing !== t.name && (
              <span className="tagmgr-actions">
                <button className="link" onClick={() => startEdit(t)}>renommer</button>
                <button className="link danger" onClick={() => { setConfirmDel(t.name); setMsg(null) }}>supprimer</button>
              </span>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
