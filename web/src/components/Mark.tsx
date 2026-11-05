import { useId } from 'react'

const ARM = 'M0 0 C 0 -16, 13 -27, 25 -21 C 35 -15, 31 -2, 21 -3 C 15 -4, 14 -11, 19 -13'

/**
 * LibreQuenda mark: a triskele inside a clock face, its ring split into turns
 * (quendas), on the blue tile with the Galician flag's band. Same geometry as
 * docs/brand/svg/mark.svg; it carries its own background, so it reads on light and dark.
 */
export function Mark({ className }: { className?: string }) {
  const id = useId()
  const tile = `${id}t`
  const face = `${id}f`
  return (
    <svg viewBox="0 0 120 120" aria-hidden="true" className={className}>
      <defs>
        <clipPath id={tile}>
          <rect width="120" height="120" rx="30" />
        </clipPath>
        <clipPath id={face}>
          <circle cx="60" cy="60" r="35" />
        </clipPath>
      </defs>
      <rect width="120" height="120" rx="30" fill="#0096d1" />
      <polygon points="-20,6 6,-20 140,114 114,140" fill="#3fb0dd" clipPath={`url(#${tile})`} />
      <g fill="none" strokeWidth="7.5" strokeLinecap="round">
        <path d="M67.51 12.59 A48 48 0 0 1 104.81 77.20" stroke="#ffffff" />
        <path d="M97.30 90.21 A48 48 0 0 1 22.70 90.21" stroke="#0b2544" />
        <path d="M15.19 77.20 A48 48 0 0 1 52.49 12.59" stroke="#0b2544" />
      </g>
      <circle cx="60" cy="60" r="35" fill="#ffffff" />
      <g clipPath={`url(#${face})`}>
        <g
          transform="translate(60 61) scale(1.1) rotate(-8)"
          fill="none"
          stroke="#0b2544"
          strokeWidth="5.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d={ARM} />
          <path d={ARM} transform="rotate(120)" />
          <path d={ARM} transform="rotate(240)" />
        </g>
      </g>
      <circle cx="60" cy="60" r="35" fill="none" stroke="#0b2544" strokeWidth="4.5" />
    </svg>
  )
}

/** The name as the logo sets it: "libre" regular, "Quenda" bold in the brand blue. */
export function Wordmark({ className }: { className?: string }) {
  return (
    <span className={className} data-logotype>
      <span className="font-normal text-brand-ink">libre</span>
      <span className="font-bold text-brand-blue">Quenda</span>
    </span>
  )
}
