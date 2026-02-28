package domain

import (
	"context"
)

type BookingsRepository interface {
	Find(ctx context.Context, request BookingRequest) (BookingResponse, error)
	FindAll(ctx context.Context) ([]BookingResponse, error)
	UpdateBooking(ctx context.Context, request UpdateBookingRequest) error
	DeleteBooking(ctx context.Context, request BookingRequest) error
}
