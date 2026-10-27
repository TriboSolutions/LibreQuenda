import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { useLocation } from 'react-router'
import { Button } from '@/components/ui/button'
import { useMe } from '@/hooks/useMe'

/**
 * Signs in again: the login (EMAIL_VERIFICATION) stops at "enter the code we emailed
 * you" and then comes back to this page, where GET /me grants the waiting role.
 */
export function useVerifyEmail() {
  const auth = useAuth()
  const location = useLocation()
  return () => void auth.signinRedirect({ prompt: 'login', state: { returnTo: `${location.pathname}${location.search}` } })
}

/**
 * Shown while a council role waits for this person: roles only follow a verified email
 * (anyone could register with someone else's address), so the email must be verified first.
 */
export function InvitationReminder() {
  const { t } = useTranslation('admin')
  const { me } = useMe()
  const verify = useVerifyEmail()
  const pending = me?.pending_invitations ?? []
  if (!me || me.email_verified || pending.length === 0) return null

  return (
    <div role="status" className="border-b bg-pill-accent-bg text-pill-accent-ink">
      <div className="mx-auto flex max-w-[1152px] flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-2.5 sm:px-6">
        <p>
          <span className="font-medium">{t('invitation.bannerTitle', { council: pending.map((p) => p.tenant_name).join(', ') })}</span>{' '}
          {t('invitation.bannerBody', { email: me.email })}
        </p>
        <Button size="sm" onClick={verify}>
          {t('invitation.verify')}
        </Button>
      </div>
    </div>
  )
}
