package application

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application/commands"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application/queries"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

// The interfaces supported by the application
type (
	App interface {
		Commands
		Queries
	}

	Commands interface {
		UpdateBooking(ctx context.Context, request domain.UpdateBookingRequest) error
	}

	Queries interface {
		GetBooking(ctx context.Context, request domain.BookingRequest) (domain.BookingResponse, error)
		GetAllBooking(ctx context.Context) ([]domain.BookingResponse, error)
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

func New(bookingsRepo domain.BookingsRepository) *Application {
	return &Application{
		appCommands{
			UpdateBookingsHandler: commands.NewUpdateBookingsHandler(domain.NewBookings(bookingsRepo)),
		},
		appQueries{
			GetBookingsHandler: queries.NewGetBookingsHandler(domain.NewBookings(bookingsRepo)),
		},
	}
}
