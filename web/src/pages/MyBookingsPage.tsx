import { QrCode, UserPlus, Users } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { BookingQrDialog } from '@/components/BookingQrDialog'
import { CancelDialog } from '@/components/CancelDialog'
import { InviteDialog } from '@/components/InviteDialog'
import { LeaveDialog } from '@/components/LeaveDialog'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { api, type Booking, type Person } from '@/lib/api'
import { formatBookingDay, formatTime, localDate } from '@/lib/time'

// Bookings can span councils; all of them are in Galicia today
const TIME_ZONE = 'Europe/Madrid'

const pill = {
  neutral: 'shrink-0 rounded-full bg-pill-neutral-bg px-2 py-0.5 text-xs font-medium text-pill-neutral-ink',
  positive: 'shrink-0 rounded-full bg-pill-positive-bg px-2 py-0.5 text-xs font-medium text-pill-positive-ink',
}

/** /reservas: every booking of the signed-in user, made or joined, upcoming and past. */
export function MyBookingsPage() {
  const { t } = useTranslation(['invites', 'bookings', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token
  const me = useMe()
  const { login } = useLogin()
  const tenants = useAsync(() => api.tenants(), 'tenants')
  // GET /me (MeProvider) first creates the user's row; the bookings load after it
  const bookings = useAsync(token && me.me ? () => api.myBookings(token) : null, `bookings:${me.me?.id}`)

  const [now] = useState(() => new Date().toISOString())
  const today = localDate(now, TIME_ZONE)
  const [toCancel, setToCancel] = useState<Booking | null>(null)
  const [toLeave, setToLeave] = useState<Booking | null>(null)
  const [toInvite, setToInvite] = useState<Booking | null>(null)
  const [showQr, setShowQr] = useState<Booking | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const confirmCancel = async (booking: Booking) => {
    if (!token) return t('bookings:cancelDialog.needLogin')
    try {
      await api.cancel(token, booking.id)
      setToCancel(null)
      setNotice(t('bookings:notice.cancelled', { facility: booking.facility_name, day: formatBookingDay(booking.starts_at, TIME_ZONE, today) }))
      bookings.reload()
      return null
    } catch (error) {
      return error instanceof Error ? error.message : t('bookings:cancelDialog.error')
    }
  }

  const confirmLeave = async (booking: Booking) => {
    if (!token) return t('bookings:cancelDialog.needLogin')
    try {
      await api.invites.leave(token, booking.id)
      setToLeave(null)
      setNotice(t('invites:notice.left', { facility: booking.facility_name }))
      bookings.reload()
      return null
    } catch (error) {
      return error instanceof Error ? error.message : t('invites:leaveDialog.error')
    }
  }

  let body
  if (auth.isLoading || (auth.isAuthenticated && !bookings.data && !bookings.error && !me.error)) {
    body = (
      <div className="mt-6 space-y-3" aria-busy="true" aria-label={t('bookings:loading')}>
        {Array.from({ length: 3 }, (_, i) => (
          <Skeleton key={i} className="h-20 w-full" />
        ))}
      </div>
    )
  } else if (!auth.isAuthenticated) {
    body = (
      <div className="mt-6 grid max-w-[760px] gap-4 rounded-lg border bg-card p-5">
        <p>{t('invites:myBookings.signIn')}</p>
        <div>
          <Button onClick={login}>{t('common:signIn')}</Button>
        </div>
      </div>
    )
  } else if (me.error || bookings.error) {
    body = (
      <div className="mt-6 flex max-w-[760px] items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
        <p>{t('bookings:loadError')}</p>
        <Button variant="ghost" size="sm" onClick={() => (me.error ? me.reload() : bookings.reload())}>
          {t('common:retry')}
        </Button>
      </div>
    )
  } else {
    const upcoming = bookings.data?.upcoming ?? []
    const history = bookings.data?.history ?? []
    body = (
      <div className="mt-6 grid max-w-[760px] gap-6">
        <div aria-live="polite">
          {notice && (
            <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
              <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
            </Alert>
          )}
        </div>

        <section aria-labelledby="upcoming-title" className="rounded-lg border bg-card p-5">
          <h2 id="upcoming-title" className="text-lg font-semibold">
            {t('invites:myBookings.upcoming')} <span className="num font-normal text-muted-foreground">· {upcoming.length}</span>
          </h2>
          {upcoming.length === 0 ? (
            <div className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">
              <p>{t('invites:myBookings.emptyUpcoming')}</p>
              <Link to="/" className="mt-1 inline-block font-medium text-foreground underline-offset-4 hover:underline">
                {t('invites:myBookings.findTurn')}
              </Link>
            </div>
          ) : (
            <ul className="mt-2">
              {upcoming.map((booking) => (
                <BookingRow key={booking.id} booking={booking} today={today}>
                  {booking.role === 'owner' ? (
                    <>
                      {booking.checkin_code && (
                        <Button variant="outline" size="sm" aria-label={t('bookings:qr.show')} onClick={() => setShowQr(booking)}>
                          <QrCode aria-hidden />
                        </Button>
                      )}
                      {booking.capacity > 0 && booking.starts_at > now && (
                        <Button variant="outline" size="sm" onClick={() => setToInvite(booking)}>
                          <UserPlus aria-hidden />
                          {t('invites:actions.invite')}
                        </Button>
                      )}
                      {booking.can_cancel && (
                        <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setToCancel(booking)}>
                          {t('bookings:cancel')}
                        </Button>
                      )}
                    </>
                  ) : (
                    booking.can_leave && (
                      <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setToLeave(booking)}>
                        {t('invites:actions.leave')}
                      </Button>
                    )
                  )}
                </BookingRow>
              ))}
            </ul>
          )}
        </section>

        <section aria-labelledby="history-title" className="rounded-lg border bg-card p-5">
          <h2 id="history-title" className="text-lg font-semibold">
            {t('invites:myBookings.history')}
          </h2>
          {history.length === 0 ? (
            <p className="mt-4 text-muted-foreground">{t('invites:myBookings.emptyHistory')}</p>
          ) : (
            <ul className="mt-2">
              {history.map((booking) => (
                <BookingRow key={booking.id} booking={booking} today={today}>
                  <span className={booking.status === 'cancelled' ? pill.neutral : pill.positive}>
                    {booking.status === 'cancelled' ? t('bookings:status.cancelled') : t('bookings:status.done')}
                  </span>
                </BookingRow>
              ))}
            </ul>
          )}
        </section>
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={null} />
      <main className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('invites:myBookings.title')}</h1>
        <p className="mt-1 text-base text-muted-foreground">{t('invites:myBookings.subtitle')}</p>
        {body}
      </main>

      <CancelDialog key={toCancel?.id ?? 'none'} booking={toCancel} timeZone={TIME_ZONE} today={today} onClose={() => setToCancel(null)} onConfirm={confirmCancel} />
      <LeaveDialog key={toLeave?.id ?? 'none'} booking={toLeave} timeZone={TIME_ZONE} today={today} onClose={() => setToLeave(null)} onConfirm={confirmLeave} />
      <InviteDialog
        key={toInvite?.id ?? 'none'}
        booking={toInvite}
        timeZone={TIME_ZONE}
        today={today}
        onClose={() => setToInvite(null)}
        onChanged={bookings.reload}
      />
      <BookingQrDialog booking={showQr} timeZone={TIME_ZONE} today={today} onClose={() => setShowQr(null)} />
    </div>
  )
}

