import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Skeleton } from '@/components/ui/skeleton'
import type { Tenant } from '@/lib/api'

type Props = {
  section: string // what this page of the council is: "Instalacións deportivas", "Xestión do concello"
  council: Tenant | undefined // undefined while the list loads
  otherCouncils: boolean // more than one council on this site: offer the way to the others
}

/** The page title of a council: its logo and name, under the section of the site it is in. */
export function CouncilHeading({ section, council, otherCouncils }: Props) {
  const { t } = useTranslation('councils')
  return (
    <div className="min-w-0">
      <p className="flex flex-wrap items-center gap-x-2 text-sm font-medium text-muted-foreground">
        {section}
        {otherCouncils && (
          <>
            <span aria-hidden="true">·</span>
            <Link to="/" className="rounded-sm text-primary underline-offset-4 hover:underline">
              {t('councils:allCouncils')}
            </Link>
          </>
        )}
      </p>
      <h1 className="mt-1 flex min-w-0 items-center gap-3 text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">
        {council ? (
          <>
            {council.logo_url && <img src={council.logo_url} alt="" className="size-9 shrink-0 object-contain sm:size-10" />}
            <span className="min-w-0">{council.name}</span>
          </>
        ) : (
          <Skeleton className="h-9 w-72 max-w-full" />
        )}
      </h1>
    </div>
  )
}
