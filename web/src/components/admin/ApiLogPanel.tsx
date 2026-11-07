import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import type { ApiLogFilters, ApiLogLine, ApiLogPage } from '@/lib/api'
import { cn } from '@/lib/utils'

const LEVELS = ['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL'] as const

type Props = { load: (filters: ApiLogFilters) => Promise<ApiLogPage> }

const levelColour: Record<ApiLogLine['level'], string> = {
  DEBUG: 'text-muted-foreground',
  INFO: 'text-foreground',
  WARN: 'text-amber-900 dark:text-amber-300',
  ERROR: 'text-destructive',
  FATAL: 'text-destructive font-semibold',
}

/** The API's log files: newest first, by level and text, a page at a time. */
export function ApiLogPanel({ load }: Props) {
  const { t } = useTranslation('admin')
  const [levels, setLevels] = useState<string[]>([])
  const [typed, setTyped] = useState('')
  const [query, setQuery] = useState('')
  const [lines, setLines] = useState<ApiLogLine[] | null>(null)
  const [next, setNext] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchPage = async (before?: string) => {
    setLoading(true)
    setError(null)
    try {
      const page = await load({ level: levels.join(','), q: query, before })
      setLines((shown) => (before && shown ? [...shown, ...page.lines] : page.lines))
      setNext(page.next)
      setEnabled(page.enabled)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void fetchPage()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once per set of filters
  }, [levels, query])

  // the search goes out once typing pauses
  useEffect(() => {
    const timer = setTimeout(() => setQuery(typed.trim()), 350)
    return () => clearTimeout(timer)
  }, [typed])

  return (
    <section aria-labelledby="api-log-title" className="grid gap-4 rounded-lg border bg-card p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 id="api-log-title" className="text-lg font-semibold">
            {t('apiLog.title')}
          </h2>
          <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('apiLog.help')}</p>
        </div>
        <Button type="button" variant="outline" disabled={loading} onClick={() => void fetchPage()}>
          <RefreshCw aria-hidden />
          {t('apiLog.refresh')}
        </Button>
      </div>

      <div role="search" aria-label={t('apiLog.filters')} className="flex flex-wrap items-end gap-3">
        <div className="grid w-full gap-1.5 sm:w-72">
          <Label htmlFor="api-log-q">{t('apiLog.search')}</Label>
          <Input id="api-log-q" type="search" value={typed} maxLength={100} onChange={(e) => setTyped(e.target.value)} placeholder={t('apiLog.searchPlaceholder')} />
        </div>
        <div className="grid gap-1.5">
          <span id="api-log-levels" className="text-sm font-medium">
            {t('apiLog.levels')}
          </span>
          <ToggleGroup type="multiple" variant="outline" value={levels} onValueChange={setLevels} aria-labelledby="api-log-levels" className="flex-wrap bg-card">
            {LEVELS.map((level) => (
              <ToggleGroupItem key={level} value={level} className="h-10 px-3 data-[state=on]:bg-accent data-[state=on]:text-accent-foreground">
                {level}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {!enabled ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('apiLog.disabled')}</p>
      ) : lines === null ? (
        <Skeleton className="h-40 w-full" />
      ) : lines.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('apiLog.empty')}</p>
      ) : (
        <ol aria-label={t('apiLog.caption')} className="grid max-h-[70vh] overflow-auto rounded-md border bg-background font-mono text-xs">
          {lines.map((line, i) => (
            <li key={`${line.at}-${i}`} className="grid gap-x-3 border-b px-3 py-1.5 last:border-b-0 sm:grid-cols-[auto_auto_1fr]">
              <span className="whitespace-nowrap text-muted-foreground">{line.at}</span>
              <span className={cn('whitespace-nowrap', levelColour[line.level])}>
                {line.level} <span className="text-muted-foreground">{line.caller}</span>
              </span>
              <span className="break-words whitespace-pre-wrap">{line.text}</span>
            </li>
          ))}
        </ol>
      )}

      <div aria-live="polite" className="flex items-center gap-3">
        {next && (
          <Button type="button" variant="outline" disabled={loading} onClick={() => void fetchPage(next)}>
            {loading ? t('audit.loading') : t('apiLog.older')}
          </Button>
        )}
      </div>
    </section>
  )
}
