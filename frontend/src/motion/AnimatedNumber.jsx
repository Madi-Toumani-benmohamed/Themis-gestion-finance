import { useEffect, useRef } from 'react'
import { m } from 'motion/react'
import { eur } from '../api'
import { useMotionReduced } from './preference'
import { ease, timing } from './tokens'

// Format first, animate glyphs only. The real text/accessible value is always
// the exact target; generated decorative glyphs never enter copied text.
export function AnimatedNumber({ value, format = eur, missing = '—', className = '', delay = 0 }) {
  const reduced = useMotionReduced(), previous = useRef(value)
  const valid = typeof value === 'number' && Number.isFinite(value)
  const formatted = valid ? format(value) : missing
  const direction = typeof previous.current === 'number' && value < previous.current ? -1 : 1
  useEffect(() => { previous.current = value }, [value])
  if (reduced || !valid) return <span className={`animated-number ${className}`}>{formatted}</span>
  const characters = Array.from(String(formatted))
  return <span className={`animated-number ${className}`} data-motion-number="true">
    <span className="motion-number-value">{formatted}</span>
    <span className="motion-number-reel" aria-hidden="true">
      {characters.map((character, index) => <span className="motion-digit-slot" key={characters.length - index}>
        <m.span className="motion-digit" data-character={character} key={character}
          initial={{ y: `${direction * 85}%`, opacity: 0 }} animate={{ y: '0%', opacity: 1 }}
          transition={{ duration: timing.number, ease, delay: Math.min(delay / 1000 + index * 0.016, 0.25) }} />
      </span>)}
    </span>
  </span>
}
