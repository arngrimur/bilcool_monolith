import { apiFetch } from './client';
import type {
  BookingResponse,
  UpdateBookingRequest,
  EndBookingRequest,
  PauseBookingRequest,
  PauseBookingResponse,
  TrackPoint,
  BookingRoute,
} from '../types/api';

const BASE = '/api/v1';

export const listBookings = () =>
  apiFetch<BookingResponse[]>(`${BASE}/bookings`);

export const getBooking = (id: string) =>
  apiFetch<BookingResponse>(`${BASE}/bookings/${id}`);

export const upsertBooking = (body: UpdateBookingRequest) =>
  apiFetch<void>(`${BASE}/bookings`, { method: 'PUT', body: JSON.stringify(body) });

export const deleteBooking = (id: string) =>
  apiFetch<void>(`${BASE}/bookings/${id}`, { method: 'DELETE' });

export const endBooking = (id: string, body: EndBookingRequest) =>
  apiFetch<void>(`${BASE}/bookings/${id}/end`, { method: 'POST', body: JSON.stringify(body) });

export const pauseBooking = (id: string, body: PauseBookingRequest) =>
  apiFetch<void>(`${BASE}/bookings/${id}/pause`, { method: 'POST', body: JSON.stringify(body) });

export const resumeBooking = (id: string) =>
  apiFetch<PauseBookingResponse>(`${BASE}/bookings/${id}/resume`, { method: 'POST' });

export const addTrackPoints = (id: string, points: TrackPoint[]) =>
  apiFetch<void>(`${BASE}/bookings/${id}/track`, {
    method: 'POST',
    body: JSON.stringify({ points }),
  });

export const getTrackPoints = (id: string) =>
  apiFetch<TrackPoint[]>(`${BASE}/bookings/${id}/track`);

// Falls back to the plain track points when the route endpoint is unavailable, so the map
// still works while the UI and the bookings service are being deployed out of step.
export const getRoute = async (id: string): Promise<BookingRoute> => {
  try {
    return await apiFetch<BookingRoute>(`${BASE}/bookings/${id}/route`);
  } catch (err) {
    if ((err as { status?: number }).status === 401) throw err;
    return { points: await getTrackPoints(id), route: [] };
  }
};
