# BilCool React Frontend — Build Prompt

This document is a self-contained specification for building the complete BilCool React web application. No clarifying questions are necessary. Every section is authoritative; where two sections appear to conflict, the more specific one wins.

---

## 1. Technology Stack

| Concern | Choice |
|---|---|
| Framework | React 19 (functional components and hooks only) |
| Build tool | Vite 6 (TypeScript template) |
| Language | TypeScript 5 (`"strict": true` in tsconfig) |
| UI components | shadcn/ui (New York style, Radix UI + Tailwind CSS v4, neutral base colour) |
| CSS | Tailwind CSS v4 |
| State management | Zustand 5 |
| Server state | TanStack Query v5 |
| Routing | React Router v7 (`createBrowserRouter`) |
| i18n | react-i18next + i18next |
| WebAuthn client | `@simplewebauthn/browser` v13 |
| Date/time | date-fns v4 |
| Calendar | `@fullcalendar/react` with `daygrid`, `timegrid`, `interaction` plugins |
| Forms | React Hook Form v7 + Zod |
| HTTP | native `fetch` wrapped in a typed API layer (no axios) |
| Icons | Lucide React |
| Testing | Vitest + React Testing Library |

### Install commands

```bash
npm create vite@latest bilcool-ui -- --template react-ts
cd bilcool-ui
npm install react-router-dom@7 @tanstack/react-query@5 zustand@5 \
  react-i18next i18next \
  @simplewebauthn/browser@13 date-fns@4 \
  @fullcalendar/react @fullcalendar/daygrid @fullcalendar/timegrid @fullcalendar/interaction \
  react-hook-form @hookform/resolvers zod lucide-react
npm install -D tailwindcss@4 @tailwindcss/vite vitest @testing-library/react @testing-library/user-event jsdom
npx shadcn@latest init
```

---

## 2. Project Directory Layout

```
bilcool-ui/
  public/
  src/
    api/
      client.ts               # base fetch helper: attaches JWT, handles 401
      auth.ts
      bookings.ts
      events.ts
    components/
      layout/
        AppShell.tsx
        Sidebar.tsx
        Header.tsx
        ThemeToggle.tsx
        LanguageToggle.tsx
      bookings/
        BookingCard.tsx
        BookingForm.tsx
        BookingStartEndDialog.tsx
        BookingDeleteConfirm.tsx
      calendar/
        CalendarView.tsx
        CalendarEvent.tsx
      ui/                     # shadcn generated components (do not hand-edit)
    hooks/
      useAuth.ts
      useBookings.ts
      useUsers.ts
    i18n/
      index.ts
      locales/
        en/
          common.json
          bookings.json
        sv/
          common.json
          bookings.json
    pages/
      auth/
        LoginPage.tsx
        OtpPage.tsx
      CalendarPage.tsx
      BookingsPage.tsx
      ProfilePage.tsx
      AdminUsersPage.tsx
      NotFoundPage.tsx
    stores/
      authStore.ts
      settingsStore.ts
    types/
      api.ts
    utils/
      bookingUtils.ts         # snapToQuarterHour, overlap check
      dateUtils.ts            # date-fns formatting helpers
    App.tsx
    main.tsx
    router.tsx
  index.html
  vite.config.ts
  tsconfig.json
```

---

## 3. TypeScript API Types (`src/types/api.ts`)

```typescript
export interface CreateUserRequest {
  username: string;
  email: string;
}

export interface UserResponse {
  user_ref: string;
  username: string;
  email: string;
  role: 'admin' | 'user';
}

export interface LoginBeginRequest {
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

export interface CompletedBookingPayload {
  booking: BookingResponse;
  distance: {
    start_distance: number;
    end_distance: number;
  };
}
```

---

## 4. Backend Services and Vite Proxy

### Service ports (docker-compose defaults)

| Service | Port |
|---|---|
| Authentication | 8082 |
| Bookings | 8081 |
| Event Ledger | 8083 |

