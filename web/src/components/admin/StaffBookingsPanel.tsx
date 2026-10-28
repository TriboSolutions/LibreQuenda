import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import QRCode from 'qrcode'
import { ChevronLeft, ChevronRight, Printer } from 'lucide-react'
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
import { useAsync } from '@/hooks/useAsync'
import { api, type AdminFacilitySummary, type StaffBooking, type StaffDay, type StaffSlot } from '@/lib/api'
import { addDays, formatLongDate, formatTime } from '@/lib/time'

// facilityId: the one chosen with its row's "Reservas" button (kept in the URL); "" = the first
type Props = { slug: string; facilities: AdminFacilitySummary[]; facilityId: string; onFacilityChange: (id: string) => void }

const pill = 'rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap'

/*
  The bookings view (each facility's "Reservas" button): one facility's day, turn by turn, with who holds each
  booking. Staff book a free turn for someone who phoned or came to the desk
  (their account, or just a name), and cancel any booking before it starts.
*/
export function StaffBookingsPanel({ slug, facilities, facilityId, onFacilityChange }: Props) {
  const { t } = useTranslation('admin')
  const token = useAuth().user?.access_token ?? ''
  const active = facilities.filter((f) => f.is_active)
  const [date, setDate] = useState('') // "" = today, in the council's time zone
  const facility = active.find((f) => f.id === facilityId) ?? active[0]
  const day = useAsync(facility ? () => api.council.staffDay(token, slug, facility.id, date) : null, `staff-day:${slug}:${facility?.id}:${date}`)
  const [booking, setBooking] = useState<StaffSlot | null>(null)
  const [cancelling, setCancelling] = useState<StaffBooking | null>(null)

  if (active.length === 0) {
    return <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('staff.noFacilities')}</p>
  }
  const shownDate = day.data?.date ?? date

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
            <Button variant="outline" size="icon" disabled={!shownDate} onClick={() => setDate(addDays(shownDate, -1))} aria-label={t('staff.previousDay')}>
              <ChevronLeft />
            </Button>
            <DatePicker id="staff-date" value={shownDate} onChange={setDate} className="w-44" />
            <Button variant="outline" size="icon" disabled={!shownDate} onClick={() => setDate(addDays(shownDate, 1))} aria-label={t('staff.nextDay')}>
              <ChevronRight />
            </Button>
          </div>
        </div>
        <Button variant="ghost" onClick={() => setDate('')}>
          {t('staff.today')}
        </Button>
      </div>

      {day.data?.min_age && <p className="text-muted-foreground">{t('staff.minAge', { age: day.data.min_age })}</p>}

      {day.error && !day.data ? (
        <Alert variant="destructive">
          <AlertDescription>{day.error.message}</AlertDescription>
        </Alert>
      ) : !day.data ? (
        <Skeleton className="h-64 w-full" />
      ) : (
        <DayTable day={day.data} facilityName={facility?.name ?? ''} onBook={setBooking} onCancel={setCancelling} />
      )}

      {booking && day.data && facility && (
        <BookDialog
          slug={slug}
          token={token}
          facilityName={facility.name}
          day={day.data}
          slot={booking}
          onClose={() => setBooking(null)}
          onBooked={day.reload}
        />
      )}
      {cancelling && day.data && facility && (
        <CancelDialog
          slug={slug}
          token={token}
          facilityName={facility.name}
          timeZone={day.data.timezone}
          booking={cancelling}
          onClose={() => setCancelling(null)}
          onCancelled={() => {
            setCancelling(null)
            day.reload()
          }}
        />
      )}
    </section>
  )
}

type DayTableProps = { day: StaffDay; facilityName: string; onBook: (slot: StaffSlot) => void; onCancel: (booking: StaffBooking) => void }

