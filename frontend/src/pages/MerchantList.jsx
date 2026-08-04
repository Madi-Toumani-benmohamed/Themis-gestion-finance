import { useEffect, useState } from 'react'
import { api } from '../api'
import { useToast } from '../Toast'

// Consultation et rectification des marchands déjà nommés. Contrairement à
// MerchantRenamer (qui liste les libellés NON reconnus), cette vue montre les
// renommages existants et permet de les corriger si on s'est trompé.
export default function MerchantList() {
  const toast = useToast()
  const [aliases, setAliases] = useState(null)
  const [editing, setEditing] = useState(null)   // id en cours d'édition
  const [draft, setDraft] = useState('')
  const [filter, setFilter] = useState('')

  const load = () => api.merchantAliases()
    .then((a) => setAliases(Array.isArray(a) ? a : []))
    .catch(() => setAliases([]))

  useEffect(() => { load() }, [])

  const startEdit = (a) => { setEditing(a.id); setDraft(a.display) }

  const save = async (a) => {
    const name = draft.trim()
    if (!name || name === a.display) { setEditing(null); return }
    try {
      // setAlias fait un upsert sur match_key : on réutilise la même clé.
      const r = await api.setMerchantAlias(a.match_key, name)
      toast.ok(`Renommé en « ${name} »${r.updated ? ` · ${r.updated} transaction(s)` : ''}.`)
      setEditing(null)
      load()
    } catch (e) { toast.err(e.message) }
  }

  const remove = async (a) => {
    try {
      await api.deleteMerchantAlias(a.id)
      toast.ok('Renommage retiré (les transactions gardent leur nom actuel).')
      load()
    } catch (e) { toast.err(e.message) }
  }

  if (!aliases) return null

  const shown = filter
    ? aliases.filter((a) =>
        a.display.toLowerCase().includes(filter.toLowerCase()) ||
        a.match_key.toLowerCase().includes(filter.toLowerCase()))
    : aliases

  return (
    <div className="renamer">
      <div className="section-head">
        <h2>Marchands nommés</h2>
        <span className="muted">les renommages que tu as faits — corrige-les si besoin</span>
      </div>

      {aliases.length === 0 ? (
        <p className="muted" style={{ fontSize: 13 }}>
          Aucun marchand renommé pour l'instant. Utilise « à nommer » pour donner
          des noms lisibles à tes libellés.
        </p>
      ) : (
        <>
          <input
            className="merchant-filter"
            placeholder="filtrer…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
          <div className="card-grid card-grid-sm">
            {shown.map((a) => (
              <div key={a.id} className="entity-card">
                <div className="entity-card-head">
                  <span className="entity-icon" aria-hidden="true">
                    {(a.display || '?').trim().charAt(0).toUpperCase()}
                  </span>
                </div>
                {editing === a.id ? (
                  <input
                    className="tagmgr-input"
                    value={draft}
                    autoFocus
                    onChange={(e) => setDraft(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') save(a); if (e.key === 'Escape') setEditing(null) }}
                    onBlur={() => save(a)}
                  />
                ) : (
                  <div className="entity-name">{a.display}</div>
                )}
                <div className="entity-sub" title="libellé bancaire d'origine">{a.match_key}</div>
                {editing !== a.id && (
                  <div className="entity-actions-row">
                    <button className="link" onClick={() => startEdit(a)}>corriger</button>
                    <button className="link danger" onClick={() => remove(a)}>retirer</button>
                  </div>
                )}
              </div>
            ))}
          </div>
          {shown.length === 0 && <p className="muted" style={{ fontSize: 13 }}>Aucun marchand ne correspond à « {filter} ».</p>}
        </>
      )}
    </div>
  )
}
