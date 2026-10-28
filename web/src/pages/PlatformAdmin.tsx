import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { CouncilsPanel } from '@/components/admin/CouncilsPanel'
import { SuperadminsPanel } from '@/components/admin/SuperadminsPanel'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

/** /admin: the superadministrator's area (councils and superadministrators). */
export function PlatformAdmin() {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const { me, loading } = useMe()
  const { login } = useLogin()
  usePageTitle(t('admin:title'))

  let body
  if (auth.isLoading || (auth.isAuthenticated && (loading || !me))) {
    body = <Skeleton className="mt-6 h-40 w-full" />
  } else if (!auth.isAuthenticated) {
    body = (
      <div className="mt-6 grid max-w-[760px] gap-4 rounded-lg border bg-card p-5">
        <p>{t('admin:guard.signIn')}</p>
        <div>
          <Button onClick={login}>{t('common:signIn')}</Button>
        </div>
      </div>
    )
  } else if (!me?.is_superadmin) {
    body = (
      <div className="mt-6 grid max-w-[760px] gap-3 rounded-lg border bg-card p-5">
        <h2 className="text-lg font-semibold">{t('admin:guard.forbiddenTitle')}</h2>
        <p className="text-muted-foreground">{t('admin:guard.forbiddenBody')}</p>
        <div>
          <Button asChild variant="outline">
            <Link to="/">{t('admin:guard.back')}</Link>
          </Button>
        </div>
      </div>
    )
  } else {
    body = (
      <Tabs defaultValue="councils" className="mt-6 gap-6">
        <TabsList>
          <TabsTrigger value="councils">{t('admin:tabs.councils')}</TabsTrigger>
          <TabsTrigger value="superadmins">{t('admin:tabs.superadmins')}</TabsTrigger>
        </TabsList>
        <TabsContent value="councils">
          <CouncilsPanel />
        </TabsContent>
        <TabsContent value="superadmins">
          <SuperadminsPanel />
        </TabsContent>
      </Tabs>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar tenants={[]} tenantSlug={null} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('admin:title')}</h1>
        <p className="mt-1 text-base text-muted-foreground">{t('admin:subtitle')}</p>
        {body}
      </main>
      <SiteFooter />
    </div>
  )
}
