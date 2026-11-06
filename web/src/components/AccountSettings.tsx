import { Check, X } from 'lucide-react'
import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { useSearchParams } from 'react-router'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { SettingsRow } from '@/components/SettingsRow'
import { useAsync } from '@/hooks/useAsync'
import { useBranding } from '@/hooks/useBranding'
import { useMe } from '@/hooks/useMe'
import { api, type Account } from '@/lib/api'

type Status = { notice: string | null; error: string | null }
const IDLE: Status = { notice: null, error: null }

/** Runs an account change and keeps its own message, next to its own card. */
function useAction() {
  const [busy, setBusy] = useState(false)
  const [status, setStatus] = useState<Status>(IDLE)
  const run = async (action: () => Promise<void>, notice: string) => {
    setBusy(true)
    setStatus(IDLE)
    try {
      await action()
      setStatus({ notice, error: null })
      return true
    } catch (e) {
      setStatus({ notice: null, error: e instanceof Error ? e.message : String(e) })
      return false
    } finally {
      setBusy(false)
    }
  }
  return { busy, status, run }
}

function Messages({ status }: { status: Status }) {
  return (
    <div aria-live="polite" className="empty:hidden">
      {status.notice && (
        <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
          <AlertDescription className="text-pill-positive-ink">{status.notice}</AlertDescription>
        </Alert>
      )}
      {status.error && (
        <Alert variant="destructive">
          <AlertDescription>{status.error}</AlertDescription>
        </Alert>
      )}
    </div>
  )
}

/** One part of the account card, ruled off from the one above. */
function Card({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <section aria-labelledby={id} className="grid content-start gap-4 p-5 not-first:border-t">
      <h3 id={id} className="font-semibold">
        {title}
      </h3>
      {children}
    </section>
  )
}

/**
 * The sign-in account (name, email, password), kept by Zitadel but edited here:
 * residents never go to Zitadel's console.
 */
export function AccountSettings() {
  const { t } = useTranslation(['profile', 'common'])
  const { branding } = useBranding()
  const auth = useAuth()
  const token = auth.user?.access_token ?? ''
  const account = useAsync(token ? () => api.account.get(token) : null, `account:${auth.user?.profile.sub}`)

  return (
    <SettingsRow id="sec-sign-in" title={t('accountSettings.title')} description={t('accountSettings.help', { app: branding.name })}>
      {account.error ? (
        <div className="flex items-center justify-between gap-4 rounded-md bg-pill-negative-bg px-4 py-3 text-pill-negative-ink">
          <p>{account.error.message}</p>
          <Button variant="ghost" size="sm" onClick={account.reload}>
            {t('common:retry')}
          </Button>
        </div>
      ) : !account.data ? (
        <Skeleton className="h-96" />
      ) : (
        <div className="rounded-lg border bg-card">
          <NameCard token={token} account={account.data} />
          <EmailCard token={token} account={account.data} onChanged={account.reload} />
          <PasswordCard token={token} account={account.data} />
        </div>
      )}
    </SettingsRow>
  )
}

