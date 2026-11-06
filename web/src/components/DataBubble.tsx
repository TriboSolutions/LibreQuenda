import { Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link, matchPath, useLocation } from 'react-router'
import { useMe } from '@/hooks/useMe'

// pages without the bubble: its own page, sign-in, and the ones shown on a screen or
// printed (the widget inside a council's website, the entrance, the reader, the print)
const HIDDEN = ['/datos', '/callback', '/lector', '/:slug/widget', '/:slug/widget/:facility', '/:slug/admin/facilities/:id/imprimir', '/:slug/:facility/entrada']

/**
 * «Talk to the data», floating in the bottom right corner on every page for the
 * people who can use the assistant (superadministrators and councils' administrators)
 * while a superadministrator has it switched on. In a council's administration it
 * opens on that council's data.
 */
export function DataBubble() {
  const { t } = useTranslation('admin')
  const { me } = useMe()
  const { pathname } = useLocation()
  if (!me?.analytics_enabled || HIDDEN.some((path) => matchPath(path, pathname))) return null

  const slug = matchPath('/:slug/admin/*', pathname)?.params.slug
  const runs = slug && me.roles.some((r) => r.tenant_slug === slug && r.role === 'admin')
  return (
    <Link
      to={runs ? `/datos?concello=${encodeURIComponent(slug)}` : '/datos'}
      aria-label={t('analytics.bubble')}
      className="fixed right-4 bottom-4 z-40 inline-flex items-center gap-2 rounded-full bg-primary p-3.5 text-primary-foreground shadow-lg transition hover:shadow-xl focus-visible:ring-4 focus-visible:ring-ring/50 focus-visible:outline-none sm:right-6 sm:bottom-6 sm:px-5"
    >
      <Sparkles className="size-5" aria-hidden />
      <span className="hidden font-medium sm:inline">{t('analytics.bubble')}</span>
    </Link>
  )
}
