import { useEffect } from 'react'

/** The tab's title: what the page is about, then the app (WCAG 2.4.2). Parts left empty are skipped. */
export function usePageTitle(...parts: (string | null | undefined)[]) {
  const title = [...parts.filter(Boolean), 'LibreQuenda'].join(' · ')
  useEffect(() => {
    document.title = title
  }, [title])
}
