import { CircleCheck, Wrench } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { IssuesSummary } from '@/lib/api'
import { formatLongDate, localDate } from '@/lib/time'

type Props = {
  issues: IssuesSummary
  councilName: string
  timeZone: string
}

/**
 * The public side of the maintenance list: what is known to be broken (so nobody
 * reports it twice), what was fixed lately, and how long the council takes on
 * average. Categories and dates only: never what anyone wrote, nor who.
 */
export function FacilityIssues({ issues, councilName, timeZone }: Props) {
  const { t } = useTranslation('facility')
  const day = (iso: string) => formatLongDate(localDate(iso, timeZone))
  return (
    <section aria-labelledby="state-title" className="grid gap-3 rounded-lg border bg-card p-5">
      <h2 id="state-title" className="text-lg font-semibold">
        {t('state.title')}
      </h2>
      {issues.open.length === 0 ? (
        <p className="flex items-start gap-2">
          <CircleCheck className="mt-0.5 size-4 shrink-0 text-pill-positive-ink" aria-hidden />
          {t('state.noKnown')}
        </p>
      ) : (
        <div className="grid gap-1.5">
          <h3 className="font-medium">{t('state.known')}</h3>
          <ul className="grid gap-2">
            {issues.open.map((issue) => (
              <li key={issue.category} className="flex items-start gap-2">
                <Wrench className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
                <span className="grid gap-0.5">
                  <span>{t(`issue.categories.${issue.category}`)}</span>
                  <span className="num text-sm text-muted-foreground">
                    {issue.in_progress ? t('state.inProgress') : t('state.reported')} · {t('state.since', { date: day(issue.first_reported_at) })}
                    {issue.reports > 1 && ` · ${t('state.reports', { count: issue.reports })}`}
                  </span>
                </span>
              </li>
            ))}
          </ul>
          <p className="text-sm text-muted-foreground">{t('state.alreadyKnown')}</p>
        </div>
      )}
      {issues.recently_solved.length > 0 && (
        <div className="grid gap-1.5">
          <h3 className="font-medium">{t('state.solved')}</h3>
          <ul className="grid gap-2">
            {issues.recently_solved.map((issue) => (
              <li key={issue.category} className="flex items-start gap-2">
                <CircleCheck className="mt-0.5 size-4 shrink-0 text-pill-positive-ink" aria-hidden />
                <span className="grid gap-0.5">
                  <span>{t(`issue.categories.${issue.category}`)}</span>
                  <span className="num text-sm text-muted-foreground">{t('state.solvedOn', { date: day(issue.solved_at) })}</span>
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
      {issues.average_days_to_solve !== null && (
        <p className="num rounded-md bg-muted px-3 py-2 text-sm">
          {t('state.average', { council: councilName, days: formatDays(issues.average_days_to_solve), count: issues.solved_count })}
        </p>
      )}
    </section>
  )
}

/** 3.8 → "3,8" in the current language */
function formatDays(days: number) {
  return new Intl.NumberFormat(document.documentElement.lang || 'gl', { maximumFractionDigits: 1 }).format(days)
}
