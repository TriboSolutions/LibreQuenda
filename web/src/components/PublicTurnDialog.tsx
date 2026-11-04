import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { useAuth } from 'react-oidc-context'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { useAsync } from '@/hooks/useAsync'
import { api, type Facility, type PublicTurn } from '@/lib/api'
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
  const attending = joined || mine.data?.upcoming.some((b) => b.id === booking.id)
  const count = booking.people.length
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
            {formatBookingDay(booking.starts_at, timeZone, today)} · {formatTime(booking.starts_at, timeZone)}–{formatTime(booking.ends_at, timeZone)}
          </DialogDescription>
        </DialogHeader>
        <p className="whitespace-pre-wrap break-words">{booking.description || t(event ? 'public.noEventDescription' : 'public.noDescription')}</p>
        <section aria-labelledby="public-people-title" className="grid gap-2">
          <h3 id="public-people-title" className="font-medium">{t('public.people', { count: count + (joined ? 1 : 0), capacity: booking.capacity })}</h3>
          <ul className="flex flex-wrap gap-2">
            {booking.people.map((name, index) => <li key={index} className="rounded-md bg-muted px-2 py-1 text-sm">{name || t('public.person')}</li>)}
            {joined && <li className="rounded-md bg-muted px-2 py-1 text-sm">{t('public.you')}</li>}
          </ul>
        </section>
        {facility.usage_rules && <p className="whitespace-pre-wrap text-sm text-muted-foreground">{facility.usage_rules}</p>}
        {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
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
