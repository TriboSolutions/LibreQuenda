import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { CalendarClock, Repeat, Sparkles } from 'lucide-react'
import { Mark } from '@/components/Mark'
import { useAsync } from '@/hooks/useAsync'
import { useMe } from '@/hooks/useMe'
import { useTags } from '@/hooks/useTags'
import { api, type Recommendation } from '@/lib/api'
import { formatTime } from '@/lib/time'

// facilities are in Galicia today, like the rest of the home page
const TIME_ZONE = 'Europe/Madrid'

const ICONS = { usual_today: Repeat, favourite: CalendarClock, similar: Sparkles }

/**
 * "For you" on the home page, signed in: their usual turn when it falls today
 * and is free, the facilities they book most with free turns today, and some
 * alike to theirs they have not tried. Nothing shows without bookings.
 */
export function ForYou() {
  const { t } = useTranslation('home')
  const auth = useAuth()
  const token = auth.user?.access_token
  const { me } = useMe()
  // GET /me (MeProvider) first creates the user's row
  const recommendations = useAsync(token && me ? () => api.recommendations(token) : null, `recommendations:${me?.id}`)
  const items = recommendations.data?.recommendations ?? []
  if (items.length === 0) return null

  return (
    <section aria-labelledby="for-you-title" className="mt-6">
      <h2 id="for-you-title" className="text-lg font-semibold">
        {t('forYou.title')}
      </h2>
      <p className="text-sm text-muted-foreground">{t('forYou.help')}</p>
      <ul className="mt-3 flex snap-x gap-3 overflow-x-auto pb-2">
        {items.map((item) => (
          <ForYouCard key={item.facility.id} item={item} />
        ))}
      </ul>
    </section>
  )
}

function ForYouCard({ item }: { item: Recommendation }) {
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
    <li className="relative flex w-72 shrink-0 snap-start gap-3 rounded-lg border bg-card p-3 focus-within:ring-2 focus-within:ring-ring hover:border-primary/60">
      <span className="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-md bg-muted">
        {f.image_url ? <img src={f.image_url} alt="" className="size-full object-cover" /> : <Mark className="size-8" />}
      </span>
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
