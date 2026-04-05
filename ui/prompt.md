Build a React SPA for **BilCool**, a booking management application. Use TypeScript, React Router, and Tailwind CSS. Mock all API calls with realistic in-memory data — no backend needed.

## Data Model

```ts
type Booking = {
  bookingReference: string  // UUID
  userRef: string           // UUID
  startDate: string         // ISO 8601
  endDate: string           // ISO 8601
}
```

## Three Views

### Bookings
#### 1. `/bookings` — Bookings List
- Display all bookings a calendar-like view
- It is possible to choose a date range, and view bookings by day, week or month
- Each row/card shows: booking reference (truncated), user ref (truncated), start date, end date, duration
- Clicking a row navigates to the detail view
- A prominent "New Booking" button navigates to the create form
- Show an empty state when no bookings exist

#### 2. `/bookings/:id` — Booking Detail
- Display all fields for a single booking in full (no truncation)
- Show a human-readable duration (e.g. "3 days")
- Show start and end times in the local timezone
- "Edit" button that navigates to the edit form pre-filled with current data
- "Delete" button with a confirmation step before removing the booking
- "Back to list" navigation

#### 3. `/bookings/new` and `/bookings/:id/edit` — Create / Edit Form
- Fields: User Reference (UUID input), Start Date (date-time picker), End Date (date-time picker)
- Validate that end date is after start date
- On save, navigate to the detail view for the saved booking
- On cancel, navigate back without saving
- For new bookings, auto-generate a UUID for the booking reference

### Users
 - Display all users in a list
 - Clicking a row navigates to the detail view
 - Show an empty state when no users exist
 - "New User" button navigates to the create form
 - "Edit" button that navigates to the edit form pre-filled with current data
 - "Delete" button with a confirmation step before removing the user
 - "Back to list" navigation
 - To create a new user the logged in user must have admin privileges
 - To edit a user the logged in user must have admin privileges or the user must be the logged in user

## Design Requirements

- Pick a distinctive visual theme — avoid generic purple gradients and Inter/Roboto fonts
- Use a dark or deeply saturated color palette with sharp accent colors
- Commit to a single cohesive aesthetic across all views
- Animate route transitions and list item entrances
- Responsive layout that works on both desktop and mobile
- Empty states and loading states should be styled, not plain text
