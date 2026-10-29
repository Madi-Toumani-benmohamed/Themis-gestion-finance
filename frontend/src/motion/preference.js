import { useSyncExternalStore } from 'react'

const query = '(prefers-reduced-motion: reduce)'
const changeEvent = 'themis-motion-change'
const choices = new Set(['auto', 'full', 'reduced', 'off'])
let preference

const readPreference = () => {
  if (preference) return preference
  try { const saved = localStorage.getItem('themis-motion'); return choices.has(saved) ? saved : 'auto' } catch { return 'auto' }
}
export const motionMode = () => {
  if (typeof window === 'undefined') return 'reduced'
  const selected = readPreference()
  if (selected === 'off') return 'off'
  if (window.matchMedia?.(query).matches) return 'reduced'
  // CPU count is not an accessibility preference or a reliable GPU benchmark.
  if (selected === 'auto') return navigator.connection?.saveData ? 'reduced' : 'full'
  return selected
}
export const motionReduced = () => motionMode() !== 'full'
export const subscribeMotion = callback => {
  const media = window.matchMedia(query)
  const storage = event => { if (!event.key || event.key === 'themis-motion') { preference = undefined; callback() } }
  media.addEventListener('change', callback)
  window.addEventListener(changeEvent, callback)
  window.addEventListener('storage', storage)
  navigator.connection?.addEventListener?.('change', callback)
  return () => {
    media.removeEventListener('change', callback)
    window.removeEventListener(changeEvent, callback)
    window.removeEventListener('storage', storage)
    navigator.connection?.removeEventListener?.('change', callback)
  }
}
export function setMotionPreference(value) {
  if (!choices.has(value)) return
  preference = value
  try { localStorage.setItem('themis-motion', value) } catch { /* The preference still works for this session. */ }
  document.documentElement.dataset.motion = motionMode()
  window.dispatchEvent(new Event(changeEvent))
}
export const useMotionReduced = () => useSyncExternalStore(subscribeMotion, motionReduced, () => true)
export const useMotionMode = () => useSyncExternalStore(subscribeMotion, motionMode, () => 'reduced')
export const useMotionPreference = () => useSyncExternalStore(subscribeMotion, readPreference, () => 'auto')
