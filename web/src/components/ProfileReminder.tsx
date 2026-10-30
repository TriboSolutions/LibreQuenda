import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useLocation } from 'react-router'
import { Button } from '@/components/ui/button'
import { useMe } from '@/hooks/useMe'

/** Shown under the top bar until the signed-in user has what booking requires: a date of birth and a verified email. */
export function ProfileReminder() {
  const { t } = useTranslation('profile')
  const auth = useAuth()
  const { me } = useMe()
  const { pathname } = useLocation()
  if (!auth.isAuthenticated || !me || (me.profile_complete && me.email_verified) || pathname === '/perfil') return null
  // the birth date first; then the email, which booking also needs verified
  const [title, body, action] = me.profile_complete
    ? [t('unverifiedTitle'), t('unverifiedBody'), t('unverifiedAction')]
    : [t('incompleteTitle'), t('incompleteBody'), t('incompleteAction')]

  return (
    <div role="status" className="border-b bg-pill-accent-bg text-pill-accent-ink">
      <div className="mx-auto flex max-w-[1152px] flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-2.5 sm:px-6">
        <p>
          <span className="font-medium">{title}.</span> {body}
        </p>
        <Button asChild size="sm">
          <Link to="/perfil">{action}</Link>
        </Button>
      </div>
    </div>
  )
}
