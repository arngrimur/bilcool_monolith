package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

var ErrBookingAlreadyStarted = errors.New("booking has already started")
var ErrUserNotFound = errors.New("user not found")

type UpdateBookingRequest extdomain.BookingResponse

type BookingRequest struct {
	BookingReference uuid.UUID `uri:"id" json:"booking_reference" validate:"required,uuid" binding:"required,uuid"`
}

type EndBookingRequest struct {
	BookingRequest
	extdomain.Distance
}

func NewBookingResponse(bookingRef uuid.UUID, startTime time.Time, endTime time.Time, userRef uuid.UUID, distance *extdomain.Distance) extdomain.BookingResponse {
	return extdomain.BookingResponse{
		UserRef:          userRef,
		BookingReference: bookingRef,
		StartDate:        startTime,
		EndDate:          endTime,
		Distance:         distance,
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

func (b Bookings) FindAll(ctx context.Context) ([]extdomain.BookingResponse, error) {
	return b.r.FindAll(ctx)
}

func (b Bookings) Find(ctx context.Context, request BookingRequest) (extdomain.BookingResponse, error) {
	return b.r.Find(ctx, request)
}

func (b Bookings) EndBooking(ctx context.Context, request EndBookingRequest) error {
	return b.r.EndBooking(ctx, request)
}
