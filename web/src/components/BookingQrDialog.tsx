import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import QRCode from 'qrcode'
import { CircleCheck } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import type { Booking } from '@/lib/api'
import { formatBookingDay, formatTime } from '@/lib/time'

type Props = {
  booking: Booking | null
  timeZone: string
  today: string
  onClose: () => void
}

/** The entry QR of a booking, for the reader at the facility entrance. */
export function BookingQrDialog({ booking, timeZone, today, onClose }: Props) {
  const { t } = useTranslation('bookings')
  const [svg, setSvg] = useState<string | null>(null)
  const code = booking?.checkin_code ?? ''

  useEffect(() => {
    if (!code) return
    let cancelled = false
    // dark on white even in dark mode: readers expect the standard contrast
    QRCode.toString(code, { type: 'svg', errorCorrectionLevel: 'M', margin: 2, color: { dark: '#000000', light: '#ffffff' } })
      .then((markup) => !cancelled && setSvg(markup))
      .catch(() => !cancelled && setSvg(null))
    return () => {
      cancelled = true
    }
  }, [code])

  if (!booking) return <Dialog open={false} />

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{t('qr.title')}</DialogTitle>
          <DialogDescription className="num">
            {t('qr.description', {
              facility: booking.facility_name,
              day: formatBookingDay(booking.starts_at, timeZone, today),
              start: formatTime(booking.starts_at, timeZone),
              end: formatTime(booking.ends_at, timeZone),
            })}
          </DialogDescription>
        </DialogHeader>

        <div className="mx-auto w-full max-w-64 rounded-lg bg-white p-2 ring-1 ring-border">
          {svg ? (
            <img
              src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`}
              alt={t('qr.alt', { facility: booking.facility_name })}
              className="aspect-square w-full [image-rendering:pixelated]"
            />
          ) : (
            <Skeleton className="aspect-square w-full bg-muted" />
          )}
        </div>

        {booking.checked_in_at && (
          <p className="num flex items-center justify-center gap-1.5 rounded-md bg-pill-positive-bg px-3 py-2 text-pill-positive-ink">
            <CircleCheck className="size-4" aria-hidden />
            {t('qr.checkedIn', { time: formatTime(booking.checked_in_at, timeZone) })}
          </p>
        )}

        <div className="space-y-1.5 text-muted-foreground">
          <p>{t('qr.hint')}</p>
          <p>{t('qr.brightness')} {t('qr.email')}</p>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('qr.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
