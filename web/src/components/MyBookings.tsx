import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { QrCode } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { Booking, MyBookings as MyBookingsData } from '@/lib/api'
import { formatBookingDay, formatDateTime, formatTime } from '@/lib/time'

type Props = {
  authenticated: boolean
  data: MyBookingsData | null
  loading: boolean
  error: Error | null
  timeZone: string
  today: string
  notice: string | null
  onRetry: () => void
  onLogin: () => void
  onRegister: () => void
  onCancel: (booking: Booking) => void
  onShowQr: (booking: Booking) => void
}

function When({ booking, timeZone, today }: { booking: Booking; timeZone: string; today: string }) {
  return (
    <p className="num text-muted-foreground">
      {formatBookingDay(booking.starts_at, timeZone, today)}, {formatTime(booking.starts_at, timeZone)}–{formatTime(booking.ends_at, timeZone)}
    </p>
  )
}

export function MyBookings({ authenticated, data, loading, error, timeZone, today, notice, onRetry, onLogin, onRegister, onCancel, onShowQr }: Props) {
  const { t } = useTranslation(['bookings', 'common', 'invites'])
  const upcoming = data?.upcoming ?? []
  const history = (data?.history ?? []).slice(0, 4)

  return (
    <section aria-labelledby="my-bookings-title" className="rounded-lg border bg-card p-5">
      <div className="flex items-baseline justify-between gap-4">
        <h2 id="my-bookings-title" className="text-lg font-semibold">
          {t('bookings:title')}
        </h2>
        {authenticated && data && (
          <span className="num text-muted-foreground">
            {t('bookings:upcoming', { count: upcoming.length })} ·{' '}
            <Link to="/reservas" className="font-medium text-foreground underline-offset-4 hover:underline">
              {t('invites:myBookings.seeAll')}
            </Link>
          </span>
        )}
      </div>

      <div aria-live="polite">
        {notice && (
          <Alert className="mt-4 border-0 bg-pill-positive-bg text-pill-positive-ink">
            <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
          </Alert>
        )}
      </div>

      {!authenticated ? (
        <div className="mt-3 space-y-4">
          <p>{t('bookings:signedOut')}</p>
          <div className="flex flex-wrap gap-2">
            <Button onClick={onLogin}>{t('common:signIn')}</Button>
            <Button variant="outline" onClick={onRegister}>
              {t('bookings:createAccount')}
            </Button>
          </div>
        </div>
      ) : error ? (
        <div className="mt-4 flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
          <p>{t('bookings:loadError')}</p>
          <Button variant="ghost" size="sm" onClick={onRetry}>
            {t('common:retry')}
          </Button>
        </div>
      ) : loading && !data ? (
        <ul className="mt-3" aria-busy="true" aria-label={t('bookings:loading')}>
          {Array.from({ length: 3 }, (_, i) => (
            <li key={i} className="space-y-1.5 border-t py-3 first:border-t-0">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-32" />
            </li>
          ))}
        </ul>
      ) : (
        <>
          {upcoming.length === 0 ? (
            <p className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">
              {t('bookings:empty')}
            </p>
          ) : (
            <ul className="mt-2">
              {upcoming.map((booking) => (
                <li key={booking.id} className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 border-t py-3 first:border-t-0">
                  <div className="min-w-48 flex-1">
                    <p className="truncate font-medium">{booking.facility_name}</p>
                    <When booking={booking} timeZone={timeZone} today={today} />
                    {booking.role === 'participant' && (
                      <span className="mt-1 mr-1 inline-block rounded-full bg-pill-neutral-bg px-2 py-0.5 text-xs font-medium text-pill-neutral-ink">
                        {booking.owner_name ? t('invites:role.invitedBy', { name: booking.owner_name }) : t('invites:role.invited')}
                      </span>
                    )}
                    {booking.checked_in_at && (
                      <span className="mt-1 inline-block rounded-full bg-pill-positive-bg px-2 py-0.5 text-xs font-medium text-pill-positive-ink">
                        {t('bookings:status.checkedIn')}
                      </span>
                    )}
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    {booking.checkin_code && (
                      <Button variant="outline" size="sm" aria-label={t('bookings:qr.show')} onClick={() => onShowQr(booking)}>
                        <QrCode aria-hidden />
                        QR
                      </Button>
                    )}
                    {booking.role === 'participant' ? null : booking.can_cancel ? (
                      <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => onCancel(booking)}>
                        {t('bookings:cancel')}
                      </Button>
                    ) : (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span tabIndex={0} className="shrink-0 rounded-sm text-muted-foreground">
                            {t('bookings:noCancel')}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent className="num">
                          {t('bookings:cancelDeadlinePassed', { date: formatDateTime(booking.cancel_until, timeZone) })}
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}

          {history.length > 0 && (
            <div className="mt-5">
              <h3 className="font-medium">{t('bookings:history')}</h3>
              <ul className="mt-1">
                {history.map((booking) => (
                  <li key={booking.id} className="flex items-center justify-between gap-3 border-t py-2.5 first:border-t-0">
                    <div className="min-w-0">
                      <p className="truncate">{booking.facility_name}</p>
                      <When booking={booking} timeZone={timeZone} today={today} />
                    </div>
                    <span
                      className={
                        booking.status === 'cancelled'
                          ? 'shrink-0 rounded-full bg-pill-neutral-bg px-2 py-0.5 text-xs font-medium text-pill-neutral-ink'
                          : 'shrink-0 rounded-full bg-pill-positive-bg px-2 py-0.5 text-xs font-medium text-pill-positive-ink'
                      }
                    >
                      {booking.status === 'cancelled' ? t('bookings:status.cancelled') : t('bookings:status.done')}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </section>
  )
}
