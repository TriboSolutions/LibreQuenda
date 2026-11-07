import { useCallback, useEffect, useRef, useState } from 'react'

type State<T> = { data: T | null; error: Error | null; loading: boolean }

/** How long a list's filters wait to settle before asking the API (ticking several sports, typing). */
export const FILTER_DEBOUNCE_MS = 350

/**
 * Runs `load` whenever `key` changes; `reload()` refetches and keeps the old data on screen meanwhile.
 * debounceMs: a new key waits that long for the next one (filters changed in a row ask the API
 * once, for the last); the first load and reloads go at once.
 */
export function useAsync<T>(load: (() => Promise<T>) | null, key: string, { debounceMs = 0 }: { debounceMs?: number } = {}) {
  const [state, setState] = useState<State<T>>({ data: null, error: null, loading: load !== null })
  const [version, setVersion] = useState(0)
  const first = useRef(true)

  useEffect(() => {
    if (!load) {
      setState({ data: null, error: null, loading: false })
      return
    }
    let active = true
    setState((s) => ({ ...s, loading: true, error: null }))
    const run = () =>
      load()
        .then((data) => active && setState({ data, error: null, loading: false }))
        .catch((error: Error) => active && setState((s) => ({ ...s, error, loading: false })))
    const wait = first.current ? 0 : debounceMs
    first.current = false
    const timer = wait > 0 ? setTimeout(run, wait) : (void run(), undefined)
    return () => {
      active = false
      if (timer !== undefined) clearTimeout(timer)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- `key` identifies the request
  }, [key])

  // a reload goes at once
  useEffect(() => {
    if (version === 0 || !load) return
    let active = true
    setState((s) => ({ ...s, loading: true, error: null }))
    load()
      .then((data) => active && setState({ data, error: null, loading: false }))
      .catch((error: Error) => active && setState((s) => ({ ...s, error, loading: false })))
    return () => {
      active = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- a reload asks again for the same key
  }, [version])

  const reload = useCallback(() => setVersion((v) => v + 1), [])
  return { ...state, reload }
}
