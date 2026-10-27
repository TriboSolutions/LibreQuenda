import { useMe } from '@/hooks/useMe'

/** Whether the signed-in user may manage this council: its administrator, or a superadministrator. */
export function useCouncilAdmin(slug: string | null) {
  const { me } = useMe()
  if (!me || !slug) return false
  return me.is_superadmin || me.roles.some((role) => role.tenant_slug === slug && role.role === 'admin')
}
