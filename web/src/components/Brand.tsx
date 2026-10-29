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
