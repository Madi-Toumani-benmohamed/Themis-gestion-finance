import { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { api, eur } from './api'

// CommandPalette — la palette « Cmd+K » : navigation, actions et recherche de
// transactions au clavier, sans toucher la souris. S'ouvre partout dans l'app.
export default function CommandPalette({ open, onClose, tabs, onNavigate, onUpload }) {
  const [q, setQ] = useState('')
  const [sel, setSel] = useState(0)
  const [txResults, setTxResults] = useState([])
  const inputRef = useRef()

  // Commandes statiques : navigation + actions.
  const commands = useMemo(() => {
    const nav = tabs.map(([id, label]) => ({
      type: 'nav', id, label, hint: 'Aller à',
    }))
    const actions = [
      { type: 'action', id: 'upload', label: 'Importer un relevé', hint: 'Action' },
    ]
    return [...nav, ...actions]
  }, [tabs])

  // Filtrage flou simple des commandes (sous-séquence insensible à la casse/accents).
  const norm = (s) => s.toLowerCase().normalize('NFD').replace(/[\u0300-\u036f]/g, '')
  const filteredCommands = useMemo(() => {
    if (!q.trim()) return commands
    const nq = norm(q)
    return commands.filter((c) => norm(c.label).includes(nq))
  }, [q, commands])

  // Recherche de transactions (débounce léger) dès 2 caractères.
  useEffect(() => {
    if (q.trim().length < 2) { setTxResults([]); return }
    const t = setTimeout(() => {
      api.transactions({ q: q.trim() })
        .then((rows) => setTxResults(rows.slice(0, 6)))
        .catch(() => setTxResults([]))
    }, 180)
    return () => clearTimeout(t)
  }, [q])

  // Liste unifiée : commandes puis résultats de transactions.
  const items = useMemo(() => {
    const cmd = filteredCommands.map((c) => ({ ...c, _k: `${c.type}:${c.id}` }))
    const tx = txResults.map((t) => ({ type: 'tx', tx: t, _k: `tx:${t.id}` }))
    return [...cmd, ...tx]
  }, [filteredCommands, txResults])

  useEffect(() => { setSel(0) }, [items.length])

  // Focus à l'ouverture, reset à la fermeture.
  useEffect(() => {
    if (open) {
      setQ(''); setSel(0); setTxResults([])
      setTimeout(() => inputRef.current?.focus(), 30)
    }
  }, [open])

  const choose = useCallback((item) => {
    if (!item) return
    if (item.type === 'nav') onNavigate(item.id)
    else if (item.type === 'action' && item.id === 'upload') onUpload()
    else if (item.type === 'tx') onNavigate('transactions', { q: item.tx.label })
    onClose()
  }, [onNavigate, onUpload, onClose])

  const onKey = useCallback((e) => {
    if (e.key === 'Escape') { onClose(); return }
    if (e.key === 'ArrowDown') { e.preventDefault(); setSel((s) => Math.min(s + 1, items.length - 1)) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setSel((s) => Math.max(s - 1, 0)) }
    else if (e.key === 'Enter') { e.preventDefault(); choose(items[sel]) }
  }, [items, sel, choose, onClose])

  if (!open) return null

  return createPortal(
    <div className="cmdk-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div className="cmdk" role="dialog" aria-label="Palette de commande">
        <div className="cmdk-input-row">
          <span className="cmdk-search-icon">⌕</span>
          <input
            ref={inputRef}
            className="cmdk-input"
            placeholder="Chercher une transaction, aller à une page, lancer une action…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={onKey}
          />
          <kbd className="cmdk-esc">Échap</kbd>
        </div>

        <div className="cmdk-results">
          {items.length === 0 ? (
            <div className="cmdk-empty">Aucun résultat pour « {q} »</div>
          ) : (
            items.map((item, i) => (
              <button
                key={item._k}
                className={'cmdk-item' + (i === sel ? ' active' : '')}
                onMouseEnter={() => setSel(i)}
                onClick={() => choose(item)}
              >
                {item.type === 'tx' ? (
                  <>
                    <span className="cmdk-item-icon">€</span>
                    <span className="cmdk-item-label">{item.tx.label}</span>
                    <span className={'cmdk-item-hint amt ' + (item.tx.amount >= 0 ? 'credit' : '')}>
                      {eur(item.tx.amount)}
                    </span>
                  </>
                ) : (
                  <>
                    <span className="cmdk-item-icon">{item.type === 'action' ? '⚡' : '→'}</span>
                    <span className="cmdk-item-label">{item.label}</span>
                    <span className="cmdk-item-hint">{item.hint}</span>
                  </>
                )}
              </button>
            ))
          )}
        </div>

        <div className="cmdk-foot">
          <span><kbd>↑</kbd><kbd>↓</kbd> naviguer</span>
          <span><kbd>↵</kbd> ouvrir</span>
          <span><kbd>Échap</kbd> fermer</span>
        </div>
      </div>
    </div>,
    document.body,
  )
}
