import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { ShieldAlert } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useMe } from '@/hooks/useMe'
import { MFA_REQUIRED_EVENT } from '@/lib/api'

/**
 * Under the top bar once the administration refuses a sign-in without a second
 * factor: set one up in the sign-in account (Zitadel's own page), then sign in
 * again with it, since the current sign-in stays password-only.
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
  if (!shown || !auth.isAuthenticated) return null

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
            <a href={setUp}>{t('mfa.setUp')}</a>
          </Button>
          <Button size="sm" variant="outline" onClick={() => void auth.signinRedirect({ prompt: 'login', state: { returnTo: window.location.pathname + window.location.search } })}>
            {t('mfa.signInAgain')}
          </Button>
        </div>
      </div>
    </div>
  )
}
