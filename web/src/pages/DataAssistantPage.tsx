import { ArrowLeft, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router'
import { AnalyticsAssistant } from '@/components/admin/AnalyticsAssistant'
import { SiteFooter } from '@/components/SiteFooter'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { usePageTitle } from '@/hooks/usePageTitle'
import { api } from '@/lib/api'

const ALL = '*' // every council: superadministrators only

/**
 * /datos: «talk to the data», for superadministrators (every council, or one) and
 * councils' administrators (the councils they run). ?concello=<slug> picks the
 * council; the bubble links here from any page while the assistant is on.
 */
export function DataAssistantPage() {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const { me, loading } = useMe()
  const { login } = useLogin()
  const navigate = useNavigate()
  const location = useLocation()
  const [params, setParams] = useSearchParams()
  usePageTitle(t('admin:analytics.title'))
  const token = auth.user?.access_token ?? ''

  // whose data: every council (superadministrators) and each council they run
  const councils = me?.roles.filter((r) => r.role === 'admin') ?? []
  const wanted = params.get('concello')
  const scope = wanted && councils.some((c) => c.tenant_slug === wanted) ? wanted : me?.is_superadmin ? ALL : (councils[0]?.tenant_slug ?? ALL)
  const scopeName = scope === ALL ? t('admin:analytics.everyCouncil') : (councils.find((c) => c.tenant_slug === scope)?.tenant_name ?? scope)
  const options = [...(me?.is_superadmin ? [{ value: ALL, label: t('admin:analytics.everyCouncil') }] : []), ...councils.map((c) => ({ value: c.tenant_slug, label: c.tenant_name }))]

  let body
  if (auth.isLoading || (auth.isAuthenticated && (loading || !me))) {
    body = <Skeleton className="mt-6 h-40 w-full" />
  } else if (!auth.isAuthenticated) {
    body = (
      <div className="mt-6 grid max-w-[760px] gap-4 rounded-lg border bg-card p-5">
        <p>{t('admin:guard.signIn')}</p>
        <div>
          <Button onClick={login}>{t('common:signIn')}</Button>
        </div>
      </div>
    )
  } else if (!me?.analytics_enabled) {
    body = <p className="mt-6 max-w-[760px] text-muted-foreground">{t('admin:analytics.off')}</p>
  } else {
    body = (
      <div className="mt-4 grid gap-6">
        <p className="text-muted-foreground">{t('admin:analytics.help', { scope: scopeName })}</p>
        {options.length > 1 && (
          <div className="grid gap-1.5">
            <Label htmlFor="assistant-scope">{t('admin:analytics.scope')}</Label>
            <Select
              value={scope}
              onValueChange={(value) =>
                setParams(value === ALL ? {} : { concello: value }, {
                  replace: true,
                })
              }
            >
              <SelectTrigger id="assistant-scope" className="w-full sm:w-80">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {options.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        {/* a new council, a new conversation */}
        <AnalyticsAssistant key={scope} ask={(question) => (scope === ALL ? api.admin.askAnalytics(token, question) : api.council.askAnalytics(token, scope, question))} />
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[860px] px-4 pt-8 pb-8 sm:px-6">
        <p>
          <Link
            to="/"
            onClick={(event) => {
              if (location.key === 'default') return
              event.preventDefault()
              navigate(-1)
            }}
            className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="size-4" aria-hidden />
            {t('admin:analytics.back')}
          </Link>
        </p>
        <h1 className="mt-3 flex items-center gap-2 text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">
          <Sparkles className="size-7 text-primary" aria-hidden />
          {t('admin:analytics.title')}
        </h1>
        {body}
      </main>
      <SiteFooter />
    </div>
  )
}
