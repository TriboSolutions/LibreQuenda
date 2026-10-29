import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { CircleCheck } from 'lucide-react'
import { DoorQrScanner } from '@/components/DoorQrScanner'
import { EntranceAnswer } from '@/components/EntranceAnswer'
import { QrImage } from '@/components/QrImage'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { api, type Booking, type CheckinMethod, type EntranceResult } from '@/lib/api'
import { GRANTED, locate } from '@/lib/entrance'
import { formatBookingDay, formatTime } from '@/lib/time'

type Props = {
  booking: Booking | null
  token: string | undefined
  timeZone: string
  today: string
  onClose: () => void
  onCheckedIn: () => void
}

// the same order as the flash booking's tabs
const ORDER: CheckinMethod[] = ['reader', 'entry_code', 'printed_qr', 'staff']

/** Every way to check in the booking's facility takes, one tab each, like the flash booking. */
export function CheckinDialog({ booking, token, timeZone, today, onClose, onCheckedIn }: Props) {
  const { t } = useTranslation(['bookings', 'entrance', 'reader'])
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<EntranceResult | null>(null)
  const [problem, setProblem] = useState<string | null>(null)
  const ways = ORDER.filter((way) => booking?.checkin_ways?.includes(way))
  const [way, setWay] = useState<CheckinMethod | undefined>(ways[0])

  if (!booking) return <Dialog open={false} />

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!token) return
    setBusy(true)
    setProblem(null)
    setResult(null)
    try {
      let where
      try {
        where = await locate()
      } catch (e) {
        setProblem(t(`entrance:location.${e instanceof Error ? e.message : 'unavailable'}`))
        return
      }
      const answer = await api.entrance.checkIn(token, booking.tenant_slug, booking.facility_slug, { code, ...where })
      setResult(answer)
      if (GRANTED.includes(answer.result)) onCheckedIn()
    } catch (e) {
      setProblem(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const body: Record<CheckinMethod, React.ReactNode> = {
    entry_code: (
      <form onSubmit={(e) => void submit(e)} className="grid gap-3">
        <p id="checkin-code-help" className="text-muted-foreground">
          {t('checkin.entry_code.help')}
        </p>
        <div className="grid gap-2 sm:max-w-xs">
          <Label htmlFor="checkin-code">{t('entrance:code')}</Label>
          <Input
            id="checkin-code"
            inputMode="numeric"
            autoComplete="one-time-code"
            pattern="[0-9 \-]*"
            maxLength={8}
            required
            value={code}
            onChange={(e) => setCode(e.target.value)}
            aria-describedby="checkin-code-help"
            className="num font-mono text-lg tracking-widest"
          />
        </div>
        <div>
          <Button type="submit" disabled={busy}>
            {busy ? t('entrance:checking') : t('checkin.button')}
          </Button>
        </div>
        <EntranceAnswer result={result} problem={problem} />
      </form>
    ),
    printed_qr: (
      <div className="grid gap-4">
        <p className="text-muted-foreground">{t('checkin.printed_qr.help')}</p>
        <DoorQrScanner book={false} onCheckedIn={onCheckedIn} />
      </div>
    ),
    reader: (
      <div className="grid gap-4">
        <p className="text-muted-foreground">{t('checkin.reader.help')}</p>
        {booking.checkin_code && <QrImage code={booking.checkin_code} alt={t('qr.alt', { facility: booking.facility_name })} className="max-w-64" />}
      </div>
    ),
    staff: <p className="text-muted-foreground">{t('checkin.staff.help')}</p>,
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('checkin.title')}</DialogTitle>
          <DialogDescription className="num">
            {t('checkin.description', {
              facility: booking.facility_name,
              day: formatBookingDay(booking.starts_at, timeZone, today),
              start: formatTime(booking.starts_at, timeZone),
              end: formatTime(booking.ends_at, timeZone),
            })}
          </DialogDescription>
        </DialogHeader>

        {booking.checked_in_at ? (
          <p className="num flex items-center gap-1.5 rounded-md bg-pill-positive-bg px-3 py-2 text-pill-positive-ink">
            <CircleCheck className="size-4" aria-hidden />
            {t('checkin.done', { time: formatTime(booking.checked_in_at, timeZone) })}
          </p>
        ) : ways.length === 1 ? (
          body[ways[0]]
        ) : (
          <Tabs value={way} onValueChange={(value) => setWay(value as CheckinMethod)} className="gap-4">
            <TabsList aria-label={t('checkin.ways')} className={`grid h-auto w-full ${ways.length > 2 ? 'grid-cols-2 sm:grid-cols-4' : 'grid-cols-2'}`}>
              {ways.map((key) => (
                <TabsTrigger key={key} value={key}>
                  {t(`checkin.${key}.tab`)}
                </TabsTrigger>
              ))}
            </TabsList>
            {ways.map((key) => (
              <TabsContent key={key} value={key}>
                {body[key]}
              </TabsContent>
            ))}
          </Tabs>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('checkin.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
