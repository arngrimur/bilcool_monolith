package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BookingResponse struct {
	UserRef          uuid.UUID `json:"user_ref" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426655440000" type:"uuid" binding:"required,uuid"`
	BookingReference uuid.UUID `json:"booking_reference" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426655440000" binding:"required,uuid"`
	StartDate        time.Time `json:"start_date" example:"2021-01-01 10:00:00" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate          time.Time `json:"end_date" example:"2021-01-01 12:00:00" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
}

type UpdateBookingRequest BookingResponse

type BookingRequest struct {
	BookingReference uuid.UUID `uri:"id" json:"booking_reference" validate:"required,uuid" binding:"required,uuid"`
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
	// TODO check that the booing has not started
	return b.r.DeleteBooking(ctx, request)
}

func (b Bookings) FindAll(ctx context.Context) ([]BookingResponse, error) {
	return b.r.FindAll(ctx)
}

func (b Bookings) Find(ctx context.Context, request BookingRequest) (BookingResponse, error) {
	return b.r.Find(ctx, request)
}
