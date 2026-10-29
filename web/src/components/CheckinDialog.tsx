import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckCircle2, CircleCheck, Hash, QrCode, ScanLine, UserCheck, XCircle } from 'lucide-react'
import { QrImage } from '@/components/QrImage'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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

const ICONS: Record<CheckinMethod, typeof QrCode> = { entry_code: Hash, printed_qr: QrCode, reader: ScanLine, staff: UserCheck }

/** Every way in the booking's facility takes, in one place: type its code here, or how to use the others. */
export function CheckinDialog({ booking, token, timeZone, today, onClose, onCheckedIn }: Props) {
  const { t } = useTranslation(['bookings', 'entrance', 'reader'])
  const [code, setCode] = useState('')
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<EntranceResult | null>(null)
  const [problem, setProblem] = useState<string | null>(null)

  if (!booking) return <Dialog open={false} />
  const ways = booking.checkin_ways ?? []
  const granted = result && GRANTED.includes(result.result)

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
        <p id="checkin-code-help" className="text-sm text-muted-foreground">
          {t('checkin.entry_code.help')}
        </p>
        <div>
          <Button type="submit" disabled={busy}>
            {busy ? t('entrance:checking') : t('checkin.button')}
          </Button>
        </div>
        <div role="status" aria-live="polite" className="empty:hidden">
          {result && (
            <p
              className={`flex items-start gap-2 rounded-md px-3 py-2 ${granted ? 'bg-pill-positive-bg text-pill-positive-ink' : 'bg-pill-negative-bg text-pill-negative-ink'}`}
            >
              {granted ? <CheckCircle2 className="mt-0.5 size-4 shrink-0" aria-hidden /> : <XCircle className="mt-0.5 size-4 shrink-0" aria-hidden />}
              <span>
                {t(`entrance:result.${result.result}`, { defaultValue: t('entrance:result.refused') })}
                {result.result === 'location_inaccurate' && ` ${t('entrance:inaccurateHelp')}`}
                {result.result === 'not_booked' && result.reason && ` ${t(`reader:reason.${result.reason}`, { defaultValue: t('reader:reason.other') })}`}
              </span>
            </p>
          )}
          {problem && <p className="rounded-md bg-pill-negative-bg px-3 py-2 text-pill-negative-ink">{problem}</p>}
        </div>
      </form>
    ),
    printed_qr: <p className="text-muted-foreground">{t('checkin.printed_qr.help')}</p>,
    reader: (
      <div className="grid gap-2">
        <p className="text-muted-foreground">{t('checkin.reader.help')}</p>
        {booking.checkin_code && <QrImage code={booking.checkin_code} alt={t('qr.alt', { facility: booking.facility_name })} className="max-w-56" />}
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
        ) : (
          <>
            <p>{t('checkin.intro')}</p>
            <ul className="grid gap-3">
              {ways.map((way) => {
                const Icon = ICONS[way]
                return (
                  <li key={way} className="grid gap-3 rounded-lg border p-4">
                    <h3 className="flex items-center gap-2 font-medium">
                      <Icon className="size-4" aria-hidden />
                      {t(`checkin.${way}.title`)}
                    </h3>
                    {body[way]}
                  </li>
                )
              })}
            </ul>
          </>
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
