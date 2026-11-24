// The voice line (voice/, behind Caddy at /voice/*): on only where the installation runs it.

export type VoiceConfig = {
  enabled: boolean
  maxCallSeconds: number
  // who else gets what during a call: the panel says it before the call starts
  thirdParties: { voice: string[]; text: string[] }
  // signed out, a call can still book with a name and a phone number
  guestBooking?: boolean
}

let loaded: Promise<VoiceConfig | null> | null = null

/** The voice line's runtime config, or null when it is off (no voice service answers). */
export function loadVoiceConfig(): Promise<VoiceConfig | null> {
  loaded ??= fetch('/voice/config.json', { cache: 'no-store' })
    .then((res) => (res.ok ? res.json() : null))
    .then((config: VoiceConfig | null) => (config?.enabled === true ? config : null))
    .catch(() => null)
  return loaded
}

export class VoiceError extends Error {}

/** A one-time ticket for the call's WebSocket: the access token goes in a header, never in its URL. */
export async function startVoiceSession(token: string | undefined): Promise<string> {
  const res = await fetch('/voice/session', {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new VoiceError(body.error === 'busy' ? 'busy' : 'unavailable')
  }
  return (await res.json()).ticket
}

export function voiceSocketUrl(): string {
  return `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/voice/ws`
}