function DayTable({ day, facilityName, onBook, onCancel }: DayTableProps) {
  const { t } = useTranslation('admin')
  const zone = day.timezone
  const byId = new Map(day.bookings.map((b) => [b.id, b]))
  // a booking is listed once, on its first turn (it may cover more than one)
  const listed = new Set<string>()
  type Row = { key: string; starts_at: string; ends_at: string; slot: StaffSlot | null; booking: StaffBooking | null }
  const rows: Row[] = []
  for (const slot of day.slots) {
    const booking = slot.booking_id ? (byId.get(slot.booking_id) ?? null) : null
    if (booking && listed.has(booking.id)) continue
    if (booking) listed.add(booking.id)
    rows.push({ key: slot.starts_at, starts_at: booking?.starts_at ?? slot.starts_at, ends_at: booking?.ends_at ?? slot.ends_at, slot, booking })
  }
  // bookings outside today's turns (the hours changed after they were made)
  for (const booking of day.bookings) {
    if (!listed.has(booking.id)) rows.push({ key: booking.id, starts_at: booking.starts_at, ends_at: booking.ends_at, slot: null, booking })
  }
  rows.sort((a, b) => a.starts_at.localeCompare(b.starts_at))

  if (rows.length === 0) {
    return <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('staff.closedDay')}</p>
  }

  return (
    <div className="overflow-x-auto rounded-lg border bg-card">
      <table className="w-full min-w-[640px]">
        <caption className="sr-only">{t('staff.caption', { date: formatLongDate(day.date) })}</caption>
        <thead>
          <tr className="text-xs text-muted-foreground">
            <th scope="col" className="px-4 py-2 text-left font-normal">
              {t('staff.colTime')}
            </th>
            <th scope="col" className="px-4 py-2 text-left font-normal">
              {t('staff.colHolder')}
            </th>
            <th scope="col" className="px-4 py-2 text-left font-normal">
              {t('staff.colContact')}
            </th>
            <th scope="col" className="px-4 py-2">
              <span className="sr-only">{t('staff.colActions')}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const time = `${formatTime(row.starts_at, zone)}–${formatTime(row.ends_at, zone)}`
            return (
              <tr key={row.key} className="border-t align-top">
                <th scope="row" className="num px-4 py-3 text-left font-medium whitespace-nowrap">
                  {time}
                </th>
                {row.booking ? (
                  <BookingCells booking={row.booking} zone={zone} time={time} facilityName={facilityName} onCancel={onCancel} />
                ) : (
                  <SlotCells slot={row.slot!} time={time} onBook={onBook} />
                )}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

type BookingCellsProps = { booking: StaffBooking; zone: string; time: string; facilityName: string; onCancel: (b: StaffBooking) => void }

function BookingCells({ booking: b, zone, time, facilityName, onCancel }: BookingCellsProps) {
  const { t } = useTranslation('admin')
  const labels = { title: t('staff.ticketTitle'), hint: t('staff.ticketHint') }
  return (
    <>
      <td className="px-4 py-3">
        <span className="block font-medium">{b.name || <span className="text-muted-foreground">{t('staff.noName')}</span>}</span>
        <span className="mt-1 flex flex-wrap gap-1.5">
          <span className={`${pill} ${b.has_account ? 'bg-pill-accent-bg text-pill-accent-ink' : 'bg-pill-neutral-bg text-pill-neutral-ink'}`}>
            {b.has_account ? t('staff.withAccount') : t('staff.withoutAccount')}
          </span>
          {b.booked_by_staff && <span className={`${pill} bg-pill-neutral-bg text-pill-neutral-ink`}>{t('staff.bookedByStaff')}</span>}
          {b.checked_in_at && (
            <span className={`${pill} num bg-pill-positive-bg text-pill-positive-ink`}>{t('staff.checkedIn', { time: formatTime(b.checked_in_at, zone) })}</span>
          )}
          {b.participant_count > 0 && <span className={`${pill} num bg-pill-neutral-bg text-pill-neutral-ink`}>{t('staff.joined', { count: b.participant_count })}</span>}
        </span>
      </td>
      <td className="px-4 py-3 break-all">
        {b.phone && (
          <a href={`tel:${b.phone}`} className="num block underline-offset-4 hover:underline">
            {b.phone}
          </a>
        )}
        {b.email && <span className="block text-muted-foreground">{b.email}</span>}
      </td>
      <td className="px-4 py-3 text-right whitespace-nowrap">
        {b.checkin_code && (
          <Button variant="ghost" size="sm" onClick={() => void printTicket(b, zone, facilityName, labels)} aria-label={`${t('staff.printQr')}: ${b.name}, ${time}`}>
            <Printer />
            {t('staff.printQr')}
          </Button>
        )}
        {b.can_cancel && (
          <Button variant="ghost" size="sm" className="text-destructive" onClick={() => onCancel(b)} aria-label={`${t('staff.cancel')}: ${b.name}, ${time}`}>
            {t('staff.cancel')}
          </Button>
        )}
      </td>
    </>
  )
}

function SlotCells({ slot, time, onBook }: { slot: StaffSlot; time: string; onBook: (slot: StaffSlot) => void }) {
  const { t } = useTranslation('admin')
  return (
    <>
      <td className="px-4 py-3" colSpan={2}>
        {slot.status === 'free' ? (
          <span className={`${pill} bg-pill-positive-bg text-pill-positive-ink`}>{t('staff.status.free')}</span>
        ) : (
          <span className="text-muted-foreground">{t(`staff.status.${slot.status}`)}</span>
        )}
      </td>
      <td className="px-4 py-3 text-right">
        {slot.status === 'free' && (
          <Button size="sm" onClick={() => onBook(slot)} aria-label={`${t('staff.book')}: ${time}`}>
            {t('staff.book')}
          </Button>
        )}
      </td>
    </>
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
            {booked.checkin_code && (
              <>
                <QrImage code={booked.checkin_code} alt={t('staff.qrAlt', { name: booked.name })} className="max-w-48" />
                <p className="text-muted-foreground">{t('staff.qrHint')}</p>
              </>
            )}
            <DialogFooter>
              {booked.checkin_code && (
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
                <Input id="staff-email" type="email" autoComplete="off" required value={email} onChange={(e) => setEmail(e.target.value)} aria-describedby="staff-email-help" />
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
        <p>{booking.has_account ? t('staff.cancelNotice') : booking.phone ? t('staff.cancelCall', { phone: booking.phone }) : t('staff.cancelNoContact')}</p>
        {booking.has_account && (
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
