import { LoaderCircle, LocateFixed, MapPin, Search, Zap } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { HomeMap } from '@/components/HomeMap'
import { ForYou } from '@/components/ForYou'
import { Mark } from '@/components/Mark'
import { SiteFooter } from '@/components/SiteFooter'
import { TagFilters } from '@/components/TagFilters'
import { BookingKindFilter } from '@/components/BookingKindFilter'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { useAsync } from '@/hooks/useAsync'
import { usePageTitle } from '@/hooks/usePageTitle'
import { useTags } from '@/hooks/useTags'
import { api, type FacilityCard, type GeoPoint, type JoinableKind } from '@/lib/api'
import { inView, type MapView } from '@/lib/geo'
import { cn } from '@/lib/utils'

// Distances offered once the browser knows where the resident is; 0 = any distance
const RADII_KM = [0, 5, 10, 25, 50]
const ANY_COUNCIL = '*' // a select item needs a value, and no council slug is "*"

function normalise(text: string) {
  return text
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .toLowerCase()
}

type Position = { state: 'off' } | { state: 'locating' } | { state: 'denied' } | { state: 'on'; point: GeoPoint }

/**
 * "/": every facility of every council, as a list beside a map. The council is one more
 * filter, with the words, the tags and the distance; the facility is what people look for.
 */
