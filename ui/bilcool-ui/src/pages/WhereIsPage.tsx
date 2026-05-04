import { useEffect, useState, useRef } from 'react'
import { MapContainer, TileLayer, Marker, Popup, Polyline, useMap } from 'react-leaflet'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { listFinishedBookings } from '../api/events'
import { format } from 'date-fns'
import { Navigation, MapPin, Locate, AlertCircle } from 'lucide-react'

// Fix Leaflet default marker icons broken by bundlers
delete (L.Icon.Default.prototype as unknown as Record<string, unknown>)._getIconUrl
L.Icon.Default.mergeOptions({
  iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
  iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon.png',
  shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png',
})

const deviceIcon = new L.Icon({
  iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon.png',
  iconRetinaUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
  shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png',
  iconSize: [25, 41],
  iconAnchor: [12, 41],
  popupAnchor: [1, -34],
})

const userIcon = new L.DivIcon({
  html: `<div style="width:16px;height:16px;border-radius:50%;background:#3b82f6;border:3px solid white;box-shadow:0 0 0 2px #3b82f6;"></div>`,
  className: '',
  iconSize: [16, 16],
  iconAnchor: [8, 8],
})

function haversineKm(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const R = 6371
  const dLat = ((lat2 - lat1) * Math.PI) / 180
  const dLon = ((lon2 - lon1) * Math.PI) / 180
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos((lat1 * Math.PI) / 180) * Math.cos((lat2 * Math.PI) / 180) * Math.sin(dLon / 2) ** 2
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
}

function FitBounds({ positions }: { positions: [number, number][] }) {
  const map = useMap()
  const fitted = useRef(false)
  useEffect(() => {
    if (!fitted.current && positions.length >= 2) {
      map.fitBounds(L.latLngBounds(positions), { padding: [60, 60] })
      fitted.current = true
    } else if (!fitted.current && positions.length === 1) {
      map.setView(positions[0], 14)
      fitted.current = true
    }
  }, [map, positions])
  return null
}

export default function WhereIsPage() {
  const { t } = useTranslation('where_is')
  const [userPos, setUserPos] = useState<{ lat: number; lon: number } | null>(null)
  const [geoError, setGeoError] = useState<string | null>(null)
  const [locating, setLocating] = useState(false)

  const { data: bookings, isLoading, isError } = useQuery({
    queryKey: ['finished-bookings-where-is'],
    queryFn: () => listFinishedBookings({}),
    staleTime: 60_000,
  })

  const lastKnown = bookings?.find((b) => b.position != null) ?? null

  function locate() {
    if (!navigator.geolocation) {
      setGeoError(t('geo_not_supported'))
      return
    }
    setLocating(true)
    setGeoError(null)
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setUserPos({ lat: pos.coords.latitude, lon: pos.coords.longitude })
        setLocating(false)
      },
      (err) => {
        setGeoError(err.message)
        setLocating(false)
      },
      { enableHighAccuracy: true, timeout: 10000 },
    )
  }

  function openNavigation() {
    if (!lastKnown?.position) return
    const { lat, lon } = lastKnown.position
    const origin = userPos ? `${userPos.lat},${userPos.lon}` : ''
    const url = origin
      ? `https://www.google.com/maps/dir/?api=1&origin=${origin}&destination=${lat},${lon}&travelmode=walking`
      : `https://www.google.com/maps/search/?api=1&query=${lat},${lon}`
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  const mapPositions: [number, number][] = []
  if (lastKnown?.position) mapPositions.push([lastKnown.position.lat, lastKnown.position.lon])
  if (userPos) mapPositions.push([userPos.lat, userPos.lon])

  const distanceKm =
    userPos && lastKnown?.position
      ? haversineKm(userPos.lat, userPos.lon, lastKnown.position.lat, lastKnown.position.lon)
      : null

  const defaultCenter: [number, number] = lastKnown?.position
    ? [lastKnown.position.lat, lastKnown.position.lon]
    : [59.3293, 18.0686] // Stockholm fallback

  return (
    <div className="flex flex-col h-full gap-4 p-4">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div>
          <h1 className="text-xl font-semibold">{t('title')}</h1>
          {lastKnown && (
            <p className="text-sm text-muted-foreground">
              {t('last_seen', {
                date: format(new Date(lastKnown.end_date), 'PPp'),
              })}
            </p>
          )}
        </div>
        <div className="flex gap-2 flex-wrap">
          <button
            onClick={locate}
            disabled={locating}
            className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium bg-secondary text-secondary-foreground hover:bg-secondary/80 disabled:opacity-50 transition-colors"
          >
            <Locate className="h-4 w-4" />
            {locating ? t('locating') : t('find_me')}
          </button>
          {lastKnown?.position && (
            <button
              onClick={openNavigation}
              className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <Navigation className="h-4 w-4" />
              {t('navigate')}
            </button>
          )}
        </div>
      </div>

      {geoError && (
        <div className="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {geoError}
        </div>
      )}

      {distanceKm !== null && (
        <div className="flex items-center gap-2 rounded-lg border bg-muted px-3 py-2 text-sm">
          <MapPin className="h-4 w-4 text-muted-foreground shrink-0" />
          {distanceKm < 1
            ? t('distance_meters', { meters: Math.round(distanceKm * 1000) })
            : t('distance_km', { km: distanceKm.toFixed(1) })}
        </div>
      )}

      <div className="flex-1 min-h-[400px] rounded-xl overflow-hidden border">
        {isLoading && (
          <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
            {t('loading')}
          </div>
        )}
        {isError && (
          <div className="flex h-full items-center justify-center text-destructive text-sm">
            {t('error')}
          </div>
        )}
        {!isLoading && !isError && !lastKnown?.position && (
          <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
            {t('no_position')}
          </div>
        )}
        {!isLoading && !isError && (lastKnown?.position || userPos) && (
          <MapContainer
            center={defaultCenter}
            zoom={13}
            style={{ height: '100%', width: '100%' }}
            scrollWheelZoom
          >
            <TileLayer
              attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
              url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
            />
            <FitBounds positions={mapPositions} />
            {lastKnown?.position && (
              <Marker position={[lastKnown.position.lat, lastKnown.position.lon]} icon={deviceIcon}>
                <Popup>
                  <div className="text-sm space-y-1">
                    <p className="font-semibold">{t('device_position')}</p>
                    <p className="text-muted-foreground">
                      {format(new Date(lastKnown.end_date), 'PPp')}
                    </p>
                    <p>
                      {lastKnown.position.lat.toFixed(5)}, {lastKnown.position.lon.toFixed(5)}
                    </p>
                  </div>
                </Popup>
              </Marker>
            )}
            {userPos && (
              <Marker position={[userPos.lat, userPos.lon]} icon={userIcon}>
                <Popup>
                  <p className="text-sm font-semibold">{t('your_position')}</p>
                </Popup>
              </Marker>
            )}
            {userPos && lastKnown?.position && (
              <Polyline
                positions={[
                  [userPos.lat, userPos.lon],
                  [lastKnown.position.lat, lastKnown.position.lon],
                ]}
                pathOptions={{ color: '#3b82f6', weight: 2, dashArray: '6 6' }}
              />
            )}
          </MapContainer>
        )}
      </div>
    </div>
  )
}
