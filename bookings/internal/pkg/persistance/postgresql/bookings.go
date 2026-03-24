package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	outbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type BookingRepository struct {
	DbActions
	Transactioner
}

func NewBookingsRepository(a *sql.DB) BookingRepository {
	return BookingRepository{DbActions: a, Transactioner: a}
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
		bookings   = []domain.BookingResponse{}
		sTime      time.Time
		eTime      time.Time
		uRef       uuid.UUID
		bookingRef uuid.UUID
	)

	rows, err := bdb.QueryContext(ctx, query)
	if err != nil {
		return bookings, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&bookingRef, &sTime, &eTime, &uRef)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, domain.NewBookingResponse(bookingRef, sTime, eTime, uRef))
	}
	if err = rows.Err(); err != nil {
		return nil, err
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

func (bdb BookingRepository) EndBooking(ctx context.Context, request domain.EndBookingRequest) error {
	local_bdb, tx, err := bdb.createTransaction(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()
	type tmpData struct {
		id        int
		completed domain.CompletedBooking
	} // Get booking
	booking := &tmpData{}
	err = local_bdb.QueryRowContext(ctx, "SELECT id, booking_reference, user_ref, start_date, end_date FROM bookings WHERE booking_reference = $1", request.BookingReference).
		Scan(
			&booking.id,
			&booking.completed.Booking.BookingReference,
			&booking.completed.Booking.UserRef,
			&booking.completed.Booking.StartDate,
			&booking.completed.Booking.EndDate,
		)
	if err != nil {
		return err
	}
	q := "INSERT INTO distances (fk_booking_id, start_distance, end_distance) VALUES ($1, $2, $3)"
	_, err = local_bdb.ExecContext(ctx, q, booking.id, request.StartDistance, request.EndDistance)
	if err != nil {
		return err
	}
	// insert msg in outbox
	booking.completed.Distance = request.Distance

	bytes, err := json.Marshal(booking.completed)
	if err != nil {
		return err
	}

	e := outbox.Event{
		EventId:       uuid.New(),
		Type:          "booking_ended",
		CorrelationId: uuid.New(),
		Producer:      "bookings",
		Payload:       bytes,
	}
	err = outbox.Insert(ctx, tx, e)
	if err != nil {
		return err
	}
	// Commit
	return tx.Commit()
}

func (bdb BookingRepository) createTransaction(ctx context.Context) (BookingRepository, *sql.Tx, error) {
	tx, err := bdb.BeginTx(ctx, nil)
	if err != nil {
		return BookingRepository{}, nil, err
	}
	return BookingRepository{DbActions: tx, Transactioner: bdb.Transactioner}, tx, nil
}
