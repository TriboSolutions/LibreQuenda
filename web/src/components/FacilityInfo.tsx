import { ExternalLink, MapPin } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import type { FacilityDetails } from '@/lib/api'
import { directionsUrl } from '@/lib/maps'

/** Where to go and what to know before going: address, directions, conditions and the rules of use. */
export function FacilityInfo({ facility, withAddress = true }: { facility: FacilityDetails; withAddress?: boolean }) {
  const { t } = useTranslation(['facility', 'invites'])
  const hasPin = facility.latitude !== null && facility.longitude !== null

  return (
    <div className="grid gap-4">
      {withAddress && (facility.address || hasPin) && (
        <div className="flex flex-wrap items-start justify-between gap-3">
          <p className="flex min-w-0 items-start gap-2">
            <MapPin className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
            <span>{facility.address || t('facility:noLocation')}</span>
          </p>
          {hasPin && (
            <Button asChild variant="outline" size="sm">
              <a href={directionsUrl(facility.latitude!, facility.longitude!)} target="_blank" rel="noopener noreferrer">
                {t('facility:directions')}
                <ExternalLink aria-hidden />
              </a>
            </Button>
          )}
        </div>
      )}

      <section aria-labelledby="facility-info-conditions" className="grid gap-1.5">
        <h3 id="facility-info-conditions" className="font-medium">
          {t('facility:conditions')}
        </h3>
        <ul className="num grid gap-1 text-muted-foreground">
          {facility.sport && <li>{facility.sport}</li>}
          <li>{t('facility:ruleSlot', { count: facility.slot_minutes })}</li>
          <li>{t('facility:ruleCancel', { count: facility.min_cancel_hours })}</li>
          {facility.min_age !== null && <li className="font-medium text-foreground">{t('facility:minAge', { count: facility.min_age })}</li>}
        </ul>
      </section>

      {facility.accessibility_info && (
        <section aria-labelledby="facility-info-accessibility" className="grid gap-1.5">
          <h3 id="facility-info-accessibility" className="font-medium">
            {t('invites:details.accessibility')}
          </h3>
          <p className="whitespace-pre-line text-muted-foreground">{facility.accessibility_info}</p>
        </section>
      )}

      <section aria-labelledby="facility-info-rules" className="grid gap-1.5 rounded-md bg-muted p-3">
        <h3 id="facility-info-rules" className="font-medium">
          {t('facility:rules')}
        </h3>
        <p className="max-h-48 overflow-y-auto whitespace-pre-line">{facility.usage_rules || t('facility:noRules')}</p>
      </section>
    </div>
  )
}
