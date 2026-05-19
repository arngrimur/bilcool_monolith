package commands

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

type DistanceCalculator interface {
	CalculateRoadDistance(ctx context.Context, points []extdomain.TrackPoint) (int, error)
}

type UpdateBookingsHandler struct {
	*domain.Bookings
	distanceCalc DistanceCalculator
}

func NewUpdateBookingsHandler(bookings *domain.Bookings, distanceCalc DistanceCalculator) UpdateBookingsHandler {
	return UpdateBookingsHandler{
		Bookings:     bookings,
		distanceCalc: distanceCalc,
	}
}

func (h UpdateBookingsHandler) UpdateBooking(ctx context.Context, b domain.UpdateBookingRequest) error {
	return h.Bookings.UpdateBooking(ctx, b)
}

func (h UpdateBookingsHandler) DeleteBooking(ctx context.Context, request domain.BookingRequest) error {
	return h.Bookings.DeleteBooking(ctx, request)
}

func (h UpdateBookingsHandler) EndBooking(ctx context.Context, request domain.EndBookingRequest) error {
	if request.OdometerStart != nil && request.OdometerEnd != nil {
		request.Distance = extdomain.Distance{
			StartDistance: *request.OdometerStart,
			EndDistance:   *request.OdometerEnd,
		}
	} else {
		points, err := h.Bookings.GetTrackPoints(ctx, request.BookingRequest)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to get track points for distance calculation")
		} else if len(points) >= 2 {
			meters, err := h.distanceCalc.CalculateRoadDistance(ctx, points)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("mapbox distance calculation failed, distance will be 0")
			} else {
				request.Distance = extdomain.Distance{StartDistance: 0, EndDistance: meters}
			}
		}
	}
	return h.Bookings.EndBooking(ctx, request)
}

func (h UpdateBookingsHandler) PauseBooking(ctx context.Context, request domain.PauseBookingRequest) error {
	return h.Bookings.PauseBooking(ctx, request)
}

func (h UpdateBookingsHandler) ResumeBooking(ctx context.Context, request domain.BookingRequest) (domain.PauseBookingResponse, error) {
	return h.Bookings.ResumeBooking(ctx, request)
}

func (h UpdateBookingsHandler) AddTrackPoints(ctx context.Context, request domain.AddTrackPointsRequest) error {
	return h.Bookings.AddTrackPoints(ctx, request)
}
