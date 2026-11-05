import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { CircleCheck, House, Phone } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMe } from '@/hooks/useMe'
import { api, type Tenant } from '@/lib/api'
import { formatDateWithYear } from '@/lib/time'

/** «Só para veciños»: the marker on a residents-only facility, and on the events held there. */
export function ResidentsOnlyBadge({ className = '' }: { className?: string }) {
  const { t } = useTranslation('facility')
  return (
    <span className={`inline-flex w-fit items-center gap-1.5 rounded-full bg-pill-neutral-bg px-2.5 py-0.5 text-sm font-medium text-pill-neutral-ink ${className}`}>
      <House className="size-3.5 shrink-0" aria-hidden />
      {t('residentRequest.badge')}
    </span>
  )
}

type Props = {
  tenant: Tenant
  open: boolean
  onClose: () => void
}

/**
 * Asking the council to verify the person as its resident: phone it, or send
 * what it needs to look them up in its census (DNI, name, postal code, street).
 * The request reaches its administrators (Veciños tab) and is deleted once
 * they decide, or after 15 days. A pending request can be withdrawn here.
 */
export function ResidentRequestDialog({ tenant, open, onClose }: Props) {
  const { t } = useTranslation('facility')
  const auth = useAuth()
  const { me, reload } = useMe()
  const pending = me?.resident_requests?.find((r) => r.tenant_slug === tenant.slug)
  const [dni, setDni] = useState('')
  const [fullName, setFullName] = useState(me?.full_name ?? '')
  const [postalCode, setPostalCode] = useState('')
  const [street, setStreet] = useState('')
  const [busy, setBusy] = useState(false)
  const [sent, setSent] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const close = () => {
    setSent(false)
    setError(null)
    onClose()
  }

  const run = async (action: (token: string) => Promise<unknown>) => {
    const token = auth.user?.access_token
    if (!token) return false
    setBusy(true)
    setError(null)
    try {
      await action(token)
      reload()
      return true
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return false
    } finally {
      setBusy(false)
    }
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    const input = { dni: dni.trim(), full_name: fullName.trim(), postal_code: postalCode.trim(), street: street.trim() }
    if (await run((token) => api.requestResidency(token, tenant.slug, input))) {
      setSent(true)
      setDni('')
      setPostalCode('')
      setStreet('')
    }
  }

  const withdraw = async () => {
    await run((token) => api.withdrawResidency(token, tenant.slug))
  }

  return (
    <Dialog open={open} onOpenChange={(value) => !value && close()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('residentRequest.title', { council: tenant.name })}</DialogTitle>
          <DialogDescription>{t('residentRequest.lead', { council: tenant.name })}</DialogDescription>
        </DialogHeader>

        {/* the first way: phone the council */}
        <section aria-labelledby="resident-call" className="grid gap-1.5">
          <h3 id="resident-call" className="font-medium">
            {t('residentRequest.callTitle')}
          </h3>
          {tenant.contact_phone ? (
            <a href={`tel:${tenant.contact_phone.replace(/\s/g, '')}`} className="inline-flex w-fit items-center gap-2 font-medium text-primary underline-offset-4 hover:underline">
              <Phone className="size-4" aria-hidden />
              <span className="num">{tenant.contact_phone}</span>
            </a>
          ) : (
            <p className="text-muted-foreground">{tenant.contact_email ? t('residentRequest.callEmail', { email: tenant.contact_email }) : t('residentRequest.callOffice')}</p>
          )}
        </section>

        <section aria-labelledby="resident-form" className="grid gap-3 border-t pt-4">
          <h3 id="resident-form" className="font-medium">
            {t('residentRequest.formTitle')}
          </h3>
          {sent || pending ? (
            <div className="grid gap-3">
              <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
                <CircleCheck className="size-4" aria-hidden />
                <AlertDescription className="text-pill-positive-ink">
                  {pending ? t('residentRequest.pending', { date: formatDateWithYear(pending.requested_at.slice(0, 10)) }) : t('residentRequest.sent')}
                </AlertDescription>
              </Alert>
              {pending && (
                <Button type="button" variant="outline" className="w-fit" disabled={busy} onClick={() => void withdraw()}>
                  {t('residentRequest.withdraw')}
                </Button>
              )}
            </div>
          ) : (
            <form id="resident-request-form" onSubmit={(e) => void submit(e)} className="grid gap-3">
              <div className="grid gap-1.5">
                <Label htmlFor="rr-dni">{t('residentRequest.dni')}</Label>
                <Input id="rr-dni" required maxLength={12} autoComplete="off" value={dni} onChange={(e) => setDni(e.target.value)} aria-describedby="rr-dni-help" />
                <p id="rr-dni-help" className="text-sm text-muted-foreground">
                  {t('residentRequest.dniHelp')}
                </p>
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="rr-name">{t('residentRequest.fullName')}</Label>
                <Input id="rr-name" required maxLength={200} autoComplete="name" value={fullName} onChange={(e) => setFullName(e.target.value)} />
              </div>
              <div className="grid gap-3 sm:grid-cols-[140px_minmax(0,1fr)]">
                <div className="grid gap-1.5">
                  <Label htmlFor="rr-postal">{t('residentRequest.postalCode')}</Label>
                  <Input id="rr-postal" required inputMode="numeric" pattern="[0-9]{5}" maxLength={5} autoComplete="postal-code" value={postalCode} onChange={(e) => setPostalCode(e.target.value)} />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="rr-street">{t('residentRequest.street')}</Label>
                  <Input id="rr-street" required maxLength={200} autoComplete="address-line1" value={street} onChange={(e) => setStreet(e.target.value)} />
                </div>
              </div>
              <p className="text-sm text-muted-foreground">{t('residentRequest.privacy')}</p>
            </form>
          )}
          <div aria-live="polite">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </div>
        </section>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={close}>
            {t('residentRequest.close')}
          </Button>
          {!sent && !pending && (
            <Button type="submit" form="resident-request-form" disabled={busy}>
              {t('residentRequest.send')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