function BookingRow({ booking, today, children }: { booking: Booking; today: string; children: React.ReactNode }) {
  const { t } = useTranslation('invites')
  return (
    <li className="flex flex-col gap-2 border-t py-3 first:border-t-0 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
      <div className="min-w-0">
        <p className="truncate font-medium">
          {booking.facility_name} <span className="font-normal text-muted-foreground">· {booking.tenant_name}</span>
        </p>
        <p className="num text-muted-foreground">
          {formatBookingDay(booking.starts_at, TIME_ZONE, today)}, {formatTime(booking.starts_at, TIME_ZONE)}–{formatTime(booking.ends_at, TIME_ZONE)}
        </p>
        <p className="mt-1 flex flex-wrap items-center gap-2 text-sm">
          <span className={booking.role === 'owner' ? pill.neutral : pill.positive}>
            {booking.role === 'owner' ? t('role.owner') : booking.owner_name ? t('role.invitedBy', { name: booking.owner_name }) : t('role.invited')}
          </span>
          {booking.capacity > 0 && (
            <span className="num inline-flex items-center gap-1 text-muted-foreground">
              <Users className="size-3.5" aria-hidden />
              {t('spots', { taken: booking.participant_count + 1, capacity: booking.capacity })}
            </span>
          )}
        </p>
        {booking.people.length > 0 && <PeopleList people={booking.people} />}
      </div>
      <div className="flex shrink-0 flex-wrap items-center gap-2">{children}</div>
    </li>
  )
}

/** Who is coming, on the booking itself. */
function PeopleList({ people }: { people: Person[] }) {
  const { t } = useTranslation('invites')
  return (
    <ul aria-label={t('people.label')} className="mt-2 flex flex-wrap gap-1.5">
      {people.map((person, i) => (
        <li
          key={i}
          className={`rounded-full border px-2.5 py-0.5 text-sm ${person.is_me ? 'border-transparent bg-accent font-medium text-accent-foreground' : ''}`}
        >
          {person.is_me ? t('people.you') : person.name || t('people.unnamed')}
          {person.is_owner && <span className="text-muted-foreground"> · {t('people.organiser')}</span>}
          {person.is_guest && <span className="text-muted-foreground"> · {t('people.noAccount')}</span>}
        </li>
      ))}
    </ul>
  )
}
