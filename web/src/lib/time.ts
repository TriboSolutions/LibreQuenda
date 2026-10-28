import i18n from '@/i18n'

// Dates and times are always shown in the council's time zone. Day and month
// names and date formats come from the translations (common:date.*), not from
// Intl: not every browser ships Galician locale data.

type NameList = 'weekdays' | 'weekdaysShort' | 'months' | 'monthsShort'

function name(list: NameList, index: number) {
  return i18n.t(`common:date.${list}.${index}`)
}

/** Local parts of an instant in a time zone, using numeric output only (locale independent). */
function localParts(iso: string, timeZone: string) {
  const parts = new Intl.DateTimeFormat('en-GB', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone,
  }).formatToParts(new Date(iso))
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? '0'
  return {
    date: `${get('year')}-${get('month')}-${get('day')}`,
    hour: Number(get('hour')) % 24,
    minute: Number(get('minute')),
  }
}

function pad(n: number) {
  return String(n).padStart(2, '0') // a clock, not an index
}

/** "21:00" */
export function formatTime(iso: string, timeZone: string) {
  const { hour, minute } = localParts(iso, timeZone)
  return `${pad(hour)}:${pad(minute)}`
}

/** Minutes since local midnight in the council's time zone. */
export function minutesOfDay(iso: string, timeZone: string) {
  const { hour, minute } = localParts(iso, timeZone)
  return hour * 60 + minute
}

/** "YYYY-MM-DD" of an instant in the council's time zone. */
export function localDate(iso: string, timeZone: string) {
  return localParts(iso, timeZone).date
}

export function clockToMinutes(clock: string) {
  const [h, m] = clock.split(':').map(Number)
  return h * 60 + m
}

/** "YYYY-MM-DD" plus n days, without time zone drift. */
export function addDays(date: string, days: number) {
  const d = new Date(`${date}T12:00:00Z`)
  d.setUTCDate(d.getUTCDate() + days)
  return d.toISOString().slice(0, 10)
}

function calendar(date: string) {
  const d = new Date(`${date}T12:00:00Z`)
  return { weekday: d.getUTCDay(), day: d.getUTCDate(), month: d.getUTCMonth() }
}

/** gl: "martes, 27 de outubro" */
export function formatLongDate(date: string) {
  const { weekday, day, month } = calendar(date)
  return i18n.t('common:date.longDate', { weekday: name('weekdays', weekday), day, month: name('months', month) })
}

/** A date with its year, for documents: gl "28 de outubro de 2026", ca "28 d’octubre de 2026" */
export function formatDateWithYear(date: string) {
  const { day, month } = calendar(date)
  const monthName = name('months', month)
  // Catalan elides "de" before a vowel (d’octubre); the other languages ignore ofMonth
  const ofMonth = /^[aeiou]/i.test(monthName) ? `d’${monthName}` : `de ${monthName}`
  return i18n.t('common:date.dateWithYear', { day, month: monthName, ofMonth, year: date.slice(0, 4) })
}

/** Short weekday and day number: gl "xov" and "29" */
export function shortDateParts(date: string) {
  const { weekday, day } = calendar(date)
  return { weekday: name('weekdaysShort', weekday), day: String(day) }
}

/** gl: "xov 29" */
export function formatShortDate(date: string) {
  return i18n.t('common:date.shortDate', shortDateParts(date))
}

/** gl: "Hoxe", "Mañá" or "xov 29" relative to the council's today. */
export function dayLabel(date: string, today: string) {
  if (date === today) return i18n.t('common:date.today')
  if (date === addDays(today, 1)) return i18n.t('common:date.tomorrow')
  return formatShortDate(date)
}

function shortDateMonth(date: string) {
  const { weekday, day, month } = calendar(date)
  return i18n.t('common:date.shortDateMonth', { weekday: name('weekdaysShort', weekday), day, month: name('monthsShort', month) })
}

/** Booking day for lists, gl: "Hoxe", "Mañá" or "xov 29 out". */
export function formatBookingDay(iso: string, timeZone: string, today: string) {
  const date = localDate(iso, timeZone)
  if (date === today || date === addDays(today, 1)) return dayLabel(date, today)
  return shortDateMonth(date)
}

/** gl: "xov 29 out, 19:00" */
export function formatDateTime(iso: string, timeZone: string) {
  return i18n.t('common:date.dateTime', { date: shortDateMonth(localDate(iso, timeZone)), time: formatTime(iso, timeZone) })
}
