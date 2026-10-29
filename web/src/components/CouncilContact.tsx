import { Mail, Phone, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Mark } from '@/components/Mark'
import type { Facility, Tenant } from '@/lib/api'

/** How to reach the council's sports service. Each facility's booking rules live on its own page. */
export function CouncilContact({ tenant }: { tenant: Tenant }) {
  const { t } = useTranslation('home')
  return (
    <ContactCard
      id="contact-council"
      title={t('contact.title')}
      name={tenant.name}
      logo={tenant.logo_url ? <img src={tenant.logo_url} alt="" className="size-10 shrink-0 object-contain" /> : null}
      lead={t('contact.lead')}
      none={t('contact.none')}
      phone={tenant.contact_phone}
      email={tenant.contact_email}
    />
  )
}

/** The facility's own phone and email, when it has them; shown above the council's on its page. */
export function FacilityContact({ facility }: { facility: Pick<Facility, 'name' | 'contact_phone' | 'contact_email'> }) {
  const { t } = useTranslation('home')
  return (
    <ContactCard
      id="contact-facility"
      title={t('contact.facilityTitle')}
      name={facility.name}
      logo={
        <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted">
          <Mark className="size-6" />
        </span>
      }
      lead={t('contact.facilityLead')}
      none=""
      phone={facility.contact_phone}
      email={facility.contact_email}
    />
  )
}

type CardProps = { id: string; title: string; name: string; logo: React.ReactNode; lead: string; none: string; phone: string; email: string }

function ContactCard({ id, title, name, logo, lead, none, phone, email }: CardProps) {
  const { t } = useTranslation('home')
  const ways: { icon: LucideIcon; label: string; value: string; href: string }[] = []
  if (phone) ways.push({ icon: Phone, label: t('contact.phone'), value: phone, href: `tel:${phone.replace(/\s+/g, '')}` })
  if (email) ways.push({ icon: Mail, label: t('contact.email'), value: email, href: `mailto:${email}` })

  return (
    <section aria-labelledby={id} className="rounded-lg border bg-card p-5">
      <div className="flex items-center gap-3">
        {logo}
        <div className="min-w-0">
          <h2 id={id} className="text-lg font-semibold">
            {title}
          </h2>
          <p className="truncate text-sm text-muted-foreground">{name}</p>
        </div>
      </div>

      {ways.length === 0 ? (
        <p className="mt-4 text-muted-foreground">{none}</p>
      ) : (
        <>
          <p className="mt-3 text-muted-foreground">{lead}</p>
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
