import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { QrImage } from '@/components/QrImage'
import { SettingsRow } from '@/components/SettingsRow'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { api, type Me, type PersonalQr as Qr } from '@/lib/api'

const RETRY_MS = 10_000

/*
  The personal QR on /perfil. At a facility's reader it checks the person in to
  their booking there, or books the turn that starts within 15 minutes. The code
  changes every minute and works once, so a screenshot soon stops working.
*/
export function PersonalQr({ me }: { me: Me }) {
  const { t } = useTranslation('profile')
  const auth = useAuth()
  const token = auth.user?.access_token
  const [qr, setQr] = useState<{ value: Qr; loadedAt: number } | null>(null)
  const [failed, setFailed] = useState(false)
  const [attempt, setAttempt] = useState(0)
  const [now, setNow] = useState(() => Date.now())

  // a new code every refresh_seconds while the page is in view, and right away when it comes back
  useEffect(() => {
    if (!token) return
    let active = true
    let timer: ReturnType<typeof setTimeout> | undefined
    const load = () => {
      clearTimeout(timer)
      if (document.visibilityState === 'hidden') return
      api
        .personalQr(token)
        .then((value) => {
          if (!active) return
          setQr({ value, loadedAt: Date.now() })
          setFailed(false)
          timer = setTimeout(load, value.refresh_seconds * 1000)
        })
        .catch(() => {
          if (!active) return
          setFailed(true)
          timer = setTimeout(load, RETRY_MS)
        })
    }
    const onVisible = () => document.visibilityState === 'visible' && load()
    load()
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      active = false
      clearTimeout(timer)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [token, attempt])

  useEffect(() => {
    const tick = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(tick)
  }, [])

  const expired = qr !== null && Date.parse(qr.value.expires_at) <= now
  const secondsLeft = qr ? Math.max(0, Math.ceil((qr.loadedAt + qr.value.refresh_seconds * 1000 - now) / 1000)) : 0

  let code
  if (qr && !expired) {
    code = (
      <div className="grid w-full max-w-56 gap-2">
        <QrImage code={qr.value.code} alt={t('personalQr.alt')} className="max-w-56" />
        {/* visual only: announcing every second would drown out a screen reader */}
        <div aria-hidden className="grid gap-1">
          <div className="h-1 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full bg-primary transition-[width] duration-1000 ease-linear"
              style={{ width: `${(secondsLeft / qr.value.refresh_seconds) * 100}%` }}
            />
          </div>
          <p className="num text-center text-xs text-muted-foreground">{t('personalQr.refreshIn', { seconds: secondsLeft })}</p>
        </div>
      </div>
    )
  } else if (failed || expired) {
    code = (
      <div className="grid w-full max-w-56 content-start gap-3">
        <Alert variant="destructive">
          <AlertDescription>{t('personalQr.failed')}</AlertDescription>
        </Alert>
        <Button variant="outline" onClick={() => setAttempt((n) => n + 1)}>
          {t('personalQr.retry')}
        </Button>
      </div>
    )
  } else {
    code = <Skeleton className="aspect-square w-full max-w-56" />
  }

  return (
    <SettingsRow id="sec-qr" title={t('personalQr.title')} description={t('personalQr.help')}>
      <div className="grid gap-5 rounded-lg border bg-card p-5 sm:grid-cols-[14rem_minmax(0,1fr)]">
        <div aria-live="polite" className="flex justify-center">
          {code}
        </div>
        <div className="grid content-start gap-3">
          {!me.profile_complete && (
            <Alert>
              <AlertDescription>{t('personalQr.incomplete')}</AlertDescription>
            </Alert>
          )}
          <ul className="grid list-disc gap-2 pl-5">
            <li>{t('personalQr.checkIn')}</li>
            <li>{t('personalQr.book')}</li>
          </ul>
          <p className="text-muted-foreground">{t('personalQr.private')}</p>
          <p className="text-muted-foreground">{t('personalQr.brightness')}</p>
        </div>
      </div>
    </SettingsRow>
  )
}
