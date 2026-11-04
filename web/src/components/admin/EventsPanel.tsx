import { CalendarOff, PartyPopper, Plus } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { DatePicker } from '@/components/DatePicker'
import { TimeSelect } from '@/components/TimeSelect'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { useAsync } from '@/hooks/useAsync'
import { api, type AdminFacilitySummary, type CouncilEvent, type EventInTheWay, type EventInput } from '@/lib/api'
import { formatDateTime, formatTime, localDate } from '@/lib/time'

type Props = {
  slug: string
  facilities: AdminFacilitySummary[]
}

/**
 * The council's events («Torneo de fútbol 5v5»): residents find them on the facility's
 * page and in «For you», and join them like a public booking. Any start and end, not
 * the turns; they never take a resident's place, so whatever is in the way is cancelled
 * first from manual booking.
 */
export function EventsPanel({ slug, facilities }: Props) {
  const { t } = useTranslation('admin')
  const token = useAuth().user?.access_token ?? ''
  const active = facilities.filter((f) => f.is_active)
  const events = useAsync(() => api.council.events(token, slug), `council-events:${slug}`)
  const [editing, setEditing] = useState<CouncilEvent | 'new' | null>(null)
  const [cancelling, setCancelling] = useState<CouncilEvent | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const zone = events.data?.timezone ?? 'Europe/Madrid'

  const done = (message: string) => {
    setEditing(null)
    setCancelling(null)
    setNotice(message)
    events.reload()
  }

  return (
    <section aria-labelledby="events-title" className="grid gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="events-title" className="text-lg font-semibold">
            {t('events.title')}
          </h2>
          <p className="max-w-[70ch] text-muted-foreground">{t('events.help')}</p>
        </div>
        {active.length > 0 && (
          <Button onClick={() => { setNotice(null); setEditing('new') }}>
            <Plus />
            {t('events.create')}
          </Button>
        )}
      </div>

      <div aria-live="polite" className="empty:hidden">
        {notice && (
          <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
            <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
          </Alert>
        )}
        {events.error && (
          <Alert variant="destructive">
            <AlertDescription>{events.error.message}</AlertDescription>
          </Alert>
        )}
      </div>

      {active.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('staff.noFacilities')}</p>
      ) : events.loading && !events.data ? (
        <Skeleton className="h-24 w-full" />
      ) : events.data?.events.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('events.empty')}</p>
      ) : (
        <ul className="divide-y rounded-md border">
          {events.data?.events.map((event) => (
            <li key={event.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
              <div className="flex min-w-0 items-start gap-3">
                <PartyPopper className="mt-0.5 size-5 shrink-0 text-muted-foreground" aria-hidden />
                <div className="min-w-0">
                  <p className="font-medium break-words">{event.title}</p>
                  <p className="num text-sm text-muted-foreground">
                    {event.facility_name} · {formatDateTime(event.starts_at, zone)} – {sameDay(event, zone) ? formatTime(event.ends_at, zone) : formatDateTime(event.ends_at, zone)}
                  </p>
                  <p className="num text-sm text-muted-foreground">{t('events.joined', { count: event.participant_count, capacity: event.capacity })}</p>
                </div>
              </div>
              {event.can_change && (
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={() => { setNotice(null); setEditing(event) }}>
                    {t('events.edit')}
                  </Button>
                  <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => { setNotice(null); setCancelling(event) }}>
                    {t('events.cancel')}
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      {editing && (
        <EventDialog
          slug={slug}
          token={token}
          zone={zone}
          facilities={active}
          event={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={(saved) => done(t(editing === 'new' ? 'events.created' : 'events.saved', { title: saved.title }))}
        />
      )}
      {cancelling && (
        <CancelEventDialog
          slug={slug}
          token={token}
          zone={zone}
          event={cancelling}
          onClose={() => setCancelling(null)}
          onCancelled={() => done(t('events.cancelled', { title: cancelling.title }))}
        />
      )}
    </section>
  )
}

const sameDay = (event: CouncilEvent, zone: string) => localDate(event.starts_at, zone) === localDate(event.ends_at, zone)

type DialogProps = {
  slug: string
  token: string
  zone: string
  facilities: AdminFacilitySummary[]
  event: CouncilEvent | null // null: a new one
  onClose: () => void
  onSaved: (event: CouncilEvent) => void
}

function EventDialog({ slug, token, zone, facilities, event, onClose, onSaved }: DialogProps) {
  const { t } = useTranslation(['admin', 'common'])
  const [today] = useState(() => localDate(new Date().toISOString(), zone))
  const [fields, setFields] = useState(() => ({
    facilityId: event?.facility_id ?? facilities[0]?.id ?? '',
    fromDate: event ? localDate(event.starts_at, zone) : '',
    fromTime: event ? formatTime(event.starts_at, zone) : '10:00',
    untilDate: event ? localDate(event.ends_at, zone) : '',
    untilTime: event ? formatTime(event.ends_at, zone) : '14:00',
    title: event?.title ?? '',
    description: event?.description ?? '',
    capacity: event?.own_capacity ? String(event.own_capacity) : '',
  }))
  const set = (field: keyof typeof fields, value: string) => setFields((f) => ({ ...f, [field]: value }))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [inTheWay, setInTheWay] = useState<EventInTheWay | null>(null)

  const input: EventInput = {
    facility_id: fields.facilityId,
    starts_at: fields.fromDate && `${fields.fromDate}T${fields.fromTime}`,
    ends_at: fields.untilDate && `${fields.untilDate}T${fields.untilTime}`,
    title: fields.title,
    description: fields.description,
    capacity: fields.capacity.trim() ? Number(fields.capacity) : null,
  }
  const missing = !fields.facilityId || !fields.fromDate || !fields.untilDate || !fields.title.trim()

  // first what is in the way; with nothing, save
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (busy || missing) return
    setBusy(true)
    setError(null)
    setInTheWay(null)
    try {
      const found = await api.council.previewEvent(token, slug, input, event?.id)
      if (found.bookings.length > 0 || found.closures.length > 0) {
        setInTheWay(found)
        return
      }
      onSaved(event ? await api.council.updateEvent(token, slug, event.id, input) : await api.council.createEvent(token, slug, input))
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && !busy && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t(event ? 'events.editTitle' : 'events.createTitle')}</DialogTitle>
          <DialogDescription>{t('events.formHelp')}</DialogDescription>
        </DialogHeader>
        <form id="event-form" onSubmit={(e) => void submit(e)} className="grid gap-3 sm:grid-cols-2">
          <div className="grid gap-1.5 sm:col-span-2">
            <Label htmlFor="event-title">{t('events.name')}</Label>
            <Input id="event-title" required maxLength={120} placeholder={t('events.namePlaceholder')} value={fields.title} onChange={(e) => set('title', e.target.value)} />
          </div>
          <div className="grid gap-1.5 sm:col-span-2">
            <Label htmlFor="event-facility">{t('staff.facility')}</Label>
            {/* an event stays where it was created: moving it would leave who joined somewhere else */}
            <Select value={fields.facilityId} onValueChange={(id) => set('facilityId', id)} disabled={!!event}>
              <SelectTrigger id="event-facility" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {facilities.map((f) => (
                  <SelectItem key={f.id} value={f.id}>
                    {f.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="event-from">{t('events.from')}</Label>
            <div className="flex gap-2">
              <DatePicker
                id="event-from"
                value={fields.fromDate}
                onChange={(date) => setFields((f) => ({ ...f, fromDate: date, untilDate: !f.untilDate || f.untilDate < date ? date : f.untilDate }))}
                min={today}
                defaultMonth={fields.fromDate || today}
                placeholder={t('closures.pickDay')}
                className="min-w-0 flex-1"
              />
              <TimeSelect value={fields.fromTime} onChange={(time) => set('fromTime', time)} aria-label={t('events.fromTime')} className="w-24" />
            </div>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="event-until">{t('events.until')}</Label>
            <div className="flex gap-2">
              <DatePicker
                id="event-until"
                value={fields.untilDate}
                onChange={(date) => set('untilDate', date)}
                min={fields.fromDate || today}
                defaultMonth={fields.untilDate || fields.fromDate || today}
                placeholder={t('closures.pickDay')}
                className="min-w-0 flex-1"
              />
              <TimeSelect value={fields.untilTime} onChange={(time) => set('untilTime', time)} aria-label={t('events.untilTime')} className="w-24" />
            </div>
          </div>
          <div className="grid gap-1.5 sm:col-span-2">
            <Label htmlFor="event-capacity">
              {t('events.capacity')} <span className="font-normal text-muted-foreground">({t('staff.optional')})</span>
            </Label>
            <Input
              id="event-capacity"
              type="number"
              inputMode="numeric"
              min={Math.max(1, event?.participant_count ?? 1)}
              max={10000}
              className="num w-32"
              value={fields.capacity}
              onChange={(e) => set('capacity', e.target.value)}
              aria-describedby="event-capacity-help"
            />
            <p id="event-capacity-help" className="text-sm text-muted-foreground">{t('events.capacityHelp')}</p>
          </div>
          <div className="grid gap-1.5 sm:col-span-2">
            <Label htmlFor="event-description">
              {t('events.description')} <span className="font-normal text-muted-foreground">({t('staff.optional')})</span>
            </Label>
            <Textarea id="event-description" maxLength={1000} placeholder={t('events.descriptionPlaceholder')} value={fields.description} onChange={(e) => set('description', e.target.value)} />
          </div>
        </form>

        <div aria-live="polite" className="empty:hidden">
          {inTheWay && <InTheWay found={inTheWay} zone={zone} slug={slug} facilityId={fields.facilityId} />}
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={busy}>
            {t('form.cancel')}
          </Button>
          <Button type="submit" form="event-form" disabled={busy || missing}>
            {busy ? t('events.saving') : t(event ? 'events.save' : 'events.createConfirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** What stops the event: the bookings to cancel first from manual booking, and the closures. */
function InTheWay({ found, zone, slug, facilityId }: { found: EventInTheWay; zone: string; slug: string; facilityId: string }) {
  const { t } = useTranslation('admin')
  return (
    <Alert className="border-pill-accent-ink bg-pill-accent-bg text-pill-accent-ink">
      <AlertDescription className="grid gap-2 text-pill-accent-ink">
        {found.bookings.length > 0 && (
          <>
            <p className="font-medium">{t('events.bookingsInTheWay', { count: found.bookings.length })}</p>
            <ul className="grid gap-1">
              {found.bookings.map((b) => (
                <li key={b.id} className="num">
                  {formatDateTime(b.starts_at, zone)}–{formatTime(b.ends_at, zone)} · {b.holder || t('staff.noName')}
                  {b.is_event && ` (${t('events.label')})`}
                </li>
              ))}
            </ul>
            <p>
              <Link to={`/${slug}/admin?vista=reservas&instalacion=${facilityId}`} className="font-medium underline underline-offset-4">
                {t('events.goCancel')}
              </Link>
            </p>
          </>
        )}
        {found.closures.length > 0 && (
          <>
            <p className="font-medium">{t('events.closuresInTheWay')}</p>
            <ul className="grid gap-1">
              {found.closures.map((c) => (
                <li key={c.starts_at} className="num flex items-center gap-1.5">
                  <CalendarOff className="size-4 shrink-0" aria-hidden />
                  {formatDateTime(c.starts_at, zone)} – {formatDateTime(c.ends_at, zone)}
                  {c.reason && ` · ${c.reason}`}
                </li>
              ))}
            </ul>
          </>
        )}
      </AlertDescription>
    </Alert>
  )
}

type CancelProps = {
  slug: string
  token: string
  zone: string
  event: CouncilEvent
  onClose: () => void
  onCancelled: () => void
}

/** Everyone who joined with an account is told, with the reason when there is one. */
function CancelEventDialog({ slug, token, zone, event, onClose, onCancelled }: CancelProps) {
  const { t } = useTranslation('admin')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const confirm = async () => {
    setSaving(true)
    setError(null)
    try {
      await api.council.staffCancel(token, slug, event.id, reason.trim())
      onCancelled()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && !saving && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('events.cancelTitle', { title: event.title })}</DialogTitle>
          <DialogDescription className="num">
            {event.facility_name} · {formatDateTime(event.starts_at, zone)}
          </DialogDescription>
        </DialogHeader>
        <p>{t('events.cancelNotice', { count: event.participant_count })}</p>
        <div className="grid gap-1.5">
          <Label htmlFor="event-reason">
            {t('staff.reason')} <span className="font-normal text-muted-foreground">({t('staff.optional')})</span>
          </Label>
          <Textarea id="event-reason" maxLength={255} value={reason} onChange={(e) => setReason(e.target.value)} />
        </div>
        <div aria-live="polite" className="empty:hidden">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={saving}>
            {t('staff.keep')}
          </Button>
          <Button variant="destructive" disabled={saving} onClick={() => void confirm()}>
            {saving ? t('staff.cancelling') : t('events.confirmCancel')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
