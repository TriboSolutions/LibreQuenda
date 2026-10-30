import { ArrowLeft, ExternalLink } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useNavigate, useParams } from 'react-router'
import { BookingRulesSettings } from '@/components/admin/BookingRulesSettings'
import { CouncilExport } from '@/components/admin/CouncilExport'
import { CouncilFormDialog } from '@/components/admin/CouncilFormDialog'
import { SmtpSettings } from '@/components/admin/SmtpSettings'
import { CouncilHeading } from '@/components/CouncilHeading'
import { SiteFooter } from '@/components/SiteFooter'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useCouncilAdmin } from '@/hooks/useCouncilAdmin'
import { useMe } from '@/hooks/useMe'
import { usePageTitle } from '@/hooks/usePageTitle'
import { api } from '@/lib/api'

/**
 * /:slug/admin/axustes: everything about one council, like a facility's editor:
 * its details, administrators, booking window, email relay, data export, and
 * (superadministrators) deleting it.
 */
export function CouncilSettingsPage() {
  const { slug = '' } = useParams()
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const { me } = useMe()
  const navigate = useNavigate()
  const allowed = useCouncilAdmin(slug)
  // a superadministrator also switches the council on or off and deletes it
  const superadmin = me?.is_superadmin ?? false
  const token = auth.user?.access_token ?? ''
  const tenants = useAsync(() => api.tenants(), 'tenants')
  const council = useAsync(allowed ? () => api.admin.council(token, slug) : null, `council:${slug}:${allowed}`)
  const [editing, setEditing] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const heading = tenants.data?.tenants.find((c) => c.slug === slug)
  usePageTitle(t('admin:councilSettings.title'), heading?.name)

  const run = async (action: () => Promise<unknown>, message: string) => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await action()
      council.reload()
      setNotice(message)
      return true
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return false
    } finally {
      setBusy(false)
    }
  }

  const invite = async (event: React.FormEvent) => {
    event.preventDefault()
    const email = inviteEmail.trim()
    if (await run(() => api.admin.invite(token, slug, email), t('admin:notice.invited', { email }))) setInviteEmail('')
  }

  const remove = async () => {
    setBusy(true)
    setError(null)
    try {
      await api.admin.deleteCouncil(token, slug)
      navigate('/admin')
    } catch (e) {
      setConfirmDelete(false)
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const data = council.data

  let body
  if (!allowed) {
    body = auth.isLoading || !me ? (
      <Skeleton className="mt-6 h-40 w-full" />
    ) : (
      <div className="mt-6 grid max-w-[760px] gap-3 rounded-lg border bg-card p-5">
        <h2 className="text-lg font-semibold">{t('admin:guard.forbiddenTitle')}</h2>
        <p className="text-muted-foreground">{t('admin:facilities.guardBody')}</p>
      </div>
    )
  } else if (!data) {
    body = council.error ? <p className="mt-6 text-destructive">{council.error.message}</p> : <Skeleton className="mt-6 h-64 w-full" />
  } else {
    body = (
      <>
        <div aria-live="polite" className="mt-6 empty:hidden">
          {notice && (
            <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
              <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
            </Alert>
          )}
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>
        <div className="mt-6 grid gap-6 lg:grid-cols-2 lg:items-start">
          <div className="grid gap-6">
          <section aria-labelledby="sheet-details" className="grid content-start gap-3 rounded-lg border bg-card p-5">
            <div className="flex items-center justify-between gap-3">
              <h2 id="sheet-details" className="text-lg font-semibold">
                {t('admin:manage.details')}
              </h2>
              <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
                {t('admin:manage.edit')}
              </Button>
            </div>
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5">
              <dt className="text-muted-foreground">{t('admin:councils.colStatus')}</dt>
              <dd>{data.is_active ? t('admin:councils.active') : t('admin:councils.inactive')}</dd>
              <dt className="text-muted-foreground">{t('admin:form.timezone')}</dt>
              <dd>{data.timezone}</dd>
              <dt className="text-muted-foreground">{t('admin:form.contactEmail')}</dt>
              <dd className="truncate">{data.contact_email || '—'}</dd>
              <dt className="text-muted-foreground">{t('admin:form.contactPhone')}</dt>
              <dd>{data.contact_phone || '—'}</dd>
              <dt className="text-muted-foreground">{t('admin:councils.colBookings')}</dt>
              <dd className="num">{data.booking_count}</dd>
            </dl>
            <div className="flex flex-wrap gap-x-5 gap-y-1">
              {superadmin && (
                <Link to={`/${slug}/admin`} className="inline-flex items-center gap-1.5 text-primary hover:underline">
                  {t('admin:manage.facilities')}
                </Link>
              )}
              {data.is_active && (
                <Link target="_blank" rel="noopener noreferrer" to={`/${slug}`} className="inline-flex items-center gap-1.5 text-primary hover:underline">
                  {t('admin:manage.viewPage')}
                  <ExternalLink className="size-3.5" aria-hidden="true" />
                </Link>
              )}
            </div>
          </section>

          <section aria-labelledby="sheet-admins" className="grid content-start gap-3 rounded-lg border bg-card p-5">
            <h2 id="sheet-admins" className="text-lg font-semibold">
              {t('admin:manage.admins')}
            </h2>
            {data.admins.length === 0 && data.invitations.length === 0 ? (
              <p className="text-muted-foreground">{t('admin:manage.noAdmins')}</p>
            ) : (
              <ul>
                {data.admins.map((admin) => (
                  <li key={admin.user_id} className="flex items-center justify-between gap-3 border-t py-2.5 first:border-t-0">
                    <div className="min-w-0">
                      <p className="truncate font-medium">{admin.full_name || admin.email}</p>
                      <p className="truncate text-muted-foreground">{admin.email}</p>
                    </div>
                    {/* a council's administrator can't lock themselves out */}
                    {(superadmin || admin.user_id !== me?.id) && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        disabled={busy}
                        onClick={() => void run(() => api.admin.removeAdmin(token, slug, admin.user_id), t('admin:notice.saved'))}
                      >
                        {t('admin:manage.remove')}
                      </Button>
                    )}
                  </li>
                ))}
                {data.invitations.map((invitation) => (
                  <li key={invitation.id} className="flex items-center justify-between gap-3 border-t py-2.5 first:border-t-0">
                    <div className="min-w-0">
                      <p className="truncate font-medium">{invitation.email}</p>
                      <span className="rounded-full bg-pill-accent-bg px-2 py-0.5 text-xs font-medium text-pill-accent-ink">
                        {t('admin:manage.pending')}
                      </span>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={busy}
                      onClick={() => void run(() => api.admin.withdrawInvitation(token, slug, invitation.id), t('admin:notice.saved'))}
                    >
                      {t('admin:manage.withdraw')}
                    </Button>
                  </li>
                ))}
              </ul>
            )}
            <form onSubmit={(e) => void invite(e)} className="grid gap-2">
              <Label htmlFor="invite-email">{t('admin:manage.inviteLabel')}</Label>
              <div className="flex gap-2">
                <Input id="invite-email" type="email" required value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} />
                <Button type="submit" variant="outline" disabled={busy}>
                  {t('admin:manage.invite')}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">{t('admin:form.adminEmailHelp')}</p>
            </form>
          </section>


          {superadmin && (
            <section aria-labelledby="sheet-delete" className="grid content-start gap-3 rounded-lg border bg-card p-5">
              <h2 id="sheet-delete" className="text-lg font-semibold">
                {t('admin:manage.deleteTitle')}
              </h2>
              {data.booking_count > 0 ? (
                <p className="text-muted-foreground">{t('admin:manage.deleteBlocked', { count: data.booking_count })}</p>
              ) : (
                <>
                  <p className="text-muted-foreground">{t('admin:manage.deleteHelp')}</p>
                  <div>
                    <Button variant="outline" className="text-destructive hover:text-destructive" onClick={() => setConfirmDelete(true)}>
                      {t('admin:manage.delete')}
                    </Button>
                  </div>
                </>
              )}
            </section>
          )}
          </div>
          <div className="grid gap-6">
            <BookingRulesSettings slug={slug} token={token} />
            <SmtpSettings slug={slug} token={token} />
            <CouncilExport slug={slug} token={token} />
          </div>
        </div>
      </>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={slug} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <p>
          <Link to={`/${slug}/admin`} className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground">
            <ArrowLeft className="size-4" aria-hidden />
            {t('admin:councilSettings.back')}
          </Link>
        </p>
        <div className="mt-3">
          <CouncilHeading section={t('admin:councilSettings.title')} council={heading} otherCouncils={false} />
        </div>
        {body}
      </main>
      <SiteFooter />
      {editing && data && (
        <CouncilFormDialog
          council={data}
          canDeactivate={superadmin}
          onClose={() => setEditing(false)}
          onSaved={(saved) => {
            setEditing(false)
            setNotice(t('admin:notice.saved'))
            council.reload()
            tenants.reload()
            // its address changed: this page moves with it
            if (saved.slug !== slug) navigate(`/${saved.slug}/admin/axustes`, { replace: true })
          }}
        />
      )}

      {confirmDelete && data && (
        <Dialog open onOpenChange={(open) => !open && setConfirmDelete(false)}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>{t('admin:manage.deleteConfirmTitle', { name: data.name })}</DialogTitle>
              <DialogDescription>{t('admin:manage.deleteConfirmBody')}</DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setConfirmDelete(false)}>
                {t('admin:form.cancel')}
              </Button>
              <Button variant="destructive" disabled={busy} onClick={() => void remove()}>
                {t('admin:manage.deleteConfirm')}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}
