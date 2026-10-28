import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useParams } from 'react-router'
import { Availability } from '@/components/Availability'
import { BookingDialog } from '@/components/BookingDialog'
import { CancelDialog } from '@/components/CancelDialog'
import { DayPicker } from '@/components/DayPicker'
import { MyBookings } from '@/components/MyBookings'
import { BookingQrDialog } from '@/components/BookingQrDialog'
import { CouncilContact } from '@/components/CouncilContact'
import { CouncilHeading } from '@/components/CouncilHeading'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { api, ApiError, type Booking, type Day, type Facility, type Slot } from '@/lib/api'
import { formatBookingDay, formatTime, localDate } from '@/lib/time'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

const FALLBACK_TIME_ZONE = 'Europe/Madrid'

function councilToday(day: Day) {
  return localDate(day.now, day.timezone)
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
  const council = tenants.data?.tenants.find((c) => c.slug === tenantSlug)
  usePageTitle(t('home:title'), council?.name)

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
            <CouncilHeading section={t('home:title')} council={council} otherCouncils={(tenants.data?.tenants.length ?? 0) > 1} />
          </div>
          {today && <DayPicker today={today} value={selectedDate} onChange={setDate} />}
        </div>

        {/*
          Grid at 1440: column 1152 − 2 × 24 padding = 1104 = 740 (availability) + 24 + 340 (bookings).
          On a phone a signed-in resident sees their bookings first: that block carries actions.
        */}
        <div className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px] lg:items-start">
          <div className={auth.isAuthenticated ? 'order-2 lg:order-1' : ''}>
            <Availability
              councilSlug={tenantSlug}
              day={day.data}
              isToday={!!day.data && day.data.date === today}
              loading={day.loading}
              error={day.error}
              onRetry={day.reload}
              onPick={(facility, slot) => setPick({ facility, slot })}
            />
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
            {council && <CouncilContact council={council} />}
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

