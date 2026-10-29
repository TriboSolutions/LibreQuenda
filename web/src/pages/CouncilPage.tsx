import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import i18n from '@/i18n'
import { useAuth } from 'react-oidc-context'
import { Link, useParams, useSearchParams } from 'react-router'
import { Availability } from '@/components/Availability'
import { BookingDialog } from '@/components/BookingDialog'
import { CancelDialog } from '@/components/CancelDialog'
import { DayPicker } from '@/components/DayPicker'
import { MyBookings } from '@/components/MyBookings'
import { TagFilters } from '@/components/TagFilters'
import { BookingQrDialog } from '@/components/BookingQrDialog'
import { CouncilContact } from '@/components/CouncilContact'
import { CouncilHeading } from '@/components/CouncilHeading'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { useTags } from '@/hooks/useTags'
import { api, ApiError, type Booking, type Day, type Facility, type Slot } from '@/lib/api'
import { formatBookingDay, formatLongDate, formatTime, localDate } from '@/lib/time'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

const FALLBACK_TIME_ZONE = 'Europe/Madrid'

function councilToday(day: Day) {
  return localDate(day.now, day.timezone)
}

/** The line under the title: the day, how much is free, and the soonest free turn. */
function finding(day: Day, today: string) {
  const t = i18n.t
  const free = day.facilities.flatMap((f) => f.slots.filter((s) => s.status === 'free').map((slot) => ({ facility: f, slot })))
  const date = formatLongDate(day.date)
  const when = day.date === today ? t('home:finding.today', { date }) : date.charAt(0).toUpperCase() + date.slice(1)
  if (day.facilities.every((f) => f.windows.length === 0)) return t('home:finding.closed', { when })
  if (free.length === 0) return t('home:finding.full', { when })

  const withFree = new Set(free.map((f) => f.facility.id)).size
  const first = free.reduce((a, b) => (a.slot.starts_at <= b.slot.starts_at ? a : b))
  return t('home:finding.free', {
    when,
    count: free.length,
    facilities: t('home:finding.facilities', { count: withFree }),
    facility: first.facility.name,
    time: formatTime(first.slot.starts_at, day.timezone),
  })
}

