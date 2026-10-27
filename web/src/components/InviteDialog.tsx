import { Check, Copy, RefreshCw, Share2, X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type Booking, type Invite } from '@/lib/api'
import { formatBookingDay, formatTime } from '@/lib/time'

type Props = {
  booking: Booking | null
  timeZone: string
  today: string
  onClose: () => void
  onChanged: () => void // someone was removed or the link changed: the bookings list refreshes its counts
}

function inviteUrl(token: string) {
  return `${window.location.origin}/invitacion/${token}`
}

/** The owner shares a booking: the invite link, the spots left and who joined. */
export function InviteDialog({ booking, timeZone, today, onClose, onChanged }: Props) {
  if (!booking) return <Dialog open={false} />
  return <InviteDialogOpen booking={booking} timeZone={timeZone} today={today} onClose={onClose} onChanged={onChanged} />
}

function InviteDialogOpen({ booking, timeZone, today, onClose, onChanged }: Props & { booking: Booking }) {
  const { t } = useTranslation('invites')
  const auth = useAuth()
  const token = auth.user?.access_token
  const loaded = useAsync(token ? () => api.invites.get(token, booking.id) : null, `invite:${booking.id}`)
  const [invite, setInvite] = useState<Invite | null>(null)
  const [busy, setBusy] = useState(false)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const current = invite ?? loaded.data
  const day = formatBookingDay(booking.starts_at, timeZone, today)
  const start = formatTime(booking.starts_at, timeZone)
  const link = current?.token ? inviteUrl(current.token) : ''

  const run = async (action: () => Promise<Invite | void>) => {
    if (!token) return
    setBusy(true)
    setError(null)
    try {
      const next = await action()
      if (next) setInvite(next)
      setCopied(false)
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('dialog.error'))
    } finally {
      setBusy(false)
    }
  }

  const share = () => run(() => api.invites.share(token!, booking.id))
  const remove = (participantId: string) =>
    run(async () => {
      await api.invites.removeParticipant(token!, booking.id, participantId)
      return api.invites.get(token!, booking.id)
    })

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(link)
      setCopied(true)
    } catch {
      // no clipboard permission: the field stays selectable
    }
  }
  const nativeShare = () =>
    void navigator.share({ title: booking.facility_name, text: t('dialog.shareText', { facility: booking.facility_name, day, time: start }), url: link }).catch(() => {})

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('dialog.title', { facility: booking.facility_name })}</DialogTitle>
          <DialogDescription className="num">
            {t('dialog.description', { day, start, end: formatTime(booking.ends_at, timeZone) })}
          </DialogDescription>
        </DialogHeader>

        {loaded.error && !current ? (
          <Alert variant="destructive">
            <AlertDescription>{loaded.error.message || t('dialog.error')}</AlertDescription>
          </Alert>
        ) : !current ? (
          <div className="space-y-3" aria-busy="true">
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : (
          <div className="space-y-5">
            <p className="num font-medium" aria-live="polite">
              {current.state === 'closed'
                ? t('dialog.closed')
                : current.state === 'full'
                  ? t('dialog.full')
                  : t('dialog.spotsLeft', { count: current.spots_left })}
              <span className="ml-2 font-normal text-muted-foreground">
                {t('spots', { taken: current.capacity - current.spots_left, capacity: current.capacity })}
              </span>
            </p>

            {current.state !== 'closed' &&
              (link ? (
                <div className="space-y-2">
                  <Label htmlFor="invite-link">{t('dialog.link')}</Label>
                  <div className="flex gap-2">
                    <Input id="invite-link" readOnly value={link} onFocus={(e) => e.currentTarget.select()} className="min-w-0 font-mono text-xs" />
                    <Button variant="outline" onClick={() => void copy()} className="shrink-0">
                      {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
                      {copied ? t('dialog.copied') : t('dialog.copy')}
                    </Button>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    {typeof navigator.share === 'function' && (
                      <Button size="sm" onClick={nativeShare}>
                        <Share2 aria-hidden />
                        {t('dialog.share')}
                      </Button>
                    )}
                    <Button variant="ghost" size="sm" onClick={() => void share()} disabled={busy} title={t('dialog.resetHint')}>
                      <RefreshCw aria-hidden />
                      {t('dialog.reset')}
                    </Button>
                  </div>
                  <p className="text-sm text-muted-foreground">{t('dialog.resetHint')}</p>
                </div>
              ) : (
                <Button onClick={() => void share()} disabled={busy}>
                  {busy ? t('dialog.creating') : t('dialog.create')}
                </Button>
              ))}

            <div>
              <h3 className="font-medium">{t('dialog.people')}</h3>
              <ul className="mt-1">
                <li className="border-t py-2 first:border-t-0">
                  {t('dialog.you')} <span className="text-muted-foreground">· {t('dialog.owner')}</span>
                </li>
                {current.participants.map((p) => (
                  <li key={p.id} className="flex items-center justify-between gap-3 border-t py-2">
                    <span className="min-w-0">
                      <span className="block truncate">{p.name}</span>
                      {p.is_guest && <span className="num text-sm text-muted-foreground">{t('dialog.guestAge', { age: p.age })}</span>}
                    </span>
                    {current.state !== 'closed' && (
                      <Button variant="ghost" size="icon" aria-label={t('dialog.remove', { name: p.name })} onClick={() => void remove(p.id)} disabled={busy}>
                        <X aria-hidden />
                      </Button>
                    )}
                  </li>
                ))}
              </ul>
              {current.participants.length === 0 && <p className="text-sm text-muted-foreground">{t('dialog.nobody')}</p>}
            </div>
          </div>
        )}

        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('dialog.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
