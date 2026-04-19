import { apiFetch } from './client';
import type { EventQueryParams, EventResponse, FinishedBooking, FinishedBookingParams } from '../types/api';

const BASE = '/api/events/api/v1';
const JOURNAL_BASE = '/api/journal/api/v1';

export const listEvents = (params: EventQueryParams) => {
  const qs = new URLSearchParams(
    Object.entries(params)
      .filter(([, v]) => v !== undefined)
      .map(([k, v]) => [k, String(v)])
  ).toString();
  return apiFetch<EventResponse[]>(`${BASE}/events${qs ? `?${qs}` : ''}`);
};

export const listFinishedBookings = (params: FinishedBookingParams) => {
  const qs = new URLSearchParams(
    Object.entries(params)
      .filter(([, v]) => v !== undefined)
      .map(([k, v]) => [k, String(v)])
  ).toString();
  return apiFetch<FinishedBooking[]>(`${JOURNAL_BASE}/bookings${qs ? `?${qs}` : ''}`);
};
