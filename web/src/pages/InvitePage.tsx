import { CalendarClock, CircleCheck, MapPin, Users } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, Navigate, useParams } from 'react-router'
import { FacilityInfo } from '@/components/FacilityInfo'
import { SpotsMeter } from '@/components/SpotsMeter'
import { TopBar } from '@/components/TopBar'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { useAsync } from '@/hooks/useAsync'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import { api, ApiError, type PublicInvite } from '@/lib/api'
import { formatBookingDay, formatTime, localDate } from '@/lib/time'
import { usePageTitle } from '@/hooks/usePageTitle'
import { SiteFooter } from '@/components/SiteFooter'

/** /invitacion/:token: anyone with the link joins a booking, with an account or as a guest. */
export function InvitePage() {
  const { token: inviteToken = '' } = useParams()
  const { t } = useTranslation(['invites', 'common'])
  const tenants = useAsync(() => api.tenants(), 'tenants')
  const invite = useAsync(() => api.invites.open(inviteToken), `invite:${inviteToken}`)
  usePageTitle(t('invites:page.title'), invite.data?.facility_name)

  let body
  if (invite.error) {
    body = (
      <Alert variant={invite.error instanceof ApiError && invite.error.status === 404 ? 'default' : 'destructive'} className="mt-6 max-w-[640px]">
        <AlertDescription>
          {invite.error instanceof ApiError && invite.error.status === 404 ? t('invites:page.notFound') : invite.error.message}
        </AlertDescription>
      </Alert>
    )
  } else if (!invite.data) {
    body = <Skeleton className="mt-6 h-72 w-full max-w-[640px]" />
  } else {
    body = <InviteCard invite={invite.data} inviteToken={inviteToken} onChanged={invite.reload} />
  }

  return (
    <div className="min-h-screen">
      <TopBar tenants={tenants.data?.tenants ?? []} tenantSlug={invite.data?.tenant_slug ?? null} />
      <main id="main" tabIndex={-1} className="mx-auto max-w-[1152px] px-4 pt-8 pb-16 sm:px-6">
        <h1 className="text-[28px] leading-9 font-bold tracking-[-0.02em] sm:text-[32px] sm:leading-10">
          {invite.data?.owner_name ? t('invites:page.invitedBy', { name: invite.data.owner_name }) : t('invites:page.title')}
        </h1>
        {body}
      </main>
      <SiteFooter />
    </div>
  )
}

