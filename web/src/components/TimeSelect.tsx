import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'

const pad = (n: number) => String(n).padStart(2, '0')

// every `step` minutes, 24-hour clock; 23:59 closes a day
function times(step: number) {
  const list = Array.from({ length: (24 * 60) / step }, (_, i) => `${pad(Math.floor((i * step) / 60))}:${pad((i * step) % 60)}`)
  return [...list, '23:59']
}
const QUARTERS = times(15)

type Props = {
  id?: string
  value: string // HH:MM
  onChange: (value: string) => void
  'aria-label'?: string
  className?: string
}

/** A time of day from a menu, in 24 hours whatever the browser's locale (no AM/PM). */
export function TimeSelect({ id, value, onChange, className, ...rest }: Props) {
  // a saved time off the 15-minute grid still shows, put in its place (no sorting: the grid is in order)
  const at = value && !QUARTERS.includes(value) ? QUARTERS.findIndex((q) => q > value) : -2
  const options = at === -2 ? QUARTERS : at === -1 ? [...QUARTERS, value] : [...QUARTERS.slice(0, at), value, ...QUARTERS.slice(at)]
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger id={id} aria-label={rest['aria-label']} className={cn('num', className)}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="max-h-64">
        {options.map((time) => (
          <SelectItem key={time} value={time} className="num">
            {time}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
