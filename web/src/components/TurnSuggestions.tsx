import { ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Button } from '@/components/ui/button'
import type { BusyLevel, Slot, Suggestions } from '@/lib/api'
import { formatTime } from '@/lib/time'

type Props = {
  suggestions: Suggestions
  tenantSlug: string
  sport: string
  date: string // the day shown, carried over to the other facilities
  timeZone: string
  full: boolean // no free turns left here on this day
  onPick: (slot: Slot) => void
}

const PILL: Record<BusyLevel, string> = {
  quiet: 'bg-pill-positive-bg text-pill-positive-ink',
  moderate: 'bg-pill-neutral-bg text-pill-neutral-ink',
  busy: 'bg-pill-accent-bg text-pill-accent-ink',
}

/**
 * Turns worth suggesting, by plain counting: the free turns that are usually
 * quietest here, and other facilities of the same sport with free turns that day.
 */
export function TurnSuggestions({ suggestions, tenantSlug, sport, date, timeZone, full, onPick }: Props) {
  const { t, i18n } = useTranslation('facility')
  const { quiet, alternatives } = suggestions
  if (quiet.length === 0 && alternatives.length === 0) return null
  const km = new Intl.NumberFormat(i18n.language, { maximumFractionDigits: 1 })

  return (
    <section aria-labelledby="suggest-title" className="grid gap-5 rounded-lg border bg-card p-5">
      <h2 id="suggest-title" className="text-lg font-semibold">
        {t('suggest.title')}
      </h2>

      {quiet.length > 0 && (
        <div className="grid gap-2">
          <h3 className="font-medium">{t('suggest.quietTitle')}</h3>
          <p className="text-muted-foreground">{t('suggest.quietHint')}</p>
          <ul className="grid gap-2">
            {quiet.map((slot) => (
              <li key={slot.starts_at}>
                <Button
                  variant="outline"
                  className="h-auto w-full flex-wrap justify-between gap-2 border-primary px-4 py-2.5 whitespace-normal"
                  onClick={() => onPick({ starts_at: slot.starts_at, ends_at: slot.ends_at, status: 'free' })}
                >
                  <span className="num text-primary">
                    {formatTime(slot.starts_at, timeZone)}–{formatTime(slot.ends_at, timeZone)}
                  </span>
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${PILL[slot.level]}`}>{t(`suggest.level.${slot.level}`)}</span>
                </Button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {alternatives.length > 0 && (
        <div className="grid gap-2">
          <h3 className="font-medium">{t('suggest.nearbyTitle')}</h3>
          <p className="text-muted-foreground">{t(full ? 'suggest.nearbyFull' : 'suggest.nearbyHint', { sport })}</p>
          <ul className="grid gap-2">
            {alternatives.map((alt) => (
              <li key={alt.slug}>
                <Link
                  to={`/${tenantSlug}/${alt.slug}?dia=${date}`}
                  className="flex items-center gap-3 rounded-md border px-4 py-3 transition-colors hover:bg-muted"
                >
                  <span className="grid min-w-0 flex-1 gap-0.5">
                    <span className="font-medium">{alt.name}</span>
                    <span className="num text-sm text-muted-foreground">
                      {[
                        alt.distance_km !== null && (alt.distance_km < 0.1 ? t('suggest.nextDoor') : t('suggest.distance', { km: km.format(alt.distance_km) })),
                        t('suggest.freeCount', { count: alt.free_slots }),
                        alt.slots
                          .map((s) => s.starts_at)
                          .sort()
                          .map((iso) => formatTime(iso, timeZone))
                          .join(', '),
                      ]
                        .filter(Boolean)
                        .join(' · ')}
                    </span>
                  </span>
                  <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                </Link>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}
