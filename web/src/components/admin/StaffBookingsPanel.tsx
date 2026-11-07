import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import QRCode from 'qrcode'
import { CircleCheck, ChevronLeft, ChevronRight, PartyPopper, Plus, Printer, Undo2, UserCheck, UserPlus } from 'lucide-react'
import { DatePicker } from '@/components/DatePicker'
import { QrImage } from '@/components/QrImage'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { FILTER_DEBOUNCE_MS, useAsync } from '@/hooks/useAsync'
import { api, type AdminFacilitySummary, type StaffBooking, type StaffDay, type StaffSlot } from '@/lib/api'
import { addDays, formatLongDate, formatTime, localDate, minutesOfDay, shortDateParts } from '@/lib/time'
import { cn } from '@/lib/utils'

// facilityId: the one chosen with its row's "Reservas" button (kept in the URL); "" = the first
type Props = { slug: string; facilities: AdminFacilitySummary[]; facilityId: string; onFacilityChange: (id: string) => void }

type View = 'week' | 'day'

const pill = 'rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap'
const HOUR_PX = 56 // one hour of the calendar's height
const DEFAULT_AXIS = { start: 8 * 60, end: 22 * 60 } // a day with no turns at all

/** ISO weekday of a "YYYY-MM-DD": 1 = Monday … 7 = Sunday. */
function isoWeekday(date: string) {
  const day = new Date(`${date}T12:00:00Z`).getUTCDay()
  return day === 0 ? 7 : day
}

const startOfWeek = (date: string) => addDays(date, 1 - isoWeekday(date))

// a phone gets one day: seven columns do not fit
const narrowScreen = () => typeof window !== 'undefined' && window.matchMedia('(max-width: 767px)').matches

/** The chosen day first (it says which date "today" is), then, for the week view, the rest of its week. */
async function loadCalendar(token: string, slug: string, facilityId: string, date: string, view: View) {
  const now = new Date().toISOString() // for "today" and the line at the current time
  const first = await api.council.staffDay(token, slug, facilityId, date)
  if (view === 'day') return { date: first.date, now, days: [first] }
  const monday = startOfWeek(first.date)
  const days = await Promise.all(
    Array.from({ length: 7 }, (_, i) => addDays(monday, i)).map((d) => (d === first.date ? first : api.council.staffDay(token, slug, facilityId, d))),
  )
  return { date: first.date, now, days }
}