export function HomePage() {
  const { t } = useTranslation(['councils', 'common', 'home'])
  const tenants = useAsync(() => api.tenants(), 'tenants')
  const catalogue = useTags()
  const [query, setQuery] = useState('')
  const [council, setCouncil] = useState('')
  const [tags, setTags] = useState<string[]>([])
  const [kinds, setKinds] = useState<JoinableKind[]>([])
  const [position, setPosition] = useState<Position>({ state: 'off' })
  const [radius, setRadius] = useState(0)
  // nearest first, within the radius, is the API's job: it gets where the person is
  const near = position.state === 'on' ? position.point : null
  const me = near
  const facilities = useAsync(
    () => api.facilities(near, radius),
    `facilities:${near ? `${near.latitude.toFixed(4)},${near.longitude.toFixed(4)}` : ''}:${radius}`,
  )
  const [highlighted, setHighlighted] = useState<string | null>(null)
  const [view, setView] = useState<MapView | null>(null)
  const [fitKey, setFitKey] = useState(0)
  usePageTitle(t('councils:title'))

  const all = facilities.data?.facilities ?? []
  const councils = tenants.data?.tenants ?? []

  const locate = () => {
    if (position.state === 'on') {
      setPosition({ state: 'off' })
      setRadius(0)
      return
    }
    if (!('geolocation' in navigator)) {
      setPosition({ state: 'denied' })
      return
    }
    setPosition({ state: 'locating' })
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => setPosition({ state: 'on', point: { latitude: coords.latitude, longitude: coords.longitude } }),
      () => setPosition({ state: 'denied' }),
      { maximumAge: 10 * 60 * 1000, timeout: 15000 },
    )
  }

  // every word has to appear in the facility's name, its council, its sport or its tags
  const words = normalise(query).split(/\s+/).filter(Boolean)
  // in the API's order; the words, council and tags only leave some out
  const shown: { facility: FacilityCard; km: number | null }[] = all.flatMap((facility) => {
    if (council && facility.council.slug !== council) return []
    if (!tags.every((tag) => facility.tags.includes(tag))) return []
    if (kinds.length > 0 && !kinds.some((kind) => facility.joinable?.includes(kind))) return []
    const text = normalise([facility.name, facility.council.name, facility.sport, ...facility.tags.map(catalogue.label)].join(' '))
    if (!words.every((word) => text.includes(word))) return []
    return [{ facility, km: facility.distance_km ?? null }]
  })
  // the list follows the map: zoomed or panned in, only what is in view (and then not the
  // facilities without a pin); while the map frames every match, all of them
  const pinnedOf = (f: FacilityCard) => (f.latitude !== null && f.longitude !== null ? { latitude: f.latitude, longitude: f.longitude } : null)
  const zoomedIn =
    view !== null &&
    shown.some(({ facility }) => {
      const pin = pinnedOf(facility)
      return pin !== null && !inView(view, pin)
    })
  const listed = zoomedIn
    ? shown.filter(({ facility }) => {
        const pin = pinnedOf(facility)
        return pin !== null && inView(view, pin)
      })
    : shown
  const filtering = query !== '' || council !== '' || tags.length > 0 || kinds.length > 0 || radius > 0

  const clear = () => {
    setQuery('')
    setCouncil('')
    setTags([])
    setKinds([])
    setRadius(0)
  }

  return (
    <div className="min-h-screen">
      {/* on a desktop, the header and the page fill exactly one screen: the list scrolls inside, the page only to reach the footer */}
      <div className="flex min-h-svh flex-col lg:h-svh">
        <TopBar tenantSlug={null} />

        <main id="main" tabIndex={-1} className="mx-auto flex w-full max-w-[1440px] flex-1 flex-col px-4 pt-6 pb-6 sm:px-6 lg:min-h-0">
          <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('councils:title')}</h1>
          {facilities.data ? (
            <p className="mt-1 text-muted-foreground">{t('councils:subtitle', { count: all.length, councils: councils.length })}</p>
          ) : (
            <Skeleton className="mt-2 h-5 w-80 max-w-full" />
          )}

          <ForYou near={me} />

          <div role="search" className="mt-6 flex flex-wrap items-center gap-2" aria-label={t('councils:filters')}>
            <div className="relative w-full sm:w-72">
              <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
              <Input type="search" value={query} onChange={(e) => setQuery(e.target.value)} placeholder={t('councils:search')} aria-label={t('councils:search')} className="pl-9" />
            </div>
            {councils.length > 1 && (
              <Select value={council || ANY_COUNCIL} onValueChange={(value) => setCouncil(value === ANY_COUNCIL ? '' : value)}>
                <SelectTrigger className={cn('w-full sm:w-52', council && 'border-primary text-primary')} aria-label={t('councils:council')}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ANY_COUNCIL}>{t('councils:allCouncils')}</SelectItem>
                  {councils.map((c) => (
                    <SelectItem key={c.slug} value={c.slug}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            <TagFilters groups={catalogue.grouped(all.flatMap((f) => f.tags))} chosen={tags} onChange={setTags} clearAll={false} />
            <BookingKindFilter chosen={kinds} onChange={setKinds} />
            <Button variant={me ? 'default' : 'outline'} aria-pressed={me !== null} disabled={position.state === 'locating'} onClick={locate}>
              {position.state === 'locating' ? <LoaderCircle className="animate-spin" /> : <LocateFixed />}
              {position.state === 'locating' ? t('councils:locating') : t('councils:nearMe')}
            </Button>
            {me && (
              <ToggleGroup type="single" variant="outline" value={String(radius)} onValueChange={(value) => value && setRadius(Number(value))} aria-label={t('councils:radius')}>
                {RADII_KM.map((km) => (
                  <ToggleGroupItem key={km} value={String(km)} className="px-3">
                    {km === 0 ? t('councils:anyDistance') : t('councils:withinKm', { km })}
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
            )}
            {filtering && (
              <Button variant="ghost" onClick={clear}>
                {t('home:tags.clear')}
              </Button>
            )}
          </div>
          <div aria-live="polite" className="empty:hidden">
            {position.state === 'denied' && <p className="mt-3 text-muted-foreground">{t('councils:locationDenied')}</p>}
          </div>

          {/* the list on the left, the map on the right; on a phone the map goes first, shorter */}
          <div className="mt-4 grid gap-6 lg:min-h-0 lg:flex-1 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.15fr)]">
            <section aria-labelledby="facilities-title" className="order-2 flex flex-col lg:order-1 lg:min-h-0">
              <h2 id="facilities-title" className="sr-only">
                {t('councils:listTitle')}
              </h2>
              {facilities.data && (
                <div className="mb-3 flex min-h-8 flex-wrap items-center gap-x-3 gap-y-1">
                  <p aria-live="polite" className="text-sm text-muted-foreground">
                    {zoomedIn ? t('councils:showingInView', { count: listed.length }) : t('councils:showing', { count: shown.length })}
                  </p>
                  {zoomedIn && (
                    <Button variant="link" size="sm" className="h-auto p-0" onClick={() => setFitKey((k) => k + 1)}>
                      {t('councils:seeAll', { count: shown.length })}
                    </Button>
                  )}
                </div>
              )}
              {facilities.error ? (
                <div className="flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
                  <p>{t('councils:loadError')}</p>
                  <Button variant="ghost" size="sm" onClick={facilities.reload}>
                    {t('common:retry')}
                  </Button>
                </div>
              ) : !facilities.data ? (
                <ul className="grid gap-3" aria-busy="true" aria-label={t('councils:loading')}>
                  {Array.from({ length: 5 }, (_, i) => (
                    <li key={i}>
                      <Skeleton className="h-24 w-full rounded-lg" />
                    </li>
                  ))}
                </ul>
              ) : all.length === 0 ? (
                <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('councils:empty')}</p>
              ) : shown.length === 0 ? (
                <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('councils:noResults')}</p>
              ) : listed.length === 0 ? (
                <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('councils:noneInView')}</p>
              ) : (
                <ul className="grid content-start gap-3 lg:min-h-0 lg:flex-1 lg:overflow-y-auto lg:pr-1">
                  {listed.map(({ facility, km }) => (
                    <li key={facility.id} onMouseEnter={() => setHighlighted(facility.id)} onMouseLeave={() => setHighlighted(null)}>
                      <FacilityRow facility={facility} km={km} active={facility.id === highlighted} labels={facility.tags.map(catalogue.label)} />
                    </li>
                  ))}
                </ul>
              )}
            </section>

            <div className="order-1 lg:order-2 lg:min-h-0">
              <HomeMap
                facilities={shown.map(({ facility }) => facility)}
                highlighted={highlighted}
                me={me}
                label={t('councils:mapLabel')}
                fitKey={fitKey}
                onViewChange={setView}
                className="h-72 lg:h-full"
              />
            </div>
          </div>
        </main>
      </div>
      <SiteFooter />
    </div>
  )
}

type RowProps = { facility: FacilityCard; km: number | null; active: boolean; labels: string[] }

/** One facility: photo, name (opens it), its council (opens the council), tags, free turns today and distance. */
function FacilityRow({ facility: f, km, active, labels }: RowProps) {
  const { t, i18n } = useTranslation('councils')
  const distance = km === null ? null : new Intl.NumberFormat(i18n.resolvedLanguage ?? 'gl', { maximumFractionDigits: km < 10 ? 1 : 0 }).format(km)
  return (
    <article
      className={cn(
        'relative flex gap-4 rounded-lg border bg-card p-3 transition-colors focus-within:ring-2 focus-within:ring-ring hover:border-primary/60',
        active && 'border-primary/60',
      )}
    >
      <span className="flex size-20 shrink-0 items-center justify-center overflow-hidden rounded-md bg-muted">
        {f.image_url ? <img src={f.image_url} alt="" className="size-full object-cover" /> : <Mark className="size-10" />}
      </span>
      <div className="grid min-w-0 flex-1 content-start gap-1">
        <h3 className="leading-5 font-semibold">
          {/* stretched over the card; the council link below sits above it */}
          <Link to={`/${f.council.slug}/${f.slug}`} className="outline-none after:absolute after:inset-0 after:rounded-lg">
            {f.name}
          </Link>
        </h3>
        <p className="flex min-w-0 items-center gap-1.5 text-sm text-muted-foreground">
          {f.council.logo_url && <img src={f.council.logo_url} alt="" className="size-4 shrink-0 object-contain" />}
          <Link to={`/${f.council.slug}`} className="relative z-10 truncate underline-offset-4 hover:text-foreground hover:underline">
            {f.council.name}
          </Link>
        </p>
        {labels.length > 0 && <p className="line-clamp-1 text-xs text-muted-foreground">{labels.join(' · ')}</p>}
        <p className="mt-1 flex flex-wrap gap-1.5">
          <span
            className={cn(
              'num inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
              f.free_today > 0 ? 'bg-pill-positive-bg text-pill-positive-ink' : 'bg-pill-neutral-bg text-pill-neutral-ink',
            )}
          >
            <Zap className="size-3" aria-hidden="true" />
            {t('card.freeTodayLong', { count: f.free_today })}
          </span>
          {distance !== null && (
            <span className="num inline-flex items-center gap-1 rounded-full bg-pill-accent-bg px-2 py-0.5 text-xs font-medium text-pill-accent-ink">
              <MapPin className="size-3" aria-hidden="true" />
              {t('card.distance', { km: distance })}
            </span>
          )}
        </p>
      </div>
    </article>
  )
}
