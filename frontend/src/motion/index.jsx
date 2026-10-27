import { useCallback, useEffect, useId, useRef } from 'react'
import { flushSync } from 'react-dom'
import { LazyMotion, domMax, MotionConfig, m, LayoutGroup, useAnimationControls, useInView, useIsPresent } from 'motion/react'
import { motionReduced, useMotionReduced } from './preference'
import { ease, revealDelay, spring, timing } from './tokens'

export { m, AnimatePresence } from 'motion/react'
export { useMotionReduced } from './preference'
export { timing, ease, spring } from './tokens'

export function MotionProvider({ children }) {
  const reduced = useMotionReduced()
  return <LazyMotion features={domMax} strict><MotionConfig reducedMotion={reduced ? 'always' : 'never'} transition={{ duration: reduced ? 0 : timing.feedback, ease }}>{children}</MotionConfig></LazyMotion>
}

// Snapshot transitions keep one live page, one focus tree and one set of effects.
export function usePageNavigation() {
  const generation = useRef(0), pending = useRef(null)
  useEffect(() => {
    const media = window.matchMedia('(prefers-reduced-motion: reduce)')
    const stop = () => { if (media.matches) pending.current?.skipTransition() }
    media.addEventListener('change', stop)
    return () => {
      generation.current++
      pending.current?.skipTransition()
      delete document.documentElement.dataset.motionNavigation
      media.removeEventListener('change', stop)
    }
  }, [])
  return useCallback(update => {
    const revision = ++generation.current
    pending.current?.skipTransition()
    if (motionReduced() || !document.startViewTransition) {
      delete document.documentElement.dataset.motionNavigation
      update()
      return
    }
    document.documentElement.dataset.motionNavigation = 'active'
    const transition = document.startViewTransition(() => {
      if (revision === generation.current) flushSync(update)
    })
    pending.current = transition
    transition.ready.catch(() => {})
    transition.finished.catch(() => {}).finally(() => {
      if (revision === generation.current) {
        delete document.documentElement.dataset.motionNavigation
        pending.current = null
      }
    })
  }, [])
}

export function PageTransition({ children, className = '', page }) {
  return <main className={`${className} motion-page`} data-page={page}>{children}</main>
}

export function NavigationMotion({ children }) {
  return <LayoutGroup id="themis-navigation">{children}</LayoutGroup>
}

// Exiting content can remain painted, but must no longer accept an action.
export function PresencePanel({ children, ...props }) {
  const present = useIsPresent()
  return <m.div {...props} inert={present ? undefined : ''} aria-hidden={present ? undefined : true}>{children}</m.div>
}

export function ActiveNavigation() {
  const reduced = useMotionReduced()
  return <m.span aria-hidden="true" className="motion-nav-indicator" layoutId="navigation-selection" transition={reduced ? { duration: 0 } : spring}><i /></m.span>
}

export function AnimatedTabs({ children, className = '', as: Tag = 'div', ...props }) {
  const id = useId()
  return <LayoutGroup id={id}><Tag {...props} className={`motion-tabs ${className}`}>{children}</Tag></LayoutGroup>
}

export function TabIndicator() {
  const reduced = useMotionReduced()
  return <m.span aria-hidden="true" className="motion-tab-indicator" layoutId="tab-selection" transition={reduced ? { duration: 0 } : spring} />
}

export function Reveal({ children, delay = 0, className = '', as = 'div', direction = 'up', hover = false, style, ...props }) {
  const reduced = useMotionReduced(), ref = useRef(null), controls = useAnimationControls()
  const visible = useInView(ref, { once: true, amount: 0.08, margin: '0px 0px -20px 0px' })
  useEffect(() => { if (visible || reduced) controls.start('visible') }, [visible, reduced, controls])
  const Tag = m[as]
  const offset = direction === 'left' ? { x: -18, y: 0 } : direction === 'right' ? { x: 18, y: 0 } : { x: 0, y: 18 }
  return <Tag {...props} ref={ref} className={`motion-reveal ${className}`} style={style}
    initial={reduced ? false : 'hidden'} animate={controls}
    variants={{ hidden: { opacity: 0, ...offset }, visible: { opacity: 1, x: 0, y: 0, transition: { duration: reduced ? 0 : timing.reveal, ease, delay: reduced ? 0 : Math.min(delay / 1000, 0.32) } } }}
    onFocusCapture={() => controls.set('visible')}
    whileHover={hover && !reduced ? { y: -3, transition: { duration: 0.2, ease } } : undefined}>
    {children}
  </Tag>
}

export function MotionCard({ index = 0, ...props }) {
  return <Reveal hover delay={revealDelay(index) * 1000} {...props} />
}

export function AnimatedRegion({ revision, children, className = '', as: Tag = 'div', ...props }) {
  const ref = useRef(null), reduced = useMotionReduced()
  useEffect(() => {
    if (reduced || !ref.current?.animate) return
    const animation = ref.current.animate([{ opacity: 0.55, transform: 'translateY(6px)' }, { opacity: 1, transform: 'none' }], { duration: 280, easing: 'cubic-bezier(.22,1,.36,1)' })
    return () => animation.cancel()
  }, [revision, reduced])
  return <Tag {...props} ref={ref} className={`motion-region ${className}`}>{children}</Tag>
}

export function ProgressFill({ percent, className = '', style }) {
  const reduced = useMotionReduced()
  const ratio = Number.isFinite(percent) ? Math.max(0, Math.min(100, percent)) / 100 : 0
  return <m.span className={`motion-progress-fill ${className}`} style={{ ...style, width: '100%', transformOrigin: 'left center' }} initial={reduced ? false : { scaleX: 0 }} animate={{ scaleX: ratio }} transition={{ duration: reduced ? 0 : 0.5, ease }} />
}
