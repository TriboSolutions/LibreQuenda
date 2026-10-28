import { useAuth } from 'react-oidc-context'
import { useLocation } from 'react-router'

/**
 * Signs in again: the login (EMAIL_VERIFICATION) stops at "enter the code we emailed
 * you" and then comes back to this page, where GET /me grants the waiting role.
 */
export function useVerifyEmail() {
  const auth = useAuth()
  const location = useLocation()
  return () => void auth.signinRedirect({ prompt: 'login', state: { returnTo: `${location.pathname}${location.search}` } })
}
