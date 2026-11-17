import { useEffect, useState } from 'react'
import { api } from '../api'
import AgentConsole from '../components/agent/AgentConsole'
import '../components/Assistant.css'

export default function Assistant({ pageContext, onNavigate, refreshKey }) {
  const [enabled, setEnabled] = useState(null),
    [error, setError] = useState('')
  useEffect(() => {
    let active = true
    api
      .health()
      .then((h) => {
        if (active) setEnabled(h.ai_enabled)
      })
      .catch(() => {
        if (active) setError('État de l’IA indisponible.')
      })
    return () => {
      active = false
    }
  }, [])
  return (
    <div className="assistant">
      <div className="section-head">
        <h2>Assistant</h2>
      </div>
      <p className="muted">
        Un objectif, des analyses sourcées et des actions à valider. Reprends
        tes missions et tes conversations quand tu le souhaites.
      </p>
      {error ? (
        <p role="alert">{error}</p>
      ) : enabled === null ? (
        <p role="status">Chargement de l’assistant…</p>
      ) : enabled ? (
        <AgentConsole
          pageContext={pageContext}
          onNavigate={onNavigate}
          refreshKey={refreshKey}
        />
      ) : (
        <div className="assistant-empty">
          <p>
            Configurez votre fournisseur d’IA dans <strong>Admin</strong> pour
            commencer.
          </p>
        </div>
      )}
    </div>
  )
}
