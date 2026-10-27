import { WebStorageStateStore } from 'oidc-client-ts'
import type { ReactNode } from 'react'
import { AuthProvider } from 'react-oidc-context'
import { useNavigate } from 'react-router'
import type { RuntimeConfig } from '@/config'

// The session is shared by every tab: a link from one of our emails opens in a new
// tab, and with the default (sessionStorage, one per tab) it would start signed out.
const userStore = new WebStorageStateStore({ store: window.localStorage })

/** Inside the router so the login can return to the page it started from (useLogin passes returnTo). */
export function AuthWithRouter({ config, children }: { config: RuntimeConfig; children: ReactNode }) {
  const navigate = useNavigate()

  return (
    <AuthProvider
      authority={config.authority}
      client_id={config.clientId}
      redirect_uri={`${window.location.origin}/callback`}
      post_logout_redirect_uri={`${window.location.origin}/`}
      scope={config.scope}
      userStore={userStore}
      onSigninCallback={(user) => {
        const returnTo = (user?.state as { returnTo?: string } | undefined)?.returnTo
        // only paths inside this app: never an external or protocol-relative URL
        navigate(returnTo?.startsWith('/') && !returnTo.startsWith('//') ? returnTo : '/', { replace: true })
      }}
    >
      {children}
    </AuthProvider>
  )
}
