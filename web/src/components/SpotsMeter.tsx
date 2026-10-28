import { useTranslation } from 'react-i18next'

type Props = {
  capacity: number // people including the owner
  taken: number // the owner plus who joined
}

// Up to this many people each one gets a cell; a bigger group gets one continuous bar
const MAX_CELLS = 12

/** How many people fit, how many are in and how many spots are left. */
export function SpotsMeter({ capacity, taken }: Props) {
  const { t } = useTranslation('invites')
  const left = Math.max(capacity - taken, 0)
  const label = t('meter.label', { taken, capacity })

  return (
    <div className="grid gap-1.5">
      <p className="num flex flex-wrap items-baseline justify-between gap-x-3">
        <span className="font-medium">{left === 0 ? t('meter.full') : t('meter.left', { count: left })}</span>
        <span className="text-sm text-muted-foreground">{label}</span>
      </p>
      {capacity <= MAX_CELLS ? (
        <div role="img" aria-label={label} className="flex gap-1">
          {Array.from({ length: capacity }, (_, i) => (
            <span key={i} className={`h-2 flex-1 rounded-full ${i < taken ? 'bg-primary' : 'bg-muted'}`} />
          ))}
        </div>
      ) : (
        <div role="img" aria-label={label} className="h-2 overflow-hidden rounded-full bg-muted">
          <div className="h-full rounded-full bg-primary" style={{ width: `${Math.min(taken / capacity, 1) * 100}%` }} />
        </div>
      )}
    </div>
  )
}
