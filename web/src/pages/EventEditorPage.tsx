import { ArrowLeft, CalendarOff } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useNavigate, useParams } from 'react-router'
import { DatePicker } from '@/components/DatePicker'
import { SiteFooter } from '@/components/SiteFooter'
import { TimeSelect } from '@/components/TimeSelect'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { useAsync } from '@/hooks/useAsync'
import { useCouncilAdmin } from '@/hooks/useCouncilAdmin'
import { usePageTitle } from '@/hooks/usePageTitle'
import { api, type AdminFacilitySummary, type CouncilEvent, type EventInTheWay, type EventInput } from '@/lib/api'
import { formatDateTime, formatTime, localDate } from '@/lib/time'

/** Where the editor goes back to: the council admin's Events tab. */
const eventsTab = (slug: string) => `/${slug}/admin?vista=eventos`

/**
 * /:slug/admin/eventos/:id ("novo" to create): one of the council's events. Saving goes
 * back to the Events tab, which says what was saved.
 */
export function EventEditorPage() {
  const { slug = '', id = 'novo' } = useParams()
  const creating = id === 'novo'
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const allowed = useCouncilAdmin(slug)
  const tenant = useAsync(() => api.tenant(slug), `tenant:${slug}`)
  const facilities = useAsync(allowed ? () => api.council.facilities(token, slug) : null, `council-admin:${slug}:${allowed}`)
  const events = useAsync(allowed ? () => api.council.events(token, slug) : null, `council-events:${slug}:${allowed}`)
  const event = creating ? null : events.data?.events.find((e) => e.id === id)
  usePageTitle(creating ? t('admin:events.createTitle') : t('admin:events.editTitle'), tenant.data?.name)

  const active = facilities.data?.facilities.filter((f) => f.is_active) ?? []
  const failed = facilities.error ?? events.error
  const ready = facilities.data && events.data
  // an event that has started (or was cancelled) is no longer in the list of ones to change
  const gone = ready && !creating && (!event || !event.can_change)

  return (
    <div className="min-h-screen">
      <TopBar tenantSlug={slug} tenantName={tenant.data?.name} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <Link to={eventsTab(slug)} className="inline-flex items-center gap-1.5 text-muted-foreground hover:text-foreground">
          <ArrowLeft className="size-4" aria-hidden="true" />
          {t('admin:events.back')}
        </Link>
        <h1 className="mt-3 text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">
          {t(creating ? 'admin:events.createTitle' : 'admin:events.editTitle')}
        </h1>
        <p className="mt-2 max-w-[70ch] text-muted-foreground">{t('admin:events.formHelp')}</p>

        {failed ? (
          <Alert variant="destructive" className="mt-6 max-w-[640px]">
            <AlertDescription>{failed.message}</AlertDescription>
          </Alert>
        ) : !allowed && !auth.isLoading ? null : !ready ? (
          <Skeleton className="mt-6 h-96 w-full max-w-[640px]" />
        ) : gone ? (
          <p className="mt-6 max-w-[640px] rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('admin:events.notFound')}</p>
        ) : active.length === 0 ? (
          <p className="mt-6 max-w-[640px] rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('admin:staff.noFacilities')}</p>
        ) : (
          <EventForm key={id} slug={slug} token={token} zone={events.data?.timezone ?? 'Europe/Madrid'} facilities={active} event={event ?? null} />
        )}
      </main>
      <SiteFooter />
    </div>
  )
}

type FormProps = {
  slug: string
  token: string
  zone: string
  facilities: AdminFacilitySummary[]
  event: CouncilEvent | null // null: a new one
}

function EventForm({ slug, token, zone, facilities, event }: FormProps) {
  const { t } = useTranslation(['admin', 'common'])
  const navigate = useNavigate()
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

  // first what is in the way; with nothing, save and go back to the list, which says so
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
      const saved = event ? await api.council.updateEvent(token, slug, event.id, input) : await api.council.createEvent(token, slug, input)
      navigate(eventsTab(slug), { state: { notice: t(event ? 'events.saved' : 'events.created', { title: saved.title }) } })
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={(e) => void submit(e)} className="mt-6 grid max-w-[640px] gap-5 sm:grid-cols-2">
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
        <Textarea id="event-description" rows={5} maxLength={1000} placeholder={t('events.descriptionPlaceholder')} value={fields.description} onChange={(e) => set('description', e.target.value)} />
      </div>

      <div aria-live="polite" className="empty:hidden sm:col-span-2">
        {inTheWay && <InTheWay found={inTheWay} zone={zone} slug={slug} facilityId={fields.facilityId} />}
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>

      <div className="flex flex-wrap gap-2 sm:col-span-2">
        <Button type="submit" disabled={busy || missing}>
          {busy ? t('events.saving') : t(event ? 'events.save' : 'events.createConfirm')}
        </Button>
        <Button asChild variant="outline">
          <Link to={eventsTab(slug)}>{t('form.cancel')}</Link>
        </Button>
      </div>
    </form>
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
