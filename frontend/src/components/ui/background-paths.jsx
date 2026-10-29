import { useMemo } from 'react'
import { m } from 'motion/react'
import { useMotionReduced } from '../../motion/preference'
import './background-paths.css'

function FloatingPaths({ position, reduced }) {
  const paths = useMemo(() => Array.from({ length: 36 }, (_, i) => ({
    id: i,
    d: `M-${380 - i * 5 * position} -${189 + i * 6}C-${380 - i * 5 * position} -${189 + i * 6} -${312 - i * 5 * position} ${216 - i * 6} ${152 - i * 5 * position} ${343 - i * 6}C${616 - i * 5 * position} ${470 - i * 6} ${684 - i * 5 * position} ${875 - i * 6} ${684 - i * 5 * position} ${875 - i * 6}`,
    width: 0.5 + i * 0.03,
    opacity: 0.2 + i * 0.013,
    duration: 1.8 + (i % 5) * 0.22,
    delay: (i % 18) * 0.025,
  })), [position])

  return (
    <div className="background-paths__layer" aria-hidden="true" data-static={reduced}>
      <svg className="background-paths__svg" viewBox="0 0 696 316" fill="none" preserveAspectRatio="xMidYMid slice">
        {paths.map(path => (
          <path
            key={path.id}
            d={path.d}
            stroke="currentColor"
            strokeWidth={path.width}
            strokeOpacity={path.opacity}
            pathLength="1"
            strokeDasharray="0.22 0.78"
            className="background-paths__stroke"
            style={{ '--path-duration': `${path.duration}s`, '--path-delay': `${path.delay}s` }}
          />
        ))}
      </svg>
    </div>
  )
}

/** Decorative, reusable headline background using Thémis's existing Motion system. */
export function BackgroundPaths({
  title = 'Background Paths',
  actionLabel,
  onAction,
  className = '',
  ambient = false,
}) {
  const reduced = useMotionReduced()
  const words = title.split(' ')

  return (
    <section className={`background-paths ${ambient ? 'background-paths--ambient' : ''} ${className}`.trim()} aria-label={ambient ? undefined : title} aria-hidden={ambient || undefined}>
      <div className="background-paths__art" aria-hidden="true">
        <FloatingPaths position={1} reduced={reduced} />
        <FloatingPaths position={-1} reduced={reduced} />
      </div>
      {!ambient && <div className="background-paths__content">
        <h1 className="background-paths__title" aria-label={title}>
          {words.map((word, wordIndex) => (
            <span className="background-paths__word" key={`${word}-${wordIndex}`}>
              {word.split('').map((letter, letterIndex) => (
                <m.span
                  key={`${wordIndex}-${letterIndex}`}
                  initial={reduced ? false : { y: 36, opacity: 0 }}
                  animate={{ y: 0, opacity: 1 }}
                  transition={reduced ? { duration: 0 } : {
                    delay: wordIndex * 0.08 + letterIndex * 0.018,
                    type: 'spring',
                    stiffness: 150,
                    damping: 25,
                  }}
                  className="background-paths__letter"
                >
                  {letter}
                </m.span>
              ))}
            </span>
          ))}
        </h1>
        {actionLabel && onAction && (
          <button className="background-paths__action" type="button" onClick={onAction}>
            <span>{actionLabel}</span><span aria-hidden="true">→</span>
          </button>
        )}
      </div>}
    </section>
  )
}
