import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { ImagePicker } from '@/components/ImagePicker'
import { Mark, Wordmark } from '@/components/Mark'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { DEFAULT_APP_NAME, useBranding } from '@/hooks/useBranding'
import { api, type Branding, type StoredImage } from '@/lib/api'

const MAX_NAME = 60

/**
 * The installation's own name and logo, for a council or a group of councils that
 * hosts LibreQuenda under its own brand. Empty fields go back to LibreQuenda's.
 */
export function BrandingPanel() {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const { branding, reload } = useBranding()
  const [name, setName] = useState('')
  const [logo, setLogo] = useState<StoredImage | null>(null)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  // the form starts from each load of the saved branding
  const [loaded, setLoaded] = useState<Branding | null>(null)
  if (branding !== loaded) {
    setLoaded(branding)
    setName(branding.custom_name)
    setLogo(branding.logo_ref ? { ref: branding.logo_ref, url: branding.logo_url } : null)
  }

  const save = async (event: FormEvent) => {
    event.preventDefault()
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await api.admin.updateBranding(token, { name: name.trim(), logo_ref: logo?.ref ?? '' })
      setNotice(t('admin:notice.saved'))
      reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const shownName = name.trim()
  const isDefault = !shownName && !logo

  return (
    <section aria-labelledby="branding-title" className="max-w-[760px] rounded-lg border bg-card p-5">
      <h2 id="branding-title" className="text-lg font-semibold">
        {t('admin:branding.title')}
      </h2>
      <p className="mt-1 text-muted-foreground">{t('admin:branding.help')}</p>

      <form onSubmit={(e) => void save(e)} className="mt-4 grid gap-5">
        <div className="grid gap-2">
          <Label htmlFor="branding-name">{t('admin:branding.name')}</Label>
          <Input
            id="branding-name"
            value={name}
            maxLength={MAX_NAME}
            placeholder={DEFAULT_APP_NAME}
            aria-describedby="branding-name-help"
            onChange={(e) => setName(e.target.value)}
          />
          <p id="branding-name-help" className="text-xs text-muted-foreground">
            {t('admin:branding.nameHelp', { app: DEFAULT_APP_NAME })}
          </p>
        </div>

        <div className="grid gap-2">
          <ImagePicker label={t('admin:branding.logo')} folder="branding" value={logo} onChange={setLogo} />
          <p className="text-xs text-muted-foreground">{t('admin:branding.logoHelp', { app: DEFAULT_APP_NAME })}</p>
        </div>

        <div className="grid gap-2">
          <h3 id="branding-preview" className="text-sm leading-none font-medium">
            {t('admin:branding.preview')}
          </h3>
          {/* the top bar as it will look, with what is in the form */}
          <div aria-labelledby="branding-preview" role="group" className="flex h-14 items-center gap-2 rounded-md border bg-background px-4">
            {logo ? <img src={logo.url} alt="" className="size-7 rounded-md object-contain" /> : <Mark className="size-7" />}
            {shownName ? (
              <span className="truncate text-[17px] font-semibold tracking-tight text-brand-ink">{shownName}</span>
            ) : (
              <Wordmark className="text-[17px] tracking-tight" />
            )}
          </div>
        </div>

        <div aria-live="polite">
          {notice && (
            <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
              <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
            </Alert>
          )}
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>

        <div className="flex flex-wrap justify-end gap-2 border-t pt-4">
          <Button
            type="button"
            variant="ghost"
            disabled={busy || isDefault}
            onClick={() => {
              setName('')
              setLogo(null)
            }}
          >
            {t('admin:branding.useDefault', { app: DEFAULT_APP_NAME })}
          </Button>
          <Button type="submit" disabled={busy}>
            {busy ? t('admin:branding.saving') : t('admin:branding.save')}
          </Button>
        </div>
      </form>
    </section>
  )
}
