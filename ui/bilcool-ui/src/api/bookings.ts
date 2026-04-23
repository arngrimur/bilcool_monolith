import { apiFetch } from './client';
import type {
  BookingResponse,
  UpdateBookingRequest,
  EndBookingRequest,
} from '../types/api';

const BASE = '/api/book/api/v1';

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
