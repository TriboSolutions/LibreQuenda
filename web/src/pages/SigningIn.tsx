import { useTranslation } from 'react-i18next'
import { usePageTitle } from '@/hooks/usePageTitle'

/** /callback: shown for a moment while the login code is exchanged (see main.tsx). */
export function SigningIn() {
  const { t } = useTranslation()
  usePageTitle(t('signingIn'))
  return <p className="mx-auto max-w-[1152px] px-4 pt-8 text-muted-foreground sm:px-6">{t('signingIn')}</p>
}
