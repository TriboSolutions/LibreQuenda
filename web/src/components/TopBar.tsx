import { useState } from 'react'
import { Building2, CalendarDays, LogOut, QrCode, ShieldCheck, UserRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from 'react-oidc-context'
import { Link, useNavigate } from 'react-router'
import { InvitationReminder } from '@/components/InvitationReminder'
import { LanguageMenu } from '@/components/LanguageMenu'
import { NotificationBell } from '@/components/NotificationBell'
import { PersonalQrDialog } from '@/components/PersonalQr'
import { ProfileReminder } from '@/components/ProfileReminder'
import { Mark, Wordmark } from '@/components/Mark'
import { ThemeMenu } from '@/components/ThemeMenu'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useCouncilAdmin } from '@/hooks/useCouncilAdmin'
import { useLogin } from '@/hooks/useLogin'
import { useMe } from '@/hooks/useMe'
import type { Tenant } from '@/lib/api'

type Props = {
  tenants: Tenant[]
  tenantSlug: string | null // the council the page is about (its name is in the page title); null when none
}

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('')
}

export function TopBar({ tenants, tenantSlug }: Props) {
  const { t } = useTranslation(['common', 'admin', 'profile', 'invites'])
  const auth = useAuth()
  const { me } = useMe()
  const managesCouncil = useCouncilAdmin(tenantSlug)
  const navigate = useNavigate()
  const { login, register } = useLogin()
  const tenant = tenants.find((t) => t.slug === tenantSlug)
  const profile = auth.user?.profile
  const name = profile?.name ?? profile?.email ?? ''
  const [qrOpen, setQrOpen] = useState(false)

  return (
    <>
    {/* WCAG 2.4.1: the first stop of the Tab key jumps over the bar */}
    <a
      href="#main"
      className="sr-only rounded-md bg-primary px-4 py-3 font-medium text-primary-foreground focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[100]"
    >
      {t('common:skipToContent')}
    </a>
    <header className="border-b bg-card">
      <div className="mx-auto flex h-14 max-w-[1152px] items-center gap-3 px-4 sm:px-6">
        <Link to="/" className="flex shrink-0 items-center gap-2 rounded-md font-semibold" aria-label={t('common:home')}>
          <Mark className="size-7" />
          <Wordmark className="hidden text-[17px] tracking-tight sm:inline" />
        </Link>

        <div className="ml-auto flex shrink-0 items-center gap-1">
          <LanguageMenu />
          <ThemeMenu />
          {auth.isAuthenticated && <NotificationBell />}
          {auth.isAuthenticated ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="gap-2 px-2" aria-label={t('common:account', { name })}>
                  {me?.avatar_url ? (
                    <img src={me.avatar_url} alt="" className="size-7 rounded-full object-cover" />
                  ) : (
                    // the initials are decoration (CSS content): the button is named after the person (WCAG 2.5.3)
                    <span
                      aria-hidden="true"
                      data-initials={initials(name)}
                      className="flex size-7 items-center justify-center rounded-full bg-accent text-xs font-semibold text-accent-foreground before:content-[attr(data-initials)]"
                    />
                  )}
                  <span className="hidden max-w-40 truncate sm:inline">{name}</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuLabel className="font-normal">
                  <span className="block truncate font-medium">{profile?.name}</span>
                  <span className="block truncate text-muted-foreground">{profile?.email}</span>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onSelect={() => setQrOpen(true)}>
                  <QrCode />
                  {t('profile:personalQr.menuLink')}
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => navigate('/reservas')}>
                  <CalendarDays />
                  {t('invites:myBookings.menuLink')}
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => navigate('/perfil')}>
                  <UserRound />
                  {t('profile:menuLink')}
                </DropdownMenuItem>
                {/* a superadministrator reaches every council from the platform administration */}
                {tenantSlug && managesCouncil && !me?.is_superadmin && (
                  <DropdownMenuItem onSelect={() => navigate(`/${tenantSlug}/admin`)}>
                    <Building2 />
                    {t('admin:councilMenuLink', { council: tenant?.name ?? tenantSlug })}
                  </DropdownMenuItem>
                )}
                {me?.is_superadmin && (
                  <DropdownMenuItem onSelect={() => navigate('/admin')}>
                    <ShieldCheck />
                    {t('admin:menuLink')}
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem onSelect={() => void auth.signoutRedirect()}>
                  <LogOut />
                  {t('common:signOut')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <>
              <Button variant="ghost" size="sm" className="hidden sm:inline-flex" onClick={register} disabled={auth.isLoading}>
                {t('common:createAccount')}
              </Button>
              <Button variant="outline" size="sm" onClick={login} disabled={auth.isLoading}>
                {t('common:signIn')}
              </Button>
            </>
          )}
        </div>
      </div>
    </header>
    {auth.isAuthenticated && <PersonalQrDialog open={qrOpen} onOpenChange={setQrOpen} />}
    <InvitationReminder />
    <ProfileReminder />
    </>
  )
}
