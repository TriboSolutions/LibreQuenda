import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { useAuth } from 'react-oidc-context'
import { Button } from '@/components/ui/button'
import { RatingBadge } from '@/components/Stars'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { useAsync } from '@/hooks/useAsync'
import { api, type Facility, type PublicPerson, type PublicTurn } from '@/lib/api'
import { formatBookingDay, formatTime } from '@/lib/time'

export function PublicTurnDialog({ booking, facility, timeZone, today, now, onClose, onJoined, onLogin }: {
  booking: PublicTurn
  facility: Facility
  timeZone: string
  today: string
  now: string
  onClose: () => void
  onJoined: () => void
  onLogin: () => void
}) {
  const { t } = useTranslation(['bookings', 'home'])
  const auth = useAuth()
  const token = auth.user?.access_token
  const mine = useAsync(token ? () => api.myBookings(token) : null, token ?? 'anonymous')
  const [busy, setBusy] = useState(false)
  const [joined, setJoined] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [seeing, setSeeing] = useState(false) // who joined, in place of the rest (signed in only)
  const attending = joined || mine.data?.upcoming.some((b) => b.id === booking.id)
  const count = booking.people_count
  const full = count >= booking.capacity
  const started = Date.parse(booking.starts_at) <= Date.parse(now)
  const event = booking.kind === 'event' // the council's: a title, no owner

  const join = async () => {
    if (!token || busy) return
    setBusy(true)
    setError(null)
    try {
      await api.joinPublic(token, booking.id)
      setJoined(true)
      onJoined()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && !busy && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="break-words">{event ? booking.title : t('public.turnTitle', { facility: facility.name })}</DialogTitle>
          <DialogDescription>
            {event && <>{t('public.eventAt', { facility: facility.name })}<br /></>}
            {/* whose it is: who booked answers for it */}
            {!event && booking.owner?.name && <>{t('public.by', { name: booking.owner.name })}<br /></>}
            {formatBookingDay(booking.starts_at, timeZone, today)} · {formatTime(booking.starts_at, timeZone)}–{formatTime(booking.ends_at, timeZone)}
          </DialogDescription>
        </DialogHeader>
        {seeing ? (
          <section aria-labelledby="public-joined-title" className="grid gap-2">
            <h3 id="public-joined-title" className="font-medium">{t('public.peopleTitle')}</h3>
            <ul className="divide-y">
              {booking.owner && <PersonRow person={booking.owner} organiser />}
              {booking.joined.map((person, index) => <PersonRow key={index} person={person} />)}
            </ul>
            <Button variant="outline" className="justify-self-start" onClick={() => setSeeing(false)}>{t('public.back')}</Button>
          </section>
        ) : (
          <>
          <p className="whitespace-pre-wrap break-words">{booking.description || t(event ? 'public.noEventDescription' : 'public.noDescription')}</p>
          <section aria-labelledby="public-people-title" className="grid gap-2">
            <h3 id="public-people-title" className="font-medium">{t('public.people', { count: count + (joined ? 1 : 0), capacity: booking.capacity })}</h3>
            {/* who answers for it; who joined, signed in only, on a view of their own */}
            {booking.owner && <ul><PersonRow person={booking.owner} organiser /></ul>}
            {token && booking.joined.length > 0 && (
              <Button variant="outline" className="justify-self-start" onClick={() => setSeeing(true)}>{t('public.seePeople')}</Button>
            )}
          </section>
          {facility.usage_rules && <p className="whitespace-pre-wrap text-sm text-muted-foreground">{facility.usage_rules}</p>}
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          </>
        )}
        <DialogFooter>
          {attending ? (
            <Button asChild><Link to="/reservas">{t('home:forYou.seeBookings')}</Link></Button>
          ) : started ? (
            <p className="text-sm text-muted-foreground">{t(event ? 'public.eventStarted' : 'public.started')}</p>
          ) : full ? (
            <p className="text-sm text-muted-foreground">{t('public.full')}</p>
          ) : !token ? (
            <Button onClick={onLogin}>{t('public.login')}</Button>
          ) : (
            <Button disabled={busy || mine.loading} onClick={() => void join()}>{t(busy ? 'home:forYou.joining' : 'home:forYou.join')}</Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function PersonRow({ person, organiser }: { person: PublicPerson; organiser?: boolean }) {
  const { t } = useTranslation(['bookings', 'common'])
  return (
    <li className="flex items-center justify-between gap-3 py-2">
      <span className="min-w-0 break-words">
        <span className={organiser ? 'font-medium' : undefined}>{person.name || t('public.person')}</span>
        {organiser && <span className="text-muted-foreground"> · {t('public.organiser')}</span>}
      </span>
      {person.rating ? <RatingBadge rating={person.rating} className="shrink-0" /> : <span className="shrink-0 text-sm text-muted-foreground">{t('common:rating.none')}</span>}
    </li>
  )
}
