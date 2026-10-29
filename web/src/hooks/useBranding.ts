import { createContext, useContext } from 'react'
import type { Branding } from '@/lib/api'

export const DEFAULT_APP_NAME = 'LibreQuenda'

export const DEFAULT_BRANDING: Branding = { name: DEFAULT_APP_NAME, custom_name: '', logo_ref: '', logo_url: '' }

export type BrandingState = { branding: Branding; reload: () => void }

export const BrandingContext = createContext<BrandingState>({ branding: DEFAULT_BRANDING, reload: () => {} })

/** The installation's name and logo, loaded once by BrandingProvider; LibreQuenda's until they arrive. */
export function useBranding() {
  return useContext(BrandingContext)
}
