import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { CalendarClock, ChevronDown, EyeOff, Repeat, Sparkles, Users } from 'lucide-react'
import { RatingBadge } from '@/components/Stars'
import { Mark } from '@/components/Mark'
import { Button } from '@/components/ui/button'
import { useAsync } from '@/hooks/useAsync'
import { useMe } from '@/hooks/useMe'
import { useTags } from '@/hooks/useTags'
import { api, type Recommendation } from '@/lib/api'
import { formatBookingDay, formatTime, localDate } from '@/lib/time'

// facilities are in Galicia today, like the rest of the home page
const TIME_ZONE = 'Europe/Madrid'

const ICONS = { usual_today: Repeat, favourite: CalendarClock, similar: Sparkles, public_booking: Users }

// one row on a desktop; the rest on demand, so the list and the map keep their room
const FIRST = 3

/**
 * "For you" on the home page, signed in: their usual turn when it falls today
 * and is free, the facilities they book most with free turns today, some alike
 * to theirs they have not tried, and other people's public bookings to join
 * there or near them. near: the home page's «near me», when it is on.
 */
export function ForYou({ near }: { near?: { latitude: number; longitude: number } | null }) {
  const { t } = useTranslation('home')
  const auth = useAuth()
  const token = auth.user?.access_token
  const { me, reload } = useMe()
  const [hiding, setHiding] = useState(false)
  // GET /me (MeProvider) first creates the user's row
  const on = !!me?.show_recommendations
  const nearKey = near ? `${near.latitude.toFixed(3)},${near.longitude.toFixed(3)}` : ''
  const recommendations = useAsync(token && me && on ? () => api.recommendations(token, near) : null, `recommendations:${me?.id}:${on}:${nearKey}`)
  const items = recommendations.data?.recommendations ?? []
  const [all, setAll] = useState(false)
  if (!on || items.length === 0) return null

  const hide = async () => {
    if (!token) return
    setHiding(true)
    try {
      await api.setShowRecommendations(token, false)
      reload()
    } finally {
      setHiding(false)
    }
  }
  const shown = all ? items : items.slice(0, FIRST)

  return (
    <section aria-labelledby="for-you-title" className="mt-6">
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1">
        <div>
          <h2 id="for-you-title" className="text-lg font-semibold">
            {t('forYou.title')}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t('forYou.help')}{' '}
            <Link to="/perfil" className="underline underline-offset-4">
              {t('forYou.settings')}
            </Link>
          </p>
        </div>
        <Button variant="ghost" size="sm" disabled={hiding} onClick={() => void hide()}>
          <EyeOff aria-hidden />
          {t('forYou.hide')}
        </Button>
      </div>
      <ul id="for-you-list" className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {shown.map((item) => (
          <ForYouCard key={item.booking?.id ?? item.facility.id} item={item} />
        ))}
      </ul>
      {items.length > FIRST && (
        <Button variant="ghost" size="sm" className="mt-2" aria-expanded={all} aria-controls="for-you-list" onClick={() => setAll((v) => !v)}>
          <ChevronDown className={all ? 'rotate-180' : ''} aria-hidden />
          {all ? t('forYou.less') : t('forYou.more', { count: items.length - FIRST })}
        </Button>
      )}
    </section>
  )
}

function ForYouCard({ item }: { item: Recommendation }) {
  if (item.kind === 'public_booking' && item.booking) return <PublicBookingCard item={item} booking={item.booking} />
  return <SuggestionCard item={item} />
}

