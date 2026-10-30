import { useEffect, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'

const VIEW = 256 // the frame on screen, CSS px
const OUT = 512 // the square photo that is uploaded
const MAX_ZOOM = 4
const STEP = 10 // px per arrow key; Shift moves four times as much

type Props = {
  source: string // an object URL or a same-origin URL: the canvas must be able to read it
  onCancel: () => void
  onCrop: (photo: File) => void
}

type View = { scale: number; x: number; y: number } // the image's top-left corner inside the frame

/**
 * Frames a profile photo in a circle: drag or arrow keys to move it, the slider
 * or + and − to zoom. "Use this photo" draws what the circle shows into a
 * square image, so the app, its storage and the Zitadel copy are the same one.
 */
export function AvatarCropDialog({ source, onCancel, onCrop }: Props) {
  const { t } = useTranslation()
  const [image, setImage] = useState<HTMLImageElement | null>(null)
  const [failed, setFailed] = useState(false)
  const [zoom, setZoom] = useState(1)
  const [view, setView] = useState<View>({ scale: 1, x: 0, y: 0 })
  const [busy, setBusy] = useState(false)
  const drag = useRef<{ x: number; y: number; view: View } | null>(null)

  useEffect(() => {
    const img = new Image()
    img.onload = () => {
      const scale = VIEW / Math.min(img.naturalWidth, img.naturalHeight) // fills the circle
      setImage(img)
      setView({ scale, x: (VIEW - img.naturalWidth * scale) / 2, y: (VIEW - img.naturalHeight * scale) / 2 })
    }
    img.onerror = () => setFailed(true)
    img.src = source
  }, [source])

  // the image always covers the whole frame
  const clamp = (next: View): View => {
    if (!image) return next
    const w = image.naturalWidth * next.scale
    const h = image.naturalHeight * next.scale
    return { scale: next.scale, x: Math.min(0, Math.max(VIEW - w, next.x)), y: Math.min(0, Math.max(VIEW - h, next.y)) }
  }

  const move = (dx: number, dy: number) => setView((v) => clamp({ ...v, x: v.x + dx, y: v.y + dy }))

  // zooms around the frame's centre, so what is in the middle stays there
  const zoomTo = (next: number) => {
    if (!image) return
    const z = Math.min(MAX_ZOOM, Math.max(1, next))
    setZoom(z)
    setView((v) => {
      const base = VIEW / Math.min(image.naturalWidth, image.naturalHeight)
      const scale = base * z
      const cx = (VIEW / 2 - v.x) / v.scale
      const cy = (VIEW / 2 - v.y) / v.scale
      return clamp({ scale, x: VIEW / 2 - cx * scale, y: VIEW / 2 - cy * scale })
    })
  }

  const onKeyDown = (event: KeyboardEvent) => {
    const step = event.shiftKey ? STEP * 4 : STEP
    const keys: Record<string, () => void> = {
      ArrowLeft: () => move(step, 0),
      ArrowRight: () => move(-step, 0),
      ArrowUp: () => move(0, step),
      ArrowDown: () => move(0, -step),
      '+': () => zoomTo(zoom + 0.1),
      '=': () => zoomTo(zoom + 0.1),
      '-': () => zoomTo(zoom - 0.1),
    }
    const action = keys[event.key]
    if (action) {
      event.preventDefault()
      action()
    }
  }

  const onPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    event.currentTarget.setPointerCapture(event.pointerId)
    drag.current = { x: event.clientX, y: event.clientY, view }
  }
  const onPointerMove = (event: PointerEvent<HTMLDivElement>) => {
    const start = drag.current
    if (!start) return
    setView(clamp({ ...start.view, x: start.view.x + event.clientX - start.x, y: start.view.y + event.clientY - start.y }))
  }

  const crop = () => {
    if (!image) return
    setBusy(true)
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = OUT
    const context = canvas.getContext('2d')
    if (!context) return setBusy(false)
    const size = VIEW / view.scale
    context.drawImage(image, -view.x / view.scale, -view.y / view.scale, size, size, 0, 0, OUT, OUT)
    canvas.toBlob(
      (blob) => {
        setBusy(false)
        if (blob) onCrop(new File([blob], `avatar.${blob.type.split('/')[1] ?? 'png'}`, { type: blob.type }))
        else setFailed(true)
      },
      'image/webp', // browsers without WebP encoding give a PNG; the upload takes both
      0.9,
    )
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('imagePicker.cropTitle')}</DialogTitle>
          <DialogDescription id="avatar-crop-help">{t('imagePicker.cropHelp')}</DialogDescription>
        </DialogHeader>

        {failed ? (
          <p role="alert" className="text-destructive">
            {t('imagePicker.cropFailed')}
          </p>
        ) : (
          <div className="grid justify-items-center gap-4">
            <div
              role="group"
              tabIndex={0}
              aria-label={t('imagePicker.cropFrame')}
              aria-describedby="avatar-crop-help"
              onKeyDown={onKeyDown}
              onPointerDown={onPointerDown}
              onPointerMove={onPointerMove}
              onPointerUp={() => (drag.current = null)}
              onPointerCancel={() => (drag.current = null)}
              className="relative cursor-move touch-none overflow-hidden rounded-md bg-muted outline-none select-none focus-visible:ring-3 focus-visible:ring-ring"
              style={{ width: VIEW, height: VIEW }}
            >
              {image && (
                <img
                  src={source}
                  alt=""
                  draggable={false}
                  className="pointer-events-none absolute max-w-none"
                  style={{ left: view.x, top: view.y, width: image.naturalWidth * view.scale, height: image.naturalHeight * view.scale }}
                />
              )}
              {/* what stays: the circle; the corners are shaded */}
              <span aria-hidden className="pointer-events-none absolute inset-0 rounded-full shadow-[0_0_0_9999px_rgb(0_0_0/0.55)] ring-2 ring-white/80" />
            </div>
            <div className="grid w-full max-w-64 gap-2">
              <Label htmlFor="avatar-zoom">{t('imagePicker.zoom')}</Label>
              <input
                id="avatar-zoom"
                type="range"
                min={1}
                max={MAX_ZOOM}
                step={0.05}
                value={zoom}
                disabled={!image}
                onChange={(e) => zoomTo(Number(e.target.value))}
                className="w-full accent-primary"
              />
            </div>
          </div>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onCancel}>
            {t('imagePicker.cropCancel')}
          </Button>
          <Button type="button" disabled={!image || busy || failed} onClick={crop}>
            {t('imagePicker.cropUse')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
