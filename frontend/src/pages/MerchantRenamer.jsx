import { useState, useEffect } from 'react'
import { api } from '../api'

// Panneau « Noms des marchands » : présente les libellés que le pipeline
// automatique n'a pas su nommer (les plus fréquents d'abord), permet de les
// renommer à la main, et — si l'IA est configurée — de demander des
// suggestions. L'IA propose ; c'est le clic de l'utilisateur qui applique.
export default function MerchantRenamer({ aiEnabled, onRenamed }) {
  const [items, setItems] = useState(null)
  const [drafts, setDrafts] = useState({})       // key -> nom en cours de saisie
  const [suggests, setSuggests] = useState({})   // key -> nom suggéré par l'IA
  const [busy, setBusy] = useState(false)
  const [patterns, setPatterns] = useState({})   // key -> renommer toutes les variantes
  const [msg, setMsg] = useState(null)
  const [aiRaw, setAiRaw] = useState(null)   // aperçu réponse brute IA (diagnostic)

  const load = () => api.unrecognizedMerchants().then(setItems).catch(() => setItems([]))
  useEffect(() => { load() }, [])

  // Pour un renommage « par motif », on dérive une clé stable = les 2 premiers
  // mots significatifs du libellé (« Acheel France Ref-927b… » → « Acheel
  // France »), ce qui couvre toutes les variantes à référence changeante.
  const patternKeyOf = (key) => key.split(/\s+/).filter((w) => w.length > 1).slice(0, 2).join(' ')

  const save = async (key) => {
    const name = (drafts[key] ?? suggests[key] ?? '').trim()
    if (!name) return
    const asPattern = !!patterns[key]
    const matchKey = asPattern ? patternKeyOf(key) : key
    try {
      const r = await api.setMerchantAlias(matchKey, name, asPattern)
      setMsg(asPattern
        ? `« ${name} » appliqué à ${r.updated} transaction(s) (toutes les variantes de « ${matchKey} »).`
        : `« ${name} » appliqué à ${r.updated} transaction(s).`)
      setItems((list) => (list || []).filter((x) => x.key !== key))
      onRenamed?.()
    } catch (e) { setMsg(e.message) }
  }

  const askAI = async () => {
    if (!items || items.length === 0) return
    setBusy(true); setMsg(null)
    try {
      const keys = items.slice(0, 20).map((x) => x.key)
      const r = await api.suggestMerchantNames(keys)
      const map = {}
      for (const s of r.suggestions || []) if (s.key && s.name) map[s.key] = s.name
      setSuggests(map)
      if (r.ai_error) {
        // L'IA a échoué, mais on a peut-être un repli (noms nettoyés).
        const extra = Object.keys(map).length > 0
          ? ` ${Object.keys(map).length} nom(s) nettoyé(s) proposé(s) à la place.`
          : ''
        setMsg(r.ai_error + extra)
        setAiRaw(r.ai_raw || null) // aperçu de la réponse brute, pour diagnostic
        return
      }
      setAiRaw(null)
      setMsg(`${Object.keys(map).length} suggestion(s) proposée(s) — vérifie puis applique.`)
    } catch (e) { setMsg(e.message) }
    finally { setBusy(false) }
  }

  if (!items) return null
  if (items.length === 0) {
    return (
      <div className="renamer">
        <div className="section-head"><h2>Noms des marchands</h2></div>
        <div className="notice ok">Tous tes marchands portent un nom lisible. Rien à renommer.</div>
      </div>
    )
  }

  const eur = (n) => new Intl.NumberFormat('fr-FR', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(n)

  return (
    <div className="renamer">
      <div className="section-head">
        <h2>Noms des marchands</h2>
        <span className="section-sub">libellés que thémis n'a pas su nommer — renomme-les une fois, ça s'applique partout</span>
      </div>
      {aiEnabled && (
        <div className="toolbar">
          <button className="btn-secondary" onClick={askAI} disabled={busy}>
            {busy && <span className="spinner" />} Proposer des noms avec l'IA
          </button>
          <span className="muted" style={{ fontSize: 12 }}>l'IA suggère, tu valides d'un clic</span>
        </div>
      )}
      {msg && <div className="notice">{msg}</div>}
      {aiRaw && (
        <details className="ai-raw">
          <summary>Voir ce que l'IA a répondu (diagnostic)</summary>
          <pre>{aiRaw}</pre>
        </details>
      )}
      <ul className="renamer-list">
        {items.map((it) => (
          <li key={it.key} className="renamer-item">
            <div className="renamer-raw">
              <span className="renamer-key">{it.display || it.key}</span>
              <span className="renamer-meta">{it.count} tx · {eur(it.spent)}</span>
              <span className="renamer-orig" title="libellé bancaire brut">{it.key}</span>
            </div>
            <div className="renamer-edit">
              <input
                type="text"
                placeholder={suggests[it.key] || 'nouveau nom…'}
                value={drafts[it.key] ?? ''}
                onChange={(e) => setDrafts((d) => ({ ...d, [it.key]: e.target.value }))}
                onKeyDown={(e) => { if (e.key === 'Enter') save(it.key) }}
              />
              {suggests[it.key] && !drafts[it.key] && (
                <button className="renamer-sugg" onClick={() => setDrafts((d) => ({ ...d, [it.key]: suggests[it.key] }))}>
                  ✨ {suggests[it.key]}
                </button>
              )}
              <label className="renamer-pattern" title="Applique le même nom à toutes les transactions de ce marchand, même avec des références différentes">
                <input
                  type="checkbox"
                  checked={!!patterns[it.key]}
                  onChange={(e) => setPatterns((p) => ({ ...p, [it.key]: e.target.checked }))}
                />
                toutes les variantes
              </label>
              <button className="btn-secondary" onClick={() => save(it.key)}>Renommer</button>
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}
