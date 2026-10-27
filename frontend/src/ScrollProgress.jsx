import { useEffect, useRef } from 'react'

// ScrollProgress — une fine barre en haut de l'écran qui reflète la progression
// de lecture de la page courante. Discrète, teintée accent : un repère de
// position, pas une décoration criarde. Se cache quand la page ne défile pas.
export default function ScrollProgress({ scrollRef }) {
  const bar = useRef(null)

  useEffect(() => {
    const el = scrollRef?.current || document.querySelector('.main-scroll') || window
    const target = el === window ? document.documentElement : el

    let frame = 0
    const paint = () => {
      frame = 0
      const scrollTop = el === window ? window.scrollY : el.scrollTop
      const height = target.scrollHeight - target.clientHeight
      if (bar.current) bar.current.style.transform = `scaleX(${height > 0 ? Math.min(1, scrollTop / height) : 0})`
    }
    const onScroll = () => { if (!frame) frame = requestAnimationFrame(paint) }
    const listener = el === window ? window : el
    listener.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll, { passive: true })
    onScroll()
    const observer = new ResizeObserver(onScroll)
    observer.observe(target === document.documentElement ? document.body : target)
    return () => { listener.removeEventListener('scroll', onScroll); window.removeEventListener('resize', onScroll); observer.disconnect(); cancelAnimationFrame(frame) }
  }, [scrollRef])

  return (
    <div className="scroll-progress" aria-hidden="true">
      <div ref={bar} className="scroll-progress-bar" style={{ transform: 'scaleX(0)' }} />
    </div>
  )
}
