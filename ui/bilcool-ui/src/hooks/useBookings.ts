import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { listBookings, upsertBooking, deleteBooking, endBooking, pauseBooking, resumeBooking, addTrackPoints } from '../api/bookings'
import type { UpdateBookingRequest, EndBookingRequest, PauseBookingRequest, TrackPoint } from '../types/api'

export function useBookings() {
  return useQuery({
    queryKey: ['bookings'],
    queryFn: listBookings,
    staleTime: 30_000,
  })
}

export function useUpsertBooking() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: UpdateBookingRequest) => upsertBooking(body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bookings'] })
    },
  })
}

export function useDeleteBooking() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteBooking(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bookings'] })
    },
  })
}

export function useEndBooking() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: EndBookingRequest }) =>
      endBooking(id, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bookings'] })
    },
  })
}

export function usePauseBooking() {
  return useMutation({
    mutationFn: ({ id, body }: { id: string; body: PauseBookingRequest }) =>
      pauseBooking(id, body),
  })
}

export function useResumeBooking() {
  return useMutation({
    mutationFn: (id: string) => resumeBooking(id),
  })
}

export function useAddTrackPoints() {
  return useMutation({
    mutationFn: ({ id, points }: { id: string; points: TrackPoint[] }) =>
      addTrackPoints(id, points),
  })
}
