import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Check, Copy, ExternalLink, ScanLine } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type CheckinDevice, type CreatedCheckinDevice } from '@/lib/api'
import { formatDateTime } from '@/lib/time'

type Props = { slug: string; facilityId: string; token: string }

// last-seen times in the administrator's own zone
const localZone = Intl.DateTimeFormat().resolvedOptions().timeZone

/** The reader page signs in with the key in the fragment, which never reaches a server log. */
export function readerUrl(key: string) {
  return `${window.location.origin}/lector#key=${key}`
}

function CopyButton({ value, label }: { value: string; label: string }) {
  const { t } = useTranslation('admin')
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    await navigator.clipboard.writeText(value)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  return (
    <Button type="button" variant="outline" size="sm" onClick={() => void copy()} aria-label={label}>
      {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
      {copied ? t('readers.copied') : t('readers.copy')}
    </Button>
  )
}

/** QR readers at the facility entrance: each one signs in with its own key. */
export function CheckinDevices({ slug, facilityId, token }: Props) {
  const { t } = useTranslation(['admin', 'common'])
  const devices = useAsync(() => api.council.devices(token, slug, facilityId), `${slug}/${facilityId}`)
  const [name, setName] = useState('')
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [created, setCreated] = useState<CreatedCheckinDevice | null>(null)
  const [toRemove, setToRemove] = useState<CheckinDevice | null>(null)
  const [removing, setRemoving] = useState(false)

  const add = async (event: FormEvent) => {
    event.preventDefault()
    setAdding(true)
    setError(null)
    try {
      setCreated(await api.council.createDevice(token, slug, facilityId, name.trim()))
      setName('')
      devices.reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setAdding(false)
    }
  }

  const remove = async () => {
    if (!toRemove) return
    setRemoving(true)
    setError(null)
    try {
      await api.council.deleteDevice(token, slug, facilityId, toRemove.id)
      devices.reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setRemoving(false)
      setToRemove(null)
    }
  }

  const list = devices.data?.devices ?? []

  return (
    <section aria-labelledby="sec-readers" className="grid gap-4 rounded-lg border bg-card p-5">
      <div>
        <h2 id="sec-readers" className="text-lg font-semibold">
          {t('admin:readers.title')}
        </h2>
        <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('admin:readers.help')}</p>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {devices.error ? (
        <div className="flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
          <p>{t('admin:readers.loadError')}</p>
          <Button variant="ghost" size="sm" onClick={devices.reload}>
            {t('common:retry')}
          </Button>
        </div>
      ) : devices.loading && !devices.data ? (
        <Skeleton className="h-12 w-full" />
      ) : list.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('admin:readers.empty')}</p>
      ) : (
        <ul className="divide-y rounded-md border">
          {list.map((device) => (
            <li key={device.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
              <div className="flex min-w-0 items-center gap-3">
                <ScanLine className="size-5 shrink-0 text-muted-foreground" aria-hidden />
                <div className="min-w-0">
                  <p className="truncate font-medium">{device.name}</p>
                  <p className="num text-muted-foreground">
                    {t('admin:readers.keyHint', { hint: device.key_hint })}
                    {' · '}
                    {device.last_seen_at
                      ? t('admin:readers.lastSeen', { date: formatDateTime(device.last_seen_at, localZone) })
                      : t('admin:readers.neverSeen')}
                  </p>
                </div>
              </div>
              <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => setToRemove(device)}>
                {t('admin:readers.remove')}
              </Button>
            </li>
          ))}
        </ul>
      )}

      <form onSubmit={(e) => void add(e)} className="flex flex-wrap items-end gap-3">
        <div className="grid min-w-56 flex-1 gap-1.5 sm:max-w-sm">
          <Label htmlFor="reader-name">{t('admin:readers.name')}</Label>
          <Input
            id="reader-name"
            value={name}
            maxLength={80}
            placeholder={t('admin:readers.namePlaceholder')}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <Button type="submit" disabled={adding || name.trim() === ''}>
          {adding ? t('admin:readers.adding') : t('admin:readers.add')}
        </Button>
      </form>

      {created && (
        <Dialog open onOpenChange={(open) => !open && setCreated(null)}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>{t('admin:readers.createdTitle', { name: created.name })}</DialogTitle>
              <DialogDescription>{t('admin:readers.createdBody')}</DialogDescription>
            </DialogHeader>
            <div className="grid gap-4">
              <div className="grid gap-1.5">
                <Label>{t('admin:readers.link')}</Label>
                <code className="block rounded-md bg-muted px-3 py-2 text-xs break-all">{readerUrl(created.key)}</code>
                <div className="flex flex-wrap gap-2">
                  <CopyButton value={readerUrl(created.key)} label={t('admin:readers.copyLink')} />
                  <Button variant="outline" size="sm" asChild>
                    <a href={readerUrl(created.key)} target="_blank" rel="noreferrer">
                      <ExternalLink aria-hidden />
                      {t('admin:readers.open')}
                    </a>
                  </Button>
                </div>
              </div>
              <div className="grid gap-1.5">
                <Label>{t('admin:readers.key')}</Label>
                <code className="block rounded-md bg-muted px-3 py-2 text-xs break-all">{created.key}</code>
                <div>
                  <CopyButton value={created.key} label={t('admin:readers.copyKey')} />
                </div>
                <p className="text-muted-foreground">{t('admin:readers.keyHelp')}</p>
              </div>
              <Alert className="border-0 bg-pill-accent-bg text-pill-accent-ink">
                <AlertDescription className="text-pill-accent-ink">{t('admin:readers.onlyOnce')}</AlertDescription>
              </Alert>
            </div>
            <DialogFooter>
              <Button onClick={() => setCreated(null)}>{t('admin:readers.done')}</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      {toRemove && (
        <Dialog open onOpenChange={(open) => !open && setToRemove(null)}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>{t('admin:readers.removeTitle', { name: toRemove.name })}</DialogTitle>
              <DialogDescription>{t('admin:readers.removeBody')}</DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setToRemove(null)}>
                {t('admin:form.cancel')}
              </Button>
              <Button variant="destructive" disabled={removing} onClick={() => void remove()}>
                {t('admin:readers.removeConfirm')}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </section>
  )
}
