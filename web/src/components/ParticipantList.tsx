import { X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import type { Participant } from '@/lib/api'

type Props = {
  participants: Participant[]
  removable: boolean // false once the booking takes no more changes
  busy: boolean
  onRemove: (participantId: string) => void
}

/** The owner's view of who is coming: themselves first, then who joined, each removable (after asking). */
export function ParticipantList({ participants, removable, busy, onRemove }: Props) {
  const { t } = useTranslation('invites')
  // taking someone out keeps them out and tells them: asked once, on their row
  const [confirming, setConfirming] = useState<string | null>(null)
  return (
    <div>
      <h3 className="font-medium">{t('dialog.people')}</h3>
      <ul className="mt-1">
        <li className="border-t py-2 first:border-t-0">
          {t('dialog.you')} <span className="text-muted-foreground">· {t('dialog.owner')}</span>
        </li>
        {participants.map((p) => (
          <li key={p.id} className="grid gap-2 border-t py-2">
            <div className="flex items-center justify-between gap-3">
              <span className="min-w-0">
                <span className="block truncate">{p.name}</span>
                {p.is_guest && <span className="num text-sm text-muted-foreground">{t('dialog.guestAge', { age: p.age })}</span>}
              </span>
              {removable && confirming !== p.id && (
                <Button variant="ghost" size="icon" aria-label={t('dialog.remove', { name: p.name })} onClick={() => setConfirming(p.id)} disabled={busy}>
                  <X aria-hidden />
                </Button>
              )}
            </div>
            {removable && confirming === p.id && (
              <div role="group" aria-label={t('dialog.remove', { name: p.name })} className="grid gap-2 rounded-md bg-muted p-3 text-sm">
                <p>{p.is_guest ? t('dialog.removeGuestConfirm', { name: p.name }) : t('dialog.removeConfirm', { name: p.name })}</p>
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    variant="destructive"
                    disabled={busy}
                    onClick={() => {
                      setConfirming(null)
                      onRemove(p.id)
                    }}
                  >
                    {t('dialog.removeYes')}
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => setConfirming(null)}>
                    {t('dialog.removeKeep')}
                  </Button>
                </div>
              </div>
            )}
          </li>
        ))}
      </ul>
      {participants.length === 0 && <p className="text-sm text-muted-foreground">{t('dialog.nobody')}</p>}
    </div>
  )
}