function NameCard({ token, account }: { token: string; account: Account }) {
  const { t } = useTranslation('profile')
  const { reload } = useMe()
  const [given, setGiven] = useState(account.given_name)
  const [family, setFamily] = useState(account.family_name)
  const { busy, status, run } = useAction()

  const save = (event: FormEvent) => {
    event.preventDefault()
    void run(async () => {
      await api.account.setName(token, given, family)
      reload() // the top bar shows the new name
    }, t('accountSettings.nameSaved'))
  }

  return (
    <Card id="acc-name" title={t('accountSettings.name')}>
      <form onSubmit={save} className="grid gap-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="grid gap-2">
            <Label htmlFor="acc-given">{t('accountSettings.givenName')}</Label>
            <Input id="acc-given" required maxLength={100} autoComplete="given-name" value={given} onChange={(e) => setGiven(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="acc-family">{t('accountSettings.familyName')}</Label>
            <Input id="acc-family" required maxLength={100} autoComplete="family-name" value={family} onChange={(e) => setFamily(e.target.value)} />
          </div>
        </div>
        <Messages status={status} />
        <div>
          <Button type="submit" variant="outline" disabled={busy}>
            {busy ? t('saving') : t('accountSettings.saveName')}
          </Button>
        </div>
      </form>
    </Card>
  )
}

function EmailCard({ token, account, onChanged }: { token: string; account: Account; onChanged: () => void }) {
  const { t } = useTranslation('profile')
  const { reload } = useMe()
  const [params, setParams] = useSearchParams()
  const [editing, setEditing] = useState(false)
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [sentTo, setSentTo] = useState<string | null>(null) // a code is on its way to this address
  const { busy, status, run } = useAction()
  const awaitingCode = !account.email_verified || sentTo !== null

  const verify = (value: string) =>
    run(async () => {
      await api.account.verifyEmail(token, value.trim())
      setCode('')
      setSentTo(null)
      onChanged()
      reload() // a council role waiting for a verified email is granted now
    }, t('accountSettings.emailVerified'))

  // the link in the verification email lands here: /perfil?verificar_correo=<code>
  const linkCode = params.get('verificar_correo')
  const usedLink = useRef(false)
  useEffect(() => {
    if (!linkCode || usedLink.current) return
    usedLink.current = true
    setParams((p) => {
      p.delete('verificar_correo')
      return p
    }, { replace: true })
    void verify(linkCode)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once, for the code in the link
  }, [linkCode])

  const change = (event: FormEvent) => {
    event.preventDefault()
    const address = email.trim()
    void run(async () => {
      await api.account.setEmail(token, address)
      setSentTo(address)
      setEditing(false)
      setEmail('')
      onChanged()
      reload()
    }, t('accountSettings.codeSent', { email: address }))
  }

  return (
    <Card id="acc-email" title={t('accountSettings.email')}>
      <p className="min-w-0 break-all">
        {account.email}{' '}
        <span className={account.email_verified ? 'text-pill-positive-ink' : 'text-muted-foreground'}>
          · {account.email_verified ? t('verified') : t('notVerified')}
        </span>
      </p>

      {awaitingCode && (
        <form
          className="grid gap-2"
          onSubmit={(event) => {
            event.preventDefault()
            void verify(code)
          }}
        >
          <Label htmlFor="acc-code">{t('accountSettings.code')}</Label>
          <p className="text-sm text-muted-foreground">{t('accountSettings.codeHelp', { email: sentTo ?? account.email })}</p>
          <div className="flex flex-wrap gap-2">
            <Input
              id="acc-code"
              required
              maxLength={32}
              autoComplete="one-time-code"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              className="num w-40 uppercase"
            />
            <Button type="submit" disabled={busy}>
              {t('accountSettings.verify')}
            </Button>
          </div>
          <div>
            <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => void run(() => api.account.resendEmailCode(token), t('accountSettings.codeSent', { email: sentTo ?? account.email }))}>
              {t('accountSettings.resend')}
            </Button>
          </div>
        </form>
      )}

      {editing ? (
        <form onSubmit={change} className="grid gap-2">
          <Label htmlFor="acc-new-email">{t('accountSettings.newEmail')}</Label>
          <Input id="acc-new-email" type="email" required maxLength={255} autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          <p className="text-sm text-muted-foreground">{t('accountSettings.newEmailHelp')}</p>
          <div className="flex flex-wrap gap-2">
            <Button type="submit" variant="outline" disabled={busy}>
              {t('accountSettings.sendCode')}
            </Button>
            <Button type="button" variant="ghost" onClick={() => setEditing(false)}>
              {t('accountSettings.cancel')}
            </Button>
          </div>
        </form>
      ) : (
        <div>
          <Button type="button" variant="outline" onClick={() => setEditing(true)}>
            {t('accountSettings.changeEmail')}
          </Button>
        </div>
      )}
      <Messages status={status} />
    </Card>
  )
}

function PasswordCard({ token, account }: { token: string; account: Account }) {
  const { t } = useTranslation('profile')
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [repeat, setRepeat] = useState('')
  const { busy, status, run } = useAction()
  const rules = account.password_rules
  // the same character classes Zitadel checks ([^A-Za-z0-9] is a symbol)
  const checks = [
    { ok: next.length >= rules.min_length, label: t('accountSettings.rule.length', { count: rules.min_length }) },
    ...(rules.uppercase ? [{ ok: /[A-Z]/.test(next), label: t('accountSettings.rule.uppercase') }] : []),
    ...(rules.lowercase ? [{ ok: /[a-z]/.test(next), label: t('accountSettings.rule.lowercase') }] : []),
    ...(rules.number ? [{ ok: /[0-9]/.test(next), label: t('accountSettings.rule.number') }] : []),
    ...(rules.symbol ? [{ ok: /[^A-Za-z0-9]/.test(next), label: t('accountSettings.rule.symbol') }] : []),
  ]
  const mismatch = repeat !== '' && repeat !== next

  const save = (event: FormEvent) => {
    event.preventDefault()
    void run(async () => {
      await api.account.setPassword(token, current, next)
      setCurrent('')
      setNext('')
      setRepeat('')
    }, t('accountSettings.passwordSaved'))
  }

  return (
    <Card id="acc-password" title={t('accountSettings.password')}>
      <form onSubmit={save} className="grid gap-4">
        <div className="grid gap-2 sm:w-1/2 sm:pr-2">
          <Label htmlFor="acc-current">{t('accountSettings.currentPassword')}</Label>
          <Input id="acc-current" type="password" required maxLength={72} autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} />
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="grid content-start gap-2">
            <Label htmlFor="acc-next">{t('accountSettings.newPassword')}</Label>
            <Input id="acc-next" type="password" required maxLength={72} autoComplete="new-password" aria-describedby="acc-rules" value={next} onChange={(e) => setNext(e.target.value)} />
            <ul id="acc-rules" className="grid gap-0.5 text-sm">
              {checks.map((check) => (
                <li key={check.label} className={`flex items-center gap-1.5 ${check.ok ? 'text-pill-positive-ink' : 'text-muted-foreground'}`}>
                  {check.ok ? <Check className="size-3.5" aria-hidden /> : <X className="size-3.5" aria-hidden />}
                  {check.label}
                </li>
              ))}
            </ul>
          </div>
          <div className="grid content-start gap-2">
            <Label htmlFor="acc-repeat">{t('accountSettings.repeatPassword')}</Label>
            <Input
              id="acc-repeat"
              type="password"
              maxLength={72}
              required
              autoComplete="new-password"
              aria-invalid={mismatch}
              value={repeat}
              onChange={(e) => setRepeat(e.target.value)}
            />
            {mismatch && <p className="text-sm text-destructive">{t('accountSettings.mismatch')}</p>}
          </div>
        </div>
        <Messages status={status} />
        <div>
          <Button type="submit" variant="outline" disabled={busy || mismatch || checks.some((c) => !c.ok) || current === ''}>
            {busy ? t('saving') : t('accountSettings.savePassword')}
          </Button>
        </div>
      </form>
    </Card>
  )
}
