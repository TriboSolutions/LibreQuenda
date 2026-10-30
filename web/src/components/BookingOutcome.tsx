import { useTranslation } from 'react-i18next'
import type { Booking } from '@/lib/api'
import { cn } from '@/lib/utils'

const tone = {
  neutral: 'bg-pill-neutral-bg text-pill-neutral-ink',
  positive: 'bg-pill-positive-bg text-pill-positive-ink',
  negative: 'bg-pill-negative-bg text-pill-negative-ink',
}

/** How a past booking ended: cancelled, or whether anyone checked in (reader, code, door QR or staff). */
export function BookingOutcome({ booking }: { booking: Booking }) {
  const { t } = useTranslation('bookings')
  const [label, color] =
    booking.status === 'cancelled'
      ? [t('status.cancelled'), tone.neutral]
      : booking.attendance === 'attended'
        ? [t('status.attended'), tone.positive]
        : booking.attendance === 'no_show'
          ? [t('status.noShow'), tone.negative]
          : [t('status.ended'), tone.neutral] // the facility took no check-ins then: nobody could check in
  return <span className={cn('shrink-0 rounded-full px-2 py-0.5 text-xs font-medium', color)}>{label}</span>
}
