import { useTranslation } from 'react-i18next'
import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useNewVersion } from '@/hooks/useNewVersion'

/**
 * Under the top bar once the API runs a newer version than this page: until it
 * reloads, the API refuses its calls, so whatever is being done would fail.
 * Not reloaded by itself, so nothing typed is lost.
 */
export function NewVersionNotice() {
  const { t } = useTranslation('common')
  if (!useNewVersion()) return null

  return (
    <div role="alert" className="border-b bg-pill-accent-bg text-pill-accent-ink">
      <div className="mx-auto flex max-w-[1152px] flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-2.5 sm:px-6">
        <p className="flex items-start gap-2">
          <RefreshCw className="mt-0.5 size-4 shrink-0" aria-hidden />
          <span>
            <span className="font-medium">{t('newVersion.title')}</span> {t('newVersion.body')}
          </span>
        </p>
        <Button size="sm" onClick={() => window.location.reload()}>
          {t('newVersion.reload')}
        </Button>
      </div>
    </div>
  )
}
