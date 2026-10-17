import { useState, useEffect, useRef } from 'react'
import { api } from '../api'
import '../components/Assistant.css'

export default function Assistant() {
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [conversationId, setConversationId] = useState(null)
  const [aiEnabled, setAiEnabled] = useState(false)
  const bottomRef = useRef(null)

  useEffect(() => {
    api.health().then((h) => setAiEnabled(h.ai_enabled)).catch(() => {})
  }, [])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const suggestions = [
    'Combien ai-je dépensé ce mois-ci ?',
    'Compare mes dépenses avec le mois dernier.',
    'Quelles sont mes principales catégories de dépenses ?',
    'Quelles données manquent pour analyser mes finances ?',
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
        body: JSON.stringify({
          conversation_id: conversationId,
          messages: [...messages, userMsg],
        }),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error || `Erreur ${res.status}`)
      }
      const data = await res.json()
      if (data.conversation_id) setConversationId(data.conversation_id)
      setMessages((m) => [...m, {
        role: 'assistant',
        content: data.answer,
        evidence: data.evidence,
        status: data.status,
      }])
    } catch (e) {
      setMessages((m) => [...m, { role: 'assistant', content: `Erreur : ${e.message}`, error: true }])
    } finally {
      setLoading(false)
    }
  }

  if (!aiEnabled) {
    return (
      <div className="assistant">
        <div className="section-head"><h2>Assistant</h2></div>
        <p className="muted" style={{ marginTop: -12, marginBottom: 24, fontSize: 13.5 }}>
          L'assistant nécessite une IA configurée. Rendez-vous dans l'onglet Admin pour activer Ollama ou Anthropic.
        </p>
        <div className="assistant-empty">
          <p>Configurez votre fournisseur d'IA dans <strong>Admin</strong> pour commencer.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="assistant">
      <div className="section-head"><h2>Assistant</h2></div>
      <p className="muted" style={{ marginTop: -12, marginBottom: 20, fontSize: 13.5 }}>
        Posez vos questions sur vos finances. Les réponses sont calculées à partir de vos données réelles.
      </p>

      <div className="assistant-messages">
        {messages.length === 0 ? (
          <div className="assistant-suggestions">
            <p className="muted" style={{ marginBottom: 12 }}>Exemples de questions :</p>
            {suggestions.map((s, i) => (
              <button key={i} className="assistant-suggestion" onClick={() => send(s)}>{s}</button>
            ))}
          </div>
        ) : (
          messages.map((msg, i) => (
            <div key={i} className={`assistant-msg assistant-msg-${msg.role}${msg.error ? ' error' : ''}`}>
              <div className="assistant-msg-avatar">{msg.role === 'user' ? 'V' : 'T'}</div>
              <div className="assistant-msg-body">
                <div className="assistant-msg-text">{msg.content}</div>
                {msg.evidence && msg.evidence.length > 0 && (
                  <details className="assistant-evidence">
                    <summary>Voir les données sources</summary>
                    {msg.evidence.map((ev, j) => (
                      <div key={j} className="assistant-evidence-item">
                        <strong>{ev.type || ev.key || 'données'}</strong>
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
          <div className="assistant-msg assistant-msg-assistant">
            <div className="assistant-msg-avatar">T</div>
            <div className="assistant-msg-body">
              <span className="spinner" />Réflexion en cours…
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <form className="assistant-input" onSubmit={(e) => { e.preventDefault(); send(input) }}>
        <input
          className="assistant-input-field"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Votre question…"
          disabled={loading}
          aria-label="Votre question"
        />
        <button type="submit" disabled={loading || !input.trim()} className="btn assistant-send">
          {loading ? '…' : 'Envoyer'}
        </button>
      </form>
    </div>
  )
}
