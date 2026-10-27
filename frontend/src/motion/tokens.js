export const timing = Object.freeze({ press: 0.12, feedback: 0.18, page: 0.32, reveal: 0.52, number: 0.44, chart: 0.86 })
export const ease = [0.22, 1, 0.36, 1]
export const spring = Object.freeze({ type: 'spring', stiffness: 420, damping: 38, mass: 0.8 })
export const revealDelay = index => Math.min(Math.max(index, 0), 5) * 0.055
