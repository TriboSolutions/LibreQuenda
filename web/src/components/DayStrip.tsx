import { useTranslation } from 'react-i18next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { Facility, Slot } from '@/lib/api'
import { cn } from '@/lib/utils'
import { clockToMinutes, formatTime, minutesOfDay } from '@/lib/time'

export type Axis = { start: number; end: number } // minutes since local midnight

const statusKey: Record<Slot['status'], string> = {
  free: 'slot.free',
  booked: 'slot.booked',
  closed: 'slot.closure', // a closure inside the opening hours
  past: 'slot.past',
  too_far: 'slot.tooFar',
}

const statusClass: Record<Slot['status'], string> = {
  free: 'bg-slot-free hover:bg-slot-free-hover cursor-pointer',
  booked: 'bg-slot-booked',
  closed: 'hatch hatch-closure',
  past: 'bg-slot-past',
  too_far: 'bg-slot-past',
}

function percent(minutes: number, axis: Axis) {
  return ((minutes - axis.start) / (axis.end - axis.start)) * 100
}

/** Hour labels ("07", "09"…) on the shared axis, every two hours from the first opening to the last closing. */
export function AxisLabels({ axis }: { axis: Axis }) {
  const hours: number[] = []
  for (let m = axis.start; m <= axis.end; m += 120) hours.push(m)

  return (
    <div className="relative h-4 text-xs text-muted-foreground num" aria-hidden="true">
      {hours.map((m, i) => (
        <span
          key={m}
          className={cn('absolute top-0', i === 0 ? '' : i === hours.length - 1 ? '-translate-x-full' : '-translate-x-1/2')}
          style={{ left: `${percent(m, axis)}%` }}
        >
          {String(Math.floor(m / 60)).padStart(2, '0')}
        </span>
      ))}
    </div>
  )
}

type Props = {
  facility: Facility
  axis: Axis
  timeZone: string
  nowMinutes: number | null // only on today
  onPick: (slot: Slot) => void
}

/** Stretches of the axis outside the opening hours, hatched. */
function outsideHours(facility: Facility, axis: Axis) {
  const gaps: { start: number; end: number }[] = []
  let from = axis.start
  for (const w of [...facility.windows].sort((a, b) => a.opens_at.localeCompare(b.opens_at))) {
    const opens = clockToMinutes(w.opens_at)
    if (opens > from) gaps.push({ start: from, end: opens })
    from = Math.max(from, clockToMinutes(w.closes_at) || 24 * 60)
  }
  if (from < axis.end) gaps.push({ start: from, end: axis.end })
  return gaps
}

/** Back-to-back closed turns as one stretch, the way a closure is one thing. */
function groupClosed(slots: Slot[]) {
  const groups: Slot[] = []
  for (const slot of slots) {
    const last = groups[groups.length - 1]
    if (slot.status === 'closed' && last?.status === 'closed' && last.ends_at === slot.starts_at) {
      groups[groups.length - 1] = { ...last, ends_at: slot.ends_at }
    } else {
      groups.push(slot)
    }
  }
  return groups
}

/** The reason of the closure covering a stretch, if the council gave one. */
function closureReason(facility: Facility, slot: Slot) {
  const start = Date.parse(slot.starts_at)
  const end = Date.parse(slot.ends_at)
  return facility.closures.find((c) => Date.parse(c.starts_at) < end && Date.parse(c.ends_at) > start)?.reason ?? ''
}

/**
 * The day of one facility on the shared axis: every turn as a block, free turns
 * in the brand hue and clickable, booked in grey, closures hatched in red, and the hours
 * it does not open hatched.
 */
export function DayStrip({ facility, axis, timeZone, nowMinutes, onPick }: Props) {
  const { t } = useTranslation('home')

  return (
    <div className="relative h-11 rounded-sm bg-muted/60">
      {outsideHours(facility, axis).map((gap) => (
        <span
          key={gap.start}
          className="hatch absolute inset-y-0 rounded-[3px] overflow-hidden"
          style={{ left: `${percent(gap.start, axis)}%`, width: `calc(${percent(gap.end, axis) - percent(gap.start, axis)}% - 2px)` }}
          aria-hidden="true"
        />
      ))}
      {groupClosed(facility.slots).map((slot) => {
        const start = minutesOfDay(slot.starts_at, timeZone)
        const end = minutesOfDay(slot.ends_at, timeZone) || 24 * 60
        const label = `${formatTime(slot.starts_at, timeZone)}–${formatTime(slot.ends_at, timeZone)}`
        const style = { left: `${percent(start, axis)}%`, width: `calc(${percent(end, axis) - percent(start, axis)}% - 2px)` }
        const className = cn('absolute inset-y-0 rounded-[3px] overflow-hidden', statusClass[slot.status])

        if (slot.status === 'free') {
          return (
            <Tooltip key={slot.starts_at}>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  className={cn(className, 'outline-offset-2 transition-colors')}
                  style={style}
                  aria-label={t('slot.bookAria', { facility: facility.name, time: label })}
                  onClick={() => onPick(slot)}
                />
              </TooltipTrigger>
              <TooltipContent className="num">
                {label} · {t('slot.free')}
              </TooltipContent>
            </Tooltip>
          )
        }
        const reason = slot.status === 'closed' ? closureReason(facility, slot) : ''
        const text = [label, t(statusKey[slot.status]), reason].filter(Boolean).join(' · ')
        return (
          <Tooltip key={slot.starts_at}>
            <TooltipTrigger asChild>
              <span className={className} style={style} role="img" aria-label={text} />
            </TooltipTrigger>
            <TooltipContent className="num">{text}</TooltipContent>
          </Tooltip>
        )
      })}
      {nowMinutes !== null && nowMinutes > axis.start && nowMinutes < axis.end && (
        <span className="absolute -inset-y-1 w-px bg-now" style={{ left: `${percent(nowMinutes, axis)}%` }} aria-hidden="true" />
      )}
    </div>
  )
}

/** The swatches are the marks themselves, scaled down. */
export function StripLegend() {
  const { t } = useTranslation('home')
  const items = [
    { label: t('slot.free'), className: 'bg-slot-free' },
    { label: t('slot.booked'), className: 'bg-slot-booked' },
    { label: t('slot.closed'), className: 'hatch' },
    { label: t('slot.closure'), className: 'hatch hatch-closure' },
    { label: t('slot.past'), className: 'bg-slot-past' },
  ]
  return (
    <ul className="hidden flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground md:flex">
      {items.map((item) => (
        <li key={item.label} className="flex items-center gap-1.5">
          <span className={cn('relative size-2.5 rounded-[2px] overflow-hidden', item.className)} aria-hidden="true" />
          {item.label}
        </li>
      ))}
    </ul>
  )
}
