// OpenStreetMap shows the map; directions open the visitor's own Google Maps
// through a plain link (no API key, nothing embedded, the visitor's choice).

export const TILE_URL = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
export const TILE_ATTRIBUTION = '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'

// Galicia, when there is no pin yet
export const DEFAULT_CENTER: [number, number] = [42.75, -8.0]

export function directionsUrl(latitude: number, longitude: number) {
  return `https://www.google.com/maps/dir/?api=1&destination=${latitude},${longitude}`
}
