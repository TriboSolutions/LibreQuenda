import type { GeoPoint } from '@/lib/api'

const EARTH_RADIUS_KM = 6371

/** Straight-line (great-circle) distance in km: enough to tell near from far, not a route. */
export function distanceKm(a: GeoPoint, b: GeoPoint) {
  const rad = (degrees: number) => (degrees * Math.PI) / 180
  const dLat = rad(b.latitude - a.latitude)
  const dLon = rad(b.longitude - a.longitude)
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(rad(a.latitude)) * Math.cos(rad(b.latitude)) * Math.sin(dLon / 2) ** 2
  return 2 * EARTH_RADIUS_KM * Math.asin(Math.sqrt(h))
}

/** The part of the world a map shows. */
export type MapView = { south: number; west: number; north: number; east: number }

export function inView(view: MapView, point: GeoPoint) {
  return point.latitude >= view.south && point.latitude <= view.north && point.longitude >= view.west && point.longitude <= view.east
}
