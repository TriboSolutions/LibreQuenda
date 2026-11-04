import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import type { FacilityCard, GeoPoint } from '@/lib/api'
import type { MapView } from '@/lib/geo'
import { DEFAULT_CENTER, TILE_ATTRIBUTION, TILE_URL } from '@/lib/maps'
import { cn } from '@/lib/utils'

type Props = {
  facilities: FacilityCard[] // the ones the filters left; those without a pin are not drawn
  highlighted: string | null // a facility id, pointed at in the list
  me: GeoPoint | null // the resident, once "near me" is on
  label: string // accessible name of the map
  fitKey: number // bump it to frame every facility again ("see all")
  onViewChange: (view: MapView) => void // after every pan or zoom: the list keeps what is in view
  className?: string
}

// Zoomed out, each council is one bubble; from this zoom on, every facility has its own pin,
// and from the next one on, its name beside it
const FACILITY_ZOOM = 12
const LABEL_ZOOM = 14

const pinIcon = (active: boolean) =>
  L.divIcon({ className: cn('lq-pin', active && 'is-active'), html: '<span></span>', iconSize: [28, 28], iconAnchor: [14, 28], popupAnchor: [0, -26] })
const meIcon = L.divIcon({ className: 'lq-me', html: '<span></span>', iconSize: [16, 16], iconAnchor: [8, 8] })

type Pinned = FacilityCard & { latitude: number; longitude: number }
const pinned = (f: FacilityCard): f is Pinned => f.latitude !== null && f.longitude !== null

/**
 * Pins close together would stack their names into an unreadable pile: a name that
 * would cover one already shown is hidden (its pin still names itself on hover and
 * to screen readers). Zooming in spreads the pins out and brings the names back.
 * Only the names of the pins just drawn: the ones they replace linger while Leaflet
 * fades them out, and would otherwise hide every new name on top of them.
 */
function declutter(labels: HTMLElement[]) {
  const shown: DOMRect[] = []
  for (const label of labels) {
    label.style.visibility = ''
    const box = label.getBoundingClientRect()
    const covers = shown.some((s) => box.left < s.right && box.right > s.left && box.top < s.bottom && box.bottom > s.top)
    if (covers) label.style.visibility = 'hidden'
    else shown.push(box)
  }
}

/** A link inside a popup that moves within the app instead of reloading the page. */
function popupLink(text: string, path: string, go: (path: string) => void, className = '') {
  const a = document.createElement('a')
  a.href = path
  a.textContent = text
  a.className = cn('underline-offset-4 hover:underline', className)
  a.addEventListener('click', (event) => {
    event.preventDefault()
    go(path)
  })
  return a
}

/**
 * The home page's map (OpenStreetMap, Leaflet driven directly): one bubble per council
 * when zoomed out, its facilities' pins when zoomed in. A bubble's popup leads to the
 * council's page or zooms into it; a pin's popup leads to the facility.
 */
