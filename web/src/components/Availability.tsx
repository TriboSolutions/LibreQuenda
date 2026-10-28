import { useState } from 'react'
import { Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { AxisLabels, type Axis, DayStrip, StripLegend } from '@/components/DayStrip'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { Day, Facility, Slot } from '@/lib/api'
import { clockToMinutes, formatTime, minutesOfDay } from '@/lib/time'

type Props = {
  councilSlug: string
  day: Day | null
  isToday: boolean
  loading: boolean
  error: Error | null
  onRetry: () => void
  onPick: (facility: Facility, slot: Slot) => void
}

// Desktop row: name 180 + gap 24 + strip (fluid) + gap 24 + the two actions (fit to content)
// One grid for the axis labels and every row (subgrid), so the strips share a width
// whatever each row's actions say.
const columns = 'md:grid md:grid-cols-[180px_minmax(0,1fr)_auto] md:gap-x-6'
const rowGrid = 'md:col-span-3 md:grid md:grid-cols-subgrid md:items-center'

/** One axis for every row, from the earliest opening to the latest closing, on whole two-hour steps. */
function sharedAxis(facilities: Facility[]): Axis | null {
  const windows = facilities.flatMap((f) => f.windows)
  if (windows.length === 0) return null
  const start = Math.floor(Math.min(...windows.map((w) => clockToMinutes(w.opens_at))) / 60) * 60
  let end = Math.ceil(Math.max(...windows.map((w) => clockToMinutes(w.closes_at))) / 60) * 60
  if ((end - start) % 120 !== 0) end += 60
  return { start, end }
}

// Quick booking is for walking in now: today, and a free turn starting within this hour.
const QUICK_BOOK_WITHIN_MS = 60 * 60_000

function FacilityRow({ councilSlug, facility, axis, timeZone, now, nowMinutes, onPick }: {
  councilSlug: string
  facility: Facility
  axis: Axis
  timeZone: string
  now: string | null // only when the day shown is today
  nowMinutes: number | null
  onPick: (facility: Facility, slot: Slot) => void
}) {
  const { t } = useTranslation('home')
  const [showAll, setShowAll] = useState(false)
  const free = facility.slots.filter((s) => s.status === 'free')
  const closedToday = facility.windows.length === 0
  const visibleTimes = showAll ? free : free.slice(0, 4)
  // "Full" only while turns remain and people booked them all; after hours, nothing
  const full = free.length === 0 && facility.slots.some((s) => s.status === 'booked')
  const afterHours = now !== null && facility.slots.length > 0 && facility.slots.every((s) => s.status === 'past')
  const quick = now && free[0] && Date.parse(free[0].starts_at) - Date.parse(now) < QUICK_BOOK_WITHIN_MS ? free[0] : null

  return (
    <li className={`border-t py-3 first:border-t-0 ${rowGrid}`}>
      <div className="min-w-0">
        <Link to={`/${councilSlug}/${facility.slug}`} className="block truncate font-medium hover:text-primary hover:underline">
          {facility.name}
        </Link>
        <p className="truncate text-muted-foreground">
          {facility.sport} · {t('availability.minutes', { count: facility.slot_minutes })} ·{' '}
          <span className="num">{closedToday ? t('availability.closedShort') : t('availability.free', { count: free.length })}</span>
        </p>
      </div>

      {/* desktop: the day strip */}
      <div className="hidden md:block">
        {closedToday ? (
          <p className="text-muted-foreground">{t('availability.closedDay')}</p>
        ) : (
          <DayStrip facility={facility} axis={axis} timeZone={timeZone} nowMinutes={nowMinutes} onPick={(slot) => onPick(facility, slot)} />
        )}
      </div>
      <div className="hidden items-center justify-end gap-1 md:flex">
        {quick && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="num"
                aria-label={t('availability.quickBook', { time: formatTime(quick.starts_at, timeZone) })}
                onClick={() => onPick(facility, quick)}
              >
                <Zap aria-hidden="true" />
                {formatTime(quick.starts_at, timeZone)}
              </Button>
            </TooltipTrigger>
            <TooltipContent className="num">{t('availability.quickBook', { time: formatTime(quick.starts_at, timeZone) })}</TooltipContent>
          </Tooltip>
        )}
        {full && <span className="px-3 text-muted-foreground">{t('availability.full')}</span>}
        {afterHours && <span className="px-3 text-muted-foreground">{t('availability.afterHours')}</span>}
        <Button asChild variant="ghost" size="sm">
          <Link to={`/${councilSlug}/${facility.slug}`} aria-label={t('availability.details', { facility: facility.name })}>
            {t('availability.book')}
          </Link>
        </Button>
      </div>

      {/* phone: the free times as targets a thumb can hit */}
      {free.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-2 md:hidden">
          {visibleTimes.map((slot) => (
            <Button key={slot.starts_at} variant="outline" className="num h-11 min-w-16" onClick={() => onPick(facility, slot)}>
              {formatTime(slot.starts_at, timeZone)}
            </Button>
          ))}
          {!showAll && free.length > visibleTimes.length && (
            <Button variant="ghost" className="h-11" onClick={() => setShowAll(true)}>
              {t('availability.more', { count: free.length - visibleTimes.length })}
            </Button>
          )}
          <Button asChild variant="ghost" className="h-11">
            <Link to={`/${councilSlug}/${facility.slug}`}>{t('availability.book')}</Link>
          </Button>
        </div>
      )}
    </li>
  )
}

