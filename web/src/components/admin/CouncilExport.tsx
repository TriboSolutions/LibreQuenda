import { Download } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { SettingsRow } from '@/components/SettingsRow'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { useBranding } from '@/hooks/useBranding'
import { api } from '@/lib/api'

type Props = { slug: string; token: string }

/** Download everything the council keeps here, in open formats, to take it elsewhere. */
export function CouncilExport({ slug, token }: Props) {
  const { t } = useTranslation(['admin'])
  const { branding } = useBranding()
  const [busy, setBusy] = useState(false)
  const [personal, setPersonal] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const download = async () => {
    setBusy(true)
    setError(null)
    try {
      await api.council.exportData(token, slug, personal)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <SettingsRow id="sheet-export" title={t('admin:councilExport.title')} description={t('admin:councilExport.help', { app: branding.name })}>
      <div className="grid content-start gap-4 rounded-lg border bg-card p-5">
        <p className="text-muted-foreground">{t('admin:councilExport.privacy')}</p>
        <div className="flex items-start gap-2">
          <Checkbox id="export-personal" checked={personal} onCheckedChange={(checked) => setPersonal(checked === true)} aria-describedby="export-personal-help" className="mt-0.5" />
          <div className="grid gap-1">
            <Label htmlFor="export-personal" className="leading-5 font-normal">{t('admin:councilExport.personal')}</Label>
            <p id="export-personal-help" className="max-w-[70ch] text-xs text-muted-foreground">
              {t('admin:councilExport.personalHelp')}
            </p>
          </div>
        </div>
        {error && (
          <Alert variant="destructive" aria-live="assertive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        <div>
          <Button variant="outline" disabled={busy} aria-busy={busy} onClick={() => void download()}>
            <Download aria-hidden="true" />
            {busy ? t('admin:councilExport.downloading') : t('admin:councilExport.download')}
          </Button>
        </div>
      </div>
    </SettingsRow>
  )
}
