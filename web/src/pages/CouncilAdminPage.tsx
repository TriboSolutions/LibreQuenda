import { ChevronRight, Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useParams } from 'react-router'
import { useVerifyEmail } from '@/components/InvitationReminder'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useCouncilAdmin } from '@/hooks/useCouncilAdmin'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { api } from '@/lib/api'

/** /:slug/admin: the council's facilities, for its administrators. */
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
  const facilities = useAsync(allowed ? () => api.council.facilities(token, slug) : null, `council-admin:${slug}:${allowed}`)
  const council = tenants.data?.tenants.find((c) => c.slug === slug)

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
    body = (
      <section aria-labelledby="facilities-title" className="mt-6 rounded-lg border bg-card p-5">
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
          <div className="mt-3 overflow-x-auto">
            <table className="w-full min-w-[620px]">
              <thead>
                <tr className="text-xs text-muted-foreground">
                  <th scope="col" className="py-2 text-left font-normal">
                    {t('admin:facilities.colFacility')}
                  </th>
                  <th scope="col" className="py-2 pl-4 text-right font-normal">
                    {t('admin:facilities.colSchedule')}
                  </th>
                  <th scope="col" className="py-2 pl-4 text-right font-normal">
                    {t('admin:facilities.colUpcoming')}
                  </th>
                  <th scope="col" className="py-2 pl-4 text-left font-normal">
                    {t('admin:councils.colStatus')}
                  </th>
                  <th scope="col" className="py-2">
                    <span className="sr-only">{t('admin:facilities.edit')}</span>
                  </th>
                </tr>
              </thead>
              <tbody className="num">
                {list.map((f) => (
                  <tr key={f.id} className="border-t">
                    <th scope="row" className="py-3 text-left font-normal">
                      <span className="flex items-center gap-3">
                        {f.image_url ? (
                          <img src={f.image_url} alt="" className="h-10 w-14 shrink-0 rounded-sm object-cover" />
                        ) : (
                          <span className="h-10 w-14 shrink-0 rounded-sm bg-muted" aria-hidden="true" />
                        )}
                        <span>
                          <span className="block font-medium">{f.name}</span>
                          <span className="block text-muted-foreground">
                            {f.sport ? `${f.sport} · ` : ''}/{slug}/{f.slug}
                          </span>
                        </span>
                      </span>
                    </th>
                    <td className="py-3 pl-4 text-right">
                      {f.opening_window_count > 0 ? f.opening_window_count : <span className="text-muted-foreground">{t('admin:facilities.noSchedule')}</span>}
                    </td>
                    <td className="py-3 pl-4 text-right">{f.upcoming_booking_count}</td>
                    <td className="py-3 pl-4">
                      <span
                        className={
                          f.is_active
                            ? 'rounded-full bg-pill-positive-bg px-2 py-0.5 text-xs font-medium whitespace-nowrap text-pill-positive-ink'
                            : 'rounded-full bg-pill-neutral-bg px-2 py-0.5 text-xs font-medium whitespace-nowrap text-pill-neutral-ink'
                        }
                      >
                        {f.is_active ? t('admin:councils.active') : t('admin:councils.inactive')}
                      </span>
                    </td>
                    <td className="py-3 pl-4 text-right">
                      <Button asChild variant="ghost" size="sm">
                        <Link to={`/${slug}/admin/facilities/${f.id}`} aria-label={`${t('admin:facilities.edit')}: ${f.name}`}>
                          {t('admin:facilities.edit')}
                          <ChevronRight />
                        </Link>
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={slug} />
      <main className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('admin:facilities.title')}</h1>
        <p className="mt-1 text-base text-muted-foreground">{t('admin:facilities.subtitle', { council: council?.name ?? slug })}</p>
        {body}
      </main>
    </div>
  )
}
