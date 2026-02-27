package commands

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/persistance"
)

type UpdateBookingsHandler struct {
	persistance.BookingsRepository
}

func NewUpdateBookingsHandler(bookings persistance.BookingsRepository) UpdateBookingsHandler {
	return UpdateBookingsHandler{
		BookingsRepository: bookings,
	}
}

func (h UpdateBookingsHandler) UpdateBooking(ctx context.Context, b domain.UpdateBookingRequest) error {
	return h.BookingsRepository.UpdateBooking(ctx, b)
}

func (h UpdateBookingsHandler) DeleteBooking(ctx context.Context, request domain.BookingRequest) error {
	return h.BookingsRepository.DeleteBooking(ctx, request)
}
