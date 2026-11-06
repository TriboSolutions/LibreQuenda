import { X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AddToEvent } from '@/components/admin/StaffBookingsPanel'
import { RatingBadge } from '@/components/Stars'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type CouncilEvent, type EventPerson } from '@/lib/api'
import { formatDateTime } from '@/lib/time'

type Props = {
  slug: string
  token: string
  zone: string
  event: CouncilEvent
  onClose: () => void
  onChanged: () => void // the count in the list
}

const pill = 'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium'

/**
 * Who is in one of the council's events: name and surnames, email, the stars others
 * gave them and whether the council verified them as residents (or a guest's name
 * and age). Staff sign people up and take them out: someone with an account is told,
 * and cannot sign up again.
 */
export function EventPeopleDialog({ slug, token, zone, event, onClose, onChanged }: Props) {
  const { t } = useTranslation('admin')
  const people = useAsync(() => api.council.eventPeople(token, slug, event.id), `event-people:${event.id}`)
  const [busy, setBusy] = useState(false)
  const [confirming, setConfirming] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)

  const remove = async (person: EventPerson) => {
    setBusy(true)
    setError(null)
    setStatus(null)
    try {
      await api.council.removeEventParticipant(token, slug, event.id, person.id)
      setConfirming(null)
      setStatus(t('events.peopleList.removed', { name: person.name }))
      people.reload()
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const list = people.data?.people ?? []
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="break-words">{t('events.peopleList.peopleTitle', { title: event.title })}</DialogTitle>
          <DialogDescription className="num">
            {event.facility_name} · {formatDateTime(event.starts_at, zone)} · {t('events.joined', { count: people.data ? list.length : event.participant_count, capacity: event.capacity })}
          </DialogDescription>
        </DialogHeader>

        {people.error ? (
          <p className="text-destructive">{people.error.message}</p>
        ) : !people.data ? (
          <Skeleton className="h-24 w-full" />
        ) : list.length === 0 ? (
          <p className="text-muted-foreground">{t('events.peopleList.nobody')}</p>
        ) : (
          <ul className="divide-y">
            {list.map((p) => (
              <li key={p.id} className="grid gap-2 py-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="grid min-w-0 gap-1">
                    <span className="font-medium break-words">{p.name}</span>
                    {p.has_account ? (
                      <a href={`mailto:${p.email}`} className="text-sm break-all text-muted-foreground underline-offset-4 hover:underline">
                        {p.email}
                      </a>
                    ) : (
                      <span className="num text-sm text-muted-foreground">{t('events.peopleList.guest', { age: p.age })}</span>
                    )}
                    {p.has_account && (
                      <span className="flex flex-wrap items-center gap-2">
                        <span className={`${pill} ${p.is_resident ? 'bg-pill-positive-bg text-pill-positive-ink' : 'bg-pill-neutral-bg text-pill-neutral-ink'}`}>
                          {p.is_resident ? t('events.peopleList.resident') : t('events.peopleList.notResident')}
                        </span>
                        {p.rating ? <RatingBadge rating={p.rating} /> : <span className="text-xs text-muted-foreground">{t('events.peopleList.noRating')}</span>}
                      </span>
                    )}
                  </div>
                  {confirming !== p.id && (
                    <Button variant="ghost" size="icon" aria-label={t('events.peopleList.remove', { name: p.name })} disabled={busy} onClick={() => setConfirming(p.id)}>
                      <X aria-hidden />
                    </Button>
                  )}
                </div>
                {confirming === p.id && (
                  <div role="group" aria-label={t('events.peopleList.remove', { name: p.name })} className="grid gap-2 rounded-md bg-muted p-3 text-sm">
                    <p>{p.has_account ? t('events.peopleList.removeConfirm', { name: p.name }) : t('events.peopleList.removeGuestConfirm', { name: p.name })}</p>
                    <div className="flex flex-wrap gap-2">
                      <Button size="sm" variant="destructive" disabled={busy} onClick={() => void remove(p)}>
                        {t('events.peopleList.removeYes')}
                      </Button>
                      <Button size="sm" variant="outline" onClick={() => setConfirming(null)}>
                        {t('events.peopleList.removeKeep')}
                      </Button>
                    </div>
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}

        <div aria-live="polite" className="empty:hidden">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          {status && <p className="text-sm text-muted-foreground">{status}</p>}
        </div>

        {/* signing people up, until it starts (as joining it themselves) */}
        {event.can_change && (
          <AddToEvent
            slug={slug}
            token={token}
            bookingId={event.id}
            onAdded={(name) => {
              setError(null)
              setStatus(t('events.added', { name }))
              people.reload()
              onChanged()
            }}
          />
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('events.peopleList.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
