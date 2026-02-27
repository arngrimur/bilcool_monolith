package queries

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/persistance"
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
