import type { ReactNode } from 'react'
import { useAuth } from 'react-oidc-context'
import { useAsync } from '@/hooks/useAsync'
import { MeContext } from '@/hooks/useMe'
import { api } from '@/lib/api'

/**
 * GET /me once per signed-in user: it creates the user's row, accepts pending
 * invitations and returns the roles every screen needs.
 */
export function MeProvider({ children }: { children: ReactNode }) {
  const auth = useAuth()
  const token = auth.user?.access_token
  const subject = auth.user?.profile.sub ?? null
  const me = useAsync(token && subject ? () => api.me(token) : null, `me:${subject}`)

  return <MeContext value={{ me: me.data, loading: me.loading, error: me.error, reload: me.reload }}>{children}</MeContext>
}
