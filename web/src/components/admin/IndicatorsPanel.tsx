import { Download, Maximize2, Minimize2, RefreshCw } from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from 'recharts'
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
import { type AdminFacilitySummary, api, type Indicators, type IndicatorsExport } from '@/lib/api'
import { formatShortDate, formatTime } from '@/lib/time'
import { cn } from '@/lib/utils'
import i18n from '@/i18n'

const PERIODS = ['7', '30', '90', '365'] as const
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

function percent(value: number | null) {
  return value === null ? '—' : i18n.t('admin:indicators.percent', { value: Math.round(value * 100) })
}

function hours(value: number) {
  return Math.round(value).toLocaleString(i18n.resolvedLanguage)
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

/** The council's indicators ("indicadores básicos"), computed live by the API. */
export function IndicatorsPanel({ slug, facilities }: { slug: string; facilities: AdminFacilitySummary[] }) {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const [days, setDays] = useState<string>('30')
  const [facility, setFacility] = useState<string>(ALL)
  const [refresh, setRefresh] = useState<string>(storedRefresh)
  const [exportError, setExportError] = useState('')
  const [excel, setExcel] = useState(storedExcel)
  const query = { days: Number(days), facility: facility === ALL ? undefined : facility }
  const indicators = useAsync(() => api.council.indicators(token, slug, query), `indicators:${slug}:${days}:${facility}`)
  useAutoRefresh(indicators.reload, Number(refresh))
  const { setTarget: setDashboard, full: dashboardFull, toggle: toggleDashboard, supported: canFullscreen } = useFullscreen<HTMLElement>()

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

  const data = indicators.data
  // what a board in full screen shows under its title, away from the filters
  const caption = [
    facility === ALL ? t('admin:indicators.allFacilities') : (facilities.find((f) => f.id === facility)?.name ?? ''),
    t(`admin:indicators.periods.${days}`),
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
      <div className="flex flex-wrap items-end justify-between gap-3 rounded-lg border bg-card p-4">
        {dashboardFull ? (
          // the filter and export menus open outside the full-screen element, so they stay on the page
          <div>
            <p className="text-2xl font-semibold" aria-hidden="true">
              {t('admin:indicators.title')}
            </p>
            <p className="num text-sm text-muted-foreground">{caption}</p>
          </div>
        ) : (
          <div className="flex flex-wrap items-end gap-3">
            <div className="grid gap-1.5">
              <Label htmlFor="indicators-facility">{t('admin:indicators.facility')}</Label>
              <Select value={facility} onValueChange={setFacility}>
                <SelectTrigger id="indicators-facility" className="w-56">
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
            <div className="grid gap-1.5">
              <Label htmlFor="indicators-period">{t('admin:indicators.period')}</Label>
              <Select value={days} onValueChange={setDays}>
                <SelectTrigger id="indicators-period" className="w-44">
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
            <div className="grid gap-1.5">
              <Label htmlFor="indicators-refresh">{t('admin:indicators.autoRefresh')}</Label>
              <Select value={refresh} onValueChange={changeRefresh}>
                <SelectTrigger id="indicators-refresh" className="w-44">
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
            </div>
          </div>
        )}
        <div className="flex flex-wrap items-center gap-2">
          {data && !dashboardFull && (
            <span className="num text-sm text-muted-foreground" aria-live="polite">
              {t('admin:indicators.updatedAt', { time: formatTime(data.generated_at, data.timezone) })}
            </span>
          )}
          <Button variant="outline" size="icon" onClick={indicators.reload} disabled={indicators.loading} aria-label={t('admin:indicators.refresh')}>
            <RefreshCw className={indicators.loading ? 'animate-spin motion-reduce:animate-none' : undefined} />
          </Button>
          {!dashboardFull && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline">
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
          )}
          {canFullscreen && <FullscreenButton full={dashboardFull} board={t('admin:indicators.title')} onClick={toggleDashboard} />}
        </div>
      </div>
      <h2 id="indicators-title" className="sr-only">
        {t('admin:indicators.title')}
      </h2>
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
      {!data ? (
        !indicators.error && <Skeleton className="h-96 w-full" />
      ) : (
        <>
          <Kpis data={data} caption={caption} />
          <BookingsChart data={data} caption={caption} />
          <BusyHours data={data} caption={caption} />
          <FacilityTable data={data} caption={caption} />
          <p className="text-sm text-muted-foreground">{t('admin:indicators.definitions')}</p>
        </>
      )}
    </section>
  )
}

