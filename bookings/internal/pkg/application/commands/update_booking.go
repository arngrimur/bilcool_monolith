package commands

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

type UpdateBookingsHandler struct {
	*domain.Bookings
}

func NewUpdateBookingsHandler(bookings *domain.Bookings) UpdateBookingsHandler {
	return UpdateBookingsHandler{
		Bookings: bookings,
	}
}

func (h UpdateBookingsHandler) UpdateBooking(ctx context.Context, b domain.UpdateBookingRequest) error {
	return h.Bookings.UpdateBooking(ctx, b)
}

func (h UpdateBookingsHandler) DeleteBooking(ctx context.Context, request domain.BookingRequest) error {
	return h.Bookings.DeleteBooking(ctx, request)
}

func (h UpdateBookingsHandler) EndBooking(ctx context.Context, request domain.EndBookingRequest) error {
	return h.Bookings.EndBooking(ctx, request)
}
