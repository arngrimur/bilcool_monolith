package domain

import (
	"context"

	ext_domain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

//go:generate mockgen -source=bookings_repository.go -destination=bookings_repository_mock.go -package=domain github.com/arngrimur/bilcool_monolith/bookings
type BookingsRepository interface {
	Find(ctx context.Context, request BookingRequest) (ext_domain.BookingResponse, error)
	FindAll(ctx context.Context) ([]ext_domain.BookingResponse, error)
	UpdateBooking(ctx context.Context, request UpdateBookingRequest) error
	DeleteBooking(ctx context.Context, request BookingRequest) error
	EndBooking(ctx context.Context, request EndBookingRequest) error
	PauseBooking(ctx context.Context, request PauseBookingRequest) error
	ResumeBooking(ctx context.Context, request BookingRequest) (PauseBookingResponse, error)
	AddUser(ctx context.Context, message brokerpostgres.Message) error
	DeleteUser(ctx context.Context, message brokerpostgres.Message) error
}
