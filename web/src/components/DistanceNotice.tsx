import { useTranslation } from 'react-i18next'
import { LocateFixed } from 'lucide-react'

/**
 * How close the phone must be for the fixed ways in (the code, the QR on the
 * door): the facility's distance when it is known, else the usual one.
 */
export function DistanceNotice({ metres }: { metres?: number }) {
  const { t } = useTranslation('entrance')
  return (
    <p className="flex items-start gap-2 text-sm text-muted-foreground">
      <LocateFixed className="mt-0.5 size-4 shrink-0" aria-hidden />
      {metres ? t('distance', { metres }) : t('distanceAny')}
    </p>
  )
}
