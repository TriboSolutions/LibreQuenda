import { ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import type { JoinableKind } from '@/lib/api'

const KINDS: JoinableKind[] = ['public', 'event']

type Props = {
  chosen: JoinableKind[]
  onChange: (kinds: JoinableKind[]) => void
}

/** «Type of booking»: only the facilities with residents' public bookings or the council's events to join; any of those chosen. */
export function BookingKindFilter({ chosen, onChange }: Props) {
  const { t } = useTranslation('home')
  const toggle = (kind: JoinableKind, on: boolean) => onChange(on ? [...chosen, kind] : chosen.filter((k) => k !== kind))
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" className={chosen.length > 0 ? 'border-primary text-primary' : ''}>
          {chosen.length === 1 ? t(`bookingKind.${chosen[0]}`) : t('bookingKind.title')}
          {/* the name keeps the visible text first (WCAG 2.5.3), then what it filters */}
          {chosen.length === 1 && <span className="sr-only">, {t('bookingKind.title')}</span>}
          {chosen.length > 1 && <span className="num rounded-full bg-primary px-1.5 text-xs text-primary-foreground">{chosen.length}</span>}
          <ChevronDown className="opacity-60" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-52">
        <DropdownMenuLabel>{t('bookingKind.title')}</DropdownMenuLabel>
        {KINDS.map((kind) => (
          <DropdownMenuCheckboxItem
            key={kind}
            checked={chosen.includes(kind)}
            onCheckedChange={(on) => toggle(kind, on)}
            onSelect={(event) => event.preventDefault()} // stays open to pick both
          >
            {t(`bookingKind.${kind}`)}
          </DropdownMenuCheckboxItem>
        ))}
        {chosen.length > 0 && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => onChange([])}>{t('tags.clearCategory')}</DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
