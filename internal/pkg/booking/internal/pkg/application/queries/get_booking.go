package queries

import (
	"context"

	"bilcool_monolith/internal/pkg/booking/internal/pkg/domain"
	"bilcool_monolith/internal/pkg/booking/internal/pkg/persistance"
)

type GetBookingsHandler struct {
	persistance.BookingsRepository
}

func NewGetBookingsHandler(bookings persistance.BookingsRepository) GetBookingsHandler {
	return GetBookingsHandler{
		BookingsRepository: bookings,
	}
}

func (h GetBookingsHandler) GetBooking(ctx context.Context, b domain.BookingRequest) (domain.BookingResponse, error) {
	return h.Find(ctx, b)
}
