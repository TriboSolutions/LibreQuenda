import { ChevronRight, Plus } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { CouncilFormDialog } from '@/components/admin/CouncilFormDialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api } from '@/lib/api'

export function CouncilsPanel() {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const councils = useAsync(() => api.admin.councils(token), 'admin:councils')
  const [creating, setCreating] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)

  const list = councils.data?.tenants ?? []

  return (
    <section aria-labelledby="councils-panel-title" className="rounded-lg border bg-card p-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-baseline gap-3">
          <h2 id="councils-panel-title" className="text-lg font-semibold">
            {t('admin:tabs.councils')}
          </h2>
          {councils.data && <span className="num text-muted-foreground">{t('admin:councils.count', { count: list.length })}</span>}
        </div>
        <Button onClick={() => setCreating(true)}>
          <Plus />
          {t('admin:councils.new')}
        </Button>
      </div>

      <div aria-live="polite">
        {notice && (
          <Alert className="mt-4 border-0 bg-pill-positive-bg text-pill-positive-ink">
            <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
          </Alert>
        )}
      </div>

      {councils.error ? (
        <div className="mt-4 flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
          <p>{t('admin:councils.loadError')}</p>
          <Button variant="ghost" size="sm" onClick={councils.reload}>
            {t('common:retry')}
          </Button>
        </div>
      ) : councils.loading && !councils.data ? (
        <div className="mt-4 grid gap-3" aria-busy="true">
          {Array.from({ length: 3 }, (_, i) => (
            <Skeleton key={i} className="h-5 w-full" />
          ))}
        </div>
      ) : list.length === 0 ? (
        <p className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('admin:councils.empty')}</p>
      ) : (
        <div className="mt-3 overflow-x-auto">
          <table className="w-full min-w-[640px]">
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th scope="col" className="py-2 text-left font-normal">
                  {t('admin:councils.colName')}
                </th>
                <th scope="col" className="py-2 pl-4 text-right font-normal">
                  {t('admin:councils.colFacilities')}
                </th>
                <th scope="col" className="py-2 pl-4 text-right font-normal">
                  {t('admin:councils.colAdmins')}
                </th>
                <th scope="col" className="py-2 pl-4 text-right font-normal">
                  {t('admin:councils.colBookings')}
                </th>
                <th scope="col" className="py-2 pl-4 text-left font-normal">
                  {t('admin:councils.colStatus')}
                </th>
                <th scope="col" className="py-2">
                  <span className="sr-only">{t('admin:councils.manage')}</span>
                </th>
              </tr>
            </thead>
            <tbody className="num">
              {list.map((council) => (
                <tr key={council.slug} className="border-t">
                  <th scope="row" className="py-3 text-left font-normal">
                    <span className="flex items-center gap-3">
                      {council.logo_url ? (
                        <img src={council.logo_url} alt="" className="size-8 shrink-0 object-contain" />
                      ) : (
                        <span className="size-8 shrink-0 rounded-md bg-muted" aria-hidden="true" />
                      )}
                      <span>
                        <span className="block font-medium">{council.name}</span>
                        <span className="block text-muted-foreground">/{council.slug}</span>
                      </span>
                    </span>
                  </th>
                  <td className="py-3 pl-4 text-right">{council.facility_count}</td>
                  <td className="py-3 pl-4 text-right whitespace-nowrap">
                    {council.admin_count}
                    {council.pending_invitation_count > 0 && (
                      <span className="block text-xs text-muted-foreground">
                        {t('admin:councils.pending', { count: council.pending_invitation_count })}
                      </span>
                    )}
                  </td>
                  <td className="py-3 pl-4 text-right">{council.booking_count}</td>
                  <td className="py-3 pl-4">
                    <span
                      className={
                        council.is_active
                          ? 'rounded-full bg-pill-positive-bg px-2 py-0.5 text-xs font-medium whitespace-nowrap text-pill-positive-ink'
                          : 'rounded-full bg-pill-neutral-bg px-2 py-0.5 text-xs font-medium whitespace-nowrap text-pill-neutral-ink'
                      }
                    >
                      {council.is_active ? t('admin:councils.active') : t('admin:councils.inactive')}
                    </span>
                  </td>
                  <td className="py-3 pl-4 text-right">
                    <Button asChild variant="ghost" size="sm">
                      <Link to={`/${council.slug}/admin/axustes`} aria-label={`${t('admin:councils.manage')}: ${council.name}`}>
                        {t('admin:councils.manage')}
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

      {creating && (
        <CouncilFormDialog
          council={null}
          onClose={() => setCreating(false)}
          onSaved={(saved, adminEmail) => {
            setCreating(false)
            setNotice(t('admin:notice.created', { name: saved.name, email: adminEmail }))
            councils.reload()
          }}
        />
      )}
    </section>
  )
}
