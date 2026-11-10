import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useMe } from '@/hooks/useMe'
import { MFA_REQUIRED_EVENT } from '@/lib/api'

/** The page that sent the person to set up a second factor, kept for the way back. */
const SETUP_PENDING = 'lq:mfa-setup-return'
/** Older than this, the setup was abandoned: an ordinary visit doesn't sign in again. */
const SETUP_PENDING_MS = 30 * 60 * 1000

function readPending(): { returnTo: string; at: number } | null {
  try {
    const raw = sessionStorage.getItem(SETUP_PENDING)
    sessionStorage.removeItem(SETUP_PENDING)
    const pending = raw ? (JSON.parse(raw) as { returnTo?: unknown; at?: unknown }) : null
    if (!pending || typeof pending.returnTo !== 'string' || typeof pending.at !== 'number') return null
    return Date.now() - pending.at < SETUP_PENDING_MS ? { returnTo: pending.returnTo, at: pending.at } : null
  } catch {
    return null
  }
}

/**
 * Under the top bar once the administration refuses a sign-in without a second
 * factor: set one up in the sign-in account (Zitadel's own page), then sign in
 * again with it, since the current sign-in stays password-only.
 *
 * Zitadel's setup page runs outside a sign-in from this app, so it ends by coming
 * back here (DEFAULT_REDIRECT_URI on zitadel-login); the sign-in with the new
 * factor then starts on its own and returns to the page the notice was on.
 */
export function MfaRequiredNotice() {
  const { t } = useTranslation('common')
  const auth = useAuth()
  const { me } = useMe()
  const [shown, setShown] = useState(false)
  useEffect(() => {
    const show = () => setShown(true)
    window.addEventListener(MFA_REQUIRED_EVENT, show)
    return () => window.removeEventListener(MFA_REQUIRED_EVENT, show)
  }, [])
  const signInAgain = auth.signinRedirect
  useEffect(() => {
    if (auth.isLoading) return
    const pending = readPending()
    if (pending) void signInAgain({ prompt: 'login', state: { returnTo: pending.returnTo } })
  }, [auth.isLoading, signInAgain])
  if (!shown || !auth.isAuthenticated) return null

  const returnTo = window.location.pathname + window.location.search
  const rememberReturn = () => {
    try {
      sessionStorage.setItem(SETUP_PENDING, JSON.stringify({ returnTo, at: Date.now() }))
    } catch {
      // no storage: the person signs in again from the notice
    }
  }

  const setUp = `${auth.settings.authority}/ui/v2/login/mfa/set${me ? `?loginName=${encodeURIComponent(me.email)}` : ''}`
  return (
    <div role="alert" className="border-b bg-pill-negative-bg text-pill-negative-ink">
      <div className="mx-auto grid max-w-[1152px] gap-2 px-4 py-3 sm:px-6">
        <p className="flex items-start gap-2">
          <ShieldAlert className="mt-0.5 size-4 shrink-0" aria-hidden />
          <span>
            <span className="font-medium">{t('mfa.title')}</span> {t('mfa.body')}
          </span>
        </p>
        <div className="flex flex-wrap gap-2">
          <Button asChild size="sm">
            <a href={setUp} onClick={rememberReturn}>
              {t('mfa.setUp')}
            </a>
          </Button>
          <Button size="sm" variant="outline" onClick={() => void auth.signinRedirect({ prompt: 'login', state: { returnTo } })}>
            {t('mfa.signInAgain')}
          </Button>
        </div>
      </div>
    </div>
  )
}
