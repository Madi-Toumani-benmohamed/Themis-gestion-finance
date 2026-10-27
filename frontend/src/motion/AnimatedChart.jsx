import { useEffect, useRef } from 'react'
import { useInView } from 'motion/react'
import { useMotionReduced } from './preference'

// Reveal actual SVG geometry; never interpolate source values or bridge gaps.
// Only new shapes are observed. Brush/axis d-attribute updates do not restart it.
export function AnimatedChart({ children, revision, className = '', style, dense = false, growBars = false, ...props }) {
  const ref = useRef(null), reduced = useMotionReduced()
  const visible = useInView(ref, { once: true, amount: 0.08 })
  useEffect(() => {
    const el = ref.current
    if (!el || reduced || !visible || !el.animate) return
    const animated = new WeakSet(), running = new Set()
    let frame = 0
    const selector = '.recharts-line-curve, .recharts-area-curve, .recharts-area-area, .recharts-bar-rectangle, .recharts-pie-sector'
    const reveal = () => {
      frame = 0
      const shapes = [...el.querySelectorAll(selector)].filter(shape => !shape.closest('.recharts-brush'))
      shapes.slice(0, 60).forEach((shape, index) => {
        if (animated.has(shape)) return
        animated.add(shape)
        let keyframes = [{ opacity: 0 }, { opacity: 1 }]
        const curve = shape.classList.contains('recharts-line-curve') || shape.classList.contains('recharts-area-curve')
        if (curve && !dense && shape.getTotalLength) {
          const length = shape.getTotalLength()
          if (length > 0) keyframes = [{ strokeDasharray: `${length} ${length}`, strokeDashoffset: length }, { strokeDasharray: `${length} ${length}`, strokeDashoffset: 0 }]
        } else if (shape.classList.contains('recharts-bar-rectangle') && growBars && !dense) {
          keyframes = [{ opacity: 0, transform: 'scaleY(.05)' }, { opacity: 1, transform: 'scaleY(1)' }]
          shape.style.transformBox = 'fill-box'
          shape.style.transformOrigin = 'center bottom'
        } else if (shape.classList.contains('recharts-pie-sector') && !dense) {
          keyframes = [{ opacity: 0, transform: 'scale(.92)' }, { opacity: 1, transform: 'scale(1)' }]
          shape.style.transformBox = 'view-box'
          shape.style.transformOrigin = 'center'
        }
        const animation = shape.animate(keyframes, { duration: dense ? 180 : curve ? 950 : 620, delay: dense ? 0 : Math.min(index * 65, 260), easing: 'cubic-bezier(.22,1,.36,1)', fill: 'backwards' })
        running.add(animation)
        animation.finished.catch(() => {}).finally(() => running.delete(animation))
      })
    }
    const schedule = () => { if (!frame) frame = requestAnimationFrame(reveal) }
    const observer = new MutationObserver(records => {
      // Tooltips and focus dots change during hover; they must not cause SVG scans.
      if (records.some(record => [...record.addedNodes].some(node => node.nodeType === 1 && (node.matches(selector) || node.querySelector(selector))))) schedule()
    })
    observer.observe(el, { subtree: true, childList: true })
    schedule()
    return () => { observer.disconnect(); cancelAnimationFrame(frame); running.forEach(animation => animation.cancel()) }
  }, [visible, reduced, revision, dense, growBars])
  return <div {...props} ref={ref} className={`motion-chart ${className}`} style={style}>{children}</div>
}

export function ChartFocusDot({ cx, cy, fill, stroke }) {
  if (!Number.isFinite(cx) || !Number.isFinite(cy)) return null
  const color = stroke || fill || 'var(--accent)'
  return <g className="motion-chart-focus" aria-hidden="true" key={`${cx}-${cy}`}>
    <circle className="motion-chart-halo" cx={cx} cy={cy} r={11} fill={color} fillOpacity={0.12} />
    <circle cx={cx} cy={cy} r={4} fill={color} stroke="var(--paper)" strokeWidth={2} />
  </g>
}
