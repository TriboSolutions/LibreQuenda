import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { CircleCheck, CircleX, Clock, WifiOff } from 'lucide-react'
import { BrandLogo } from '@/components/Brand'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAsync } from '@/hooks/useAsync'
import { api, ApiError, type CheckinResult, type Reader } from '@/lib/api'
import { formatBookingDay, formatTime, localDate } from '@/lib/time'
import { usePageTitle } from '@/hooks/usePageTitle'
import { useQrCamera } from '@/hooks/useQrCamera'
import { CameraProblem } from '@/components/CameraProblem'

/*
  Entrance reader: a demo of what a device at the door runs. It signs in with
  the reader key (from /lector#key=… or typed in), asks for the camera, reads
  booking QR codes, and the flash booking QR (which books a free turn on the
  spot), and shows the answer full screen for a few seconds.
  An IoT device can skip this page and call POST /api/reader/checkin itself.
*/

const KEY_STORAGE = 'lq.readerKey'
const SHOW_GRANTED_MS = 2500
const SHOW_REFUSED_MS = 4000

function storedKey() {
  try {
    return localStorage.getItem(KEY_STORAGE) ?? ''
  } catch {
    return ''
  }
}

function storeKey(key: string) {
  try {
    if (key) localStorage.setItem(KEY_STORAGE, key)
    else localStorage.removeItem(KEY_STORAGE)
  } catch {
    // private mode: the key lasts until the page is closed
  }
}

/** Takes the key from the link's fragment once, and drops it from the address bar. */
function keyFromLink() {
  const key = new URLSearchParams(window.location.hash.slice(1)).get('key')
  if (!key) return ''
  storeKey(key)
  window.history.replaceState(null, '', window.location.pathname)
  return key
}

export function ReaderPage() {
  const [key, setKey] = useState(() => keyFromLink() || storedKey())
  const reader = useAsync(key ? () => api.reader.me(key) : null, key)
  const rejected = reader.error instanceof ApiError && reader.error.status === 401
  const { t } = useTranslation('reader')
  usePageTitle(reader.data?.facility_name ?? t('signIn.title'))

  const signIn = (value: string) => {
    storeKey(value)
    setKey(value)
  }
  const signOut = () => {
    storeKey('')
    setKey('')
  }

  if (!key || rejected) return <KeyForm rejected={rejected} onSubmit={signIn} />
  if (!reader.data) return <Splash error={reader.error} onRetry={reader.reload} />
  return <Scanner deviceKey={key} reader={reader.data} onSignOut={signOut} />
}

function Splash({ error, onRetry }: { error: Error | null; onRetry: () => void }) {
  const { t } = useTranslation('reader')
  return (
    <div className="grid min-h-dvh place-items-center bg-background p-6">
      {error ? (
        <div className="grid justify-items-center gap-4 text-center">
          <WifiOff className="size-10 text-muted-foreground" aria-hidden />
          <p className="text-lg">{t('offline.title')}</p>
          <Button onClick={onRetry}>{t('retry')}</Button>
        </div>
      ) : (
        <BrandLogo className="size-12 animate-pulse" />
      )}
    </div>
  )
}

function KeyForm({ rejected, onSubmit }: { rejected: boolean; onSubmit: (key: string) => void }) {
  const { t } = useTranslation('reader')
  const [value, setValue] = useState('')
  const submit = (event: FormEvent) => {
    event.preventDefault()
    if (value.trim()) onSubmit(value.trim())
  }
  return (
    <main id="main" className="grid min-h-dvh place-items-center bg-background p-4">
      <form onSubmit={submit} className="grid w-full max-w-md gap-4 rounded-lg border bg-card p-6">
        <div className="flex items-center gap-3">
          <BrandLogo className="size-8" />
          <h1 className="text-xl font-semibold">{t('signIn.title')}</h1>
        </div>
        <p className="text-muted-foreground">{t('signIn.help')}</p>
        {rejected && (
          <Alert variant="destructive">
            <AlertDescription>{t('signIn.rejected')}</AlertDescription>
          </Alert>
        )}
        <div className="grid gap-1.5">
          <Label htmlFor="reader-key">{t('signIn.key')}</Label>
          <Input
            id="reader-key"
            value={value}
            autoComplete="off"
            spellCheck={false}
            className="font-mono"
            placeholder="lqd_…"
            onChange={(e) => setValue(e.target.value)}
          />
        </div>
        <Button type="submit" disabled={!value.trim()}>
          {t('signIn.submit')}
        </Button>
      </form>
    </main>
  )
}

