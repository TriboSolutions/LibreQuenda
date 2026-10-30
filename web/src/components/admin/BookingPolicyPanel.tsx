import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { useAsync } from '@/hooks/useAsync'
import { api } from '@/lib/api'

/** The superadministrators' booking rules across councils: one council at a time per person. */
export function BookingPolicyPanel() {
  const { t } = useTranslation('admin')
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const policy = useAsync(() => api.admin.bookingPolicy(token), 'booking-policy')
  const [busy, setBusy] = useState(false)
  const [status, setStatus] = useState<string | null>(null)

  const change = async (on: boolean) => {
    setBusy(true)
    setStatus(null)
    try {
      await api.admin.saveBookingPolicy(token, { one_council_per_account: on })
      policy.reload()
      setStatus(t('bookingPolicy.saved'))
    } catch (e) {
      setStatus(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section aria-labelledby="booking-policy-title" className="mt-6 grid max-w-[760px] gap-4 rounded-lg border bg-card p-5">
      <h2 id="booking-policy-title" className="text-lg font-semibold">
        {t('bookingPolicy.title')}
      </h2>
      {!policy.data ? (
        <Skeleton className="h-10 w-full" />
      ) : (
        <div className="flex items-start gap-3">
          <Switch
            id="one-council"
            checked={policy.data.one_council_per_account}
            disabled={busy}
            aria-describedby="one-council-help"
            onCheckedChange={(on) => void change(on)}
          />
          <div className="grid gap-1">
            <Label htmlFor="one-council">{t('bookingPolicy.oneCouncil')}</Label>
            <p id="one-council-help" className="text-sm text-muted-foreground">
              {t('bookingPolicy.oneCouncilHelp')}
            </p>
          </div>
        </div>
      )}
      <p role="status" className="text-sm text-muted-foreground empty:hidden">
        {status}
      </p>
    </section>
  )
}
