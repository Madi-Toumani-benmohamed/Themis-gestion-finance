import { useEffect, useState } from 'react'

// ScrollProgress — une fine barre en haut de l'écran qui reflète la progression
// de lecture de la page courante. Discrète, teintée accent : un repère de
// position, pas une décoration criarde. Se cache quand la page ne défile pas.
export default function ScrollProgress({ scrollRef }) {
  const [pct, setPct] = useState(0)

  useEffect(() => {
    const el = scrollRef?.current || document.querySelector('.main-scroll') || window
    const target = el === window ? document.documentElement : el

    const onScroll = () => {
      const scrollTop = el === window ? window.scrollY : el.scrollTop
      const height = target.scrollHeight - target.clientHeight
      setPct(height > 0 ? Math.min(1, scrollTop / height) : 0)
    }
    const listener = el === window ? window : el
    listener.addEventListener('scroll', onScroll, { passive: true })
    onScroll()
    return () => listener.removeEventListener('scroll', onScroll)
  }, [scrollRef])

  return (
    <div className="scroll-progress" aria-hidden="true">
      <div className="scroll-progress-bar" style={{ transform: `scaleX(${pct})` }} />
    </div>
  )
}
