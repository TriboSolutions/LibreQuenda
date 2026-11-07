import { CalendarClock, CalendarPlus, ChevronDown, DoorOpen, ExternalLink, MapPin, Star, UserPlus, Users, Wrench } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useSearchParams } from 'react-router'
import { BookingOutcome } from '@/components/BookingOutcome'
import { CancelDialog } from '@/components/CancelDialog'
import { CheckinDialog } from '@/components/CheckinDialog'
import { FacilityInfo } from '@/components/FacilityInfo'
import { ChangeBookingDialog } from '@/components/ChangeBookingDialog'
import { InviteDialog } from '@/components/InviteDialog'
import { PeopleDialog } from '@/components/PeopleDialog'
import { ReportIssueDialog } from '@/components/ReportIssueDialog'
import { LeaveDialog } from '@/components/LeaveDialog'
import { RateDialog } from '@/components/RateDialog'
import { PeopleList } from '@/components/PeopleList'
import { SpotsMeter } from '@/components/SpotsMeter'
import { TopBar } from '@/components/TopBar'
import { Pager } from '@/components/Pager'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { useNow } from '@/hooks/useNow'
import { api, type Booking } from '@/lib/api'
import { directionsUrl } from '@/lib/maps'
import { formatBookingDay, formatDateTime, formatDuration, formatTime, localDate } from '@/lib/time'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

// Bookings can span councils; all of them are in Galicia today
const TIME_ZONE = 'Europe/Madrid'

