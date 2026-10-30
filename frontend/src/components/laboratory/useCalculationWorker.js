import { useCallback, useEffect, useRef, useState } from 'react'
export function useCalculationWorker() {
  const worker = useRef(null),
    pending = useRef(null),
    sequence = useRef(0)
  const [state, setState] = useState({
    running: false,
    result: null,
    error: null,
  })
  const stop = useCallback(() => {
    if (pending.current) {
      clearTimeout(pending.current.timer)
      pending.current.reject(new DOMException('Calcul annulé', 'AbortError'))
      pending.current = null
    }
    worker.current?.terminate()
    worker.current = null
  }, [])
  useEffect(() => stop, [stop])
  const run = useCallback(
    (payload) => {
      stop()
      setState({ running: true, result: null, error: null })
      const id = ++sequence.current
      return new Promise((resolve, reject) => {
        const finish = (result, error) => {
          const p = pending.current
          if (!p || p.id !== id) return
          clearTimeout(p.timer)
          pending.current = null
          setState({ running: false, result, error })
          if (error)
            reject(
              Object.assign(new Error(error.message), { code: error.code }),
            )
          else resolve(result)
        }
        let engine
        try {
          engine = new Worker(
            new URL(
              '../../lib/laboratory/calculationWorker.js',
              import.meta.url,
            ),
            { type: 'module' },
          )
        } catch (e) {
          setState({
            running: false,
            result: null,
            error: { message: 'Moteur de calcul indisponible.' },
          })
          reject(e)
          return
        }
        worker.current = engine
        engine.onmessage = (event) => {
          if (event.data.id === id)
            finish(event.data.result || null, event.data.error || null)
        }
        engine.onerror = (event) => {
          event.preventDefault()
          finish(null, {
            code: '#ENGINE!',
            message: 'Moteur de calcul indisponible.',
          })
          engine.terminate()
        }
        const timer = setTimeout(() => {
          finish(null, {
            code: '#LIMIT!',
            message: 'Calcul trop long, interrompu après trois secondes.',
          })
          engine.terminate()
        }, 3000)
        pending.current = { id, resolve, reject, timer }
        engine.postMessage({ ...payload, id })
      })
    },
    [stop],
  )
  const clear = useCallback(() => {
    stop()
    setState({ running: false, result: null, error: null })
  }, [stop])
  return { ...state, run, clear }
}
