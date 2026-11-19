import { lazy, Suspense, useEffect, useState } from 'react'
const AgentConsole = lazy(() => import('./agent/AgentConsole'))
import './AssistantFloating.css'

export default function AssistantFloating({
  aiEnabled,
  pageContext,
  onNavigate,
  refreshKey,
}) {
  const [open, setOpen] = useState(false)
  const [hasOpened, setHasOpened] = useState(false)
  useEffect(() => {
    const show = () => {
      setHasOpened(true)
      setOpen(true)
    }
    window.addEventListener('themis:agent-open', show)
    return () => window.removeEventListener('themis:agent-open', show)
  }, [])
  useEffect(() => {
    const key = (e) => {
      if (e.key === 'Escape') setOpen(false)
    }
    if (open) window.addEventListener('keydown', key)
    return () => window.removeEventListener('keydown', key)
  }, [open])
  if (!aiEnabled) return null
  return (
    <>
      <button
        className={`assistant-float-btn ${open ? 'open' : ''}`}
        aria-label="Ouvrir l'assistant"
        aria-expanded={open}
        title="Assistant thémis"
        onClick={() => {
          setHasOpened(true)
          setOpen((v) => !v)
        }}
      >
        <svg
          aria-hidden="true"
          width="22"
          height="22"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.7"
        >
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
      </button>
      {(hasOpened || open) && (
        <div
          className="assistant-panel"
          hidden={!open}
          style={open ? undefined : { display: 'none' }}
          role="region"
          aria-label="Assistant thémis"
        >
          <div className="assistant-panel-head">
            <span>Assistant thémis</span>
            <button
              className="assistant-panel-close"
              aria-label="Fermer"
              onClick={() => setOpen(false)}
            >
              ✕
            </button>
          </div>
          <div className="assistant-panel-body">
            <Suspense fallback={<p role="status">Chargement de l’agent…</p>}>
              <AgentConsole
                compact
                pageContext={pageContext}
                onNavigate={onNavigate}
                refreshKey={refreshKey}
              />
            </Suspense>
          </div>
        </div>
      )}
    </>
  )
}
