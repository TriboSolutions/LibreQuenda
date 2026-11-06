import { ChevronDown, Star } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuLabel, DropdownMenuRadioGroup, DropdownMenuRadioItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'

/** The minimum averages offered; 0 is any (and a facility nobody rated stays). */
const MINIMUMS = [0, 3, 4, 4.5]

type Props = {
  min: number
  onChange: (min: number) => void
}

/** «Rating»: only facilities residents rated at least this on average (the API filters, ?min_rating=). */
export function RatingFilter({ min, onChange }: Props) {
  const { t, i18n } = useTranslation('home')
  const stars = (value: number) => value.toLocaleString(i18n.resolvedLanguage, { maximumFractionDigits: 1 })
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className={min > 0 ? 'border-primary text-primary' : ''}>
          {min > 0 ? (
            <>
              <Star className="fill-current" aria-hidden />
              {t('rating.atLeast', { stars: stars(min) })}
              {/* the name keeps the visible text first (WCAG 2.5.3), then what it filters */}
              <span className="sr-only">, {t('rating.title')}</span>
            </>
          ) : (
            t('rating.title')
          )}
          <ChevronDown className="opacity-60" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-52">
        <DropdownMenuLabel>{t('rating.title')}</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={String(min)} onValueChange={(value) => onChange(Number(value))}>
          {MINIMUMS.map((value) => (
            <DropdownMenuRadioItem key={value} value={String(value)}>
              {value === 0 ? t('rating.any') : t('rating.atLeast', { stars: stars(value) })}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
