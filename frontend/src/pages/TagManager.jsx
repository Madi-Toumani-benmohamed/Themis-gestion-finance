import { useState, useEffect, useRef } from 'react'
import { api } from '../api'

// TagManager — renommer ou supprimer les tags existants. Le renommage fusionne
// automatiquement si le nom cible existe déjà (les transactions se regroupent).
export default function TagManager() {
  const [tags, setTags] = useState(null)
  const activeEdit = useRef(null)
  const [editing, setEditing] = useState(null) // nom en cours d'édition
  const [draft, setDraft] = useState('')
  const [confirmDel, setConfirmDel] = useState(null)
  const [msg, setMsg] = useState(null)
  const [error, setError] = useState(null),
    [busy, setBusy] = useState(false)

  const load = () => {
    setError(null)
    setTags(null)
    return api
      .tags()
      .then(setTags)
      .catch((error) => setError(error.message))
  }
  useEffect(() => {
    load()
  }, [])

  const startEdit = (t) => {
    activeEdit.current = t.name
    setEditing(t.name)
    setDraft(t.name)
    setMsg(null)
  }

  const save = async (oldName) => {
    if (activeEdit.current !== oldName) return // déjà traité (évite le double Enter+blur)
    const neo = draft.trim()
    activeEdit.current = null
    setEditing(null)
    if (!neo || neo === oldName) return
    setBusy(true)
    try {
      const r = await api.renameTag(oldName, neo)
      setMsg(
        r.merged
          ? `« ${oldName} » fusionné avec « ${neo} ».`
          : `« ${oldName} » renommé en « ${neo} ».`,
      )
      await load()
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }

  const remove = async (name) => {
    setBusy(true)
    try {
      await api.deleteTag(name)
      setMsg(`Tag « ${name} » supprimé.`)
      setConfirmDel(null)
      await load()
    } catch (e) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }

  if (error)
    return (
      <div role="alert">
        {error}{' '}
        <button className="btn-secondary" onClick={load}>
          Réessayer
        </button>
      </div>
    )
  if (!tags) return <p role="status">Chargement des tags…</p>
  if (tags.length === 0) {
    return (
      <div className="tagmgr" aria-busy={busy}>
        <div className="section-head">
          <h2>Gérer les tags</h2>
        </div>
        <p className="intel-note">
          Aucun tag pour l'instant. Ajoute des tags à tes transactions pour les
          retrouver ici.
        </p>
      </div>
    )
  }

  return (
    <div className="tagmgr" aria-busy={busy}>
      <div className="section-head">
        <h2>Gérer les tags</h2>
        <span className="section-sub">
          renommer fusionne les doublons ; supprimer retire le tag des
          transactions
        </span>
      </div>
      {msg && (
        <div className="notice" role="status">
          {msg}
        </div>
      )}
      <div className="card-grid card-grid-sm">
        {tags.map((t) => (
          <div key={t.name} className="entity-card tag-card">
            <div className="entity-card-head">
              <span className="entity-icon tag-icon" aria-hidden="true">
                #
              </span>
              <span className="entity-meta">{t.count} tx</span>
            </div>
            {editing === t.name ? (
              <input
                className="tagmgr-input"
                aria-label={`Nouveau nom de ${t.name}`}
                value={draft}
                autoFocus
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') save(t.name)
                  if (e.key === 'Escape') {
                    activeEdit.current = null
                    setEditing(null)
                  }
                }}
                onBlur={() => save(t.name)}
              />
            ) : (
              <div className="entity-name">{t.name}</div>
            )}
            {confirmDel === t.name ? (
              <div className="entity-actions-row">
                <span className="tagmgr-confirm">Supprimer ?</span>
                <button
                  disabled={busy}
                  className="link danger"
                  onClick={() => remove(t.name)}
                >
                  oui
                </button>
                <button
                  disabled={busy}
                  className="link"
                  onClick={() => setConfirmDel(null)}
                >
                  non
                </button>
              </div>
            ) : (
              editing !== t.name && (
                <div className="entity-actions-row">
                  <button
                    disabled={busy}
                    className="link"
                    onClick={() => startEdit(t)}
                  >
                    renommer
                  </button>
                  <button
                    disabled={busy}
                    className="link danger"
                    onClick={() => {
                      setConfirmDel(t.name)
                      setMsg(null)
                    }}
                  >
                    supprimer
                  </button>
                </div>
              )
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
