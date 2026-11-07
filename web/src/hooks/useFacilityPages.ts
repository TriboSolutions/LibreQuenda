import { useEffect, useState } from 'react'
import { api, type FacilityCard, type FacilityFilters } from '@/lib/api'

/** value, once it has stayed the same for ms (typing, panning the map). */
export function useDebounced<T>(value: T, ms: number): T {
  const [settled, setSettled] = useState(value)
  const key = JSON.stringify(value)
  useEffect(() => {
    const timer = setTimeout(() => setSettled(value), ms)
    return () => clearTimeout(timer)
    // the value's content, not its identity: a new object with the same numbers is the same view
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, ms])
  return settled
}

const PAGE_SIZE = 20

/** The box that scrolls the node: its nearest ancestor that scrolls, or the page (null). */
function scrollParent(node: HTMLElement): HTMLElement | null {
  for (let el = node.parentElement; el; el = el.parentElement) {
    const overflow = getComputedStyle(el).overflowY
    if (overflow === 'auto' || overflow === 'scroll') return el
  }
  return null
}

type Shown = { items: FacilityCard[]; total: number }

type Pages = {
  key: string
  items: FacilityCard[]
  total: number | null
  hasMore: boolean
  error: Error | null
  wanted: number
  attempt: number
  // what the last filters found: shown, dimmed, while the new ones load, so the list never empties and flashes
  previous: Shown | null
}

/**
 * The home page's list, a page at a time: the first page whenever the filters
 * change (key), the next one when the sentinel (put after the last row) comes
 * into view. total: the API's count of every match; null until it answers.
 */
export function useFacilityPages(filters: FacilityFilters, key: string) {
  const [state, setState] = useState<Pages>({ key, items: [], total: null, hasMore: false, error: null, wanted: 1, attempt: 0, previous: null })
  const [node, setNode] = useState<HTMLElement | null>(null)
  // new filters: the first page again, the last results on screen until it comes
  const current: Pages =
    state.key === key
      ? state
      : { key, items: [], total: null, hasMore: false, error: null, wanted: 1, attempt: 0, previous: state.total !== null ? { items: state.items, total: state.total } : state.previous }
  if (current !== state) setState(current)
  const loaded = Math.ceil(current.items.length / PAGE_SIZE)
  const page = current.wanted

  // fetch the page wanted, once
  useEffect(() => {
    if (page <= loaded && current.total !== null) return // already here
    let stale = false
    api
      .facilities(filters, page, PAGE_SIZE)
      .then((answer) => {
        if (stale) return
        setState((s) =>
          s.key !== key ? s : { ...s, items: page === 1 ? answer.facilities : [...s.items, ...answer.facilities], total: answer.pagination.total, hasMore: answer.pagination.has_more, error: null },
        )
      })
      .catch((e: unknown) => {
        if (!stale) setState((s) => (s.key !== key ? s : { ...s, error: e instanceof Error ? e : new Error(String(e)) }))
      })
    return () => {
      stale = true
    }
    // the filters are in key; page and key say what to fetch
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, page, current.attempt])

  // the next page as the sentinel after the last row comes into view
  const ready = current.hasMore && loaded >= page
  useEffect(() => {
    if (!node || !ready) return
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) setState((s) => (s.key === key ? { ...s, wanted: Math.max(s.wanted, Math.ceil(s.items.length / PAGE_SIZE) + 1) } : s))
      },
      // within the box that scrolls (on a desktop the list scrolls inside), well ahead of the end
      { root: scrollParent(node), rootMargin: '600px' },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [node, ready, key])

  // the new filters' first page has not come yet: the last results, marked as refreshing
  const refreshing = current.total === null && current.previous !== null
  return {
    items: refreshing ? current.previous!.items : current.items,
    total: refreshing ? current.previous!.total : current.total,
    refreshing,
    hasMore: !refreshing && current.hasMore,
    error: current.error,
    loading: page > loaded,
    sentinel: setNode,
    // after an error: the same page again
    retry: () => setState((s) => ({ ...s, error: null, attempt: s.attempt + 1 })),
  }
}
