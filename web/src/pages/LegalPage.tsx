import { Mail, Phone } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SiteFooter } from '@/components/SiteFooter'
import { TopBar } from '@/components/TopBar'
import { useBranding } from '@/hooks/useBranding'
import { usePageTitle } from '@/hooks/usePageTitle'

type Section = { title: string; body: string[] }

/**
 * /condicions and /privacidade: the terms of use and the privacy policy, the
 * same texts the Android app shows (legal.json). The installation's name goes in
 * where the texts say {{app}}; questions go to its support contact.
 */
export function LegalPage({ which }: { which: 'terms' | 'privacy' }) {
  const { t } = useTranslation('legal')
  const { branding } = useBranding()
  usePageTitle(t(`${which}.title`))
  const app = branding.name
  const sections = t(`${which}.sections`, { returnObjects: true, app }) as Section[]

  return (
    <div className="flex min-h-screen flex-col">
      <TopBar />
      <main id="main" tabIndex={-1} className="mx-auto w-full max-w-[1152px] flex-1 px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t(`${which}.title`)}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{t('updated')}</p>
        {/* long reading: lines under 80 characters and roomy leading (WCAG 1.4.8) */}
        <div className="mt-6 grid max-w-[68ch] gap-6 text-base leading-relaxed">
          <p>{t(`${which}.intro`, { app })}</p>
          {Array.isArray(sections) &&
            sections.map((section, i) => (
              <section key={section.title} aria-labelledby={`legal-${i}`} className="grid gap-2">
                <h2 id={`legal-${i}`} className="text-lg font-semibold">
                  {section.title}
                </h2>
                {section.body.map((paragraph) => (
                  <p key={paragraph}>{paragraph}</p>
                ))}
              </section>
            ))}
          <SupportContact />
        </div>
      </main>
      <SiteFooter />
    </div>
  )
}

/** The installation's support contact, when it has one (SUPPORT_EMAIL, SUPPORT_PHONE). */
export function SupportContact({ compact = false }: { compact?: boolean }) {
  const { t } = useTranslation('legal')
  const { branding } = useBranding()
  const { support_email: email, support_phone: phone } = branding
  if (!email && !phone) return null
  const link = 'inline-flex min-h-11 w-fit items-center gap-2 underline-offset-4 hover:underline'
  const links = (
    <>
      {email && (
        <a href={`mailto:${email}`} className={`${link} break-all`}>
          <Mail className="size-4 shrink-0" aria-hidden />
          {email}
        </a>
      )}
      {phone && (
        <a href={`tel:${phone.replace(/\s+/g, '')}`} className={`${link} num`}>
          <Phone className="size-4 shrink-0" aria-hidden />
          {phone}
        </a>
      )}
    </>
  )
  if (compact) return links
  return (
    <section aria-labelledby="legal-support" className="grid gap-2 rounded-lg border bg-card p-5">
      <h2 id="legal-support" className="text-lg font-semibold">
        {t('support.title')}
      </h2>
      <p>{t('support.help')}</p>
      <div className="grid">{links}</div>
    </section>
  )
}
