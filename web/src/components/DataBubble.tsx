import { Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link, matchPath, useLocation } from 'react-router'
import { useMe } from '@/hooks/useMe'

/**
 * «Talk to the data», floating in the bottom right corner of a council's indicators
 * (/:slug/admin?vista=indicadores) while a superadministrator has the assistant
 * switched on, for that council's administrators and superadministrators; it opens
 * the conversation on that council's data.
 */
export function DataBubble() {
  const { t } = useTranslation('admin')
  const { me } = useMe()
  const { pathname, search } = useLocation()
  const slug = matchPath('/:slug/admin', pathname)?.params.slug
  if (!me?.analytics_enabled || !slug || new URLSearchParams(search).get('vista') !== 'indicadores') return null
  if (!me.is_superadmin && !me.roles.some((r) => r.tenant_slug === slug && r.role === 'admin')) return null
  return (
    <Link
      to={`/datos?concello=${encodeURIComponent(slug)}`}
      aria-label={t('analytics.bubble')}
      className="fixed right-4 bottom-4 z-40 inline-flex items-center gap-2 rounded-full bg-primary p-3.5 text-primary-foreground shadow-lg transition hover:shadow-xl focus-visible:ring-4 focus-visible:ring-ring/50 focus-visible:outline-none sm:right-6 sm:bottom-6 sm:px-5"
    >
      <Sparkles className="size-5" aria-hidden />
      <span className="hidden font-medium sm:inline">{t('analytics.bubble')}</span>
    </Link>
  )
}
