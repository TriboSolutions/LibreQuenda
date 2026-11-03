import { useState, type ChangeEvent, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { SettingsRow } from '@/components/SettingsRow'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useBranding } from '@/hooks/useBranding'
import { api, type CouncilSmtp, type CouncilSmtpInput } from '@/lib/api'

type Props = { slug: string; token: string }

const empty: CouncilSmtpInput = {
  host: '',
  port: 587,
  username: '',
  password: '',
  from: '',
}

function toInput(smtp: CouncilSmtp): CouncilSmtpInput {
  return smtp.configured
    ? {
        host: smtp.host,
        port: smtp.port,
        username: smtp.username,
        password: '',
        from: smtp.from,
      }
    : empty
}

/** The council's own SMTP relay, in the council sheet. Without one, its emails go through the platform's; with neither, they wait. */
export function SmtpSettings({ slug, token }: Props) {
  const { t } = useTranslation(['admin', 'common'])
  const { branding } = useBranding()
  const smtp = useAsync(() => api.council.smtp(token, slug), `smtp:${slug}`)
  const [form, setForm] = useState<CouncilSmtpInput>(empty)
  const [busy, setBusy] = useState<'save' | 'test' | 'remove' | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [confirmRemove, setConfirmRemove] = useState(false)
  // the form starts from each load of the saved settings
  const [loaded, setLoaded] = useState<CouncilSmtp | null>(null)
  if (smtp.data && smtp.data !== loaded) {
    setLoaded(smtp.data)
    setForm(toInput(smtp.data))
  }

  const set = (field: keyof CouncilSmtpInput) => (e: ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({
      ...f,
      [field]: field === 'port' ? Number(e.target.value) || 0 : e.target.value,
    }))

  const run = async (kind: 'save' | 'test' | 'remove', action: () => Promise<string>) => {
    setBusy(kind)
    setError(null)
    setNotice(null)
    try {
      setNotice(await action())
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  const save = (event: FormEvent) => {
    event.preventDefault()
    void run('save', async () => {
      await api.council.saveSmtp(token, slug, {
        ...form,
        host: form.host.trim(),
        username: form.username.trim(),
        from: form.from.trim(),
      })
      smtp.reload()
      return t('admin:smtp.saved')
    })
  }

  const test = () =>
    void run('test', async () => {
      const result = await api.council.testSmtp(token, slug)
      return t(result.via === 'council' ? 'admin:smtp.testSentCouncil' : 'admin:smtp.testSentPlatform', { email: result.to })
    })

  const remove = () => {
    setConfirmRemove(false)
    void run('remove', async () => {
      await api.council.deleteSmtp(token, slug)
      smtp.reload()
      return t('admin:smtp.removed')
    })
  }

  const data = smtp.data
  let status = null
  if (data) {
    const [pill, label] = data.configured
      ? ['bg-pill-positive-bg text-pill-positive-ink', t('admin:smtp.statusCouncil')]
      : data.env_relay
        ? ['bg-pill-neutral-bg text-pill-neutral-ink', t('admin:smtp.statusPlatform')]
        : ['bg-pill-negative-bg text-pill-negative-ink', t('admin:smtp.statusNone')]
    status = <span className={`rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ${pill}`}>{label}</span>
  }

  return (
    <SettingsRow id="sheet-smtp" title={t('admin:smtp.title')} description={t('admin:smtp.help', { app: branding.name })}>
      <div className="grid content-start gap-4 rounded-lg border bg-card p-5">
        {status && <div>{status}</div>}

        {data && !data.configured && !data.env_relay && (
          <Alert variant="destructive">
            <AlertDescription>{t('admin:smtp.noneWarning')}</AlertDescription>
          </Alert>
        )}
        {data?.unreadable && (
          <Alert variant="destructive">
            <AlertDescription>{t('admin:smtp.unreadable')}</AlertDescription>
          </Alert>
        )}
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        {notice && (
          <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
            <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
          </Alert>
        )}

        {smtp.error ? (
          <div className="flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
            <p>{smtp.error.message}</p>
            <Button variant="ghost" size="sm" onClick={smtp.reload}>
              {t('common:retry')}
            </Button>
          </div>
        ) : !data ? (
          <Skeleton className="h-64 w-full" />
        ) : (
          <form onSubmit={save} className="grid gap-4">
            <div className="grid gap-4 sm:grid-cols-[1fr_8rem]">
              <div className="grid gap-1.5">
                <Label htmlFor="smtp-host">{t('admin:smtp.host')}</Label>
                <Input
                  id="smtp-host"
                  value={form.host}
                  onChange={set('host')}
                  placeholder="smtp.concello.gal"
                  maxLength={255}
                  required
                  autoComplete="off"
                />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="smtp-port">{t('admin:smtp.port')}</Label>
                <Input
                  id="smtp-port"
                  type="number"
                  inputMode="numeric"
                  min={1}
                  max={65535}
                  value={form.port || ''}
                  onChange={set('port')}
                  className="num"
                />
              </div>
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="smtp-from">{t('admin:smtp.from')}</Label>
              <Input
                id="smtp-from"
                value={form.from}
                onChange={set('from')}
                placeholder={t('admin:smtp.fromPlaceholder')}
                maxLength={255}
                required
              />
              <p className="text-muted-foreground">{t('admin:smtp.fromHelp')}</p>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-1.5">
                <Label htmlFor="smtp-username">{t('admin:smtp.username')}</Label>
                <Input id="smtp-username" value={form.username} onChange={set('username')} maxLength={255} autoComplete="off" />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="smtp-password">{t('admin:smtp.password')}</Label>
                <Input
                  id="smtp-password"
                  type="password"
                  value={form.password}
                  onChange={set('password')}
                  placeholder={data.has_password ? t('admin:smtp.passwordKept') : ''}
                  disabled={form.username.trim() === ''}
                  autoComplete="new-password"
                />
              </div>
            </div>
            <p className="text-muted-foreground">{t('admin:smtp.tlsHelp')}</p>
            <div className="flex flex-wrap gap-3">
              <Button type="submit" disabled={busy !== null}>
                {busy === 'save' ? t('admin:smtp.saving') : t('admin:smtp.save')}
              </Button>
              <Button type="button" variant="outline" disabled={busy !== null || (!data.configured && !data.env_relay)} onClick={test}>
                {busy === 'test' ? t('admin:smtp.testing') : t('admin:smtp.test')}
              </Button>
              {data.configured && (
                <Button
                  type="button"
                  variant="ghost"
                  className="text-destructive hover:text-destructive"
                  disabled={busy !== null}
                  onClick={() => setConfirmRemove(true)}
                >
                  {t('admin:smtp.remove')}
                </Button>
              )}
            </div>
          </form>
        )}

        {confirmRemove && (
          <Dialog open onOpenChange={(open) => !open && setConfirmRemove(false)}>
            <DialogContent className="sm:max-w-md">
              <DialogHeader>
                <DialogTitle>{t('admin:smtp.removeTitle')}</DialogTitle>
                <DialogDescription>{data?.env_relay ? t('admin:smtp.removeBodyPlatform') : t('admin:smtp.removeBodyNone')}</DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button variant="outline" onClick={() => setConfirmRemove(false)}>
                  {t('admin:form.cancel')}
                </Button>
                <Button variant="destructive" onClick={remove}>
                  {t('admin:smtp.removeConfirm')}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        )}
      </div>
    </SettingsRow>
  )
}
