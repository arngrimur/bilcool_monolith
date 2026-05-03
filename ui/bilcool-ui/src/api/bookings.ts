import { apiFetch } from './client';
import type {
  BookingResponse,
  UpdateBookingRequest,
  EndBookingRequest,
  PauseBookingRequest,
  PauseBookingResponse,
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
