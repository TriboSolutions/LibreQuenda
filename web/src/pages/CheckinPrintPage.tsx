import { ArrowLeft, Printer } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useParams, useSearchParams } from 'react-router'
import { BrailleCode } from '@/components/BrailleCode'
import { QrImage } from '@/components/QrImage'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useCouncilAdmin } from '@/hooks/useCouncilAdmin'
import { usePageTitle } from '@/hooks/usePageTitle'
import { api } from '@/lib/api'
import { entranceUrl } from '@/lib/entrance'

/**
 * /:slug/admin/facilities/:id/imprimir: a page made to print, for the door.
 * ?que=qr the facility's check-in QR; ?que=codigo its entry code (&braille=1 adds it in braille).
 */
export function CheckinPrintPage() {
  const { slug = '', id = '' } = useParams()
  const [params, setParams] = useSearchParams()
  const what = params.get('que') === 'codigo' ? 'code' : 'qr'
  const braille = params.get('braille') === '1'
  const setBraille = (on: boolean) =>
    setParams(
      (p) => {
        if (on) p.set('braille', '1')
        else p.delete('braille')
        return p
      },
      { replace: true },
    )
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const allowed = useCouncilAdmin(slug)
  const facility = useAsync(allowed ? () => api.council.facility(token, slug, id) : null, `print-facility:${slug}:${id}:${allowed}`)
  const settings = useAsync(allowed ? () => api.council.checkinSettings(token, slug, id) : null, `print-checkin:${slug}:${id}:${allowed}`)
  usePageTitle(what === 'qr' ? t('admin:checkin.print.qrTitle') : t('admin:checkin.print.codeTitle'), facility.data?.name)

  const error = facility.error ?? settings.error
  const ready = facility.data && settings.data
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
        {what === 'code' && (
          <div className="flex items-center gap-2">
            <Checkbox id="print-braille" checked={braille} onCheckedChange={(on) => setBraille(on === true)} aria-describedby="print-braille-help" />
            <Label htmlFor="print-braille">{t('admin:checkin.print.braille')}</Label>
          </div>
        )}
        <Button type="button" onClick={() => window.print()} disabled={!ready}>
          <Printer aria-hidden />
          {t('admin:checkin.print.print')}
        </Button>
      </div>
      {what === 'code' && (
        <p id="print-braille-help" className="-mt-4 mb-8 text-sm text-neutral-700 print:hidden">
          {t('admin:checkin.print.brailleHelp')}
        </p>
      )}

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
        <article className="grid justify-items-center gap-6 text-center">
          <h1 className="text-4xl font-bold">{t('admin:checkin.print.codeTitle')}</h1>
          <p className="text-2xl">{facility.data!.name}</p>
          <p className="num font-mono text-7xl font-bold tracking-widest">
            {settings.data!.code.slice(0, 3)} {settings.data!.code.slice(3)}
          </p>
          {braille && <BrailleCode code={settings.data!.code} className="text-black" />}
          <ol className="grid max-w-[46ch] list-decimal gap-2 pl-6 text-left text-lg">
            <li>{t('admin:checkin.print.codeStep1', { url: entrance })}</li>
            <li>{t('admin:checkin.print.qrStep2')}</li>
            <li>{settings.data!.flash_booking ? t('admin:checkin.print.qrStep3Flash') : t('admin:checkin.print.qrStep3')}</li>
          </ol>
        </article>
      )}
    </main>
  )
}
