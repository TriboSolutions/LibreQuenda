import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import type { Participant } from '@/lib/api'

type Props = {
  participants: Participant[]
  removable: boolean // false once the booking takes no more changes
  busy: boolean
  onRemove: (participantId: string) => void
}

/** The owner's view of who is coming: themselves first, then who joined, each removable. */
export function ParticipantList({ participants, removable, busy, onRemove }: Props) {
  const { t } = useTranslation('invites')
  return (
    <div>
      <h3 className="font-medium">{t('dialog.people')}</h3>
      <ul className="mt-1">
        <li className="border-t py-2 first:border-t-0">
          {t('dialog.you')} <span className="text-muted-foreground">· {t('dialog.owner')}</span>
        </li>
        {participants.map((p) => (
          <li key={p.id} className="flex items-center justify-between gap-3 border-t py-2">
            <span className="min-w-0">
              <span className="block truncate">{p.name}</span>
              {p.is_guest && <span className="num text-sm text-muted-foreground">{t('dialog.guestAge', { age: p.age })}</span>}
            </span>
            {removable && (
              <Button variant="ghost" size="icon" aria-label={t('dialog.remove', { name: p.name })} onClick={() => onRemove(p.id)} disabled={busy}>
                <X aria-hidden />
              </Button>
            )}
          </li>
        ))}
      </ul>
      {participants.length === 0 && <p className="text-sm text-muted-foreground">{t('dialog.nobody')}</p>}
    </div>
  )
}
