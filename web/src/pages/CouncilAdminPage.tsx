import { ChevronRight, Plus, Settings } from 'lucide-react'
import { lazy, Suspense, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router'
import { StaffBookingsPanel } from '@/components/admin/StaffBookingsPanel'
import { CouncilSheet } from '@/components/admin/CouncilSheet'
import { useVerifyEmail } from '@/hooks/useVerifyEmail'
import { CouncilHeading } from '@/components/CouncilHeading'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useAsync } from '@/hooks/useAsync'
import { useCouncilAdmin } from '@/hooks/useCouncilAdmin'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { api } from '@/lib/api'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

// charts (Recharts) load only when an administrator opens the indicators
const IndicatorsPanel = lazy(() => import('@/components/admin/IndicatorsPanel').then((m) => ({ default: m.IndicatorsPanel })))

/** /:slug/admin: the council's facilities, manual booking (?vista=reservas&instalacion=<id>) and indicators (?vista=indicadores), as tabs for its administrators; the gear opens the council's settings. */
export function CouncilAdminPage() {
  const { slug = '' } = useParams()
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const { me, loading } = useMe()
  const { login } = useLogin()
  const allowed = useCouncilAdmin(slug)
  // invited here, but the role waits for a verified email
  const invitedHere = me?.email_verified ? undefined : me?.pending_invitations?.find((p) => p.tenant_slug === slug)
  const verify = useVerifyEmail()
  const token = auth.user?.access_token ?? ''
  const tenants = useAsync(() => api.tenants(), 'tenants')
  usePageTitle(t('admin:councilAdmin.title'), tenants.data?.tenants.find((c) => c.slug === slug)?.name)
  const facilities = useAsync(allowed ? () => api.council.facilities(token, slug) : null, `council-admin:${slug}:${allowed}`)
  const council = tenants.data?.tenants.find((c) => c.slug === slug)
  const [searchParams, setSearchParams] = useSearchParams()
  const vista = searchParams.get('vista')
  const view = vista === 'indicadores' ? 'indicators' : vista === 'reservas' ? 'bookings' : 'facilities'
  const viewParam: Record<string, string> = { indicators: 'indicadores', bookings: 'reservas' }
  const changeView = (value: string) => setSearchParams(viewParam[value] ? { vista: viewParam[value] } : {}, { replace: true })
  const bookingsFacility = searchParams.get('instalacion') ?? ''
  const [settingsOpen, setSettingsOpen] = useState(false)
  const navigate = useNavigate()

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
  } else if (!allowed && invitedHere) {
    body = (
      <div className="mt-6 grid max-w-[760px] gap-3 rounded-lg border bg-card p-5">
        <h2 className="text-lg font-semibold">{t('admin:invitation.pageTitle', { council: invitedHere.tenant_name })}</h2>
        <p className="text-muted-foreground">{t('admin:invitation.pageBody', { email: me?.email ?? '' })}</p>
        <div>
          <Button onClick={verify}>{t('admin:invitation.verify')}</Button>
        </div>
      </div>
    )
  } else if (!allowed) {
    body = (
      <div className="mt-6 grid max-w-[760px] gap-3 rounded-lg border bg-card p-5">
        <h2 className="text-lg font-semibold">{t('admin:guard.forbiddenTitle')}</h2>
        <p className="text-muted-foreground">{t('admin:facilities.guardBody')}</p>
        <div>
          <Button asChild variant="outline">
            <Link to={`/${slug}`}>{t('admin:guard.back')}</Link>
          </Button>
        </div>
      </div>
    )
  } else {
    const list = facilities.data?.facilities ?? []
    const facilitiesSection = (
      <section aria-labelledby="facilities-title" className="rounded-lg border bg-card p-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-baseline gap-3">
            <h2 id="facilities-title" className="text-lg font-semibold">
              {t('admin:facilities.title')}
            </h2>
            {facilities.data && <span className="num text-muted-foreground">{t('admin:facilities.count', { count: list.length })}</span>}
          </div>
          <Button asChild>
            <Link to={`/${slug}/admin/facilities/new`}>
              <Plus />
              {t('admin:facilities.new')}
            </Link>
          </Button>
        </div>
        {facilities.error ? (
          <p className="mt-4 text-destructive">{facilities.error.message}</p>
        ) : !facilities.data ? (
          <Skeleton className="mt-4 h-24 w-full" />
        ) : list.length === 0 ? (
          <p className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('admin:facilities.empty')}</p>
        ) : (
          <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {list.map((f) => (
              <li
                key={f.id}
                className="relative flex gap-3 rounded-lg border p-3 transition-colors focus-within:ring-2 focus-within:ring-ring hover:border-primary/60"
              >
                {f.image_url ? (
                  <img src={f.image_url} alt="" className="size-16 shrink-0 rounded-md object-cover" />
                ) : (
                  <span className="size-16 shrink-0 rounded-md bg-muted" aria-hidden="true" />
                )}
                <div className="grid min-w-0 flex-1 content-start gap-1">
                  <h3 className="leading-5 font-semibold">
                    {/* stretched over the card: the whole card opens the editor */}
                    <Link to={`/${slug}/admin/facilities/${f.id}`} className="outline-none after:absolute after:inset-0 after:rounded-lg">
                      {f.name}
                    </Link>
                  </h3>
                  {f.sport && <p className="truncate text-sm text-muted-foreground">{f.sport}</p>}
                  <p className="num text-sm text-muted-foreground">
                    {f.opening_window_count > 0 ? t('admin:facilities.windows', { count: f.opening_window_count }) : t('admin:facilities.noSchedule')}
                    {' · '}
                    {t('admin:facilities.upcoming', { count: f.upcoming_booking_count })}
                  </p>
                  <p>
                    <span
                      className={
                        f.is_active
                          ? 'rounded-full bg-pill-positive-bg px-2 py-0.5 text-xs font-medium whitespace-nowrap text-pill-positive-ink'
                          : 'rounded-full bg-pill-neutral-bg px-2 py-0.5 text-xs font-medium whitespace-nowrap text-pill-neutral-ink'
                      }
                    >
                      {f.is_active ? t('admin:councils.active') : t('admin:councils.inactive')}
                    </span>
                  </p>
                </div>
                <ChevronRight className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
              </li>
            ))}
          </ul>
        )}
      </section>
    )
    const settingsButton = (
      <Button variant="outline" size="icon" onClick={() => setSettingsOpen(true)} aria-label={t('admin:councilAdmin.settings')} title={t('admin:councilAdmin.settings')}>
        <Settings />
      </Button>
    )
    body = (
      <Tabs value={view} onValueChange={changeView} className="mt-6 gap-6">
        <div className="flex items-center justify-between gap-3">
          <TabsList>
            <TabsTrigger value="facilities">{t('admin:facilities.title')}</TabsTrigger>
            <TabsTrigger value="bookings">{t('admin:staff.tab')}</TabsTrigger>
            <TabsTrigger value="indicators">{t('admin:indicators.title')}</TabsTrigger>
          </TabsList>
          {/* the council's settings belong with its facilities and figures, not with the desk's calendar */}
          {view !== 'bookings' && settingsButton}
        </div>
        <TabsContent value="facilities">{facilitiesSection}</TabsContent>
        <TabsContent value="bookings">
          <StaffBookingsPanel
            slug={slug}
            facilities={list}
            facilityId={bookingsFacility}
            onFacilityChange={(id) => setSearchParams({ vista: 'reservas', instalacion: id }, { replace: true })}
          />
        </TabsContent>
        <TabsContent value="indicators">
          <Suspense fallback={<Skeleton className="h-96 w-full" />}>
            <IndicatorsPanel slug={slug} facilities={list} />
          </Suspense>
        </TabsContent>
      </Tabs>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={slug} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <CouncilHeading section={t('admin:councilAdmin.title')} council={council} otherCouncils={false} />
        <p className="mt-1 text-base text-muted-foreground">{t('admin:councilAdmin.subtitle')}</p>
        {body}
      </main>
      <SiteFooter />
      {settingsOpen && allowed && (
        <CouncilSheet
          slug={slug}
          superadmin={me?.is_superadmin ?? false}
          onClose={() => setSettingsOpen(false)}
          onChanged={() => tenants.reload()}
          onDeleted={() => navigate('/admin')}
          onSlugChanged={(next) => navigate(`/${next}/admin${viewParam[view] ? `?vista=${viewParam[view]}` : ''}`, { replace: true })}
        />
      )}
    </div>
  )
}
