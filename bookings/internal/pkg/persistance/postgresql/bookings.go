package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

type BookingRepository struct {
	DbActions
}

func NewBookingsRepository(a *sql.DB) BookingRepository {
	return BookingRepository{DbActions: a}
}

func (bdb BookingRepository) Find(ctx context.Context, request domain.BookingRequest) (domain.BookingResponse, error) {
	const query = `SELECT  start_date, end_date, user_ref 
FROM bookings 
WHERE booking_reference = $1`

	var (
		sTime time.Time
		eTime time.Time
		uRef  uuid.UUID
	)

	err := bdb.QueryRowContext(ctx, query, request.BookingReference).Scan(&sTime, &eTime, &uRef)
	if err != nil {
		return domain.BookingResponse{}, err
	}

	response := domain.NewBookingResponse(request.BookingReference, sTime, eTime, uRef)

	return response, err
}

func (bdb BookingRepository) FindAll(ctx context.Context) ([]domain.BookingResponse, error) {
	const query = `SELECT booking_reference, start_date, end_date, user_ref 
FROM bookings`
	var (
		bookings  = []domain.BookingResponse{}
		sTime     time.Time
		eTime     time.Time
		uRef      uuid.UUID
		bookinRef uuid.UUID
	)

	rows, err := bdb.QueryContext(ctx, query)
	if err != nil {
		return bookings, err
	}
	for rows.Next() {
		err = rows.Scan(&bookinRef, &sTime, &eTime, &uRef)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, domain.NewBookingResponse(bookinRef, sTime, eTime, uRef))
	}

	return bookings, nil
}

func (bdb BookingRepository) UpdateBooking(ctx context.Context, request domain.UpdateBookingRequest) error {
	const query = `
WITH overlap AS (
    SELECT EXISTS (
        SELECT 1 FROM bookings
        WHERE booking_reference <> $1 AND start_date < $3 AND end_date > $2
    ) AS has_overlap
)
INSERT INTO bookings (booking_reference, start_date, end_date, user_ref)
SELECT $1, $2, $3, $4 WHERE NOT (SELECT has_overlap FROM overlap)
ON CONFLICT (booking_reference) DO UPDATE
SET start_date = EXCLUDED.start_date, end_date = EXCLUDED.end_date
WHERE NOT (SELECT has_overlap FROM overlap)
RETURNING booking_reference`

	var ref uuid.UUID
	err := bdb.QueryRowContext(ctx, query, request.BookingReference, request.StartDate, request.EndDate, request.UserRef).Scan(&ref)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("booking overlaps with an existing booking")
	}
	return err
}

func (bdb BookingRepository) DeleteBooking(ctx context.Context, request domain.BookingRequest) error {
	const query = `DELETE FROM bookings WHERE booking_reference = $1`
	result, err := bdb.ExecContext(ctx, query, request.BookingReference)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("no booking found with reference %s", request.BookingReference)
	}
	return nil
}
