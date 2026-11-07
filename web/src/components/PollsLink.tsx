import { ChevronRight, Vote } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { SettingsRow } from '@/components/SettingsRow'

/** On /perfil: every poll of their councils, to vote again or follow the results. */
export function PollsLink() {
  const { t } = useTranslation('home')
  return (
    <SettingsRow id="sec-polls" title={t('polls.profileTitle')} description={t('polls.profileHelp')}>
      <Link to="/enquisas" className="flex items-center gap-3 rounded-lg border bg-card p-5 font-medium hover:border-primary/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
        <Vote className="size-5 shrink-0 text-muted-foreground" aria-hidden />
        <span className="flex-1">{t('polls.profileLink')}</span>
        <ChevronRight className="size-5 shrink-0 text-muted-foreground" aria-hidden />
      </Link>
    </SettingsRow>
  )
}
