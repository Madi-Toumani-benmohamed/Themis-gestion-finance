import { createContext, useContext, useState, useCallback, useRef } from 'react'

const ToastCtx = createContext(() => {})

export const useToast = () => useContext(ToastCtx)

let seq = 0

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([])
  const timers = useRef({})

  const dismiss = useCallback((id) => {
    setToasts((ts) => ts.filter((t) => t.id !== id))
    clearTimeout(timers.current[id])
    delete timers.current[id]
  }, [])

  // opts : { ttl, action: { label, onClick }, group }
  const push = useCallback((kind, text, opts = {}) => {
    const id = ++seq
    const ttl = opts.ttl ?? (kind === 'err' ? 6000 : 4200)
    setToasts((ts) => {
      // Si un groupe est fourni, la nouvelle notification remplace les
      // précédentes du même groupe (évite l'empilement des propositions).
      let next = ts
      if (opts.group) {
        next = ts.filter((t) => {
          if (t.group === opts.group) { clearTimeout(timers.current[t.id]); delete timers.current[t.id]; return false }
          return true
        })
      }
      return [...next, { id, kind, text, action: opts.action, group: opts.group }]
    })
    if (ttl > 0) timers.current[id] = setTimeout(() => dismiss(id), ttl)
    return id
  }, [dismiss])

  const toast = useCallback(Object.assign(
    (text) => push('info', text),
    {
      ok: (t, opts) => push('ok', t, opts),
      err: (t, opts) => push('err', t, opts),
      info: (t, opts) => push('info', t, opts),
      // toast d'action : disparaît seul après un délai généreux (le temps de
      // décider), et une seule proposition du même groupe à la fois.
      action: (text, label, onClick, opts = {}) =>
        push('action', text, { ttl: 9000, action: { label, onClick }, group: 'rule', ...opts }),
      dismiss,
    },
  ), [push, dismiss])

  const icon = { ok: '✓', err: '✕', info: '•', action: '?' }

  return (
    <ToastCtx.Provider value={toast}>
      {children}
      <div className="toast-stack" role="status" aria-live="polite">
        {toasts.map((t) => (
          <div key={t.id} className={`toast toast-${t.kind}`}>
            <span className="toast-icon">{icon[t.kind]}</span>
            <span className="toast-text">{t.text}</span>
            {t.action ? (
              <span className="toast-actions">
                <button className="toast-btn" onClick={() => { t.action.onClick(); dismiss(t.id) }}>
                  {t.action.label}
                </button>
                <button className="toast-btn ghost" onClick={() => dismiss(t.id)}>Ignorer</button>
              </span>
            ) : (
              <>
                <button className="toast-close" onClick={() => dismiss(t.id)} aria-label="Fermer">×</button>
                <span className="toast-bar" />
              </>
            )}
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  )
}
