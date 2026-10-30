import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { format } from 'date-fns'
import { CalendarIcon } from 'lucide-react'
import { ca, enGB, es, gl, type Locale } from 'react-day-picker/locale'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'

const locales: Record<string, Locale> = { gl, es, en: enGB, ca }

type Props = {
  id?: string
  value: string // YYYY-MM-DD, "" when empty
  onChange: (value: string) => void
  min?: string // YYYY-MM-DD
  max?: string
  defaultMonth?: string // month shown while empty
  placeholder?: string
  className?: string
  'aria-describedby'?: string
}

// Calendar days, not instants: parsed and written in local time so no time zone shifts the day.
function parse(value: string | undefined) {
  if (!value) return undefined
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day)
}

function write(date: Date) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

/** A date field with the shadcn calendar, in the app's language (not the browser's). */
export function DatePicker({ id, value, onChange, min, max, defaultMonth, placeholder, className, ...rest }: Props) {
  const { t, i18n } = useTranslation()
  const [open, setOpen] = useState(false)
  // date-fns carries its own Galician: browsers' Intl may not have it
  const locale = locales[i18n.resolvedLanguage ?? 'gl'] ?? gl
  const selected = parse(value)
  const first = parse(min)
  const last = parse(max)
  const label = selected ? format(selected, 'PPP', { locale }) : placeholder

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          aria-describedby={rest['aria-describedby']}
          className={cn('num justify-between gap-3 font-normal', !selected && 'text-muted-foreground', className)}
        >
          {label}
          <CalendarIcon aria-hidden className="text-muted-foreground" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start" aria-label={label || t('day')}>
        <Calendar
          mode="single"
          locale={locale}
          captionLayout="dropdown" // month and year menus: a birth date is decades back
          selected={selected}
          defaultMonth={selected ?? parse(defaultMonth) ?? last}
          startMonth={first}
          endMonth={last}
          disabled={[...(first ? [{ before: first }] : []), ...(last ? [{ after: last }] : [])]}
          formatters={{ formatMonthDropdown: (date) => format(date, 'LLL', { locale }) }}
          onSelect={(date) => {
            if (!date) return
            onChange(write(date))
            setOpen(false)
          }}
        />
      </PopoverContent>
    </Popover>
  )
}
