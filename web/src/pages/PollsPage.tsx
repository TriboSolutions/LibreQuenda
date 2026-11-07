import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { useSearchParams } from 'react-router'
import { PollList } from '@/components/CouncilPolls'
import { Pager } from '@/components/Pager'
import { SiteFooter } from '@/components/SiteFooter'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { usePageTitle } from '@/hooks/usePageTitle'
import { api } from '@/lib/api'

/**
 * /enquisas: the polls of the councils the person is part of. They arrive as
 * notices (the bell, the email): ?enquisa=<id> opens on the one a notice named,
 * to vote in it or, once it closed, to see its results.
 */
export function PollsPage() {
  const { t } = useTranslation(['home', 'common'])
  const auth = useAuth()
  const { login } = useLogin()
  const { me } = useMe()
  const [params] = useSearchParams()
  const token = auth.user?.access_token
  const [page, setPage] = useState(1)
  const polls = useAsync(token && me ? () => api.myPolls(token, page) : null, `polls:${me?.id}:${page}`)
  usePageTitle(t('home:polls.title'))

  let body
  if (!auth.isLoading && !auth.isAuthenticated) {
    body = (
      <div className="mt-6 grid max-w-[760px] gap-4">
        <p>{t('home:polls.signIn')}</p>
        <Button className="w-fit" onClick={login}>
          {t('common:signIn')}
        </Button>
      </div>
    )
  } else if (polls.error) {
    body = (
      <Alert variant="destructive" className="mt-6">
        <AlertDescription>{polls.error.message}</AlertDescription>
      </Alert>
    )
  } else if (!polls.data) {
    body = <Skeleton className="mt-6 h-48 w-full" />
  } else if (polls.data.polls.length === 0) {
    body = <p className="mt-6 rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('home:polls.empty')}</p>
  } else {
    body = (
      <div className="mt-6">
        <PollList polls={polls.data.polls} onVoted={polls.reload} focus={params.get('enquisa') ?? undefined} />
        <div className="mt-4">
          <Pager pagination={polls.data.pagination} onChange={setPage} label={t('home:polls.pages')} />
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('home:polls.title')}</h1>
        <p className="mt-1 max-w-[70ch] text-base text-muted-foreground">{t('home:polls.help')}</p>
        {body}
      </main>
      <SiteFooter />
    </div>
  )
}
