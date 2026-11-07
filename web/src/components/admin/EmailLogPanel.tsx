import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Download } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DatePicker } from '@/components/DatePicker'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import type { EmailLogEntry, EmailLogFilters, EmailLogPage } from '@/lib/api'
import { formatDateTime } from '@/lib/time'

const TIME_ZONE = 'Europe/Madrid'
const ANY = '_any'

type Props = {
  load: (filters: EmailLogFilters) => Promise<EmailLogPage>
  download: (filters: EmailLogFilters) => Promise<void>
  councils: { slug: string; name: string }[]
}

/** Every email LibreQuenda tried to send: searched, filtered, newest first, a page at a time, downloadable as CSV. */
export function EmailLogPanel({ load, download, councils }: Props) {
  const { t } = useTranslation('admin')
  const [filters, setFilters] = useState<EmailLogFilters>({})
  const [typed, setTyped] = useState('')
  const [page, setPage] = useState<EmailLogPage | null>(null)
  const [entries, setEntries] = useState<EmailLogEntry[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchPage = async (current: EmailLogFilters, before?: string) => {
    setLoading(true)
    setError(null)
    try {
      const result = await load({ ...current, before })
      setEntries((shown) => (before && shown ? [...shown, ...result.entries] : result.entries))
      setPage(result)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  // a new set of filters starts from the newest emails
  useEffect(() => {
    void fetchPage(filters)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once per set of filters
  }, [filters])

  // the search goes out once typing pauses
  useEffect(() => {
    const timer = setTimeout(() => setFilters((f) => (f.q === typed.trim() || (!f.q && !typed.trim()) ? f : { ...f, q: typed.trim() })), 350)
    return () => clearTimeout(timer)
  }, [typed])

  const set = (key: keyof EmailLogFilters, value: string) => setFilters((f) => ({ ...f, [key]: value === ANY ? '' : value }))

  return (
    <section aria-labelledby="email-log-title" className="grid gap-4 rounded-lg border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 id="email-log-title" className="text-lg font-semibold">
            {t('emailLog.title')}
          </h2>
          <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('emailLog.help')}</p>
          {page && (
            <p className="mt-2 text-sm">
              {t('emailLog.last24h', { sent: page.sent_24h, failed: page.failed_24h })}
            </p>
          )}
        </div>
        <Button type="button" variant="outline" onClick={() => void download(filters).catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))}>
          <Download aria-hidden />
          {t('emailLog.download')}
        </Button>
      </div>

      <div role="search" aria-label={t('emailLog.filters')} className="flex flex-wrap items-end gap-3">
        <div className="grid w-full gap-1.5 sm:w-72">
          <Label htmlFor="email-log-q">{t('emailLog.search')}</Label>
          <Input id="email-log-q" type="search" value={typed} maxLength={100} onChange={(e) => setTyped(e.target.value)} placeholder={t('emailLog.searchPlaceholder')} />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="email-log-status">{t('emailLog.status')}</Label>
          <Select value={filters.status || ANY} onValueChange={(v) => set('status', v)}>
            <SelectTrigger id="email-log-status" className="w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ANY}>{t('emailLog.anyStatus')}</SelectItem>
              <SelectItem value="sent">{t('emailLog.sent')}</SelectItem>
              <SelectItem value="failed">{t('emailLog.failed')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="email-log-kind">{t('emailLog.kind')}</Label>
          <Select value={filters.kind || ANY} onValueChange={(v) => set('kind', v)}>
            <SelectTrigger id="email-log-kind" className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ANY}>{t('emailLog.anyKind')}</SelectItem>
              {(page?.kinds ?? []).map((kind) => (
                <SelectItem key={kind} value={kind}>
                  {t(`emailLog.kinds.${kind}`, { defaultValue: kind })}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="email-log-council">{t('audit.council')}</Label>
          <Select value={filters.council || ANY} onValueChange={(v) => set('council', v)}>
            <SelectTrigger id="email-log-council" className="w-56">
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
        <div className="grid gap-1.5">
          <Label htmlFor="email-log-from">{t('audit.from')}</Label>
          <DatePicker id="email-log-from" value={filters.from ?? ''} max={filters.to || undefined} onChange={(value) => set('from', value)} placeholder={t('audit.anyDate')} className="w-full sm:w-64" />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="email-log-to">{t('audit.to')}</Label>
          <DatePicker id="email-log-to" value={filters.to ?? ''} min={filters.from || undefined} onChange={(value) => set('to', value)} placeholder={t('audit.anyDate')} className="w-full sm:w-64" />
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
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('emailLog.empty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[820px] text-sm">
            <caption className="sr-only">{t('emailLog.caption')}</caption>
            <thead>
              <tr className="text-left text-xs text-muted-foreground">
                <th scope="col" className="py-2 pr-4 font-normal">{t('audit.colWhen')}</th>
                <th scope="col" className="py-2 pr-4 font-normal">{t('emailLog.status')}</th>
                <th scope="col" className="py-2 pr-4 font-normal">{t('emailLog.colTo')}</th>
                <th scope="col" className="py-2 pr-4 font-normal">{t('emailLog.colSubject')}</th>
                <th scope="col" className="py-2 pr-4 font-normal">{t('audit.council')}</th>
                <th scope="col" className="py-2 font-normal">{t('emailLog.colVia')}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr key={entry.id} className="border-t align-top">
                  <td className="num py-2.5 pr-4 whitespace-nowrap">{formatDateTime(entry.at, TIME_ZONE)}</td>
                  <td className="py-2.5 pr-4">
                    <Badge variant={entry.status === 'failed' ? 'destructive' : 'secondary'}>{t(`emailLog.${entry.status}`)}</Badge>
                    <span className="mt-1 block text-xs text-muted-foreground">{t(`emailLog.kinds.${entry.kind}`, { defaultValue: entry.kind || '—' })}</span>
                  </td>
                  <td className="py-2.5 pr-4 break-all">{entry.recipient}</td>
                  <td className="py-2.5 pr-4 break-words">
                    {entry.subject || '—'}
                    {entry.error && <span className="mt-1 block text-xs break-words text-destructive">{entry.error}</span>}
                  </td>
                  <td className="py-2.5 pr-4">{entry.tenant_name || '—'}</td>
                  <td className="py-2.5 break-all text-muted-foreground">
                    <span className="block">{entry.sender || '—'}</span>
                    <span className="block text-xs">{entry.relay}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div aria-live="polite" className="flex items-center gap-3">
        {page?.next && (
          <Button type="button" variant="outline" disabled={loading} onClick={() => void fetchPage(filters, page.next ?? undefined)}>
            {loading ? t('audit.loading') : t('audit.more')}
          </Button>
        )}
      </div>
    </section>
  )
}
