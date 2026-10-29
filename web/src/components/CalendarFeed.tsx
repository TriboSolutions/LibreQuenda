import { CalendarPlus, Check, Copy, ExternalLink, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { SettingsRow } from '@/components/SettingsRow'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useBranding } from '@/hooks/useBranding'
import { api, type CalendarFeed as Feed } from '@/lib/api'

/*
  The user's bookings as an .ics feed (iCalendar, RFC 5545) for any calendar app.
  The address carries a secret token, since calendar apps cannot sign in: a new
  address revokes the old one.
*/
/*
  One-click subscribe links. Google and Outlook fetch the feed from their own
  servers, so they only work once the address is public over https (not on
  localhost). Personal and work Microsoft accounts live on different hosts.
*/
function subscribeLinks(url: string, name: string) {
  const webcal = url.replace(/^https?:/, 'webcal:')
  const outlook = (host: string) => `https://${host}/calendar/0/addfromweb?url=${encodeURIComponent(url)}&name=${encodeURIComponent(name)}`
  return {
    google: `https://calendar.google.com/calendar/render?cid=${encodeURIComponent(webcal)}`,
    outlook: outlook('outlook.live.com'),
    outlookWork: outlook('outlook.office.com'),
    // webcal:// opens the subscription in the system's calendar app
    apple: webcal,
  }
}

export function CalendarFeed() {
  const { t } = useTranslation('profile')
  const { branding } = useBranding()
  const auth = useAuth()
  const token = auth.user?.access_token
  const loaded = useAsync(token ? () => api.calendarFeed.get(token) : null, `calendar:${token ?? ''}`)
  const [feed, setFeed] = useState<Feed | null>(null)
  const [busy, setBusy] = useState(false)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const current = feed ?? loaded.data
  const url = current?.url ?? ''

  const run = async (action: () => Promise<Feed>) => {
    setBusy(true)
    setError(null)
    setCopied(false)
    try {
      setFeed(await action())
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }
  const reset = () => run(() => api.calendarFeed.reset(token!))
  const turnOff = () => run(() => api.calendarFeed.turnOff(token!).then(() => ({ url: '' })))
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
    } catch {
      // no clipboard permission: the field stays selectable
    }
  }

  let body
  if (!current) {
    body = loaded.error ? (
      <Alert variant="destructive">
        <AlertDescription>{loaded.error.message}</AlertDescription>
      </Alert>
    ) : (
      <Skeleton className="h-20 w-full" />
    )
  } else if (!url) {
    body = (
      <div>
        <Button onClick={() => void reset()} disabled={busy}>
          <CalendarPlus aria-hidden />
          {t('calendar.turnOn')}
        </Button>
      </div>
    )
  } else {
    const links = subscribeLinks(url, branding.name)
    const web = { target: '_blank', rel: 'noopener noreferrer' }
    body = (
      <>
        <div className="grid gap-2">
          {/* stacked on phones, where the longer labels need to wrap */}
          <div className="grid gap-2 sm:flex sm:flex-wrap">
            <Button asChild className="h-auto min-h-11 whitespace-normal">
              <a href={links.google} {...web}>
                <ExternalLink aria-hidden />
                {t('calendar.google')}
              </a>
            </Button>
            <Button asChild className="h-auto min-h-11 whitespace-normal">
              <a href={links.outlook} {...web}>
                <ExternalLink aria-hidden />
                {t('calendar.outlook')}
              </a>
            </Button>
            <Button asChild className="h-auto min-h-11 whitespace-normal">
              <a href={links.outlookWork} {...web}>
                <ExternalLink aria-hidden />
                {t('calendar.outlookWork')}
              </a>
            </Button>
            <Button asChild className="h-auto min-h-11 whitespace-normal">
              <a href={links.apple}>
                <CalendarPlus aria-hidden />
                {t('calendar.apple')}
              </a>
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">{t('calendar.googleSlow')}</p>
        </div>
        <div className="grid gap-2">
          <Label htmlFor="calendar-url">{t('calendar.address')}</Label>
          <div className="flex gap-2">
            <Input id="calendar-url" readOnly value={url} onFocus={(e) => e.currentTarget.select()} className="min-w-0 font-mono text-xs" />
            <Button variant="outline" onClick={() => void copy()} className="shrink-0">
              {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
              {copied ? t('calendar.copied') : t('calendar.copy')}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">{t('calendar.private')}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => void reset()} disabled={busy} title={t('calendar.resetHint')}>
            <RefreshCw aria-hidden />
            {t('calendar.reset')}
          </Button>
          <Button variant="ghost" onClick={() => void turnOff()} disabled={busy}>
            {t('calendar.turnOff')}
          </Button>
        </div>
      </>
    )
  }

  return (
    <SettingsRow id="sec-calendar" title={t('calendar.title')} description={t('calendar.help')}>
      <div className="grid gap-4 rounded-lg border bg-card p-5">
        <p className="text-muted-foreground">{t('calendar.how')}</p>
        <div aria-live="polite" className="empty:hidden">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>
        {body}
      </div>
    </SettingsRow>
  )
}
