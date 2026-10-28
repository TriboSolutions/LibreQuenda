import { ExternalLink } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Mark, Wordmark } from '@/components/Mark'

const REPO_URL = 'https://github.com/TriboSolutions/LibreQuenda'
const LICENSE_URL = `${REPO_URL}/blob/main/LICENSE`
// read once: a tab left open over New Year's Eve can keep last year's
const YEAR = new Date().getFullYear()

const linkClass = 'inline-flex min-h-11 items-center gap-1.5 underline-offset-4 hover:text-foreground hover:underline'

/**
 * The foot of every page: LibreQuenda as free software (source and licence) and the
 * accessibility statement, reachable from anywhere (RD 1112/2018, art. 10).
 */
export function SiteFooter() {
  const { t } = useTranslation(['common', 'accessibility'])
  return (
    <footer className="border-t bg-card">
      <div className="mx-auto grid max-w-[1152px] gap-8 px-4 py-10 text-sm text-muted-foreground sm:px-6 md:grid-cols-[1.5fr_1fr_1fr]">
        <div>
          <p className="flex items-center gap-2">
            <Mark className="size-7" />
            <Wordmark className="text-[17px] tracking-tight" />
          </p>
          <p className="mt-3 max-w-[44ch]">{t('common:footer.about')}</p>
          <p className="mt-4 inline-flex items-center rounded-full bg-pill-accent-bg px-3 py-1 text-xs font-semibold text-pill-accent-ink">
            {t('common:footer.badge')}
          </p>
        </div>

        <nav aria-labelledby="footer-project">
          <h2 id="footer-project" className="font-semibold text-foreground">
            {t('common:footer.project')}
          </h2>
          <ul className="mt-1">
            <li>
              <a href={REPO_URL} target="_blank" rel="noreferrer" className={linkClass}>
                {t('common:footer.source')}
                <ExternalLink className="size-3.5" aria-hidden="true" />
                <span className="sr-only">{t('common:newTab')}</span>
              </a>
            </li>
            <li>
              <a href={LICENSE_URL} target="_blank" rel="noreferrer license" className={linkClass}>
                {t('common:footer.license')}
                <ExternalLink className="size-3.5" aria-hidden="true" />
                <span className="sr-only">{t('common:newTab')}</span>
              </a>
            </li>
          </ul>
        </nav>

        <nav aria-labelledby="footer-legal">
          <h2 id="footer-legal" className="font-semibold text-foreground">
            {t('common:footer.legal')}
          </h2>
          <ul className="mt-1">
            <li>
              <Link to="/accesibilidade" className={linkClass}>
                {t('accessibility:footerLink')}
              </Link>
            </li>
          </ul>
        </nav>
      </div>
      <div className="border-t">
        <p className="mx-auto max-w-[1152px] px-4 py-4 text-xs text-muted-foreground sm:px-6">{t('common:footer.copyright', { year: YEAR })}</p>
      </div>
    </footer>
  )
}
