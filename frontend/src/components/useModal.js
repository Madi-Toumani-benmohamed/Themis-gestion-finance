import { useCallback, useEffect, useRef } from 'react'
import { motionReduced } from '../motion/preference'

function animateDismissal(dialog) {
  const bounds = dialog.getBoundingClientRect()
  // A decorative snapshot can finish exiting after React releases the real
  // dialog. It owns no handlers, focus or delayed application-state updates.
  const snapshot = dialog.cloneNode(true)
  snapshot.inert = true
  snapshot.setAttribute('aria-hidden', 'true')
  snapshot.removeAttribute('id')
  snapshot.querySelectorAll('[id]').forEach(el => el.removeAttribute('id'))
  Object.assign(snapshot.style, { position: 'fixed', inset: 'auto', left: `${bounds.left}px`, top: `${bounds.top}px`, width: `${bounds.width}px`, height: `${bounds.height}px`, margin: '0', zIndex: '1000', pointerEvents: 'none' })
  ;(dialog.closest('.app') || document.body).appendChild(snapshot)
  const sourceElements = [dialog, ...dialog.querySelectorAll('*')]
  ;[snapshot, ...snapshot.querySelectorAll('*')].forEach((el, index) => {
    el.scrollTop = sourceElements[index].scrollTop
    el.scrollLeft = sourceElements[index].scrollLeft
  })
  const animation = snapshot.animate([
    { opacity: 1, transform: 'none' },
    { opacity: 0, transform: dialog.classList.contains('transaction-drawer') ? 'translateX(24px)' : 'translateY(6px) scale(.99)' },
  ], { duration: 140, easing: 'ease-out', fill: 'forwards' })
  const media = window.matchMedia('(prefers-reduced-motion: reduce)')
  const finish = () => { if (media.matches) animation.finish() }
  media.addEventListener('change', finish)
  animation.finished.catch(() => {}).finally(() => {
    snapshot.remove()
    media.removeEventListener('change', finish)
  })
}

export function useModal(ref, fallbackSelector = '.transactions-heading h1') {
  const running = useRef(null), closing = useRef(false)
  useEffect(() => {
    closing.current = false
    const opener = document.activeElement
    const dialog = ref.current
    dialog?.showModal()
    if (dialog) {
      dialog.dataset.motionDialog = ''
      if (!motionReduced() && dialog.animate) running.current = dialog.animate([
        { opacity: 0, transform: dialog.classList.contains('transaction-drawer') ? 'translateX(40px)' : 'translateY(14px) scale(.98)' },
        { opacity: 1, transform: 'none' },
      ], { duration: 320, easing: 'cubic-bezier(.22,1,.36,1)' })
    }
    const media = window.matchMedia('(prefers-reduced-motion: reduce)')
    const finishMotion = () => { if (media.matches) running.current?.finish() }
    media.addEventListener('change', finishMotion)
    const cycleFocus = (event) => {
      if (event.key !== 'Tab') return
      const controls = [
        ...dialog.querySelectorAll(
          'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex="0"]',
        ),
      ].filter((el) => el.getClientRects().length)
      const first = controls[0],
        last = controls.at(-1)
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last?.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first?.focus()
      }
    }
    dialog?.addEventListener('keydown', cycleFocus)
    return () => {
      running.current?.cancel()
      media.removeEventListener('change', finishMotion)
      dialog?.removeEventListener('keydown', cycleFocus)
      dialog?.close()
      const target = opener?.isConnected
        ? opener
        : opener?.id
          ? document.getElementById(opener.id)
          : null
      if (target) target.focus()
      else {
        const heading = document.querySelector(fallbackSelector)
        if (heading) {
          heading.tabIndex = -1
          heading.focus()
        }
      }
    }
  }, [])
  return useCallback((onClose) => {
    if (closing.current) return
    closing.current = true
    running.current?.cancel()
    if (!motionReduced() && ref.current?.animate) animateDismissal(ref.current)
    onClose()
  }, [ref])
}
