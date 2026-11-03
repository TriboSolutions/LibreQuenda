import type { EntranceInput } from '@/lib/api'

/** The address written on the facility's NFC tag: the same entrance page, with the tag's own token. */
export function nfcUrl(slug: string, facilitySlug: string, nfcToken: string) {
  return `${window.location.origin}/${slug}/${facilitySlug}/entrada?n=${nfcToken}`
}

/** The address the printed QR opens: the facility's entrance page, with the token that makes it work. */
export function entranceUrl(slug: string, facilitySlug: string, qrToken: string) {
  return `${window.location.origin}/${slug}/${facilitySlug}/entrada?c=${qrToken}`
}

export type Position = Pick<EntranceInput, 'latitude' | 'longitude' | 'accuracy'>

/** The phone's location, asked for each check-in and never kept. Rejects with a key of entrance:location. */
export function locate(): Promise<Position> {
  return new Promise((resolve, reject) => {
    if (!('geolocation' in navigator)) {
      reject(new Error('unsupported'))
      return
    }
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => resolve({ latitude: coords.latitude, longitude: coords.longitude, accuracy: coords.accuracy }),
      (error) => reject(new Error(error.code === error.PERMISSION_DENIED ? 'denied' : error.code === error.TIMEOUT ? 'timeout' : 'unavailable')),
      { enableHighAccuracy: true, timeout: 20000, maximumAge: 0 },
    )
  })
}

/** The results that leave the person in. */
export const GRANTED = ['checked_in', 'already_checked_in', 'booked']

/** The council, facility and token in a printed QR's address (see entranceUrl); null for any other QR. */
export function parseEntranceUrl(text: string): { slug: string; facility: string; qrToken: string } | null {
  let url: URL
  try {
    url = new URL(text)
  } catch {
    return null
  }
  const match = /^\/([^/]+)\/([^/]+)\/entrada\/?$/.exec(url.pathname)
  const qrToken = url.searchParams.get('c')
  if (!match || !qrToken) return null
  return { slug: decodeURIComponent(match[1]), facility: decodeURIComponent(match[2]), qrToken }
}
