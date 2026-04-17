package domain

import (
	"time"

	"github.com/google/uuid"
)

const EventBookingEnded string = "booking.ended"

type CompletedBooking struct {
	Booking  BookingResponse `json:"booking"`
	Distance Distance        `json:"distance"`
}

type BookingResponse struct {
	UserRef          uuid.UUID `json:"user_ref" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426655440000" type:"uuid" binding:"required,uuid"`
	BookingReference uuid.UUID `json:"booking_reference" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426655440000" binding:"required,uuid"`
	StartDate        time.Time `json:"start_date" example:"2021-01-01 10:00:00" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate          time.Time `json:"end_date" example:"2021-01-01 12:00:00" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	Distance         *Distance `json:"distance,omitempty"`
}

// Distance is the distance between the start and end point of the booking in meters
type Distance struct {
	StartDistance int `json:"start_distance" validate:"required,gte=0" example:"100"`
	EndDistance   int `json:"end_distance" validate:"required,gte=0" example:"200"`
}

func (d Distance) Distance() int {
	return d.EndDistance - d.StartDistance
}
