import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useLocation, useNavigate } from 'react-router'
import { BookingPolicyPanel } from '@/components/admin/BookingPolicyPanel'
import { AISettingsPanel } from '@/components/admin/AISettingsPanel'
import { SiteFooter } from '@/components/SiteFooter'
import { TopBar } from '@/components/TopBar'
import { Skeleton } from '@/components/ui/skeleton'
import { useMe } from '@/hooks/useMe'
import { usePageTitle } from '@/hooks/usePageTitle'

/** /admin/axustes: the superadministrators' settings for the whole platform, laid out like a council's settings. */
export function PlatformSettingsPage() {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const { me, loading } = useMe()
  const navigate = useNavigate()
  const location = useLocation()
  usePageTitle(t('admin:platformSettings.title'))

  let body
  if (auth.isLoading || (auth.isAuthenticated && (loading || !me))) {
    body = <Skeleton className="mt-6 h-40 w-full" />
  } else if (!me?.is_superadmin) {
    body = (
      <div className="mt-6 grid max-w-[760px] gap-3 rounded-lg border bg-card p-5">
        <h2 className="text-lg font-semibold">{t('admin:guard.forbiddenTitle')}</h2>
        <p className="text-muted-foreground">{t('admin:guard.forbiddenBody')}</p>
      </div>
    )
  } else {
    body = (
      <div className="mt-6 grid gap-10">
        <BookingPolicyPanel />
        <AISettingsPanel />
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <p>
          {/* back to where they came from, like the browser's back; straight from a link, to the administration */}
          <Link
            to="/admin"
            onClick={(event) => {
              if (location.key === 'default') return
              event.preventDefault()
              navigate(-1)
            }}
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="size-4" aria-hidden />
            {t('admin:platformSettings.back')}
          </Link>
        </p>
        <h1 className="mt-3 text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('admin:platformSettings.title')}</h1>
        {body}
      </main>
      <SiteFooter />
    </div>
  )
}
