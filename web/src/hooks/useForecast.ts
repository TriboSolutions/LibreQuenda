import { useAsync } from '@/hooks/useAsync'
import { api, type DayForecast } from '@/lib/api'

// Facilities in the open air: those with this tag get the forecast.
export const OUTDOOR_TAG = 'aire-libre'

/** MeteoGalicia's forecast of an outdoor facility's day, or null (indoors, unknown, or still loading). */
export function useForecast(slug: string, facility: string, tags: string[], date: string | null): DayForecast | null {
  const outdoor = tags.includes(OUTDOOR_TAG) && date !== null
  const weather = useAsync(outdoor ? () => api.weather(slug, facility, date!) : null, `weather:${slug}/${facility}:${date}:${outdoor}`)
  return weather.data?.forecast ?? null
}
