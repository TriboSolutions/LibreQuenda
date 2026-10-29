import type { EntranceInput } from '@/lib/api'

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
