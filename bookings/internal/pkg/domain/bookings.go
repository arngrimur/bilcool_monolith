package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BookingResponse struct {
	UserRef          uuid.UUID `json:"user_ref" validate:"required,uuid"`
	BookingReference uuid.UUID `json:"booking_reference" validate:"required,uuid"`
	StartDate        time.Time `json:"start_date" validate:"required,date_format=2006-01-02"`
	EndDate          time.Time `json:"end_date" validate:"required,date_format=2006-01-02"`
}

type UpdateBookingRequest BookingResponse

type BookingRequest struct {
	BookingReference uuid.UUID `json:"booking_reference" validate:"required,uuid"`
}

func NewBookingResponse(bookingRef uuid.UUID, startTime time.Time, endTime time.Time, userRef uuid.UUID) BookingResponse {
	return BookingResponse{
		UserRef:          userRef,
		BookingReference: bookingRef,
		StartDate:        startTime,
		EndDate:          endTime,
	}
}

type Bookings struct {
	r BookingsRepository
}

func NewBookings(r BookingsRepository) *Bookings {
	return &Bookings{r}
}

func (b Bookings) UpdateBooking(ctx context.Context, request UpdateBookingRequest) error {
	return b.r.UpdateBooking(ctx, request)
}

func (b Bookings) DeleteBooking(ctx context.Context, request BookingRequest) error {
	return b.r.DeleteBooking(ctx, request)
}

func (b Bookings) FindAll(ctx context.Context) ([]BookingResponse, error) {
	return b.r.FindAll(ctx)
}

func (b Bookings) Find(ctx context.Context, request BookingRequest) (BookingResponse, error) {
	return b.r.Find(ctx, request)
}
