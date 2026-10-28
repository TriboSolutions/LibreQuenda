import { useEffect, useState } from 'react'
import QRCode from 'qrcode'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

/** A QR for the entrance readers. A new code replaces the old one only once it is drawn. */
export function QrImage({ code, alt, className }: { code: string; alt: string; className?: string }) {
  const [svg, setSvg] = useState<string | null>(null)

  useEffect(() => {
    if (!code) return
    let cancelled = false
    // dark on white even in dark mode: readers expect the standard contrast
    QRCode.toString(code, { type: 'svg', errorCorrectionLevel: 'M', margin: 2, color: { dark: '#000000', light: '#ffffff' } })
      .then((markup) => !cancelled && setSvg(markup))
      .catch(() => !cancelled && setSvg(null))
    return () => {
      cancelled = true
    }
  }, [code])

  return (
    <div className={cn('mx-auto w-full max-w-64 rounded-lg bg-white p-2 ring-1 ring-border', className)}>
      {svg ? (
        <img src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`} alt={alt} className="aspect-square w-full [image-rendering:pixelated]" />
      ) : (
        <Skeleton className="aspect-square w-full bg-muted" />
      )}
    </div>
  )
}
