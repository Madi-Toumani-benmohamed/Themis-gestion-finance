import { useEffect, useRef, useState } from 'react'
import { MotionControl, useMotion } from './Motion'

// The ornament is decorative. Its geometry never represents a financial value.
function AstralScene({ opening }) {
  const ref = useRef(null)
  const { full } = useMotion()
  useEffect(() => {
    if (!full || !opening || !ref.current?.animate) return
    const animations = [...ref.current.querySelectorAll('[data-draw]')].map((path, index) => {
      const length = path.getTotalLength()
      return path.animate([
        { strokeDasharray: `${length} ${length}`, strokeDashoffset: length, opacity: .15 },
        { strokeDasharray: `${length} ${length}`, strokeDashoffset: 0, opacity: 1 },
      ], { duration: 1250, delay: index * 55, easing: 'cubic-bezier(.22,1,.36,1)', fill: 'backwards' })
    })
    return () => animations.forEach(animation => animation.cancel())
  }, [full, opening])
  return <div ref={ref} className="cinematic-scene" aria-hidden="true">
    <div className="scene-grid" /><div className="scene-light" />
    <svg className="scene-orbit-system" viewBox="0 0 360 260" fill="none">
      <g className="scene-orbits" stroke="currentColor">
        <ellipse data-draw cx="180" cy="130" rx="152" ry="63" transform="rotate(-24 180 130)" />
        <ellipse data-draw cx="180" cy="130" rx="127" ry="82" transform="rotate(37 180 130)" />
        <circle data-draw cx="180" cy="130" r="104" strokeDasharray="2 8" />
      </g>
      <g className="scene-crystal" stroke="currentColor">
        <path data-draw d="M180 38 226 115 180 221 134 115Z" fill="currentColor" fillOpacity=".04" />
        <path data-draw d="M180 38v183m-46-106 46 24 46-24M180 38l17 77-17 106-17-106Z" />
        <path d="m180 38 17 77-17 24Z" fill="currentColor" fillOpacity=".15" stroke="none" />
        <circle cx="180" cy="139" r="3" fill="currentColor" stroke="none" />
      </g>
      <g className="scene-stars" fill="currentColor"><circle cx="61" cy="83" r="2" /><circle cx="295" cy="174" r="2" /><circle cx="254" cy="57" r="1.5" /></g>
    </svg>
  </div>
}

export function CinematicDashboard({ header, children }) {
  const { session, full } = useMotion()
  const headerRef = useRef(null), pointerFrame = useRef(0)
  const [opening, setOpening] = useState(() => !session?.dashboardSeen && full)
  useEffect(() => {
    if (session) session.dashboardSeen = true
    if (!full) { setOpening(false); return }
    const timer = setTimeout(() => setOpening(false), 1900)
    return () => clearTimeout(timer)
  }, [session, full])
  useEffect(() => {
    const header = headerRef.current
    if (!header || !full) return
    const move = event => {
      if (event.pointerType !== 'mouse') return
      cancelAnimationFrame(pointerFrame.current)
      pointerFrame.current = requestAnimationFrame(() => {
        const bounds = header.getBoundingClientRect()
        const x = ((event.clientX - bounds.left) / bounds.width - .5) * 10
        const y = ((event.clientY - bounds.top) / bounds.height - .5) * 8
        header.style.setProperty('--scene-shift-x', `${x.toFixed(1)}px`)
        header.style.setProperty('--scene-shift-y', `${y.toFixed(1)}px`)
      })
    }
    const reset = () => {
      cancelAnimationFrame(pointerFrame.current)
      header.style.setProperty('--scene-shift-x', '0px')
      header.style.setProperty('--scene-shift-y', '0px')
    }
    header.addEventListener('pointermove', move, { passive: true })
    header.addEventListener('pointerleave', reset, { passive: true })
    return () => {
      cancelAnimationFrame(pointerFrame.current)
      header.removeEventListener('pointermove', move)
      header.removeEventListener('pointerleave', reset)
      reset()
    }
  }, [full])
  return <div className="section dashboard cinematic-dashboard" data-entrance={opening ? 'first' : 'settled'}>
    <div ref={headerRef} className="cinematic-header">
      <AstralScene opening={opening} />
      <div className="cinematic-masthead"><span className="cinematic-edition"><i aria-hidden="true" /> Finances personnelles</span><MotionControl /></div>
      {header}
    </div>
    {children}
  </div>
}
