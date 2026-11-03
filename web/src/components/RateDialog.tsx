import { CircleCheck } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { StarInput } from '@/components/Stars'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type BookingRating } from '@/lib/api'
import { formatBookingDay, formatDateTime, formatTime } from '@/lib/time'

type Props = {
  bookingId: string | null // null: closed
  timeZone: string
  today: string
  onClose: () => void
}

/** After a booking: stars for the facility and, if it was public, for each of the others with an account. */
export function RateDialog({ bookingId, timeZone, today, onClose }: Props) {
  const auth = useAuth()
  const token = auth.user?.access_token
  const rating = useAsync(token && bookingId ? () => api.ratings.get(token, bookingId) : null, `rating:${bookingId}`)
  if (!bookingId) return <Dialog open={false} />
  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        {rating.data ? (
          <RateForm key={rating.data.booking_id} rating={rating.data} token={token} timeZone={timeZone} today={today} onClose={onClose} />
        ) : (
          <Loading error={rating.error} onClose={onClose} />
        )}
      </DialogContent>
    </Dialog>
  )
}

function Loading({ error, onClose }: { error: unknown; onClose: () => void }) {
  const { t } = useTranslation('bookings')
  return (
    <>
      <DialogHeader>
        <DialogTitle>{t('rating.title')}</DialogTitle>
        <DialogDescription className="sr-only">{t('rating.loading')}</DialogDescription>
      </DialogHeader>
      {error ? (
        <Alert variant="destructive">
          <AlertDescription>{error instanceof Error ? error.message : t('rating.loadError')}</AlertDescription>
        </Alert>
      ) : (
        <div className="grid gap-3" aria-busy="true" aria-label={t('rating.loading')}>
          <Skeleton className="h-6 w-1/2" />
          <Skeleton className="h-11 w-full" />
        </div>
      )}
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>
          {t('rating.close')}
        </Button>
      </DialogFooter>
    </>
  )
}

type FormProps = { rating: BookingRating; token: string | undefined; timeZone: string; today: string; onClose: () => void }

function RateForm({ rating, token, timeZone, today, onClose }: FormProps) {
  const { t } = useTranslation(['bookings', 'invites'])
  const [facilityStars, setFacilityStars] = useState(rating.facility_stars)
  const [people, setPeople] = useState(() => Object.fromEntries(rating.people.map((p) => [p.user_id, p.stars])))
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const when = t('rating.when', {
    facility: rating.facility_name,
    day: formatBookingDay(rating.starts_at, timeZone, today),
    time: formatTime(rating.starts_at, timeZone),
  })
  const anything = facilityStars > 0 || Object.values(people).some((stars) => stars > 0)

  const save = async () => {
    if (!token) return
    setBusy(true)
    setError(null)
    try {
      await api.ratings.save(token, rating.booking_id, {
        facility_stars: facilityStars,
        people: Object.entries(people).map(([user_id, stars]) => ({ user_id, stars })),
      })
      setSaved(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('rating.saveError'))
    } finally {
      setBusy(false)
    }
  }

  if (saved) {
    return (
      <>
        <DialogHeader>
          <DialogTitle>{t('rating.thanksTitle')}</DialogTitle>
          <DialogDescription>{t('rating.thanks')}</DialogDescription>
        </DialogHeader>
        <CircleCheck aria-hidden className="mx-auto size-10 text-pill-positive-ink" />
        <DialogFooter>
          <Button onClick={onClose}>{t('rating.close')}</Button>
        </DialogFooter>
      </>
    )
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t('rating.title')}</DialogTitle>
        <DialogDescription className="num">{when}</DialogDescription>
      </DialogHeader>
      {!rating.can_rate ? (
        <p className="text-sm text-muted-foreground">{t('rating.closed')}</p>
      ) : (
        <div className="grid gap-5">
          <StarInput label={t('rating.facility', { facility: rating.facility_name })} value={facilityStars} onChange={setFacilityStars} disabled={busy} />
          {rating.people.length > 0 && (
            <section aria-labelledby="rate-people" className="grid gap-4 border-t pt-4">
              <div>
                <h3 id="rate-people" className="font-medium">
                  {t('rating.peopleTitle')}
                </h3>
                <p className="text-sm text-muted-foreground">{t('rating.peopleHelp')}</p>
              </div>
              {rating.people.map((person) => (
                <StarInput
                  key={person.user_id}
                  label={`${person.name || t('invites:people.unnamed')}${person.is_owner ? ` · ${t('invites:people.organiser')}` : ''}`}
                  value={people[person.user_id] ?? 0}
                  onChange={(stars) => setPeople((current) => ({ ...current, [person.user_id]: stars }))}
                  disabled={busy}
                />
              ))}
            </section>
          )}
          <p className="num text-xs text-muted-foreground">{t('rating.private', { date: formatDateTime(rating.rate_until, timeZone) })}</p>
        </div>
      )}
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      <DialogFooter>
        <Button variant="outline" onClick={onClose} disabled={busy}>
          {t(rating.can_rate ? 'rating.later' : 'rating.close')}
        </Button>
        {rating.can_rate && (
          <Button onClick={() => void save()} disabled={busy || !anything}>
            {t(busy ? 'rating.saving' : 'rating.save')}
          </Button>
        )}
      </DialogFooter>
    </>
  )
}
