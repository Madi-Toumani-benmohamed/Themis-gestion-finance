import { useEffect, useRef } from 'react'
import { useInView } from 'motion/react'
import { useMotionReduced } from './preference'

// Reveal actual SVG geometry; never interpolate source values or bridge gaps.
// Only new shapes are observed. Brush/axis d-attribute updates do not restart it.
export function AnimatedChart({ children, revision, className = '', style, dense = false, growBars = false, ...props }) {
  const ref = useRef(null), reduced = useMotionReduced()
  const revealed = useRef(false), previousRevision = useRef(revision)
  const visible = useInView(ref, { once: true, amount: 0.08 })
  useEffect(() => {
    const el = ref.current
    if (!el || reduced || !visible || !el.animate) return
    if (revealed.current && previousRevision.current === revision) return
    const changed = revealed.current && previousRevision.current !== revision
    previousRevision.current = revision
    if (changed) {
      const transition = el.animate([{ opacity: .72 }, { opacity: 1 }], { duration: 240, easing: 'cubic-bezier(.22,1,.36,1)' })
      return () => transition.cancel()
    }
    revealed.current = true
    const animated = new WeakSet(), running = new Set()
    let frame = 0
    const selector = '.recharts-line-curve, .recharts-area-curve, .recharts-area-area, .recharts-bar-rectangle, .recharts-pie-sector, .recharts-reference-dot'
    const reveal = () => {
      frame = 0
      el.classList.add('motion-chart-entered')
      const structures = [...el.querySelectorAll('.recharts-cartesian-grid, .recharts-cartesian-axis')]
      const legend = el.parentElement?.querySelector(':scope > .cinematic-chart-legend')
      if (legend) structures.push(legend)
      structures.forEach((node, index) => {
        if (node.dataset.motionRevealed) return
        node.dataset.motionRevealed = 'true'
        const animation = node.animate([{ opacity: 0 }, { opacity: 1 }], {
          duration: index === 0 ? 260 : 340,
          delay: index === 0 ? 0 : index === 1 ? 110 : 1420,
          easing: 'cubic-bezier(.22,1,.36,1)', fill: 'both',
        })
        running.add(animation)
        animation.finished.catch(() => {}).finally(() => running.delete(animation))
      })
      const shapes = [...el.querySelectorAll(selector)].filter(shape => !shape.closest('.recharts-brush'))
      shapes.slice(0, 60).forEach((shape, index) => {
        if (animated.has(shape)) return
        animated.add(shape)
        let keyframes = [{ opacity: 0 }, { opacity: 1 }]
        const curve = shape.classList.contains('recharts-line-curve') || shape.classList.contains('recharts-area-curve')
        const marker = shape.classList.contains('recharts-reference-dot')
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
        const curveDelay = curve ? 320 + (shape.classList.contains('recharts-line-curve') ? 420 : index * 145) : 0
        const fillDelay = shape.classList.contains('recharts-area-area') ? 1080 + Math.min(index * 110, 220) : 220 + Math.min(index * 55, 220)
        const delay = dense ? 0 : marker ? 1580 : curve ? curveDelay : fillDelay
        const animation = shape.animate(keyframes, { duration: dense ? 180 : marker ? 320 : curve ? 880 : 520, delay, easing: 'cubic-bezier(.22,1,.36,1)', fill: 'backwards' })
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
