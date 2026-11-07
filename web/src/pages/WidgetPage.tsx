import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams, useSearchParams } from 'react-router'
import { ChevronLeft, ChevronRight, ExternalLink } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { FILTER_DEBOUNCE_MS, useAsync } from '@/hooks/useAsync'
import { usePageTitle } from '@/hooks/usePageTitle'
import { useReloadOnNewVersion } from '@/hooks/useNewVersion'
import { useBranding } from '@/hooks/useBranding'
import i18n, { languages } from '@/i18n'
import { api } from '@/lib/api'
import { addDays, formatLongDate, formatTime } from '@/lib/time'

// how many days ahead the widget lets people page through
const DAYS_AHEAD = 6

/**
 * /:slug/widget and /:slug/widget/:facility: the council's free turns, made to be
 * embedded with an <iframe> in its own website (the proxy lets only these pages
 * be framed). No top bar or footer; ?theme=light|dark and ?lang=gl|es|en|ca set
 * how it looks; each turn opens LibreQuenda in a new tab to book it.
 */
export function WidgetPage() {
  const { slug = '', facility: only } = useParams()
  const [params] = useSearchParams()
  const { t } = useTranslation('councils')
  const { branding } = useBranding()
  useReloadOnNewVersion()
  const [date, setDate] = useState<string | null>(null)
  const day = useAsync(() => api.day(slug, date ?? undefined), `widget:${slug}:${date}`, { debounceMs: FILTER_DEBOUNCE_MS })
  const today = day.data && date === null ? day.data.date : null
  const [first, setFirst] = useState<string | null>(null)
  if (today && first === null) setFirst(today)

  // the page's own look, chosen by the embedding site; never saved
  const theme = params.get('theme')
  useEffect(() => {
    const dark = theme === 'dark' || (theme !== 'light' && matchMedia('(prefers-color-scheme: dark)').matches)
    document.documentElement.classList.toggle('dark', dark)
  }, [theme])
  const lang = params.get('lang')
  useEffect(() => {
    if (lang && languages.some((l) => l.code === lang) && i18n.resolvedLanguage !== lang) void i18n.changeLanguage(lang)
  }, [lang])

  const facilities = (day.data?.facilities ?? []).filter((f) => !only || f.slug === only)
  const shown = day.data?.date ?? ''
  const timeZone = day.data?.timezone ?? 'Europe/Madrid'
  const title = only ? (facilities[0]?.name ?? '') : (day.data?.tenant.name ?? '')
  usePageTitle(t('widget.title'), title)
  const origin = window.location.origin

  return (
    // named: inside a host page (and the settings preview) it is not the page's only main
    <main id="main" aria-labelledby="widget-title" className="min-h-screen bg-background p-3 text-foreground sm:p-4">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 id="widget-title" className="text-lg leading-6 font-semibold">
          {t('widget.title')}
          {title && <span className="font-normal text-muted-foreground"> · {title}</span>}
        </h1>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            aria-label={t('widget.previous')}
            disabled={!shown || shown === first}
            onClick={() => setDate(addDays(shown, -1))}
          >
            <ChevronLeft aria-hidden />
          </Button>
          <p className="num min-w-36 text-center text-sm font-medium" aria-live="polite">
            {shown ? formatLongDate(shown) : ''}
          </p>
          <Button
            variant="ghost"
            size="icon"
            aria-label={t('widget.next')}
            disabled={!shown || !first || shown >= addDays(first, DAYS_AHEAD)}
            onClick={() => setDate(addDays(shown, 1))}
          >
            <ChevronRight aria-hidden />
          </Button>
        </div>
      </div>

      {day.error ? (
        <p className="text-sm text-muted-foreground">{t('widget.error')}</p>
      ) : !day.data ? (
        <Skeleton className="h-24 w-full" />
      ) : facilities.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t('widget.none')}</p>
      ) : (
        <ul className="grid gap-3">
          {facilities.map((f) => {
            const free = f.slots.filter((s) => s.status === 'free')
            const page = `${origin}/${slug}/${f.slug}?dia=${shown}`
            return (
              <li key={f.id} className="grid gap-2 rounded-lg border bg-card p-3">
                {!only && (
                  <h2 className="font-medium">
                    <a href={page} target="_blank" rel="noopener noreferrer" className="underline-offset-4 hover:underline">
                      {f.name}
                      <span className="sr-only"> ({t('widget.newTab')})</span>
                    </a>
                  </h2>
                )}
                {free.length === 0 ? (
                  <p className="text-sm text-muted-foreground">{t('widget.noFree')}</p>
                ) : (
                  <ul className="flex flex-wrap gap-1.5" aria-label={t('widget.freeTurns', { facility: f.name })}>
                    {free.map((s) => {
                      const time = formatTime(s.starts_at, timeZone)
                      return (
                        <li key={s.starts_at}>
                          <a
                            href={page}
                            target="_blank"
                            rel="noopener noreferrer"
                            aria-label={t('widget.book', { facility: f.name, time })}
                            className="num inline-flex min-h-9 items-center rounded-md border border-pill-positive-ink/40 bg-pill-positive-bg px-2.5 text-sm font-medium text-pill-positive-ink hover:border-pill-positive-ink"
                          >
                            {time}
                          </a>
                        </li>
                      )
                    })}
                  </ul>
                )}
              </li>
            )
          })}
        </ul>
      )}

      <p className="mt-3 text-xs text-muted-foreground">
        <a href={`${origin}/${slug}`} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 underline underline-offset-4">
          {t('widget.bookIn', { app: branding.name })}
          <ExternalLink className="size-3" aria-hidden />
          <span className="sr-only"> ({t('widget.newTab')})</span>
        </a>
      </p>
    </main>
  )
}
