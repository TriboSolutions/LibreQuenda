import type { AppNotification } from '@/lib/api'

/**
 * The words for a notice: a cancellation says its cause its own way (as the email
 * and the app do, cronJobs/notificationEmails.go cancellationVariant).
 */
export function noticeKey(n: AppNotification): string {
  if (n.type === 'booking_cancelled' && n.cause === 'owner') return 'type.booking_cancelled_by_owner'
  if (n.type === 'booking_cancelled' && n.cause === 'account_deleted') return 'type.booking_cancelled_account_deleted'
  if (n.type === 'booking_cancelled_by_council') {
    if (n.closed_from && n.closed_until) return 'type.booking_cancelled_by_closure'
    if (n.cause === 'resident_revoked') return 'type.booking_cancelled_resident_revoked'
    if (n.cause === 'resident_revoked_left') return 'type.booking_left_resident_revoked'
    if (n.title) return 'type.event_cancelled'
    if (n.by_staff) return 'type.booking_cancelled_by_staff'
  }
  if (n.type === 'rating_requested' && n.is_public) return 'type.rating_requested_public'
  return `type.${n.type}`
}
