import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { useSearchParams } from 'react-router'
import { DatePicker } from '@/components/DatePicker'
import { AccountSettings } from '@/components/AccountSettings'
import { ImagePicker } from '@/components/ImagePicker'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { SettingsRow } from '@/components/SettingsRow'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { languages } from '@/i18n'
import { api, type Me, type StoredImage } from '@/lib/api'

const SAME_AS_WEBSITE = 'auto' // Select items cannot have an empty value

/** /perfil: what LibreQuenda keeps about the user. Name, email and password belong to the sign-in service. */
export function ProfilePage() {
  const { t } = useTranslation(['profile', 'common'])
  const auth = useAuth()
  const { me } = useMe()
  const { login } = useLogin()
  const [params] = useSearchParams()
  const tenants = useAsync(() => api.tenants(), 'tenants')

  let body
  if (auth.isLoading || (auth.isAuthenticated && !me)) {
    body = <Skeleton className="h-64 w-full max-w-[760px]" />
  } else if (!auth.isAuthenticated || !me) {
    body = (
      <div className="grid max-w-[760px] gap-4 rounded-lg border bg-card p-5">
        {/* the sign-in comes back here with the code still in the address, and verifies it */}
        <p>{params.has('verificar_correo') ? t('profile:signInToVerify') : t('profile:signIn')}</p>
        <div>
          <Button onClick={login}>{t('common:signIn')}</Button>
        </div>
      </div>
    )
  } else {
    body = (
      <>
        <ProfileForm key={me.id} me={me} />
        <AccountSettings />
      </>
    )
  }

  return (
    <div className="min-h-screen">
      <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={null} />
      <main className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <h1 className="mb-6 text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">{t('profile:title')}</h1>
        {body}
      </main>
    </div>
  )
}

function ProfileForm({ me }: { me: Me }) {
  const { t } = useTranslation(['profile', 'common'])
  const auth = useAuth()
  const { reload } = useMe()
  const [avatar, setAvatar] = useState<StoredImage | null>(me.avatar_ref ? { ref: me.avatar_ref, url: me.avatar_url } : null)
  const [birthDate, setBirthDate] = useState(me.birth_date)
  const [phone, setPhone] = useState(me.phone)
  const [language, setLanguage] = useState(me.preferred_language || SAME_AS_WEBSITE)
  const [reminders, setReminders] = useState(me.email_reminders)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [today] = useState(() => new Date().toISOString().slice(0, 10)) // the date picker's upper bound, fixed while the form is open

  const save = async (event: React.FormEvent) => {
    event.preventDefault()
    const token = auth.user?.access_token
    if (!token) return
    if (!birthDate) {
      setNotice(null)
      setError(t('profile:birthDateRequired'))
      return
    }
    setSaving(true)
    setError(null)
    setNotice(null)
    try {
      await api.updateProfile(token, {
        birth_date: birthDate,
        phone,
        avatar_ref: avatar?.ref ?? '', // the reference, never the URL
        preferred_language: language === SAME_AS_WEBSITE ? '' : language,
        email_reminders: reminders,
      })
      reload()
      setNotice(t('profile:saved'))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <SettingsRow id="sec-details" title={t('profile:details')} description={t('profile:subtitle')}>
      <form onSubmit={(e) => void save(e)} className="grid gap-5 rounded-lg border bg-card p-5">
        <ImagePicker label={t('profile:photo')} folder="users" shape="avatar" value={avatar} onChange={setAvatar} />
        <div className="grid gap-5 border-t pt-5 sm:grid-cols-2">
          <div className="grid content-start gap-2">
            <Label htmlFor="p-birth">{t('profile:birthDate')}</Label>
            <DatePicker
              id="p-birth"
              min="1906-01-01"
              max={today}
              defaultMonth={`${Number(today.slice(0, 4)) - 30}-01-01`}
              value={birthDate}
              placeholder={t('profile:birthDatePlaceholder')}
              aria-describedby="p-birth-help"
              onChange={setBirthDate}
              className="w-full"
            />
            <p id="p-birth-help" className="text-xs text-muted-foreground">
              {t('profile:birthDateHelp')}
            </p>
          </div>
          <div className="grid content-start gap-2">
            <Label htmlFor="p-phone">
              {t('profile:phone')} <span className="font-normal text-muted-foreground">({t('profile:optional')})</span>
            </Label>
            <Input id="p-phone" type="tel" maxLength={30} value={phone} aria-describedby="p-phone-help" onChange={(e) => setPhone(e.target.value)} className="num" />
            <p id="p-phone-help" className="text-xs text-muted-foreground">
              {t('profile:phoneHelp')}
            </p>
          </div>
          <div className="grid content-start gap-2">
            <Label htmlFor="p-language">{t('profile:language')}</Label>
            <Select value={language} onValueChange={setLanguage}>
              <SelectTrigger id="p-language" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={SAME_AS_WEBSITE}>{t('profile:languageAuto')}</SelectItem>
                {languages.map((l) => (
                  <SelectItem key={l.code} value={l.code} lang={l.code}>
                    {l.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <div className="flex items-start gap-2">
          <Checkbox id="p-reminders" checked={reminders} onCheckedChange={(value) => setReminders(value === true)} className="mt-0.5" />
          <Label htmlFor="p-reminders" className="leading-5 font-normal">
            {t('profile:reminders')}
          </Label>
        </div>

        <div aria-live="polite" className="empty:hidden">
          {notice && (
            <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
              <AlertDescription className="text-pill-positive-ink">{notice}</AlertDescription>
            </Alert>
          )}
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
        </div>
        <div className="flex justify-end border-t pt-4">
          <Button type="submit" disabled={saving}>
            {saving ? t('profile:saving') : t('profile:save')}
          </Button>
        </div>
      </form>
    </SettingsRow>
  )
}
