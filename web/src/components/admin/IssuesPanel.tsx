import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link } from 'react-router'
import { CalendarX, Mail, TriangleAlert } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import { useAsync } from '@/hooks/useAsync'
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
  const [status, setStatus] = useState<IssueStatus | typeof ALL>(ALL)
  const issues = useAsync(
    () => api.council.issues(token, slug, { facility: facility === ALL ? '' : facility, status: status === ALL ? '' : status }),
    `issues:${slug}:${facility}:${status}`,
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
        <div className="grid gap-1.5">
          <Label htmlFor="issues-facility">{t('issues.facility')}</Label>
          <Select value={facility} onValueChange={setFacility}>
            <SelectTrigger id="issues-facility" className="w-64">
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
        <div className="grid gap-1.5">
          <Label htmlFor="issues-status">{t('issues.status')}</Label>
          <Select value={status} onValueChange={(value) => setStatus(value as IssueStatus | typeof ALL)}>
            <SelectTrigger id="issues-status" className="w-48">
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
    <li aria-labelledby={titleId} className="grid gap-3 rounded-md border p-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="grid gap-1">
          <h3 id={titleId} className="font-semibold">
            {issue.facility_name} · {t(`facility:issue.categories.${issue.category}`)}
          </h3>
          <p className="num text-sm text-muted-foreground">
            {t('admin:issues.reported', {
              date: formatDateTime(issue.created_at, TIME_ZONE),
              name: issue.reporter_name || t('admin:issues.someone'),
            })}
            {issue.booking_starts_at && ` · ${t('admin:issues.booking', { date: formatDateTime(issue.booking_starts_at, TIME_ZONE) })}`}
          </p>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {issue.urgent && (
            <span className="inline-flex items-center gap-1 rounded-full bg-pill-negative-bg px-2 py-0.5 text-xs font-medium text-pill-negative-ink">
              <TriangleAlert className="size-3" aria-hidden />
              {t('admin:issues.urgent')}
            </span>
          )}
          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusPill[issue.status]}`}>{t(`admin:issues.statuses.${issue.status}`)}</span>
        </div>
      </div>
      <p className="whitespace-pre-line">{issue.description}</p>
      {issue.photo_url && (
        <a href={issue.photo_url} target="_blank" rel="noopener noreferrer" className="w-fit">
          <img src={issue.photo_url} alt={t('admin:issues.photo')} className="max-h-48 rounded-md object-contain" />
        </a>
      )}
      {issue.reporter_email && (
        <p className="flex items-center gap-1.5 text-sm">
          <Mail className="size-4 text-muted-foreground" aria-hidden />
          <a href={`mailto:${issue.reporter_email}`} className="underline underline-offset-4">
            {issue.reporter_email}
          </a>
        </p>
      )}

      <form onSubmit={(e) => void save(e)} className="grid gap-3 border-t pt-3 sm:grid-cols-[12rem_1fr_auto] sm:items-end">
        <div className="grid gap-1.5">
          <Label htmlFor={`${titleId}-status`}>{t('admin:issues.status')}</Label>
          <Select value={status} onValueChange={(value) => setStatus(value as IssueStatus)}>
            <SelectTrigger id={`${titleId}-status`} className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUSES.map((s) => (
                <SelectItem key={s} value={s}>
                  {t(`admin:issues.statuses.${s}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
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
        <Button type="submit" disabled={busy || !dirty}>
          {busy ? t('admin:issues.saving') : t('admin:issues.save')}
        </Button>
      </form>
      <div aria-live="polite" className="empty:hidden">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        {saved && !error && <p className="text-sm text-muted-foreground">{t('admin:issues.saved')}</p>}
      </div>
      <p>
        <Link to={`/${slug}/admin/facilities/${issue.facility_id}`} className="inline-flex items-center gap-1.5 text-sm text-primary underline-offset-4 hover:underline">
          <CalendarX className="size-4" aria-hidden />
          {t('admin:issues.close')}
        </Link>
      </p>
    </li>
  )
}
