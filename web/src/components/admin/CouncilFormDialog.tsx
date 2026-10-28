import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { ImagePicker } from '@/components/ImagePicker'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { api, type CouncilDetail, type CouncilInput, type StoredImage } from '@/lib/api'
import { slugify } from '@/lib/slug'

// Spain's two civil time zones
const TIME_ZONES = [
  { value: 'Europe/Madrid', key: 'form.tzMadrid' },
  { value: 'Atlantic/Canary', key: 'form.tzCanary' },
] as const

type Props = {
  council: CouncilDetail | null // null: create
  canDeactivate?: boolean // superadministrators only: a council's own administrators can't switch it off
  onClose: () => void
  onSaved: (council: CouncilDetail, adminEmail: string | null) => void
}

export function CouncilFormDialog({ council, canDeactivate = true, onClose, onSaved }: Props) {
  const { t } = useTranslation('admin')
  const auth = useAuth()
  const creating = council === null
  const [form, setForm] = useState<CouncilInput>(() => ({
    name: council?.name ?? '',
    slug: council?.slug ?? '',
    timezone: council?.timezone ?? 'Europe/Madrid',
    contact_email: council?.contact_email ?? '',
    contact_phone: council?.contact_phone ?? '',
    logo_ref: council?.logo_ref ?? '',
    is_active: council?.is_active ?? true,
    admin_email: '',
  }))
  const [logo, setLogo] = useState<StoredImage | null>(council?.logo_ref ? { ref: council.logo_ref, url: council.logo_url } : null)
  const [slugTouched, setSlugTouched] = useState(!creating)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const set = (field: keyof CouncilInput, value: string | boolean) => setForm((f) => ({ ...f, [field]: value }))

  const submit = async (event: React.FormEvent) => {
    event.preventDefault()
    const token = auth.user?.access_token
    if (!token) return
    setSaving(true)
    setError(null)
    try {
      const { admin_email, is_active, ...rest } = form
      const fields = { ...rest, logo_ref: logo?.ref ?? '' } // save the reference, never the URL
      const saved = creating
        ? await api.admin.createCouncil(token, { ...fields, admin_email })
        : await api.admin.updateCouncil(token, council.slug, { ...fields, is_active })
      onSaved(saved, creating ? (admin_email ?? null) : null)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg">
        <form onSubmit={(e) => void submit(e)} className="grid gap-5">
          <DialogHeader>
            <DialogTitle>{creating ? t('form.createTitle') : council.name}</DialogTitle>
            {creating && <DialogDescription>{t('form.createDescription')}</DialogDescription>}
          </DialogHeader>

          <div className="grid gap-2">
            <Label htmlFor="council-name">{t('form.name')}</Label>
            <Input
              id="council-name"
              required
              maxLength={255}
              placeholder={t('form.namePlaceholder')}
              value={form.name}
              onChange={(e) => {
                set('name', e.target.value)
                if (!slugTouched) set('slug', slugify(e.target.value))
              }}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="council-slug">{t('form.slug')}</Label>
            <Input
              id="council-slug"
              required
              maxLength={64}
              pattern="[a-z0-9]+(-[a-z0-9]+)*"
              value={form.slug}
              aria-describedby="council-slug-help"
              onChange={(e) => {
                setSlugTouched(true)
                set('slug', e.target.value.toLowerCase())
              }}
            />
            <p id="council-slug-help" className="text-xs text-muted-foreground">
              {t('form.slugHelp', { url: `${window.location.host}/${form.slug || '…'}` })}
            </p>
          </div>

          <ImagePicker label={t('form.logo')} folder="councils" value={logo} onChange={setLogo} />

          <div className="grid gap-2">
            <Label htmlFor="council-timezone">{t('form.timezone')}</Label>
            <Select value={form.timezone} onValueChange={(value) => set('timezone', value)}>
              <SelectTrigger id="council-timezone" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {TIME_ZONES.map((tz) => (
                  <SelectItem key={tz.value} value={tz.value}>
                    {t(tz.key)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="council-email">
                {t('form.contactEmail')} <span className="font-normal text-muted-foreground">({t('form.optional')})</span>
              </Label>
              <Input id="council-email" type="email" value={form.contact_email} onChange={(e) => set('contact_email', e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="council-phone">
                {t('form.contactPhone')} <span className="font-normal text-muted-foreground">({t('form.optional')})</span>
              </Label>
              <Input id="council-phone" type="tel" maxLength={50} value={form.contact_phone} onChange={(e) => set('contact_phone', e.target.value)} />
            </div>
          </div>

          {creating ? (
            <div className="grid gap-2">
              <Label htmlFor="council-admin">{t('form.adminEmail')}</Label>
              <Input
                id="council-admin"
                type="email"
                required
                value={form.admin_email}
                aria-describedby="council-admin-help"
                onChange={(e) => set('admin_email', e.target.value)}
              />
              <p id="council-admin-help" className="text-xs text-muted-foreground">
                {t('form.adminEmailHelp')}
              </p>
            </div>
          ) : (
            canDeactivate && (
              <div className="flex items-start justify-between gap-4">
                <div className="grid gap-1">
                  <Label htmlFor="council-active">{t('form.active')}</Label>
                  <p id="council-active-help" className="text-xs text-muted-foreground">
                    {t('form.activeHelp')}
                  </p>
                </div>
                <Switch
                  id="council-active"
                  checked={form.is_active}
                  aria-describedby="council-active-help"
                  onCheckedChange={(checked) => set('is_active', checked)}
                />
              </div>
            )
          )}

          {error && (
            <Alert variant="destructive" aria-live="assertive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t('form.cancel')}
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? t('form.saving') : creating ? t('form.create') : t('form.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
