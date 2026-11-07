import { Megaphone } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import type { PublicNotice } from '@/lib/api'
import { formatDateTime, formatLongDate, localDate } from '@/lib/time'

type Props = {
  notices: PublicNotice[]
  timeZone: string
  /** the council's page links each one to the facility it is about */
  tenantSlug?: string
}

/** The council's current announcements («Pechamos a piscina polas obras»), newest first. */
export function CouncilNotices({ notices, timeZone, tenantSlug }: Props) {
  const { t } = useTranslation('facility')
  if (notices.length === 0) return null
  return (
    <section aria-labelledby="notices-title" className="grid gap-2">
      <h2 id="notices-title" className="sr-only">
        {t('notices.title')}
      </h2>
      <ul className="grid gap-2">
        {notices.map((notice) => (
          <li key={notice.id} className="flex items-start gap-3 rounded-lg border border-pill-accent-ink/40 bg-pill-accent-bg px-4 py-3 text-pill-accent-ink">
            <Megaphone className="mt-0.5 size-5 shrink-0" aria-hidden />
            <div className="grid min-w-0 gap-1">
              <p className="font-semibold break-words">
                <span className="sr-only">{t('notices.label')}: </span>
                {notice.title}
              </p>
              <p className="whitespace-pre-line break-words">{notice.body}</p>
              <p className="num text-sm opacity-90">
                {tenantSlug && notice.facility_slug ? (
                  <>
                    <Link to={`/${tenantSlug}/${notice.facility_slug}`} className="font-medium underline underline-offset-4">
                      {notice.facility_name}
                    </Link>
                    {' · '}
                  </>
                ) : null}
                {t('notices.published', { date: formatDateTime(notice.created_at, timeZone) })}
                {' · '}
                {t('notices.until', { date: formatLongDate(lastDay(notice.expires_at, timeZone)) })}
              </p>
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}

/** expires_at is the first instant after the last day it is shown. */
function lastDay(expiresAt: string, timeZone: string) {
  return localDate(new Date(Date.parse(expiresAt) - 1).toISOString(), timeZone)
}
