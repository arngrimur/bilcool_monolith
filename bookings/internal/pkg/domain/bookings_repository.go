package domain

import (
	"context"
)

//go:generate mockgen -source=bookings_repository.go -destination=bookings_repository_mock.go -package=domain github.com/arngrimur/bilcool_monolith/bookings
type BookingsRepository interface {
	Find(ctx context.Context, request BookingRequest) (BookingResponse, error)
	FindAll(ctx context.Context) ([]BookingResponse, error)
	UpdateBooking(ctx context.Context, request UpdateBookingRequest) error
	DeleteBooking(ctx context.Context, request BookingRequest) error
	EndBooking(ctx context.Context, request EndBookingRequest) error
}
