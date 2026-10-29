import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Hash, MonitorSmartphone, Nfc, QrCode, ScanLine, UserCheck } from 'lucide-react'
import { CheckinDevices } from '@/components/admin/CheckinDevices'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

type Props = { slug: string; facilityId: string; facilitySlug: string; token: string }

type Method = 'reader' | 'printedQr' | 'nfc' | 'rotating' | 'dailySheet' | 'staff'

// the ways in from the README's "Check-in at the entrance"; only the reader is built
const METHODS: { key: Method; icon: typeof QrCode; built?: boolean }[] = [
  { key: 'reader', icon: ScanLine, built: true },
  { key: 'printedQr', icon: QrCode },
  { key: 'nfc', icon: Nfc },
  { key: 'rotating', icon: MonitorSmartphone },
  { key: 'dailySheet', icon: Hash },
  { key: 'staff', icon: UserCheck },
]

/**
 * How residents check in at this facility: one or several ways at once.
 * Only a preview for now: nothing here is saved, except the readers, which already work.
 */
export function CheckinMethods({ slug, facilityId, facilitySlug, token }: Props) {
  const { t } = useTranslation('admin')
  const [enabled, setEnabled] = useState<Record<Method, boolean>>({
    reader: true,
    printedQr: false,
    nfc: false,
    rotating: false,
    dailySheet: false,
    staff: false,
  })
  const [flashBooking, setFlashBooking] = useState(true)
  const [windowMinutes, setWindowMinutes] = useState(15)
  const [location, setLocation] = useState(false)
  const [radius, setRadius] = useState(100)
  const [tagType, setTagType] = useState('signed')
  const [codeMinutes, setCodeMinutes] = useState('2')
  const [codeDigits, setCodeDigits] = useState('6')

  const settings: Record<Method, ReactNode> = {
    reader: (
      <>
        <div className="flex items-start gap-2">
          <Switch id="ci-flash" checked={flashBooking} onCheckedChange={setFlashBooking} aria-describedby="ci-flash-help" />
          <div className="grid gap-1">
            <Label htmlFor="ci-flash">{t('checkin.reader.flash')}</Label>
            <p id="ci-flash-help" className="text-sm text-muted-foreground">
              {t('checkin.reader.flashHelp')}
            </p>
          </div>
        </div>
        <CheckinDevices nested slug={slug} facilityId={facilityId} token={token} />
      </>
    ),
    printedQr: (
      <>
        <p className="text-sm text-muted-foreground">
          {t('checkin.printedQr.address', { url: `${window.location.host}/${slug}/${facilitySlug}/entrada` })}
        </p>
        <div className="grid gap-2 sm:max-w-xs">
          <Label htmlFor="ci-window">{t('checkin.printedQr.window')}</Label>
          <Input
            id="ci-window"
            type="number"
            min={0}
            max={60}
            value={windowMinutes}
            onChange={(e) => setWindowMinutes(Number(e.target.value))}
            className="num"
          />
        </div>
        <LocationCheck id="ci-location-qr" on={location} radius={radius} onToggle={setLocation} onRadius={setRadius} />
      </>
    ),
    nfc: (
      <>
        <div className="grid gap-2 sm:max-w-sm">
          <Label htmlFor="ci-tag">{t('checkin.nfc.tag')}</Label>
          <Select value={tagType} onValueChange={setTagType}>
            <SelectTrigger id="ci-tag" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="signed">{t('checkin.nfc.signed')}</SelectItem>
              <SelectItem value="plain">{t('checkin.nfc.plain')}</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-sm text-muted-foreground">{tagType === 'signed' ? t('checkin.nfc.signedHelp') : t('checkin.nfc.plainHelp')}</p>
        </div>
        {tagType === 'plain' && <LocationCheck id="ci-location-nfc" on={location} radius={radius} onToggle={setLocation} onRadius={setRadius} />}
      </>
    ),
    rotating: (
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid gap-2">
          <Label htmlFor="ci-code-minutes">{t('checkin.rotating.every')}</Label>
          <Select value={codeMinutes} onValueChange={setCodeMinutes}>
            <SelectTrigger id="ci-code-minutes" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {['1', '2', '5'].map((m) => (
                <SelectItem key={m} value={m}>
                  {t('checkin.rotating.minutes', { count: Number(m) })}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-2">
          <Label htmlFor="ci-code-digits">{t('checkin.rotating.digits')}</Label>
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
        <p className="text-sm text-muted-foreground sm:col-span-2">{t('checkin.rotating.help')}</p>
      </div>
    ),
    dailySheet: <p className="text-sm text-muted-foreground">{t('checkin.dailySheet.help')}</p>,
    staff: (
      <p className="text-sm text-muted-foreground">
        {t('checkin.staff.help')}{' '}
        <Link to={`/${slug}/admin?vista=reservas&instalacion=${facilityId}`} className="text-primary underline underline-offset-4">
          {t('checkin.staff.open')}
        </Link>
      </p>
    ),
  }

  return (
    <section aria-labelledby="sec-checkin" className="grid gap-4 rounded-lg border bg-card p-5">
      <div>
        <h2 id="sec-checkin" className="text-lg font-semibold">
          {t('checkin.title')}
        </h2>
        <p className="mt-1 max-w-[70ch] text-muted-foreground">{t('checkin.help')}</p>
      </div>
      <Alert>
        <AlertDescription>{t('checkin.preview')}</AlertDescription>
      </Alert>

      <ul className="grid gap-4">
        {METHODS.map(({ key, icon: Icon, built }) => (
          <li key={key} className="grid gap-4 rounded-md border p-4">
            <div className="flex items-start gap-3">
              <Icon className="mt-0.5 size-5 shrink-0 text-muted-foreground" aria-hidden />
              <div className="grid min-w-0 flex-1 gap-1">
                <div className="flex flex-wrap items-center gap-2">
                  <Label htmlFor={`ci-${key}`} className="text-base font-semibold">
                    {t(`checkin.${key}.name`)}
                  </Label>
                  <Badge variant={built ? 'default' : 'outline'}>{built ? t('checkin.built') : t('checkin.soon')}</Badge>
                </div>
                <p id={`ci-${key}-help`} className="text-muted-foreground">
                  {t(`checkin.${key}.how`)}
                </p>
                <dl className="mt-1 flex flex-wrap gap-x-5 gap-y-1 text-sm">
                  <div className="flex gap-1.5">
                    <dt className="text-muted-foreground">{t('checkin.door')}</dt>
                    <dd>{t(`checkin.${key}.door`)}</dd>
                  </div>
                  <div className="flex gap-1.5">
                    <dt className="text-muted-foreground">{t('checkin.cost')}</dt>
                    <dd>{t(`checkin.${key}.cost`)}</dd>
                  </div>
                  <div className="flex gap-1.5">
                    <dt className="text-muted-foreground">{t('checkin.cheat')}</dt>
                    <dd>{t(`checkin.${key}.cheat`)}</dd>
                  </div>
                </dl>
              </div>
              <Switch
                id={`ci-${key}`}
                checked={enabled[key]}
                aria-describedby={`ci-${key}-help`}
                onCheckedChange={(checked) => setEnabled((e) => ({ ...e, [key]: checked }))}
              />
            </div>
            {enabled[key] && <div className="grid gap-4 border-t pt-4">{settings[key]}</div>}
          </li>
        ))}
      </ul>
    </section>
  )
}

function LocationCheck({
  id,
  on,
  radius,
  onToggle,
  onRadius,
}: {
  id: string
  on: boolean
  radius: number
  onToggle: (on: boolean) => void
  onRadius: (metres: number) => void
}) {
  const { t } = useTranslation('admin')
  return (
    <div className="grid gap-3">
      <div className="flex items-start gap-2">
        <Switch id={id} checked={on} onCheckedChange={onToggle} aria-describedby={`${id}-help`} />
        <div className="grid gap-1">
          <Label htmlFor={id}>{t('checkin.location.label')}</Label>
          <p id={`${id}-help`} className="text-sm text-muted-foreground">
            {t('checkin.location.help')}
          </p>
        </div>
      </div>
      {on && (
        <div className="grid gap-2 sm:max-w-xs">
          <Label htmlFor={`${id}-radius`}>{t('checkin.location.radius')}</Label>
          <Input
            id={`${id}-radius`}
            type="number"
            min={10}
            max={1000}
            step={10}
            value={radius}
            onChange={(e) => onRadius(Number(e.target.value))}
            className="num"
          />
        </div>
      )}
    </div>
  )
}
