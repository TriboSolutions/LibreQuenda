import { Download } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'

type Props = { slug: string; token: string }

/** Download everything the council keeps here, in open formats, to take it elsewhere. */
export function CouncilExport({ slug, token }: Props) {
  const { t } = useTranslation(['admin'])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const download = async () => {
    setBusy(true)
    setError(null)
    try {
      await api.council.exportData(token, slug)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section aria-labelledby="sheet-export" className="grid gap-3 border-t pt-5">
      <h3 id="sheet-export" className="font-semibold">
        {t('admin:councilExport.title')}
      </h3>
      <p className="max-w-[70ch] text-muted-foreground">{t('admin:councilExport.help')}</p>
      <p className="max-w-[70ch] text-xs text-muted-foreground">{t('admin:councilExport.privacy')}</p>
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
    </section>
  )
}
