import { PartyPopper, Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useLocation, useNavigate } from 'react-router'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { useAsync } from '@/hooks/useAsync'
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
  const events = useAsync(() => api.council.events(token, slug), `council-events:${slug}`)
  const [cancelling, setCancelling] = useState<CouncilEvent | null>(null)
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
                  <Button asChild variant="outline" size="sm">
                    <Link to={`/${slug}/admin/eventos/${event.id}`}>
                      {t('events.edit')}
                    </Link>
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
