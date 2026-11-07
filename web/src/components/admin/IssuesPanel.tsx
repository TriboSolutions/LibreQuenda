import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { CalendarX, Mail, Phone, TriangleAlert } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { useAsync } from '@/hooks/useAsync'
import { Pager } from '@/components/Pager'
import { api, type AdminFacilitySummary, type CouncilIssue, type IssueStatus } from '@/lib/api'
import { formatDateTime } from '@/lib/time'

// the council's facilities are in Galicia today
const TIME_ZONE = 'Europe/Madrid'
const STATUSES: IssueStatus[] = ['new', 'in_progress', 'solved', 'rejected']
const ALL = 'all'

const statusPill: Record<IssueStatus, string> = {
  new: 'bg-pill-accent-bg text-pill-accent-ink',
  in_progress: 'bg-pill-neutral-bg text-pill-neutral-ink',
  solved: 'bg-pill-positive-bg text-pill-positive-ink',
  rejected: 'bg-pill-negative-bg text-pill-negative-ink',
}

type Props = { slug: string; facilities: AdminFacilitySummary[]; onChanged: () => void }

/** The council's maintenance list: problems residents reported, newest first, to move along and answer. */
export function IssuesPanel({ slug, facilities, onChanged }: Props) {
  const { t } = useTranslation('admin')
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const [facility, setFacility] = useState(ALL)
  const [status, setStatus] = useState<IssueStatus | typeof ALL>('new')
  const [page, setPage] = useState(1)
  const issues = useAsync(
    () => api.council.issues(token, slug, { facility: facility === ALL ? '' : facility, status: status === ALL ? '' : status, page }),
    `issues:${slug}:${facility}:${status}:${page}`,
  )
  const list = issues.data?.issues ?? []

  return (
    <section aria-labelledby="issues-title" className="grid gap-4 rounded-lg border bg-card p-5">
      <div>
        <h2 id="issues-title" className="text-lg font-semibold">
          {t('issues.title')}
        </h2>
        <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('issues.help')}</p>
      </div>
      <div className="flex flex-wrap gap-3">
        <div className="grid min-w-0 max-w-full gap-1.5">
          <Label htmlFor="issues-facility">{t('issues.facility')}</Label>
          <Select value={facility} onValueChange={(value) => { setFacility(value); setPage(1) }}>
            <SelectTrigger id="issues-facility" className="w-64 max-w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t('issues.allFacilities')}</SelectItem>
              {facilities.map((f) => (
                <SelectItem key={f.id} value={f.id}>
                  {f.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="grid min-w-0 max-w-full gap-1.5">
          <Label htmlFor="issues-status">{t('issues.status')}</Label>
          <Select value={status} onValueChange={(value) => { setStatus(value as IssueStatus | typeof ALL); setPage(1) }}>
            <SelectTrigger id="issues-status" className="w-48 max-w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>{t('issues.allStatuses')}</SelectItem>
              {STATUSES.map((s) => (
                <SelectItem key={s} value={s}>
                  {t(`issues.statuses.${s}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {issues.error ? (
        <p className="text-destructive">{issues.error.message}</p>
      ) : !issues.data ? (
        <Skeleton className="h-40 w-full" />
      ) : list.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('issues.empty')}</p>
      ) : (
        <ul className="grid gap-3">
          {list.map((issue) => (
            <IssueItem
              key={`${issue.id}:${issue.status}:${issue.staff_note}`}
              slug={slug}
              token={token}
              issue={issue}
              onSaved={() => {
                issues.reload()
                onChanged()
              }}
            />
          ))}
        </ul>
      )}
      <Pager pagination={issues.data?.pagination} onChange={setPage} label={t('issues.pages')} />
    </section>
  )
}

function IssueItem({ slug, token, issue, onSaved }: { slug: string; token: string; issue: CouncilIssue; onSaved: () => void }) {
  const { t } = useTranslation(['admin', 'facility'])
  const [status, setStatus] = useState<IssueStatus>(issue.status)
  const [note, setNote] = useState(issue.staff_note)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const dirty = status !== issue.status || note !== issue.staff_note

  const save = async (event: FormEvent) => {
    event.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await api.council.changeIssue(token, slug, issue.id, { status, staff_note: note })
      setSaved(true)
      onSaved()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const titleId = `issue-${issue.id}`
  return (
    <li aria-labelledby={`${titleId} ${titleId}-where`} className="grid gap-4 rounded-md border p-4">
      {/* what and where, and its state */}
      <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-3">
        {issue.facility_image_url ? (
          <img src={issue.facility_image_url} alt="" className="size-12 rounded-md object-cover" loading="lazy" />
        ) : (
          <span aria-hidden className="size-12 rounded-md bg-muted" />
        )}
        <div className="grid min-w-0 gap-0.5">
          <h3 id={titleId} className="font-semibold break-words">
            {t(`facility:issue.categories.${issue.category}`)}
          </h3>
          <p id={`${titleId}-where`} className="num text-sm text-muted-foreground">
            {issue.facility_name} ·{' '}
            {t('admin:issues.reported', {
              date: formatDateTime(issue.created_at, TIME_ZONE),
              name: issue.reporter_name || t('admin:issues.someone'),
            })}
          </p>
        </div>
        <div className="flex flex-wrap justify-end gap-1.5">
          {issue.urgent && (
            <span className="inline-flex items-center gap-1 rounded-full bg-pill-negative-bg px-2 py-0.5 text-xs font-medium text-pill-negative-ink">
              <TriangleAlert className="size-3" aria-hidden />
              {t('admin:issues.urgent')}
            </span>
          )}
          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusPill[issue.status]}`}>{t(`admin:issues.statuses.${issue.status}`)}</span>
        </div>
      </div>

      {/* what the resident wrote, their photo and, if they shared it, how to reach them */}
      <div className="grid gap-2">
        <p className="whitespace-pre-line">{issue.description}</p>
        {issue.photo_url && (
          <a href={issue.photo_url} target="_blank" rel="noopener noreferrer" className="w-fit">
            <img src={issue.photo_url} alt={t('admin:issues.photo')} className="max-h-40 rounded-md object-contain" />
          </a>
        )}
        {(issue.booking_starts_at || issue.reporter_email || issue.reporter_phone) && (
          <p className="num flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
            {issue.booking_starts_at && <span>{t('admin:issues.booking', { date: formatDateTime(issue.booking_starts_at, TIME_ZONE) })}</span>}
            {issue.reporter_email && (
              <a href={`mailto:${issue.reporter_email}`} className="inline-flex items-center gap-1.5 text-foreground underline underline-offset-4">
                <Mail className="size-4 text-muted-foreground" aria-hidden />
                {issue.reporter_email}
              </a>
            )}
            {issue.reporter_phone && (
              <a href={`tel:${issue.reporter_phone.replace(/\s/g, '')}`} className="inline-flex items-center gap-1.5 text-foreground underline underline-offset-4">
                <Phone className="size-4 text-muted-foreground" aria-hidden />
                {issue.reporter_phone}
              </a>
            )}
          </p>
        )}
      </div>

      {/* the answer: a state and a note for who reported it, saved together */}
      <form onSubmit={(e) => void save(e)} className="grid gap-3 border-t pt-4">
        <div className="grid gap-1.5">
          <span id={`${titleId}-status`} className="text-sm leading-none font-medium">
            {t('admin:issues.status')}
          </span>
          <ToggleGroup
            type="single"
            variant="outline"
            value={status}
            onValueChange={(value) => value && setStatus(value as IssueStatus)}
            aria-labelledby={`${titleId}-status`}
            className="w-full flex-wrap bg-card sm:w-fit"
          >
            {STATUSES.map((s) => (
              <ToggleGroupItem key={s} value={s} className="h-10 flex-1 px-3 data-[state=on]:bg-accent data-[state=on]:text-accent-foreground sm:flex-none">
                {t(`admin:issues.statuses.${s}`)}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor={`${titleId}-note`}>{t('admin:issues.note')}</Label>
          <Textarea
            id={`${titleId}-note`}
            rows={2}
            maxLength={1000}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            aria-describedby={`${titleId}-note-help`}
          />
          <p id={`${titleId}-note-help`} className="text-xs text-muted-foreground">
            {t('admin:issues.noteHelp')}
          </p>
        </div>
        <div aria-live="polite" className="empty:hidden">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          {saved && !error && <p className="text-sm text-muted-foreground">{t('admin:issues.saved')}</p>}
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link to={`/${slug}/admin/facilities/${issue.facility_id}`} className="inline-flex items-center gap-1.5 text-sm text-primary underline-offset-4 hover:underline">
            <CalendarX className="size-4" aria-hidden />
            {t('admin:issues.close')}
          </Link>
          <Button type="submit" className="ml-auto" disabled={busy || !dirty}>
            {busy ? t('admin:issues.saving') : t('admin:issues.save')}
          </Button>
        </div>
      </form>
    </li>
  )
}
