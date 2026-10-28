import { createContext, useContext, useEffect, useMemo, useRef, useState } from 'react'

const MotionContext = createContext({ mode: 'off', animated: false, full: false })
const choices = new Set(['auto', 'full', 'reduced', 'off'])
const media = query => typeof window !== 'undefined' && window.matchMedia(query)

export function MotionProvider({ children }) {
  const [preference, setPreference] = useState(() => {
    try { const saved = localStorage.getItem('themis-motion'); return choices.has(saved) ? saved : 'auto' } catch { return 'auto' }
  })
  const [systemReduced, setSystemReduced] = useState(() => media('(prefers-reduced-motion: reduce)')?.matches ?? true)
  const dashboardSeen = useRef(false)
  const surfacesSeen = useRef(new Set())
  useEffect(() => {
    const query = media('(prefers-reduced-motion: reduce)')
    const update = () => setSystemReduced(query.matches)
    query.addEventListener('change', update)
    return () => query.removeEventListener('change', update)
  }, [])
  const constrained = typeof navigator !== 'undefined' && (navigator.connection?.saveData || (navigator.hardwareConcurrency && navigator.hardwareConcurrency <= 4))
  const mode = preference === 'off' ? 'off' : systemReduced ? 'reduced' : preference === 'auto' ? (constrained ? 'reduced' : 'full') : preference
  useEffect(() => {
    document.documentElement.dataset.motion = mode
    try { localStorage.setItem('themis-motion', preference) } catch { /* optional storage */ }
  }, [mode, preference])
  const value = useMemo(() => ({ mode, animated: mode === 'full', full: mode === 'full', preference, setPreference, dashboardSeen, surfacesSeen }), [mode, preference])
  return <MotionContext.Provider value={value}>{children}</MotionContext.Provider>
}

export const useMotion = () => useContext(MotionContext)

export function MotionControl() {
  const { preference, setPreference, mode } = useMotion()
  return <label className="motion-control"><span aria-hidden="true">◈</span><span className="motion-control-label">Ambiance</span>
    <select aria-label="Animations de l’interface" value={preference} onChange={event => setPreference(event.target.value)}>
      <option value="auto">Automatique</option><option value="full">Cinématique</option><option value="reduced">Réduites</option><option value="off">Désactivées</option>
    </select><span className="motion-control-state" title="Les préférences du système restent prioritaires">{mode === 'full' ? 'Cinématique' : mode === 'off' ? 'Statique' : 'Réduite'}</span>
  </label>
}

// Content is always visible. Observation starts an enhancement, never gates data.
export function useSceneReveal(ref) {
  const { full } = useMotion()
  const [visible, setVisible] = useState(false)
  useEffect(() => {
    if (!full || !ref.current || typeof IntersectionObserver === 'undefined') { setVisible(true); return }
    const observer = new IntersectionObserver(entries => {
      if (entries.some(entry => entry.isIntersecting)) { setVisible(true); observer.disconnect() }
    }, { threshold: .08 })
    observer.observe(ref.current)
    return () => observer.disconnect()
  }, [ref, full])
  return visible
}

export function MotionSurface({ children, className = '', as: Tag = 'div', delay = 0, revealKey, style, ...props }) {
  const ref = useRef(null)
  const frame = useRef(0)
  const { full, surfacesSeen } = useMotion()
  const alreadyPlayed = useRef(revealKey && surfacesSeen?.current.has(revealKey))
  const visible = useSceneReveal(ref)
  useEffect(() => {
    if (visible && revealKey) surfacesSeen?.current.add(revealKey)
  }, [visible, revealKey, surfacesSeen])
  useEffect(() => {
    const element = ref.current
    if (!full || !media('(hover: hover) and (pointer: fine)')?.matches) return
    let bounds
    const reset = () => {
      cancelAnimationFrame(frame.current)
      element.style.removeProperty('--tilt-x'); element.style.removeProperty('--tilt-y')
      element.style.removeProperty('--light-x'); element.style.removeProperty('--light-y')
      bounds = null
    }
    const enter = () => { bounds = element.getBoundingClientRect() }
    const move = event => {
      if (event.pointerType === 'touch' || !bounds || document.hidden) return
      cancelAnimationFrame(frame.current)
      const x = Math.max(0, Math.min(1, (event.clientX - bounds.left) / bounds.width))
      const y = Math.max(0, Math.min(1, (event.clientY - bounds.top) / bounds.height))
      frame.current = requestAnimationFrame(() => {
        element.style.setProperty('--tilt-x', `${(0.5 - y) * 3}deg`)
        element.style.setProperty('--tilt-y', `${(x - .5) * 3}deg`)
        element.style.setProperty('--light-x', `${x * 100}%`)
        element.style.setProperty('--light-y', `${y * 100}%`)
      })
    }
    element.addEventListener('pointerenter', enter)
    element.addEventListener('pointermove', move)
    element.addEventListener('pointerleave', reset)
    element.addEventListener('pointercancel', reset)
    window.addEventListener('scroll', reset, { passive: true })
    document.addEventListener('visibilitychange', reset)
    return () => {
      reset()
      element.removeEventListener('pointerenter', enter); element.removeEventListener('pointermove', move)
      element.removeEventListener('pointerleave', reset); element.removeEventListener('pointercancel', reset)
      window.removeEventListener('scroll', reset); document.removeEventListener('visibilitychange', reset)
    }
  }, [full])
  return <Tag {...props} ref={ref} className={`motion-surface ${className}`} data-revealed={visible && !alreadyPlayed.current} style={{ ...style, '--reveal-delay': `${delay}ms` }}>{children}</Tag>
}

// Keyed by the existing page identity: does not change routing, focus or mount order.
export function PageTransition({ children, pageKey, className = '' }) {
  const ref = useRef(null)
  const { full } = useMotion()
  useEffect(() => {
    if (!full || !ref.current?.animate) return
    const animation = ref.current.animate([
      { opacity: .7, transform: 'perspective(1600px) translate3d(0,10px,-24px) rotateX(.5deg)' },
      { opacity: 1, transform: 'perspective(1600px) translate3d(0,0,0) rotateX(0)' },
    ], { duration: 320, easing: 'cubic-bezier(.16,1,.3,1)' })
    return () => animation.cancel()
  }, [pageKey, full])
  return <main ref={ref} className={`page cinematic-page ${className}`}>{children}</main>
}
