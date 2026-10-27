import { useTranslation } from 'react-i18next'
import type { Day } from '@/lib/api'

/** What a resident must know before booking each facility, straight from its rules. */
export function BookingRules({ day }: { day: Day }) {
  const { t } = useTranslation('home')

  return (
    <section aria-labelledby="rules-title" className="rounded-lg border bg-card p-5">
      <div className="flex items-baseline justify-between gap-4">
        <h2 id="rules-title" className="text-lg font-semibold">
          {t('rules.title')}
        </h2>
        <span className="text-muted-foreground" title={t('rules.figuresHelp')}>
          {t('rules.figures')}
        </span>
      </div>
      <table className="mt-2 w-full">
        <caption className="sr-only">{t('rules.figuresHelp')}</caption>
        <tbody className="num">
          {day.facilities.map((f) => (
            <tr key={f.id} className="border-t first:border-t-0">
              <th scope="row" className="py-2 text-left font-normal">
                {f.name}
              </th>
              <td className="py-2 pl-3 text-right whitespace-nowrap">
                {f.max_advance_days}
                <span className="text-muted-foreground"> {t('rules.days')} · </span>
                {f.min_cancel_hours}
                <span className="text-muted-foreground"> {t('rules.hours')} · </span>
                {f.max_active_bookings_per_user}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}
