import { CheckCircle2, LocateFixed, XCircle } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useParams, useSearchParams } from 'react-router'
import { SiteFooter } from '@/components/SiteFooter'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { usePageTitle } from '@/hooks/usePageTitle'
import { api, type EntranceResult } from '@/lib/api'
import { GRANTED, locate, type Position } from '@/lib/entrance'
import { formatTime } from '@/lib/time'

/**
 * /:slug/:facility/entrada: check in from the resident's own phone at the door.
 * The printed QR opens it with ?c=<token>, the NFC tag with ?n=<token>; otherwise the resident types the
 * facility's entry code. Either way the phone's location is checked.
 */
export function EntrancePage() {
  const { slug = '', facility: facilitySlug = '' } = useParams()
  const [params] = useSearchParams()
  const qrToken = params.get('c') ?? ''
  const nfcToken = params.get('n') ?? '' // the NFC tag on the door opens /entrada?n=<token>
  const { t } = useTranslation(['entrance', 'reader', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token
  const me = useMe()
  const { login } = useLogin()
  // GET /me (MeProvider) first creates the user's row
  const entrance = useAsync(token && me.me ? () => api.entrance.get(token, slug, facilitySlug) : null, `entrance:${slug}/${facilitySlug}:${me.me?.id}`)
  usePageTitle(t('entrance:title'), entrance.data?.facility_name)

  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<EntranceResult | null>(null)
  const [problem, setProblem] = useState<string | null>(null)
  const [position, setPosition] = useState<Position | null>(null)

  const data = entrance.data
  const byQr = qrToken !== '' && !!data?.printed_qr
  const byNfc = !byQr && nfcToken !== '' && !!data?.nfc
  const byCode = !byQr && !byNfc && !!data?.entry_code
  const timeZone = data?.timezone ?? 'Europe/Madrid'

  const send = async (book: boolean) => {
    if (!token) return
    setBusy(true)
    setProblem(null)
    try {
      // a fresh location for each try; "book" right after reuses the one just checked
      let where = book ? position : null
      if (!where) {
        try {
          where = await locate()
        } catch (e) {
          setResult(null)
          setProblem(t(`entrance:location.${e instanceof Error ? e.message : 'unavailable'}`))
          return
        }
        setPosition(where)
      }
      const proof = byQr ? { qr_token: qrToken } : byNfc ? { nfc_token: nfcToken } : { code }
      const answer = await api.entrance.checkIn(token, slug, facilitySlug, { ...proof, ...where, book })
      setResult(answer)
      if (GRANTED.includes(answer.result)) entrance.reload()
    } catch (e) {
      setResult(null)
      setProblem(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const submit = (event: FormEvent) => {
    event.preventDefault()
    void send(false)
  }

  let body
  if (auth.isLoading || (auth.isAuthenticated && !data && !entrance.error && !me.error)) {
    body = <Skeleton className="h-48 w-full" aria-busy="true" aria-label={t('entrance:loading')} />
  } else if (!auth.isAuthenticated) {
    body = (
      <div className="grid gap-4 rounded-lg border bg-card p-5">
        <p>{t('entrance:signIn')}</p>
        <div>
          <Button onClick={login}>{t('common:signIn')}</Button>
        </div>
      </div>
    )
  } else if (me.error || entrance.error || !data) {
    body = (
      <div className="flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
        <p>{entrance.error?.message ?? t('entrance:loadError')}</p>
        <Button variant="ghost" size="sm" onClick={() => (me.error ? me.reload() : entrance.reload())}>
          {t('common:retry')}
        </Button>
      </div>
    )
  } else if (!byQr && !byNfc && !byCode) {
    body = (
      <Alert>
        <AlertDescription>{qrToken || nfcToken ? t('entrance:qrOff') : data.printed_qr ? t('entrance:scanQr') : t('entrance:notHere')}</AlertDescription>
      </Alert>
    )
  } else {
    const granted = result && GRANTED.includes(result.result)
    const booking = result?.booking
    body = (
      <div className="grid gap-5">
        <div className="rounded-lg border bg-card p-5">
          <h2 className="font-semibold">{t('entrance:yourBooking')}</h2>
          <p className="mt-1 text-muted-foreground">
            {data.booking
              ? data.booking.checked_in_at
                ? t('entrance:bookingIn', {
                    start: formatTime(data.booking.starts_at, timeZone),
                    end: formatTime(data.booking.ends_at, timeZone),
                    time: formatTime(data.booking.checked_in_at, timeZone),
                  })
                : t('entrance:booking', { start: formatTime(data.booking.starts_at, timeZone), end: formatTime(data.booking.ends_at, timeZone) })
              : data.flash_booking
                ? t('entrance:noBookingFlash')
                : t('entrance:noBooking')}
          </p>
        </div>

        <form onSubmit={submit} className="grid gap-4 rounded-lg border bg-card p-5">
          {byCode && (
            <div className="grid gap-2 sm:max-w-xs">
              <Label htmlFor="entrance-code">{t('entrance:code')}</Label>
              <Input
                id="entrance-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                pattern="[0-9 \-]*"
                maxLength={8}
                required
                value={code}
                onChange={(e) => setCode(e.target.value)}
                aria-describedby="entrance-code-help"
                className="num font-mono text-lg tracking-widest"
              />
              <p id="entrance-code-help" className="text-sm text-muted-foreground">
                {t('entrance:codeHelp')}
              </p>
            </div>
          )}
          <p className="flex items-start gap-2 text-sm text-muted-foreground">
            <LocateFixed className="mt-0.5 size-4 shrink-0" aria-hidden />
            {t('entrance:locationNote', { metres: data.radius_m })}
          </p>
          <div>
            <Button type="submit" disabled={busy}>
              {busy ? t('entrance:checking') : t('entrance:checkIn')}
            </Button>
          </div>
        </form>

        <div role="status" aria-live="polite" className="empty:hidden">
          {result && (
            <div
              className={`grid gap-3 rounded-lg p-5 ${granted ? 'bg-pill-positive-bg text-pill-positive-ink' : 'bg-pill-negative-bg text-pill-negative-ink'}`}
            >
              <p className="flex items-center gap-2 text-lg font-semibold">
                {granted ? <CheckCircle2 className="size-5" aria-hidden /> : <XCircle className="size-5" aria-hidden />}
                {t(`entrance:result.${result.result}`, { defaultValue: t('entrance:result.refused') })}
              </p>
              {booking && (
                <p>
                  {t('entrance:slot', {
                    facility: booking.facility_name,
                    start: formatTime(booking.starts_at, timeZone),
                    end: formatTime(booking.ends_at, timeZone),
                  })}
                </p>
              )}
              {result.result === 'not_booked' && result.reason && <p>{t(`reader:reason.${result.reason}`, { defaultValue: t('reader:reason.other') })}</p>}
              {result.result === 'location_inaccurate' && <p>{t('entrance:inaccurateHelp')}</p>}
              {result.result === 'no_booking' && result.can_book && (
                <div>
                  <Button type="button" disabled={busy} onClick={() => void send(true)}>
                    {t('entrance:bookAndCheckIn')}
                  </Button>
                </div>
              )}
            </div>
          )}
          {problem && <p className="rounded-lg bg-pill-negative-bg p-5 text-pill-negative-ink">{problem}</p>}
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[760px] px-4 pt-8 pb-16 sm:px-6">
        <p className="text-muted-foreground">
          <Link to={`/${slug}/${facilitySlug}`} className="underline underline-offset-4">
            {data?.facility_name ?? t('entrance:facility')}
          </Link>
        </p>
        <h1 className="mt-1 mb-6 text-3xl font-semibold">{t('entrance:title')}</h1>
        {body}
      </main>
      <SiteFooter />
    </div>
  )
}
