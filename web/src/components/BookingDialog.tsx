import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { useMe } from '@/hooks/useMe'
import type { Facility, Slot } from '@/lib/api'
import { formatBookingDay, formatDateTime, formatTime } from '@/lib/time'

type Props = {
  pick: { facility: Facility; slot: Slot } | null
  timeZone: string
  today: string
  authenticated: boolean
  onClose: () => void
  onConfirm: (facility: Facility, slot: Slot) => Promise<string | null> // error message, or null when booked
  onLogin: () => void
}

/** Confirms a turn. The rules are shown first and must be accepted before booking. */
export function BookingDialog({ pick, timeZone, today, authenticated, onClose, onConfirm, onLogin }: Props) {
  const { t } = useTranslation(['bookings', 'common', 'home', 'profile'])
  const { me } = useMe()
  const needsProfile = authenticated && me !== null && !me.profile_complete
  const [submitting, setSubmitting] = useState(false)
  const [accepted, setAccepted] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const close = () => {
    setError(null)
    onClose()
  }

  if (!pick) return <Dialog open={false} />

  const { facility, slot } = pick
  const time = `${formatTime(slot.starts_at, timeZone)}–${formatTime(slot.ends_at, timeZone)}`
  const cancelUntil = new Date(new Date(slot.starts_at).getTime() - facility.min_cancel_hours * 3600_000).toISOString()

  const confirm = async () => {
    setSubmitting(true)
    setError(null)
    const message = await onConfirm(facility, slot)
    setSubmitting(false)
    if (message) setError(message)
  }

  return (
    <Dialog open onOpenChange={(open) => !open && close()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('bookings:dialog.title', { facility: facility.name })}</DialogTitle>
          <DialogDescription className="num">
            {formatBookingDay(slot.starts_at, timeZone, today)}, {time}
          </DialogDescription>
        </DialogHeader>

        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
          <dt className="text-muted-foreground">{t('bookings:dialog.place')}</dt>
          <dd>{facility.address || '—'}</dd>
          <dt className="text-muted-foreground">{t('bookings:dialog.duration')}</dt>
          <dd className="num">{t('home:availability.minutes', { count: facility.slot_minutes })}</dd>
          <dt className="text-muted-foreground">{t('bookings:dialog.cancellation')}</dt>
          <dd className="num">{t('bookings:dialog.cancellationUntil', { date: formatDateTime(cancelUntil, timeZone) })}</dd>
          {facility.min_age !== null && (
            <>
              <dt className="text-muted-foreground">{t('bookings:dialog.minAge')}</dt>
              <dd className="num">{t('bookings:dialog.minAgeValue', { count: facility.min_age })}</dd>
            </>
          )}
        </dl>

        <section aria-labelledby="booking-rules" className="grid gap-2 rounded-md bg-muted p-3">
          <h3 id="booking-rules" className="font-medium">
            {t('bookings:dialog.rules')}
          </h3>
          <p tabIndex={0} className="max-h-40 overflow-y-auto rounded-sm whitespace-pre-line">{facility.usage_rules || t('bookings:dialog.noRules')}</p>
          {authenticated && !needsProfile && (
            <div className="flex items-start gap-2 pt-1">
              <Checkbox id="accept-rules" checked={accepted} onCheckedChange={(value) => setAccepted(value === true)} className="mt-0.5" />
              <Label htmlFor="accept-rules" className="leading-5 font-normal">
                {t('bookings:dialog.acceptRules')}
              </Label>
            </div>
          )}
        </section>

        {needsProfile && (
          <Alert className="border-0 bg-pill-accent-bg text-pill-accent-ink">
            <AlertDescription className="text-pill-accent-ink">
              <span className="font-medium">{t('profile:incompleteTitle')}.</span> {t('profile:incompleteBody')}
            </AlertDescription>
          </Alert>
        )}

        {error && (
          <Alert variant="destructive" aria-live="assertive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={close}>
            {t('common:back')}
          </Button>
          {needsProfile ? (
            <Button asChild>
              <Link to="/perfil">{t('profile:incompleteAction')}</Link>
            </Button>
          ) : authenticated ? (
            <Button onClick={() => void confirm()} disabled={submitting || !accepted}>
              {submitting ? t('bookings:dialog.confirming') : t('bookings:dialog.confirm')}
            </Button>
          ) : (
            <Button onClick={onLogin}>{t('bookings:dialog.loginToBook')}</Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
