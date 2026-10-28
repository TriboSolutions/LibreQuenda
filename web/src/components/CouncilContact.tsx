import type { ReactNode } from 'react'
import { Mail, MapPin, Phone, UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Tenant } from '@/lib/api'

// The link stretches over its whole row (label + value), so the target stays 44 px tall (WCAG 2.5.5)
const link = 'break-all underline underline-offset-4 after:absolute after:inset-0'

/** Who to reach at the council about its facilities. Hidden when it has no contact details. */
export function CouncilContact({ council }: { council: Tenant }) {
  const { t } = useTranslation('home')
  const { contact_person: person, contact_address: address, contact_email: email, contact_phone: phone } = council
  if (!person && !address && !email && !phone) return null

  return (
    <section aria-labelledby="contact-title" className="rounded-lg border bg-card p-5">
      <h2 id="contact-title" className="text-lg font-semibold">
        {t('contact.title')}
      </h2>
      <dl className="mt-3 grid gap-3">
        {person && (
          <Row icon={<UserRound />} label={t('contact.person')}>
            {person}
          </Row>
        )}
        {address && (
          <Row icon={<MapPin />} label={t('contact.address')}>
            {address}
          </Row>
        )}
        {email && (
          <Row icon={<Mail />} label={t('contact.email')}>
            <a href={`mailto:${email}`} className={link}>
              {email}
            </a>
          </Row>
        )}
        {phone && (
          <Row icon={<Phone />} label={t('contact.phone')}>
            <a href={`tel:${phone.replace(/\s+/g, '')}`} className={`num ${link}`}>
              {phone}
            </a>
          </Row>
        )}
      </dl>
    </section>
  )
}

function Row({ icon, label, children }: { icon: ReactNode; label: string; children: ReactNode }) {
  return (
    <div className="relative grid min-h-11 grid-cols-[auto_1fr] gap-x-3">
      <span className="row-span-2 mt-0.5 text-muted-foreground [&_svg]:size-4" aria-hidden>
        {icon}
      </span>
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  )
}
