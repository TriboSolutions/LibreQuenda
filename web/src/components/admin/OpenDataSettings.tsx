import { useState, type FormEvent } from 'react'
import { Trans, useTranslation } from 'react-i18next'
import { Check, Copy } from 'lucide-react'
import { SettingsRow } from '@/components/SettingsRow'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { useAsync } from '@/hooks/useAsync'
import { api, type CouncilOpenData } from '@/lib/api'

type Props = { slug: string; token: string }

const DIR3 = /^[A-Z][A-Z0-9]{8}$/

/**
 * The council's open data, in the council sheet: publish a DCAT-AP-ES catalog
 * of its facilities and their monthly use, under its DIR3 code, and the
 * address to register at datos.gob.es as a federation source.
 */
export function OpenDataSettings({ slug, token }: Props) {
  const { t } = useTranslation(['admin', 'common'])
  const openData = useAsync(() => api.council.openData(token, slug), `open-data:${slug}`)
  const [enabled, setEnabled] = useState(false)
  const [dir3, setDir3] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  // the form starts from each load of the saved settings
  const [loaded, setLoaded] = useState<CouncilOpenData | null>(null)
  if (openData.data && openData.data !== loaded) {
    setLoaded(openData.data)
    setEnabled(openData.data.enabled)
    setDir3(openData.data.dir3_code)
  }

  const base = `${window.location.origin}/api/tenants/${encodeURIComponent(slug)}/open-data`
  const catalogUrl = `${base}/catalog.rdf`
  const code = dir3.trim().toUpperCase()

  const save = async (event: FormEvent) => {
    event.preventDefault()
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await api.council.saveOpenData(token, slug, { enabled, dir3_code: code })
      openData.reload()
      setNotice(t(enabled ? 'admin:openData.savedOn' : 'admin:openData.savedOff'))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(catalogUrl)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      // no clipboard (http, permissions): the address stays selectable in the box
    }
  }

  const data = openData.data
  const published = data?.enabled ?? false
  const files = [
    { name: 'facilities', label: t('admin:openData.fileFacilities') },
    { name: 'monthly-use', label: t('admin:openData.fileUse') },
  ]

  return (
    <SettingsRow id="sheet-open-data" title={t('admin:openData.title')} description={t('admin:openData.help')}>
      <div className="grid content-start gap-4 rounded-lg border bg-card p-5">
        {data && (
          <div>
            <span
              className={`rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap ${published ? 'bg-pill-positive-bg text-pill-positive-ink' : 'bg-pill-neutral-bg text-pill-neutral-ink'}`}
            >
              {published ? t('admin:openData.statusOn') : t('admin:openData.statusOff')}
            </span>
          </div>
        )}
        <div aria-live="polite" className="grid gap-2 empty:hidden">
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
        </div>

        {openData.error ? (
          <div className="flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
            <p>{openData.error.message}</p>
            <Button variant="ghost" size="sm" onClick={openData.reload}>
              {t('common:retry')}
            </Button>
          </div>
        ) : !data ? (
          <Skeleton className="h-40 w-full" />
        ) : (
          <>
            <form onSubmit={(e) => void save(e)} className="grid gap-4">
              <div className="flex items-start justify-between gap-4">
                <div className="grid gap-1">
                  <Label htmlFor="open-data-enabled">{t('admin:openData.enabled')}</Label>
                  <p id="open-data-enabled-help" className="text-xs text-muted-foreground">
                    {t('admin:openData.enabledHelp')}
                  </p>
                </div>
                <Switch id="open-data-enabled" checked={enabled} aria-describedby="open-data-enabled-help" onCheckedChange={setEnabled} />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="open-data-dir3">{t('admin:openData.dir3')}</Label>
                <Input
                  id="open-data-dir3"
                  value={dir3}
                  onChange={(e) => setDir3(e.target.value)}
                  placeholder="L01150301"
                  maxLength={9}
                  required
                  autoComplete="off"
                  spellCheck={false}
                  aria-describedby="open-data-dir3-help"
                  className="w-40 font-mono uppercase"
                />
                <p id="open-data-dir3-help" className="text-muted-foreground">
                  <Trans
                    t={t}
                    i18nKey="admin:openData.dir3Help"
                    components={{
                      search: <a href="https://administracionelectronica.gob.es/ctt/dir3" target="_blank" rel="noreferrer" className="underline" />,
                    }}
                  />
                </p>
              </div>
              <div>
                <Button type="submit" disabled={busy || !DIR3.test(code)}>
                  {busy ? t('admin:openData.saving') : t('admin:openData.save')}
                </Button>
              </div>
            </form>

            {published && (
              <div className="grid gap-4 border-t pt-4">
                <div className="grid gap-1.5">
                  <Label htmlFor="open-data-url">{t('admin:openData.catalogUrl')}</Label>
                  <div className="flex gap-2">
                    <Input id="open-data-url" readOnly value={catalogUrl} onFocus={(e) => e.currentTarget.select()} className="font-mono text-xs" />
                    <Button type="button" variant="outline" onClick={() => void copy()}>
                      {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
                      {copied ? t('admin:openData.copied') : t('admin:openData.copy')}
                    </Button>
                  </div>
                  <span role="status" className="sr-only">
                    {copied ? t('admin:openData.copied') : ''}
                  </span>
                </div>
                <div className="grid gap-1.5">
                  <p className="text-sm font-medium">{t('admin:openData.stepsTitle')}</p>
                  <ol className="grid list-decimal gap-1 pl-5 text-muted-foreground">
                    <li>
                      <Trans
                        t={t}
                        i18nKey="admin:openData.step1"
                        values={{ dir3: data.dir3_code }}
                        components={{
                          portal: <a href="https://datos.gob.es/es/preguntas-frecuentes/como-federar-mi-catalogo-de-datos-abiertos-con-datosgobes" target="_blank" rel="noreferrer" className="underline" />,
                        }}
                      />
                    </li>
                    <li>{t('admin:openData.step2')}</li>
                    <li>{t('admin:openData.step3')}</li>
                  </ol>
                </div>
                <div className="grid gap-1.5">
                  <p className="text-sm font-medium">{t('admin:openData.files')}</p>
                  <ul className="grid gap-1">
                    {files.map((file) => (
                      <li key={file.name} className="flex flex-wrap items-baseline gap-x-3">
                        <span>{file.label}</span>
                        <a href={`${base}/${file.name}.csv`} className="text-sm underline">
                          CSV
                        </a>
                        <a href={`${base}/${file.name}.json`} target="_blank" rel="noreferrer" className="text-sm underline">
                          JSON
                        </a>
                      </li>
                    ))}
                  </ul>
                  <p className="text-xs text-muted-foreground">{t('admin:openData.privacy')}</p>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </SettingsRow>
  )
}
