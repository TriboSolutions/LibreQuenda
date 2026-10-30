import { WebStorageStateStore } from 'oidc-client-ts'
import { useEffect, type ReactNode } from 'react'
import { AuthProvider, useAuth } from 'react-oidc-context'
import { useNavigate } from 'react-router'
import type { RuntimeConfig } from '@/config'
import { onTokenExpired } from '@/lib/api'

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
      // access tokens are short (Zitadel, 10 minutes): renewed with the refresh token before they expire
      automaticSilentRenew
      onSigninCallback={(user) => {
        const returnTo = (user?.state as { returnTo?: string } | undefined)?.returnTo
        // only paths inside this app: never an external or protocol-relative URL
        navigate(returnTo?.startsWith('/') && !returnTo.startsWith('//') ? returnTo : '/', { replace: true })
      }}
    >
      <TokenRenewal />
      {children}
    </AuthProvider>
  )
}

/** Lets API calls renew an access token that expired just before them (lib/api request). */
function TokenRenewal() {
  const auth = useAuth()
  useEffect(() => {
    onTokenExpired(async () => {
      try {
        return (await auth.signinSilent())?.access_token ?? null
      } catch {
        return null // the refresh token is gone too: the person signs in again
      }
    })
    return () => onTokenExpired(null)
  }, [auth])
  return null
}
