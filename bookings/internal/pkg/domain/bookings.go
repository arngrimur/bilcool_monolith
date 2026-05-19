package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

var ErrBookingAlreadyStarted = errors.New("booking has already started")
var ErrUserNotFound = errors.New("user not found")
var ErrBookingAlreadyPaused = errors.New("booking is already paused")
var ErrBookingNotPaused = errors.New("booking is not paused")

type UpdateBookingRequest extdomain.BookingResponse

type BookingRequest struct {
	BookingReference uuid.UUID `uri:"id" json:"booking_reference" validate:"required,uuid" binding:"required,uuid"`
}

type EndBookingRequest struct {
	BookingRequest
	OdometerStart *int                `json:"odometer_start,omitempty"`
	OdometerEnd   *int                `json:"odometer_end,omitempty"`
	Position      *extdomain.Position `json:"position,omitempty"`
	Distance      extdomain.Distance  // populated by command handler, not from HTTP
}

type AddTrackPointsRequest struct {
	BookingRequest
	Points []extdomain.TrackPoint `json:"points"`
}

type PauseBookingRequest struct {
	BookingRequest
	Position extdomain.Position
}

type PauseBookingResponse struct {
	Position extdomain.Position `json:"position"`
}

func NewBookingResponse(bookingRef uuid.UUID, startTime time.Time, endTime time.Time, userRef uuid.UUID, distance *extdomain.Distance) extdomain.BookingResponse {
	return extdomain.BookingResponse{
		UserRef:          userRef,
		BookingReference: bookingRef,
		StartDate:        startTime,
		EndDate:          endTime,
		Distance:         distance,
	}
}

type Bookings struct {
	r BookingsRepository
}

func NewBookings(r BookingsRepository) *Bookings {
	return &Bookings{r}
}

func (b Bookings) UpdateBooking(ctx context.Context, request UpdateBookingRequest) error {
	return b.r.UpdateBooking(ctx, request)
}

func (b Bookings) DeleteBooking(ctx context.Context, request BookingRequest) error {
	return b.r.DeleteBooking(ctx, request)
}

func (b Bookings) FindAll(ctx context.Context) ([]extdomain.BookingResponse, error) {
	return b.r.FindAll(ctx)
}

func (b Bookings) Find(ctx context.Context, request BookingRequest) (extdomain.BookingResponse, error) {
	return b.r.Find(ctx, request)
}

func (b Bookings) EndBooking(ctx context.Context, request EndBookingRequest) error {
	return b.r.EndBooking(ctx, request)
}

func (b Bookings) PauseBooking(ctx context.Context, request PauseBookingRequest) error {
	return b.r.PauseBooking(ctx, request)
}

func (b Bookings) ResumeBooking(ctx context.Context, request BookingRequest) (PauseBookingResponse, error) {
	return b.r.ResumeBooking(ctx, request)
}

func (b Bookings) AddTrackPoints(ctx context.Context, request AddTrackPointsRequest) error {
	return b.r.AddTrackPoints(ctx, request)
}

func (b Bookings) GetTrackPoints(ctx context.Context, request BookingRequest) ([]extdomain.TrackPoint, error) {
	return b.r.GetTrackPoints(ctx, request)
}