/*
  The bookings view (each facility's "Reservas" button): one facility's week (or day) as a calendar,
  with who holds each booking. Staff book a free turn for someone who phoned or came to the desk
  (their account, or just a name), and cancel any booking before it starts.
*/
export function StaffBookingsPanel({ slug, facilities, facilityId, onFacilityChange }: Props) {
  const { t } = useTranslation('admin')
  const token = useAuth().user?.access_token ?? ''
  const active = facilities.filter((f) => f.is_active)
  const [date, setDate] = useState('') // "" = today, in the council's time zone
  const [view, setView] = useState<View>(() => (narrowScreen() ? 'day' : 'week'))
  const facility = active.find((f) => f.id === facilityId) ?? active[0]
  const calendar = useAsync(facility ? () => loadCalendar(token, slug, facility.id, date, view) : null, `staff-calendar:${slug}:${facility?.id}:${date}:${view}`, { debounceMs: FILTER_DEBOUNCE_MS })
  const [booking, setBooking] = useState<{ day: StaffDay; slot: StaffSlot } | null>(null)
  const [details, setDetails] = useState<{ day: StaffDay; booking: StaffBooking } | null>(null)
  const [cancelling, setCancelling] = useState<{ day: StaffDay; booking: StaffBooking } | null>(null)

  if (active.length === 0) {
    return <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('staff.noFacilities')}</p>
  }
  const shownDate = calendar.data?.date ?? date
  const step = view === 'week' ? 7 : 1
  const minAge = calendar.data?.days[0]?.min_age
  const residentsOnly = calendar.data?.days[0]?.residents_only === true

  return (
    <section aria-labelledby="staff-title" className="grid gap-4">
      <div>
        <h2 id="staff-title" className="text-lg font-semibold">
          {t('staff.title')}
        </h2>
        <p className="text-muted-foreground">{t('staff.help')}</p>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <div className="grid min-w-56 flex-1 gap-1.5 sm:flex-none">
          <Label htmlFor="staff-facility">{t('staff.facility')}</Label>
          <Select value={facility?.id} onValueChange={onFacilityChange}>
            <SelectTrigger id="staff-facility" className="w-full sm:w-64">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {active.map((f) => (
                <SelectItem key={f.id} value={f.id}>
                  {f.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="staff-date">{t('staff.date')}</Label>
          <div className="flex items-center gap-1">
            <Button variant="outline" size="icon" disabled={!shownDate} onClick={() => setDate(addDays(shownDate, -step))} aria-label={view === 'week' ? t('staff.previousWeek') : t('staff.previousDay')}>
              <ChevronLeft />
            </Button>
            <DatePicker id="staff-date" value={shownDate} onChange={setDate} className="w-44" />
            <Button variant="outline" size="icon" disabled={!shownDate} onClick={() => setDate(addDays(shownDate, step))} aria-label={view === 'week' ? t('staff.nextWeek') : t('staff.nextDay')}>
              <ChevronRight />
            </Button>
          </div>
        </div>
        <Button variant="outline" onClick={() => setDate('')}>
          {t('staff.today')}
        </Button>
        <ToggleGroup
          type="single"
          variant="outline"
          value={view}
          onValueChange={(value) => value && setView(value as View)}
          aria-label={t('staff.viewLabel')}
          className="sm:ml-auto"
        >
          <ToggleGroupItem value="day">{t('staff.viewDay')}</ToggleGroupItem>
          <ToggleGroupItem value="week">{t('staff.viewWeek')}</ToggleGroupItem>
        </ToggleGroup>
      </div>

      {minAge && <p className="text-muted-foreground">{t('staff.minAge', { age: minAge })}</p>}
      {residentsOnly && <p className="text-muted-foreground">{t('staff.residentsOnly')}</p>}

      {calendar.error && !calendar.data ? (
        <Alert variant="destructive">
          <AlertDescription>{calendar.error.message}</AlertDescription>
        </Alert>
      ) : !calendar.data ? (
        <Skeleton className="h-[480px] w-full" />
      ) : (
        <Calendar
          days={calendar.data.days}
          now={calendar.data.now}
          onBook={(day, slot) => setBooking({ day, slot })}
          onOpen={(day, b) => setDetails({ day, booking: b })}
          onPickDay={(d) => {
            setDate(d)
            setView('day')
          }}
        />
      )}
      <StatusLegend />

      {booking && facility && (
        <BookDialog
          slug={slug}
          token={token}
          facilityName={facility.name}
          day={booking.day}
          slot={booking.slot}
          onClose={() => setBooking(null)}
          onBooked={calendar.reload}
        />
      )}
      {details && facility && (
        <DetailsDialog
          slug={slug}
          token={token}
          booking={details.booking}
          zone={details.day.timezone}
          facilityName={facility.name}
          reader={details.day.checkin_ways.includes('reader')}
          onClose={() => setDetails(null)}
          onChanged={calendar.reload}
          onCancel={() => {
            setCancelling(details)
            setDetails(null)
          }}
        />
      )}
      {cancelling && facility && (
        <CancelDialog
          slug={slug}
          token={token}
          facilityName={facility.name}
          timeZone={cancelling.day.timezone}
          booking={cancelling.booking}
          onClose={() => setCancelling(null)}
          onCancelled={() => {
            setCancelling(null)
            calendar.reload()
          }}
        />
      )}
    </section>
  )
}

/** The hours every column shows: from the earliest turn to the latest, on whole hours. */
function axisOf(days: StaffDay[]) {
  const spans = days.flatMap((d) => [...d.slots, ...d.bookings].map((x) => [minutesOfDay(x.starts_at, d.timezone), minutesOfDay(x.ends_at, d.timezone) || 24 * 60]))
  if (spans.length === 0) return DEFAULT_AXIS
  return { start: Math.floor(Math.min(...spans.map(([s]) => s)) / 60) * 60, end: Math.ceil(Math.max(...spans.map(([, e]) => e)) / 60) * 60 }
}

type CalendarProps = {
  days: StaffDay[]
  now: string
  onBook: (day: StaffDay, slot: StaffSlot) => void
  onOpen: (day: StaffDay, booking: StaffBooking) => void
  onPickDay: (date: string) => void
}

/** Columns of days, hours down the side, every booking a block at its time (like a calendar app). */
function Calendar({ days, now, onBook, onOpen, onPickDay }: CalendarProps) {
  const { t } = useTranslation('admin')
  const zone = days[0].timezone
  const today = localDate(now, zone)
  const axis = axisOf(days)
  const hours = Array.from({ length: (axis.end - axis.start) / 60 }, (_, i) => axis.start + i * 60)
  const height = ((axis.end - axis.start) / 60) * HOUR_PX
  const week = days.length > 1
  const columns = { gridTemplateColumns: `3.5rem repeat(${days.length}, minmax(0, 1fr))` }

  if (!week && days[0].slots.length === 0 && days[0].bookings.length === 0) {
    return <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('staff.closedDay')}</p>
  }

  return (
    <div className="overflow-x-auto rounded-lg border bg-card">
      <div className={cn(week && 'min-w-[760px]')}>
        {/* the days across the top */}
        <div className="grid border-b" style={columns}>
          <span />
          {days.map((day) => {
            const { weekday, day: number } = shortDateParts(day.date)
            const isToday = day.date === today
            const content = (
              <>
                <span className={cn('text-xs uppercase', isToday ? 'text-primary' : 'text-muted-foreground')}>{weekday}</span>
                <span className={cn('num flex size-8 items-center justify-center rounded-full text-lg font-semibold', isToday && 'bg-primary text-primary-foreground')}>{number}</span>
              </>
            )
            return (
              <div key={day.date} className="flex justify-center border-l py-2">
                {week ? (
                  <button
                    type="button"
                    onClick={() => onPickDay(day.date)}
                    className="flex flex-col items-center rounded-md px-2 hover:bg-muted"
                    title={t('staff.openDay', { date: formatLongDate(day.date) })}
                  >
                    {content}
                  </button>
                ) : (
                  <h3 className="flex flex-col items-center" aria-label={formatLongDate(day.date)}>
                    {content}
                  </h3>
                )}
              </div>
            )
          })}
        </div>

        {/* the hours down the side and a column per day */}
        <div className="grid max-h-[70vh] overflow-y-auto" style={columns}>
          <div className="relative" style={{ height }} aria-hidden="true">
            {hours.map((minutes, i) => (
              <span key={minutes} className="num absolute right-2 -translate-y-1/2 text-xs text-muted-foreground" style={{ top: i * HOUR_PX }}>
                {i === 0 ? '' : `${String(minutes / 60).padStart(2, '0')}:00`}
              </span>
            ))}
          </div>
          {days.map((day) => (
            <DayColumn key={day.date} day={day} axis={axis} height={height} now={day.date === today ? now : null} onBook={onBook} onOpen={onOpen} />
          ))}
        </div>
      </div>
    </div>
  )
}

type DayColumnProps = {
  day: StaffDay
  axis: { start: number; end: number }
  height: number
  now: string | null // only on today's column
  onBook: (day: StaffDay, slot: StaffSlot) => void
  onOpen: (day: StaffDay, booking: StaffBooking) => void
}

function DayColumn({ day, axis, height, now, onBook, onOpen }: DayColumnProps) {
  const { t } = useTranslation('admin')
  const zone = day.timezone
  const place = (startsAt: string, endsAt: string) => {
    const start = minutesOfDay(startsAt, zone)
    const end = minutesOfDay(endsAt, zone) || 24 * 60 // a turn ending at midnight
    return { top: ((start - axis.start) / 60) * HOUR_PX, height: Math.max(((end - start) / 60) * HOUR_PX - 2, 18) }
  }
  const time = (startsAt: string, endsAt: string) => `${formatTime(startsAt, zone)}–${formatTime(endsAt, zone)}`
  const nowMinutes = now ? minutesOfDay(now, zone) : null

  return (
    <ul
      className="relative border-l bg-[repeating-linear-gradient(to_bottom,transparent_0,transparent_55px,var(--border)_55px,var(--border)_56px)]"
      style={{ height }}
      aria-label={t('staff.caption', { date: formatLongDate(day.date) })}
    >
      {/* free turns and closures; booked turns are drawn by their booking */}
      {day.slots.map((slot) => {
        if (slot.status !== 'free' && slot.status !== 'closed') return null
        const label = time(slot.starts_at, slot.ends_at)
        return (
          <li key={slot.starts_at} className="absolute inset-x-1" style={place(slot.starts_at, slot.ends_at)}>
            {slot.status === 'free' ? (
              <button
                type="button"
                onClick={() => onBook(day, slot)}
                className="group flex size-full items-start gap-1 overflow-hidden rounded-md border border-dashed border-pill-positive-ink/40 px-1.5 py-0.5 text-left text-xs text-pill-positive-ink hover:bg-pill-positive-bg"
                aria-label={`${t('staff.book')}: ${label}, ${t('staff.status.free').toLowerCase()}`}
              >
                <Plus className="mt-px size-3 shrink-0 opacity-60 group-hover:opacity-100" aria-hidden="true" />
                <span className="num truncate">{formatTime(slot.starts_at, zone)}</span>
              </button>
            ) : (
              <span className="hatch block size-full overflow-hidden rounded-md px-1.5 py-0.5 text-xs text-muted-foreground">
                <span className="sr-only">{label}: </span>
                {t('staff.status.closed')}
              </span>
            )}
          </li>
        )
      })}

      {day.bookings.map((b) => {
        const box = place(b.starts_at, b.ends_at)
        const label = time(b.starts_at, b.ends_at)
        return (
          <li key={b.id} className="absolute inset-x-1 z-10" style={box}>
            <button
              type="button"
              onClick={() => onOpen(day, b)}
              className={cn(
                'flex size-full flex-col overflow-hidden rounded-md border-l-4 px-1.5 py-0.5 text-left text-xs shadow-xs transition-[filter] hover:brightness-95 dark:hover:brightness-125',
                b.is_event
                  ? 'border-pill-positive-ink bg-pill-positive-bg text-pill-positive-ink'
                  : b.has_account ? 'border-primary bg-pill-accent-bg text-pill-accent-ink' : 'border-pill-neutral-ink/50 bg-pill-neutral-bg text-pill-neutral-ink',
              )}
            >
              <span className="flex items-center gap-1 font-semibold">
                {b.is_event && <PartyPopper className="size-3 shrink-0" aria-label={t('events.label')} />}
                <span className="truncate">{b.name || <span className="italic">{t('staff.noNameShort')}</span>}</span>
                {b.checked_in_at && <CircleCheck className="size-3 shrink-0" aria-hidden="true" />}
              </span>
              {/* a short block has room for the name only: the time is still read out */}
              <span className={cn('num truncate', box.height < 34 && 'sr-only')}>{label}</span>
            </button>
          </li>
        )
      })}

      {nowMinutes !== null && nowMinutes >= axis.start && nowMinutes <= axis.end && (
        <li className="pointer-events-none absolute inset-x-0 z-20 h-0.5 bg-now" style={{ top: ((nowMinutes - axis.start) / 60) * HOUR_PX }} aria-hidden="true">
          <span className="absolute -top-1 -left-1 size-2.5 rounded-full bg-now" />
        </li>
      )}
    </ul>
  )
}

function StatusLegend() {
  const { t } = useTranslation('admin')
  const items = [
    { label: t('staff.withAccount'), className: 'border-l-4 border-primary bg-pill-accent-bg' },
    { label: t('staff.withoutAccount'), className: 'border-l-4 border-pill-neutral-ink/50 bg-pill-neutral-bg' },
    { label: t('staff.status.free'), className: 'border border-dashed border-pill-positive-ink/60' },
    { label: t('staff.status.closed'), className: 'hatch' },
  ]
  return (
    <ul className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
      {items.map((item) => (
        <li key={item.label} className="flex items-center gap-1.5">
          <span className={cn('relative size-3 overflow-hidden rounded-[3px]', item.className)} aria-hidden="true" />
          {item.label}
        </li>
      ))}
    </ul>
  )
}

type DetailsProps = {
  slug: string
  token: string
  booking: StaffBooking
  zone: string
  facilityName: string
  reader: boolean // the facility has a QR reader: only then is the booking's QR any use
  onClose: () => void
  onCancel: () => void
  onChanged: () => void
}

/** One booking, opened from the calendar: who holds it, how to reach them, and what staff can do. */
function DetailsDialog({ slug, token, booking, zone, facilityName, reader, onClose, onCancel, onChanged }: DetailsProps) {
  const { t } = useTranslation('admin')
  const labels = { title: t('staff.ticketTitle'), hint: t('staff.ticketHint') }
  const [b, setB] = useState(booking)
  const [saving, setSaving] = useState(false)
  const [status, setStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  // after marking, focus goes to the undo button that takes the mark button's place, and back again
  const [focusNext, setFocusNext] = useState<'undo' | 'mark' | null>(null)
  const focusRef = (target: 'undo' | 'mark') => (el: HTMLButtonElement | null) => {
    if (el && focusNext === target) {
      el.focus()
      setFocusNext(null)
    }
  }

  const change = async (undo: boolean) => {
    setSaving(true)
    setError(null)
    setStatus(null)
    try {
      const updated = undo ? await api.council.undoStaffCheckIn(token, slug, b.id) : await api.council.staffCheckIn(token, slug, b.id)
      setB(updated)
      setStatus(undo ? t('staff.checkinUndone') : t('staff.checkinMarked', { time: formatTime(updated.checked_in_at ?? new Date().toISOString(), zone) }))
      setFocusNext(undo ? 'mark' : 'undo')
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  // the API only sends undo_until while the undo window is open; past it, the undo is refused with a reason
  const canUndo = b.undo_until !== null
  const opensAt = new Date(new Date(b.starts_at).getTime() - 15 * 60_000).toISOString()

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{b.name || t('staff.noName')}</DialogTitle>
          <DialogDescription className="num">
            {t('staff.when', { facility: facilityName, date: formatLongDate(localDate(b.starts_at, zone)), start: formatTime(b.starts_at, zone), end: formatTime(b.ends_at, zone) })}
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <p className="flex flex-wrap gap-1.5">
            {b.is_event ? (
              <span className={`${pill} bg-pill-positive-bg text-pill-positive-ink`}>{t('events.label')}</span>
            ) : (
              <span className={`${pill} ${b.has_account ? 'bg-pill-accent-bg text-pill-accent-ink' : 'bg-pill-neutral-bg text-pill-neutral-ink'}`}>
                {b.has_account ? t('staff.withAccount') : t('staff.withoutAccount')}
              </span>
            )}
            {b.booked_by_staff && !b.is_event && <span className={`${pill} bg-pill-neutral-bg text-pill-neutral-ink`}>{t('staff.bookedByStaff')}</span>}
            {b.checked_in_at && <span className={`${pill} num bg-pill-positive-bg text-pill-positive-ink`}>{t('staff.checkedIn', { time: formatTime(b.checked_in_at, zone) })}</span>}
            {b.participant_count > 0 && <span className={`${pill} num bg-pill-neutral-bg text-pill-neutral-ink`}>{t('staff.joined', { count: b.participant_count })}</span>}
          </p>
          {/* an upcoming event: staff sign people up, at the desk or on the phone */}
          {b.is_event && b.can_cancel && (
            <AddToEvent
              slug={slug}
              token={token}
              bookingId={b.id}
              onAdded={(name) => {
                setB((current) => ({ ...current, participant_count: current.participant_count + 1 }))
                setError(null)
                setStatus(t('events.added', { name }))
                onChanged()
              }}
            />
          )}
          {b.checked_in_at && b.checked_in_method && (
            <p className="text-sm text-muted-foreground">
              {b.checked_in_method === 'staff' && b.checked_in_by
                ? t('staff.checkedInBy', { name: b.checked_in_by })
                : t(`staff.checkedInMethod.${b.checked_in_method}`)}
              {canUndo && <span className="num"> {t('staff.undoUntil', { time: formatTime(b.undo_until!, zone) })}</span>}
            </p>
          )}
          {b.checkin_state === 'too_early' && <p className="num text-sm text-muted-foreground">{t('staff.checkinTooEarly', { time: formatTime(opensAt, zone) })}</p>}
          {b.checkin_state === 'ended' && <p className="text-sm text-muted-foreground">{t('staff.checkinEnded')}</p>}
          {(b.phone || b.email) && (
            <dl className="grid gap-1 break-all">
              <dt className="text-xs text-muted-foreground">{t('staff.colContact')}</dt>
              {b.phone && (
                <dd>
                  <a href={`tel:${b.phone}`} className="num underline-offset-4 hover:underline">
                    {b.phone}
                  </a>
                </dd>
              )}
              {b.email && <dd className="text-muted-foreground">{b.email}</dd>}
            </dl>
          )}
          <div aria-live="polite" className="empty:hidden">
            {error ? (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            ) : (
              status && <p className="num text-sm font-medium">{status}</p>
            )}
          </div>
        </div>
        <DialogFooter>
          {b.checkin_code && reader && (
            <Button variant="outline" onClick={() => void printTicket(b, zone, facilityName, labels)}>
              <Printer />
              {t('staff.printQr')}
            </Button>
          )}
          {b.can_cancel && (
            <Button variant="outline" className="text-destructive" onClick={onCancel}>
              {b.is_event ? t('events.confirmCancel') : t('staff.cancel')}
            </Button>
          )}
          {canUndo && (
            <Button ref={focusRef('undo')} variant="outline" disabled={saving} onClick={() => void change(true)}>
              <Undo2 />
              {saving ? t('staff.undoing') : t('staff.undoCheckin')}
            </Button>
          )}
          {b.checkin_state === 'open' && (
            <Button ref={focusRef('mark')} disabled={saving} onClick={() => void change(false)}>
              <UserCheck />
              {saving ? t('staff.marking') : t('staff.markCheckin')}
            </Button>
          )}
          <Button variant={b.checkin_state === 'open' ? 'outline' : 'default'} onClick={onClose}>
            {t('staff.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** Inside an event's dialog: sign someone up, with their account or by their name and age. */
export function AddToEvent({ slug, token, bookingId, onAdded }: { slug: string; token: string; bookingId: string; onAdded: (name: string) => void }) {
  const { t } = useTranslation('admin')
  const [open, setOpen] = useState(false)
  const [kind, setKind] = useState<'account' | 'name'>('name')
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [age, setAge] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!open) {
    return (
      <div>
        <Button variant="outline" size="sm" onClick={() => setOpen(true)}>
          <UserPlus />
          {t('events.addPerson')}
        </Button>
      </div>
    )
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    try {
      await api.council.addEventParticipant(token, slug, bookingId, kind === 'account' ? { email } : { name, age: Number(age) })
      onAdded(kind === 'account' ? email : name)
      setEmail('')
      setName('')
      setAge('')
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={(e) => void submit(e)} className="grid gap-3 rounded-md border p-3" aria-labelledby="add-to-event-title">
      <p id="add-to-event-title" className="font-medium">
        {t('events.addPerson')}
      </p>
      <ToggleGroup type="single" variant="outline" value={kind} onValueChange={(value) => value && setKind(value as 'account' | 'name')} aria-label={t('events.addWho')}>
        <ToggleGroupItem value="name">{t('staff.byName')}</ToggleGroupItem>
        <ToggleGroupItem value="account">{t('staff.byAccount')}</ToggleGroupItem>
      </ToggleGroup>
      {kind === 'account' ? (
        <div className="grid gap-1.5">
          <Label htmlFor="event-add-email">{t('staff.email')}</Label>
          <Input id="event-add-email" type="email" required maxLength={255} autoComplete="off" value={email} onChange={(e) => setEmail(e.target.value)} />
        </div>
      ) : (
        <div className="grid grid-cols-[1fr_6rem] gap-2">
          <div className="grid gap-1.5">
            <Label htmlFor="event-add-name">{t('staff.name')}</Label>
            <Input id="event-add-name" required maxLength={80} autoComplete="off" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="event-add-age">{t('events.age')}</Label>
            <Input id="event-add-age" type="number" required min={1} max={120} value={age} onChange={(e) => setAge(e.target.value)} />
          </div>
        </div>
      )}
      <div aria-live="polite" className="empty:hidden">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>
      <div>
        <Button type="submit" size="sm" disabled={saving}>
          {saving ? t('events.adding') : t('events.add')}
        </Button>
      </div>
    </form>
  )
}

type BookProps = {
  slug: string
  token: string
  facilityName: string
  day: StaffDay
  slot: StaffSlot
  onClose: () => void
  onBooked: () => void
}

function BookDialog({ slug, token, facilityName, day, slot, onClose, onBooked }: BookProps) {
  const { t } = useTranslation('admin')
  const zone = day.timezone
  const [kind, setKind] = useState<'account' | 'name'>('name')
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [told, setTold] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [booked, setBooked] = useState<StaffBooking | null>(null)
  const when = t('staff.when', { facility: facilityName, date: formatLongDate(day.date), start: formatTime(slot.starts_at, zone), end: formatTime(slot.ends_at, zone) })
  const reader = day.checkin_ways.includes('reader')
  // someone without an account has no app: only the reader (with the printed QR) and the staff let them in
  const ways = booked?.has_account ? day.checkin_ways : day.checkin_ways.filter((way) => way === 'reader' || way === 'staff')

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    try {
      const result = await api.council.staffBook(token, slug, {
        facility_id: day.facility_id,
        starts_at: slot.starts_at,
        email: kind === 'account' ? email : '',
        name: kind === 'name' ? name : '',
        phone: kind === 'name' ? phone : '',
        accept_rules: told,
      })
      setBooked(result)
      onBooked()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{booked ? t('staff.bookedTitle') : t('staff.bookTitle')}</DialogTitle>
          <DialogDescription className="num">{when}</DialogDescription>
        </DialogHeader>

        {booked ? (
          <div className="grid gap-4">
            <p>{booked.has_account ? t('staff.bookedAccount', { name: booked.name }) : t('staff.bookedName', { name: booked.name })}</p>
            {/* how they get in: the facility's ways; without an account, only the reader's QR (printed) and the staff work */}
            {ways.length > 0 ? (
              <div className="grid gap-1">
                <p className="font-medium">{t('staff.waysTitle')}</p>
                <ul className="grid list-disc gap-1 pl-5">
                  {ways.map((way) => (
                    <li key={way}>{t(`staff.ways.${way}`)}</li>
                  ))}
                </ul>
              </div>
            ) : (
              <p className="text-muted-foreground">{t(booked.has_account ? 'staff.waysNone' : 'staff.waysNoneWithoutAccount')}</p>
            )}
            {booked.checkin_code && reader && (
              <>
                <QrImage code={booked.checkin_code} alt={t('staff.qrAlt', { name: booked.name })} className="max-w-48" />
                <p className="text-muted-foreground">{t('staff.qrHint')}</p>
              </>
            )}
            <DialogFooter>
              {booked.checkin_code && reader && (
                <Button variant="outline" onClick={() => void printTicket(booked, zone, facilityName, { title: t('staff.ticketTitle'), hint: t('staff.ticketHint') })}>
                  <Printer />
                  {t('staff.printQr')}
                </Button>
              )}
              <Button onClick={onClose}>{t('staff.done')}</Button>
            </DialogFooter>
          </div>
        ) : (
          <form onSubmit={(e) => void submit(e)} className="grid gap-4">
            <ToggleGroup
              type="single"
              variant="outline"
              value={kind}
              onValueChange={(value) => value && setKind(value as 'account' | 'name')}
              aria-label={t('staff.whoLabel')}
              className="w-full"
            >
              <ToggleGroupItem value="name" className="flex-1">
                {t('staff.byName')}
              </ToggleGroupItem>
              <ToggleGroupItem value="account" className="flex-1">
                {t('staff.byAccount')}
              </ToggleGroupItem>
            </ToggleGroup>

            {kind === 'account' ? (
              <div className="grid gap-1.5">
                <Label htmlFor="staff-email">{t('staff.email')}</Label>
                <Input id="staff-email" type="email" autoComplete="off" required maxLength={255} value={email} onChange={(e) => setEmail(e.target.value)} aria-describedby="staff-email-help" />
                <p id="staff-email-help" className="text-xs text-muted-foreground">
                  {t('staff.emailHelp')}
                </p>
              </div>
            ) : (
              <>
                <div className="grid gap-1.5">
                  <Label htmlFor="staff-name">{t('staff.name')}</Label>
                  <Input id="staff-name" autoComplete="off" required maxLength={80} value={name} onChange={(e) => setName(e.target.value)} />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="staff-phone">
                    {t('staff.phone')} <span className="font-normal text-muted-foreground">({t('staff.optional')})</span>
                  </Label>
                  <Input id="staff-phone" type="tel" autoComplete="off" maxLength={30} value={phone} onChange={(e) => setPhone(e.target.value)} className="num" aria-describedby="staff-phone-help" />
                  <p id="staff-phone-help" className="text-xs text-muted-foreground">
                    {t('staff.privacyHelp')}
                  </p>
                </div>
              </>
            )}

            {day.min_age && <p className="text-muted-foreground">{t('staff.minAge', { age: day.min_age })}</p>}
            <div className="flex items-start gap-2">
              <Checkbox id="staff-told" checked={told} onCheckedChange={(value) => setTold(value === true)} className="mt-0.5" />
              <Label htmlFor="staff-told" className="leading-5 font-normal">
                {t('staff.toldRules')}
              </Label>
            </div>

            <div aria-live="polite" className="empty:hidden">
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={onClose}>
                {t('staff.close')}
              </Button>
              <Button type="submit" disabled={saving || !told}>
                {saving ? t('staff.booking') : t('staff.confirmBook')}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}

type CancelProps = {
  slug: string
  token: string
  facilityName: string
  timeZone: string
  booking: StaffBooking
  onClose: () => void
  onCancelled: () => void
}

function CancelDialog({ slug, token, facilityName, timeZone, booking, onClose, onCancelled }: CancelProps) {
  const { t } = useTranslation('admin')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const confirm = async () => {
    setSaving(true)
    setError(null)
    try {
      await api.council.staffCancel(token, slug, booking.id, reason.trim())
      onCancelled()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('staff.cancelTitle')}</DialogTitle>
          <DialogDescription className="num">
            {t('staff.cancelWhat', {
              name: booking.name || t('staff.noName'),
              facility: facilityName,
              start: formatTime(booking.starts_at, timeZone),
              end: formatTime(booking.ends_at, timeZone),
            })}
          </DialogDescription>
        </DialogHeader>
        <p>
          {booking.is_event
            ? t('events.cancelNotice', { count: booking.participant_count })
            : booking.has_account ? t('staff.cancelNotice') : booking.phone ? t('staff.cancelCall', { phone: booking.phone }) : t('staff.cancelNoContact')}
        </p>
        {(booking.has_account || booking.is_event) && (
          <div className="grid gap-1.5">
            <Label htmlFor="staff-reason">
              {t('staff.reason')} <span className="font-normal text-muted-foreground">({t('staff.optional')})</span>
            </Label>
            <Textarea id="staff-reason" maxLength={255} value={reason} onChange={(e) => setReason(e.target.value)} />
          </div>
        )}
        <div aria-live="polite" className="empty:hidden">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('staff.keep')}
          </Button>
          <Button variant="destructive" disabled={saving} onClick={() => void confirm()}>
            {saving ? t('staff.cancelling') : t('staff.confirmCancel')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

const escapeHtml = (text: string) => text.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`)

/** A small ticket with the entry QR, for someone without the app. */
async function printTicket(booking: StaffBooking, zone: string, facilityName: string, labels: { title: string; hint: string }) {
  const svg = await QRCode.toString(booking.checkin_code, { type: 'svg', errorCorrectionLevel: 'M', margin: 2, color: { dark: '#000000', light: '#ffffff' } })
  const ticket = window.open('', '_blank', 'width=420,height=640')
  if (!ticket) return
  const start = formatTime(booking.starts_at, zone)
  const end = formatTime(booking.ends_at, zone)
  const date = new Intl.DateTimeFormat(document.documentElement.lang || undefined, { dateStyle: 'full', timeZone: zone }).format(new Date(booking.starts_at))
  ticket.document.write(`<!doctype html><html lang="${escapeHtml(document.documentElement.lang)}"><head><meta charset="utf-8"><title>${escapeHtml(labels.title)}</title></head>
<body style="font-family:system-ui,sans-serif;text-align:center;padding:24px;color:#000">
<h1 style="font-size:20px;margin:0 0 8px">${escapeHtml(booking.name)}</h1>
<p style="margin:0 0 4px;font-size:16px">${escapeHtml(facilityName)}</p>
<p style="margin:0 0 16px;font-size:16px">${escapeHtml(date)}, ${start}–${end}</p>
<div style="width:240px;margin:0 auto">${svg}</div>
<p style="margin:16px 0 0;font-size:14px">${escapeHtml(labels.hint)}</p>
</body></html>`)
  ticket.document.close()
  ticket.focus()
  ticket.print()
}