### `vite.config.ts`

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 3000,
    proxy: {
      '/api/auth':   { target: 'http://localhost:8082', rewrite: path => path.replace('/api/auth', '') },
      '/api/book':   { target: 'http://localhost:8081', rewrite: path => path.replace('/api/book', '') },
      '/api/events': { target: 'http://localhost:8083', rewrite: path => path.replace('/api/events', '') },
    },
  },
});
```

Use these prefixes in all API modules:
- Auth: `/api/auth/api/v1`
- Bookings: `/api/book/api/v1`
- Events: `/api/events/api/v1`

---

## 5. API Layer

### `src/api/client.ts`

```typescript
export async function apiFetch<T>(input: RequestInfo, init?: RequestInit): Promise<T> {
  const token = localStorage.getItem('bilcool_token');
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...init?.headers,
  };
  const res = await fetch(input, { ...init, headers });
  if (res.status === 401) {
    localStorage.removeItem('bilcool_token');
    window.dispatchEvent(new Event('auth:logout'));
    throw new Error('Unauthorized');
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: 'Unknown error' }));
    throw Object.assign(new Error(err.message ?? 'Request failed'), { status: res.status, body: err });
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') return undefined as T;
  return res.json();
}
```

### `src/api/auth.ts`

```typescript
const BASE = '/api/auth/api/v1';

export const beginLogin = (body: LoginBeginRequest) =>
  apiFetch<LoginBeginResponse>(`${BASE}/users/login`, { method: 'POST', body: JSON.stringify(body) });

export const verifyToken = (body: VerifyTokenRequest) =>
  apiFetch<VerifyTokenResponse>(`${BASE}/users/login/token`, { method: 'POST', body: JSON.stringify(body) });

export const completeLogin = (body: LoginCompleteRequest) =>
  apiFetch<LoginCompleteResponse>(`${BASE}/users/login/complete`, { method: 'POST', body: JSON.stringify(body) });

export const createUser = (body: CreateUserRequest) =>
  apiFetch<UserResponse>(`${BASE}/users`, { method: 'POST', body: JSON.stringify(body) });

export const getUser = (id: string) =>
  apiFetch<UserResponse>(`${BASE}/users/${id}`);

export const deleteUser = (id: string) =>
  apiFetch<void>(`${BASE}/users/${id}`, { method: 'DELETE' });
```

### `src/api/bookings.ts`

```typescript
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
```

### `src/api/events.ts`

```typescript
const BASE = '/api/events/api/v1';

