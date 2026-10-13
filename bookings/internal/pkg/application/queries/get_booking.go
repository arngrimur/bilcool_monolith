package queries

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

// RouteMatcher snaps GPS points to the road network and returns one line per matched segment.
type RouteMatcher interface {
	MatchRoute(ctx context.Context, points []extdomain.TrackPoint) ([][]extdomain.Position, error)
}

type GetBookingsHandler struct {
	*domain.Bookings
	routeMatcher RouteMatcher
}

func NewGetBookingsHandler(bookings *domain.Bookings, routeMatcher RouteMatcher) GetBookingsHandler {
	return GetBookingsHandler{
		Bookings:     bookings,
		routeMatcher: routeMatcher,
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

// GetRoute returns the recorded track points of a booking together with the track snapped to
// the road network. Snapping is best effort: when it fails the route is empty and the caller
// is expected to draw the points.
func (h GetBookingsHandler) GetRoute(ctx context.Context, b domain.BookingRequest) (extdomain.RouteResponse, error) {
	points, err := h.GetTrackPoints(ctx, b)
	if err != nil {
		return extdomain.RouteResponse{}, err
	}

	route := [][]extdomain.Position{}
	if len(points) >= 2 && h.routeMatcher != nil {
		matched, err := h.routeMatcher.MatchRoute(ctx, points)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to snap route to roads, returning raw points only")
		} else if matched != nil {
			route = matched
		}
	}
	return extdomain.RouteResponse{Points: points, Route: route}, nil
}
