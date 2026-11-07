import { useTranslation } from 'react-i18next'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

/** Current or history: the same list, the API filters and sorts it. */
export function HistoryToggle({ history, onChange, current }: { history: boolean; onChange: (history: boolean) => void; current: string }) {
  const { t } = useTranslation('admin')
  return (
    <ToggleGroup
      type="single"
      variant="outline"
      value={history ? 'past' : 'current'}
      onValueChange={(value) => value && onChange(value === 'past')}
      aria-label={t('participation.show')}
      className="w-fit bg-card"
    >
      <ToggleGroupItem value="current" className="h-10 px-4 data-[state=on]:bg-accent data-[state=on]:text-accent-foreground">
        {current}
      </ToggleGroupItem>
      <ToggleGroupItem value="past" className="h-10 px-4 data-[state=on]:bg-accent data-[state=on]:text-accent-foreground">
        {t('participation.history')}
      </ToggleGroupItem>
    </ToggleGroup>
  )
}
