import { useTranslation } from 'react-i18next'
import { Mail, MapPin, Phone, UserRound } from 'lucide-react'
import type { StaffContact } from '@/lib/api'

/** The people residents can ask on site, as the council lists them: name, phone to call, where to find them. */
export function StaffContacts({ contacts }: { contacts: StaffContact[] }) {
  const { t } = useTranslation('bookings')
  if (contacts.length === 0) return null
  return (
    <ul aria-label={t('checkin.staff.contacts')} className="grid gap-2">
      {contacts.map((contact, i) => (
        <li key={i} className="grid gap-1 rounded-md border px-3 py-2">
          <p className="flex items-center gap-2 font-medium">
            <UserRound className="size-4 shrink-0 text-muted-foreground" aria-hidden />
            {contact.name}
          </p>
          {contact.place && (
            <p className="flex items-start gap-2 text-sm text-muted-foreground">
              <MapPin className="mt-0.5 size-4 shrink-0" aria-hidden />
              {contact.place}
            </p>
          )}
          {contact.phone && (
            <p className="flex items-center gap-2 text-sm">
              <Phone className="size-4 shrink-0 text-muted-foreground" aria-hidden />
              <a href={`tel:${contact.phone.replace(/\s+/g, '')}`} className="num font-medium underline underline-offset-4">
                {contact.phone}
              </a>
            </p>
          )}
          {contact.email && (
            <p className="flex items-center gap-2 text-sm">
              <Mail className="size-4 shrink-0 text-muted-foreground" aria-hidden />
              <a href={`mailto:${contact.email}`} className="font-medium break-all underline underline-offset-4">
                {contact.email}
              </a>
            </p>
          )}
        </li>
      ))}
    </ul>
  )
}
