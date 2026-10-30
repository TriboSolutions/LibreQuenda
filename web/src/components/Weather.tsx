import { useTranslation } from 'react-i18next'
import {
  Cloud,
  CloudDrizzle,
  CloudFog,
  CloudHail,
  CloudLightning,
  CloudRain,
  CloudSnow,
  CloudSun,
  Cloudy,
  Droplets,
  Haze,
  Snowflake,
  Sun,
  TriangleAlert,
  Wind,
} from 'lucide-react'
import type { DayForecast, WeatherPeriod } from '@/lib/api'
import { formatDateTime, formatTime } from '@/lib/time'

const SKY_ICONS: Record<string, typeof Sun> = {
  clear: Sun,
  partly_cloudy: CloudSun,
  cloudy: Cloud,
  overcast: Cloudy,
  fog: CloudFog,
  haze: Haze,
  drizzle: CloudDrizzle,
  rain: CloudRain,
  showers: CloudRain,
  storm: CloudLightning,
  snow: Snowflake,
  sleet: CloudSnow,
  hail: CloudHail,
}

/** The period a turn falls in, as MeteoGalicia splits the day: morning until 14:00, afternoon until 21:00. */
function periodOf(startsAt: string, timeZone: string): WeatherPeriod['period'] {
  const hour = Number(formatTime(startsAt, timeZone).slice(0, 2))
  return hour < 14 ? 'morning' : hour < 21 ? 'afternoon' : 'night'
}

function Attribution() {
  const { t } = useTranslation('facility')
  return (
    <p className="text-xs text-muted-foreground">
      {t('weather.source')}{' '}
      <a href="https://www.meteogalicia.gal" target="_blank" rel="noopener noreferrer" className="underline underline-offset-4">
        MeteoGalicia
      </a>
    </p>
  )
}

function PeriodLine({ p }: { p: WeatherPeriod }) {
  const { t } = useTranslation('facility')
  const Icon = SKY_ICONS[p.sky] ?? Cloud
  return (
    <>
      <Icon className="size-5 shrink-0" aria-hidden />
      <span>
        {p.sky ? t(`weather.sky.${p.sky}`) : t('weather.unknown')}
        {p.temperature !== null && <span className="num">{` · ${t('weather.temperature', { value: p.temperature })}`}</span>}
      </span>
      {p.rain_chance !== null && (
        <span className="num inline-flex items-center gap-1 text-muted-foreground">
          <Droplets className="size-3.5" aria-hidden />
          {t('weather.rainChance', { value: p.rain_chance })}
        </span>
      )}
      {p.wind && p.wind !== 'calm' && (
        <span className="inline-flex items-center gap-1 text-muted-foreground">
          <Wind className="size-3.5" aria-hidden />
          {t(`weather.wind.${p.wind}`)}
          {p.wind_from && ` ${t(`weather.from.${p.wind_from}`)}`}
        </span>
      )}
    </>
  )
}

/** The facility page's strip for the day shown: morning, afternoon and night, and any warning. */
export function WeatherStrip({ forecast }: { forecast: DayForecast }) {
  const { t } = useTranslation('facility')
  return (
    <section aria-labelledby="weather-title" className="grid gap-2 rounded-md border px-4 py-3">
      <h3 id="weather-title" className="text-sm font-medium">
        {t('weather.title')}
        {forecast.temp_min !== null && forecast.temp_max !== null && (
          <span className="num font-normal text-muted-foreground">{` · ${t('weather.range', { min: forecast.temp_min, max: forecast.temp_max })}`}</span>
        )}
      </h3>
      <ul className="grid gap-1.5 text-sm sm:grid-cols-3 sm:gap-4">
        {forecast.periods.map((p) => (
          <li key={p.period} className="grid gap-0.5">
            <span className="font-medium">{t(`weather.period.${p.period}`)}</span>
            <span className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
              <PeriodLine p={p} />
            </span>
          </li>
        ))}
      </ul>
      {forecast.warnings.length > 0 && (
        <p className="flex items-start gap-2 rounded-md bg-pill-accent-bg px-3 py-2 text-sm text-pill-accent-ink">
          <TriangleAlert className="mt-0.5 size-4 shrink-0" aria-hidden />
          {t('weather.dayWarning', { list: forecast.warnings.map((w) => t(`weather.warning.${w}`)).join(', ') })}
        </p>
      )}
      <Attribution />
    </section>
  )
}

/** In the booking dialogs: the forecast of the turn's period, and what to do if it looks bad. */
export function TurnWeather({ forecast, startsAt, timeZone, cancelUntil }: { forecast: DayForecast; startsAt: string; timeZone: string; cancelUntil: string }) {
  const { t } = useTranslation('facility')
  const period = forecast.periods.find((p) => p.period === periodOf(startsAt, timeZone))
  if (!period) return null
  const warnings = [...period.warnings, ...(forecast.warnings.includes('alert') ? (['alert'] as const) : [])]
  return (
    <section aria-labelledby="turn-weather" className="grid gap-2 rounded-md border px-3 py-2">
      <h3 id="turn-weather" className="text-sm font-medium">
        {t('weather.forTurn', { time: formatTime(startsAt, timeZone) })}
      </h3>
      <p className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-sm">
        <PeriodLine p={period} />
      </p>
      {warnings.length > 0 && (
        <p className="flex items-start gap-2 rounded-md bg-pill-accent-bg px-3 py-2 text-sm text-pill-accent-ink">
          <TriangleAlert className="mt-0.5 size-4 shrink-0" aria-hidden />
          {t('weather.turnWarning', {
            list: warnings.map((w) => t(`weather.warning.${w}`)).join(', '),
            time: formatTime(startsAt, timeZone),
            date: formatDateTime(cancelUntil, timeZone),
          })}
        </p>
      )}
      <Attribution />
    </section>
  )
}
