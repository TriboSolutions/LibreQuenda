import { useEffect, type ReactNode } from 'react'
import { useAsync } from '@/hooks/useAsync'
import { BrandingContext, DEFAULT_BRANDING } from '@/hooks/useBranding'
import { api } from '@/lib/api'

/** Swaps the tab icons for the installation's logo, and puts LibreQuenda's back when it is removed. */
function useFavicon(logoUrl: string) {
  useEffect(() => {
    const links = document.querySelectorAll<HTMLLinkElement>('link[rel="icon"], link[rel="apple-touch-icon"]')
    for (const link of links) {
      link.dataset.default ??= link.href
      link.href = logoUrl || link.dataset.default
      if (link.type) {
        link.dataset.defaultType ??= link.type
        link.type = logoUrl ? '' : link.dataset.defaultType
      }
    }
  }, [logoUrl])
}

/** GET /branding once: every page shows the installation's name and logo (LibreQuenda's until set, or if it fails). */
export function BrandingProvider({ children }: { children: ReactNode }) {
  const state = useAsync(() => api.branding(), 'branding')
  const branding = state.data ?? DEFAULT_BRANDING
  useFavicon(branding.logo_url)
  return <BrandingContext value={{ branding, reload: state.reload }}>{children}</BrandingContext>
}
