import { useLocation } from 'react-router'
import { useAuth } from 'react-oidc-context'

/** Sign in or register, coming back afterwards to the page the resident was on. */
export function useLogin() {
  const auth = useAuth()
  const location = useLocation()
  const state = { returnTo: `${location.pathname}${location.search}` }

  return {
    login: () => void auth.signinRedirect({ state }),
    register: () => void auth.signinRedirect({ prompt: 'create', state }),
  }
}
