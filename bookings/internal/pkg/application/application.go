package application

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application/commands"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application/queries"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

// The interfaces supported by the application
type (
	App interface {
		Commands
		Queries
	}

	Commands interface {
		UpdateBooking(ctx context.Context, request domain.UpdateBookingRequest) error
		DeleteBooking(ctx context.Context, request domain.BookingRequest) error
		EndBooking(ctx context.Context, request domain.EndBookingRequest) error
		PauseBooking(ctx context.Context, request domain.PauseBookingRequest) error
		ResumeBooking(ctx context.Context, request domain.BookingRequest) (domain.PauseBookingResponse, error)
		AddTrackPoints(ctx context.Context, request domain.AddTrackPointsRequest) error
	}

	Queries interface {
		GetBooking(ctx context.Context, request domain.BookingRequest) (extdomain.BookingResponse, error)
		GetAllBooking(ctx context.Context) ([]extdomain.BookingResponse, error)
	}
)

// The concrete application implementation
type (
	Application struct {
		appCommands
		appQueries
	}
	appCommands struct {
		commands.UpdateBookingsHandler
	}
	appQueries struct {
		queries.GetBookingsHandler
	}
)

// Dummy for interface
var _ App = (*Application)(nil)

func New(bookingsRepo domain.BookingsRepository, distanceCalc commands.DistanceCalculator) *Application {
	return &Application{
		appCommands{
			UpdateBookingsHandler: commands.NewUpdateBookingsHandler(domain.NewBookings(bookingsRepo), distanceCalc),
		},
		appQueries{
			GetBookingsHandler: queries.NewGetBookingsHandler(domain.NewBookings(bookingsRepo)),
		},
	}
}
