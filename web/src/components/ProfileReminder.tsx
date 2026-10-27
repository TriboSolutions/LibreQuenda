import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useLocation } from 'react-router'
import { Button } from '@/components/ui/button'
import { useMe } from '@/hooks/useMe'

/** Shown under the top bar until the signed-in user adds the date of birth that booking requires. */
export function ProfileReminder() {
  const { t } = useTranslation('profile')
  const auth = useAuth()
  const { me } = useMe()
  const { pathname } = useLocation()
  if (!auth.isAuthenticated || !me || me.profile_complete || pathname === '/perfil') return null

  return (
    <div role="status" className="border-b bg-pill-accent-bg text-pill-accent-ink">
      <div className="mx-auto flex max-w-[1152px] flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-2.5 sm:px-6">
        <p>
          <span className="font-medium">{t('incompleteTitle')}.</span> {t('incompleteBody')}
        </p>
        <Button asChild size="sm">
          <Link to="/perfil">{t('incompleteAction')}</Link>
        </Button>
      </div>
    </div>
  )
}
