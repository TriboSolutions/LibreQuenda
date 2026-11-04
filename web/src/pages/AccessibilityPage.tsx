import { ExternalLink, Mail, Phone } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SiteFooter } from '@/components/SiteFooter'
import { TopBar } from '@/components/TopBar'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useBranding } from '@/hooks/useBranding'
import { usePageTitle } from '@/hooks/usePageTitle'
import { api } from '@/lib/api'
import { formatDateWithYear } from '@/lib/time'

// Update both with every accessibility review (the statement must say when it was last checked)
const PREPARED = '2026-10-28'
const REVIEWED = '2026-10-28'

const LAW_URL = 'https://www.boe.es/buscar/act.php?id=BOE-A-2018-12699' // Real Decreto 1112/2018
const WCAG_URL = 'https://www.w3.org/TR/WCAG21/'

const section = 'grid gap-3 rounded-lg border bg-card p-5'

/**
 * /accesibilidade: the accessibility statement, following the model of
 * Decision (EU) 2018/1523 that Real Decreto 1112/2018 adopts. Each council is
 * the body responsible for its own bookings, so the contact list names them all.
 */
export function AccessibilityPage() {
  const { t } = useTranslation(['accessibility', 'common'])
  const { branding } = useBranding()
  usePageTitle(t('title'))
  const tenants = useAsync(() => api.tenants(), 'tenants')
  const councils = tenants.data?.tenants ?? []

  const list = (key: string) => t(key, { returnObjects: true }) as string[]

  return (
    <div className="flex min-h-screen flex-col">
      <TopBar tenantSlug={null} />
      <main id="main" tabIndex={-1} className="mx-auto w-full max-w-[1152px] flex-1 px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('title')}</h1>
        {/* long reading: lines under 80 characters and roomy leading (WCAG 1.4.8) */}
        <div className="mt-6 grid max-w-[68ch] gap-6 text-base leading-relaxed">
          <p>{t('intro', { host: window.location.host, app: branding.name })}</p>

          <section aria-labelledby="a11y-status" className={section}>
            <h2 id="a11y-status" className="text-lg font-semibold">
              {t('status.title')}
            </h2>
            <p>{t('status.body')}</p>
          </section>

          <section aria-labelledby="a11y-content" className={section}>
            <h2 id="a11y-content" className="text-lg font-semibold">
              {t('content.title')}
            </h2>
            <p>{t('content.intro')}</p>
            <h3 className="font-semibold">{t('content.nonCompliance.title')}</h3>
            <p>{t('content.nonCompliance.body')}</p>
            <h3 className="font-semibold">{t('content.burden.title')}</h3>
            <p>{t('content.burden.body')}</p>
            <h3 className="font-semibold">{t('content.outOfScope.title')}</h3>
            <ul className="grid list-disc gap-1.5 pl-5">
              {list('content.outOfScope.items').map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
            <h3 className="font-semibold">{t('content.beyond.title')}</h3>
            <p>{t('content.beyond.body')}</p>
            <ul className="grid list-disc gap-1.5 pl-5">
              {list('content.beyond.items').map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </section>

          <section aria-labelledby="a11y-features" className={section}>
            <h2 id="a11y-features" className="text-lg font-semibold">
              {t('features.title')}
            </h2>
            <ul className="grid list-disc gap-1.5 pl-5">
              {list('features.items').map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </section>

          <section aria-labelledby="a11y-preparation" className={section}>
            <h2 id="a11y-preparation" className="text-lg font-semibold">
              {t('preparation.title')}
            </h2>
            <p>{t('preparation.prepared', { date: formatDateWithYear(PREPARED) })}</p>
            <p>{t('preparation.method')}</p>
            <p>{t('preparation.reviewed', { date: formatDateWithYear(REVIEWED) })}</p>
          </section>

          <section aria-labelledby="a11y-contact" className={section}>
            <h2 id="a11y-contact" className="text-lg font-semibold">
              {t('contact.title')}
            </h2>
            <p>{t('contact.body')}</p>
            <ul className="grid list-disc gap-1.5 pl-5">
              {list('contact.topics').map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
            <p>{t('contact.response')}</p>
            <h3 className="font-semibold">{t('contact.councils')}</h3>
            {tenants.error ? (
              <p className="text-destructive">{t('contact.loadError')}</p>
            ) : !tenants.data ? (
              <Skeleton className="h-16 w-full" />
            ) : councils.length === 0 ? (
              <p className="text-muted-foreground">{t('contact.noCouncils')}</p>
            ) : (
              <ul className="grid gap-3">
                {councils.map((council) => (
                  <li key={council.slug} className="grid gap-1 border-t pt-3 first:border-t-0 first:pt-0">
                    <span className="font-medium">{council.name}</span>
                    {!council.contact_email && !council.contact_phone && <span className="text-muted-foreground">{t('contact.noDetails')}</span>}
                    {council.contact_email && (
                      <a href={`mailto:${council.contact_email}`} className="inline-flex min-h-11 w-fit items-center gap-2 break-all underline underline-offset-4">
                        <Mail className="size-4 shrink-0" aria-hidden />
                        <span className="sr-only">{t('contact.email')}: </span>
                        {council.contact_email}
                      </a>
                    )}
                    {council.contact_phone && (
                      <a href={`tel:${council.contact_phone.replace(/\s+/g, '')}`} className="num inline-flex min-h-11 w-fit items-center gap-2 underline underline-offset-4">
                        <Phone className="size-4 shrink-0" aria-hidden />
                        <span className="sr-only">{t('contact.phone')}: </span>
                        {council.contact_phone}
                      </a>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section aria-labelledby="a11y-complaints" className={section}>
            <h2 id="a11y-complaints" className="text-lg font-semibold">
              {t('complaints.title')}
            </h2>
            <p>{t('complaints.body')}</p>
            <p>{t('complaints.unit')}</p>
          </section>

          <section aria-labelledby="a11y-references" className={section}>
            <h2 id="a11y-references" className="text-lg font-semibold">
              {t('references.title')}
            </h2>
            <ul className="grid gap-1">
              {[
                { href: LAW_URL, label: t('references.law') },
                { href: WCAG_URL, label: t('references.wcag') },
              ].map((link) => (
                <li key={link.href}>
                  <a href={link.href} target="_blank" rel="noopener noreferrer" className="inline-flex min-h-11 items-center gap-1.5 underline underline-offset-4">
                    {link.label}
                    <ExternalLink className="size-4 shrink-0" aria-hidden />
                    <span className="sr-only">{t('common:newTab')}</span>
                  </a>
                </li>
              ))}
            </ul>
          </section>
        </div>
      </main>
      <SiteFooter />
    </div>
  )
}
