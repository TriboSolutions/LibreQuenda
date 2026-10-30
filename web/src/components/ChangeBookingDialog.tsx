import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { DatePicker } from '@/components/DatePicker'
import { ParticipantList } from '@/components/ParticipantList'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, ApiError, type Booking, type Slot } from '@/lib/api'
import { addDays, formatBookingDay, formatDateTime, formatTime, localDate } from '@/lib/time'

type Props = {
  booking: Booking | null
  timeZone: string
  today: string
  onClose: () => void
  onChanged: (booking: Booking) => void // moved: the page shows a notice and reloads
  onPeopleChanged: () => void // someone was removed: the bookings list refreshes its counts
}

/** The owner moves a booking to another day or time of the same facility, and can remove people who joined. */
export function ChangeBookingDialog({ booking, ...rest }: Props) {
  if (!booking) return <Dialog open={false} />
  return <ChangeBookingDialogOpen booking={booking} {...rest} />
}

function ChangeBookingDialogOpen({ booking, timeZone, today, onClose, onChanged, onPeopleChanged }: Props & { booking: Booking }) {
  const { t } = useTranslation(['bookings', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token
  const [date, setDate] = useState(() => localDate(booking.starts_at, timeZone))
  const [picked, setPicked] = useState<Slot | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const page = useAsync(() => api.facilityPage(booking.tenant_slug, booking.facility_slug, date), `change:${booking.id}:${date}`)
  const invite = useAsync(token && booking.capacity > 0 ? () => api.invites.get(token, booking.id) : null, `invite:${booking.id}`)

  const facility = page.data?.facility
  const slots = facility?.slots ?? []

  const pickDate = (value: string) => {
    setDate(value)
    setPicked(null)
    setError(null)
  }

  const save = async () => {
    if (!token || !picked) return
    setBusy(true)
    setError(null)
    try {
      onChanged(await api.changeBooking(token, booking.id, picked.starts_at))
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        setPicked(null)
        page.reload()
      }
      setError(e instanceof Error ? e.message : t('changeDialog.error'))
    } finally {
      setBusy(false)
    }
  }

  const remove = async (participantId: string) => {
    if (!token) return
    setBusy(true)
    setError(null)
    try {
      await api.invites.removeParticipant(token, booking.id, participantId)
      invite.reload()
      onPeopleChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('changeDialog.error'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('changeDialog.title', { facility: booking.facility_name })}</DialogTitle>
          <DialogDescription className="num">
            {t('changeDialog.description', {
              day: formatBookingDay(booking.starts_at, timeZone, today),
              start: formatTime(booking.starts_at, timeZone),
              end: formatTime(booking.ends_at, timeZone),
            })}
          </DialogDescription>
        </DialogHeader>

        <section aria-labelledby="change-when" className="space-y-3">
          <h3 id="change-when" className="font-medium">
            {t('changeDialog.when')}
          </h3>
          <div className="grid gap-1.5">
            <Label htmlFor="change-date">{t('changeDialog.date')}</Label>
            <DatePicker
              id="change-date"
              value={date}
              onChange={pickDate}
              min={today}
              max={facility ? addDays(today, Math.ceil(facility.max_advance_hours / 24)) : undefined}
              className="w-full sm:w-64"
            />
          </div>

          {page.error ? (
            <div className="flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
              <p>{page.error.message}</p>
              <Button variant="ghost" size="sm" onClick={page.reload}>
                {t('common:retry')}
              </Button>
            </div>
          ) : !page.data || page.loading ? (
            <div className="grid grid-cols-3 gap-2 sm:grid-cols-4" aria-busy="true">
              {Array.from({ length: 8 }, (_, i) => (
                <Skeleton key={i} className="h-11" />
              ))}
            </div>
          ) : slots.length === 0 ? (
            <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('changeDialog.closedDay')}</p>
          ) : (
            <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4" aria-label={t('changeDialog.times')}>
              {slots.map((slot) => {
                const current = slot.starts_at === booking.starts_at
                const selected = slot.starts_at === picked?.starts_at
                return (
                  <li key={slot.starts_at}>
                    <Button
                      variant="outline"
                      aria-pressed={selected}
                      className="num h-11 w-full flex-col gap-0 leading-4 disabled:opacity-100 disabled:bg-muted disabled:text-muted-foreground data-[free=true]:border-primary data-[free=true]:text-primary data-[free=false]:line-through data-[current=true]:no-underline data-[current=true]:bg-accent data-[current=true]:text-accent-foreground aria-pressed:bg-primary aria-pressed:text-primary-foreground"
                      data-free={slot.status === 'free'}
                      data-current={current}
                      disabled={slot.status !== 'free'}
                      onClick={() => setPicked(slot)}
                    >
                      {/* named by what it shows, the end time added for screen readers (WCAG 2.5.3) */}
                      <span>
                        {formatTime(slot.starts_at, timeZone)}
                        <span className="sr-only">–{formatTime(slot.ends_at, timeZone)}</span>
                      </span>
                      {current && (
                        <span className="text-xs">
                          <span className="sr-only">, </span>
                          {t('changeDialog.current')}
                        </span>
                      )}
                    </Button>
                  </li>
                )
              })}
            </ul>
          )}
          {picked && facility && (
            <p className="num text-sm text-muted-foreground">
              {t('changeDialog.cancelUntil', {
                date: formatDateTime(new Date(Date.parse(picked.starts_at) - facility.min_cancel_hours * 3600_000).toISOString(), timeZone),
              })}
            </p>
          )}
        </section>

        {booking.capacity > 0 &&
          (invite.data ? (
            <ParticipantList participants={invite.data.participants} removable busy={busy} onRemove={(id) => void remove(id)} />
          ) : (
            invite.loading && <Skeleton className="h-16 w-full" />
          ))}

        {error && (
          <Alert variant="destructive" aria-live="assertive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('changeDialog.close')}
          </Button>
          <Button onClick={() => void save()} disabled={busy || !picked}>
            {busy && picked
              ? t('changeDialog.saving')
              : picked
                ? t('changeDialog.confirm', { day: formatBookingDay(picked.starts_at, timeZone, today), time: formatTime(picked.starts_at, timeZone) })
                : t('changeDialog.pick')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
