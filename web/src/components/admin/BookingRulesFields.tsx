import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { BookingRules } from '@/lib/api'

type Props = {
  id: string // prefix for the fields' ids, so two forms can share a page
  value: BookingRules
  onChange: (rules: BookingRules) => void
}

const number = (value: string) => Math.max(0, Math.round(Number(value) || 0))

/** The booking window's fields: how far ahead, how close to the start, the limits per person and the no-show penalty. */
const WEEKDAYS = [1, 2, 3, 4, 5, 6, 7] // ISO; common:date.weekdays counts from Sunday = 0

const clock = (minute: number) => `${String(Math.floor(minute / 60)).padStart(2, '0')}:${String(minute % 60).padStart(2, '0')}`

export function BookingRulesFields({ id, value, onChange }: Props) {
  const { t } = useTranslation(['admin', 'common'])
  const set = <K extends keyof BookingRules>(key: K, v: BookingRules[K]) => onChange({ ...value, [key]: v })
  // days when it is a whole number of them; the value itself is kept in hours
  const [unit, setUnit] = useState<'days' | 'hours'>(value.max_advance_hours % 24 === 0 ? 'days' : 'hours')
  const advance = unit === 'days' ? value.max_advance_hours / 24 : value.max_advance_hours

  const field = (key: keyof BookingRules, label: string, min: number, max: number, help?: string) => (
    <div className="grid content-start gap-2">
      <Label htmlFor={`${id}-${key}`}>{label}</Label>
      <Input
        id={`${id}-${key}`}
        type="number"
        required
        min={min}
        max={max}
        value={value[key] as number}
        aria-describedby={help ? `${id}-${key}-help` : undefined}
        onChange={(e) => set(key, number(e.target.value))}
        className="num"
      />
      {help && (
        <p id={`${id}-${key}-help`} className="text-xs text-muted-foreground">
          {help}
        </p>
      )}
    </div>
  )

  return (
    <div className="grid gap-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid content-start gap-2 sm:col-span-2">
          <Label htmlFor={`${id}-mode`}>{t('rules.opening')}</Label>
          <Select value={value.weekly_opening ? 'weekly' : 'rolling'} onValueChange={(mode) => set('weekly_opening', mode === 'weekly')}>
            <SelectTrigger id={`${id}-mode`} className="w-full sm:max-w-sm" aria-describedby={`${id}-mode-help`}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="rolling">{t('rules.rolling')}</SelectItem>
              <SelectItem value="weekly">{t('rules.weekly')}</SelectItem>
            </SelectContent>
          </Select>
          <p id={`${id}-mode-help`} className="text-xs text-muted-foreground">
            {value.weekly_opening ? t('rules.weeklyHelp') : t('rules.rollingHelp')}
          </p>
        </div>
        {value.weekly_opening ? (
          <div className="grid content-start gap-2">
            <span id={`${id}-opens`} className="text-sm leading-none font-medium">
              {t('rules.opensAt')}
            </span>
            <div role="group" aria-labelledby={`${id}-opens`} className="flex gap-2">
              <Select value={String(value.opens_weekday)} onValueChange={(day) => set('opens_weekday', Number(day))}>
                <SelectTrigger className="w-full" aria-label={t('rules.opensDay')}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {WEEKDAYS.map((day) => (
                    <SelectItem key={day} value={String(day)}>
                      {t(`common:date.weekdays.${day % 7}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Input
                type="time"
                required
                aria-label={t('rules.opensTime')}
                value={clock(value.opens_minute)}
                onChange={(e) => {
                  const [h, m] = e.target.value.split(':').map(Number)
                  if (!Number.isNaN(h) && !Number.isNaN(m)) set('opens_minute', h * 60 + m)
                }}
                className="num w-32 shrink-0"
              />
            </div>
          </div>
        ) : (
          <div className="grid content-start gap-2">
            <Label htmlFor={`${id}-advance`}>{t('rules.maxAdvance')}</Label>
            <div className="flex gap-2">
              <Input
                id={`${id}-advance`}
                type="number"
                required
                min={1}
                max={unit === 'days' ? 365 : 365 * 24}
                value={advance}
                aria-describedby={`${id}-advance-help`}
                onChange={(e) => set('max_advance_hours', Math.max(1, number(e.target.value)) * (unit === 'days' ? 24 : 1))}
                className="num"
              />
              <Select
                value={unit}
                onValueChange={(next) => {
                  setUnit(next as 'days' | 'hours')
                  // keep what is typed as the number, in the new unit
                  set('max_advance_hours', Math.max(1, advance) * (next === 'days' ? 24 : 1))
                }}
              >
                <SelectTrigger className="w-32 shrink-0" aria-label={t('rules.unit')}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="days">{t('rules.days')}</SelectItem>
                  <SelectItem value="hours">{t('rules.hours')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <p id={`${id}-advance-help`} className="text-xs text-muted-foreground">
              {t('rules.maxAdvanceHelp')}
            </p>
          </div>
        )}
        {field('min_advance_minutes', t('rules.minAdvance'), 0, 1440, t('rules.minAdvanceHelp'))}
        {field('min_cancel_hours', t('rules.minCancel'), 0, 720, t('rules.minCancelHelp'))}
        {field('max_active_bookings_per_user', t('rules.maxActive'), 1, 50, t('rules.maxActiveHelp'))}
        {field('max_bookings_per_day', t('rules.perDay'), 0, 24, t('rules.zeroNoLimit'))}
        {field('max_turns_in_a_row', t('rules.inARow'), 0, 24, t('rules.zeroNoLimit'))}
        {field('max_bookings_per_week', t('rules.perWeek'), 0, 100, t('rules.zeroNoLimit'))}
      </div>

      <div className="grid gap-4 rounded-md border p-4">
        <div className="flex items-start gap-3">
          <Switch
            id={`${id}-penalty`}
            checked={value.no_show_penalty}
            onCheckedChange={(on) => set('no_show_penalty', on)}
            aria-describedby={`${id}-penalty-help`}
          />
          <div className="grid gap-1">
            <Label htmlFor={`${id}-penalty`}>{t('rules.penalty')}</Label>
            <p id={`${id}-penalty-help`} className="text-sm text-muted-foreground">
              {t('rules.penaltyHelp')}
            </p>
          </div>
        </div>
        {value.no_show_penalty && (
          <div className="grid gap-4 sm:grid-cols-3">
            {field('no_show_limit', t('rules.noShows'), 1, 20)}
            {field('no_show_window_days', t('rules.window'), 1, 365)}
            {field('no_show_block_days', t('rules.block'), 1, 365)}
          </div>
        )}
      </div>
    </div>
  )
}

/** The window in a few lines, read-only: what a facility inherits from the council. */
export function BookingRulesSummary({ rules }: { rules: BookingRules }) {
  const { t } = useTranslation(['admin', 'common'])
  const advance = rules.max_advance_hours % 24 === 0 ? t('rules.daysN', { count: rules.max_advance_hours / 24 }) : t('rules.hoursN', { count: rules.max_advance_hours })
  return (
    <ul className="num grid list-disc gap-1 pl-5 text-sm text-muted-foreground">
      <li>
        {rules.weekly_opening
          ? t('rules.sumWeekly', { weekday: t(`common:date.weekdays.${rules.opens_weekday % 7}`), time: clock(rules.opens_minute) })
          : t('rules.sumAdvance', { advance })}
      </li>
      {rules.min_advance_minutes > 0 && <li>{t('rules.sumMinAdvance', { count: rules.min_advance_minutes })}</li>}
      <li>{t('rules.sumCancel', { count: rules.min_cancel_hours })}</li>
      <li>{t('rules.sumActive', { count: rules.max_active_bookings_per_user })}</li>
      {rules.max_bookings_per_day > 0 && <li>{t('rules.sumPerDay', { count: rules.max_bookings_per_day })}</li>}
      {rules.max_turns_in_a_row > 0 && <li>{t('rules.sumInARow', { count: rules.max_turns_in_a_row })}</li>}
      {rules.max_bookings_per_week > 0 && <li>{t('rules.sumPerWeek', { count: rules.max_bookings_per_week })}</li>}
      <li>
        {rules.no_show_penalty
          ? t('rules.sumPenalty', { limit: rules.no_show_limit, window: rules.no_show_window_days, block: rules.no_show_block_days })
          : t('rules.sumNoPenalty')}
      </li>
    </ul>
  )
}
