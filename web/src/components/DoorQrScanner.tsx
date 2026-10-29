import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Camera } from 'lucide-react'
import { CameraProblem } from '@/components/CameraProblem'
import { EntranceAnswer } from '@/components/EntranceAnswer'
import { Button } from '@/components/ui/button'
import { useQrCamera } from '@/hooks/useQrCamera'
import { api, type EntranceResult } from '@/lib/api'
import { GRANTED, locate, parseEntranceUrl } from '@/lib/entrance'

type Props = {
  book: boolean // without a booking: book the free turn and check in (flash booking)
  onCheckedIn?: () => void
}

/**
 * The QR printed on the door, read with the phone's own camera from the app:
 * then the phone's location is checked, as the QR never changes.
 */
export function DoorQrScanner({ book, onCheckedIn }: Props) {
  const { t } = useTranslation('entrance')
  const [scanning, setScanning] = useState(false)
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<EntranceResult | null>(null)
  const [problem, setProblem] = useState<string | null>(null)

  const answer = (next: EntranceResult | null, trouble: string | null) => {
    setResult(next)
    setProblem(trouble)
    if (next && GRANTED.includes(next.result)) onCheckedIn?.()
  }

  const start = () => {
    setResult(null)
    setProblem(null)
    setScanning(true)
  }

  return (
    <div className="grid gap-3">
      {scanning ? (
        // mounted only while scanning, so the camera turns off as soon as a door QR is read
        <DoorCamera book={book} onDoor={() => setScanning(false)} onCancel={() => setScanning(false)} onAnswer={answer} setBusy={setBusy} />
      ) : (
        <div>
          <Button type="button" disabled={busy} onClick={start}>
            <Camera aria-hidden />
            {busy ? t('checking') : result || problem ? t('scan.again') : t('scan.open')}
          </Button>
        </div>
      )}
      <EntranceAnswer result={result} problem={problem} />
    </div>
  )
}

type CameraProps = {
  book: boolean
  onDoor: () => void
  onCancel: () => void
  onAnswer: (result: EntranceResult | null, problem: string | null) => void
  setBusy: (busy: boolean) => void
}

function DoorCamera({ book, onDoor, onCancel, onAnswer, setBusy }: CameraProps) {
  const { t } = useTranslation('entrance')
  const auth = useAuth()
  const token = auth.user?.access_token
  const done = useRef(false)
  const [notOurs, setNotOurs] = useState(false)

  const read = async (text: string) => {
    const door = parseEntranceUrl(text)
    if (!door) {
      setNotOurs(true)
      return
    }
    if (!token) return
    done.current = true
    onDoor()
    setBusy(true)
    try {
      let where
      try {
        where = await locate()
      } catch (e) {
        onAnswer(null, t(`location.${e instanceof Error ? e.message : 'unavailable'}`))
        return
      }
      onAnswer(await api.entrance.checkIn(token, door.slug, door.facility, { qr_token: door.qrToken, ...where, book }), null)
    } catch (e) {
      onAnswer(null, e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }
  const { video, camera, mirrored, retry } = useQrCamera((text) => void read(text), () => done.current)

  return (
    <div className="grid gap-3">
      <div className="relative aspect-square w-full max-w-72 overflow-hidden rounded-lg bg-black">
        <video ref={video} muted playsInline className={`size-full object-cover ${mirrored ? '-scale-x-100' : ''}`} />
        {camera === 'running' ? (
          <div className="pointer-events-none absolute inset-[14%] rounded-lg border-4 border-white/80" aria-hidden />
        ) : (
          <CameraProblem state={camera} onRetry={retry} />
        )}
      </div>
      <p aria-live="polite" className="text-sm text-muted-foreground">
        {notOurs ? t('scan.notOurs') : t('scan.aim')}
      </p>
      <div>
        <Button type="button" variant="outline" onClick={onCancel}>
          {t('scan.cancel')}
        </Button>
      </div>
    </div>
  )
}
