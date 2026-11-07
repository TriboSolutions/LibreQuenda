import { noticeKey } from '@/lib/notices'
import { AlarmClock, BadgeCheck, Bell, CalendarCheck, CalendarClock, CalendarX, House, Megaphone, ShieldCheck, Star, UserPlus, Vote, Wrench } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useNotifications } from '@/hooks/useNotifications'
import type { AppNotification, NotificationType } from '@/lib/api'
import { formatBookingDay, formatDateTime, formatTime, localDate } from '@/lib/time'

// Notifications are about Galician councils
const TIME_ZONE = 'Europe/Madrid'

const icons: Record<NotificationType, typeof Bell> = {
  booking_confirmed: CalendarCheck,
  booking_reminder: AlarmClock,
  booking_changed: CalendarClock,
  booking_joined: UserPlus,
  booking_cancelled: CalendarX,
  booking_cancelled_by_council: CalendarX,
  council_admin_granted: ShieldCheck,
  issue_reported: Wrench,
  issue_solved: Wrench,
  issue_rejected: Wrench,
  rating_requested: Star,
  resident_requested: House,
  resident_verified: BadgeCheck,
  resident_rejected: House,
  resident_revoked: House,
  participant_removed: CalendarX,
  council_notice: Megaphone,
  poll_opened: Vote,
  poll_closed: Vote,
}

/** Where a notice leads: the council's administration, the facility of a reported problem, the rating dialog, or My bookings. */
function target(n: AppNotification) {
  if (n.type === 'rating_requested' && n.booking_id) return `/reservas?valorar=${encodeURIComponent(n.booking_id)}`
  if (n.type === 'issue_reported') return `/${n.tenant_slug}/admin?vista=incidencias`
  if (n.type === 'resident_requested') return `/${n.tenant_slug}/admin?vista=vecinos`
  if (n.type === 'resident_verified' || n.type === 'resident_rejected' || n.type === 'resident_revoked') return `/${n.tenant_slug}`
  if (n.type === 'council_admin_granted') return `/${n.tenant_slug}/admin`
  if (n.type === 'poll_opened' || n.type === 'poll_closed') return `/enquisas?enquisa=${encodeURIComponent(n.poll_id)}`
  if (n.type === 'council_notice') return n.facility_slug ? `/${n.tenant_slug}/${n.facility_slug}` : `/${n.tenant_slug}`
  if ((n.type === 'issue_solved' || n.type === 'issue_rejected') && n.facility_slug) return `/${n.tenant_slug}/${n.facility_slug}`
  return '/reservas'
}

/** The bell in the top bar: polled notices, marked read when opened. */
export function NotificationBell() {
  const { t } = useTranslation('notifications')
  const navigate = useNavigate()
  const { data, error, markAllRead } = useNotifications()
  // what was unread when the menu opened keeps its highlight while it stays open
  const [highlighted, setHighlighted] = useState<Set<string>>(new Set())
  const [today] = useState(() => localDate(new Date().toISOString(), TIME_ZONE))
  const unread = data?.unread_count ?? 0

  const onOpenChange = (open: boolean) => {
    if (!open || !data) return
    setHighlighted(new Set(data.notifications.filter((n) => !n.read).map((n) => n.id)))
    if (unread > 0) void markAllRead()
  }

  return (
    <DropdownMenu onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="relative" aria-label={unread > 0 ? t('open', { count: unread }) : t('title')}>
          <Bell aria-hidden />
          {unread > 0 && (
            <span aria-hidden="true" className="num absolute top-1 right-1 flex min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] leading-4 font-semibold text-background">
              {unread > 9 ? '9+' : unread}
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[min(360px,calc(100vw-32px))]">
        <DropdownMenuLabel>{t('title')}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {error && !data ? (
          <p className="px-2 py-3 text-sm text-muted-foreground">{t('loadError')}</p>
        ) : !data || data.notifications.length === 0 ? (
          <p className="px-2 py-3 text-sm text-muted-foreground">{t('empty')}</p>
        ) : (
          <div className="max-h-[60vh] overflow-y-auto">
            {data.notifications.map((n) => (
              <NotificationItem
                key={n.id}
                notification={n}
                today={today}
                highlighted={highlighted.has(n.id)}
                onSelect={() => navigate(target(n))}
              />
            ))}
          </div>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => navigate('/reservas')} className="justify-center font-medium">
          {t('seeBookings')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

type ItemProps = { notification: AppNotification; today: string; highlighted: boolean; onSelect: () => void }

function NotificationItem({ notification: n, today, highlighted, onSelect }: ItemProps) {
  const { t } = useTranslation('notifications')
  const Icon = icons[n.type] ?? Bell
  const sorry = n.type === 'booking_cancelled_by_council'
  return (
    <DropdownMenuItem onSelect={onSelect} className={`items-start gap-3 py-2.5 ${highlighted ? 'bg-accent/60' : ''}`}>
      <Icon aria-hidden className={`mt-0.5 shrink-0 ${sorry ? 'text-destructive' : 'text-muted-foreground'}`} />
      <span className="min-w-0 whitespace-normal">
        <span className={`block text-sm ${highlighted ? 'font-medium' : ''}`}>
          {t(noticeKey(n), {
            name: n.name || t('someone'),
            facility: n.facility_name,
            day: n.starts_at ? formatBookingDay(n.starts_at, TIME_ZONE, today) : '',
            time: n.starts_at ? formatTime(n.starts_at, TIME_ZONE) : '',
            from: n.closed_from ? formatDateTime(n.closed_from, TIME_ZONE) : '',
            until: n.closed_until ? formatDateTime(n.closed_until, TIME_ZONE) : '',
            council: n.tenant_name,
            title: n.title,
          })}
        </span>
        {n.reason && (
          <span className="mt-0.5 block text-sm text-muted-foreground">
            {t(n.type === 'issue_solved' || n.type === 'issue_rejected' ? 'councilNote' : 'reason', { reason: n.reason })}
          </span>
        )}
        <span className="num mt-0.5 block text-xs text-muted-foreground">{formatDateTime(n.created_at, TIME_ZONE)}</span>
      </span>
    </DropdownMenuItem>
  )
}
