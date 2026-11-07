import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import type { Pagination } from '@/lib/api'

type Props = {
  /** what the API said of the page it sent */
  pagination?: Pagination
  onChange: (page: number) => void
  /** which list it pages, for screen readers: «Páxinas dos comunicados» */
  label: string
}

/**
 * Previous and next under a list the API sends a page at a time (?page=), with
 * where one is. Nothing while it all fits in one page.
 */
export function Pager({ pagination, onChange, label }: Props) {
  const { t } = useTranslation('common')
  if (!pagination) return null
  const pages = Math.max(1, Math.ceil(pagination.total / Math.max(1, pagination.page_size)))
  if (pages <= 1 && pagination.page <= 1) return null
  return (
    <nav aria-label={label} className="flex items-center justify-between gap-3 pt-1">
      <Button variant="outline" size="sm" disabled={pagination.page <= 1} onClick={() => onChange(pagination.page - 1)}>
        <ChevronLeft aria-hidden />
        {t('pager.previous')}
      </Button>
      <span className="num text-sm text-muted-foreground">{t('pager.page', { page: pagination.page, pages })}</span>
      <Button variant="outline" size="sm" disabled={!pagination.has_more} onClick={() => onChange(pagination.page + 1)}>
        {t('pager.next')}
        <ChevronRight aria-hidden />
      </Button>
    </nav>
  )
}
