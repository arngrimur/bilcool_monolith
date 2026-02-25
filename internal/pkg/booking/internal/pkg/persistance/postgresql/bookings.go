package postgresql

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"bilcool_monolith/internal/pkg/booking/internal/pkg/domain"
	"bilcool_monolith/internal/pkg/booking/internal/pkg/persistance"
)

type bookingDatabase struct {
	persistance.DbActions
}

func NewBookingsDb(a *sql.DB) bookingDatabase {
	return bookingDatabase{DbActions: a}
}

func (bdb bookingDatabase) Get(ctx context.Context, br domain.BookingRequest) domain.BookingResponse {
	query := `SELECT  start_date, end_date, user_ref 
FROM bookings 
WHERE booking_reference = $1`

	var (
		stime time.Time
		etime sql.NullTime
		uRef  uuid.UUID
	)

	bdb.QueryRowContext(ctx, query, br.BookingReference).Scan(&stime, &etime, &uRef)
	var et *time.Time = nil
	if etime.Valid {
		et = &etime.Time
	}

	response := domain.NewBookingResponse(br.BookingReference, stime, et, uRef)

	return response
}
