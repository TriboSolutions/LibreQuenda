import { ArrowDownRight, ArrowLeft, ArrowRight, ArrowUpRight, Download, Maximize2, Minimize2, Pencil, RefreshCw } from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { Bar, BarChart, CartesianGrid, ComposedChart, Line, XAxis, YAxis } from 'recharts'
import { DatePicker } from '@/components/DatePicker'
import { Button } from '@/components/ui/button'
import { type ChartConfig, ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { type AdminFacilitySummary, api, type Indicators, type IndicatorsCompare, type IndicatorsExport, type IndicatorsQuery } from '@/lib/api'
import { addDays, formatShortDate, formatTime } from '@/lib/time'
import { cn } from '@/lib/utils'
import i18n from '@/i18n'

const PERIODS = ['7', '30', '90', '365', 'custom'] as const
const COMPARE = ['previous', 'year', 'none'] as const
const REFRESH = ['0', '30', '60', '300', '900'] as const // seconds; 0 = only by hand
const REFRESH_KEY = 'lq:indicators-refresh'
const EXCEL_KEY = 'lq:indicators-excel'
const EXPORTS = ['summary', 'daily', 'hours', 'usage'] as const
const ALL = 'all'

function storedExcel(): boolean {
  try {
    return localStorage.getItem(EXCEL_KEY) === '1'
  } catch {
    return false
  }
}

function storedRefresh(): string {
  try {
    const value = localStorage.getItem(REFRESH_KEY)
    if (value && (REFRESH as readonly string[]).includes(value)) return value
  } catch {
    // private window: keep the default
  }
  return '60'
}

/** Today in the browser, "YYYY-MM-DD": the default end of a range of your own. */
function today() {
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function percent(value: number | null) {
  return value === null ? '—' : i18n.t('admin:indicators.percent', { value: Math.round(value * 100) })
}

function hours(value: number) {
  return Math.round(value).toLocaleString(i18n.resolvedLanguage)
}

function decimal(value: number) {
  return value.toLocaleString(i18n.resolvedLanguage, { maximumFractionDigits: 1 })
}

/** How long before playing people book: hours under two days, days after. */
function leadTime(value: number | null) {
  if (value === null) return '—'
  if (value < 48) return i18n.t('admin:indicators.habits.leadHours', { value: Math.round(value) })
  return i18n.t('admin:indicators.habits.leadDays', { value: decimal(value / 24) })
}

/** gl: "1 set 2026" */
function dayMonthYear(date: string) {
  const d = new Date(`${date}T12:00:00Z`)
  return `${d.getUTCDate()} ${i18n.t(`common:date.monthsShort.${d.getUTCMonth()}`)} ${d.getUTCFullYear()}`
}

/** x-axis label of a period: gl "xov 29", "29 out" (week) or "out 2026" (month). */
function periodLabel(period: string, bucket: Indicators['bucket']) {
  const d = new Date(`${period}T12:00:00Z`)
  const month = i18n.t(`common:date.monthsShort.${d.getUTCMonth()}`)
  if (bucket === 'month') return `${month} ${d.getUTCFullYear()}`
  if (bucket === 'week') return `${d.getUTCDate()} ${month}`
  return formatShortDate(period)
}

/** ISO weekday (1 = Monday) to the index of common:date.weekdays* (0 = Sunday). */
function weekdayName(iso: number, list: 'weekdays' | 'weekdaysShort') {
  return i18n.t(`common:date.${list}.${iso % 7}`)
}

// --- change against the compared period ---------------------------------------

type Change = { text: string; direction: 'up' | 'down' | 'flat' }

/** Growth of a count or an average, in %; null when the compared period had none. */
function growth(now: number | null, before: number | null | undefined): Change | null {
  if (now === null || before === null || before === undefined || before === 0) return null
  const value = ((now - before) / before) * 100
  const rounded = Math.round(value)
  if (rounded === 0) return { text: i18n.t('admin:indicators.percent', { value: 0 }), direction: 'flat' }
  return { text: i18n.t('admin:indicators.percent', { value: `${rounded > 0 ? '+' : '−'}${Math.abs(rounded)}` }), direction: rounded > 0 ? 'up' : 'down' }
}

/** Change of a rate, in percentage points: 30 % → 36 % is "+6 pp", not "+20 %". */
function points(now: number | null, before: number | null | undefined): Change | null {
  if (now === null || before === null || before === undefined) return null
  const value = Math.round((now - before) * 1000) / 10
  if (value === 0) return { text: i18n.t('admin:indicators.points', { value: 0 }), direction: 'flat' }
  return { text: i18n.t('admin:indicators.points', { value: `${value > 0 ? '+' : '−'}${decimal(Math.abs(value))}` }), direction: value > 0 ? 'up' : 'down' }
}

/** The change as a pill: green when it is good news, red when not, grey when neither. */
function Delta({ change, good, large }: { change: Change | null | undefined; good?: 'up' | 'down'; large?: boolean }) {
  const { t } = useTranslation('admin')
  if (change === undefined) return null // nothing compared
  if (change === null) return <span className={cn('text-xs text-muted-foreground', large && 'text-base')}>{t('indicators.change.noBase')}</span>
  const tone = change.direction === 'flat' || !good ? 'neutral' : change.direction === good ? 'positive' : 'negative'
  const Icon = change.direction === 'up' ? ArrowUpRight : change.direction === 'down' ? ArrowDownRight : ArrowRight
  return (
    <span
      className={cn(
        'num inline-flex w-fit items-center gap-0.5 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap',
        tone === 'positive' && 'bg-pill-positive-bg text-pill-positive-ink',
        tone === 'negative' && 'bg-pill-negative-bg text-pill-negative-ink',
        tone === 'neutral' && 'bg-pill-neutral-bg text-pill-neutral-ink',
        large && 'px-3 py-1 text-lg',
      )}
    >
      <Icon aria-hidden className={cn('size-3.5', large && 'size-5')} />
      {change.text}
      <span className="sr-only"> {t(`indicators.change.${tone}`)}</span>
    </span>
  )
}

// --- refresh and full screen --------------------------------------------------

/** Reloads every `seconds` while the tab is visible, and on coming back to it. */
function useAutoRefresh(reload: () => void, seconds: number) {
  useEffect(() => {
    if (seconds <= 0) return
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') reload()
    }, seconds * 1000)
    const onVisible = () => document.visibilityState === 'visible' && reload()
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [reload, seconds])
}

/** Puts the element given to `setTarget` (as its ref) in full screen (and back); `full` while it is. */
function useFullscreen<T extends HTMLElement>() {
  const [element, setElement] = useState<T | null>(null)
  const [full, setFull] = useState(false)
  useEffect(() => {
    const onChange = () => setFull(element !== null && document.fullscreenElement === element)
    document.addEventListener('fullscreenchange', onChange)
    return () => document.removeEventListener('fullscreenchange', onChange)
  }, [element])
  const toggle = useCallback(() => {
    if (!element) return
    if (document.fullscreenElement === element) void document.exitFullscreen()
    else void element.requestFullscreen().catch(() => undefined)
  }, [element])
  // phones without the Fullscreen API (iPhone Safari) get no button
  return { setTarget: setElement, full, toggle, supported: document.fullscreenEnabled }
}

function FullscreenButton({ full, board, onClick }: { full: boolean; board: string; onClick: () => void }) {
  const { t } = useTranslation('admin')
  const label = full ? t('indicators.exitFullscreen') : t('indicators.fullscreen', { board })
  return (
    <Button variant="outline" size="icon" onClick={onClick} aria-label={label} title={label} className="shrink-0">
      {full ? <Minimize2 /> : <Maximize2 />}
    </Button>
  )
}

/** A card of the dashboard that can open on its own in full screen (for a screen on the wall). */
function Board({ id, title, hint, caption, children }: { id: string; title: string; hint?: string; caption: string; children: (full: boolean) => ReactNode }) {
  const { setTarget, full, toggle, supported } = useFullscreen<HTMLElement>()
  return (
    <section ref={setTarget} aria-labelledby={id} className={cn('min-w-0 rounded-lg border bg-card p-5', full && 'overflow-auto rounded-none border-0 p-8')}>
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 id={id} className={cn('font-semibold', full && 'text-2xl')}>
            {title}
          </h3>
          {hint && <p className="text-sm text-muted-foreground">{hint}</p>}
          {full && <p className="num mt-1 text-sm text-muted-foreground">{caption}</p>}
        </div>
        {supported && <FullscreenButton full={full} board={title} onClick={toggle} />}
      </div>
      {children(full)}
    </section>
  )
}

// --- the dashboard --------------------------------------------------------------

/** The council's indicators ("indicadores básicos"), computed live by the API; with a facility chosen, its detail. */
export function IndicatorsPanel({ slug, facilities }: { slug: string; facilities: AdminFacilitySummary[] }) {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const [period, setPeriod] = useState<string>('30')
  const [from, setFrom] = useState(() => addDays(today(), -29))
  const [to, setTo] = useState(today)
  const [compare, setCompare] = useState<IndicatorsCompare>('previous')
  const [facility, setFacility] = useState<string>(ALL)
  const [refresh, setRefresh] = useState<string>(storedRefresh)
  const [exportError, setExportError] = useState('')
  const [excel, setExcel] = useState(storedExcel)
  const custom = period === 'custom'
  const rangeValid = !custom || (from !== '' && to !== '' && from <= to)
  const query: IndicatorsQuery = {
    ...(custom ? { from, to } : { days: Number(period) }),
    facility: facility === ALL ? undefined : facility,
    compare,
  }
  const indicators = useAsync(
    rangeValid ? () => api.council.indicators(token, slug, query) : null,
    `indicators:${slug}:${period}:${from}:${to}:${compare}:${facility}`,
  )
  useAutoRefresh(indicators.reload, Number(refresh))
  const { setTarget: setDashboard, full: dashboardFull, toggle: toggleDashboard, supported: canFullscreen } = useFullscreen<HTMLElement>()
  const chosen = facilities.find((f) => f.id === facility)

  const changeRefresh = (value: string) => {
    setRefresh(value)
    try {
      localStorage.setItem(REFRESH_KEY, value)
    } catch {
      // not remembered, still applied
    }
  }

  const changeExcel = (value: boolean) => {
    setExcel(value)
    try {
      localStorage.setItem(EXCEL_KEY, value ? '1' : '0')
    } catch {
      // not remembered, still applied
    }
  }

  const download = async (options: IndicatorsExport) => {
    setExportError('')
    try {
      await api.council.exportIndicators(token, slug, query, options)
    } catch (error) {
      setExportError((error as Error).message)
    }
  }

  const openFacility = (id: string) => {
    setFacility(id)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const data = indicators.data
  // what a board in full screen shows under its title, away from the filters
  const caption = [
    chosen?.name ?? t('admin:indicators.allFacilities'),
    custom ? `${dayMonthYear(from)} – ${dayMonthYear(to)}` : t(`admin:indicators.periods.${period}`),
    data?.previous ? t('admin:indicators.comparedWith', { from: dayMonthYear(data.previous.from), to: dayMonthYear(data.previous.to) }) : '',
    data ? t('admin:indicators.updatedAt', { time: formatTime(data.generated_at, data.timezone) }) : '',
  ]
    .filter(Boolean)
    .join(' · ')
  return (
    <section
      ref={setDashboard}
      aria-labelledby="indicators-title"
      className={cn('grid min-w-0 grid-cols-1 gap-4', dashboardFull && 'content-start overflow-auto bg-background p-6')}
    >
      <div className="rounded-lg border bg-card">
        {dashboardFull ? (
          // the filter and export menus open outside the full-screen element, so they stay on the page
          <div className="flex flex-wrap items-center justify-between gap-3 p-4">
            <div>
              <p className="text-2xl font-semibold" aria-hidden="true">
                {chosen?.name ?? t('admin:indicators.title')}
              </p>
              <p className="num text-sm text-muted-foreground">{caption}</p>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon"
                onClick={indicators.reload}
                disabled={indicators.loading || !rangeValid}
                aria-label={t('admin:indicators.refresh')}
              >
                <RefreshCw className={indicators.loading ? 'animate-spin motion-reduce:animate-none' : undefined} />
              </Button>
              {canFullscreen && <FullscreenButton full={dashboardFull} board={chosen?.name ?? t('admin:indicators.title')} onClick={toggleDashboard} />}
            </div>
          </div>
        ) : (
          <>
            {/* what the figures cover: one row that fills the card, the dates joining it for a range of your own */}
            <div
              className={cn(
                'grid gap-3 p-4 sm:grid-cols-2',
                custom
                  ? 'lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,1.2fr)]'
                  : 'lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,1.2fr)]',
              )}
            >
              <div className="grid min-w-0 gap-1.5">
                <Label htmlFor="indicators-facility">{t('admin:indicators.facility')}</Label>
                <Select value={facility} onValueChange={setFacility}>
                  <SelectTrigger id="indicators-facility" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ALL}>{t('admin:indicators.allFacilities')}</SelectItem>
                    {facilities.map((f) => (
                      <SelectItem key={f.id} value={f.id}>
                        {f.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid min-w-0 gap-1.5">
                <Label htmlFor="indicators-period">{t('admin:indicators.period')}</Label>
                <Select value={period} onValueChange={setPeriod}>
                  <SelectTrigger id="indicators-period" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PERIODS.map((p) => (
                      <SelectItem key={p} value={p}>
                        {t(`admin:indicators.periods.${p}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {custom && (
                <>
                  <div className="grid min-w-0 gap-1.5">
                    <Label htmlFor="indicators-from">{t('admin:indicators.from')}</Label>
                    <DatePicker id="indicators-from" value={from} onChange={setFrom} max={to || today()} className="w-full" />
                  </div>
                  <div className="grid min-w-0 gap-1.5">
                    <Label htmlFor="indicators-to">{t('admin:indicators.to')}</Label>
                    <DatePicker id="indicators-to" value={to} onChange={setTo} min={from} max={today()} className="w-full" />
                  </div>
                </>
              )}
              <div className="grid min-w-0 gap-1.5 sm:col-span-2 lg:col-span-1">
                <Label htmlFor="indicators-compare">{t('admin:indicators.compare')}</Label>
                <Select value={compare} onValueChange={(value) => setCompare(value as IndicatorsCompare)}>
                  <SelectTrigger id="indicators-compare" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {COMPARE.map((c) => (
                      <SelectItem key={c} value={c}>
                        {t(`admin:indicators.compareOptions.${c}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            {/* how the figures stay fresh, and what to do with them: a quieter bar under the filters */}
            <div className="flex flex-wrap items-center justify-between gap-3 border-t px-4 py-3">
              <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
                <Label htmlFor="indicators-refresh" className="font-normal text-muted-foreground">
                  {t('admin:indicators.autoRefresh')}
                </Label>
                <Select value={refresh} onValueChange={changeRefresh}>
                  <SelectTrigger id="indicators-refresh" size="sm" className="min-w-36">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {REFRESH.map((r) => (
                      <SelectItem key={r} value={r}>
                        {t(`admin:indicators.refreshOptions.${r}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {data && (
                  <span className="num text-sm text-muted-foreground" aria-live="polite">
                    {t('admin:indicators.updatedAt', { time: formatTime(data.generated_at, data.timezone) })}
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="icon"
                  onClick={indicators.reload}
                  disabled={indicators.loading || !rangeValid}
                  aria-label={t('admin:indicators.refresh')}
                >
                  <RefreshCw className={indicators.loading ? 'animate-spin motion-reduce:animate-none' : undefined} />
                </Button>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" disabled={!rangeValid}>
                      <Download />
                      {t('admin:indicators.export')}
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-72">
                    <DropdownMenuCheckboxItem checked={excel} onCheckedChange={(v) => changeExcel(v === true)} onSelect={(e) => e.preventDefault()}>
                      {t('admin:indicators.exportExcel')}
                    </DropdownMenuCheckboxItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuLabel>{t('admin:indicators.exportAnonymous')}</DropdownMenuLabel>
                    {EXPORTS.map((kind) => (
                      <DropdownMenuItem key={kind} onSelect={() => download({ kind, excel })} className="grid gap-0.5">
                        <span>{t(`admin:indicators.exports.${kind}.label`)}</span>
                        <span className="text-xs text-muted-foreground">{t(`admin:indicators.exports.${kind}.hint`)}</span>
                      </DropdownMenuItem>
                    ))}
                    <DropdownMenuSeparator />
                    <p className="px-2 py-1.5 text-xs text-muted-foreground">{t('admin:indicators.exportNote')}</p>
                  </DropdownMenuContent>
                </DropdownMenu>
                {canFullscreen && <FullscreenButton full={dashboardFull} board={chosen?.name ?? t('admin:indicators.title')} onClick={toggleDashboard} />}
              </div>
            </div>
          </>
        )}
      </div>
      <h2 id="indicators-title" className="sr-only">
        {chosen ? t('admin:indicators.facilityTitle', { name: chosen.name }) : t('admin:indicators.title')}
      </h2>
      {!rangeValid && (
        <p role="alert" className="text-destructive">
          {t('admin:indicators.customInvalid')}
        </p>
      )}
      {exportError && (
        <p role="alert" className="text-destructive">
          {exportError}
        </p>
      )}
      {indicators.error && (
        <p role="alert" className="text-destructive">
          {indicators.error.message}
        </p>
      )}
      {chosen && !dashboardFull && <FacilityHeader slug={slug} facility={chosen} onBack={() => setFacility(ALL)} />}
      {!data ? (
        rangeValid && !indicators.error && <Skeleton className="h-96 w-full" />
      ) : (
        <>
          <Kpis data={data} caption={caption} />
          <BookingsChart data={data} caption={caption} />
          <WeekdaysAndHabits data={data} caption={caption} />
          <BusyHours data={data} caption={caption} />
          {!chosen && <FacilityTable data={data} caption={caption} onOpen={openFacility} />}
          <p className="text-sm text-muted-foreground">{t('admin:indicators.definitions')}</p>
        </>
      )}
    </section>
  )
}

/** The facility being looked at, with the way back to the whole council and to its editor. */
function FacilityHeader({ slug, facility, onBack }: { slug: string; facility: AdminFacilitySummary; onBack: () => void }) {
  const { t } = useTranslation('admin')
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-card p-4">
      <div className="flex items-center gap-3">
        {facility.image_url ? (
          <img src={facility.image_url} alt="" className="h-12 w-16 shrink-0 rounded-sm object-cover" />
        ) : (
          <span className="h-12 w-16 shrink-0 rounded-sm bg-muted" aria-hidden="true" />
        )}
        <div>
          <p className="text-lg font-semibold">{facility.name}</p>
          <p className="text-sm text-muted-foreground">{facility.sport || t('indicators.facilityDetail')}</p>
        </div>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" onClick={onBack}>
          <ArrowLeft />
          {t('indicators.allFacilities')}
        </Button>
        <Button asChild variant="outline">
          <Link to={`/${slug}/admin/facilities/${facility.id}`}>
            <Pencil />
            {t('indicators.editFacility')}
          </Link>
        </Button>
      </div>
    </div>
  )
}

function Kpis({ data, caption }: { data: Indicators; caption: string }) {
  const { t } = useTranslation('admin')
  const totals = data.totals
  const before = data.previous?.totals
  // undefined: nothing compared, so no pill at all
  const compared = <T,>(change: () => T) => (before ? change() : undefined)
  const tiles = [
    {
      label: t('indicators.kpi.bookings'),
      value: totals.bookings.toLocaleString(i18n.resolvedLanguage),
      hint: t('indicators.kpi.bookingsHint', { count: totals.confirmed }),
      change: compared(() => growth(totals.bookings, before?.bookings)),
      good: 'up' as const,
    },
    {
      label: t('indicators.kpi.occupancy'),
      value: percent(totals.occupancy),
      hint: t('indicators.kpi.occupancyHint', { booked: hours(totals.booked_hours), open: hours(totals.open_hours) }),
      change: compared(() => points(totals.occupancy, before?.occupancy)),
      good: 'up' as const,
    },
    {
      label: t('indicators.kpi.cancellations'),
      value: percent(totals.cancellation_rate),
      hint: t('indicators.kpi.cancellationsHint', { user: totals.cancelled_by_user, council: totals.cancelled_by_council }),
      change: compared(() => points(totals.cancellation_rate, before?.cancellation_rate)),
      good: 'down' as const,
    },
    {
      label: t('indicators.kpi.activeUsers'),
      value: totals.active_users.toLocaleString(i18n.resolvedLanguage),
      hint: t('indicators.kpi.activeUsersHint'),
      change: compared(() => growth(totals.active_users, before?.active_users)),
      good: 'up' as const,
    },
    {
      label: t('indicators.kpi.noShows'),
      value: percent(totals.no_show_rate),
      hint:
        totals.finished > 0
          ? t('indicators.kpi.noShowsHint', { missed: totals.finished - totals.attended, finished: totals.finished })
          : t('indicators.kpi.noShowsNone'),
      change: compared(() => points(totals.no_show_rate, before?.no_show_rate)),
      good: 'down' as const,
    },
  ]
  const hint = data.previous ? t('indicators.comparedWith', { from: dayMonthYear(data.previous.from), to: dayMonthYear(data.previous.to) }) : undefined
  return (
    <Board id="kpis-title" title={t('indicators.kpiTitle')} hint={hint} caption={caption}>
      {(full) => (
        <dl
          className={cn(
            'mt-4 grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-5',
            full && 'mt-0 min-h-[calc(100dvh-10rem)] content-center gap-6 lg:grid-cols-3 xl:grid-cols-5',
          )}
        >
          {tiles.map((tile) => (
            <div key={tile.label} className={cn('grid content-start gap-1 rounded-lg border p-4', full && 'gap-3 p-8')}>
              <dt className={cn('text-sm text-muted-foreground', full && 'text-xl')}>{tile.label}</dt>
              <dd className={cn('num text-[28px] leading-9 font-semibold tracking-[-0.01em]', full && 'text-7xl leading-none')}>{tile.value}</dd>
              {tile.change !== undefined && (
                <dd>
                  <Delta change={tile.change} good={tile.good} large={full} />
                </dd>
              )}
              <dd className={cn('text-xs text-muted-foreground', full && 'text-base')}>{tile.hint}</dd>
            </div>
          ))}
        </dl>
      )}
    </Board>
  )
}

function BookingsChart({ data, caption }: { data: Indicators; caption: string }) {
  const { t } = useTranslation('admin')
  const config = {
    confirmed: { label: t('indicators.confirmed'), color: 'var(--chart-confirmed)' },
    cancelled: { label: t('indicators.cancelled'), color: 'var(--chart-cancelled)' },
    previous: { label: t('indicators.previousConfirmed'), color: 'var(--chart-previous)' },
  } satisfies ChartConfig
  // the compared period, lined up by position: its first day under this period's first day
  const rows = data.series.map((p, i) => ({ ...p, label: periodLabel(p.period, data.bucket), previous: data.previous?.series[i]?.confirmed }))
  return (
    <Board id="bookings-chart-title" title={t(`indicators.chartTitle.${data.bucket}`)} caption={caption}>
      {(full) => (
        <>
          <ChartContainer config={config} className={cn('mt-4 aspect-auto h-64 w-full', full && 'mt-8 h-[calc(100dvh-12rem)] text-sm')}>
            <ComposedChart data={rows} accessibilityLayer>
              <CartesianGrid vertical={false} />
              <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={8} minTickGap={16} />
              <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={32} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <ChartLegend content={<ChartLegendContent />} />
              <Bar dataKey="confirmed" stackId="bookings" fill="var(--color-confirmed)" />
              <Bar dataKey="cancelled" stackId="bookings" fill="var(--color-cancelled)" radius={[4, 4, 0, 0]} />
              {data.previous && <Line dataKey="previous" type="monotone" stroke="var(--color-previous)" strokeWidth={2} strokeDasharray="5 4" dot={false} />}
            </ComposedChart>
          </ChartContainer>
          {!full && (
            <details className="mt-3">
              <summary className="cursor-pointer text-sm text-muted-foreground">{t('indicators.showTable')}</summary>
              <div className="mt-2 max-h-72 overflow-auto">
                <table className="num w-full text-sm">
                  <thead>
                    <tr className="text-xs text-muted-foreground">
                      <th scope="col" className="py-1.5 text-left font-normal">
                        {t('indicators.colPeriod')}
                      </th>
                      <th scope="col" className="py-1.5 text-right font-normal">
                        {t('indicators.confirmed')}
                      </th>
                      <th scope="col" className="py-1.5 text-right font-normal">
                        {t('indicators.cancelled')}
                      </th>
                      {data.previous && (
                        <th scope="col" className="py-1.5 text-right font-normal">
                          {t('indicators.previousConfirmed')}
                        </th>
                      )}
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((p) => (
                      <tr key={p.period} className="border-t">
                        <th scope="row" className="py-1.5 text-left font-normal">
                          {data.bucket === 'week' ? t('indicators.weekOf', { date: p.label }) : p.label}
                        </th>
                        <td className="py-1.5 text-right">{p.confirmed}</td>
                        <td className="py-1.5 text-right">{p.cancelled}</td>
                        {data.previous && <td className="py-1.5 text-right">{p.previous ?? '—'}</td>}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </details>
          )}
        </>
      )}
    </Board>
  )
}

/** Which weekdays fill up, against the compared period, and how people book. */
function WeekdaysAndHabits({ data, caption }: { data: Indicators; caption: string }) {
  const { t } = useTranslation('admin')
  const config = {
    current: { label: t('indicators.thisPeriod'), color: 'var(--chart-confirmed)' },
    previous: { label: t('indicators.comparedPeriod'), color: 'var(--chart-previous)' },
  } satisfies ChartConfig
  // Monday first, as the heatmap
  const rows = data.weekdays.map((w) => {
    const before = data.previous?.weekdays.find((p) => p.weekday === w.weekday)
    return {
      label: weekdayName(w.weekday, 'weekdaysShort'),
      name: weekdayName(w.weekday, 'weekdays'),
      current: w.occupancy === null ? null : Math.round(w.occupancy * 1000) / 10,
      previous: before?.occupancy == null ? null : Math.round(before.occupancy * 1000) / 10,
    }
  })
  const totals = data.totals
  const before = data.previous?.totals
  const habits = [
    {
      label: t('indicators.habits.party'),
      value: totals.avg_party_size === null ? '—' : decimal(totals.avg_party_size),
      hint: t('indicators.habits.partyHint'),
      change: before ? growth(totals.avg_party_size, before.avg_party_size) : undefined,
    },
    {
      label: t('indicators.habits.lead'),
      value: leadTime(totals.median_lead_hours),
      hint: t('indicators.habits.leadHint'),
      change: before ? growth(totals.median_lead_hours, before.median_lead_hours) : undefined,
    },
  ]
  return (
    <Board id="weekdays-title" title={t('indicators.weekdaysTitle')} hint={t('indicators.weekdaysHint')} caption={caption}>
      {(full) => (
        <div className={cn('mt-4 grid gap-4 lg:grid-cols-[1fr_16rem]', full && 'mt-8 lg:grid-cols-[1fr_22rem]')}>
          <div className="min-w-0">
            <ChartContainer config={config} className={cn('aspect-auto h-56 w-full', full && 'h-[calc(100dvh-14rem)] text-sm')}>
              <BarChart data={rows} accessibilityLayer>
                <CartesianGrid vertical={false} />
                <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={8} />
                <YAxis domain={[0, 100]} tickFormatter={(v: number) => percent(v / 100)} tickLine={false} axisLine={false} width={44} />
                <ChartTooltip
                  content={
                    <ChartTooltipContent
                      labelFormatter={(_, payload) => payload?.[0]?.payload?.name ?? ''}
                      formatter={(value, name, item) => (
                        <>
                          <span className="size-2.5 shrink-0 rounded-[2px]" style={{ backgroundColor: item.color }} />
                          <span className="text-muted-foreground">{config[name as keyof typeof config]?.label}</span>
                          <span className="num ml-auto font-medium">{percent(Number(value) / 100)}</span>
                        </>
                      )}
                    />
                  }
                />
                <ChartLegend content={<ChartLegendContent />} />
                <Bar dataKey="current" fill="var(--color-current)" radius={[4, 4, 0, 0]} />
                {data.previous && <Bar dataKey="previous" fill="var(--color-previous)" radius={[4, 4, 0, 0]} />}
              </BarChart>
            </ChartContainer>
            {!full && (
              <details className="mt-3">
                <summary className="cursor-pointer text-sm text-muted-foreground">{t('indicators.showTable')}</summary>
                <table className="num mt-2 w-full text-sm">
                  <thead>
                    <tr className="text-xs text-muted-foreground">
                      <th scope="col" className="py-1.5 text-left font-normal">
                        {t('indicators.colWeekday')}
                      </th>
                      <th scope="col" className="py-1.5 text-right font-normal">
                        {t('indicators.thisPeriod')}
                      </th>
                      {data.previous && (
                        <th scope="col" className="py-1.5 text-right font-normal">
                          {t('indicators.comparedPeriod')}
                        </th>
                      )}
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((r) => (
                      <tr key={r.name} className="border-t">
                        <th scope="row" className="py-1.5 text-left font-normal">
                          {r.name}
                        </th>
                        <td className="py-1.5 text-right">{r.current === null ? '—' : percent(r.current / 100)}</td>
                        {data.previous && <td className="py-1.5 text-right">{r.previous === null ? '—' : percent(r.previous / 100)}</td>}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </details>
            )}
          </div>
          <dl className="grid content-start gap-3 sm:grid-cols-2 lg:grid-cols-1">
            {habits.map((h) => (
              <div key={h.label} className={cn('grid content-start gap-1 rounded-lg border p-4', full && 'gap-2 p-6')}>
                <dt className={cn('text-sm text-muted-foreground', full && 'text-lg')}>{h.label}</dt>
                <dd className={cn('num text-2xl font-semibold', full && 'text-5xl')}>{h.value}</dd>
                {h.change !== undefined && (
                  <dd>
                    <Delta change={h.change} large={full} />
                  </dd>
                )}
                <dd className={cn('text-xs text-muted-foreground', full && 'text-sm')}>{h.hint}</dd>
              </div>
            ))}
          </dl>
        </div>
      )}
    </Board>
  )
}

/** Weekday × starting hour, as a real table: each cell says its count to screen readers. */
function BusyHours({ data, caption }: { data: Indicators; caption: string }) {
  const { t } = useTranslation('admin')
  const counts = new Map(data.busy_hours.map((c) => [`${c.weekday}:${c.hour}`, c.bookings]))
  const max = Math.max(0, ...data.busy_hours.map((c) => c.bookings))
  const hourList = data.busy_hours.map((c) => c.hour)
  const first = Math.min(...hourList)
  const last = Math.max(...hourList)
  const hourRange = max === 0 ? [] : Array.from({ length: last - first + 1 }, (_, i) => first + i)
  const clock = (h: number) => `${String(h).padStart(2, '0')}:00`
  return (
    <Board id="busy-title" title={t('indicators.busyTitle')} hint={t('indicators.busyHint')} caption={caption}>
      {(full) =>
        max === 0 ? (
          <p className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('indicators.empty')}</p>
        ) : (
          <div className={cn('relative mt-4 overflow-x-auto', full && 'mt-8')}>
            <table className={cn('num w-full border-separate border-spacing-1 text-xs', full && 'border-spacing-1.5 text-base')}>
              <thead>
                <tr>
                  <td />
                  {hourRange.map((h) => (
                    <th key={h} scope="col" className="font-normal text-muted-foreground">
                      {String(h).padStart(2, '0')}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {[1, 2, 3, 4, 5, 6, 7].map((weekday) => (
                  <tr key={weekday}>
                    <th scope="row" className="pr-2 text-left font-normal text-muted-foreground">
                      <abbr title={weekdayName(weekday, 'weekdays')} className="no-underline">
                        {weekdayName(weekday, 'weekdaysShort')}
                      </abbr>
                    </th>
                    {hourRange.map((h) => {
                      const count = counts.get(`${weekday}:${h}`) ?? 0
                      const label = t('indicators.busyCell', { weekday: weekdayName(weekday, 'weekdays'), hour: clock(h), count })
                      return (
                        <td
                          key={h}
                          title={label}
                          className={cn('h-7 min-w-7 rounded-sm bg-muted', full && 'h-[calc((100dvh-18rem)/7)] min-h-10')}
                          style={
                            count > 0
                              ? { backgroundColor: `color-mix(in oklab, var(--chart-confirmed) ${Math.round(15 + (85 * count) / max)}%, transparent)` }
                              : undefined
                          }
                        >
                          <span className="sr-only">{label}</span>
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      }
    </Board>
  )
}

function FacilityTable({ data, caption, onOpen }: { data: Indicators; caption: string; onOpen: (id: string) => void }) {
  const { t } = useTranslation('admin')
  const before = new Map(data.previous?.facilities.map((f) => [f.id, f]))
  return (
    <Board id="by-facility-title" title={t('indicators.byFacilityTitle')} hint={t('indicators.byFacilityHint')} caption={caption}>
      {(full) => (
        <div className={cn('mt-3 overflow-x-auto', full && 'mt-8')}>
          <table className={cn('w-full min-w-[720px]', full && 'text-xl [&_td]:py-5 [&_th[scope=row]]:py-5')}>
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th scope="col" className="py-2 text-left font-normal">
                  {t('indicators.colFacility')}
                </th>
                <th scope="col" className="py-2 pl-4 text-right font-normal">
                  {t('indicators.colBookings')}
                </th>
                {data.previous && (
                  <th scope="col" className="py-2 pl-4 text-left font-normal">
                    {t('indicators.colGrowth')}
                  </th>
                )}
                <th scope="col" className="py-2 pl-4 text-left font-normal">
                  {t('indicators.colOccupancy')}
                </th>
                <th scope="col" className="py-2 pl-4 text-right font-normal">
                  {t('indicators.colCancellation')}
                </th>
                <th scope="col" className="py-2 pl-4 text-right font-normal">
                  {t('indicators.colNoShow')}
                </th>
              </tr>
            </thead>
            <tbody className="num">
              {data.facilities.map((f) => (
                <tr key={f.id} className="border-t">
                  <th scope="row" className="py-3 text-left font-normal">
                    <button
                      type="button"
                      onClick={() => onOpen(f.id)}
                      className="rounded-sm text-left font-medium text-primary underline-offset-4 hover:underline focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none"
                      aria-label={t('indicators.openFacility', { name: f.name })}
                    >
                      {f.name}
                    </button>
                    {!f.is_active && <span className="ml-2 text-xs text-muted-foreground">({t('admin:councils.inactive')})</span>}
                  </th>
                  <td className="py-3 pl-4 text-right">{f.bookings}</td>
                  {data.previous && (
                    <td className="py-3 pl-4">
                      <Delta change={growth(f.bookings, before.get(f.id)?.bookings)} good="up" large={full} />
                    </td>
                  )}
                  <td className="py-3 pl-4">
                    <span className="flex items-center gap-2">
                      <span className={cn('h-2 w-24 overflow-hidden rounded-full bg-muted', full && 'h-3 w-64')} aria-hidden="true">
                        <span className="block h-full bg-chart-confirmed" style={{ width: `${Math.round((f.occupancy ?? 0) * 100)}%` }} />
                      </span>
                      {percent(f.occupancy)}
                    </span>
                  </td>
                  <td className="py-3 pl-4 text-right">{percent(f.cancellation_rate)}</td>
                  <td className="py-3 pl-4 text-right">
                    {f.has_reader ? percent(f.no_show_rate) : <span className="text-muted-foreground">{t('indicators.noReader')}</span>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Board>
  )
}
