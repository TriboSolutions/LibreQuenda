import { Download } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { useNavigate } from 'react-router'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { SettingsRow } from '@/components/SettingsRow'
import { useBranding } from '@/hooks/useBranding'
import { api, type Me } from '@/lib/api'

/** RXPD / GDPR: download what LibreQuenda keeps about the user, or erase the account. */
export function PrivacySettings({ me }: { me: Me }) {
  const { t } = useTranslation('profile')
  const { branding } = useBranding()
  const [deleting, setDeleting] = useState(false)

  return (
    <SettingsRow id="sec-privacy" title={t('privacy.title')} description={t('privacy.help', { app: branding.name })}>
      <div className="rounded-lg border bg-card">
        <ExportCard />
        <section aria-labelledby="privacy-delete" className="grid content-start gap-4 border-t p-5">
          <h3 id="privacy-delete" className="font-semibold">
            {t('privacy.deleteTitle')}
          </h3>
          <p className="text-muted-foreground">{t('privacy.deleteHelp')}</p>
          <div>
            <Button variant="destructive" onClick={() => setDeleting(true)}>
              {t('privacy.delete')}
            </Button>
          </div>
        </section>
      </div>
      <DeleteAccountDialog open={deleting} email={me.email} onClose={() => setDeleting(false)} />
    </SettingsRow>
  )
}

function ExportCard() {
  const { t } = useTranslation('profile')
  const auth = useAuth()
  const [busy, setBusy] = useState<'json' | 'csv' | null>(null)
  const [error, setError] = useState<string | null>(null)

  const download = async (format: 'json' | 'csv') => {
    const token = auth.user?.access_token
    if (!token) return
    setBusy(format)
    setError(null)
    try {
      await api.exportMyData(token, format)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(null)
    }
  }

  return (
    <section aria-labelledby="privacy-export" className="grid content-start gap-4 p-5">
      <h3 id="privacy-export" className="font-semibold">
        {t('privacy.exportTitle')}
      </h3>
      <p className="text-muted-foreground">{t('privacy.exportHelp')}</p>
      <div aria-live="polite" className="empty:hidden">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" disabled={busy !== null} onClick={() => void download('json')}>
          <Download />
          {busy === 'json' ? t('privacy.preparing') : t('privacy.exportJson')}
        </Button>
        <Button variant="outline" disabled={busy !== null} onClick={() => void download('csv')}>
          <Download />
          {busy === 'csv' ? t('privacy.preparing') : t('privacy.exportCsv')}
        </Button>
      </div>
    </section>
  )
}

/** Asks for the email to be typed in before erasing: there is no undo. */
function DeleteAccountDialog({ open, email, onClose }: { open: boolean; email: string; onClose: () => void }) {
  const { t } = useTranslation('profile')
  const auth = useAuth()
  const navigate = useNavigate()
  const [typed, setTyped] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const confirmed = typed.trim().toLowerCase() === email.toLowerCase()

  const close = () => {
    setTyped('')
    setError(null)
    onClose()
  }
  const erase = async (event: React.FormEvent) => {
    event.preventDefault()
    const token = auth.user?.access_token
    if (!token || !confirmed) return
    setSubmitting(true)
    setError(null)
    try {
      await api.deleteAccount(token)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setSubmitting(false)
      return
    }
    // the identity no longer exists, so there is no session to end at the sign-in service
    await auth.removeUser()
    navigate('/perfil', { replace: true, state: { accountDeleted: true } })
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && !submitting && close()}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={(e) => void erase(e)} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>{t('privacy.deleteDialogTitle')}</DialogTitle>
            <DialogDescription>{t('privacy.deleteDialogBody')}</DialogDescription>
          </DialogHeader>
          <ul className="grid list-disc gap-1 pl-5 text-sm">
            <li>{t('privacy.deleteUpcoming')}</li>
            <li>{t('privacy.deletePast')}</li>
            <li>{t('privacy.deleteSignIn')}</li>
          </ul>
          <div className="grid gap-2">
            <Label htmlFor="privacy-confirm">{t('privacy.deleteConfirmLabel', { email })}</Label>
            <Input id="privacy-confirm" type="email" autoComplete="off" value={typed} onChange={(e) => setTyped(e.target.value)} />
          </div>
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={close} disabled={submitting}>
              {t('accountSettings.cancel')}
            </Button>
            <Button type="submit" variant="destructive" disabled={!confirmed || submitting}>
              {submitting ? t('privacy.deleting') : t('privacy.deleteConfirm')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
