import { ArrowLeft, ExternalLink, MapPin, PartyPopper } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useParams, useSearchParams } from 'react-router'
import { RatingBadge } from '@/components/Stars'
import { ResidentRequestDialog, ResidentsOnlyBadge } from '@/components/ResidentRequestDialog'
import { PublicTurnDialog } from '@/components/PublicTurnDialog'
import { BookingDialog } from '@/components/BookingDialog'
import { DatePicker } from '@/components/DatePicker'
import { DayPicker } from '@/components/DayPicker'
import { CouncilContact, FacilityContact } from '@/components/CouncilContact'
import { FacilityMap } from '@/components/FacilityMap'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { WeatherStrip } from '@/components/Weather'
import { useNow } from '@/hooks/useNow'
import { useAsync } from '@/hooks/useAsync'
import { useForecast } from '@/hooks/useForecast'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { api, ApiError, type Facility, type PublicTurn, type Slot } from '@/lib/api'
import { directionsUrl } from '@/lib/maps'
import { addDays, formatBookingDay, formatTime, localDate } from '@/lib/time'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

const WEEK = [1, 2, 3, 4, 5, 6, 7] // ISO weekdays, Monday first
const DATE = /^\d{4}-\d{2}-\d{2}$/

/**
 * /:slug/:facility: one facility, its rules and its turns. Booking always shows the rules first.
 * ?dia=YYYY-MM-DD opens a given day.
 */
