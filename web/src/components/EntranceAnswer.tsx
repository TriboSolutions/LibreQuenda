import { useTranslation } from 'react-i18next'
import { CheckCircle2, XCircle } from 'lucide-react'
import type { EntranceResult } from '@/lib/api'
import { GRANTED } from '@/lib/entrance'

/** What the phone at the door got back, announced as it appears: in, refused and why, or what went wrong. */
export function EntranceAnswer({ result, problem }: { result: EntranceResult | null; problem: string | null }) {
  const { t } = useTranslation(['entrance', 'reader'])
  const granted = result && GRANTED.includes(result.result)
  return (
    <div role="status" aria-live="polite" className="empty:hidden">
      {result && (
        <p
          className={`flex items-start gap-2 rounded-md px-3 py-2 ${granted ? 'bg-pill-positive-bg text-pill-positive-ink' : 'bg-pill-negative-bg text-pill-negative-ink'}`}
        >
          {granted ? <CheckCircle2 className="mt-0.5 size-4 shrink-0" aria-hidden /> : <XCircle className="mt-0.5 size-4 shrink-0" aria-hidden />}
          <span>
            {t(`entrance:result.${result.result}`, { defaultValue: t('entrance:result.refused') })}
            {result.result === 'location_inaccurate' && ` ${t('entrance:inaccurateHelp')}`}
            {result.result === 'not_booked' && result.reason && ` ${t(`reader:reason.${result.reason}`, { defaultValue: t('reader:reason.other') })}`}
          </span>
        </p>
      )}
      {problem && <p className="rounded-md bg-pill-negative-bg px-3 py-2 text-pill-negative-ink">{problem}</p>}
    </div>
  )
}
