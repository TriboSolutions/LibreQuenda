import { useCallback, useEffect, useState } from 'react'

type State<T> = { data: T | null; error: Error | null; loading: boolean }

/** Runs `load` whenever `key` changes; `reload()` refetches and keeps the old data on screen meanwhile. */
export function useAsync<T>(load: (() => Promise<T>) | null, key: string) {
  const [state, setState] = useState<State<T>>({ data: null, error: null, loading: load !== null })
  const [version, setVersion] = useState(0)

  useEffect(() => {
    if (!load) {
      setState({ data: null, error: null, loading: false })
      return
    }
    let active = true
    setState((s) => ({ ...s, loading: true, error: null }))
    load()
      .then((data) => active && setState({ data, error: null, loading: false }))
      .catch((error: Error) => active && setState((s) => ({ ...s, error, loading: false })))
    return () => {
      active = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- `key` identifies the request
  }, [key, version])

  const reload = useCallback(() => setVersion((v) => v + 1), [])
  return { ...state, reload }
}
