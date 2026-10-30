import { ExternalLink } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { BookingRulesSettings } from '@/components/admin/BookingRulesSettings'
import { CouncilExport } from '@/components/admin/CouncilExport'
import { CouncilFormDialog } from '@/components/admin/CouncilFormDialog'
import { SmtpSettings } from '@/components/admin/SmtpSettings'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useMe } from '@/hooks/useMe'
import { api } from '@/lib/api'

type Props = {
  slug: string
  // a superadministrator also switches the council on or off and deletes it;
  // its own administrators get the rest, for their council only
  superadmin?: boolean
  onClose: () => void
  onChanged: (message: string) => void // the list refreshes after any change
  onDeleted?: (message: string) => void
  onSlugChanged?: (slug: string) => void // default: close, and reopen from the list
}

/** Everything about one council: its details, administrators, email relay, data export, and deleting it. */
export function CouncilSheet({ slug, superadmin = true, onClose, onChanged, onDeleted, onSlugChanged }: Props) {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const { me } = useMe()
  const token = auth.user?.access_token ?? ''
  const council = useAsync(() => api.admin.council(token, slug), `council:${slug}`)
  const [editing, setEditing] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const run = async (action: () => Promise<unknown>, message: string) => {
    setBusy(true)
    setError(null)
    try {
      await action()
      council.reload()
      onChanged(message)
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
      onDeleted?.(t('admin:notice.deleted', { name: council.data?.name }))
    } catch (e) {
      setConfirmDelete(false)
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const data = council.data

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full gap-0 overflow-y-auto sm:max-w-lg">
        <SheetHeader className="border-b">
          <SheetTitle className="text-lg">{data?.name ?? slug}</SheetTitle>
          <SheetDescription className="num">/{slug}</SheetDescription>
        </SheetHeader>

        <div className="grid gap-6 p-4">
          {error && (
            <Alert variant="destructive" aria-live="assertive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {!data ? (
            council.error ? (
              <p className="text-destructive">{council.error.message}</p>
            ) : (
              <div className="grid gap-2" aria-busy="true">
                <Skeleton className="h-4 w-48" />
                <Skeleton className="h-4 w-64" />
                <Skeleton className="h-4 w-40" />
              </div>
            )
          ) : (
            <>
              <section aria-labelledby="sheet-details" className="grid gap-3">
                <div className="flex items-center justify-between gap-3">
                  <h3 id="sheet-details" className="font-semibold">
                    {t('admin:manage.details')}
                  </h3>
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

              <section aria-labelledby="sheet-admins" className="grid gap-3 border-t pt-5">
                <h3 id="sheet-admins" className="font-semibold">
                  {t('admin:manage.admins')}
                </h3>
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

              <BookingRulesSettings slug={slug} token={token} />
              <SmtpSettings slug={slug} token={token} />

              <CouncilExport slug={slug} token={token} />

              {superadmin && (
                <section aria-labelledby="sheet-delete" className="grid gap-3 border-t pt-5">
                  <h3 id="sheet-delete" className="font-semibold">
                    {t('admin:manage.deleteTitle')}
                  </h3>
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
            </>
          )}
        </div>

        {editing && data && (
          <CouncilFormDialog
            council={data}
            canDeactivate={superadmin}
            onClose={() => setEditing(false)}
            onSaved={(saved) => {
              setEditing(false)
              onChanged(t('admin:notice.saved'))
              if (saved.slug === slug) council.reload()
              else if (onSlugChanged) onSlugChanged(saved.slug)
              else onClose() // its address changed: reopen from the list
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
      </SheetContent>
    </Sheet>
  )
}
