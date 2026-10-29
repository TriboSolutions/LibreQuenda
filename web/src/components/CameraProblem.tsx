import { useTranslation } from 'react-i18next'
import { CameraOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import type { CameraState } from '@/hooks/useQrCamera'

/** Over the camera's frame while it is not reading: starting, or why it cannot. */
export function CameraProblem({ state, onRetry }: { state: CameraState; onRetry: () => void }) {
  const { t } = useTranslation('reader')
  if (state === 'starting') {
    return <p className="absolute inset-0 grid place-items-center p-6 text-center text-white/80">{t('camera.starting')}</p>
  }
  return (
    <div className="absolute inset-0 grid place-content-center justify-items-center gap-3 p-6 text-center text-white">
      <CameraOff className="size-10" aria-hidden />
      <p className="max-w-xs text-balance">{t(`camera.${state}`)}</p>
      {state !== 'insecure' && (
        <Button variant="secondary" onClick={onRetry}>
          {t('retry')}
        </Button>
      )}
    </div>
  )
}
