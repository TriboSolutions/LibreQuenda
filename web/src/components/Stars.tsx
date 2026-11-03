import { Star } from 'lucide-react'
import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import type { RatingSummary } from '@/lib/api'
import { cn } from '@/lib/utils'

const STARS = [1, 2, 3, 4, 5]

/** An average, compact: ★ 4.3 (12); read aloud in full. Nothing until someone rated. */
export function RatingBadge({ rating, className }: { rating: RatingSummary | null | undefined; className?: string }) {
  const { t, i18n } = useTranslation('common')
  if (!rating || rating.count === 0) return null
  const average = rating.average.toLocaleString(i18n.resolvedLanguage, { minimumFractionDigits: 1, maximumFractionDigits: 1 })
  return (
    <span className={cn('num inline-flex items-center gap-1 text-sm', className)}>
      <Star aria-hidden className="size-4 fill-current text-star" />
      <span aria-hidden>
        <span className="font-medium text-foreground">{average}</span> <span className="text-muted-foreground">({rating.count})</span>
      </span>
      <span className="sr-only">{t('rating.summary', { average, count: rating.count })}</span>
    </span>
  )
}

type InputProps = {
  label: string // what is rated, the group's name
  value: number // 0 = none yet
  onChange: (stars: number) => void
  disabled?: boolean
}

/** 1 to 5 stars to choose: native radio buttons (arrow keys move between them), drawn as stars. */
export function StarInput({ label, value, onChange, disabled }: InputProps) {
  const { t } = useTranslation('common')
  const name = useId()
  return (
    <fieldset className="grid gap-1" disabled={disabled}>
      <legend className="mb-1 text-sm font-medium">{label}</legend>
      <div className="flex gap-0.5">
        {STARS.map((stars) => (
          <label key={stars} className="group relative cursor-pointer has-[:disabled]:cursor-not-allowed">
            <input
              type="radio"
              name={name}
              value={stars}
              checked={value === stars}
              onChange={() => onChange(stars)}
              className="peer sr-only"
            />
            <span className="sr-only">{t('rating.stars', { count: stars })}</span>
            <span className="flex size-11 items-center justify-center rounded-md peer-focus-visible:ring-[3px] peer-focus-visible:ring-ring/50">
              <Star
                aria-hidden
                className={cn('size-7 transition-colors', stars <= value ? 'fill-star text-star' : 'text-muted-foreground group-hover:text-star')}
              />
            </span>
          </label>
        ))}
      </div>
    </fieldset>
  )
}