function InviteCard({ invite, inviteToken, onChanged }: { invite: PublicInvite; inviteToken: string; onChanged: () => void }) {
  const { t } = useTranslation(['invites', 'common', 'profile'])
  const auth = useAuth()
  const { login } = useLogin()
  const [joined, setJoined] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState<'account' | 'guest' | null>(null)
  const [error, setError] = useState<string | null>(null)
  // a refusal of the automatic join (already in, own booking, full) is news, not a mistake
  const [notice, setNotice] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [age, setAge] = useState('')

  const tz = invite.timezone
  const [today] = useState(() => localDate(new Date().toISOString(), tz))
  const day = formatBookingDay(invite.starts_at, tz, today)
  const time = formatTime(invite.starts_at, tz)
  const facilityPath = `/${invite.tenant_slug}/${invite.facility_slug}`
  const facility = invite.facility

  const join = async (kind: 'account' | 'guest', automatic = false) => {
    setSubmitting(kind)
    setError(null)
    setNotice(null)
    try {
      if (kind === 'account') {
        await api.invites.join(auth.user!.access_token, inviteToken)
        setJoined(t('invites:page.joined', { facility: invite.facility_name, day, time }))
      } else {
        await api.invites.joinAsGuest(inviteToken, name, Number(age))
        setJoined(t('invites:page.guestJoined', { name: name.trim(), facility: invite.facility_name, day, time }))
        setName('')
        setAge('')
      }
      onChanged()
    } catch (e) {
      const message = e instanceof Error ? e.message : t('invites:page.error')
      if (automatic && e instanceof ApiError && e.status === 409) setNotice(message)
      else setError(message)
      if (e instanceof ApiError && e.status === 409) onChanged() // full or closed meanwhile
    } finally {
      setSubmitting(null)
    }
  }

  // signed in, the link joins straight away: once, also on coming back from signing in
  const autoJoined = useRef(false)
  // joining with an account needs what booking needs: a birth date and a verified email
  const { me } = useMe()
  const notReady = auth.isAuthenticated && me !== null && (!me.profile_complete || !me.email_verified)
  // the owner opening their own link goes to their bookings: no joining, and no adding guests in other people's names
  const token = auth.user?.access_token
  const mine = useAsync(token ? () => api.myBookings(token) : null, `invite-own:${inviteToken}:${token ? 'in' : 'out'}`)
  const ownBooking = mine.data?.upcoming.some(
    (b) =>
      b.role === 'owner' &&
      b.status === 'confirmed' &&
      b.tenant_slug === invite.tenant_slug &&
      b.facility_slug === invite.facility_slug &&
      Date.parse(b.starts_at) === Date.parse(invite.starts_at),
  )
  const checkingOwn = Boolean(token) && !mine.data && !mine.error
  useEffect(() => {
    if (!auth.isAuthenticated || !me || notReady || checkingOwn || ownBooking || invite.state !== 'open' || autoJoined.current) return
    autoJoined.current = true
    void join('account', true)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- once per page, when the session is there
  }, [auth.isAuthenticated, me, notReady, checkingOwn, ownBooking, invite.state])

  if (ownBooking) return <Navigate to="/reservas" replace />

  return (
    <div className="mt-6 grid max-w-[640px] gap-6">
      <section className="overflow-hidden rounded-lg border bg-card">
        {facility.image_url && <img src={facility.image_url} alt="" className="aspect-[16/7] w-full object-cover" />}
        <div className="grid gap-5 p-5">
          <dl className="grid gap-3">
            <div>
              <dt className="sr-only">{t('invites:page.where')}</dt>
              <dd className="flex gap-3">
                <MapPin className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
                <div>
                  <Link to={facilityPath} className="font-medium underline-offset-4 hover:underline">
                    {invite.facility_name}
                  </Link>
                  <span className="text-muted-foreground"> · {[facility.sport, invite.tenant_name].filter(Boolean).join(' · ')}</span>
                </div>
              </dd>
            </div>
            <div>
              <dt className="sr-only">{t('invites:page.when')}</dt>
              <dd className="num flex gap-3">
                <CalendarClock className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
                <span>
                  {day}, {time}–{formatTime(invite.ends_at, tz)}
                </span>
              </dd>
            </div>
            <div>
              <dt className="sr-only">{t('invites:page.spots')}</dt>
              <dd className="flex gap-3">
                <Users className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
                <div className="min-w-0 flex-1">
                  {invite.state === 'closed' ? (
                    t('invites:page.closed')
                  ) : (
                    <SpotsMeter capacity={invite.capacity} taken={invite.capacity - invite.spots_left} />
                  )}
                </div>
              </dd>
            </div>
          </dl>
          <FacilityInfo facility={facility} headingLevel={2} />
        </div>
      </section>

      <div aria-live="polite" className="empty:hidden">
        {joined && (
          <Alert className="border-0 bg-pill-positive-bg text-pill-positive-ink">
            <CircleCheck aria-hidden />
            <AlertDescription className="text-pill-positive-ink">
              <p>{joined}</p>
              <p className="mt-2 flex flex-wrap gap-x-4">
                {auth.isAuthenticated && (
                  <Link to="/reservas" className="font-medium underline underline-offset-4">
                    {t('invites:page.seeMyBookings')}
                  </Link>
                )}
                <Link to={facilityPath} className="font-medium underline underline-offset-4">
                  {t('invites:page.facilityPage')}
                </Link>
              </p>
            </AlertDescription>
          </Alert>
        )}
        {notice && (
          <Alert>
            <AlertDescription>
              <p>{notice}</p>
              <p className="mt-2">
                <Link to="/reservas" className="font-medium underline underline-offset-4">
                  {t('invites:page.seeMyBookings')}
                </Link>
              </p>
            </AlertDescription>
          </Alert>
        )}
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </div>

      {invite.state === 'open' && (
        <>
          <section aria-labelledby="join-account" className="rounded-lg border bg-card p-5">
            <h2 id="join-account" className="text-lg font-semibold">
              {t('invites:page.withAccount')}
            </h2>
            <p className="mt-1 text-muted-foreground">{t('invites:page.withAccountHint')}</p>
            <div className="mt-4">
              {notReady && me ? (
                <div className="grid justify-items-start gap-2">
                  <p className="text-sm">{me.profile_complete ? t('profile:unverifiedBody') : t('profile:incompleteBody')}</p>
                  <Button asChild>
                    <Link to="/perfil">{me.profile_complete ? t('profile:unverifiedAction') : t('profile:incompleteAction')}</Link>
                  </Button>
                </div>
              ) : auth.isAuthenticated ? (
                <Button onClick={() => void join('account')} disabled={submitting !== null}>
                  {submitting === 'account' ? t('invites:page.joining') : t('invites:page.withAccount')}
                </Button>
              ) : (
                <Button onClick={login} disabled={auth.isLoading}>
                  {t('invites:page.signInToJoin')}
                </Button>
              )}
            </div>
          </section>

          <section aria-labelledby="join-guest" className="rounded-lg border bg-card p-5">
            <h2 id="join-guest" className="text-lg font-semibold">
              {t('invites:page.asGuest')}
            </h2>
            <p className="mt-1 text-muted-foreground">{t('invites:page.asGuestHint')}</p>
            <form
              className="mt-4 grid gap-4 sm:grid-cols-[minmax(0,1fr)_120px]"
              onSubmit={(event) => {
                event.preventDefault()
                void join('guest')
              }}
            >
              <div className="grid gap-2">
                <Label htmlFor="guest-name">{t('invites:page.name')}</Label>
                <Input id="guest-name" required maxLength={80} autoComplete="name" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="guest-age">{t('invites:page.age')}</Label>
                <Input
                  id="guest-age"
                  required
                  type="number"
                  inputMode="numeric"
                  min={facility.min_age ?? 1}
                  max={120}
                  value={age}
                  onChange={(e) => setAge(e.target.value)}
                  className="num"
                />
              </div>
              <div className="sm:col-span-2">
                <Button type="submit" variant="outline" disabled={submitting !== null}>
                  {submitting === 'guest' ? t('invites:page.joining') : t('invites:page.joinAsGuest')}
                </Button>
              </div>
            </form>
          </section>
        </>
      )}
    </div>
  )
}
