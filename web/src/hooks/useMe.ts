import { createContext, useContext } from 'react'
import type { Me } from '@/lib/api'

export type MeState = { me: Me | null; loading: boolean; error: Error | null; reload: () => void }

export const MeContext = createContext<MeState>({ me: null, loading: false, error: null, reload: () => {} })

/** The signed-in user's row and roles, loaded once by MeProvider. */
export function useMe() {
  return useContext(MeContext)
}
