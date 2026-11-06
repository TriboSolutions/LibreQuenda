import { useTranslation } from 'react-i18next'
import { ImagePicker } from '@/components/ImagePicker'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { CouncilInput, StoredImage } from '@/lib/api'
import { slugify } from '@/lib/slug'

// Spain's two civil time zones
const TIME_ZONES = [
  { value: 'Europe/Madrid', key: 'form.tzMadrid' },
  { value: 'Atlantic/Canary', key: 'form.tzCanary' },
] as const

type Props = {
  form: CouncilInput
  set: (field: keyof CouncilInput, value: string | boolean) => void
  logo: StoredImage | null
  setLogo: (logo: StoredImage | null) => void
  creating: boolean
  councilSlug: string | null // its saved address, whose storage the logo goes to; null while creating it
  canDeactivate: boolean // superadministrators only: a council's own administrators can't switch it off
}

/** A council's own fields, shared by the "new council" dialog and the council settings. */
export function CouncilFields({ form, set, logo, setLogo, creating, councilSlug, canDeactivate }: Props) {
  const { t } = useTranslation('admin')
  return (
    <>
      <div className="grid gap-2">
        <Label htmlFor="council-name">{t('form.name')}</Label>
        <Input
          id="council-name"
          required
          maxLength={255}
          placeholder={t('form.namePlaceholder')}
          value={form.name}
          aria-describedby="council-url"
          onChange={(e) => {
            set('name', e.target.value)
            // the address comes from the name when the council is created, and then stays: links to it are out there
            if (creating) set('slug', slugify(e.target.value))
          }}
        />
        <p id="council-url" className="text-xs text-muted-foreground">
          {t('form.pageUrl', { url: `${window.location.host}/${form.slug || '…'}` })}
        </p>
      </div>

      {/* each council's files live in its own storage: the logo comes once it exists, from its settings */}
      {councilSlug && <ImagePicker label={t('form.logo')} folder="councils" owner={{ council: councilSlug }} value={logo} onChange={setLogo} crop />}

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

      <div className="grid gap-2 sm:max-w-xs">
        <Label htmlFor="council-municipality">
          {t('form.municipalityCode')} <span className="font-normal text-muted-foreground">({t('form.optional')})</span>
        </Label>
        <Input
          id="council-municipality"
          inputMode="numeric"
          pattern="[0-9]{5}"
          maxLength={5}
          value={form.municipality_code}
          aria-describedby="council-municipality-help"
          onChange={(e) => set('municipality_code', e.target.value.trim())}
          className="num"
        />
        <p id="council-municipality-help" className="text-xs text-muted-foreground">
          {t('form.municipalityCodeHelp')}
        </p>
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
            <Switch id="council-active" checked={form.is_active} aria-describedby="council-active-help" onCheckedChange={(checked) => set('is_active', checked)} />
          </div>
        )
      )}
    </>
  )
}
