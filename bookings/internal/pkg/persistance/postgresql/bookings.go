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
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	outbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type BookingRepository struct {
	DbActions
	Transactioner
}

func NewBookingsRepository(a *sql.DB) BookingRepository {
	return BookingRepository{DbActions: a, Transactioner: a}
}

func (bdb BookingRepository) Find(ctx context.Context, request domain.BookingRequest) (extdomain.BookingResponse, error) {
	const query = `SELECT start_date, end_date, userref
FROM bookings_and_users
WHERE booking_reference = $1`

	var (
		sTime time.Time
		eTime time.Time
		uRef  uuid.UUID
	)

	err := bdb.QueryRowContext(ctx, query, request.BookingReference).Scan(&sTime, &eTime, &uRef)
	if err != nil {
		return extdomain.BookingResponse{}, err
	}

	response := domain.NewBookingResponse(request.BookingReference, sTime, eTime, uRef, nil)

	return response, err
}

func (bdb BookingRepository) FindAll(ctx context.Context) ([]extdomain.BookingResponse, error) {
	const query = `SELECT b.booking_reference, b.start_date, b.end_date, u.userref,
       d.start_distance, d.end_distance
FROM bookings b
INNER JOIN users u ON b.user_ref = u.id
LEFT JOIN distances d ON d.fk_booking_id = b.id`
	bookings := []extdomain.BookingResponse{}

	rows, err := bdb.QueryContext(ctx, query)
	if err != nil {
		return bookings, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var (
			bookingRef    uuid.UUID
			sTime         time.Time
			eTime         time.Time
			uRef          uuid.UUID
			startDistance sql.NullInt64
			endDistance   sql.NullInt64
		)
		err = rows.Scan(&bookingRef, &sTime, &eTime, &uRef, &startDistance, &endDistance)
		if err != nil {
			return nil, err
		}
		var dist *extdomain.Distance
		if startDistance.Valid {
			dist = &extdomain.Distance{
				StartDistance: int(startDistance.Int64),
				EndDistance:   int(endDistance.Int64),
			}
		}
		bookings = append(bookings, domain.NewBookingResponse(bookingRef, sTime, eTime, uRef, dist))
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return bookings, nil
}

func (bdb BookingRepository) UpdateBooking(ctx context.Context, request domain.UpdateBookingRequest) error {
	var userExists bool
	err := bdb.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE userref = $1 AND deleted = false)`, request.UserRef).Scan(&userExists)
	if err != nil {
		return err
	}
	if !userExists {
		return domain.ErrUserNotFound
	}

	const query = `
WITH overlap AS (
    SELECT EXISTS (
        SELECT 1 FROM bookings
        WHERE booking_reference <> $1 AND start_date < $3 AND end_date > $2
    ) AS has_overlap
),
uid AS (
    SELECT id FROM users WHERE userref = $4 AND deleted = false
)
INSERT INTO bookings (booking_reference, start_date, end_date, user_ref)
SELECT $1, $2, $3, uid.id FROM uid WHERE NOT (SELECT has_overlap FROM overlap)
ON CONFLICT (booking_reference) DO UPDATE
SET start_date = EXCLUDED.start_date, end_date = EXCLUDED.end_date
WHERE NOT (SELECT has_overlap FROM overlap)
RETURNING booking_reference`

	var ref uuid.UUID
	err = bdb.QueryRowContext(ctx, query, request.BookingReference, request.StartDate, request.EndDate, request.UserRef).Scan(&ref)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("booking overlaps with an existing booking")
	}
	return err
}

func (bdb BookingRepository) DeleteBooking(ctx context.Context, request domain.BookingRequest) error {
	const query = `DELETE FROM bookings WHERE booking_reference = $1 AND start_date > NOW()`
	result, err := bdb.ExecContext(ctx, query, request.BookingReference)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		var started bool
		err = bdb.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM bookings WHERE booking_reference = $1 AND start_date <= NOW())`, request.BookingReference).Scan(&started)
		if err != nil {
			return err
		}
		if started {
			return domain.ErrBookingAlreadyStarted
		}
		return fmt.Errorf("no booking found with reference %s", request.BookingReference)
	}
	return nil
}

func (bdb BookingRepository) EndBooking(ctx context.Context, request domain.EndBookingRequest) error {
	local_bdb, tx, err := bdb.createTransaction(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	type tmpData struct {
		id        int
		completed extdomain.CompletedBooking
	} // Get booking
	booking := &tmpData{}
	err = local_bdb.QueryRowContext(ctx, "SELECT bookings.id, booking_reference, users.userref, start_date, end_date FROM bookings INNER JOIN users ON bookings.user_ref = users.id WHERE booking_reference = $1", request.BookingReference).
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

func (bdb BookingRepository) AddUser(ctx context.Context, user uuid.UUID, eventID string) error {
	local_bdb, tx, err := bdb.createTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	_, err = local_bdb.ExecContext(ctx, `INSERT INTO inbox (message_id) VALUES ($1) ON CONFLICT DO NOTHING`, eventID)
	if err != nil {
		return err
	}
	_, err = local_bdb.ExecContext(ctx, `INSERT INTO users (userref) VALUES ($1) ON CONFLICT DO NOTHING`, user)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (bdb BookingRepository) DeleteUser(ctx context.Context, user uuid.UUID, eventID string) error {
	local_bdb, tx, err := bdb.createTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	_, err = local_bdb.ExecContext(ctx, `UPDATE users SET deleted = true, deleted_at = NOW() WHERE userref = $1 AND deleted = false`, user)
	if err != nil {
		return err
	}
	_, err = local_bdb.ExecContext(ctx, `INSERT INTO inbox (message_id) VALUES ($1)`, eventID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
