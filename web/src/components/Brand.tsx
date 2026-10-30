import logo from '@/assets/brand/logo.svg'
import logoDark from '@/assets/brand/logo-dark.svg'
import { Mark, Wordmark } from '@/components/Mark'
import { useBranding } from '@/hooks/useBranding'
import { cn } from '@/lib/utils'

/** The installation's logo, or LibreQuenda's mark when it has none. Decorative: the name goes next to it or in a label. */
export function BrandLogo({ className }: { className?: string }) {
  const { branding } = useBranding()
  if (!branding.logo_url) return <Mark className={className} />
  return <img src={branding.logo_url} alt="" className={cn('rounded-md object-contain', className)} />
}

/** The installation's name, or LibreQuenda's wordmark when it has none. */
export function BrandName({ className }: { className?: string }) {
  const { branding } = useBranding()
  if (!branding.custom_name) return <Wordmark className={className} />
  return <span className={cn('font-semibold text-brand-ink', className)}>{branding.custom_name}</span>
}

/**
 * The top bar's logo: LibreQuenda's own lockup (mark, name and "Reserva de
 * instalacións deportivas", docs/brand), the same image Zitadel's sign-in pages
 * show, light or dark with the theme; a narrow phone has room for the mark only.
 * The installation's own logo and name, when it has them. Decorative: the link
 * around it carries the name.
 */
export function BrandLockup() {
  const { branding } = useBranding()
  if (branding.logo_url || branding.custom_name) {
    return (
      <>
        <BrandLogo className="size-7" />
        <BrandName className="hidden max-w-60 truncate text-[17px] tracking-tight sm:inline" />
      </>
    )
  }
  return (
    <>
      <Mark ticks className="size-8 sm:hidden" />
      <img src={logo} alt="" className="hidden h-9 w-auto sm:block dark:sm:hidden" />
      <img src={logoDark} alt="" className="hidden h-9 w-auto dark:sm:block" />
    </>
  )
}