function Kpis({ data, caption }: { data: Indicators; caption: string }) {
  const { t } = useTranslation('admin')
  const totals = data.totals
  const tiles = [
    {
      label: t('indicators.kpi.bookings'),
      value: totals.bookings.toLocaleString(i18n.resolvedLanguage),
      hint: t('indicators.kpi.bookingsHint', { count: totals.confirmed }),
    },
    {
      label: t('indicators.kpi.occupancy'),
      value: percent(totals.occupancy),
      hint: t('indicators.kpi.occupancyHint', { booked: hours(totals.booked_hours), open: hours(totals.open_hours) }),
    },
    {
      label: t('indicators.kpi.cancellations'),
      value: percent(totals.cancellation_rate),
      hint: t('indicators.kpi.cancellationsHint', { user: totals.cancelled_by_user, council: totals.cancelled_by_council }),
    },
    { label: t('indicators.kpi.activeUsers'), value: totals.active_users.toLocaleString(i18n.resolvedLanguage), hint: t('indicators.kpi.activeUsersHint') },
    {
      label: t('indicators.kpi.noShows'),
      value: percent(totals.no_show_rate),
      hint:
        totals.finished > 0
          ? t('indicators.kpi.noShowsHint', { missed: totals.finished - totals.attended, finished: totals.finished })
          : t('indicators.kpi.noShowsNone'),
    },
  ]
  return (
    <Board id="kpis-title" title={t('indicators.kpiTitle')} caption={caption}>
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
  } satisfies ChartConfig
  const rows = data.series.map((p) => ({ ...p, label: periodLabel(p.period, data.bucket) }))
  return (
    <Board id="bookings-chart-title" title={t(`indicators.chartTitle.${data.bucket}`)} caption={caption}>
      {(full) => (
        <>
          <ChartContainer config={config} className={cn('mt-4 aspect-auto h-64 w-full', full && 'mt-8 h-[calc(100dvh-12rem)] text-sm')}>
            <BarChart data={rows} accessibilityLayer>
              <CartesianGrid vertical={false} />
              <XAxis dataKey="label" tickLine={false} axisLine={false} tickMargin={8} minTickGap={16} />
              <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={32} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <ChartLegend content={<ChartLegendContent />} />
              <Bar dataKey="confirmed" stackId="bookings" fill="var(--color-confirmed)" />
              <Bar dataKey="cancelled" stackId="bookings" fill="var(--color-cancelled)" radius={[4, 4, 0, 0]} />
            </BarChart>
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

function FacilityTable({ data, caption }: { data: Indicators; caption: string }) {
  const { t } = useTranslation('admin')
  return (
    <Board id="by-facility-title" title={t('indicators.byFacilityTitle')} caption={caption}>
      {(full) => (
        <div className={cn('mt-3 overflow-x-auto', full && 'mt-8')}>
          <table className={cn('w-full min-w-[620px]', full && 'text-xl [&_td]:py-5 [&_th[scope=row]]:py-5')}>
            <thead>
              <tr className="text-xs text-muted-foreground">
                <th scope="col" className="py-2 text-left font-normal">
                  {t('indicators.colFacility')}
                </th>
                <th scope="col" className="py-2 pl-4 text-right font-normal">
                  {t('indicators.colBookings')}
                </th>
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
                    <span className="font-medium">{f.name}</span>
                    {!f.is_active && <span className="ml-2 text-xs text-muted-foreground">({t('admin:councils.inactive')})</span>}
                  </th>
                  <td className="py-3 pl-4 text-right">{f.bookings}</td>
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
