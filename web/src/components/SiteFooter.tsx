import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'

/** The foot of every page: the accessibility statement, reachable from anywhere (RD 1112/2018, art. 10). */
export function SiteFooter() {
  const { t } = useTranslation('accessibility')
  return (
    <footer className="border-t">
      <div className="mx-auto flex max-w-[1152px] flex-wrap items-center gap-x-6 px-4 py-2 text-sm text-muted-foreground sm:px-6">
        <Link to="/accesibilidade" className="inline-flex min-h-11 items-center underline underline-offset-4 hover:text-foreground">
          {t('footerLink')}
        </Link>
      </div>
    </footer>
  )
}
