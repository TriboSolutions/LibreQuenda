import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useMe } from '@/hooks/useMe'
import { api } from '@/lib/api'

export function SuperadminsPanel() {
  const { t } = useTranslation(['admin', 'common'])
  const auth = useAuth()
  const { me } = useMe()
  const token = auth.user?.access_token ?? ''
  const superadmins = useAsync(() => api.admin.superadmins(token), 'admin:superadmins')
  const [email, setEmail] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const list = superadmins.data?.superadmins ?? []

  const run = async (action: () => Promise<unknown>, message: string) => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      await action()
      setNotice(message)
      superadmins.reload()
      return true
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return false
    } finally {
      setBusy(false)
    }
  }

  const add = async (event: React.FormEvent) => {
    event.preventDefault()
    const value = email.trim()
    if (await run(() => api.admin.addSuperadmin(token, value), t('admin:superadmins.added', { email: value }))) setEmail('')
  }

  return (
    <section aria-labelledby="superadmins-title" className="max-w-[760px] rounded-lg border bg-card p-5">
      <h2 id="superadmins-title" className="text-lg font-semibold">
        {t('admin:tabs.superadmins')}
      </h2>
      <p className="mt-1 text-muted-foreground">{t('admin:superadmins.help')}</p>

      <div aria-live="polite">
        {notice && (
          <Alert className="mt-4 border-0 bg-pill-positive-bg text-pill-positive-ink">
            <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
          </Alert>
        )}
        {error && (
          <Alert variant="destructive" className="mt-4">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>

      {superadmins.error ? (
        <p className="mt-4 text-destructive">{t('admin:superadmins.loadError')}</p>
      ) : superadmins.loading && !superadmins.data ? (
        <Skeleton className="mt-4 h-5 w-64" />
      ) : (
        <ul className="mt-3">
          {list.map((admin) => (
            <li key={admin.user_id} className="flex items-center justify-between gap-3 border-t py-2.5 first:border-t-0">
              <div className="min-w-0">
                <p className="truncate font-medium">
                  {admin.full_name || admin.email}
                  {admin.user_id === me?.id && <span className="font-normal text-muted-foreground"> · {me?.email}</span>}
                </p>
                <p className="truncate text-muted-foreground">{admin.email}</p>
              </div>
              {admin.from_env ? (
                <span className="shrink-0 rounded-full bg-pill-neutral-bg px-2 py-0.5 text-xs font-medium text-pill-neutral-ink">
                  {t('admin:superadmins.fromEnv')}
                </span>
              ) : (
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-destructive hover:text-destructive"
                  disabled={busy || list.length <= 1}
                  onClick={() => void run(() => api.admin.removeSuperadmin(token, admin.user_id), t('admin:notice.saved'))}
                >
                  {t('admin:superadmins.remove')}
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}

      <form onSubmit={(e) => void add(e)} className="mt-4 grid gap-2 border-t pt-4">
        <Label htmlFor="superadmin-email">{t('admin:superadmins.addLabel')}</Label>
        <div className="flex gap-2">
          <Input id="superadmin-email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} />
          <Button type="submit" variant="outline" disabled={busy}>
            {t('admin:superadmins.add')}
          </Button>
        </div>
      </form>
    </section>
  )
}
