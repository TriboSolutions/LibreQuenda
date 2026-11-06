import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { ParticipantList } from '@/components/ParticipantList'
import { PeopleList } from '@/components/PeopleList'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type Booking } from '@/lib/api'
import { formatBookingDay, formatTime } from '@/lib/time'

type Props = {
  booking: Booking | null
  timeZone: string
  today: string
  onClose: () => void
  onChanged: () => void // someone was taken out: the list refreshes its counts
}

/**
 * Who is coming to a booking, from its card. Its owner can take someone out (they are
 * told, and cannot join it again); anyone else in it sees the names.
 */
export function PeopleDialog({ booking, timeZone, today, onClose, onChanged }: Props) {
  if (!booking) return <Dialog open={false} />
  return <PeopleDialogOpen booking={booking} timeZone={timeZone} today={today} onClose={onClose} onChanged={onChanged} />
}

function PeopleDialogOpen({ booking, timeZone, today, onClose, onChanged }: Props & { booking: Booking }) {
  const { t } = useTranslation('invites')
  const auth = useAuth()
  const token = auth.user?.access_token
  const owner = booking.role === 'owner' && booking.kind !== 'event'
  const invite = useAsync(owner && token ? () => api.invites.get(token, booking.id) : null, `people:${booking.id}`)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const remove = async (participantId: string) => {
    if (!token) return
    setBusy(true)
    setError(null)
    try {
      await api.invites.removeParticipant(token, booking.id, participantId)
      invite.reload()
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('dialog.error'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('people.title', { facility: booking.facility_name })}</DialogTitle>
          <DialogDescription className="num">
            {formatBookingDay(booking.starts_at, timeZone, today)}, {formatTime(booking.starts_at, timeZone)}–{formatTime(booking.ends_at, timeZone)}
          </DialogDescription>
        </DialogHeader>
        {!owner ? (
          <PeopleList people={booking.people} />
        ) : invite.data ? (
          <ParticipantList participants={invite.data.participants} removable={invite.data.state !== 'closed'} busy={busy} onRemove={(id) => void remove(id)} />
        ) : invite.error ? (
          <p className="text-destructive">{invite.error.message}</p>
        ) : (
          <Skeleton className="h-24 w-full" />
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
            {t('dialog.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