/** /:slug: one council's facilities, availability and the resident's bookings. */
export function CouncilPage() {
  const { slug: tenantSlug = '' } = useParams()
  const { t } = useTranslation(['home', 'bookings', 'councils'])
  const auth = useAuth()
  const token = auth.user?.access_token
  const subject = auth.user?.profile.sub ?? null

  const tenants = useAsync(() => api.tenants(), 'tenants')
  const { login, register } = useLogin()
  const councilName = tenants.data?.tenants.find((c) => c.slug === tenantSlug)?.name
  usePageTitle(t('home:title'), councilName)

  // The chosen day belongs to the council it was picked for
  const [picked, setPicked] = useState<{ slug: string; date: string } | null>(null)
  const date = picked?.slug === tenantSlug ? picked.date : undefined
  const setDate = (value: string) => setPicked({ slug: tenantSlug, date: value })
  const day = useAsync(() => api.day(tenantSlug, date), `day:${tenantSlug}:${date ?? 'today'}`)

  // GET /me (MeProvider) first creates the user's row; the bookings load after it
  const me = useMe()
  const bookings = useAsync(token && me.me ? () => api.myBookings(token) : null, `bookings:${subject}:${me.me?.id}`)

  const [pick, setPick] = useState<{ facility: Facility; slot: Slot } | null>(null)
  const [toCancel, setToCancel] = useState<Booking | null>(null)
  const [showQr, setShowQr] = useState<Booking | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const timeZone = day.data?.timezone ?? FALLBACK_TIME_ZONE
  const today = day.data ? councilToday(day.data) : ''
  const selectedDate = day.data?.date ?? date ?? ''

  // ?etiqueta=<tag>&etiqueta=…: only the facilities with all of them (kept in the address, so a filtered list can be shared)
  const [params, setParams] = useSearchParams()
  const catalogue = useTags()
  const groups = day.data ? catalogue.grouped(day.data.facilities.flatMap((f) => f.tags)) : []
  const inUse = groups.flatMap((g) => g.tags.map((tag) => tag.slug))
  const chosen = params.getAll('etiqueta').filter((slug) => inUse.includes(slug))
  const setChosen = (slugs: string[]) => setParams(new URLSearchParams(slugs.map((slug) => ['etiqueta', slug])), { replace: true })
  const shownDay = day.data && chosen.length > 0 ? { ...day.data, facilities: day.data.facilities.filter((f) => chosen.every((slug) => f.tags.includes(slug))) } : day.data


  const confirmBooking = async (facility: Facility, slot: Slot) => {
    if (!token) return t('bookings:dialog.needLogin')
    try {
      const booking = await api.book(token, facility.id, slot.starts_at, true) // the dialog only enables booking once the rules are accepted
      setPick(null)
      setNotice(
        t('bookings:notice.booked', {
          facility: facility.name,
          day: formatBookingDay(booking.starts_at, timeZone, today),
          time: formatTime(booking.starts_at, timeZone),
        }),
      )
      day.reload()
      bookings.reload()
      return null
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) day.reload() // the strip was stale
      return error instanceof Error ? error.message : t('bookings:dialog.error')
    }
  }

  const confirmCancel = async (booking: Booking) => {
    if (!token) return t('bookings:cancelDialog.needLogin')
    try {
      await api.cancel(token, booking.id)
      setToCancel(null)
      setNotice(t('bookings:notice.cancelled', { facility: booking.facility_name, day: formatBookingDay(booking.starts_at, timeZone, today) }))
      day.reload()
      bookings.reload()
      return null
    } catch (error) {
      return error instanceof Error ? error.message : t('bookings:cancelDialog.error')
    }
  }

  if (day.error instanceof ApiError && day.error.status === 404) {
    return (
      <div className="min-h-screen">
        <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={null} />
        <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
          <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('councils:notFound.title')}</h1>
          <p className="mt-1 max-w-[62ch] text-base text-muted-foreground">{t('councils:notFound.body', { slug: tenantSlug })}</p>
          <Button asChild className="mt-6">
            <Link to="/">{t('councils:allCouncils')}</Link>
          </Button>
        </main>
        <SiteFooter />
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={tenantSlug} />

      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div className="min-w-0">
            <CouncilHeading
              section={t('home:title')}
              council={tenants.data?.tenants.find((c) => c.slug === tenantSlug)}
              otherCouncils={(tenants.data?.tenants.length ?? 0) > 1}
            />
            {day.data ? (
              <p className="mt-1 max-w-[62ch] text-base text-muted-foreground">{finding(day.data, today)}</p>
            ) : (
              <Skeleton className="mt-2 h-5 w-80 max-w-full" />
            )}
          </div>
          {today && <DayPicker today={today} value={selectedDate} onChange={setDate} />}
        </div>

        {groups.length > 0 && (
          <div role="group" aria-label={t('home:tags.label')} className="mt-6 flex flex-wrap items-center gap-2">
            <TagFilters groups={groups} chosen={chosen} onChange={setChosen} />
          </div>
        )}

        {/*
          Grid at 1440: column 1152 − 2 × 24 padding = 1104 = 740 (availability) + 24 + 340 (bookings).
          On a phone a signed-in resident sees their bookings first: that block carries actions.
        */}
        <div className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px] lg:items-start">
          <div className={auth.isAuthenticated ? 'order-2 lg:order-1' : ''}>
            {shownDay && shownDay.facilities.length === 0 && chosen.length > 0 ? (
              <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('home:tags.noMatch')}</p>
            ) : (
              <Availability
                councilSlug={tenantSlug}
                day={shownDay}
                isToday={!!day.data && day.data.date === today}
                loading={day.loading}
                error={day.error}
                onRetry={day.reload}
                onPick={(facility, slot) => setPick({ facility, slot })}
              />
            )}
          </div>
          <div className={`space-y-6 ${auth.isAuthenticated ? 'order-1 lg:order-2' : ''}`}>
            <MyBookings
              authenticated={auth.isAuthenticated}
              data={bookings.data}
              loading={me.loading || bookings.loading}
              error={me.error ?? bookings.error}
              timeZone={timeZone}
              today={today}
              notice={notice}
              onRetry={() => (me.error ? me.reload() : bookings.reload())}
              onLogin={login}
              onRegister={register}
              onCancel={setToCancel}
              onShowQr={setShowQr}
            />
            {day.data && <CouncilContact tenant={day.data.tenant} />}
          </div>
        </div>
      </main>
      <SiteFooter />

      <BookingDialog
        key={pick ? `${pick.facility.id}:${pick.slot.starts_at}` : 'none'}
        pick={pick}
        timeZone={timeZone}
        today={today}
        authenticated={auth.isAuthenticated}
        onClose={() => setPick(null)}
        onConfirm={confirmBooking}
        onLogin={login}
      />
      <BookingQrDialog key={showQr?.id ?? 'none'} booking={showQr} timeZone={timeZone} today={today} onClose={() => setShowQr(null)} />
      <CancelDialog
        key={toCancel?.id ?? 'none'}
        booking={toCancel}
        timeZone={timeZone}
        today={today}
        onClose={() => setToCancel(null)}
        onConfirm={confirmCancel}
      />
    </div>
  )
}

