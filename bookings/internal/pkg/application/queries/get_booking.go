package queries

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

type GetBookingsHandler struct {
	*domain.Bookings
}

func NewGetBookingsHandler(bookings *domain.Bookings) GetBookingsHandler {
	return GetBookingsHandler{
		Bookings: bookings,
	}
}

func (h GetBookingsHandler) GetBooking(ctx context.Context, b domain.BookingRequest) (domain.BookingResponse, error) {
	return h.Bookings.Find(ctx, b)
}

func (h GetBookingsHandler) GetAllBooking(ctx context.Context) ([]domain.BookingResponse, error) {
	return h.Bookings.FindAll(ctx)
}
