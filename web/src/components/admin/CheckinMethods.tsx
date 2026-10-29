import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Hash, MonitorSmartphone, Nfc, Printer, QrCode, RefreshCw, ScanLine, UserCheck } from 'lucide-react'
import { CheckinDevices } from '@/components/admin/CheckinDevices'
import { QrImage } from '@/components/QrImage'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { useAsync } from '@/hooks/useAsync'
import { api, type CheckinSettings, type CheckinSettingsInput } from '@/lib/api'
import { entranceUrl } from '@/lib/entrance'

type Props = { slug: string; facilityId: string; facilitySlug: string; token: string }

type Method = 'reader' | 'printedQr' | 'nfc' | 'rotating' | 'entryCode' | 'staff'

type SavedKey = 'printed_qr' | 'entry_code' | 'staff_confirm' | 'qr_reader'

// the ways in from the README's "Check-in at the entrance". saved: switched on
// and off with the facility's settings; the rest is a preview
const METHODS: { key: Method; icon: typeof QrCode; built?: boolean; saved?: SavedKey }[] = [
  { key: 'reader', icon: ScanLine, built: true, saved: 'qr_reader' },
  { key: 'printedQr', icon: QrCode, built: true, saved: 'printed_qr' },
  { key: 'entryCode', icon: Hash, built: true, saved: 'entry_code' },
  { key: 'nfc', icon: Nfc },
  { key: 'rotating', icon: MonitorSmartphone },
  { key: 'staff', icon: UserCheck, built: true, saved: 'staff_confirm' },
]

// the ways that check the phone's location against the facility's map pin
const needsPin = (key?: SavedKey) => key === 'printed_qr' || key === 'entry_code'

function toInput(settings: CheckinSettings): CheckinSettingsInput {
  return {
    printed_qr: settings.printed_qr,
    entry_code: settings.entry_code,
    radius_m: settings.radius_m,
    flash_booking: settings.flash_booking,
    staff_confirm: settings.staff_confirm,
    qr_reader: settings.qr_reader,
  }
}

/** How residents check in at this facility: one or several ways at once. */
export function CheckinMethods(props: Props) {
  const { t } = useTranslation(['admin', 'common'])
  const { slug, facilityId, token } = props
  const settings = useAsync(() => api.council.checkinSettings(token, slug, facilityId), `checkin:${slug}/${facilityId}`)

  return (
    <section aria-labelledby="sec-checkin" className="grid gap-4 rounded-lg border bg-card p-5">
      <div>
        <h2 id="sec-checkin" className="text-lg font-semibold">
          {t('admin:checkin.title')}
        </h2>
        <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('admin:checkin.help')}</p>
      </div>
      {settings.error ? (
        <div className="flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
          <p>{t('admin:checkin.loadError')}</p>
          <Button variant="ghost" size="sm" onClick={settings.reload}>
            {t('common:retry')}
          </Button>
        </div>
      ) : !settings.data ? (
        <Skeleton className="h-40 w-full" />
      ) : (
        <Methods {...props} initial={settings.data} />
      )}
    </section>
  )
}

