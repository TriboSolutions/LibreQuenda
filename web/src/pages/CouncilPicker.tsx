import { ChevronRight, Search } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Navigate } from 'react-router'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api } from '@/lib/api'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

// A search box only pays off once the list is longer than a screen's glance
const SEARCH_FROM = 8

function normalise(text: string) {
  return text.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase()
}

/** "/": choose a council. With only one council there is nothing to choose, so go straight to it. */
export function CouncilPicker() {
  const { t } = useTranslation(['councils', 'common'])
  const tenants = useAsync(() => api.tenants(), 'tenants')
  const [query, setQuery] = useState('')
  usePageTitle(t('councils:title'))

  const list = tenants.data?.tenants ?? []
  if (list.length === 1) return <Navigate to={`/${list[0].slug}`} replace />

  const shown = query ? list.filter((c) => normalise(c.name).includes(normalise(query))) : list

  return (
    <div className="min-h-screen">
      <TopBar tenants={list} tenantSlug={null} />

      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('councils:title')}</h1>
        {tenants.data ? (
          <p className="mt-1 text-base text-muted-foreground">{t('councils:subtitle', { count: list.length })}</p>
        ) : (
          <Skeleton className="mt-2 h-5 w-80 max-w-full" />
        )}

        <section aria-labelledby="councils-title" className="mt-6 max-w-[760px] rounded-lg border bg-card p-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 id="councils-title" className="text-lg font-semibold">
              {t('councils:listTitle')}
            </h2>
            {list.length >= SEARCH_FROM && (
              <div className="relative w-full sm:w-72">
                <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
                <Input
                  type="search"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder={t('councils:search')}
                  aria-label={t('councils:search')}
                  className="pl-9"
                />
              </div>
            )}
          </div>

          {tenants.error ? (
            <div className="mt-4 flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
              <p>{t('councils:loadError')}</p>
              <Button variant="ghost" size="sm" onClick={tenants.reload}>
                {t('common:retry')}
              </Button>
            </div>
          ) : tenants.loading && !tenants.data ? (
            <ul className="mt-2" aria-busy="true" aria-label={t('councils:loading')}>
              {Array.from({ length: 4 }, (_, i) => (
                <li key={i} className="border-t py-3.5 first:border-t-0">
                  <Skeleton className="h-4 w-48" />
                </li>
              ))}
            </ul>
          ) : list.length === 0 ? (
            <p className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('councils:empty')}</p>
          ) : shown.length === 0 ? (
            <p className="mt-4 rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('councils:noResults', { query })}</p>
          ) : (
            <ul className="mt-2">
              {shown.map((council) => (
                <li key={council.slug} className="border-t first:border-t-0">
                  <Link
                    to={`/${council.slug}`}
                    className="-mx-2 flex items-center justify-between gap-3 rounded-md px-2 py-3 font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
                  >
                    <span className="flex min-w-0 items-center gap-3">
                      {council.logo_url ? (
                        <img src={council.logo_url} alt="" className="size-8 shrink-0 object-contain" />
                      ) : (
                        <span className="size-8 shrink-0 rounded-md bg-muted" aria-hidden="true" />
                      )}
                      <span className="truncate">{council.name}</span>
                    </span>
                    <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>
      </main>
      <SiteFooter />
    </div>
  )
}
