import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor, cleanup } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import BookingRouteDialog from './BookingRouteDialog'
import { getTrackPoints } from '../../api/bookings'
import type { TrackPoint } from '../../types/api'

vi.mock('../../api/bookings', () => ({
  getTrackPoints: vi.fn(),
}))

// Leaflet needs real layout, which jsdom lacks, so the map components are replaced with stubs.
vi.mock('react-leaflet', () => ({
  MapContainer: ({ children }: { children: React.ReactNode }) => <div data-testid="map">{children}</div>,
  TileLayer: () => null,
  Polyline: ({ positions }: { positions: [number, number][] }) => (
    <div data-testid="polyline" data-count={positions.length} />
  ),
  CircleMarker: ({ children }: { children: React.ReactNode }) => <div data-testid="marker">{children}</div>,
  Tooltip: ({ children }: { children: React.ReactNode }) => <span>{children}</span>,
  useMap: () => ({ fitBounds: vi.fn(), setView: vi.fn() }),
}))

const points: TrackPoint[] = [
  { lat: 59.33, lon: 18.06, recorded_at: '2026-05-01T10:00:00Z' },
  { lat: 59.34, lon: 18.07, recorded_at: '2026-05-01T10:02:00Z' },
  { lat: 59.35, lon: 18.08, recorded_at: '2026-05-01T10:04:00Z' },
]

function renderDialog(open = true) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={queryClient}>
      <BookingRouteDialog bookingReference="ref-1" open={open} onOpenChange={vi.fn()} />
    </QueryClientProvider>
  )
}

describe('BookingRouteDialog', () => {
  beforeEach(() => {
    vi.mocked(getTrackPoints).mockReset()
  })
  afterEach(cleanup)

  it('draws the recorded route with start and end markers', async () => {
    vi.mocked(getTrackPoints).mockResolvedValue(points)
    renderDialog()

    expect(await screen.findByTestId('polyline')).toHaveAttribute('data-count', '3')
    expect(screen.getAllByTestId('marker')).toHaveLength(2)
    expect(getTrackPoints).toHaveBeenCalledWith('ref-1')
  })

  it('draws a single marker when only one point was recorded', async () => {
    vi.mocked(getTrackPoints).mockResolvedValue([points[0]])
    renderDialog()

    await screen.findByTestId('map')
    expect(screen.getAllByTestId('marker')).toHaveLength(1)
  })

  it('shows an empty message when no route was recorded', async () => {
    vi.mocked(getTrackPoints).mockResolvedValue([])
    renderDialog()

    expect(await screen.findByText('Ingen rutt registrerades för den här bokningen')).toBeInTheDocument()
    expect(screen.queryByTestId('map')).not.toBeInTheDocument()
  })

  it('shows an error message when loading fails', async () => {
    vi.mocked(getTrackPoints).mockRejectedValue(new Error('boom'))
    renderDialog()

    expect(await screen.findByText('Misslyckades att hämta rutten')).toBeInTheDocument()
  })

  it('does not fetch while closed', async () => {
    renderDialog(false)

    await waitFor(() => expect(getTrackPoints).not.toHaveBeenCalled())
  })
})
