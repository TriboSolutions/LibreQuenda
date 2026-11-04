import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { CouncilFields } from '@/components/admin/CouncilFields'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { councilInput } from '@/lib/councilInput'
import { api, type CouncilDetail, type CouncilInput, type StoredImage } from '@/lib/api'

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
  const [form, setForm] = useState<CouncilInput>(() => councilInput(council))
  const [logo, setLogo] = useState<StoredImage | null>(council?.logo_ref ? { ref: council.logo_ref, url: council.logo_url } : null)
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

          <CouncilFields form={form} set={set} logo={logo} setLogo={setLogo} creating={creating} councilSlug={council?.slug ?? null} canDeactivate={canDeactivate} />

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
