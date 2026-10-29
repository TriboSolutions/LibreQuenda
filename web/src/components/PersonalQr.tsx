import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { LocateFixed } from 'lucide-react'
import { DoorQrScanner } from '@/components/DoorQrScanner'
import { EntranceAnswer } from '@/components/EntranceAnswer'
import { QrImage } from '@/components/QrImage'
import { StaffContacts } from '@/components/StaffContacts'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useMe } from '@/hooks/useMe'
import { api, type EntranceResult, type Nearby, type PersonalQr as Qr } from '@/lib/api'
import { locate, type Position } from '@/lib/entrance'

const RETRY_MS = 10_000

/*
  Flash booking, opened from the account menu: at a facility with a free turn,
  book it on the spot from now until the turn under way ends or the one about
  to start (bookings.FlashTurn in the API), whichever way in the facility takes.
  - Reader: the personal QR. It changes every minute and works once, so a
    screenshot soon stops working; codes are only asked for while its tab is open.
  - Entry code: the phone's location finds the facility, then the code at the
    door books and checks in at once.
  - QR at the door and staff: how to go about it there.
  A booking already made goes in from My bookings instead.
*/
type Way = 'reader' | 'code' | 'door' | 'staff'

export function FlashBookingDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { t } = useTranslation('profile')
  const { me } = useMe()
  const [way, setWay] = useState<Way>('reader')

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('personalQr.menuLink')}</DialogTitle>
          <DialogDescription>{t('flash.help')}</DialogDescription>
        </DialogHeader>

        {me && !me.profile_complete && (
          <Alert>
            <AlertDescription>{t('personalQr.incomplete')}</AlertDescription>
          </Alert>
        )}

        <Tabs value={way} onValueChange={(value) => setWay(value as Way)} className="gap-4">
          <TabsList aria-label={t('flash.ways')} className="grid h-auto w-full grid-cols-2 sm:grid-cols-4">
            {(['reader', 'code', 'door', 'staff'] as const).map((key) => (
              <TabsTrigger key={key} value={key}>
                {t(`flash.${key}.tab`)}
              </TabsTrigger>
            ))}
          </TabsList>
          <TabsContent value="reader" className="grid gap-4">
            <p className="text-muted-foreground">{t('flash.reader.help')}</p>
            {/* mounted only while this tab is open, so it stops asking for codes once left */}
            {open && way === 'reader' && <RotatingCode />}
            <div className="space-y-1.5 text-muted-foreground">
              <p>{t('personalQr.private')}</p>
              <p>{t('personalQr.brightness')}</p>
            </div>
          </TabsContent>
          <TabsContent value="code">
            <CodeFlash />
          </TabsContent>
          <TabsContent value="door" className="grid gap-4">
            <p className="text-muted-foreground">{t('flash.door.help')}</p>
            <DoorQrScanner book />
          </TabsContent>
          <TabsContent value="staff">
            <StaffFlash />
          </TabsContent>
        </Tabs>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('personalQr.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** The entry code: find the facility by the phone's location, then book and check in with its code. */
type Place = Nearby['facilities'][number]

/** The facilities the phone is at, by its location (checked, never stored), kept to those `keep` accepts. */
function useNearby(keep: (place: Place) => boolean) {
  const { t } = useTranslation('entrance')
  const auth = useAuth()
  const token = auth.user?.access_token
  const [busy, setBusy] = useState(false)
  const [position, setPosition] = useState<Position | null>(null)
  const [places, setPlaces] = useState<Place[] | null>(null)
  const [problem, setProblem] = useState<string | null>(null)

  const find = async () => {
    if (!token) return
    setBusy(true)
    setProblem(null)
    try {
      let where: Position
      try {
        where = await locate()
      } catch (e) {
        setProblem(t(`location.${e instanceof Error ? e.message : 'unavailable'}`))
        return
      }
      const answer = await api.entrance.nearby(token, where)
      if (answer.result) {
        setProblem(t(`result.${answer.result}`))
        return
      }
      setPosition(where)
      setPlaces(answer.facilities.filter(keep))
    } catch (e) {
      setProblem(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }
  return { token, busy, setBusy, position, places, problem, setProblem, find }
}

/** Staff to ask on site: the people the council lists for the facility the phone is at. */
function StaffFlash() {
  const { t } = useTranslation('profile')
  const { busy, places, problem, find } = useNearby((place) => place.staff_contacts.length > 0)
  return (
    <div className="grid gap-4">
      <p className="text-muted-foreground">{t('flash.staff.help')}</p>
      {places === null || places.length === 0 ? (
        <div className="grid gap-3">
          {places && <p className="rounded-md border border-dashed px-3 py-2">{t('flash.staff.none')}</p>}
          <div>
            <Button type="button" variant={places ? 'outline' : 'default'} disabled={busy} onClick={() => void find()}>
              <LocateFixed aria-hidden />
              {busy ? t('flash.code.locating') : places ? t('flash.code.again') : t('flash.staff.locate')}
            </Button>
          </div>
        </div>
      ) : (
        places.map((place) => (
          <section key={`${place.tenant_slug}/${place.facility_slug}`} className="grid gap-2">
            <h3 className="font-medium">
              {place.facility_name} <span className="font-normal text-muted-foreground">· {place.tenant_name}</span>
            </h3>
            <StaffContacts contacts={place.staff_contacts} />
          </section>
        ))
      )}
      <EntranceAnswer result={null} problem={problem} />
    </div>
  )
}

function CodeFlash() {
  const { t } = useTranslation(['profile', 'entrance', 'reader'])
  const { token, busy, setBusy, position, places, problem, setProblem, find: locateAndFind } = useNearby((place) => place.entry_code)
  const [chosen, setChosen] = useState('')
  const [code, setCode] = useState('')
  const [result, setResult] = useState<EntranceResult | null>(null)

  const find = async () => {
    setResult(null)
    await locateAndFind()
  }
  // the nearest is chosen as soon as they are found
  const first = places?.[0]
  const chosenOrFirst = chosen || (first ? `${first.tenant_slug}/${first.facility_slug}` : '')

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!token || !position || !chosenOrFirst) return
    const [tenant, facility] = chosenOrFirst.split('/')
    setBusy(true)
    setProblem(null)
    try {
      setResult(await api.entrance.checkIn(token, tenant, facility, { code, ...position, book: true }))
    } catch (e) {
      setResult(null)
      setProblem(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="grid gap-4">
      <p className="text-muted-foreground">{t('flash.code.help')}</p>
      {places === null ? (
        <div>
          <Button type="button" disabled={busy} onClick={() => void find()}>
            <LocateFixed aria-hidden />
            {busy ? t('flash.code.locating') : t('flash.code.locate')}
          </Button>
        </div>
      ) : places.length === 0 ? (
        <div className="grid gap-3">
          <p className="rounded-md border border-dashed px-3 py-2">{t('flash.code.none')}</p>
          <div>
            <Button type="button" variant="outline" disabled={busy} onClick={() => void find()}>
              {t('flash.code.again')}
            </Button>
          </div>
        </div>
      ) : (
        <form onSubmit={(e) => void submit(e)} className="grid gap-4">
          <fieldset className="grid gap-2">
            <legend className="mb-2 font-medium">{t('flash.code.facility')}</legend>
            {places.map((place) => {
              const value = `${place.tenant_slug}/${place.facility_slug}`
              return (
                <label key={value} className="flex min-h-11 items-center gap-3 rounded-md border px-3 py-2 has-checked:border-primary">
                  <input type="radio" name="flash-facility" value={value} checked={chosenOrFirst === value} onChange={() => setChosen(value)} className="size-4 accent-primary" />
                  <span>
                    {place.facility_name} <span className="text-muted-foreground">· {place.tenant_name}</span>
                  </span>
                </label>
              )
            })}
          </fieldset>
          <div className="grid gap-2 sm:max-w-xs">
            <Label htmlFor="flash-code">{t('entrance:code')}</Label>
            <Input
              id="flash-code"
              inputMode="numeric"
              autoComplete="one-time-code"
              pattern="[0-9 \-]*"
              maxLength={8}
              required
              value={code}
              onChange={(e) => setCode(e.target.value)}
              aria-describedby="flash-code-help"
              className="num font-mono text-lg tracking-widest"
            />
            <p id="flash-code-help" className="text-sm text-muted-foreground">
              {t('entrance:codeHelp')}
            </p>
          </div>
          <div>
            <Button type="submit" disabled={busy}>
              {busy ? t('entrance:checking') : t('entrance:bookAndCheckIn')}
            </Button>
          </div>
        </form>
      )}
      <EntranceAnswer result={result} problem={problem} />
    </div>
  )
}

function RotatingCode() {
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
      <div className="grid w-full max-w-64 gap-2">
        <QrImage code={qr.value.code} alt={t('personalQr.alt')} />
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
      <div className="grid w-full max-w-64 content-start gap-3">
        <Alert variant="destructive">
          <AlertDescription>{t('personalQr.failed')}</AlertDescription>
        </Alert>
        <Button variant="outline" onClick={() => setAttempt((n) => n + 1)}>
          {t('personalQr.retry')}
        </Button>
      </div>
    )
  } else {
    code = <Skeleton className="aspect-square w-full max-w-64" />
  }

  return (
    <div aria-live="polite" className="flex justify-center">
      {code}
    </div>
  )
}
