export interface CreateUserRequest {
  username: string;
  email: string;
}

export interface ChangeUserRoleRequest {
  role: 'admin' | 'user';
}

export interface UserResponse {
  user_ref: string;
  username: string;
  email: string;
  role: 'admin' | 'user';
}

export interface DeletedUserResponse {
  user_ref: string;
  username: string;
  email: string;
  role: 'admin' | 'user';
  deleted_at: string;
}

export interface LoginBeginRequest {
  email: string;
}

export interface ResetLoginRequest {
  email: string;
}

export type LoginNextStep = 'verify_token' | 'passkey_assertion';

export interface LoginBeginResponse {
  next_step: LoginNextStep;
  session_id?: string;
  options?: unknown;
}

export interface VerifyTokenRequest {
  email: string;
  token: string;
}

export interface VerifyTokenResponse {
  session_id: string;
  options: unknown;
}

export interface LoginCompleteRequest {
  session_id: string;
  credential: unknown;
}

export interface LoginCompleteResponse {
  token: string;
}

export interface BookingResponse {
  user_ref: string;
  booking_reference: string;
  start_date: string;
  end_date: string;
  distance?: { start_distance: number; end_distance: number };
}

export interface UpdateBookingRequest {
  user_ref: string;
  booking_reference: string;
  start_date: string;
  end_date: string;
}

export interface EndBookingRequest {
  start_distance: number;
  end_distance: number;
  position?: { lat: number; lon: number };
}

export interface PauseBookingRequest {
  lat: number;
  lon: number;
}

export interface PauseBookingResponse {
  position: { lat: number; lon: number };
}

export interface EventResponse {
  event_id: string;
  event_type: string;
  correlation_id: string;
  producer: string;
  emitted_at: string;
  payload: unknown;
  received_at: string;
}

export interface EventQueryParams {
  event_id?: string;
  producer?: string;
  event_type?: string;
  emitted_at?: string;
  emitted_at_gte?: string;
  emitted_at_lte?: string;
  limit?: number;
  offset?: number;
  order_by?: string;
  order_direction?: 'asc' | 'desc';
}

export interface FinishedBooking {
  booking_reference: string
  user_ref: string
  start_date: string
  end_date: string
  distance_meters: number
  position?: { lat: number; lon: number }
}

export interface FinishedBookingParams {
  year?: number
  month?: number
  user_ref?: string
}

export interface CompletedBookingPayload {
  booking: BookingResponse;
  distance: {
    start_distance: number;
    end_distance: number;
  };
}
