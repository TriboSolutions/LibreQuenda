// Runtime config written by the zitadel-setup container and served by Caddy at /config.json
import i18n from './i18n'

export type RuntimeConfig = {
  authority: string
  clientId: string
  projectId: string
  scope: string
}

export async function loadConfig(): Promise<RuntimeConfig> {
  const res = await fetch('/config.json', { cache: 'no-store' })
  if (!res.ok) {
    throw new Error(i18n.t('common:configError', { status: res.status }))
  }
  return res.json()
}
