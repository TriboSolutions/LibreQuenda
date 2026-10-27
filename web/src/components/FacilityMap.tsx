import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { useEffect, useRef } from 'react'
import { DEFAULT_CENTER, TILE_ATTRIBUTION, TILE_URL } from '@/lib/maps'
import { cn } from '@/lib/utils'

type Props = {
  latitude: number | null
  longitude: number | null
  label: string // accessible name of the map
  editable?: boolean // click to place the pin, drag to move it
  onChange?: (latitude: number, longitude: number) => void
  className?: string
}

// The pin is drawn in CSS (globals.css .lq-pin): no image assets to bundle
const pinIcon = L.divIcon({ className: 'lq-pin', html: '<span></span>', iconSize: [28, 28], iconAnchor: [14, 28] })

/** OpenStreetMap with one pin, read-only or editable. Leaflet (BSD-2) is driven directly. */
export function FacilityMap({ latitude, longitude, label, editable = false, onChange, className }: Props) {
  const element = useRef<HTMLDivElement>(null)
  const map = useRef<L.Map | null>(null)
  const marker = useRef<L.Marker | null>(null)
  const onChangeRef = useRef(onChange)
  useEffect(() => {
    onChangeRef.current = onChange
  }, [onChange])

  // the map, once
  useEffect(() => {
    if (!element.current) return
    const instance = L.map(element.current, { scrollWheelZoom: false }).setView(DEFAULT_CENTER, 8)
    L.tileLayer(TILE_URL, { maxZoom: 19, attribution: TILE_ATTRIBUTION }).addTo(instance)
    if (editable) {
      instance.on('click', (e: L.LeafletMouseEvent) => onChangeRef.current?.(e.latlng.lat, e.latlng.lng))
    }
    map.current = instance
    return () => {
      instance.remove()
      map.current = null
      marker.current = null
    }
  }, [editable])

  // the pin follows the coordinates
  useEffect(() => {
    const instance = map.current
    if (!instance) return
    if (latitude === null || longitude === null) {
      marker.current?.remove()
      marker.current = null
      return
    }
    const position: L.LatLngExpression = [latitude, longitude]
    if (!marker.current) {
      marker.current = L.marker(position, { icon: pinIcon, draggable: editable, keyboard: editable, alt: label }).addTo(instance)
      marker.current.on('dragend', () => {
        const at = marker.current?.getLatLng()
        if (at) onChangeRef.current?.(at.lat, at.lng)
      })
    } else {
      marker.current.setLatLng(position)
    }
    instance.setView(position, Math.max(instance.getZoom(), 16))
  }, [latitude, longitude, editable, label])

  return <div ref={element} role="region" aria-label={label} className={cn('isolate z-0 h-64 rounded-md border', className)} />
}
