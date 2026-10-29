import { ArrowLeft, Printer } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useParams, useSearchParams } from 'react-router'
import { QrImage } from '@/components/QrImage'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useCouncilAdmin } from '@/hooks/useCouncilAdmin'
import { usePageTitle } from '@/hooks/usePageTitle'
import { api } from '@/lib/api'
import { entranceUrl } from '@/lib/entrance'
import { formatDateWithYear, formatTime } from '@/lib/time'

/**
 * /:slug/admin/facilities/:id/imprimir: a page made to print, for the door.
 * ?que=qr the facility's check-in QR; ?que=codigos&data=YYYY-MM-DD the day's turn codes.
 */
export function CheckinPrintPage() {
  const { slug = '', id = '' } = useParams()
  const [params] = useSearchParams()
  const what = params.get('que') === 'codigos' ? 'codes' : 'qr'
  const date = params.get('data') ?? ''
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const allowed = useCouncilAdmin(slug)
  const facility = useAsync(allowed ? () => api.council.facility(token, slug, id) : null, `print-facility:${slug}:${id}:${allowed}`)
  const settings = useAsync(allowed && what === 'qr' ? () => api.council.checkinSettings(token, slug, id) : null, `print-qr:${slug}:${id}:${allowed}`)
  const codes = useAsync(
    allowed && what === 'codes' ? () => api.council.turnCodes(token, slug, id, date) : null,
    `print-codes:${slug}:${id}:${date}:${allowed}`,
  )
  usePageTitle(what === 'qr' ? t('admin:checkin.print.qrTitle') : t('admin:checkin.print.codesTitle'), facility.data?.name)

  const error = facility.error ?? settings.error ?? codes.error
  const ready = facility.data && (what === 'qr' ? settings.data : codes.data)
  const entrance = `${window.location.host}/${slug}/${facility.data?.slug ?? ''}/entrada`

  return (
    // printed on white paper whatever the screen's theme
    <main id="main" tabIndex={-1} className="mx-auto min-h-screen max-w-[720px] bg-white px-4 py-8 text-black sm:px-6 print:max-w-none print:p-0">
      <div className="mb-8 flex flex-wrap items-center justify-between gap-3 print:hidden">
        <Button asChild variant="outline" className="border-neutral-400 bg-white text-black hover:bg-neutral-100">
          <Link to={`/${slug}/admin/facilities/${id}?vista=entrada`}>
            <ArrowLeft aria-hidden />
            {t('admin:checkin.print.back')}
          </Link>
        </Button>
        <Button type="button" onClick={() => window.print()} disabled={!ready}>
          <Printer aria-hidden />
          {t('admin:checkin.print.print')}
        </Button>
      </div>

      {error ? (
        <p role="alert" className="text-red-800">
          {error.message}
        </p>
      ) : !ready ? (
        <Skeleton className="h-96 w-full" />
      ) : what === 'qr' ? (
        <article className="grid justify-items-center gap-6 text-center">
          <h1 className="text-4xl font-bold">{t('admin:checkin.print.qrTitle')}</h1>
          <p className="text-2xl">{facility.data!.name}</p>
          <QrImage
            code={entranceUrl(slug, facility.data!.slug, settings.data!.qr_token)}
            alt={t('admin:checkin.printedQr.qrAlt')}
            className="max-w-80 ring-0"
          />
          <ol className="grid max-w-[46ch] list-decimal gap-2 pl-6 text-left text-lg">
            <li>{t('admin:checkin.print.qrStep1')}</li>
            <li>{t('admin:checkin.print.qrStep2')}</li>
            <li>{settings.data!.flash_booking ? t('admin:checkin.print.qrStep3Flash') : t('admin:checkin.print.qrStep3')}</li>
          </ol>
        </article>
      ) : (
        <article className="grid gap-4">
          <h1 className="text-3xl font-bold">{t('admin:checkin.print.codesTitle')}</h1>
          <p className="text-xl">
            {facility.data!.name} · {formatDateWithYear(codes.data!.date)}
          </p>
          <p>{t('admin:checkin.print.codesHelp', { url: entrance })}</p>
          {codes.data!.turns.length === 0 ? (
            <p className="rounded-md border border-dashed border-neutral-400 px-4 py-3">{t('admin:checkin.print.noTurns')}</p>
          ) : (
            <table className="w-full border-collapse text-lg">
              <caption className="sr-only">{t('admin:checkin.print.codesTitle')}</caption>
              <thead>
                <tr className="border-b-2 border-black text-left">
                  <th scope="col" className="py-2 pr-4">
                    {t('admin:checkin.print.colTurn')}
                  </th>
                  <th scope="col" className="py-2">
                    {t('admin:checkin.print.colCode')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {codes.data!.turns.map((turn) => (
                  <tr key={turn.starts_at} className="border-b border-neutral-400 break-inside-avoid">
                    <td className="num py-2 pr-4">
                      {formatTime(turn.starts_at, codes.data!.timezone)}–{formatTime(turn.ends_at, codes.data!.timezone)}
                    </td>
                    <td className="num py-2 font-mono text-2xl font-bold tracking-widest">
                      {turn.code.slice(0, 3)} {turn.code.slice(3)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <p className="text-sm">{t('admin:checkin.print.codesValid')}</p>
        </article>
      )}
    </main>
  )
}
