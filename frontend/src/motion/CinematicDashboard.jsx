import { useEffect, useRef, useState } from 'react'
import { MotionControl, useMotion } from './Motion'

// Original geometry only. The ornament never encodes financial information.
function AstralScene() {
  const ref = useRef(null)
  const { full } = useMotion()
  useEffect(() => {
    const element = ref.current
    if (!full || !window.matchMedia('(hover: hover) and (pointer: fine)').matches || !window.IntersectionObserver) return
    let frame = 0
    const start = element.getBoundingClientRect().top + window.scrollY
    const update = () => {
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(() => element.style.setProperty('--scene-parallax', `${Math.min(18, Math.max(0, window.scrollY - start) * .055)}px`))
    }
    const observer = new IntersectionObserver(entries => {
      window.removeEventListener('scroll', update)
      if (entries.some(entry => entry.isIntersecting)) { update(); window.addEventListener('scroll', update, { passive: true }) }
    })
    observer.observe(element)
    return () => { observer.disconnect(); cancelAnimationFrame(frame); window.removeEventListener('scroll', update); element.style.removeProperty('--scene-parallax') }
  }, [full])
  return <div ref={ref} className="cinematic-scene" aria-hidden="true">
    <div className="scene-nebula" />
    <div className="scene-beam scene-beam-one" /><div className="scene-beam scene-beam-two" />
    <div className="scene-orbit-system">
      <div className="scene-orbit orbit-one" /><div className="scene-orbit orbit-two" /><div className="scene-orbit orbit-three" />
      <svg className="scene-crystal" viewBox="0 0 200 260" fill="none">
        <path d="M100 12 171 96 100 248 29 96Z" fill="#AEBBFF" fillOpacity=".07" stroke="#B9CDFF" strokeOpacity=".65" />
        <path d="M100 12 123 102 100 248 76 102Z" fill="#CEE2FF" fillOpacity=".15" stroke="#CEDEFF" strokeOpacity=".8" />
        <path d="m29 96 71 30 71-30M100 12v236M76 102l24 24 23-24" stroke="#B9CDFF" strokeOpacity=".45" />
        <path d="m100 12 23 90-23 24Z" fill="#F1F7FF" fillOpacity=".32" />
        <path d="m100 126 71-30-71 152Z" fill="#849EFF" fillOpacity=".15" />
        <circle cx="100" cy="126" r="4" fill="#E4F5FF" />
      </svg>
    </div>
    <div className="scene-horizon" />
    {Array.from({ length: 16 }, (_, index) => <i key={index} className="scene-particle" style={{ '--particle-x': `${(index * 47 + 17) % 100}%`, '--particle-y': `${(index * 31 + 11) % 90}%`, '--particle-delay': `${index * -357}ms`, '--particle-size': `${index % 3 + 1}px` }} />)}
    <span className="scene-coordinate">THÉMIS / PERSPECTIVES</span>
  </div>
}

export function CinematicDashboard({ children }) {
  const { dashboardSeen, full } = useMotion()
  const [first, setFirst] = useState(() => !dashboardSeen.current)
  const opening = useRef(first)
  useEffect(() => {
    dashboardSeen.current = true
    if (!full) { setFirst(false); return }
    const timer = setTimeout(() => setFirst(false), 1900)
    return () => clearTimeout(timer)
  }, [dashboardSeen, full])
  return <div className="section dashboard cinematic-dashboard" data-entrance={first && opening.current ? 'first' : 'settled'}>
    <AstralScene />
    <div className="cinematic-masthead"><span className="cinematic-edition"><i aria-hidden="true" /> Une nouvelle perspective</span><MotionControl /></div>
    {children}
  </div>
}
