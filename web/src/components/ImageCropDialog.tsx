import { useEffect, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'

const VIEW = 256 // the frame on screen, CSS px
const OUT = 512 // the square image that is uploaded
const MAX_ZOOM = 4 // against the image filling the frame
const STEP = 10 // px per arrow key; Shift moves four times as much

export type CropFrame = 'circle' | 'square'

type Props = {
  source: string // an object URL or a same-origin URL: the canvas must be able to read it
  frame: CropFrame
  onCancel: () => void
  onCrop: (image: File) => void
}

type View = { scale: number; x: number; y: number } // the image's top-left corner inside the frame

/**
 * Frames an image: drag or arrow keys to move it, the slider or + and − to zoom.
 * - circle (a profile photo): the photo always fills the circle.
 * - square (a council's crest or logo): it opens showing the whole image, and can
 *   be zoomed in to cut it; what does not reach the edges stays transparent.
 * "Use" draws what the frame shows into a square image, so what is stored is what
 * was framed.
 */
export function ImageCropDialog({ source, frame, onCancel, onCrop }: Props) {
  const { t } = useTranslation()
  const [image, setImage] = useState<HTMLImageElement | null>(null)
  const [failed, setFailed] = useState(false)
  const [zoom, setZoom] = useState(1)
  const [view, setView] = useState<View>({ scale: 1, x: 0, y: 0 })
  const [busy, setBusy] = useState(false)
  const drag = useRef<{ x: number; y: number; view: View } | null>(null)
  const logo = frame === 'square'

  // zoom 1: the photo covers the circle; the logo fits whole in the square
  const baseScale = (img: HTMLImageElement) =>
    VIEW / (logo ? Math.max(img.naturalWidth, img.naturalHeight) : Math.min(img.naturalWidth, img.naturalHeight))
  // a logo can be zoomed in until it fills the square, and as far again as a photo
  const maxZoom = image && logo ? (MAX_ZOOM * Math.max(image.naturalWidth, image.naturalHeight)) / Math.min(image.naturalWidth, image.naturalHeight) : MAX_ZOOM

  useEffect(() => {
    const img = new Image()
    img.onload = () => {
      const scale = baseScale(img)
      setImage(img)
      setView({ scale, x: (VIEW - img.naturalWidth * scale) / 2, y: (VIEW - img.naturalHeight * scale) / 2 })
    }
    img.onerror = () => setFailed(true)
    img.src = source
    // eslint-disable-next-line react-hooks/exhaustive-deps -- the frame does not change while the dialog is open
  }, [source])

  // larger than the frame: it covers it; smaller (a zoomed-out logo): it stays inside
  const clamp = (next: View): View => {
    if (!image) return next
    const w = image.naturalWidth * next.scale
    const h = image.naturalHeight * next.scale
    const within = (pos: number, size: number) => Math.min(Math.max(0, VIEW - size), Math.max(Math.min(0, VIEW - size), pos))
    return { scale: next.scale, x: within(next.x, w), y: within(next.y, h) }
  }

  const move = (dx: number, dy: number) => setView((v) => clamp({ ...v, x: v.x + dx, y: v.y + dy }))

  // zooms around the frame's centre, so what is in the middle stays there
  const zoomTo = (next: number) => {
    if (!image) return
    const z = Math.min(maxZoom, Math.max(1, next))
    setZoom(z)
    setView((v) => {
      const scale = baseScale(image) * z
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
    // the frame's coordinates, scaled up to the output; outside the image stays transparent
    const k = OUT / VIEW
    context.drawImage(image, view.x * k, view.y * k, image.naturalWidth * view.scale * k, image.naturalHeight * view.scale * k)
    canvas.toBlob(
      (blob) => {
        setBusy(false)
        const name = logo ? 'logo' : 'avatar'
        if (blob) onCrop(new File([blob], `${name}.${blob.type.split('/')[1] ?? 'png'}`, { type: blob.type }))
        else setFailed(true)
      },
      'image/webp', // keeps transparency; browsers without WebP encoding give a PNG, the upload takes both
      0.9,
    )
  }

  const texts = logo
    ? { title: t('imagePicker.cropLogoTitle'), help: t('imagePicker.cropLogoHelp'), frame: t('imagePicker.cropLogoFrame'), use: t('imagePicker.cropLogoUse') }
    : { title: t('imagePicker.cropTitle'), help: t('imagePicker.cropHelp'), frame: t('imagePicker.cropFrame'), use: t('imagePicker.cropUse') }

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{texts.title}</DialogTitle>
          <DialogDescription id="image-crop-help">{texts.help}</DialogDescription>
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
              aria-label={texts.frame}
              aria-describedby="image-crop-help"
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
              {logo ? (
                // the whole square is kept: only its edge is marked
                <span aria-hidden className="pointer-events-none absolute inset-0 rounded-md ring-2 ring-inset ring-white/80" />
              ) : (
                // what stays: the circle; the corners are shaded
                <span aria-hidden className="pointer-events-none absolute inset-0 rounded-full shadow-[0_0_0_9999px_rgb(0_0_0/0.55)] ring-2 ring-white/80" />
              )}
            </div>
            <div className="grid w-full max-w-64 gap-2">
              <Label htmlFor="image-zoom">{t('imagePicker.zoom')}</Label>
              <input
                id="image-zoom"
                type="range"
                min={1}
                max={maxZoom}
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
            {texts.use}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
