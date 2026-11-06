import { ArrowLeft } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import { BookingRulesSettings } from '@/components/admin/BookingRulesSettings'
import { CouncilDetailsSettings } from '@/components/admin/CouncilDetailsSettings'
import { CouncilExport } from '@/components/admin/CouncilExport'
import { SmtpSettings } from '@/components/admin/SmtpSettings'
import { WidgetSettings } from '@/components/admin/WidgetSettings'
import { CouncilHeading } from '@/components/CouncilHeading'
import { SettingsRow } from '@/components/SettingsRow'
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
 * /:slug/admin/axustes: everything about one council, laid out like the profile:
 * its details, administrators, booking window, email relay, widget, data export,
 * and (superadministrators) deleting it.
 */
export function CouncilSettingsPage() {
  const { slug = '' } = useParams()
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const { me } = useMe()
  const navigate = useNavigate()
  const location = useLocation()
  const allowed = useCouncilAdmin(slug)
  // a superadministrator also switches the council on or off and deletes it
  const superadmin = me?.is_superadmin ?? false
  const token = auth.user?.access_token ?? ''
  const tenant = useAsync(() => api.tenant(slug), `tenant:${slug}`)
  const council = useAsync(allowed ? () => api.admin.council(token, slug) : null, `council:${slug}:${allowed}`)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const heading = tenant.data ?? undefined
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
    setDeleteError(null)
    try {
      await api.admin.deleteCouncil(token, slug)
      navigate('/admin')
    } catch (e) {
      setConfirmDelete(false)
      setDeleteError(e instanceof Error ? e.message : String(e))
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
      <div className="mt-6">
        <CouncilDetailsSettings
          key={data.slug}
          council={data}
          token={token}
          canDeactivate={superadmin}
          onSaved={(saved) => {
            council.reload()
            tenant.reload()
            // its address changed: this page moves with it
            if (saved.slug !== slug) navigate(`/${saved.slug}/admin/axustes`, { replace: true })
          }}
        />

        <SettingsRow id="sheet-admins" title={t('admin:manage.admins')} description={t('admin:manage.adminsHelp')}>
          <div className="grid content-start gap-4 rounded-lg border bg-card p-5">
            {data.admins.length === 0 && data.invitations.length === 0 ? (
              <p className="text-muted-foreground">{t('admin:manage.noAdmins')}</p>
            ) : (
              <ul>
                {data.admins.map((admin) => (
                  <li key={admin.user_id} className="flex items-center justify-between gap-3 border-t py-2.5 first:border-t-0 first:pt-0">
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
                  <li key={invitation.id} className="flex items-center justify-between gap-3 border-t py-2.5 first:border-t-0 first:pt-0">
                    <div className="min-w-0">
                      <p className="truncate font-medium">{invitation.email}</p>
                      <span className="rounded-full bg-pill-accent-bg px-2 py-0.5 text-xs font-medium text-pill-accent-ink">{t('admin:manage.pending')}</span>
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
            <form onSubmit={(e) => void invite(e)} className="grid gap-2 border-t pt-4">
              <Label htmlFor="invite-email">{t('admin:manage.inviteLabel')}</Label>
              <div className="flex gap-2">
                <Input id="invite-email" type="email" required maxLength={255} value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} />
                <Button type="submit" variant="outline" disabled={busy}>
                  {t('admin:manage.invite')}
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">{t('admin:form.adminEmailHelp')}</p>
            </form>
            <div aria-live="polite" className="empty:hidden">
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
          </div>
        </SettingsRow>

        <BookingRulesSettings slug={slug} token={token} />
        <SmtpSettings slug={slug} token={token} />
        <WidgetSettings slug={slug} token={token} />
        <CouncilExport slug={slug} token={token} />

        {superadmin && (
          <SettingsRow id="sheet-delete" title={t('admin:manage.deleteTitle')} description={t('admin:manage.deleteHelp')}>
            <div className="grid content-start gap-4 rounded-lg border bg-card p-5">
              {data.booking_count > 0 ? (
                <p className="text-muted-foreground">{t('admin:manage.deleteBlocked', { count: data.booking_count })}</p>
              ) : (
                <div>
                  <Button variant="destructive" onClick={() => setConfirmDelete(true)}>
                    {t('admin:manage.delete')}
                  </Button>
                </div>
              )}
              <div aria-live="polite" className="empty:hidden">
                {deleteError && (
                  <Alert variant="destructive">
                    <AlertDescription>{deleteError}</AlertDescription>
                  </Alert>
                )}
              </div>
            </div>
          </SettingsRow>
        )}
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar tenantSlug={slug} tenantName={heading?.name} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <p>
          {/* back to where they came from, like the browser's back; straight from a link, to the council's administration */}
          <Link
            to={`/${slug}/admin`}
            onClick={(event) => {
              if (location.key === 'default') return
              event.preventDefault()
              navigate(-1)
            }}
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
          >
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
