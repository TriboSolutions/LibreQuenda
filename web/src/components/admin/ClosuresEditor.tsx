import { CalendarOff } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { DatePicker } from '@/components/DatePicker'
import { TimeSelect } from '@/components/TimeSelect'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, type Closure, type ClosureInput } from '@/lib/api'
import { formatDateTime, localDate } from '@/lib/time'

type Props = {
  slug: string
  facilityId: string
  token: string
  timeZone: string // the council's: the dates are typed in it
  initial: Closure[]
}

/**
 * Closures: works, maintenance, an event. The slots show as closed, and the bookings
 * inside are cancelled with a "sorry" notice and email that carry the reason.
 */
export function ClosuresEditor({ slug, facilityId, token, timeZone, initial }: Props) {
  const { t } = useTranslation(['admin', 'common'])
  const [closures, setClosures] = useState<Closure[]>(initial)
  // a whole day unless the times are changed: closures are mostly full days
  const blank = { fromDate: '', fromTime: '00:00', untilDate: '', untilTime: '23:59', reason: '' }
  const [fields, setFields] = useState(blank)
  const [today] = useState(() => localDate(new Date().toISOString(), timeZone))
  const set = (field: keyof typeof blank, value: string) => setFields((f) => ({ ...f, [field]: value }))
  // the API takes local date-times in the council's time zone
  const form: ClosureInput = {
    starts_at: fields.fromDate && `${fields.fromDate}T${fields.fromTime}`,
    ends_at: fields.untilDate && `${fields.untilDate}T${fields.untilTime}`,
    reason: fields.reason,
  }
  const missingDates = !fields.fromDate || !fields.untilDate
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [toConfirm, setToConfirm] = useState<number | null>(null) // bookings the closure would cancel

  const attempt = async (action: () => Promise<void>) => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await action()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setToConfirm(null)
    } finally {
      setBusy(false)
    }
  }

  const create = () =>
    attempt(async () => {
      const { closure, cancelled } = await api.council.createClosure(token, slug, facilityId, form)
      setClosures((list) => [...list, closure].sort((a, b) => a.starts_at.localeCompare(b.starts_at)))
      setFields(blank)
      setToConfirm(null)
      setNotice(cancelled > 0 ? t('admin:closures.saved', { count: cancelled }) : t('admin:closures.savedNone'))
    })

  // first ask how many bookings it would cancel; with none, there is nothing to confirm
  const submit = (event: FormEvent) => {
    event.preventDefault()
    void attempt(async () => {
      const { bookings_to_cancel: count } = await api.council.previewClosure(token, slug, facilityId, form)
      if (count > 0) setToConfirm(count)
      else await create()
    })
  }

  const remove = (closure: Closure) =>
    void attempt(async () => {
      await api.council.deleteClosure(token, slug, facilityId, closure.id)
      setClosures((list) => list.filter((c) => c.id !== closure.id))
      setNotice(t('admin:closures.deleted'))
    })

  return (
    <section aria-labelledby="sec-closures" className="mt-6 grid gap-4 rounded-lg border bg-card p-5">
      <div>
        <h2 id="sec-closures" className="text-lg font-semibold">
          {t('admin:closures.title')}
        </h2>
        <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('admin:closures.help')}</p>
      </div>

      <div aria-live="polite" className="empty:hidden">
        {notice && (
          <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
            <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
          </Alert>
        )}
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>

      {closures.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('admin:closures.empty')}</p>
      ) : (
        <ul className="divide-y rounded-md border">
          {closures.map((closure) => (
            <li key={closure.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
              <div className="flex min-w-0 items-center gap-3">
                <CalendarOff className="size-5 shrink-0 text-muted-foreground" aria-hidden />
                <div className="min-w-0">
                  <p className="num font-medium">
                    {formatDateTime(closure.starts_at, timeZone)} – {formatDateTime(closure.ends_at, timeZone)}
                  </p>
                  <p className="text-muted-foreground">{closure.reason}</p>
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="text-destructive hover:text-destructive"
                onClick={() => remove(closure)}
                disabled={busy}
                title={t('admin:closures.deleteHint')}
              >
                {t('admin:closures.delete')}
              </Button>
            </li>
          ))}
        </ul>
      )}

      <form onSubmit={submit} className="grid gap-3 sm:grid-cols-2">
        <div className="grid gap-1.5">
          <Label htmlFor="closure-from">{t('admin:closures.from')}</Label>
          <div className="flex gap-2">
            <DatePicker
              id="closure-from"
              value={fields.fromDate}
              onChange={(date) => setFields((f) => ({ ...f, fromDate: date, untilDate: f.untilDate && f.untilDate < date ? date : f.untilDate }))}
              defaultMonth={today}
              placeholder={t('admin:closures.pickDay')}
              className="min-w-0 flex-1"
            />
            <TimeSelect value={fields.fromTime} onChange={(time) => set('fromTime', time)} aria-label={t('admin:closures.fromTime')} className="w-24" />
          </div>
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="closure-until">{t('admin:closures.until')}</Label>
          <div className="flex gap-2">
            <DatePicker
              id="closure-until"
              value={fields.untilDate}
              onChange={(date) => set('untilDate', date)}
              min={fields.fromDate && fields.fromDate > today ? fields.fromDate : today}
              defaultMonth={fields.fromDate || today}
              placeholder={t('admin:closures.pickDay')}
              className="min-w-0 flex-1"
            />
            <TimeSelect value={fields.untilTime} onChange={(time) => set('untilTime', time)} aria-label={t('admin:closures.untilTime')} className="w-24" />
          </div>
        </div>
        <div className="grid gap-1.5 sm:col-span-2">
          <Label htmlFor="closure-reason">{t('admin:closures.reason')}</Label>
          <Input
            id="closure-reason"
            required
            maxLength={255}
            placeholder={t('admin:closures.reasonPlaceholder')}
            value={fields.reason}
            onChange={(e) => set('reason', e.target.value)}
          />
          <p className="text-sm text-muted-foreground">{t('admin:closures.reasonHint')}</p>
        </div>
        <div className="sm:col-span-2">
          <Button type="submit" disabled={busy || missingDates}>
            {busy ? t('admin:closures.adding') : t('admin:closures.add')}
          </Button>
        </div>
      </form>

      {toConfirm !== null && (
        <Dialog open onOpenChange={(open) => !open && !busy && setToConfirm(null)}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>{t('admin:closures.confirmTitle')}</DialogTitle>
              <DialogDescription className="num">{t('admin:closures.confirmBody', { count: toConfirm })}</DialogDescription>
            </DialogHeader>
            <p className="text-muted-foreground">{t('admin:closures.confirmReason', { reason: form.reason.trim() })}</p>
            <DialogFooter>
              <Button variant="outline" onClick={() => setToConfirm(null)} disabled={busy}>
                {t('admin:form.cancel')}
              </Button>
              <Button variant="destructive" onClick={() => void create()} disabled={busy}>
                {busy ? t('admin:closures.adding') : t('admin:closures.confirm', { count: toConfirm })}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </section>
  )
}
