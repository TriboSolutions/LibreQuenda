import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import type { Booking } from '@/lib/api'
import { formatBookingDay, formatTime } from '@/lib/time'

type Props = {
  booking: Booking | null
  timeZone: string
  today: string
  onClose: () => void
  onConfirm: (booking: Booking) => Promise<string | null>
}

/** A participant leaves someone else's booking; the booking itself stays. */
export function LeaveDialog({ booking, timeZone, today, onClose, onConfirm }: Props) {
  const { t } = useTranslation('invites')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!booking) return <Dialog open={false} />

  const confirm = async () => {
    setSubmitting(true)
    const message = await onConfirm(booking)
    setSubmitting(false)
    if (message) setError(message)
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('leaveDialog.title')}</DialogTitle>
          <DialogDescription className="num">
            {t('leaveDialog.description', {
              facility: booking.facility_name,
              day: formatBookingDay(booking.starts_at, timeZone, today),
              time: formatTime(booking.starts_at, timeZone),
            })}
          </DialogDescription>
        </DialogHeader>
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('leaveDialog.keep')}
          </Button>
          <Button variant="destructive" onClick={() => void confirm()} disabled={submitting}>
            {submitting ? t('leaveDialog.confirming') : t('leaveDialog.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