// a booking can be checked in from this long before it starts until it ends, as at a reader
const EARLY_WINDOW_MS = 15 * 60_000

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
  usePageTitle(t('invites:myBookings.title'))
  // GET /me (MeProvider) first creates the user's row; the bookings load after it
  // the history a page at a time (the upcoming ones always come whole)
  const [historyPage, setHistoryPage] = useState(1)
  const bookings = useAsync(token && me.me ? () => api.myBookings(token, historyPage) : null, `bookings:${me.me?.id}:${historyPage}`)

  const [now] = useState(() => new Date().toISOString())
  const today = localDate(now, TIME_ZONE)
  const [toCancel, setToCancel] = useState<Booking | null>(null)
  const [toChange, setToChange] = useState<Booking | null>(null)
  const [toLeave, setToLeave] = useState<Booking | null>(null)
  const [toInvite, setToInvite] = useState<Booking | null>(null)
  const [toSeePeople, setToSeePeople] = useState<Booking | null>(null)
  const [toReport, setToReport] = useState<Booking | null>(null)
  const [toCheckIn, setToCheckIn] = useState<Booking | null>(null)
  // ?valorar=<booking id>: the rating notice opens the dialog straight away
  const [searchParams, setSearchParams] = useSearchParams()
  const toRate = searchParams.get('valorar')
  const rate = (bookingId: string | null) =>
    setSearchParams(
      (params) => {
        if (bookingId) params.set('valorar', bookingId)
        else params.delete('valorar')
        return params
      },
      { replace: true },
    )
  const ticking = useNow()
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

  const changed = (booking: Booking) => {
    setToChange(null)
    setNotice(
      t('bookings:notice.changed', {
        facility: booking.facility_name,
        day: formatBookingDay(booking.starts_at, TIME_ZONE, today),
        time: formatTime(booking.starts_at, TIME_ZONE),
      }),
    )
    bookings.reload()
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
      <div className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] lg:items-start">
        <div aria-live="polite" className="empty:hidden lg:col-span-2">
          {notice && (
            <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
              <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
            </Alert>
          )}
        </div>

        <section aria-labelledby="upcoming-title">
          <h2 id="upcoming-title" className="text-lg font-semibold">
            {t('invites:myBookings.upcoming')} <span className="num font-normal text-muted-foreground">· {upcoming.length}</span>
          </h2>
          {upcoming.length === 0 ? (
            <div className="mt-4 grid justify-items-start gap-3 rounded-lg border border-dashed px-5 py-4">
              <p className="text-muted-foreground">{t('invites:myBookings.emptyUpcoming')}</p>
              <Button asChild>
                <Link to="/">
                  <CalendarPlus aria-hidden />
                  {t('invites:myBookings.bookNow')}
                </Link>
              </Button>
            </div>
          ) : (
            <ul className="mt-4 grid gap-4">
              {upcoming.map((booking) => (
                <BookingRow key={booking.id} booking={booking} today={today} now={ticking} onPeople={() => setToSeePeople(booking)}>
                  {booking.role === 'owner' ? (
                    <>
                      {booking.capacity > 0 && booking.starts_at > now && <PublicSwitch booking={booking} token={token} onChanged={bookings.reload} />}
                      {booking.capacity > 0 && booking.starts_at > now && (
                        <Button variant="outline" size="sm" onClick={() => setToInvite(booking)}>
                          <UserPlus aria-hidden />
                          {t('invites:actions.invite')}
                        </Button>
                      )}
                      {booking.can_change && (
                        <Button variant="outline" size="sm" onClick={() => setToChange(booking)}>
                          <CalendarClock aria-hidden />
                          {t('bookings:change')}
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
                  {booking.status === 'confirmed' && ticking >= Date.parse(booking.starts_at) && ticking < Date.parse(booking.ends_at) && (
                    <Button variant="ghost" size="sm" onClick={() => setToReport(booking)}>
                      <Wrench aria-hidden />
                      {t('facility:issue.report')}
                    </Button>
                  )}
                  {/* the main action last, at the right: public, invite, report, then check in */}
                  {booking.role === 'owner' && booking.checkin_ways?.length ? (
                    <CheckinButton booking={booking} now={ticking} onOpen={() => setToCheckIn(booking)} />
                  ) : null}
                </BookingRow>
              ))}
            </ul>
          )}
        </section>

        <section aria-labelledby="history-title" className="rounded-lg border bg-card p-5">
          <h2 id="history-title" className="text-lg font-semibold">
            {t('invites:myBookings.history')} <span className="num font-normal text-muted-foreground">· {bookings.data?.history_pagination?.total ?? history.length}</span>
          </h2>
          {history.length === 0 ? (
            <p className="mt-4 text-muted-foreground">{t('invites:myBookings.emptyHistory')}</p>
          ) : (
            <ul className="mt-2">
              {history.map((booking) => (
                <HistoryRow key={booking.id} booking={booking} today={today} onRate={() => rate(booking.id)} />
              ))}
            </ul>
          )}
          <div className="mt-3">
            <Pager pagination={bookings.data?.history_pagination} onChange={setHistoryPage} label={t('invites:myBookings.historyPages')} />
          </div>
        </section>
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('invites:myBookings.title')}</h1>
        <p className="mt-1 text-base text-muted-foreground">{t('invites:myBookings.subtitle')}</p>
        {body}
      </main>
      <SiteFooter />

      <CancelDialog key={toCancel?.id ?? 'none'} booking={toCancel} timeZone={TIME_ZONE} today={today} onClose={() => setToCancel(null)} onConfirm={confirmCancel} />
      <ChangeBookingDialog
        key={toChange?.id ?? 'none'}
        booking={toChange}
        timeZone={TIME_ZONE}
        today={today}
        onClose={() => setToChange(null)}
        onChanged={changed}
        onPeopleChanged={bookings.reload}
      />
      <LeaveDialog key={toLeave?.id ?? 'none'} booking={toLeave} timeZone={TIME_ZONE} today={today} onClose={() => setToLeave(null)} onConfirm={confirmLeave} />
      <ReportIssueDialog
        key={`report-${toReport?.id ?? 'none'}`}
        facility={toReport ? { id: toReport.facility_id, name: toReport.facility_name } : null}
        bookingId={toReport?.id ?? ''}
        onClose={() => setToReport(null)}
      />
      <PeopleDialog key={`people-${toSeePeople?.id ?? 'none'}`} booking={toSeePeople} timeZone={TIME_ZONE} today={today} onClose={() => setToSeePeople(null)} onChanged={bookings.reload} />
      <InviteDialog
        key={toInvite?.id ?? 'none'}
        booking={toInvite}
        timeZone={TIME_ZONE}
        today={today}
        onClose={() => setToInvite(null)}
        onChanged={bookings.reload}
      />
      <RateDialog key={`rate-${toRate ?? 'none'}`} bookingId={auth.isAuthenticated ? toRate : null} timeZone={TIME_ZONE} today={today} onClose={() => rate(null)} />
      <CheckinDialog
        key={toCheckIn?.id ?? 'none'}
        booking={toCheckIn}
        token={token}
        timeZone={TIME_ZONE}
        today={today}
        onClose={() => setToCheckIn(null)}
        onCheckedIn={bookings.reload}
      />
    </div>
  )
}

type RowProps = {
  booking: Booking
  today: string
  now: number
  children?: React.ReactNode // actions
  onPeople?: () => void // who is coming, in a dialog (the owner takes people out there)
}

/** Public: anyone signed in finds the booking in «For you» and can join it; private: only by the invite link. */
function PublicSwitch({ booking, token, onChanged }: { booking: Booking; token: string | undefined; onChanged: () => void }) {
  const { t } = useTranslation('bookings')
  const [on, setOn] = useState(!!booking.is_public)
  const [description, setDescription] = useState(booking.public_description ?? '')
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const id = `public-${booking.id}`
  const change = async (next: boolean) => {
    if (!token || busy) return
    setBusy(true)
    setError(null)
    try {
      await api.setPublic(token, booking.id, next, description)
      setOn(next)
      setConfirming(false)
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <>
      <span className="inline-flex items-center gap-2">
        <Switch id={id} checked={on} disabled={busy} onCheckedChange={(next) => {
          setError(null)
          if (next) setConfirming(true)
          else void change(false)
        }} aria-describedby={error && !confirming ? `${id}-error` : undefined} />
        <Label htmlFor={id} className="text-sm font-normal" title={t('public.help')}>
          {t('public.label')}
        </Label>
        {on && <Button variant="ghost" size="sm" onClick={() => { setDescription(booking.public_description ?? ''); setConfirming(true) }}>{t('public.edit')}</Button>}
        {error && !confirming && <span id={`${id}-error`} role="alert" className="text-sm text-destructive">{error}</span>}
      </span>
      <Dialog open={confirming} onOpenChange={(open) => { if (!busy) setConfirming(open) }}>
        <DialogContent className="sm:max-w-md" onCloseAutoFocus={(event) => {
          event.preventDefault()
          document.getElementById(id)?.focus()
        }}>
          <DialogHeader>
            <DialogTitle>{t(on ? 'public.edit' : 'public.confirmTitle')}</DialogTitle>
            <DialogDescription>{t('public.confirmDescription', { facility: booking.facility_name })}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor={`${id}-description`}>{t('public.description')}</Label>
            <Textarea id={`${id}-description`} maxLength={1000} value={description} onChange={(e) => setDescription(e.target.value)} placeholder={t('public.descriptionHelp')} disabled={busy} />
          </div>
          {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
          <DialogFooter>
            <Button variant="outline" disabled={busy} onClick={() => setConfirming(false)}>{t(on ? 'public.cancel' : 'public.keepPrivate')}</Button>
            <Button disabled={busy} onClick={() => void change(true)}>{t(busy ? 'public.publishing' : on ? 'public.save' : 'public.confirm')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

/** Opens every way in the facility takes; usable from the early window until the turn ends. */
function CheckinButton({ booking, now, onOpen }: { booking: Booking; now: number; onOpen: () => void }) {
  const { t } = useTranslation('bookings')
  const opensAt = Date.parse(booking.starts_at) - EARLY_WINDOW_MS
  if (now < opensAt && !booking.checked_in_at) {
    return null
  }
  return (
    <Button variant={booking.checked_in_at ? 'outline' : 'default'} size="sm" onClick={onOpen}>
      <DoorOpen aria-hidden />
      {t('checkin.button')}
    </Button>
  )
}

/** How long until the turn, or where it stands now; refreshed every minute without being read aloud. */
function Countdown({ booking, now }: { booking: Booking; now: number }) {
  const { t } = useTranslation('bookings')
  const starts = Date.parse(booking.starts_at)
  const ends = Date.parse(booking.ends_at)
  let text
  if (booking.checked_in_at) text = t('countdown.in', { time: formatTime(booking.checked_in_at, TIME_ZONE) })
  else if (now >= starts) text = t('countdown.running', { time: formatDuration(ends - now) })
  else if (now >= starts - EARLY_WINDOW_MS) text = t('countdown.open')
  else text = t('countdown.startsIn', { time: formatDuration(starts - now) })
  const live = booking.checked_in_at || now >= starts - EARLY_WINDOW_MS
  return <span className={`num ${live ? pill.positive : pill.neutral}`}>{text}</span>
}

/** A past booking, in one line: what, when, and how it ended; rated from here for 30 days. */
function HistoryRow({ booking, today, onRate }: { booking: Booking; today: string; onRate: () => void }) {
  const { t } = useTranslation('bookings')
  return (
    <li className="grid gap-0.5 border-t py-3 first:border-t-0">
      <div className="flex items-start justify-between gap-3">
        <Link to={`/${booking.tenant_slug}/${booking.facility_slug}`} className="min-w-0 font-medium underline-offset-4 hover:text-primary hover:underline">
          {booking.kind === 'event' ? `${booking.title} · ${booking.facility_name}` : booking.facility_name}
        </Link>
        <BookingOutcome booking={booking} />
      </div>
      <p className="num text-sm text-muted-foreground">
        {formatBookingDay(booking.starts_at, TIME_ZONE, today)}, {formatTime(booking.starts_at, TIME_ZONE)}–{formatTime(booking.ends_at, TIME_ZONE)} · {booking.tenant_name}
      </p>
      {booking.can_rate && (
        <div>
          <Button variant="outline" size="sm" className="mt-1" onClick={onRate}>
            <Star aria-hidden />
            {t('rating.button')}
            <span className="sr-only">: {booking.facility_name}</span>
          </Button>
        </div>
      )}
    </li>
  )
}

/** An upcoming booking with its details in place: photo, where, when, who is coming, and the rules on demand. */
function BookingRow({ booking, today, now, children, onPeople }: RowProps) {
  const { t } = useTranslation(['invites', 'bookings', 'facility', 'common'])
  const [open, setOpen] = useState(false)
  const facility = booking.facility
  const hasPin = facility.latitude !== null && facility.longitude !== null
  const rulesId = `rules-${booking.id}`
  const opensAt = Date.parse(booking.starts_at) - EARLY_WINDOW_MS
  const showEntryTime = booking.role === 'owner' && !!booking.checkin_ways?.length && !booking.checked_in_at && now < opensAt
  const event = booking.kind === 'event' // the council's: a title, no owner taking a spot

  return (
    <li className="grid gap-4 rounded-lg border bg-card p-5">
      <div className="flex gap-4">
        {facility.image_url && <img src={facility.image_url} alt="" className="h-16 w-20 shrink-0 rounded-md object-cover sm:h-24 sm:w-36" />}
        <div className="grid min-w-0 flex-1 content-start gap-1">
          {event && <p className="min-w-0 text-lg font-semibold break-words">{booking.title}</p>}
          <p className="min-w-0 font-medium">
            <Link to={`/${booking.tenant_slug}/${booking.facility_slug}`} className="underline-offset-4 hover:text-primary hover:underline">
              {booking.facility_name}
            </Link>{' '}
            <span className="font-normal text-muted-foreground">· {booking.tenant_name}</span>
          </p>
          <p className="num text-muted-foreground">
            {formatBookingDay(booking.starts_at, TIME_ZONE, today)}, {formatTime(booking.starts_at, TIME_ZONE)}–{formatTime(booking.ends_at, TIME_ZONE)}
            {facility.sport && ` · ${facility.sport}`}
          </p>
          {(facility.address || hasPin) && (
            <p className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
              {facility.address && (
                <span className="inline-flex min-w-0 items-center gap-1.5">
                  <MapPin className="size-3.5 shrink-0" aria-hidden />
                  {facility.address}
                </span>
              )}
              {hasPin && (
                <a
                  href={directionsUrl(facility.latitude!, facility.longitude!)}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 font-medium text-foreground underline-offset-4 hover:underline"
                >
                  {t('facility:directions')}
                  <ExternalLink className="size-3.5" aria-hidden />
                  <span className="sr-only">{t('common:newTab')}</span>
                </a>
              )}
            </p>
          )}
          <p className="mt-1 flex flex-wrap items-center gap-2 text-sm">
            <Countdown booking={booking} now={now} />
            <span className={booking.role === 'owner' ? pill.neutral : pill.positive}>
              {booking.role === 'owner' ? t('role.owner') : event ? t('role.event') : booking.owner_name ? t('role.invitedBy', { name: booking.owner_name }) : t('role.invited')}
            </span>
            {booking.role === 'owner' && (
              // the deadline passed: say so instead of leaving the buttons out silently
              <span className="num text-muted-foreground">
                {t(booking.can_cancel ? 'details.cancelUntil' : 'bookings:locked', { date: formatDateTime(booking.cancel_until, TIME_ZONE) })}
              </span>
            )}
          </p>
        </div>
      </div>

      {booking.capacity > 0 && (
        <div className="max-w-sm">
          <SpotsMeter capacity={booking.capacity} taken={booking.participant_count + (event ? 0 : 1)} />
        </div>
      )}
      {booking.people.length > 0 && (
        <div className="flex flex-wrap items-end gap-x-3 gap-y-1">
          <PeopleList people={booking.people} />
          {onPeople && booking.people.length > 1 && (
            <Button variant="link" size="sm" className="h-auto px-0" onClick={onPeople}>
              <Users aria-hidden />
              {t('invites:people.see')}
            </Button>
          )}
        </div>
      )}

      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-t pt-4">
        <button
          type="button"
          aria-expanded={open}
          aria-controls={rulesId}
          onClick={() => setOpen((v) => !v)}
          className="inline-flex min-h-11 items-center gap-1 rounded-xs text-sm font-medium underline-offset-4 hover:underline"
        >
          {t('details.rules')}
          <ChevronDown className={`size-4 transition-transform ${open ? 'rotate-180' : ''}`} aria-hidden />
        </button>
      </div>
      {open && (
        <div id={rulesId}>
          <FacilityInfo facility={facility} withAddress={false} />
        </div>
      )}
      {(booking.is_public || event) && (
        <div className="grid min-w-0 gap-1">
          <h3 className="text-sm font-medium">{t('bookings:public.activity')}</h3>
          <p className="whitespace-pre-wrap break-words text-sm text-muted-foreground">
            {booking.public_description || t(event ? 'bookings:public.noEventDescription' : 'bookings:public.noDescription')}
          </p>
        </div>
      )}
      <div className="flex flex-wrap items-center justify-between gap-4">
        {showEntryTime && (
          <span className="num text-sm text-muted-foreground">
            {t('bookings:checkin.from', { time: formatTime(new Date(opensAt).toISOString(), TIME_ZONE) })}
          </span>
        )}
        {children && <div className="ml-auto flex flex-wrap items-center justify-end gap-2">{children}</div>}
      </div>
    </li>
  )
}