export function HomeMap({ facilities, highlighted, me, label, fitKey, onViewChange, className }: Props) {
  const { t } = useTranslation('councils')
  const navigate = useNavigate()
  const element = useRef<HTMLDivElement>(null)
  const map = useRef<L.Map | null>(null)
  const layer = useRef<L.LayerGroup | null>(null)
  const pins = useRef(new Map<string, L.Marker>())
  const meMarker = useRef<L.Marker | null>(null)
  const draw = useRef<() => void>(() => {})
  const onViewChangeRef = useRef(onViewChange)
  useEffect(() => {
    onViewChangeRef.current = onViewChange
  }, [onViewChange])

  // the map, once
  useEffect(() => {
    if (!element.current) return
    const instance = L.map(element.current, { scrollWheelZoom: true }).setView(DEFAULT_CENTER, 8)
    L.tileLayer(TILE_URL, { maxZoom: 19, attribution: TILE_ATTRIBUTION }).addTo(instance)
    layer.current = L.layerGroup().addTo(instance)
    instance.on('zoomend', () => draw.current())
    instance.on('moveend', () => {
      const b = instance.getBounds()
      onViewChangeRef.current({ south: b.getSouth(), west: b.getWest(), north: b.getNorth(), east: b.getEast() })
    })
    map.current = instance
    // the box changes size with the window and the filter bar's wrapping: Leaflet has to be told
    const resized = new ResizeObserver(() => instance.invalidateSize())
    resized.observe(element.current)
    return () => {
      resized.disconnect()
      instance.remove()
      map.current = null
      layer.current = null
      meMarker.current = null
    }
  }, [])

  // the markers, for the facilities shown and the zoom
  const points = facilities.filter(pinned)
  useEffect(() => {
    draw.current = () => {
      const instance = map.current
      const group = layer.current
      if (!instance || !group) return
      group.clearLayers()
      pins.current.clear()
      const go = (path: string) => navigate(path)

      if (instance.getZoom() >= FACILITY_ZOOM) {
        for (const f of points) {
          const popup = document.createElement('div')
          popup.className = 'grid gap-0.5'
          popup.append(popupLink(f.name, `/${f.council.slug}/${f.slug}`, go, 'font-semibold'))
          popup.append(popupLink(f.council.name, `/${f.council.slug}`, go, 'text-muted-foreground'))
          const free = document.createElement('span')
          free.textContent = t('card.freeTodayLong', { count: f.free_today })
          popup.append(free)
          const marker = L.marker([f.latitude, f.longitude], { icon: pinIcon(f.id === highlighted), alt: f.name, title: f.name })
            .bindPopup(popup)
            .addTo(group)
          marker.getElement()?.setAttribute('aria-label', f.name) // a div icon has no <img> for the alt (WCAG 4.1.2)
          if (instance.getZoom() >= LABEL_ZOOM) {
            // close enough to tell the pins apart: their names, always on; the pin already carries the name for screen readers
            const name = document.createElement('span')
            name.textContent = f.name // text, not HTML: a facility name is data
            name.setAttribute('aria-hidden', 'true')
            marker.bindTooltip(name, { permanent: true, direction: 'top', offset: [0, -26], className: 'lq-label' })
          }
          pins.current.set(f.id, marker)
        }
        if (instance.getZoom() >= LABEL_ZOOM) {
          const labels = [...pins.current.values()].flatMap((m) => m.getTooltip()?.getElement() ?? [])
          requestAnimationFrame(() => declutter(labels))
        }
        return
      }

      const byCouncil = new Map<string, Pinned[]>()
      for (const f of points) byCouncil.set(f.council.slug, [...(byCouncil.get(f.council.slug) ?? []), f])
      for (const list of byCouncil.values()) {
        const council = list[0].council
        const bounds = L.latLngBounds(list.map((f) => [f.latitude, f.longitude]))
        const name = t('map.council', { name: council.name, count: list.length })
        const icon = L.divIcon({ className: '', html: `<span class="lq-council"><b>${list.length}</b></span>`, iconSize: [0, 0] })
        const marker = L.marker(bounds.getCenter(), { icon, title: name }).addTo(group)
        // the name as text, not HTML: a council name is data
        const bubble = marker.getElement()?.querySelector('.lq-council')
        bubble?.append(council.name)
        marker.getElement()?.setAttribute('aria-label', name)

        const popup = document.createElement('div')
        popup.className = 'grid gap-2'
        popup.append(popupLink(t('map.openCouncil', { name: council.name }), `/${council.slug}`, go, 'font-semibold'))
        const zoom = document.createElement('button')
        zoom.type = 'button'
        zoom.textContent = t('map.zoomIn', { count: list.length })
        zoom.className = 'justify-self-start text-primary underline-offset-4 hover:underline'
        zoom.addEventListener('click', () => instance.fitBounds(bounds.pad(0.3), { maxZoom: 15 }))
        popup.append(zoom)
        marker.bindPopup(popup)
      }
    }
  })

  // redraw when the filters change; frame them all when the set itself changes, or on "see all"
  const shownKey = `${fitKey}:${points.map((f) => f.id).join(',')}`
  useEffect(() => {
    const instance = map.current
    if (!instance) return
    if (points.length > 0) instance.fitBounds(L.latLngBounds(points.map((f) => [f.latitude, f.longitude])).pad(0.2), { maxZoom: 14 })
    draw.current()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- shownKey stands for points
  }, [shownKey])

  // the pin pointed at in the list
  useEffect(() => {
    for (const [id, marker] of pins.current) {
      marker.setIcon(pinIcon(id === highlighted))
      marker.getElement()?.setAttribute('aria-label', marker.options.alt ?? '')
    }
  }, [highlighted])

  // the resident's dot
  useEffect(() => {
    const instance = map.current
    if (!instance) return
    meMarker.current?.remove()
    meMarker.current = me ? L.marker([me.latitude, me.longitude], { icon: meIcon, interactive: false, keyboard: false }).addTo(instance) : null
  }, [me])

  return <div ref={element} role="region" aria-label={label} className={cn('isolate z-0 rounded-lg border', className)} />
}
