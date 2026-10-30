import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Download } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { DatePicker } from '@/components/DatePicker'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import type { AuditEntry, AuditFilters, AuditPage } from '@/lib/api'
import { formatDateTime } from '@/lib/time'

const TIME_ZONE = 'Europe/Madrid'
const ANY = '_any'

// the action families to filter by: the first part of the codes
const FAMILIES = ['facility', 'closure', 'checkin', 'reader', 'booking_window', 'booking_policy', 'issue', 'staff', 'council', 'admin', 'superadmin', 'tag', 'tag_category', 'branding', 'smtp', 'indicators', 'account', 'other']

type Props = {
  load: (filters: AuditFilters) => Promise<AuditPage>
  download: (filters: AuditFilters) => Promise<void>
  councils?: { slug: string; name: string }[] // superadmins: filter by council, and show it per line
}

/** The audit log as a table: newest first, filtered, a page at a time, downloadable as CSV. */
export function AuditLogPanel({ load, download, councils }: Props) {
  const { t } = useTranslation('admin')
  const [filters, setFilters] = useState<AuditFilters>({})
  const [entries, setEntries] = useState<AuditEntry[] | null>(null)
  const [next, setNext] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchPage = async (current: AuditFilters, before?: string) => {
    setLoading(true)
    setError(null)
    try {
      const page = await load({ ...current, before })
      setEntries((shown) => (before && shown ? [...shown, ...page.entries] : page.entries))
      setNext(page.next)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  // a new set of filters starts from the newest entries
  useEffect(() => {
    void fetchPage(filters)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once per set of filters
  }, [filters])

  const set = (key: keyof AuditFilters, value: string) => setFilters((f) => ({ ...f, [key]: value === ANY ? '' : value }))

  return (
    <section aria-labelledby="audit-title" className="grid gap-4 rounded-lg border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 id="audit-title" className="text-lg font-semibold">
            {t('audit.title')}
          </h2>
          <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('audit.help')}</p>
        </div>
        <Button type="button" variant="outline" onClick={() => void download(filters).catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))}>
          <Download aria-hidden />
          {t('audit.download')}
        </Button>
      </div>

      <div role="search" aria-label={t('audit.filters')} className="flex flex-wrap items-end gap-3">
        <div className="grid gap-1.5">
          <Label htmlFor="audit-action">{t('audit.action')}</Label>
          <Select value={filters.action || ANY} onValueChange={(v) => set('action', v)}>
            <SelectTrigger id="audit-action" className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ANY}>{t('audit.anyAction')}</SelectItem>
              {FAMILIES.map((family) => (
                <SelectItem key={family} value={family}>
                  {t(`audit.families.${family}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {councils && (
          <div className="grid gap-1.5">
            <Label htmlFor="audit-council">{t('audit.council')}</Label>
            <Select value={filters.council || ANY} onValueChange={(v) => set('council', v)}>
              <SelectTrigger id="audit-council" className="w-56">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ANY}>{t('audit.anyCouncil')}</SelectItem>
                {councils.map((c) => (
                  <SelectItem key={c.slug} value={c.slug}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        <div className="grid gap-1.5">
          <Label htmlFor="audit-from">{t('audit.from')}</Label>
          <DatePicker id="audit-from" value={filters.from ?? ''} max={filters.to || undefined} onChange={(value) => set('from', value)} placeholder={t('audit.anyDate')} className="w-full sm:w-64" />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="audit-to">{t('audit.to')}</Label>
          <DatePicker id="audit-to" value={filters.to ?? ''} min={filters.from || undefined} onChange={(value) => set('to', value)} placeholder={t('audit.anyDate')} className="w-full sm:w-64" />
        </div>
        {(filters.from || filters.to) && (
          <Button type="button" variant="ghost" onClick={() => setFilters((current) => ({ ...current, from: '', to: '' }))}>
            {t('audit.clearDates')}
          </Button>
        )}
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {entries === null ? (
        <Skeleton className="h-40 w-full" />
      ) : entries.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('audit.empty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[720px] text-sm">
            <caption className="sr-only">{t('audit.caption')}</caption>
            <thead>
              <tr className="text-left text-xs text-muted-foreground">
                <th scope="col" className="py-2 pr-4 font-normal">
                  {t('audit.colWhen')}
                </th>
                <th scope="col" className="py-2 pr-4 font-normal">
                  {t('audit.colWho')}
                </th>
                {councils && (
                  <th scope="col" className="py-2 pr-4 font-normal">
                    {t('audit.council')}
                  </th>
                )}
                <th scope="col" className="py-2 pr-4 font-normal">
                  {t('audit.colWhat')}
                </th>
                <th scope="col" className="py-2 pr-4 font-normal">
                  {t('audit.colOn')}
                </th>
                <th scope="col" className="py-2 font-normal">
                  {t('audit.colChanges')}
                </th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr key={entry.id} className="border-t align-top">
                  <td className="num py-2.5 pr-4 whitespace-nowrap">{formatDateTime(entry.at, TIME_ZONE)}</td>
                  <td className="py-2.5 pr-4">
                    <span className="block font-medium">
                      {entry.actor_role === 'system' ? t('audit.system') : entry.actor_deleted ? t('audit.deletedAccount') : entry.actor_name}
                    </span>
                    <span className="block text-xs text-muted-foreground">{t(`audit.roles.${entry.actor_role}`)}</span>
                  </td>
                  {councils && <td className="py-2.5 pr-4">{entry.tenant_name || '—'}</td>}
                  <td className="py-2.5 pr-4">{t(`audit.actions.${entry.action}`, { defaultValue: entry.action })}</td>
                  <td className="py-2.5 pr-4">
                    <Target entry={entry} />
                  </td>
                  <td className="py-2.5 break-words text-muted-foreground">{entry.summary || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div aria-live="polite" className="flex items-center gap-3">
        {next && (
          <Button type="button" variant="outline" disabled={loading} onClick={() => void fetchPage(filters, next)}>
            {loading ? t('audit.loading') : t('audit.more')}
          </Button>
        )}
      </div>
    </section>
  )
}

function Target({ entry }: { entry: AuditEntry }) {
  const { t } = useTranslation('admin')
  if (entry.target_type === 'user') return <>{entry.target_name || t('audit.deletedAccount')}</>
  if (!entry.target_name) return <>—</>
  if (entry.target_type === 'facility' && entry.target_exists && entry.tenant_slug) {
    return (
      <Link to={`/${entry.tenant_slug}/admin/facilities/${entry.target_id}`} className="text-primary underline underline-offset-4">
        {entry.target_name}
      </Link>
    )
  }
  return <>{entry.target_name}</>
}
