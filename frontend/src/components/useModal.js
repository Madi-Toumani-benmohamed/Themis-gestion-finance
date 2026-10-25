import { useEffect } from 'react'

export function useModal(ref, fallbackSelector = '.transactions-heading h1') {
  useEffect(() => {
    const opener = document.activeElement
    const dialog = ref.current
    dialog?.showModal()
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
}
