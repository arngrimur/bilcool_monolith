package application

import (
	"context"

	"bilcool_monolith/internal/pkg/booking/internal/pkg/application/queries"
	"bilcool_monolith/internal/pkg/booking/internal/pkg/domain"
	"bilcool_monolith/internal/pkg/booking/internal/pkg/persistance"
)

// The interfaces supported by the application
type (
	App interface {
		Commands
		Queries
	}

	Commands interface{}

	Queries interface {
		GetBooking(ctx context.Context, request domain.BookingRequest) (domain.BookingResponse, error)
	}
)

// The concrete application implementation
type (
	Application struct {
		appCommands
		appQueries
	}
	appCommands struct{}
	appQueries  struct {
		queries.GetBookingsHandler
	}
)

// Dummy for interface
var _ App = (*Application)(nil)

func New(bookingsRepo persistance.BookingsRepository) *Application {
	return &Application{
		appCommands{},
		appQueries{
			GetBookingsHandler: queries.NewGetBookingsHandler(bookingsRepo),
		},
	}
}
