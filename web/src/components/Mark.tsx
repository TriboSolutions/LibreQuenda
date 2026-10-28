import { useId } from 'react'

/**
 * LibreQuenda mark: a clock face crossed by a band, a turn (quenda) taken.
 * Same geometry as docs/brand/librequenda-simbolo.svg, cropped to the ring.
 * Ticks are only drawn at larger sizes, like the brand's favicon and app icon.
 */
export function Mark({ className, ticks = false }: { className?: string; ticks?: boolean }) {
  const clip = useId()
  return (
    <svg viewBox="12 12 176 176" aria-hidden="true" className={className}>
      <clipPath id={clip}>
        <circle cx="100" cy="100" r="80" />
      </clipPath>
      <circle cx="100" cy="100" r="80" fill="#ffffff" />
      <rect x="-60" y="77" width="320" height="46" fill="#0098d4" transform="rotate(45 100 100)" clipPath={`url(#${clip})`} />
      {ticks && (
        <path
          d="M100 34v12M154 100h12M100 154v12M34 100h12"
          stroke="#0b2545"
          strokeWidth="7"
          strokeLinecap="round"
        />
      )}
      <circle cx="100" cy="100" r="80" fill="none" strokeWidth="16" className="stroke-brand-ring" />
    </svg>
  )
}

/** The name as the logo sets it: "libre" light, "quenda" bold in the brand blue. */
export function Wordmark({ className }: { className?: string }) {
  return (
    <span className={className} data-logotype>
      <span className="font-light text-brand-ink">libre</span>
      <span className="font-bold text-brand-blue">quenda</span>
    </span>
  )
}
