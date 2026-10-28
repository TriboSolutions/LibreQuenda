import { Mail, Phone, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Tenant } from '@/lib/api'

/** How to reach the council's sports service. Each facility's booking rules live on its own page. */
export function CouncilContact({ tenant }: { tenant: Tenant }) {
  const { t } = useTranslation('home')
  const ways: { icon: LucideIcon; label: string; value: string; href: string }[] = []
  if (tenant.contact_phone) ways.push({ icon: Phone, label: t('contact.phone'), value: tenant.contact_phone, href: `tel:${tenant.contact_phone.replace(/\s+/g, '')}` })
  if (tenant.contact_email) ways.push({ icon: Mail, label: t('contact.email'), value: tenant.contact_email, href: `mailto:${tenant.contact_email}` })

  return (
    <section aria-labelledby="contact-title" className="rounded-lg border bg-card p-5">
      <div className="flex items-center gap-3">
        {tenant.logo_url && <img src={tenant.logo_url} alt="" className="size-10 shrink-0 object-contain" />}
        <div className="min-w-0">
          <h2 id="contact-title" className="text-lg font-semibold">
            {t('contact.title')}
          </h2>
          <p className="truncate text-sm text-muted-foreground">{tenant.name}</p>
        </div>
      </div>

      {ways.length === 0 ? (
        <p className="mt-4 text-muted-foreground">{t('contact.none')}</p>
      ) : (
        <>
          <p className="mt-3 text-muted-foreground">{t('contact.lead')}</p>
          <ul className="mt-3 grid gap-2">
            {ways.map(({ icon: Icon, label, value, href }) => (
              <li key={href}>
                <a href={href} className="flex min-h-14 items-center gap-3 rounded-md border px-3 py-2 transition-colors hover:bg-accent hover:text-accent-foreground">
                  <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-pill-accent-bg text-pill-accent-ink">
                    <Icon className="size-4" aria-hidden="true" />
                  </span>
                  <span className="grid min-w-0">
                    <span className="text-sm text-muted-foreground">{label}</span>
                    <span className="num font-medium break-all">{value}</span>
                  </span>
                </a>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  )
}
