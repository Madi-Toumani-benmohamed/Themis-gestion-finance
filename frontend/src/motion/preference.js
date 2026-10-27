import { useSyncExternalStore } from 'react'

const query = '(prefers-reduced-motion: reduce)'
export const motionReduced = () => typeof window !== 'undefined' && Boolean(window.matchMedia?.(query).matches)
const subscribe = callback => {
  const media = window.matchMedia(query)
  media.addEventListener('change', callback)
  return () => media.removeEventListener('change', callback)
}
export const useMotionReduced = () => useSyncExternalStore(subscribe, motionReduced, () => true)
