import { useTranslation } from 'react-i18next'

/** /callback: shown for a moment while the login code is exchanged (see main.tsx). */
export function SigningIn() {
  const { t } = useTranslation()
  return <p className="mx-auto max-w-[1152px] px-4 pt-8 text-muted-foreground sm:px-6">{t('signingIn')}</p>
}
