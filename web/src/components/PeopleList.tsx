import { useTranslation } from 'react-i18next'
import type { Person } from '@/lib/api'

/** Who is coming, on the booking itself. */
export function PeopleList({ people }: { people: Person[] }) {
  const { t } = useTranslation('invites')
  return (
    <ul aria-label={t('people.label')} className="mt-2 flex flex-wrap gap-1.5">
      {people.map((person, i) => (
        <li
          key={i}
          className={`rounded-full border px-2.5 py-0.5 text-sm ${person.is_me ? 'border-transparent bg-accent font-medium text-accent-foreground' : ''}`}
        >
          {person.is_me ? t('people.you') : person.name || t('people.unnamed')}
          {person.is_owner && <span className="text-muted-foreground"> · {t('people.organiser')}</span>}
          {person.is_guest && <span className="text-muted-foreground"> · {t('people.noAccount')}</span>}
        </li>
      ))}
    </ul>
  )
}