export function Availability({ councilSlug, day, isToday, loading, error, onRetry, onPick }: Props) {
  const { t } = useTranslation(['home', 'common'])
  const axis = day ? sharedAxis(day.facilities) : null
  const nowMinutes = day && isToday ? minutesOfDay(day.now, day.timezone) : null

  return (
    <section aria-labelledby="availability-title" className="rounded-lg border bg-card p-5">
      <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
        <h2 id="availability-title" className="text-lg font-semibold">
          {t('home:availability.title')}
        </h2>
        <StripLegend />
      </div>

      {error ? (
        <div className="mt-4 flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
          <p>
            {t('home:availability.loadError')} {error.message}
          </p>
          <Button variant="ghost" size="sm" onClick={onRetry}>
            {t('common:retry')}
          </Button>
        </div>
      ) : loading && !day ? (
        <ul className={`mt-4 ${columns}`} aria-busy="true" aria-label={t('home:availability.loading')}>
          {Array.from({ length: 6 }, (_, i) => (
            <li key={i} className={`border-t py-3 first:border-t-0 ${rowGrid}`}>
              <div className="space-y-1.5">
                <Skeleton className="h-4 w-36" />
                <Skeleton className="h-4 w-28" />
              </div>
              <Skeleton className="hidden h-8 md:block" />
              <Skeleton className="ml-auto hidden h-8 w-28 md:block" />
            </li>
          ))}
        </ul>
      ) : day && day.facilities.length === 0 ? (
        <p className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">
          {t('home:availability.empty')}
        </p>
      ) : day && axis ? (
        <div className={`mt-4 ${columns}`}>
          <div className={`${rowGrid} hidden`}>
            <span />
            <AxisLabels axis={axis} />
            <span />
          </div>
          <ul className={`mt-1 ${rowGrid}`}>
            {day.facilities.map((facility) => (
              <FacilityRow
                key={facility.id}
                councilSlug={councilSlug}
                facility={facility}
                axis={axis}
                timeZone={day.timezone}
                now={isToday ? day.now : null}
                nowMinutes={nowMinutes}
                onPick={onPick}
              />
            ))}
          </ul>
        </div>
      ) : (
        day && (
          <p className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">
            {t('home:availability.allClosed')}
          </p>
        )
      )}
    </section>
  )
}