export function FacilityPage() {
  const { slug = '', facility: facilitySlug = '' } = useParams()
  const { t } = useTranslation(['facility', 'bookings', 'common'])
  const auth = useAuth()
  const { login } = useLogin()
  const { me } = useMe()

  const [searchParams] = useSearchParams()
  const linked = searchParams.get('dia') ?? ''
  const [picked, setPicked] = useState<{ key: string; date: string } | null>(null)
  const key = `${slug}/${facilitySlug}`
  const date = picked?.key === key ? picked.date : DATE.test(linked) ? linked : undefined
  const token = auth.user?.access_token
  const page = useAsync(() => api.facilityPage(slug, facilitySlug, date, token), `facility:${key}:${date ?? 'today'}:${token ? 'in' : 'out'}`)
  const [pick, setPick] = useState<{ facility: Facility; slot: Slot } | null>(null)
  const [publicPick, setPublicPick] = useState<PublicTurn | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [asking, setAsking] = useState(false)
  usePageTitle(page.data?.facility.name, page.data?.tenant.name)

  const ticking = useNow()
  const currentTime = Math.max(ticking, Date.parse(page.data?.now ?? '') || 0)
  const data = page.data
  const timeZone = data?.timezone ?? 'Europe/Madrid'
  // the day's forecast, for an outdoor facility (MeteoGalicia)
  const forecast = useForecast(slug, facilitySlug, data?.facility.show_weather ?? false, data?.date ?? null)
  const today = data ? localDate(data.now, data.timezone) : ''
  let lastBookingDate = today
  let showCalendar = false
  if (data) {
    const opening = data.facility.weekly_opening
    if (opening) {
      const weekday = new Date(`${today}T12:00:00Z`).getUTCDay() || 7
      const opened = weekday > opening.weekday || (weekday === opening.weekday && formatTime(data.now, timeZone) >= opening.at)
      lastBookingDate = addDays(today, (opened ? 14 : 7) - weekday)
      showCalendar = lastBookingDate > addDays(today, 6)
    } else {
      lastBookingDate = localDate(new Date(Date.parse(data.now) + data.facility.max_advance_hours * 3_600_000).toISOString(), timeZone)
      showCalendar = data.facility.max_advance_hours > 7 * 24
    }
  }


  const confirm = async (facility: Facility, slot: Slot, isPublic: boolean, description: string) => {
    const token = auth.user?.access_token
    if (!token) return t('bookings:dialog.needLogin')
    try {
      const booking = await api.book(token, facility.id, slot.starts_at, true, isPublic, description) // the dialog only enables booking once the rules are accepted
      setPick(null)
      setNotice(
        t('bookings:notice.booked', {
          facility: facility.name,
          day: formatBookingDay(booking.starts_at, timeZone, today),
          time: formatTime(booking.starts_at, timeZone),
        }),
      )
      page.reload()
      return null
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) page.reload()
      return error instanceof Error ? error.message : t('bookings:dialog.error')
    }
  }

  const shell = (children: React.ReactNode) => (
    <div className="min-h-screen">
      <TopBar tenantSlug={slug} tenantName={page.data?.tenant.name} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-6 pb-16 sm:px-6">
        <Link to={`/${slug}`} className="inline-flex items-center gap-1.5 text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" aria-hidden="true" />
          {page.data?.tenant.name ?? t('facility:back')}
        </Link>
        {children}
      </main>
      <SiteFooter />
    </div>
  )

  if (page.error instanceof ApiError && page.error.status === 404) {
    return shell(
      <>
        <h1 className="mt-3 text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('facility:notFoundTitle')}</h1>
        <p className="mt-1 text-base text-muted-foreground">{t('facility:notFoundBody', { path: `/${slug}/${facilitySlug}` })}</p>
      </>,
    )
  }
  if (!data) {
    return shell(page.error ? <p className="mt-4 text-destructive">{page.error.message}</p> : <Skeleton className="mt-4 h-72 w-full" />)
  }

  const f = data.facility
  const free = f.slots.filter((s) => s.status === 'free')
  const events = (f.public_bookings ?? []).filter((b) => b.kind === 'event' && Date.parse(b.ends_at) > currentTime)
  const hasPin = f.latitude !== null && f.longitude !== null

  return shell(
    <>
      <div className="mt-3">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{f.name}</h1>
        <p className="mt-1 text-base text-muted-foreground">{[f.sport, f.address].filter(Boolean).join(' · ')}</p>
        <RatingBadge rating={data.rating} className="mt-1" />
        {f.residents_only && <ResidentsOnlyBadge className="mt-2" />}
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px] lg:items-start">
        <div className="grid gap-6">
          {(f.image_url || f.description) && (
            <section className="overflow-hidden rounded-lg border bg-card">
              {f.image_url && <img src={f.image_url} alt="" className="aspect-[16/7] w-full object-cover" />}
              {f.description && <p className="p-5 text-base whitespace-pre-line">{f.description}</p>}
            </section>
          )}


          <section aria-labelledby="book-title" className="grid gap-4 rounded-lg border bg-card p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 id="book-title" className="text-lg font-semibold">
                {t('facility:bookTitle')}
              </h2>
              <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto">
                <DayPicker today={today} value={data.date} onChange={(value) => setPicked({ key, date: value })} />
                {showCalendar && (
                  <DatePicker
                    value={data.date > addDays(today, 6) ? data.date : ''}
                    placeholder={t('facility:chooseDate')}
                    defaultMonth={data.date}
                    min={today}
                    max={lastBookingDate}
                    onChange={(value) => setPicked({ key, date: value })}
                  />
                )}
              </div>
            </div>
            {forecast && <WeatherStrip forecast={forecast} />}
            {/* residents only: say so before anyone picks a turn the API would refuse */}
            {f.residents_only && !me?.resident_of?.some((r) => r.tenant_slug === slug) && (
              <Alert className="border-0 bg-pill-neutral-bg text-pill-neutral-ink">
                <AlertDescription className="grid gap-3 text-pill-neutral-ink">
                  <span>
                    {!auth.isAuthenticated
                      ? t('facility:residentsOnlySignIn', { council: data.tenant.name })
                      : me?.resident_requests?.some((r) => r.tenant_slug === slug)
                        ? t('facility:residentRequest.pendingShort', { council: data.tenant.name })
                        : t('facility:residentsOnlyNotVerified', { council: data.tenant.name })}
                  </span>
                  {auth.isAuthenticated ? (
                    <Button type="button" variant="outline" className="w-fit" onClick={() => setAsking(true)}>
                      {me?.resident_requests?.some((r) => r.tenant_slug === slug) ? t('facility:residentRequest.see') : t('facility:residentRequest.ask')}
                    </Button>
                  ) : (
                    <Button type="button" variant="outline" className="w-fit" onClick={login}>
                      {t('facility:residentRequest.signIn')}
                    </Button>
                  )}
                </AlertDescription>
              </Alert>
            )}
            <div aria-live="polite">
              {notice && (
                <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
                  <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
                </Alert>
              )}
            </div>
            {/* the council's events this day, also those outside the opening hours */}
            {!!events.length && (
              <section aria-labelledby="events-title" className="grid gap-2">
                <h3 id="events-title" className="font-medium">{t('facility:events.title')}</h3>
                <ul className="grid gap-2">
                  {events.map((event) => {
                    const left = Math.max(event.capacity - event.people_count, 0)
                    return (
                      <li key={event.id}>
                        <button
                          type="button"
                          onClick={() => setPublicPick(event)}
                          className="flex w-full min-w-0 items-start gap-3 rounded-md border border-pill-accent-ink bg-pill-accent-bg px-3 py-2 text-left text-pill-accent-ink hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                        >
                          <PartyPopper className="mt-0.5 size-4 shrink-0" aria-hidden />
                          <span className="grid min-w-0 gap-0.5">
                            <span className="font-semibold break-words">{event.title}</span>
                            {f.residents_only && <span className="text-sm font-medium">{t('facility:residentRequest.badge')}</span>}
                            <span className="num text-sm">
                              {formatTime(event.starts_at, timeZone)}–{formatTime(event.ends_at, timeZone)} ·{' '}
                              {Date.parse(event.starts_at) <= currentTime ? t('facility:publicSlot.ongoing') : left > 0 ? t('facility:events.spots', { count: left }) : t('facility:publicSlot.full')}
                            </span>
                          </span>
                        </button>
                      </li>
                    )
                  })}
                </ul>
              </section>
            )}
            {f.windows.length === 0 ? (
              <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('facility:closedDay')}</p>
            ) : (
              <>
                <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-6">
                  {f.slots.map((slot) => {
                    const publicBooking = f.public_bookings?.find((b) => Date.parse(b.ends_at) > currentTime && Date.parse(b.starts_at) < Date.parse(slot.ends_at) && Date.parse(b.ends_at) > Date.parse(slot.starts_at))
                    const publicState = publicBooking
                      ? Date.parse(publicBooking.starts_at) <= currentTime ? 'ongoing' : publicBooking.people_count >= publicBooking.capacity ? 'full' : 'joinable'
                      : undefined
                    const publicLabel = publicBooking?.kind === 'event' ? t('facility:events.label') : t('bookings:public.label')
                    return (
                    <li key={slot.starts_at}>
                      <Button
                        variant={publicBooking ? 'ghost' : 'outline'}
                        className="num h-14 w-full flex-col gap-0.5 border disabled:opacity-100 disabled:bg-muted disabled:text-muted-foreground disabled:line-through data-[free=true]:border-primary data-[free=true]:text-primary data-[public-state=joinable]:bg-pill-positive-bg data-[public-state=joinable]:text-pill-positive-ink data-[public-state=joinable]:border-pill-positive-ink data-[public-state=ongoing]:bg-pill-accent-bg data-[public-state=ongoing]:text-pill-accent-ink data-[public-state=ongoing]:border-pill-accent-ink data-[public-state=full]:bg-muted data-[public-state=full]:text-muted-foreground"
                        data-public-state={publicState}
                        data-public={!!publicBooking}
                        data-free={slot.status === 'free'}
                        disabled={!publicBooking && slot.status !== 'free'}
                        onClick={() => publicBooking ? setPublicPick(publicBooking) : setPick({ facility: f, slot })}
                        // with a public booking or an event, the name is the visible text plus what it is (WCAG 2.5.3)
                        aria-label={publicBooking ? undefined : `${formatTime(slot.starts_at, timeZone)}–${formatTime(slot.ends_at, timeZone)}`}
                      >
                        <span>
                          {formatTime(slot.starts_at, timeZone)}
                          {publicBooking && <span className="sr-only">–{formatTime(slot.ends_at, timeZone)} · {publicLabel}{publicBooking.kind === 'event' ? ` «${publicBooking.title}»` : ''} ·</span>}
                        </span>
                        {publicState && <span className="text-[11px] leading-tight">{t(`facility:publicSlot.${publicState}`)}</span>}
                      </Button>
                    </li>
                  )})}
                </ul>
                {!!f.public_bookings?.length && (
                  <ul className="flex flex-wrap gap-x-4 gap-y-2 text-xs text-muted-foreground">
                    <li className="flex items-center gap-2"><span aria-hidden className="size-3 rounded-sm border border-pill-positive-ink bg-pill-positive-bg" />{t('facility:publicSlot.joinableLegend')}</li>
                    <li className="flex items-center gap-2"><span aria-hidden className="size-3 rounded-sm border border-pill-accent-ink bg-pill-accent-bg" />{t('facility:publicSlot.ongoingLegend')}</li>
                  </ul>
                )}
                {free.length === 0 && <p className="text-muted-foreground">{t('facility:noFree')}</p>}
                {!auth.isAuthenticated && free.length > 0 && <p className="text-muted-foreground">{t('facility:signInToBook')}</p>}
              </>
            )}
          </section>

        </div>

        <aside className="grid gap-6">
          <section aria-labelledby="where-title" className="grid gap-3 rounded-lg border bg-card p-5">
            <h2 id="where-title" className="text-lg font-semibold">
              {t('facility:location')}
            </h2>
            {hasPin ? (
              <>
                <FacilityMap latitude={f.latitude} longitude={f.longitude} label={t('facility:mapLabel')} />
                {f.address && (
                  <p className="flex items-start gap-2">
                    <MapPin className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                    {f.address}
                  </p>
                )}
                <Button asChild variant="outline">
                  <a href={directionsUrl(f.latitude!, f.longitude!)} target="_blank" rel="noopener noreferrer">
                    {t('facility:directions')}
                    <ExternalLink aria-hidden="true" />
                    <span className="sr-only">{t('common:newTab')}</span>
                  </a>
                </Button>
              </>
            ) : (
              <p className="text-muted-foreground">{f.address || t('facility:noLocation')}</p>
            )}
          </section>

          {/* its own contact when it has one, and the council's always, each in its own panel */}
          {(f.contact_phone || f.contact_email) && <FacilityContact facility={f} />}
          <CouncilContact tenant={data.tenant} />

          <section aria-labelledby="conditions-title" className="grid gap-2 rounded-lg border bg-card p-5">
            <h2 id="conditions-title" className="text-lg font-semibold">
              {t('facility:conditions')}
            </h2>
            <ul className="num grid gap-1.5">
              <li>{t('facility:ruleSlot', { count: f.slot_minutes })}</li>
              <li>
                {f.weekly_opening
                  ? t('facility:ruleWeekly', { weekday: t(`common:date.weekdays.${f.weekly_opening.weekday % 7}`), time: f.weekly_opening.at })
                  : f.max_advance_hours % 24 === 0
                  ? t('facility:ruleAdvance', { count: f.max_advance_hours / 24 })
                  : t('facility:ruleAdvanceHours', { count: f.max_advance_hours })}
              </li>
              {f.min_advance_minutes > 0 && <li>{t('facility:ruleMinAdvance', { count: f.min_advance_minutes })}</li>}
              <li>{t('facility:ruleCancel', { count: f.min_cancel_hours })}</li>
              <li>{t('facility:ruleLimit', { count: f.max_active_bookings_per_user })}</li>
              {f.max_bookings_per_day > 0 && <li>{t('facility:rulePerDay', { count: f.max_bookings_per_day })}</li>}
              {f.max_turns_in_a_row > 0 && <li>{t('facility:ruleInARow', { count: f.max_turns_in_a_row })}</li>}
              {f.max_bookings_per_week > 0 && <li>{t('facility:rulePerWeek', { count: f.max_bookings_per_week })}</li>}
              {f.council_max_active_bookings > 0 && <li>{t('facility:ruleCouncilActive', { count: f.council_max_active_bookings })}</li>}
              {f.min_age !== null && <li className="font-medium">{t('facility:minAge', { count: f.min_age })}</li>}
              {f.residents_only && <li className="font-medium">{t('facility:residentsOnly', { council: data.tenant.name })}</li>}
            </ul>
          </section>

          <section aria-labelledby="rules-title" className="grid gap-2 rounded-lg border bg-card p-5">
            <h2 id="rules-title" className="text-lg font-semibold">
              {t('facility:rules')}
            </h2>
            <p className="whitespace-pre-line">{f.usage_rules || t('facility:noRules')}</p>
          </section>

          <section aria-labelledby="schedule-title" className="grid gap-2 rounded-lg border bg-card p-5">
            <h2 id="schedule-title" className="text-lg font-semibold">
              {t('facility:schedule')}
            </h2>
            <dl className="num grid grid-cols-[auto_1fr] gap-x-6 gap-y-1">
              {WEEK.map((day) => {
                const windows = data.schedule.filter((w) => w.weekday === day)
                return (
                  <div key={day} className="contents">
                    <dt className="capitalize text-muted-foreground">{t(`common:date.weekdays.${day % 7}`)}</dt>
                    <dd className="text-right">
                      {windows.length === 0 ? t('facility:closed') : windows.map((w) => `${w.opens_at}–${w.closes_at}`).join(', ')}
                    </dd>
                  </div>
                )
              })}
            </dl>
          </section>
        </aside>
      </div>

      {f.residents_only && <ResidentRequestDialog key={`${slug}:${asking}`} tenant={data.tenant} open={asking} onClose={() => setAsking(false)} />}
      {publicPick && <PublicTurnDialog
        key={publicPick.id}
        booking={publicPick}
        facility={f}
        timeZone={timeZone}
        today={today}
        now={new Date(currentTime).toISOString()}
        onClose={() => setPublicPick(null)}
        onJoined={page.reload}
        onLogin={login}
      />}
      <BookingDialog
        key={pick ? pick.slot.starts_at : 'none'}
        pick={pick}
        tenantSlug={slug}
        timeZone={timeZone}
        today={today}
        authenticated={auth.isAuthenticated}
        onClose={() => setPick(null)}
        onConfirm={confirm}
        onLogin={login}
      />
    </>,
  )
}
