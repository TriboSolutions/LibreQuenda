import { Megaphone, Plus, Trash2, Vote, X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { useSearchParams } from 'react-router'
import { DatePicker } from '@/components/DatePicker'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { HistoryToggle } from '@/components/admin/HistoryToggle'
import { Pager } from '@/components/Pager'
import { useAsync } from '@/hooks/useAsync'
import { api, type AdminFacilitySummary, type AdminNotice, type AdminPoll } from '@/lib/api'
import { addDays, formatDateTime, formatLongDate, localDate } from '@/lib/time'

type Props = {
  slug: string
  facilities: AdminFacilitySummary[]
}

const WHOLE_COUNCIL = 'all' // the Select's value for "" (Radix refuses an empty value)
const ZONE = 'Europe/Madrid'

/**
 * The council talking to its residents: announcements on its pages (and, if it
 * wants, in their bell and inbox), and quick polls that reach them as notices.
 * Each in its own tab, the current ones first and the past in their history, so
 * the page holds what matters now. ?parte=enquisas and ?historial=1 keep where
 * one was.
 */
export function ParticipationPanel({ slug, facilities }: Props) {
  const { t } = useTranslation('admin')
  const [notice, setNotice] = useState<string | null>(null)
  const [params, setParams] = useSearchParams()
  const part = params.get('parte') === 'enquisas' ? 'polls' : 'notices'
  const history = params.get('historial') === '1'
  const go = (next: { part?: string; history?: boolean }) => {
    const query: Record<string, string> = { vista: 'participacion' }
    if ((next.part ?? part) === 'polls') query.parte = 'enquisas'
    if (next.history ?? history) query.historial = '1'
    setNotice(null)
    setParams(query, { replace: true })
  }
  return (
    <div className="grid gap-4">
      <div aria-live="polite" className="empty:hidden">
        {notice && (
          <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
            <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
          </Alert>
        )}
      </div>
      <Tabs value={part} onValueChange={(value) => go({ part: value, history: false })} className="gap-4">
        <TabsList className="w-fit">
          <TabsTrigger value="notices">{t('participation.notices.title')}</TabsTrigger>
          <TabsTrigger value="polls">{t('participation.polls.title')}</TabsTrigger>
        </TabsList>
        <TabsContent value="notices">
          <NoticesSection slug={slug} facilities={facilities} history={history} onHistory={(on) => go({ history: on })} onDone={setNotice} />
        </TabsContent>
        <TabsContent value="polls">
          <PollsSection slug={slug} facilities={facilities} history={history} onHistory={(on) => go({ history: on })} onDone={setNotice} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

type SectionProps = Props & { history: boolean; onHistory: (history: boolean) => void; onDone: (message: string) => void }

function NoticesSection({ slug, facilities, history, onHistory, onDone }: SectionProps) {
  const { t } = useTranslation('admin')
  const token = useAuth().user?.access_token ?? ''
  const [page, setPage] = useState(1)
  const [shownHistory, setShownHistory] = useState(history)
  if (shownHistory !== history) {
    // another list: from its first page
    setShownHistory(history)
    setPage(1)
  }
  const notices = useAsync(() => api.council.notices(token, slug, history, page), `council-notices:${slug}:${history}:${page}`)
  const [creating, setCreating] = useState(false)
  const [withdrawing, setWithdrawing] = useState<AdminNotice | null>(null)
  const zone = notices.data?.timezone ?? ZONE

  return (
    <section aria-labelledby="notices-title" className="grid gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="notices-title" className="text-lg font-semibold">
            {t('participation.notices.title')}
          </h2>
          <p className="max-w-[70ch] text-muted-foreground">{t('participation.notices.help')}</p>
        </div>
        <Button onClick={() => setCreating(true)}>
          <Plus />
          {t('participation.notices.create')}
        </Button>
      </div>
      <HistoryToggle history={history} onChange={onHistory} current={t('participation.notices.current')} />
      {notices.error && (
        <Alert variant="destructive">
          <AlertDescription>{notices.error.message}</AlertDescription>
        </Alert>
      )}
      {notices.loading && !notices.data ? (
        <Skeleton className="h-24 w-full" />
      ) : notices.data?.notices.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{history ? t('participation.notices.emptyHistory') : t('participation.notices.empty')}</p>
      ) : (
        <ul className="divide-y rounded-md border">
          {notices.data?.notices.map((n) => (
            <li key={n.id} className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3 px-4 py-3">
              <div className="flex min-w-0 items-start gap-3">
                <Megaphone className="mt-0.5 size-5 shrink-0 text-muted-foreground" aria-hidden />
                <div className="grid min-w-0 gap-0.5">
                  <p className="font-medium break-words">{n.title}</p>
                  <p className="line-clamp-2 text-sm whitespace-pre-line text-muted-foreground">{n.body}</p>
                  <p className="num text-sm text-muted-foreground">
                    {n.facility_name || t('participation.wholeCouncil')} ·{' '}
                    {n.active
                      ? t('participation.notices.activeUntil', { date: formatLongDate(lastDay(n.expires_at, zone)) })
                      : t('participation.notices.expired', { date: formatLongDate(lastDay(n.expires_at, zone)) })}
                    {' · '}
                    {n.notify ? t('participation.notices.told', { count: n.recipients }) : t('participation.notices.pageOnly')}
                  </p>
                </div>
              </div>
              {n.active && (
                <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setWithdrawing(n)}>
                  {t('participation.notices.withdraw')}
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
      <Pager pagination={notices.data?.pagination} onChange={setPage} label={t('participation.notices.pages')} />
      {creating && (
        <NoticeDialog
          slug={slug}
          token={token}
          zone={zone}
          facilities={facilities}
          onClose={() => setCreating(false)}
          onCreated={(created) => {
            setCreating(false)
            if (history) onHistory(false) // a new one is current
            else notices.reload()
            onDone(created.notify ? t('participation.notices.publishedTold', { count: created.recipients }) : t('participation.notices.published'))
          }}
        />
      )}
      {withdrawing && (
        <ConfirmDialog
          title={t('participation.notices.withdrawTitle', { title: withdrawing.title })}
          body={t('participation.notices.withdrawBody')}
          confirm={t('participation.notices.withdraw')}
          run={() => api.council.deleteNotice(token, slug, withdrawing.id)}
          onClose={() => setWithdrawing(null)}
          onDone={() => {
            setWithdrawing(null)
            notices.reload()
            onDone(t('participation.notices.withdrawn'))
          }}
        />
      )}
    </section>
  )
}

function PollsSection({ slug, facilities, history, onHistory, onDone }: SectionProps) {
  const { t } = useTranslation('admin')
  const token = useAuth().user?.access_token ?? ''
  const [page, setPage] = useState(1)
  const [shownHistory, setShownHistory] = useState(history)
  if (shownHistory !== history) {
    setShownHistory(history)
    setPage(1)
  }
  const polls = useAsync(() => api.council.polls(token, slug, history, page), `council-polls:${slug}:${history}:${page}`)
  const [creating, setCreating] = useState(false)
  const [closing, setClosing] = useState<AdminPoll | null>(null)
  const [deleting, setDeleting] = useState<AdminPoll | null>(null)
  const zone = polls.data?.timezone ?? ZONE

  return (
    <section aria-labelledby="polls-title" className="grid gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="polls-title" className="text-lg font-semibold">
            {t('participation.polls.title')}
          </h2>
          <p className="max-w-[70ch] text-muted-foreground">{t('participation.polls.help')}</p>
        </div>
        <Button onClick={() => setCreating(true)}>
          <Plus />
          {t('participation.polls.create')}
        </Button>
      </div>
      <HistoryToggle history={history} onChange={onHistory} current={t('participation.polls.current')} />
      {polls.error && (
        <Alert variant="destructive">
          <AlertDescription>{polls.error.message}</AlertDescription>
        </Alert>
      )}
      {polls.loading && !polls.data ? (
        <Skeleton className="h-24 w-full" />
      ) : polls.data?.polls.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{history ? t('participation.polls.emptyHistory') : t('participation.polls.empty')}</p>
      ) : (
        <ul className="divide-y rounded-md border">
          {polls.data?.polls.map((poll) => (
            <li key={poll.id} className="grid gap-3 px-4 py-3">
              <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3">
                <div className="flex min-w-0 items-start gap-3">
                  <Vote className="mt-0.5 size-5 shrink-0 text-muted-foreground" aria-hidden />
                  <div className="grid min-w-0 gap-0.5">
                    <p className="font-medium break-words">{poll.question}</p>
                    <p className="num text-sm text-muted-foreground">
                      {poll.facility_name || t('participation.wholeCouncil')} ·{' '}
                      {poll.open
                        ? t('participation.polls.openUntil', { date: formatDateTime(poll.closes_at, zone) })
                        : t('participation.polls.closedOn', { date: formatDateTime(poll.closed_at ?? poll.closes_at, zone) })}
                      {poll.residents_only && ` · ${t('participation.polls.residentsOnly')}`}
                    </p>
                  </div>
                </div>
                {/* only while open: a closed poll and its results stay in the history */}
                {poll.open && (
                  <div className="flex flex-wrap justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={() => setClosing(poll)}>
                      {t('participation.polls.close')}
                    </Button>
                    <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setDeleting(poll)} aria-label={t('participation.polls.deleteLabel', { question: poll.question })}>
                      <Trash2 />
                      {t('participation.polls.delete')}
                    </Button>
                  </div>
                )}
              </div>
              <ul className="grid gap-1.5 sm:pl-8">
                {poll.options.map((option) => {
                  const votes = option.votes ?? 0
                  const share = poll.votes > 0 ? Math.round((votes / poll.votes) * 100) : 0
                  return (
                    <li key={option.id} className="grid gap-1">
                      <div className="flex items-baseline justify-between gap-3 text-sm">
                        <span className="break-words">{option.label}</span>
                        <span className="num shrink-0 text-muted-foreground">
                          {share}% · {t('participation.polls.votes', { count: votes })}
                        </span>
                      </div>
                      <div aria-hidden className="h-1.5 overflow-hidden rounded-full bg-muted">
                        <div className="h-full rounded-full bg-primary" style={{ width: `${share}%` }} />
                      </div>
                    </li>
                  )
                })}
              </ul>
              <p className="num text-sm text-muted-foreground sm:pl-8">
                {t('participation.polls.asked', { count: poll.recipients })} · {t('participation.polls.total', { count: poll.votes })}
                {poll.open && ` · ${t('participation.polls.hiddenUntilClose')}`}
              </p>
            </li>
          ))}
        </ul>
      )}
      <Pager pagination={polls.data?.pagination} onChange={setPage} label={t('participation.polls.pages')} />
      {creating && (
        <PollDialog
          slug={slug}
          token={token}
          zone={zone}
          facilities={facilities}
          onClose={() => setCreating(false)}
          onCreated={(created) => {
            setCreating(false)
            if (history) onHistory(false) // a new one is open
            else polls.reload()
            onDone(t('participation.polls.createdAsked', { count: created.recipients }))
          }}
        />
      )}
      {closing && (
        <ConfirmDialog
          title={t('participation.polls.closeTitle', { question: closing.question })}
          body={t('participation.polls.closeBody')}
          confirm={t('participation.polls.close')}
          run={() => api.council.closePoll(token, slug, closing.id)}
          onClose={() => setClosing(null)}
          onDone={() => {
            setClosing(null)
            polls.reload()
            onDone(t('participation.polls.closed'))
          }}
        />
      )}
      {deleting && (
        <ConfirmDialog
          title={t('participation.polls.deleteTitle', { question: deleting.question })}
          body={t('participation.polls.deleteBody', { count: deleting.votes })}
          confirm={t('participation.polls.delete')}
          run={() => api.council.deletePoll(token, slug, deleting.id)}
          onClose={() => setDeleting(null)}
          onDone={() => {
            setDeleting(null)
            polls.reload()
            onDone(t('participation.polls.deleted'))
          }}
        />
      )}
    </section>
  )
}

/** expires_at is the first instant after the last day it is shown. */
function lastDay(iso: string, zone: string) {
  return localDate(new Date(Date.parse(iso) - 1).toISOString(), zone)
}

function FacilitySelect({ id, value, onChange, facilities }: { id: string; value: string; onChange: (value: string) => void; facilities: AdminFacilitySummary[] }) {
  const { t } = useTranslation('admin')
  return (
    <Select value={value || WHOLE_COUNCIL} onValueChange={(v) => onChange(v === WHOLE_COUNCIL ? '' : v)}>
      <SelectTrigger id={id} className="w-full">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={WHOLE_COUNCIL}>{t('participation.wholeCouncil')}</SelectItem>
        {facilities
          .filter((f) => f.is_active)
          .map((f) => (
            <SelectItem key={f.id} value={f.id}>
              {f.name}
            </SelectItem>
          ))}
      </SelectContent>
    </Select>
  )
}

type DialogProps<T> = {
  slug: string
  token: string
  zone: string
  facilities: AdminFacilitySummary[]
  onClose: () => void
  onCreated: (created: T) => void
}

function NoticeDialog({ slug, token, zone, facilities, onClose, onCreated }: DialogProps<AdminNotice>) {
  const { t } = useTranslation('admin')
  const [today] = useState(() => localDate(new Date().toISOString(), zone))
  const [facility, setFacility] = useState('')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [until, setUntil] = useState(() => addDays(today, 7))
  const [notify, setNotify] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    try {
      onCreated(await api.council.createNotice(token, slug, { facility_id: facility, title: title.trim(), body: body.trim(), expires_on: until, notify }))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && !saving && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('participation.notices.create')}</DialogTitle>
          <DialogDescription>{t('participation.notices.dialogHelp')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={(e) => void submit(e)} className="grid gap-4">
          <div className="grid gap-1.5">
            <Label htmlFor="notice-facility">{t('participation.about')}</Label>
            <FacilitySelect id="notice-facility" value={facility} onChange={setFacility} facilities={facilities} />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="notice-title">{t('participation.notices.titleLabel')}</Label>
            <Input id="notice-title" required maxLength={120} value={title} onChange={(e) => setTitle(e.target.value)} placeholder={t('participation.notices.titlePlaceholder')} />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="notice-body">{t('participation.notices.bodyLabel')}</Label>
            <Textarea id="notice-body" required maxLength={2000} rows={5} value={body} onChange={(e) => setBody(e.target.value)} />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="notice-until">{t('participation.notices.untilLabel')}</Label>
            <DatePicker id="notice-until" value={until} onChange={setUntil} min={today} max={addDays(today, 365)} />
          </div>
          <label htmlFor="notice-notify" className="flex items-start gap-3">
            <Checkbox id="notice-notify" checked={notify} onCheckedChange={(on) => setNotify(on === true)} className="mt-0.5" />
            <span className="grid gap-0.5">
              <span className="font-medium">{t('participation.notices.notifyLabel')}</span>
              <span className="text-sm text-muted-foreground">{facility ? t('participation.notices.notifyFacility') : t('participation.notices.notifyCouncil')}</span>
            </span>
          </label>
          <div aria-live="polite" className="empty:hidden">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={saving}>
              {t('participation.cancel')}
            </Button>
            <Button type="submit" disabled={saving || !title.trim() || !body.trim() || !until}>
              {saving ? t('participation.saving') : t('participation.notices.publish')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

const MIN_OPTIONS = 2
const MAX_OPTIONS = 6

function PollDialog({ slug, token, zone, facilities, onClose, onCreated }: DialogProps<AdminPoll>) {
  const { t } = useTranslation('admin')
  const [today] = useState(() => localDate(new Date().toISOString(), zone))
  const [facility, setFacility] = useState('')
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [closes, setCloses] = useState(() => addDays(today, 14))
  const [residentsOnly, setResidentsOnly] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const filled = options.filter((o) => o.trim()).length

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    try {
      onCreated(
        await api.council.createPoll(token, slug, {
          facility_id: facility,
          question: question.trim(),
          options: options.map((o) => o.trim()).filter(Boolean),
          residents_only: residentsOnly,
          closes_on: closes,
        }),
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && !saving && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('participation.polls.create')}</DialogTitle>
          <DialogDescription>{t('participation.polls.dialogHelp')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={(e) => void submit(e)} className="grid gap-4">
          <div className="grid gap-1.5">
            <Label htmlFor="poll-question">{t('participation.polls.questionLabel')}</Label>
            <Input id="poll-question" required maxLength={200} value={question} onChange={(e) => setQuestion(e.target.value)} placeholder={t('participation.polls.questionPlaceholder')} />
          </div>
          <fieldset className="grid gap-2">
            <legend className="mb-1.5 text-sm leading-none font-medium">{t('participation.polls.optionsLabel')}</legend>
            {options.map((option, i) => (
              <div key={i} className="flex items-center gap-2">
                <Input
                  aria-label={t('participation.polls.optionLabel', { n: i + 1 })}
                  maxLength={120}
                  value={option}
                  onChange={(e) => setOptions((all) => all.map((o, j) => (j === i ? e.target.value : o)))}
                />
                {options.length > MIN_OPTIONS && (
                  <Button type="button" variant="ghost" size="icon" aria-label={t('participation.polls.removeOption', { n: i + 1 })} onClick={() => setOptions((all) => all.filter((_, j) => j !== i))}>
                    <X />
                  </Button>
                )}
              </div>
            ))}
            {options.length < MAX_OPTIONS && (
              <Button type="button" variant="outline" size="sm" className="w-fit" onClick={() => setOptions((all) => [...all, ''])}>
                <Plus />
                {t('participation.polls.addOption')}
              </Button>
            )}
          </fieldset>
          <div className="grid gap-1.5">
            <Label htmlFor="poll-facility">{t('participation.about')}</Label>
            <FacilitySelect id="poll-facility" value={facility} onChange={setFacility} facilities={facilities} />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="poll-closes">{t('participation.polls.closesLabel')}</Label>
            <DatePicker id="poll-closes" value={closes} onChange={setCloses} min={today} max={addDays(today, 90)} />
          </div>
          <label htmlFor="poll-residents" className="flex items-start gap-3">
            <Checkbox id="poll-residents" checked={residentsOnly} onCheckedChange={(on) => setResidentsOnly(on === true)} className="mt-0.5" />
            <span className="grid gap-0.5">
              <span className="font-medium">{t('participation.polls.residentsOnlyLabel')}</span>
              <span className="text-sm text-muted-foreground">{t('participation.polls.residentsOnlyHelp')}</span>
            </span>
          </label>
          <div aria-live="polite" className="empty:hidden">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose} disabled={saving}>
              {t('participation.cancel')}
            </Button>
            <Button type="submit" disabled={saving || !question.trim() || filled < MIN_OPTIONS || !closes}>
              {saving ? t('participation.saving') : t('participation.polls.publish')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

type ConfirmProps = {
  title: string
  body: string
  confirm: string
  run: () => Promise<void>
  onClose: () => void
  onDone: () => void
}

function ConfirmDialog({ title, body, confirm, run, onClose, onDone }: ConfirmProps) {
  const { t } = useTranslation('admin')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const go = async () => {
    setSaving(true)
    setError(null)
    try {
      await run()
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setSaving(false)
    }
  }
  return (
    <Dialog open onOpenChange={(open) => !open && !saving && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="break-words">{title}</DialogTitle>
          <DialogDescription>{body}</DialogDescription>
        </DialogHeader>
        <div aria-live="polite" className="empty:hidden">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={saving}>
            {t('participation.keep')}
          </Button>
          <Button variant="destructive" disabled={saving} onClick={() => void go()}>
            {confirm}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
