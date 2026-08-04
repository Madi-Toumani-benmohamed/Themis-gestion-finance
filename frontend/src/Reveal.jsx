import { useEffect, useRef, useState } from 'react'

// useReveal — observe un élément et bascule `visible` à true quand il entre
// dans le viewport. One-shot (ne rejoue pas au re-scroll), respectueux de
// prefers-reduced-motion (visible d'emblée si l'utilisateur préfère éviter les
// animations).
export function useReveal(options = {}) {
  const ref = useRef(null)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    // Respect de l'accessibilité : pas d'animation → contenu visible tout de suite.
    if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) {
      setVisible(true)
      return
    }
    const obs = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) {
            setVisible(true)
            obs.unobserve(e.target) // one-shot
          }
        }
      },
      { threshold: 0.12, rootMargin: '0px 0px -8% 0px', ...options }
    )
    obs.observe(el)
    return () => obs.disconnect()
  }, [])

  return [ref, visible]
}

// Reveal — enrobe des enfants dans un conteneur qui apparaît en fondu-montée
// à l'entrée dans le viewport. `delay` (en ms) permet d'échelonner plusieurs
// blocs pour un effet de cascade.
export function Reveal({ children, delay = 0, className = '', as: Tag = 'div' }) {
  const [ref, visible] = useReveal()
  return (
    <Tag
      ref={ref}
      className={`reveal ${visible ? 'reveal-in' : ''} ${className}`}
      style={delay ? { transitionDelay: `${delay}ms` } : undefined}
    >
      {children}
    </Tag>
  )
}
