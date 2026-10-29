import { createContext, useContext, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { m, useInView } from 'motion/react'
import { useMotionMode, useMotionPreference, setMotionPreference } from './preference'

const MotionSession = createContext(null)

// Shared with the existing Motion provider; never mount a competing motion engine.
export function MotionSessionProvider({ children }) {
  const session = useRef({ dashboardSeen: false, surfacesSeen: new Set() })
  const mode = useMotionMode()
  useLayoutEffect(() => { document.documentElement.dataset.motion = mode }, [mode])
  return <MotionSession.Provider value={session.current}>{children}</MotionSession.Provider>
}

export function useMotion() {
  const session = useContext(MotionSession)
  const mode = useMotionMode(), preference = useMotionPreference()
  return { mode, full: mode === 'full', preference, setPreference: setMotionPreference, session }
}

export function MotionControl() {
  const { preference, setPreference } = useMotion()
  return <label className="motion-control"><span className="motion-control-label">Animations</span>
    <select aria-label="Animations de l’interface" value={preference} onChange={event => setPreference(event.target.value)}>
      <option value="auto">Automatiques</option><option value="full">Complètes</option><option value="reduced">Réduites</option><option value="off">Désactivées</option>
    </select>
  </label>
}

// Never hide financial information while waiting for an observer or an animation.
export function MotionSurface({ children, className = '', as = 'div', delay = 0, revealKey, layout, style, ...props }) {
  const Tag = m[as]
  const ref = useRef(null)
  const { full, session } = useMotion()
  const [seen] = useState(() => session?.surfacesSeen.has(revealKey))
  const visible = useInView(ref, { once: true, amount: .08 })
  useEffect(() => {
    if (!visible || seen || session?.surfacesSeen.has(revealKey) || !full || !ref.current?.animate) return
    const animation = ref.current.animate([
      { opacity: .6, transform: 'translateY(18px)' },
      { opacity: 1, transform: 'none' },
    ], { duration: 650, delay: Math.min(delay, 280), easing: 'cubic-bezier(.22,1,.36,1)', fill: 'backwards' })
    // StrictMode may cancel the first attempt. Only a completed entry is remembered.
    animation.finished.then(() => { if (revealKey) session?.surfacesSeen.add(revealKey) }).catch(() => {})
    return () => animation.cancel()
  }, [visible, full, seen, delay, revealKey, session])
  return <Tag {...props} ref={ref} layout={full ? layout : undefined} transition={{ layout: { type: 'spring', stiffness: 420, damping: 38 } }} className={`motion-surface ${className}`} data-revealed={visible && !seen} style={style}>{children}</Tag>
}
