import { ArrowLeft, ExternalLink, MapPin } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useParams } from 'react-router'
import { BookingDialog } from '@/components/BookingDialog'
import { DayPicker } from '@/components/DayPicker'
import { FacilityMap } from '@/components/FacilityMap'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { api, ApiError, type Facility, type Slot } from '@/lib/api'
import { directionsUrl } from '@/lib/maps'
import { formatBookingDay, formatTime, localDate } from '@/lib/time'
import { usePageTitle } from '@/hooks/usePageTitle'

const WEEK = [1, 2, 3, 4, 5, 6, 7] // ISO weekdays, Monday first

/** /:slug/:facility: one facility, its rules and its turns. Booking always shows the rules first. */
export function FacilityPage() {
  const { slug = '', facility: facilitySlug = '' } = useParams()
  const { t } = useTranslation(['facility', 'bookings', 'common'])
  const auth = useAuth()
  const { login } = useLogin()
  const tenants = useAsync(() => api.tenants(), 'tenants')

  const [picked, setPicked] = useState<{ key: string; date: string } | null>(null)
  const key = `${slug}/${facilitySlug}`
  const date = picked?.key === key ? picked.date : undefined
  const page = useAsync(() => api.facilityPage(slug, facilitySlug, date), `facility:${key}:${date ?? 'today'}`)
  const [pick, setPick] = useState<{ facility: Facility; slot: Slot } | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  usePageTitle(page.data?.facility.name, page.data?.tenant.name)

  const data = page.data
  const timeZone = data?.timezone ?? 'Europe/Madrid'
  const today = data ? localDate(data.now, data.timezone) : ''

  const confirm = async (facility: Facility, slot: Slot) => {
    const token = auth.user?.access_token
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
      page.reload()
      return null
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) page.reload()
      return error instanceof Error ? error.message : t('bookings:dialog.error')
    }
  }

  const shell = (children: React.ReactNode) => (
    <div className="min-h-screen">
      <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={slug} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-6 pb-16 sm:px-6">
        <Link to={`/${slug}`} className="inline-flex items-center gap-1.5 text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" aria-hidden="true" />
          {tenants.data?.tenants.find((c) => c.slug === slug)?.name ?? t('facility:back')}
        </Link>
        {children}
      </main>
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
  const hasPin = f.latitude !== null && f.longitude !== null

  return shell(
    <>
      <div className="mt-3">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{f.name}</h1>
        <p className="mt-1 text-base text-muted-foreground">{[f.sport, f.address].filter(Boolean).join(' · ')}</p>
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
              <DayPicker today={today} value={data.date} onChange={(value) => setPicked({ key, date: value })} />
            </div>
            <div aria-live="polite">
              {notice && (
                <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
                  <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
                </Alert>
              )}
            </div>
            {f.windows.length === 0 ? (
              <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('facility:closedDay')}</p>
            ) : (
              <>
                <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4 md:grid-cols-6">
                  {f.slots.map((slot) => (
                    <li key={slot.starts_at}>
                      <Button
                        variant="outline"
                        className="num h-11 w-full disabled:opacity-100 disabled:bg-muted disabled:text-muted-foreground disabled:line-through data-[free=true]:border-primary data-[free=true]:text-primary"
                        data-free={slot.status === 'free'}
                        disabled={slot.status !== 'free'}
                        onClick={() => setPick({ facility: f, slot })}
                        aria-label={`${formatTime(slot.starts_at, timeZone)}–${formatTime(slot.ends_at, timeZone)}`}
                      >
                        {formatTime(slot.starts_at, timeZone)}
                      </Button>
                    </li>
                  ))}
                </ul>
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

          <section aria-labelledby="conditions-title" className="grid gap-2 rounded-lg border bg-card p-5">
            <h2 id="conditions-title" className="text-lg font-semibold">
              {t('facility:conditions')}
            </h2>
            <ul className="num grid gap-1.5">
              <li>{t('facility:ruleSlot', { count: f.slot_minutes })}</li>
              <li>{t('facility:ruleAdvance', { count: f.max_advance_days })}</li>
              <li>{t('facility:ruleCancel', { count: f.min_cancel_hours })}</li>
              <li>{t('facility:ruleLimit', { count: f.max_active_bookings_per_user })}</li>
              {f.min_age !== null && <li className="font-medium">{t('facility:minAge', { count: f.min_age })}</li>}
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

      <BookingDialog
        key={pick ? pick.slot.starts_at : 'none'}
        pick={pick}
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