type Shown = { result: CheckinResult } | { offline: true }

function beep(granted: boolean) {
  try {
    const audio = new AudioContext()
    const tone = audio.createOscillator()
    const gain = audio.createGain()
    tone.frequency.value = granted ? 880 : 220
    gain.gain.value = 0.15
    tone.connect(gain).connect(audio.destination)
    tone.start()
    tone.stop(audio.currentTime + (granted ? 0.15 : 0.4))
    tone.onended = () => void audio.close()
  } catch {
    // no sound before the first touch on some browsers: the screen is enough
  }
}

function Scanner({ deviceKey, reader, onSignOut }: { deviceKey: string; reader: Reader; onSignOut: () => void }) {
  const { t } = useTranslation('reader')
  const busy = useRef(false)
  const [shown, setShown] = useState<Shown | null>(null)
  const [now, setNow] = useState(() => new Date())

  // the clock, and the screen kept awake while the page is open
  useEffect(() => {
    const tick = setInterval(() => setNow(new Date()), 1000)
    let lock: WakeLockSentinel | null = null
    navigator.wakeLock
      ?.request('screen')
      .then((l) => (lock = l))
      .catch(() => {})
    return () => {
      clearInterval(tick)
      void lock?.release()
    }
  }, [])

  const check = useCallback(
    async (code: string) => {
      busy.current = true
      let next: Shown
      try {
        next = { result: await api.reader.checkIn(deviceKey, code) }
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          onSignOut() // the reader was removed in the administration
          return
        }
        next = { offline: true }
      }
      const granted = 'result' in next && next.result.granted
      beep(granted)
      setShown(next)
      setTimeout(() => {
        setShown(null)
        busy.current = false
      }, granted ? SHOW_GRANTED_MS : SHOW_REFUSED_MS)
    },
    [deviceKey, onSignOut],
  )
  // reading, skipped while an answer is on screen
  const { video, camera, mirrored, retry } = useQrCamera((code) => void check(code), () => busy.current)

  const dismiss = () => {
    setShown(null)
    busy.current = false
  }

  return (
    <div className="flex min-h-dvh flex-col bg-background">
      <header className="flex items-center justify-between gap-4 border-b bg-card px-4 py-3 sm:px-6">
        <div className="flex min-w-0 items-center gap-3">
          <BrandLogo className="size-9 shrink-0" />
          <div className="min-w-0">
            <h1 className="truncate text-lg font-semibold sm:text-xl">{reader.facility_name}</h1>
            <p className="truncate text-muted-foreground">{reader.tenant_name}</p>
          </div>
        </div>
        <p className="num shrink-0 text-2xl font-semibold sm:text-3xl">
          <span className="sr-only">{t('clock')}: </span>
          {formatTime(now.toISOString(), reader.timezone)}
        </p>
      </header>

      <main id="main" tabIndex={-1} className="flex flex-1 flex-col items-center justify-center gap-5 p-4 sm:p-6">
        <div className="relative aspect-square w-full max-w-[min(32rem,60dvh)] overflow-hidden rounded-xl bg-black">
          <video ref={video} muted playsInline className={`size-full object-cover ${mirrored ? '-scale-x-100' : ''}`} />
          {camera === 'running' ? (
            // the frame to aim at
            <div className="pointer-events-none absolute inset-[14%] rounded-lg border-4 border-white/80 shadow-[0_0_0_100vmax_rgb(0_0_0/0.35)]" aria-hidden />
          ) : (
            <CameraProblem state={camera} onRetry={retry} />
          )}
        </div>
        <p className="max-w-md text-center text-lg text-balance">{t('instructions')}</p>
      </main>

      <footer className="flex flex-wrap items-center justify-between gap-2 border-t px-4 py-2 text-muted-foreground sm:px-6">
        <span>{t('device', { name: reader.name })}</span>
        <Button variant="ghost" size="sm" onClick={onSignOut}>
          {t('signOut')}
        </Button>
      </footer>

      {/* always mounted, so each answer is announced as it appears (WCAG 4.1.3) */}
      <div role="status">{shown && <Answer shown={shown} reader={reader} now={now} onDismiss={dismiss} />}</div>
    </div>
  )
}


