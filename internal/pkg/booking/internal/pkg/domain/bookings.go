package domain

import (
	"time"

	"github.com/google/uuid"
)

type BookingResponse struct {
	*StartBookingRequest
	*EndBookingRequest
}

type StartBookingRequest struct {
	BookingReference uuid.UUID `json:"booking_reference" validate:"required,uuid"`
	StartDate        time.Time `json:"start_date" validate:"required,date_format=2006-01-02"`
	UserRef          uuid.UUID `json:"user_ref" validate:"required,uuid"`
}
type UpdateBookingRequest struct {
	BookingReference uuid.UUID `json:"booking_reference" validate:"required,uuid"`
	StartDate        time.Time `json:"start_date" validate:"required,date_format=2006-01-02"`
}

type EndBookingRequest struct {
	BookingReference uuid.UUID `json:"booking_reference" validate:"required,uuid"`
	EndDate          time.Time `json:"end_date" validate:"required,date_format=2006-01-02"`
}

type BookingRequest struct {
	BookingReference uuid.UUID `json:"booking_reference" validate:"required,uuid"`
}

func (b BookingRequest) GetBooking() BookingResponse {
	// todo: Get the booking from the db
	panic("implement me")
}

func NewBookingResponse(reference uuid.UUID, startTime time.Time, endTime *time.Time, ref uuid.UUID) BookingResponse {
	var e *EndBookingRequest
	if endTime != nil {
		e = &EndBookingRequest{
			reference,
			*endTime,
		}
	}

	return BookingResponse{
		StartBookingRequest: &StartBookingRequest{reference, startTime, ref},
		EndBookingRequest:   e,
	}
}
