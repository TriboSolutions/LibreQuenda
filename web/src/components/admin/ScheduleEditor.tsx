import { Plus, X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { TimeSelect } from '@/components/TimeSelect'
import { Button } from '@/components/ui/button'
import type { WeeklyWindow } from '@/lib/api'

// ISO weekdays in display order; Sunday is 7
const WEEK = [1, 2, 3, 4, 5, 6, 7]

// Saved on its own with its button (an existing facility), or part of a bigger form
// that saves it (a new one): then onChange follows every edit and there is no button.
type Props = { value: WeeklyWindow[] } & ({ saving: boolean; onSave: (windows: WeeklyWindow[]) => void; onChange?: never } | { onChange: (windows: WeeklyWindow[]) => void; saving?: never; onSave?: never })

/** The weekly opening hours: any number of windows per day (morning and afternoon, for example). */
// maxWindowsPerDay in the API (counciladminDtos.go): more opening windows in one day are refused
const MAX_WINDOWS = 6

export function ScheduleEditor({ value, saving, onSave, onChange }: Props) {
  const { t } = useTranslation(['admin', 'common', 'facility'])
  const [draft, setDraft] = useState<WeeklyWindow[]>(value)
  const windows = onChange ? value : draft
  const setWindows = (next: (list: WeeklyWindow[]) => WeeklyWindow[]) => (onChange ? onChange(next(windows)) : setDraft(next))

  const byDay = (day: number) => windows.map((w, index) => ({ ...w, index })).filter((w) => w.weekday === day)
  const update = (index: number, field: 'opens_at' | 'closes_at', time: string) =>
    setWindows((list) => list.map((w, i) => (i === index ? { ...w, [field]: time } : w)))
  // as the API takes: up to MAX_WINDOWS a day
  const remove = (index: number) => setWindows((list) => list.filter((_, i) => i !== index))
  const add = (day: number) => setWindows((list) => [...list, { weekday: day, opens_at: '09:00', closes_at: '21:00' }])
  const copyMonday = () =>
    setWindows((list) => [
      ...list.filter((w) => w.weekday === 1 || w.weekday > 5),
      ...[2, 3, 4, 5].flatMap((day) => list.filter((w) => w.weekday === 1).map((w) => ({ ...w, weekday: day }))),
    ])

  return (
    <div className="grid gap-3">
      <ul className="grid">
        {WEEK.map((day) => {
          const dayWindows = byDay(day)
          return (
            <li key={day} className="grid gap-2 border-t py-3 first:border-t-0 sm:grid-cols-[120px_1fr] sm:items-start">
              <span className="pt-2 font-medium capitalize">{t(`common:date.weekdays.${day % 7}`)}</span>
              <div className="grid gap-2">
                {dayWindows.length === 0 && <span className="pt-2 text-muted-foreground">{t('facility:closed')}</span>}
                {dayWindows.map((w) => (
                  <div key={w.index} className="flex flex-wrap items-center gap-2">
                    <TimeSelect
                      value={w.opens_at}
                      onChange={(time) => update(w.index, 'opens_at', time)}
                      aria-label={`${t(`common:date.weekdays.${day % 7}`)}: ${t('admin:editor.opens')}`}
                      className="w-24"
                    />
                    <span className="text-muted-foreground" aria-hidden="true">
                      –
                    </span>
                    <TimeSelect
                      value={w.closes_at}
                      onChange={(time) => update(w.index, 'closes_at', time)}
                      aria-label={`${t(`common:date.weekdays.${day % 7}`)}: ${t('admin:editor.closes')}`}
                      className="w-24"
                    />
                    <Button type="button" variant="ghost" size="icon" onClick={() => remove(w.index)} aria-label={t('admin:editor.removeWindow')}>
                      <X />
                    </Button>
                  </div>
                ))}
                {dayWindows.length < MAX_WINDOWS && (
                  <div>
                    <Button type="button" variant="ghost" size="sm" onClick={() => add(day)}>
                      <Plus />
                      {t('admin:editor.addWindow')}
                    </Button>
                  </div>
                )}
              </div>
            </li>
          )
        })}
      </ul>
      <div className="flex flex-wrap justify-between gap-2 border-t pt-4">
        <Button type="button" variant="outline" onClick={copyMonday}>
          {t('admin:editor.copyWeekdays')}
        </Button>
        {onSave && (
          <Button type="button" disabled={saving} onClick={() => onSave(windows)}>
            {saving ? t('admin:form.saving') : t('admin:editor.saveSchedule')}
          </Button>
        )}
      </div>
    </div>
  )
}
