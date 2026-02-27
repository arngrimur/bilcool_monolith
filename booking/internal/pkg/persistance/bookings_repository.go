package persistance

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

type BookingsRepository interface {
	Find(ctx context.Context, request domain.BookingRequest) (domain.BookingResponse, error)
	FindAll(ctx context.Context) ([]domain.BookingResponse, error)
	UpdateBooking(ctx context.Context, request domain.UpdateBookingRequest) error
}