function Methods({ slug, facilityId, facilitySlug, token, initial }: Props & { initial: CheckinSettings }) {
  const { t } = useTranslation(['admin', 'common'])
  const [saved, setSaved] = useState(initial)
  const [draft, setDraft] = useState<CheckinSettingsInput>(() => toInput(initial))
  const [saving, setSaving] = useState(false)
  const [status, setStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  // which print gets a new token or code: the ones already printed stop working
  const [rotateOpen, setRotateOpen] = useState<'printedQr' | 'entryCode' | null>(null)
  const [rotating, setRotating] = useState(false)

  // the ways that are only a preview: switched locally, never saved
  const [preview, setPreview] = useState<Record<'nfc' | 'rotating', boolean>>({ nfc: false, rotating: false })
  const [tagType, setTagType] = useState('signed')
  const [codeMinutes, setCodeMinutes] = useState('2')
  const [codeDigits, setCodeDigits] = useState('6')

  const dirty = JSON.stringify(draft) !== JSON.stringify(toInput(saved))
  const needsLocation = draft.printed_qr || draft.entry_code

  const set = <K extends keyof CheckinSettingsInput>(key: K, value: CheckinSettingsInput[K]) => setDraft((d) => ({ ...d, [key]: value }))

  const save = async () => {
    setSaving(true)
    setError(null)
    try {
      const updated = await api.council.saveCheckinSettings(token, slug, facilityId, draft)
      setSaved(updated)
      setDraft(toInput(updated))
      setStatus(t('admin:checkin.saved'))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  const rotate = async (which: 'printedQr' | 'entryCode') => {
    setRotating(true)
    setError(null)
    try {
      const updated =
        which === 'printedQr'
          ? await api.council.rotateCheckinQr(token, slug, facilityId)
          : await api.council.rotateCheckinCode(token, slug, facilityId)
      setSaved(updated)
      setStatus(t(`admin:checkin.${which}.rotated`))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setRotating(false)
      setRotateOpen(null)
    }
  }

  const printBase = `/${slug}/admin/facilities/${facilityId}/imprimir`
  const isOn = (key: Method, saved?: SavedKey) => (saved ? draft[saved] : preview[key as keyof typeof preview])
  const toggle = (key: Method, on: boolean, savedKey?: SavedKey) =>
    savedKey ? set(savedKey, on) : setPreview((p) => ({ ...p, [key]: on }))

  const settings: Record<Method, ReactNode> = {
    reader: <CheckinDevices nested slug={slug} facilityId={facilityId} token={token} />,
    printedQr: (
      <>
        <p className="text-sm text-muted-foreground">
          {t('admin:checkin.printedQr.address', { url: `${window.location.host}/${slug}/${facilitySlug}/entrada` })}
        </p>
        {saved.printed_qr ? (
          <div className="flex flex-wrap items-center gap-4">
            <QrImage
              code={entranceUrl(slug, facilitySlug, saved.qr_token)}
              alt={t('admin:checkin.printedQr.qrAlt')}
              className="mx-0 max-w-32"
            />
            <div className="flex flex-wrap gap-2">
              <Button asChild variant="outline">
                <Link to={`${printBase}?que=qr`} target="_blank">
                  <Printer aria-hidden />
                  {t('admin:checkin.printedQr.print')}
                </Link>
              </Button>
              <Button type="button" variant="ghost" onClick={() => setRotateOpen('printedQr')}>
                <RefreshCw aria-hidden />
                {t('admin:checkin.printedQr.newQr')}
              </Button>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{t('admin:checkin.saveToPrint')}</p>
        )}
      </>
    ),
    entryCode: (
      <>
        <p className="text-sm text-muted-foreground">{t('admin:checkin.entryCode.help')}</p>
        {saved.entry_code ? (
          <div className="flex flex-wrap items-center gap-4">
            <p className="num font-mono text-3xl font-bold tracking-widest" aria-label={t('admin:checkin.entryCode.codeLabel', { code: saved.code.split('').join(' ') })}>
              {saved.code.slice(0, 3)} {saved.code.slice(3)}
            </p>
            <div className="flex flex-wrap gap-2">
              <Button asChild variant="outline">
                <Link to={`${printBase}?que=codigo`} target="_blank">
                  <Printer aria-hidden />
                  {t('admin:checkin.entryCode.print')}
                </Link>
              </Button>
              <Button type="button" variant="ghost" onClick={() => setRotateOpen('entryCode')}>
                <RefreshCw aria-hidden />
                {t('admin:checkin.entryCode.newCode')}
              </Button>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">{t('admin:checkin.saveToPrint')}</p>
        )}
      </>
    ),
    nfc: (
      <div className="grid gap-2 sm:max-w-sm">
        <Label htmlFor="ci-tag">{t('admin:checkin.nfc.tag')}</Label>
        <Select value={tagType} onValueChange={setTagType}>
          <SelectTrigger id="ci-tag" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="signed">{t('admin:checkin.nfc.signed')}</SelectItem>
            <SelectItem value="plain">{t('admin:checkin.nfc.plain')}</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-sm text-muted-foreground">{tagType === 'signed' ? t('admin:checkin.nfc.signedHelp') : t('admin:checkin.nfc.plainHelp')}</p>
      </div>
    ),
    rotating: (
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid gap-2">
          <Label htmlFor="ci-code-minutes">{t('admin:checkin.rotating.every')}</Label>
          <Select value={codeMinutes} onValueChange={setCodeMinutes}>
            <SelectTrigger id="ci-code-minutes" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {['1', '2', '5'].map((m) => (
                <SelectItem key={m} value={m}>
                  {t('admin:checkin.rotating.minutes', { count: Number(m) })}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-2">
          <Label htmlFor="ci-code-digits">{t('admin:checkin.rotating.digits')}</Label>
          <Select value={codeDigits} onValueChange={setCodeDigits}>
            <SelectTrigger id="ci-code-digits" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {['4', '6', '8'].map((d) => (
                <SelectItem key={d} value={d}>
                  {d}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <p className="text-sm text-muted-foreground sm:col-span-2">{t('admin:checkin.rotating.help')}</p>
      </div>
    ),
    staff: (
      <p className="text-sm text-muted-foreground">
        {t('admin:checkin.staff.help')}{' '}
        <Link to={`/${slug}/admin?vista=reservas&instalacion=${facilityId}`} className="text-primary underline underline-offset-4">
          {t('admin:checkin.staff.open')}
        </Link>
      </p>
    ),
  }

  return (
    // not a form: the readers inside have their own
    <div className="grid gap-4">
      <Alert>
        <AlertDescription>{t('admin:checkin.preview')}</AlertDescription>
      </Alert>
      {!saved.has_location && (
        <Alert variant="destructive">
          <AlertDescription>{t('admin:checkin.location.noPin')}</AlertDescription>
        </Alert>
      )}

      <div className="grid gap-x-8 gap-y-4 rounded-md border p-4 md:grid-cols-2">
        <div className="flex items-start gap-2">
          <Switch
            id="ci-flash"
            checked={draft.flash_booking}
            onCheckedChange={(on) => set('flash_booking', on)}
            aria-describedby="ci-flash-help"
          />
          <div className="grid gap-1">
            <Label htmlFor="ci-flash">{t('admin:checkin.flash.label')}</Label>
            <p id="ci-flash-help" className="text-sm text-muted-foreground">
              {t('admin:checkin.flash.help')}
            </p>
          </div>
        </div>
        <div className="grid content-start gap-2">
          <Label htmlFor="ci-radius">{t('admin:checkin.location.radius')}</Label>
          <Input
            id="ci-radius"
            type="number"
            min={25}
            max={1000}
            step={5}
            required
            value={draft.radius_m}
            onChange={(e) => set('radius_m', Number(e.target.value))}
            aria-describedby="ci-radius-help"
            className="num sm:max-w-xs"
          />
          <p id="ci-radius-help" className="text-sm text-muted-foreground">
            {t('admin:checkin.location.help')}
          </p>
        </div>
      </div>

      <ul className="grid gap-4">
        {METHODS.map(({ key, icon: Icon, built, saved: savedKey }) => {
          const on = isOn(key, savedKey)
          return (
            <li key={key} className="grid gap-4 rounded-md border p-4">
              <div className="flex items-start gap-3">
                <Icon className="mt-0.5 size-5 shrink-0 text-muted-foreground" aria-hidden />
                <div className="grid min-w-0 flex-1 gap-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <Label htmlFor={`ci-${key}`} className="text-base font-semibold">
                      {t(`admin:checkin.${key}.name`)}
                    </Label>
                    <Badge variant={built ? 'default' : 'outline'}>{built ? t('admin:checkin.built') : t('admin:checkin.soon')}</Badge>
                  </div>
                  <p id={`ci-${key}-help`} className="text-muted-foreground">
                    {t(`admin:checkin.${key}.how`)}
                  </p>
                  <dl className="mt-1 flex flex-wrap gap-x-5 gap-y-1 text-sm">
                    <div className="flex gap-1.5">
                      <dt className="text-muted-foreground">{t('admin:checkin.door')}</dt>
                      <dd>{t(`admin:checkin.${key}.door`)}</dd>
                    </div>
                    <div className="flex gap-1.5">
                      <dt className="text-muted-foreground">{t('admin:checkin.cost')}</dt>
                      <dd>{t(`admin:checkin.${key}.cost`)}</dd>
                    </div>
                    <div className="flex gap-1.5">
                      <dt className="text-muted-foreground">{t('admin:checkin.cheat')}</dt>
                      <dd>{t(`admin:checkin.${key}.cheat`)}</dd>
                    </div>
                  </dl>
                </div>
                <Switch
                  id={`ci-${key}`}
                  checked={on}
                  disabled={needsPin(savedKey) && !saved.has_location && !on}
                  aria-describedby={`ci-${key}-help`}
                  onCheckedChange={(checked) => toggle(key, checked, savedKey)}
                />
              </div>
              {key === 'reader' && !on && <p className="text-sm text-muted-foreground">{t('admin:checkin.reader.off')}</p>}
              {/* the readers stay listed while off, to manage or remove them */}
              {(on || key === 'reader') && <div className="grid gap-4 border-t pt-4">{settings[key]}</div>}
            </li>
          )
        })}
      </ul>

      {needsLocation && (
        <p className="text-sm text-muted-foreground">{t('admin:checkin.location.required', { metres: draft.radius_m })}</p>
      )}
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}
      <div className="flex flex-wrap items-center justify-end gap-3">
        <p role="status" className="text-sm text-muted-foreground">
          {dirty ? null : status}
        </p>
        <Button type="button" disabled={!dirty || saving} onClick={() => void save()}>
          {saving ? t('admin:checkin.saving') : t('admin:checkin.save')}
        </Button>
      </div>

      <Dialog open={rotateOpen !== null} onOpenChange={(open) => !open && setRotateOpen(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t(`admin:checkin.${rotateOpen ?? 'printedQr'}.newTitle`)}</DialogTitle>
            <DialogDescription>{t(`admin:checkin.${rotateOpen ?? 'printedQr'}.newBody`)}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setRotateOpen(null)}>
              {t(`admin:checkin.${rotateOpen ?? 'printedQr'}.keep`)}
            </Button>
            <Button type="button" variant="destructive" disabled={rotating} onClick={() => rotateOpen && void rotate(rotateOpen)}>
              {t(`admin:checkin.${rotateOpen ?? 'printedQr'}.newConfirm`)}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
