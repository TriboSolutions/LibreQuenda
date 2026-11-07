import { PartyPopper, Plus, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useLocation, useNavigate } from 'react-router'
import { EventPeopleDialog } from '@/components/admin/EventPeopleDialog'
import { HistoryToggle } from '@/components/admin/HistoryToggle'
import { Pager } from '@/components/Pager'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { FILTER_DEBOUNCE_MS, useAsync } from '@/hooks/useAsync'
import { api, type AdminFacilitySummary, type CouncilEvent } from '@/lib/api'
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
  // the upcoming ones, or the history (ended or cancelled), a page at a time
  const [history, setHistory] = useState(false)
  const [page, setPage] = useState(1)
  const events = useAsync(() => api.council.events(token, slug, history, page), `council-events:${slug}:${history}:${page}`, { debounceMs: FILTER_DEBOUNCE_MS })
  const [cancelling, setCancelling] = useState<CouncilEvent | null>(null)
  const [seeing, setSeeing] = useState<CouncilEvent | null>(null) // whose people are open
  // what the editor page saved, handed over on the way back
  const location = useLocation()
  const navigate = useNavigate()
  const [notice, setNotice] = useState<string | null>(() => (location.state as { notice?: string } | null)?.notice ?? null)
  useEffect(() => {
    // said once: a reload or going back here does not say it again
    if ((location.state as { notice?: string } | null)?.notice) navigate(location.pathname + location.search, { replace: true, state: null })
  }, [location, navigate])
  const zone = events.data?.timezone ?? 'Europe/Madrid'

  const done = (message: string) => {
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
          <Button asChild>
            <Link to={`/${slug}/admin/eventos/novo`}>
              <Plus />
              {t('events.create')}
            </Link>
          </Button>
        )}
      </div>

      <HistoryToggle
        history={history}
        onChange={(on) => {
          setHistory(on)
          setPage(1)
        }}
        current={t('events.upcoming')}
      />

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
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{history ? t('events.emptyHistory') : t('events.empty')}</p>
      ) : (
        <ul className="divide-y rounded-md border">
          {events.data?.events.map((event) => (
            <li key={event.id} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-4 py-3">
              <div className="flex min-w-0 items-start gap-3">
                <PartyPopper className="mt-0.5 size-5 shrink-0 text-muted-foreground" aria-hidden />
                <div className="min-w-0">
                  <p className="font-medium break-words">
                    {event.title}
                    {event.cancelled && <span className="ml-2 rounded-full bg-pill-negative-bg px-2 py-0.5 align-middle text-xs font-medium text-pill-negative-ink">{t('events.cancelledPill')}</span>}
                  </p>
                  <p className="num text-sm text-muted-foreground">
                    {event.facility_name} · {formatDateTime(event.starts_at, zone)} – {sameDay(event, zone) ? formatTime(event.ends_at, zone) : formatDateTime(event.ends_at, zone)}
                  </p>
                  <p className="num text-sm text-muted-foreground">{t('events.joined', { count: event.participant_count, capacity: event.capacity })}</p>
                </div>
              </div>
              <div className="flex flex-wrap justify-end gap-2">
                <Button variant="outline" size="sm" onClick={() => setSeeing(event)}>
                  <Users />
                  {t('events.peopleList.people', { count: event.participant_count })}
                </Button>
                {event.can_change && (
                  <>
                    <Button asChild variant="outline" size="sm">
                      <Link to={`/${slug}/admin/eventos/${event.id}`}>{t('events.edit')}</Link>
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => {
                        setNotice(null)
                        setCancelling(event)
                      }}
                    >
                      {t('events.cancel')}
                    </Button>
                  </>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      <Pager pagination={events.data?.pagination} onChange={setPage} label={t('events.pages')} />

      {seeing && <EventPeopleDialog slug={slug} token={token} zone={zone} event={seeing} onClose={() => setSeeing(null)} onChanged={() => events.reload()} />}

      {cancelling && (
        <CancelEventDialog slug={slug} token={token} zone={zone} event={cancelling} onClose={() => setCancelling(null)} onCancelled={() => done(t('events.cancelled', { title: cancelling.title }))} />
      )}
    </section>
  )
}

const sameDay = (event: CouncilEvent, zone: string) => localDate(event.starts_at, zone) === localDate(event.ends_at, zone)

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
