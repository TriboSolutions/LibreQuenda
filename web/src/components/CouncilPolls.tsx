import { CircleCheck, Vote } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { useMe } from '@/hooks/useMe'
import { useVerifyEmail } from '@/hooks/useVerifyEmail'
import { api, ApiError, type MyPoll } from '@/lib/api'
import { formatDateTime } from '@/lib/time'

// the councils are in Galicia
const TIME_ZONE = 'Europe/Madrid'

/**
 * The councils' polls for the person, the open ones first: one vote each, which
 * can change while the poll is open; the results show once it closes. focus:
 * the poll a notice opened, scrolled to and marked.
 */
export function PollList({ polls, onVoted, focus }: { polls: MyPoll[]; onVoted: () => void; focus?: string }) {
  return (
    <ul className="grid gap-3 md:grid-cols-2">
      {polls.map((poll) => (
        <PollCard key={poll.id} poll={poll} onVoted={onVoted} focused={poll.id === focus} />
      ))}
    </ul>
  )
}

function PollCard({ poll, onVoted, focused }: { poll: MyPoll; onVoted: () => void; focused: boolean }) {
  const { t } = useTranslation('home')
  const token = useAuth().user?.access_token
  const { me } = useMe()
  const verify = useVerifyEmail()
  const [choice, setChoice] = useState(poll.my_option)
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const titleId = `poll-${poll.id}`
  const where = poll.facility_name ? (
    <>
      {poll.tenant_name} ·{' '}
      <Link to={`/${poll.tenant_slug}/${poll.facility_slug}`} className="underline underline-offset-4">
        {poll.facility_name}
      </Link>
    </>
  ) : (
    poll.tenant_name
  )

  const vote = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!token || !choice) return
    setBusy(true)
    setError(null)
    setSaved(false)
    try {
      await api.vote(token, poll.id, choice)
      setSaved(true)
      onVoted()
    } catch (e) {
      setError(e instanceof ApiError || e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <li
      id={titleId + '-card'}
      ref={(el) => {
        if (el && focused) el.scrollIntoView({ block: 'center' })
      }}
      className={`grid scroll-mt-24 content-start gap-3 rounded-lg border bg-card p-4 ${focused ? 'border-primary ring-2 ring-primary/30' : ''}`}
    >
      <div className="flex items-start gap-3">
        <Vote className="mt-0.5 size-5 shrink-0 text-muted-foreground" aria-hidden />
        <div className="grid min-w-0 gap-0.5">
          <h2 id={titleId} className="leading-5 font-semibold break-words">
            {poll.question}
          </h2>
          <p className="text-sm text-muted-foreground">{where}</p>
          <p className="num text-sm text-muted-foreground">
            {poll.open ? t('polls.closes', { date: formatDateTime(poll.closes_at, TIME_ZONE) }) : t('polls.closed', { date: formatDateTime(poll.closes_at, TIME_ZONE) })}
            {poll.residents_only && ` · ${t('polls.residentsOnly')}`}
          </p>
        </div>
      </div>

      {poll.open ? (
        <form onSubmit={(e) => void vote(e)} className="grid gap-3">
          <fieldset className="grid gap-2" aria-labelledby={titleId}>
            {poll.options.map((option) => (
              <label key={option.id} className="flex min-h-11 cursor-pointer items-center gap-3 rounded-md border px-3 py-2 has-[:checked]:border-primary has-[:checked]:bg-primary/5 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring">
                <input
                  type="radio"
                  name={titleId}
                  value={option.id}
                  checked={choice === option.id}
                  onChange={() => setChoice(option.id)}
                  className="size-4 accent-primary"
                />
                <span className="break-words">{option.label}</span>
              </label>
            ))}
          </fieldset>
          <div aria-live="polite" className="empty:hidden">
            {saved && (
              <p className="flex items-center gap-2 text-sm font-medium text-pill-positive-ink">
                <CircleCheck className="size-4" aria-hidden />
                {t('polls.saved')}
              </p>
            )}
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </div>
          {me && !me.email_verified ? (
            <div className="grid gap-2">
              <p className="text-sm text-muted-foreground">{t('polls.verifyEmail')}</p>
              <Button type="button" variant="outline" className="w-fit" onClick={verify}>
                {t('polls.verify')}
              </Button>
            </div>
          ) : (
            <Button type="submit" className="w-fit" disabled={busy || !choice || choice === poll.my_option}>
              {poll.my_option ? t('polls.change') : t('polls.vote')}
            </Button>
          )}
          {poll.my_option && <p className="text-sm text-muted-foreground">{t('polls.resultsLater')}</p>}
        </form>
      ) : (
        <Results poll={poll} />
      )}
    </li>
  )
}

/** Each answer's share, the winner first in the eye but the order kept. */
function Results({ poll }: { poll: MyPoll }) {
  const { t } = useTranslation('home')
  const total = poll.votes ?? 0
  const top = Math.max(0, ...poll.options.map((o) => o.votes ?? 0))
  return (
    <div className="grid gap-2">
      <ul className="grid gap-2">
        {poll.options.map((option) => {
          const votes = option.votes ?? 0
          const share = total > 0 ? Math.round((votes / total) * 100) : 0
          return (
            <li key={option.id} className="grid gap-1">
              <div className="flex items-baseline justify-between gap-3">
                <span className={votes === top && total > 0 ? 'font-semibold break-words' : 'break-words'}>
                  {option.label}
                  {poll.my_option === option.id && <span className="ml-1.5 text-sm font-normal text-muted-foreground">({t('polls.yours')})</span>}
                </span>
                <span className="num shrink-0 text-sm text-muted-foreground">
                  {share}% · {t('polls.votes', { count: votes })}
                </span>
              </div>
              <div aria-hidden className="h-2 overflow-hidden rounded-full bg-muted">
                <div className="h-full rounded-full bg-primary" style={{ width: `${share}%` }} />
              </div>
            </li>
          )
        })}
      </ul>
      <p className="num text-sm text-muted-foreground">{t('polls.total', { count: total })}</p>
    </div>
  )
}
