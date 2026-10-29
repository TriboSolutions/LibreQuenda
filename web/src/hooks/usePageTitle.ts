import { useEffect } from 'react'
import { useBranding } from '@/hooks/useBranding'

/** The tab's title: what the page is about, then the app's name (WCAG 2.4.2). Parts left empty are skipped. */
export function usePageTitle(...parts: (string | null | undefined)[]) {
  const { branding } = useBranding()
  const title = [...parts.filter(Boolean), branding.name].join(' · ')
  useEffect(() => {
    document.title = title
  }, [title])
}
