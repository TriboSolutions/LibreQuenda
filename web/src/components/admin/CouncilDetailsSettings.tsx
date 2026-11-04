import { ExternalLink } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { CouncilFields } from '@/components/admin/CouncilFields'
import { SettingsRow } from '@/components/SettingsRow'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { councilInput } from '@/lib/councilInput'
import { api, type CouncilDetail, type CouncilInput, type StoredImage } from '@/lib/api'

type Props = {
  council: CouncilDetail
  token: string
  canDeactivate: boolean // superadministrators only
  onSaved: (council: CouncilDetail) => void
}

/** The council's own details, edited in place like the profile's. */
export function CouncilDetailsSettings({ council, token, canDeactivate, onSaved }: Props) {
  const { t } = useTranslation('admin')
  const [form, setForm] = useState<CouncilInput>(() => councilInput(council))
  const [logo, setLogo] = useState<StoredImage | null>(council.logo_ref ? { ref: council.logo_ref, url: council.logo_url } : null)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const set = (field: keyof CouncilInput, value: string | boolean) => setForm((f) => ({ ...f, [field]: value }))

  const save = async (event: React.FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    setNotice(null)
    try {
      const { admin_email: _, ...fields } = form
      const saved = await api.admin.updateCouncil(token, council.slug, { ...fields, logo_ref: logo?.ref ?? '' }) // the reference, never the URL
      setNotice(t('notice.saved'))
      onSaved(saved)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <SettingsRow id="sheet-details" title={t('manage.details')} description={t('manage.detailsHelp')}>
      <form onSubmit={(e) => void save(e)} className="grid gap-5 rounded-lg border bg-card p-5">
        {council.is_active ? (
          <p>
            <Link target="_blank" rel="noopener noreferrer" to={`/${council.slug}`} className="inline-flex items-center gap-1.5 text-primary hover:underline">
              {t('manage.viewPage')}
              <ExternalLink className="size-3.5" aria-hidden="true" />
            </Link>
          </p>
        ) : (
          !canDeactivate && <p className="text-muted-foreground">{t('councils.inactive')}</p>
        )}
        <CouncilFields form={form} set={set} logo={logo} setLogo={setLogo} creating={false} councilSlug={council.slug} canDeactivate={canDeactivate} />
        <div aria-live="polite" className="empty:hidden">
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
        <div className="flex justify-end border-t pt-4">
          <Button type="submit" disabled={saving}>
            {saving ? t('form.saving') : t('manage.save')}
          </Button>
        </div>
      </form>
    </SettingsRow>
  )
}
