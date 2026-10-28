import { ArrowRight, Building2, CalendarCheck, Search, Users, Zap, type LucideIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, Navigate } from 'react-router'
import { Mark } from '@/components/Mark'
import { TopBar } from '@/components/TopBar'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { api, type TenantCard } from '@/lib/api'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'
import { cn } from '@/lib/utils'

// A search box only pays off once the list is longer than a screen's glance
const SEARCH_FROM = 8

function normalise(text: string) {
  return text.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase()
}

function useNumber() {
  const { i18n } = useTranslation()
  const format = new Intl.NumberFormat(i18n.resolvedLanguage ?? 'gl')
  return (value: number) => format.format(value)
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

      <main id="main" tabIndex={-1} className="pb-16">
        <Hero councils={tenants.data ? list : null} />

        <section aria-labelledby="councils-title" className="mx-auto max-w-[1152px] px-4 pt-10 sm:px-6">
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 id="councils-title" className="text-2xl font-bold tracking-[-0.01em]">
                {t('councils:title')}
              </h2>
              {tenants.data ? (
                <p className="mt-1 text-muted-foreground">{t('councils:subtitle', { count: list.length })}</p>
              ) : (
                <Skeleton className="mt-2 h-5 w-80 max-w-full" />
              )}
            </div>
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
            <div className="mt-6 flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
              <p>{t('councils:loadError')}</p>
              <Button variant="ghost" size="sm" onClick={tenants.reload}>
                {t('common:retry')}
              </Button>
            </div>
          ) : tenants.loading && !tenants.data ? (
            <ul className="mt-6 grid gap-5 md:grid-cols-2 lg:grid-cols-3" aria-busy="true" aria-label={t('councils:loading')}>
              {Array.from({ length: 3 }, (_, i) => (
                <li key={i} className="rounded-xl border bg-card p-6">
                  <div className="flex items-center gap-4">
                    <Skeleton className="size-16 rounded-lg" />
                    <Skeleton className="h-6 w-40" />
                  </div>
                  <div className="mt-6 grid grid-cols-2 gap-3">
                    {Array.from({ length: 4 }, (_, j) => (
                      <Skeleton key={j} className="h-20 rounded-lg" />
                    ))}
                  </div>
                </li>
              ))}
            </ul>
          ) : list.length === 0 ? (
            <p className="mt-6 rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('councils:empty')}</p>
          ) : shown.length === 0 ? (
            <p className="mt-6 rounded-md border border-dashed px-4 py-3 text-muted-foreground">{t('councils:noResults', { query })}</p>
          ) : (
            <ul className="mt-6 grid gap-5 md:grid-cols-2 lg:grid-cols-3">
              {shown.map((council) => (
                <li key={council.slug} className="flex">
                  <CouncilCard council={council} />
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

/** The slogan and what the whole platform adds up to across its councils. */
function Hero({ councils }: { councils: TenantCard[] | null }) {
  const { t } = useTranslation('councils')
  const number = useNumber()
  const sum = (pick: (c: TenantCard) => number) => (councils ?? []).reduce((total, c) => total + pick(c), 0)
  const totals = [
    { label: t('hero.councils', { count: councils?.length ?? 0 }), value: councils?.length ?? 0 },
    { label: t('hero.facilities', { count: sum((c) => c.stats.facilities) }), value: sum((c) => c.stats.facilities) },
    { label: t('hero.bookings', { count: sum((c) => c.stats.bookings) }), value: sum((c) => c.stats.bookings) },
    { label: t('hero.residents', { count: sum((c) => c.stats.residents) }), value: sum((c) => c.stats.residents) },
  ]

  return (
    <div className="relative overflow-hidden border-b bg-card">
      {/* the brand mark, large and faint, as a backdrop */}
      <Mark className="pointer-events-none absolute -top-24 -right-24 size-[420px] opacity-[0.07] max-sm:hidden" />
      <div className="relative mx-auto max-w-[1152px] px-4 py-12 sm:px-6 sm:py-16">
        <p className="text-sm font-semibold tracking-wide text-primary uppercase">{t('hero.eyebrow')}</p>
        <h1 className="mt-3 max-w-[18ch] text-[34px] leading-[1.1] font-bold tracking-[-0.025em] sm:text-[48px]">{t('hero.slogan')}</h1>
        <p className="mt-4 max-w-[60ch] text-lg text-muted-foreground">{t('hero.lead')}</p>

        <dl className="mt-10 grid grid-cols-2 gap-x-6 gap-y-6 sm:grid-cols-4">
          {totals.map((total) => (
            <div key={total.label} className="flex flex-col border-l-2 border-primary pl-4">
              <dt className="text-sm text-muted-foreground">{total.label}</dt>
              <dd className="order-first text-3xl font-bold tabular-nums sm:text-4xl">
                {councils ? number(total.value) : <Skeleton className="h-9 w-20" />}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  )
}

/** One council: its coat of arms, sports and figures. The whole card opens the council. */
function CouncilCard({ council }: { council: TenantCard }) {
  const { t } = useTranslation('councils')
  const number = useNumber()
  const { stats } = council
  const figures: { icon: LucideIcon; value: number; label: string; highlight?: boolean }[] = [
    { icon: Building2, value: stats.facilities, label: t('card.facilities', { count: stats.facilities }) },
    { icon: Zap, value: stats.free_today, label: t('card.freeToday', { count: stats.free_today }), highlight: stats.free_today > 0 },
    { icon: CalendarCheck, value: stats.bookings, label: t('card.bookings', { count: stats.bookings }) },
    { icon: Users, value: stats.residents, label: t('card.residents', { count: stats.residents }) },
  ]

  return (
    <article className="group relative flex w-full flex-col rounded-xl border bg-card p-6 shadow-xs transition-[border-color,box-shadow,translate] focus-within:ring-2 focus-within:ring-ring hover:-translate-y-0.5 hover:border-primary/60 hover:shadow-md motion-reduce:transition-none motion-reduce:hover:translate-y-0">
      <div className="flex items-center gap-4">
        <span className="flex size-16 shrink-0 items-center justify-center rounded-lg bg-muted p-2">
          {council.logo_url ? <img src={council.logo_url} alt="" className="size-full object-contain" /> : <Mark className="size-10" />}
        </span>
        <div className="min-w-0">
          <h3 className="text-xl leading-7 font-semibold">
            {/* stretched over the card: the council's name is the link's whole accessible name */}
            <Link to={`/${council.slug}`} className="outline-none after:absolute after:inset-0 after:rounded-xl">
              {council.name}
            </Link>
          </h3>
          {stats.sports.length > 0 && <p className="mt-0.5 truncate text-sm text-muted-foreground">{stats.sports.join(' · ')}</p>}
        </div>
      </div>

      <dl className="mt-6 grid grid-cols-2 gap-3">
        {figures.map(({ icon: Icon, value, label, highlight }) => (
          <div key={label} className={cn('flex flex-col rounded-lg p-3', highlight ? 'bg-pill-positive-bg text-pill-positive-ink' : 'bg-muted')}>
            <dt className="flex items-center gap-1.5 text-sm leading-5">
              <Icon className={cn('size-4 shrink-0', !highlight && 'text-primary')} aria-hidden="true" />
              <span className={cn(!highlight && 'text-muted-foreground')}>{label}</span>
            </dt>
            <dd className="order-first text-2xl leading-8 font-bold tabular-nums">{number(value)}</dd>
          </div>
        ))}
      </dl>

      <p className="mt-6 flex items-center gap-1.5 font-semibold text-primary" aria-hidden="true">
        {t('card.cta')}
        <ArrowRight className="size-4 transition-transform group-hover:translate-x-1 motion-reduce:transition-none" />
      </p>
    </article>
  )
}