export const listEvents = (params: EventQueryParams) => {
  const qs = new URLSearchParams(
    Object.entries(params).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])
  ).toString();
  return apiFetch<EventResponse[]>(`${BASE}/events${qs ? `?${qs}` : ''}`);
};
```

---

## 6. Authentication Flow

### Branch A — First login (no passkey registered)

1. User enters email on `LoginPage` and submits.
2. `POST /api/v1/users/login` returns `{ next_step: "verify_token" }`.
3. Navigate to `/login/otp` passing `{ state: { email } }`.
4. User enters the 6-digit OTP sent to their email.
5. `POST /api/v1/users/login/token` returns `{ session_id, options }` (WebAuthn creation options).
6. Call `startRegistration(options)` from `@simplewebauthn/browser`. The browser prompts to create a passkey.
7. `POST /api/v1/users/login/complete` with `{ session_id, credential }` returns `{ token }`.
8. Store JWT in `localStorage` key `bilcool_token`. Fetch user profile. Navigate to `/`.

### Branch B — Returning login (passkey exists)

1. User enters email on `LoginPage` and submits.
2. `POST /api/v1/users/login` returns `{ next_step: "passkey_assertion", session_id, options }` (WebAuthn request options).
3. Immediately call `startAuthentication(options)` from `@simplewebauthn/browser` — do NOT navigate away.
4. `POST /api/v1/users/login/complete` with `{ session_id, credential }` returns `{ token }`.
5. Store JWT in `localStorage` key `bilcool_token`. Fetch user profile. Navigate to `/`.

### JWT and user profile

After storing the token:
- Base64-decode the middle segment of the JWT and JSON-parse it.
- Extract `user_ref` claim.
- Call `GET /api/v1/users/:user_ref` to load the full `UserResponse`.
- Store `{ token, userRef, username, email, role }` in `authStore`.

### Logout

- Remove `bilcool_token` from `localStorage`.
- Reset `authStore`.
- Navigate to `/login`.

### Route guards

- All routes except `/login` and `/login/otp` require authentication. Redirect unauthenticated users to `/login`.
- Routes under `/admin/*` require `role === 'admin'`. Redirect non-admins to `/`.
- Implement `<RequireAuth>` and `<RequireAdmin>` wrapper components.

---

## 7. Global State (Zustand)

### `src/stores/authStore.ts`

```typescript
interface AuthState {
  token: string | null;
  userRef: string | null;
  username: string | null;
  email: string | null;
  role: 'admin' | 'user' | null;
  isAuthenticated: boolean;
  setAuth: (payload: { token: string; userRef: string; username: string; email: string; role: 'admin' | 'user' }) => void;
  clearAuth: () => void;
}
```

On app boot (`main.tsx`): read `bilcool_token` from `localStorage`. If present and the `exp` claim is in the future, restore the token and fetch user details. If expired, clear the token and do not navigate.

### `src/stores/settingsStore.ts`

```typescript
interface SettingsState {
  theme: 'light' | 'dark' | 'system';
  language: 'en' | 'sv';
  setTheme: (t: 'light' | 'dark' | 'system') => void;
  setLanguage: (l: 'en' | 'sv') => void;
}
```

Persist with Zustand's `persist` middleware, key `bilcool_settings`. Default language: `'sv'`.

Apply theme by toggling the `dark` class on `<html>`. For `'system'`, watch `window.matchMedia('(prefers-color-scheme: dark)')`.

---

## 8. Routing (`src/router.tsx`)

```
/login          → LoginPage          (public)
/login/otp      → OtpPage            (public; requires router state.email)
/               → CalendarPage       (RequireAuth)
/bookings       → BookingsPage       (RequireAuth)
/profile        → ProfilePage        (RequireAuth)
/admin/users    → AdminUsersPage     (RequireAdmin)
*               → NotFoundPage
```

All authenticated routes render inside `AppShell`.

---

## 9. Views and Pages

### 9.1 LoginPage (`/login`)

- Single centred card with BilCool logo.
- Email input (`type="email"`, `autocomplete="email"`).
- "Sign in" / "Logga in" button.
- Loading state during API call.
- Error display (user not found, passkey failed).

On submit, run Branch A or Branch B depending on `next_step`. For Branch B, the passkey dialog appears without navigating away.

### 9.2 OtpPage (`/login/otp`)

- Read `email` from React Router `location.state`. If absent, redirect to `/login`.
- Show "We sent a 6-digit code to {email}".
- Single `<input maxLength={6} inputMode="numeric" autocomplete="one-time-code">` or six individual digit inputs.
- "Verify" button and "Back" link to `/login`.
- Error: "Invalid or expired code."

On successful verification the browser passkey registration dialog appears immediately (step 6 of Branch A).

### 9.3 CalendarPage (`/`)

**FullCalendar configuration:**

| Option | Value |
|---|---|
| `initialView` | `timeGridWeek` (desktop), `timeGridDay` (mobile, ≤768px) |
| `slotDuration` | `'00:15:00'` |
| `slotMinTime` | `'06:00:00'` |
| `slotMaxTime` | `'22:00:00'` |
| `allDaySlot` | `false` |
| `selectable` | `true` |
| `headerToolbar` | `{ left: 'prev,next today', center: 'title', right: '' }` |

Custom view switcher buttons (Month / Week / Day) sit above the calendar.

Events are loaded from `listBookings()` via TanStack Query, mapped to FullCalendar event objects. Current user's bookings use the primary brand colour; other users' bookings are greyed out and non-interactive.

**`select` callback:** opens `BookingForm` modal pre-filled with the selected range, snapped to 15-minute boundaries.

**`eventClick` callback:** opens `BookingStartEndDialog` if the event belongs to the current user.

**BookingForm modal:**
- Start and end date/time pickers, values snapped to 15-minute intervals.
- End must be after start (minimum 15 minutes).
- Overlap validation against all loaded bookings.
- Save calls `upsertBooking`; on success invalidates `['bookings']`.

**BookingStartEndDialog modal:**
- Shows booking times.
- If `start_date > now`: show "Cancel Booking" button → calls `deleteBooking`.
- If `start_date <= now` and no distance recorded: show "End Booking" — inputs for Start Odometer (km) and End Odometer (km); end must be ≥ start; submits `endBooking`.
- If distance already recorded: read-only (completed).

### 9.4 BookingsPage (`/bookings`)

**Monthly Summary table** (derived client-side from `listBookings()` + event ledger):

| Column | Source |
|---|---|
| Month | Group by `YYYY-MM` |
| My Bookings | Count where `user_ref === authStore.userRef` |
| Total Bookings | Count all in month |
| My Distance (km) | Sum from `booking_ended` events in event ledger |

To get distance: query `GET /api/v1/events?producer=bookings&event_type=booking_ended`. Parse payload as `CompletedBookingPayload`. Computed km = `(end_distance - start_distance) / 1000`.

**Per-User Booking List table:**

Columns: User | Date | Start Time | End Time | Distance (km) | Status

Status values: "Upcoming" | "Active" (start ≤ now < end) | "Completed" (has distance) | "Overdue" (end in past, no distance).

Filters: month/year selector, user filter (admin sees all; regular user sees only their own bookings).

### 9.5 ProfilePage (`/profile`)

Displays: Username, Email, Role (badge), "1 passkey registered" (exact passkey listing is not available via the current API).

### 9.6 AdminUsersPage (`/admin/users`)

**Create User form:**
- Username and email inputs.
- "Create User" button → calls `createUser`; shows success toast and resets form; on error shows message.

**User list table:**

Columns: Username | Email | Role | Actions (Delete button).

Admin cannot delete themselves — compare `user_ref` to `authStore.userRef` and disable the button for the current user's row.

Data source: there is no list-all-users endpoint. Derive unique users from `listBookings()` and call `getUser(userRef)` for each via `useQueries`. Show a note: "Only users with at least one booking are listed."

---

## 10. Booking Rules (Frontend Enforcement)

1. **15-minute snap:** Implement `snapToQuarterHour(date: Date): Date` in `src/utils/bookingUtils.ts`. Always snap picker values before sending to API.
2. **End after start:** `end_date > start_date`, minimum 15 minutes.
3. **No overlap:** Before submitting, check all loaded bookings. Overlap condition: `proposed.start < existing.end && proposed.end > existing.start`. Exclude the booking being edited.
4. **Cancel only future bookings:** Only show the cancel/delete option when `start_date > new Date()`.
5. **Manual start/end:** Creation sets the time range only. Ending a booking requires explicit odometer input via `POST /:id/end`.
6. **Client-generated UUID:** When creating a new booking, generate `booking_reference` with `crypto.randomUUID()`.

---

## 11. i18n

Use `react-i18next`. Language comes from `settingsStore.language` — call `i18n.changeLanguage()` whenever it changes. Two namespaces: `common` and `bookings`. Set `lang` attribute on `<html>` when language changes.

### `src/i18n/locales/en/common.json`

```json
{
  "app_name": "BilCool",
  "nav": {
    "calendar": "Calendar",
    "bookings": "Bookings",
    "profile": "Profile",
    "admin_users": "Users",
    "sign_out": "Sign out"
  },
  "theme": { "light": "Light", "dark": "Dark", "system": "System" },
  "language": { "en": "English", "sv": "Swedish" },
  "auth": {
    "email_label": "Email address",
    "email_placeholder": "you@example.com",
    "sign_in": "Sign in",
    "otp_title": "Check your email",
    "otp_description": "We sent a 6-digit code to {{email}}",
    "otp_label": "One-time code",
    "verify": "Verify",
    "back_to_login": "Back",
    "error_user_not_found": "No account found with that email address",
    "error_invalid_token": "The code is invalid or has expired",
    "error_passkey_failed": "Passkey authentication failed",
    "error_generic": "Something went wrong. Please try again."
  },
  "profile": {
    "title": "Profile",
    "username": "Username",
    "email": "Email",
    "role": "Role",
    "passkeys": "Passkeys",
    "passkeys_registered_one": "{{count}} passkey registered",
    "passkeys_registered_other": "{{count}} passkeys registered"
  },
  "admin": {
    "title": "User Management",
    "create_user": "Create User",
    "username_label": "Username",
    "email_label": "Email",
    "col_username": "Username",
    "col_email": "Email",
    "col_role": "Role",
    "col_actions": "Actions",
    "delete_user": "Delete",
    "delete_confirm": "Are you sure you want to delete {{username}}?",
    "user_created": "User {{username}} created successfully",
    "user_deleted": "User deleted",
    "no_users": "No users with bookings found"
  },
  "errors": { "unexpected": "An unexpected error occurred" }
}
```

### `src/i18n/locales/en/bookings.json`

```json
{
  "title": "Bookings",
  "col_user": "User",
  "col_date": "Date",
  "col_start": "Start",
  "col_end": "End",
  "col_distance": "Distance (km)",
  "col_status": "Status",
  "status_upcoming": "Upcoming",
  "status_active": "Active",
  "status_completed": "Completed",
  "status_overdue": "Overdue",
  "summary_title": "Monthly Summary",
  "summary_col_month": "Month",
  "summary_col_my_bookings": "My Bookings",
  "summary_col_total": "Total Bookings",
  "summary_col_my_km": "My Distance (km)",
  "form_title_new": "New Booking",
  "form_title_edit": "Edit Booking",
  "form_start": "Start",
  "form_end": "End",
  "form_save": "Save",
  "form_cancel": "Cancel",
  "form_error_overlap": "This time slot overlaps an existing booking",
  "form_error_end_before_start": "End time must be after start time",
  "form_error_min_duration": "Minimum booking duration is 15 minutes",
  "end_booking_title": "End Booking",
  "end_booking_start_odo": "Start Odometer (km)",
  "end_booking_end_odo": "End Odometer (km)",
  "end_booking_submit": "Confirm End",
  "end_booking_error_odo": "End odometer must be greater than or equal to start",
  "cancel_booking": "Cancel Booking",
  "cancel_booking_confirm": "Cancel this booking?",
  "filter_month": "Month",
  "filter_user": "User",
  "filter_all_users": "All users",
  "no_bookings": "No bookings found"
}
```

### `src/i18n/locales/sv/common.json`

```json
{
  "app_name": "BilCool",
  "nav": {
    "calendar": "Kalender",
    "bookings": "Bokningar",
    "profile": "Profil",
    "admin_users": "Användare",
    "sign_out": "Logga ut"
  },
  "theme": { "light": "Ljust", "dark": "Mörkt", "system": "System" },
  "language": { "en": "Engelska", "sv": "Svenska" },
  "auth": {
    "email_label": "E-postadress",
    "email_placeholder": "du@exempel.se",
    "sign_in": "Logga in",
    "otp_title": "Kontrollera din e-post",
    "otp_description": "Vi skickade en 6-siffrig kod till {{email}}",
    "otp_label": "Engångskod",
    "verify": "Verifiera",
    "back_to_login": "Tillbaka",
    "error_user_not_found": "Inget konto hittades med den e-postadressen",
    "error_invalid_token": "Koden är ogiltig eller har gått ut",
    "error_passkey_failed": "Autentisering med nyckel misslyckades",
    "error_generic": "Något gick fel. Försök igen."
  },
  "profile": {
    "title": "Profil",
    "username": "Användarnamn",
    "email": "E-post",
    "role": "Roll",
    "passkeys": "Passnycklar",
    "passkeys_registered_one": "{{count}} passnyckel registrerad",
    "passkeys_registered_other": "{{count}} passnycklar registrerade"
  },
  "admin": {
    "title": "Användarhantering",
    "create_user": "Skapa användare",
    "username_label": "Användarnamn",
    "email_label": "E-post",
    "col_username": "Användarnamn",
    "col_email": "E-post",
    "col_role": "Roll",
    "col_actions": "Åtgärder",
    "delete_user": "Radera",
    "delete_confirm": "Vill du verkligen radera {{username}}?",
    "user_created": "Användare {{username}} skapades",
    "user_deleted": "Användare raderad",
    "no_users": "Inga användare med bokningar hittades"
  },
  "errors": { "unexpected": "Ett oväntat fel uppstod" }
}
```

### `src/i18n/locales/sv/bookings.json`

```json
{
  "title": "Bokningar",
  "col_user": "Användare",
  "col_date": "Datum",
  "col_start": "Start",
  "col_end": "Slut",
  "col_distance": "Sträcka (km)",
  "col_status": "Status",
  "status_upcoming": "Kommande",
  "status_active": "Aktiv",
  "status_completed": "Avslutad",
  "status_overdue": "Försenad",
  "summary_title": "Månadssammanfattning",
  "summary_col_month": "Månad",
  "summary_col_my_bookings": "Mina bokningar",
  "summary_col_total": "Totala bokningar",
  "summary_col_my_km": "Min sträcka (km)",
  "form_title_new": "Ny bokning",
  "form_title_edit": "Redigera bokning",
  "form_start": "Start",
  "form_end": "Slut",
  "form_save": "Spara",
  "form_cancel": "Avbryt",
  "form_error_overlap": "Det här tidsintervallet överlappar en befintlig bokning",
  "form_error_end_before_start": "Sluttiden måste vara efter starttiden",
  "form_error_min_duration": "Minsta bokningstid är 15 minuter",
  "end_booking_title": "Avsluta bokning",
  "end_booking_start_odo": "Startmätarställning (km)",
  "end_booking_end_odo": "Slutmätarställning (km)",
  "end_booking_submit": "Bekräfta avslut",
  "end_booking_error_odo": "Slutmätarställningen måste vara minst lika hög som startmätarställningen",
  "cancel_booking": "Avboka",
  "cancel_booking_confirm": "Avboka den här bokningen?",
  "filter_month": "Månad",
  "filter_user": "Användare",
  "filter_all_users": "Alla användare",
  "no_bookings": "Inga bokningar hittades"
}
```

---

## 12. AppShell Layout

`AppShell.tsx` wraps every authenticated page:

- **Desktop (≥768px):** fixed sidebar on the left with nav links and sign-out at the bottom.
- **Mobile (<768px):** top header with hamburger icon that opens a slide-over drawer with the same nav content.
- **Top-right controls:** `LanguageToggle` (EN / SV) and `ThemeToggle` (icon button cycling light / dark / system).

**Sidebar nav links (with Lucide icons):**

| Label key | Icon | Guard |
|---|---|---|
| `nav.calendar` | `CalendarDays` | all authenticated |
| `nav.bookings` | `ClipboardList` | all authenticated |
| `nav.profile` | `User` | all authenticated |
| `nav.admin_users` | `Users` | admin only |

Use React Router `NavLink` for active-link styling.

---

## 13. TanStack Query Setup

Wrap `App` with `QueryClientProvider`. Configure:

```typescript
new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: true },
  },
});
```

Per-query overrides:
- Bookings list: `staleTime: 30_000`
- User data: `staleTime: 5 * 60_000`

**Query keys:**
- `['bookings']` — all bookings
- `['booking', id]` — single booking
- `['user', id]` — single user
- `['events', params]` — event ledger

After any booking mutation (`upsertBooking`, `deleteBooking`, `endBooking`), call `queryClient.invalidateQueries({ queryKey: ['bookings'] })`.

---

## 14. Error Handling

| HTTP status | Behaviour |
|---|---|
| 400 | Show `error.body.message` as a form-level or inline error |
| 401 | Clear auth, navigate to `/login` (handled in `apiFetch`) |
| 403 | Toast: "You do not have permission to perform this action" |
| 404 | Show "Not found" inline |
| 409 | Show overlap/conflict error on the relevant form |
| 422 | Show "Booking already started" error |
| 500 | Toast: generic error message |

---

## 15. Accessibility

- All interactive elements have accessible labels (`aria-label`, `aria-describedby`, or visible text).
- Modal dialogs: Radix UI `Dialog` — focus trap when open, closes on Escape.
- Form inputs: each has `<label htmlFor>` pointing to the input.
- Colour contrast: WCAG AA minimum (4.5:1 normal text, 3:1 large text) in both themes.
- Full keyboard navigation: all nav, buttons, and form controls reachable by Tab.
- Calendar events: `aria-label` describing the booking on each event element.
- OTP input: `autocomplete="one-time-code"` with appropriate `aria-label`.

---

## 16. Responsive Design

- Mobile-first layout with Tailwind `md:` prefixes.
- Sidebar → bottom nav bar or drawer on screens narrower than `md` (768px).
- Calendar: `timeGridDay` default on mobile, `timeGridWeek` on desktop.
- Tables on mobile: switch to a card-based layout using CSS Grid.
- Minimum body text size: 16px on mobile.
- Minimum touch target size: 44×44px.
- No horizontal scroll from 320px viewport width upward.

---

## 17. Theme and FOUC Prevention

Add an inline script to `index.html` before `<body>` to apply the theme class before first paint:

```html
<script>
  (function () {
    var s = JSON.parse(localStorage.getItem('bilcool_settings') || '{}');
    var theme = s.state && s.state.theme || 'system';
    if (theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
      document.documentElement.classList.add('dark');
    }
  })();
</script>
```

---

## 18. Date/Time Conventions

All timestamps are ISO 8601 with timezone offset. Use `date-fns` with locale:

```typescript
import { sv, enGB } from 'date-fns/locale';
const locale = settingsStore.language === 'sv' ? sv : enGB;
```

- Time: `HH:mm` (24-hour) in both languages.
- Date: `d MMM yyyy` → "5 apr 2026" / "5 Apr 2026".
- Month/year: `MMMM yyyy` → "april 2026" / "April 2026".

---

## 19. Testing

Minimum test coverage with Vitest + React Testing Library. Mock all API calls with `vi.mock`.

1. **`authStore.test.ts`** — `setAuth` stores fields; `clearAuth` resets to null; persisted token is restored on boot; expired token is cleared.
2. **`bookingUtils.test.ts`** — `snapToQuarterHour` snaps `:00`, `:15`, `:30`, `:45` correctly; overlap detection returns correct boolean for edge cases.
3. **`LoginPage.test.tsx`** — renders email input; submit triggers `beginLogin`; navigates to `/login/otp` when `next_step === 'verify_token'`; shows error for unknown email; calls `startAuthentication` when `next_step === 'passkey_assertion'`.
4. **`BookingForm.test.tsx`** — validates overlap; validates end-after-start; calls `upsertBooking` on valid submit.
5. **`CalendarPage.test.tsx`** — renders FullCalendar; booking data from query is passed as events.

---

## 20. Build Scripts (`package.json`)

```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest",
    "lint": "eslint src --ext ts,tsx --report-unused-disable-directives --max-warnings 0"
  }
}
```

---

## 21. Implementation Order

Build in this sequence to avoid blocked dependencies:

1. Scaffold Vite project, install dependencies, configure TypeScript strict mode.
2. Set up Tailwind CSS v4 and shadcn/ui.
3. Define all types in `src/types/api.ts`.
4. Implement `src/api/client.ts` and all API modules.
5. Implement `authStore` and `settingsStore` with Zustand persist.
6. Set up i18next with EN and SV locale files; FOUC prevention script in `index.html`.
7. Build `LoginPage` and `OtpPage` with the complete WebAuthn flow.
8. Build `AppShell` with sidebar, header, theme toggle, language toggle.
9. Set up router with `RequireAuth` and `RequireAdmin` guards.
10. Build `CalendarPage` with `BookingForm` and `BookingStartEndDialog`.
11. Build `BookingsPage` with monthly summary and booking list.
12. Build `ProfilePage`.
13. Build `AdminUsersPage`.
14. Write tests.
15. Audit accessibility and responsive layout.
