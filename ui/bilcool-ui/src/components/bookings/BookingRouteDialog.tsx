import { MapContainer, TileLayer, Polyline, CircleMarker, Tooltip, useMap } from 'react-leaflet'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '../ui/dialog'
import { getTrackPoints } from '../../api/bookings'

interface BookingRouteDialogProps {
  bookingReference: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

function FitRoute({ positions }: { positions: [number, number][] }) {
  const map = useMap()
  useEffect(() => {
    if (positions.length >= 2) {
      map.fitBounds(L.latLngBounds(positions), { padding: [30, 30] })
    } else if (positions.length === 1) {
      map.setView(positions[0], 15)
    }
  }, [map, positions])
  return null
}

export default function BookingRouteDialog({ bookingReference, open, onOpenChange }: BookingRouteDialogProps) {
  const { t } = useTranslation('bookings')

  const { data: points, isLoading, isError } = useQuery({
    queryKey: ['trackPoints', bookingReference],
    queryFn: () => getTrackPoints(bookingReference),
    enabled: open,
    staleTime: 60_000,
  })

  const positions: [number, number][] = (points ?? []).map((p) => [p.lat, p.lon])
  const first = positions[0]
  const last = positions[positions.length - 1]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t('route_title')}</DialogTitle>
        </DialogHeader>
        <div className="h-[400px] rounded-lg overflow-hidden border">
          {isLoading && (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              {t('route_loading')}
            </div>
          )}
          {isError && (
            <div className="flex h-full items-center justify-center text-sm text-destructive">
              {t('route_error')}
            </div>
          )}
          {!isLoading && !isError && positions.length === 0 && (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              {t('route_empty')}
            </div>
          )}
          {!isLoading && !isError && positions.length > 0 && (
            <MapContainer center={first} zoom={13} style={{ height: '100%', width: '100%' }} scrollWheelZoom>
              <TileLayer
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              />
              <FitRoute positions={positions} />
              <Polyline positions={positions} pathOptions={{ color: '#3b82f6', weight: 4 }} />
              <CircleMarker center={first} radius={8} pathOptions={{ color: '#16a34a', fillOpacity: 1 }}>
                <Tooltip>{t('route_start')}</Tooltip>
              </CircleMarker>
              {positions.length > 1 && (
                <CircleMarker center={last} radius={8} pathOptions={{ color: '#dc2626', fillOpacity: 1 }}>
                  <Tooltip>{t('route_end')}</Tooltip>
                </CircleMarker>
              )}
            </MapContainer>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
