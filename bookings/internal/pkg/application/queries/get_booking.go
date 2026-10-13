package queries

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

type GetBookingsHandler struct {
	*domain.Bookings
}

func NewGetBookingsHandler(bookings *domain.Bookings) GetBookingsHandler {
	return GetBookingsHandler{
		Bookings: bookings,
	}
}

func (h GetBookingsHandler) GetBooking(ctx context.Context, b domain.BookingRequest) (extdomain.BookingResponse, error) {
	return h.Bookings.Find(ctx, b)
}

func (h GetBookingsHandler) GetAllBooking(ctx context.Context) ([]extdomain.BookingResponse, error) {
	return h.Bookings.FindAll(ctx)
}

func (h GetBookingsHandler) GetTrackPoints(ctx context.Context, b domain.BookingRequest) ([]extdomain.TrackPoint, error) {
	points, err := h.Bookings.GetTrackPoints(ctx, b)
	if err != nil {
		return nil, err
	}
	if points == nil {
		points = []extdomain.TrackPoint{}
	}
	return points, nil
}
