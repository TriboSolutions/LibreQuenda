import { useAsync } from '@/hooks/useAsync'
import { api, type DayForecast } from '@/lib/api'

/** MeteoGalicia's forecast of a facility's day, or null (disabled, unknown, or still loading). */
export function useForecast(slug: string, facility: string, showWeather: boolean, date: string | null): DayForecast | null {
  const enabled = showWeather && date !== null
  const weather = useAsync(enabled ? () => api.weather(slug, facility, date!) : null, `weather:${slug}/${facility}:${date}:${enabled}`)
  return weather.data?.forecast ?? null
}
