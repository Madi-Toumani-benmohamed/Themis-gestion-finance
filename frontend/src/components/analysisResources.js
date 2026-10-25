import { useEffect, useState } from 'react'

// Only the active detail is mounted. Every request is scoped and stale results
// are rejected even when a server responds after cancellation.
export function useAnalysisResource(loader, dependencies) {
  const [attempt, setAttempt] = useState(0)
  const requestKey = JSON.stringify([...dependencies, attempt])
  const [state, setState] = useState({
    key: requestKey,
    loading: true,
    data: null,
    error: null,
  })
  useEffect(() => {
    const controller = new AbortController()
    let active = true
    setState({ key: requestKey, loading: true, data: null, error: null })
    Promise.resolve()
      .then(() => loader(controller.signal))
      .then((data) => {
        if (active)
          setState({ key: requestKey, loading: false, data, error: null })
      })
      .catch((error) => {
        if (active && error.name !== 'AbortError')
          setState({
            key: requestKey,
            loading: false,
            data: null,
            error: error.message || 'Analyse indisponible.',
          })
      })
    return () => {
      active = false
      controller.abort()
    }
  }, [...dependencies, attempt])
  return {
    ...(state.key === requestKey
      ? state
      : { loading: true, data: null, error: null }),
    retry: () => setAttempt((value) => value + 1),
  }
}