/** The answer, full screen: green to go in, red (or blue when it is early) to stop. */
function Answer({ shown, reader, now, onDismiss }: { shown: Shown; reader: Reader; now: Date; onDismiss: () => void }) {
  const { t } = useTranslation('reader')
  const zone = reader.timezone
  const today = localDate(now.toISOString(), zone)

  let tone = 'bg-pill-negative-bg text-pill-negative-ink'
  let Icon = CircleX
  let title: string
  let detail = ''

  if ('offline' in shown) {
    Icon = WifiOff
    title = t('offline.title')
    detail = t('offline.detail')
  } else {
    const { result, booking } = shown.result
    const slot = booking
      ? {
          facility: booking.facility_name,
          day: formatBookingDay(booking.starts_at, zone, today),
          start: formatTime(booking.starts_at, zone),
          end: formatTime(booking.ends_at, zone),
        }
      : null
    if (shown.result.granted) {
      tone = 'bg-pill-positive-bg text-pill-positive-ink'
      Icon = CircleCheck
    }
    switch (result) {
      case 'checked_in':
        title = booking?.first_name ? t('result.welcome', { name: booking.first_name }) : t('result.checkedIn')
        detail = slot ? t('result.slot', slot) : ''
        break
      case 'already_checked_in':
        title = t('result.alreadyIn')
        detail = booking?.checked_in_at ? t('result.alreadyInDetail', { ...slot, time: formatTime(booking.checked_in_at, zone) }) : ''
        break
      case 'too_early': {
        tone = 'bg-pill-accent-bg text-pill-accent-ink'
        Icon = Clock
        title = t('result.tooEarly')
        const from = booking ? formatTime(new Date(new Date(booking.starts_at).getTime() - 15 * 60_000).toISOString(), zone) : ''
        detail = slot ? t('result.tooEarlyDetail', { ...slot, from }) : ''
        break
      }
      case 'ended':
        title = t('result.ended')
        detail = slot ? t('result.endedDetail', slot) : ''
        break
      case 'other_facility':
        title = t('result.otherFacility')
        detail = slot ? t('result.otherFacilityDetail', slot) : ''
        break
      case 'cancelled':
        title = t('result.cancelled')
        break
      case 'booked':
        title = booking?.first_name ? t('result.welcome', { name: booking.first_name }) : t('result.booked')
        detail = slot ? t('result.bookedDetail', slot) : ''
        break
      case 'not_booked':
        title = t('result.notBooked')
        detail = t(`reason.${shown.result.reason}`, { defaultValue: t('reason.other') })
        break
      case 'code_expired':
        title = t('result.codeExpired')
        detail = t('result.codeExpiredDetail')
        break
      case 'unknown_booking':
        title = t('result.unknown')
        detail = t('result.unknownDetail')
        break
      default:
        title = t('result.notABooking')
        detail = t('result.notABookingDetail')
    }
  }

  return (
    <button
      type="button"
      onClick={onDismiss}
      className={`fixed inset-0 z-50 grid cursor-default place-content-center justify-items-center gap-5 p-8 text-center ${tone}`}
    >
      <Icon className="size-24 sm:size-32" strokeWidth={1.75} aria-hidden />
      <span className="grid gap-3">
        <span className="text-3xl font-semibold text-balance sm:text-5xl">{title}</span>
        {detail && <span className="num max-w-2xl text-xl text-balance sm:text-2xl">{detail}</span>}
      </span>
    </button>
  )
}
