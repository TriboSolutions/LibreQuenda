import { BadgeCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { SettingsRow } from '@/components/SettingsRow'
import type { Me } from '@/lib/api'
import { formatDateWithYear } from '@/lib/time'

/** On /perfil: the councils that verified the person as their resident, so they book at their residents-only facilities. */
export function ResidentOf({ me }: { me: Me }) {
  const { t } = useTranslation('profile')
  const councils = (me.resident_of ?? []).filter((c) => !c.by_role) // verified by the council, not by a role
  return (
    <SettingsRow id="sec-resident" title={t('resident.title')} description={t('resident.help')}>
      <div className="grid gap-3 rounded-lg border bg-card p-5">
        {councils.length === 0 ? (
          <p className="text-muted-foreground">{t('resident.none')}</p>
        ) : (
          <ul className="grid gap-2">
            {councils.map((c) => (
              <li key={c.tenant_slug} className="flex items-start gap-2">
                <BadgeCheck className="mt-0.5 size-4 shrink-0 text-pill-positive-ink" aria-hidden="true" />
                <span>
                  <Link to={`/${c.tenant_slug}`} className="font-medium underline-offset-4 hover:underline">
                    {c.tenant_name}
                  </Link>
                  <span className="block text-sm text-muted-foreground">{t('resident.since', { date: formatDateWithYear(c.verified_at.slice(0, 10)) })}</span>
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </SettingsRow>
  )
}
