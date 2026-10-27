import { useTranslation } from 'react-i18next'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { addDays, dayLabel, formatLongDate, shortDateParts } from '@/lib/time'

type Props = {
  today: string
  value: string
  days?: number
  onChange: (date: string) => void
}

function labels(date: string, today: string, t: (key: string) => string) {
  const { weekday, day } = shortDateParts(date)
  const top = date === today ? t('date.todayShort') : date === addDays(today, 1) ? t('date.tomorrowShort') : weekday
  return { top, bottom: day, inline: dayLabel(date, today) }
}

/**
 * The next `days` days as one segmented control: a full-width grid of
 * two-line cells on a phone, one row of labels from `sm` up.
 */
export function DayPicker({ today, value, days = 7, onChange }: Props) {
  const { t } = useTranslation()
  const dates = Array.from({ length: days }, (_, i) => addDays(today, i))

  return (
    <ToggleGroup
      type="single"
      variant="outline"
      value={value}
      onValueChange={(date) => date && onChange(date)}
      aria-label={t('day')}
      className="grid w-full grid-cols-7 bg-card sm:flex sm:w-fit"
    >
      {dates.map((date) => {
        const label = labels(date, today, t)
        return (
          <ToggleGroupItem
            key={date}
            value={date}
            aria-label={formatLongDate(date)}
            className="num h-12 min-w-0 flex-col gap-0 px-1 leading-4 data-[state=on]:bg-accent data-[state=on]:text-accent-foreground sm:h-8 sm:flex-row sm:px-3"
          >
            <span className="text-xs text-muted-foreground sm:hidden">{label.top}</span>
            <span className="sm:hidden">{label.bottom}</span>
            <span className="hidden sm:inline">{label.inline}</span>
          </ToggleGroupItem>
        )
      })}
    </ToggleGroup>
  )
}