function SuggestionCard({ item }: { item: Recommendation }) {
  const { t } = useTranslation(['home', 'common'])
  const catalogue = useTags()
  const f = item.facility
  const Icon = ICONS[item.kind]
  const time = formatTime(item.starts_at, TIME_ZONE)
  const why =
    item.kind === 'usual_today'
      ? t('forYou.usual', { weekday: t(`common:date.weekdays.${(item.weekday ?? 0) % 7}`), time, count: item.times ?? 0 })
      : item.kind === 'favourite'
        ? t('forYou.favourite', { count: item.times ?? 0 })
        : t('forYou.similar', { facility: item.because, tags: (item.shared_tags ?? []).map(catalogue.label).join(', ') })

  return (
    <li className="relative flex min-w-0 gap-3 rounded-lg border bg-card p-3 focus-within:ring-2 focus-within:ring-ring hover:border-primary/60">
      <Thumb url={f.image_url} />
      <div className="grid min-w-0 content-start gap-1">
        <h3 className="leading-5 font-semibold">
          {/* stretched over the card */}
          <Link to={`/${f.council.slug}/${f.slug}`} className="outline-none after:absolute after:inset-0 after:rounded-lg">
            {f.name}
          </Link>
        </h3>
        <p className="flex items-start gap-1.5 text-sm text-muted-foreground">
          <Icon className="mt-0.5 size-3.5 shrink-0" aria-hidden />
          <span>{why}</span>
        </p>
        <p className="num text-sm font-medium text-pill-positive-ink">{t('forYou.freeAt', { time })}</p>
      </div>
    </li>
  )
}

/** Someone's public booking with spots left: join it here, no invite link needed. */
function PublicBookingCard({ item, booking }: { item: Recommendation; booking: NonNullable<Recommendation['booking']> }) {
  const { t } = useTranslation(['home', 'common'])
  const auth = useAuth()
  const [busy, setBusy] = useState(false)
  const [joined, setJoined] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [today] = useState(() => localDate(new Date().toISOString(), TIME_ZONE))
  const f = item.facility
  const when = `${formatBookingDay(booking.starts_at, TIME_ZONE, today)}, ${formatTime(booking.starts_at, TIME_ZONE)}–${formatTime(booking.ends_at, TIME_ZONE)}`

  const join = async () => {
    const token = auth.user?.access_token
    if (!token) return
    setBusy(true)
    setError(null)
    try {
      await api.joinPublic(token, booking.id)
      setJoined(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <li className="flex min-w-0 gap-3 rounded-lg border bg-card p-3">
      <Thumb url={f.image_url} />
      <div className="grid min-w-0 flex-1 content-start gap-1">
        <h3 className="leading-5 font-semibold">
          <Link to={`/${f.council.slug}/${f.slug}`} className="underline-offset-4 hover:underline">
            {f.name}
          </Link>
        </h3>
        <p className="flex items-start gap-1.5 text-sm text-muted-foreground">
          <Users className="mt-0.5 size-3.5 shrink-0" aria-hidden />
          <span>
            {booking.owner_first_name ? t('forYou.publicBy', { name: booking.owner_first_name }) : t('forYou.public')}
            <RatingBadge rating={booking.owner_rating} className="ml-1.5 align-top" />
          </span>
        </p>
        <p className="num text-sm">{when}</p>
        {booking.description && <p className="whitespace-pre-wrap break-words text-sm">{booking.description}</p>}
        <div aria-live="polite" className="grid gap-1">
          {joined ? (
            <p className="text-sm font-medium text-pill-positive-ink">
              {t('forYou.joined')}{' '}
              <Link to="/reservas" className="underline underline-offset-4">
                {t('forYou.seeBookings')}
              </Link>
            </p>
          ) : (
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
              <Button size="sm" disabled={busy} onClick={() => void join()}>
                {busy ? t('forYou.joining') : t('forYou.join')}
              </Button>
              <span className="num text-sm text-muted-foreground">{t('forYou.spots', { count: booking.spots_left })}</span>
            </div>
          )}
          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
      </div>
    </li>
  )
}

function Thumb({ url }: { url: string }) {
  return (
    <span className="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-md bg-muted">
      {url ? <img src={url} alt="" className="size-full object-cover" /> : <Mark className="size-8" />}
    </span>
  )
}
