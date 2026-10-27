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

export function CancelDialog({ booking, timeZone, today, onClose, onConfirm }: Props) {
  const { t } = useTranslation('bookings')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!booking) return <Dialog open={false} />

  const close = () => {
    setError(null)
    onClose()
  }
  const confirm = async () => {
    setSubmitting(true)
    const message = await onConfirm(booking)
    setSubmitting(false)
    if (message) setError(message)
  }

  return (
    <Dialog open onOpenChange={(open) => !open && close()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('cancelDialog.title')}</DialogTitle>
          <DialogDescription className="num">
            {t('cancelDialog.description', {
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
          <Button variant="outline" onClick={close}>
            {t('cancelDialog.keep')}
          </Button>
          <Button variant="destructive" onClick={() => void confirm()} disabled={submitting}>
            {submitting ? t('cancelDialog.confirming') : t('cancelDialog.confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
