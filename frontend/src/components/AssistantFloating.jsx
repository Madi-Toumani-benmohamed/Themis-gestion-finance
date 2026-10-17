import { useState, useEffect, useRef } from 'react'
import { api } from '../api'
import './AssistantFloating.css'

export default function AssistantFloating({ aiEnabled }) {
  const [open, setOpen] = useState(false)
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [convId, setConvId] = useState(null)
  const bottomRef = useRef(null)

  useEffect(() => {
    if (open) bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, open])

  // Fermer avec Escape
  useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') setOpen(false) }
    if (open) window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open])

  const suggestions = [
    'Combien ai-je dépensé ce mois-ci ?',
    'Compare avec le mois dernier.',
    'Mes principales catégories ?',
    'État de mes données ?',
  ]

  const send = async (text) => {
    if (!text.trim() || loading) return
    const userMsg = { role: 'user', content: text }
    setMessages((m) => [...m, userMsg])
    setInput('')
    setLoading(true)
    try {
      const res = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'same-origin',
        body: JSON.stringify({ conversation_id: convId, messages: [...messages, userMsg] }),
      })
      if (!res.ok) { const err = await res.json().catch(() => ({})); throw new Error(err.error || `Erreur ${res.status}`) }
      const data = await res.json()
      if (data.conversation_id) setConvId(data.conversation_id)
      setMessages((m) => [...m, { role: 'assistant', content: data.answer, evidence: data.evidence, status: data.status }])
    } catch (e) {
      setMessages((m) => [...m, { role: 'assistant', content: `Erreur : ${e.message}`, error: true }])
    } finally { setLoading(false) }
  }

  // Pas d'IA configurée → bouton visible mais inactif avec tooltip
  if (!aiEnabled) return null

  return (
    <>
      {/* Bouton flottant */}
      <button
        className={`assistant-float-btn ${open ? 'open' : ''}`}
        onClick={() => setOpen((v) => !v)}
        aria-label="Ouvrir l'assistant"
        title="Assistant thémis"
      >
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
      </button>

      {/* Panneau */}
      {open && (
        <div className="assistant-panel" onClick={(e) => e.stopPropagation()}>
          <div className="assistant-panel-head">
            <span>Assistant thémis</span>
            <button className="assistant-panel-close" onClick={() => setOpen(false)} aria-label="Fermer">✕</button>
          </div>
          <div className="assistant-panel-body">
            <div className="assistant-panel-messages">
              {messages.length === 0 ? (
                <div className="assistant-panel-suggestions">
                  <p className="muted" style={{ fontSize: 12, marginBottom: 8 }}>Exemples :</p>
                  {suggestions.map((s, i) => (
                    <button key={i} className="assistant-panel-sug" onClick={() => send(s)}>{s}</button>
                  ))}
                </div>
              ) : (
                messages.map((msg, i) => (
                  <div key={i} className={`assistant-panel-msg assistant-panel-msg-${msg.role}${msg.error ? ' error' : ''}`}>
                    <div className="assistant-panel-msg-avatar">{msg.role === 'user' ? 'V' : 'T'}</div>
                    <div className="assistant-panel-msg-body">
                      <div className="assistant-panel-msg-text">{msg.content}</div>
                      {msg.evidence && msg.evidence.length > 0 && (
                        <details className="assistant-panel-evidence">
                          <summary>Détails</summary>
                          {msg.evidence.map((ev, j) => (
                            <div key={j} className="assistant-panel-ev-item">
                              <strong>{ev.key || ev.type || 'données'}</strong>
                              {ev.value_a != null && <span>{ev.value_a} €</span>}
                              {ev.value_b != null && <span>{ev.value_b} €</span>}
                              {ev.absolute_change != null && <span>Δ {ev.absolute_change > 0 ? '+' : ''}{ev.absolute_change} €</span>}
                              {ev.relative_change_percent != null && <span>({ev.relative_change_percent > 0 ? '+' : ''}{ev.relative_change_percent}%)</span>}
                            </div>
                          ))}
                        </details>
                      )}
                    </div>
                  </div>
                ))
              )}
              {loading && (
                <div className="assistant-panel-msg assistant-panel-msg-assistant">
                  <div className="assistant-panel-msg-avatar">T</div>
                  <div className="assistant-panel-msg-body">
                    <span className="spinner" />Réflexion…
                  </div>
                </div>
              )}
              <div ref={bottomRef} />
            </div>
            <form className="assistant-panel-input" onSubmit={(e) => { e.preventDefault(); send(input) }}>
              <input
                className="assistant-panel-field"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Votre question…"
                disabled={loading}
                aria-label="Votre question"
              />
              <button type="submit" disabled={loading || !input.trim()} className="btn assistant-panel-send">
                {loading ? '…' : '→'}
              </button>
            </form>
          </div>
        </div>
      )}
    </>
  )
}
