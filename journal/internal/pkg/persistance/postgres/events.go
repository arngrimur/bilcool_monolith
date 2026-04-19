package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	authdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

var ErrDuplicateEvent = errors.New("duplicate event")

type FinishedBooking struct {
	BookingRef     uuid.UUID `json:"booking_reference"`
	UserRef        uuid.UUID `json:"user_ref"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	DistanceMeters int       `json:"distance_meters"`
}

type EventRepository struct {
	DbActions
	Transactioner
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{DbActions: db, Transactioner: db}
}

func (r EventRepository) SaveUserCreated(ctx context.Context, e postgres.Message) error {
	txr, tx, err := r.createTransaction(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = txr.saveInbox(ctx, e); err != nil {
		if errors.Is(err, ErrDuplicateEvent) {
			return nil
		}
		return err
	}
	if err = txr.handleUserCreated(ctx, e.Message.Payload); err != nil {
		return err
	}
	return tx.Commit()
}

func (r EventRepository) SaveUserDeleted(ctx context.Context, e postgres.Message) error {
	txr, tx, err := r.createTransaction(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = txr.saveInbox(ctx, e); err != nil {
		if errors.Is(err, ErrDuplicateEvent) {
			return nil
		}
		return err
	}
	if err = txr.handleUserDeleted(ctx, e.Message.Payload); err != nil {
		return err
	}
	return tx.Commit()
}

func (r EventRepository) SaveBookingEnded(ctx context.Context, e postgres.Message) error {
	txr, tx, err := r.createTransaction(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = txr.saveInbox(ctx, e); err != nil {
		if errors.Is(err, ErrDuplicateEvent) {
			return nil
		}
		return err
	}
	if err = txr.handleBookingEnded(ctx, e.Message.Payload); err != nil {
		return err
	}
	return tx.Commit()
}

func (r EventRepository) saveInbox(ctx context.Context, e postgres.Message) error {
	q := `INSERT INTO inbox (event_id, type, correlation_id, producer, emitted_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (event_id) DO NOTHING`
	result, err := r.ExecContext(ctx, q, e.Message.EventId.String(), e.Message.Type, e.Message.CorrelationId, e.Message.Producer, e.Message.EmittedAt)
	if err != nil {
		log.Ctx(ctx).Debug().Err(err).Msg("failed to save message")
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDuplicateEvent
	}
	return nil
}

func (r EventRepository) handleUserCreated(ctx context.Context, payload json.RawMessage) error {
	user := authdomain.UserResponse{}
	if err := json.Unmarshal(payload, &user); err != nil {
		return err
	}
	_, err := r.ExecContext(ctx,
		`INSERT INTO users (user_ref, created_at) VALUES ($1, NOW()) ON CONFLICT (user_ref) DO NOTHING`,
		user.UserRef,
	)
	return err
}

func (r EventRepository) handleUserDeleted(ctx context.Context, payload json.RawMessage) error {
	user := authdomain.UserResponse{}
	if err := json.Unmarshal(payload, &user); err != nil {
		return err
	}
	_, err := r.ExecContext(ctx,
		`UPDATE users SET deleted_at = NOW() WHERE user_ref = $1 AND deleted_at IS NULL`,
		user.UserRef,
	)
	return err
}

func (r EventRepository) handleBookingEnded(ctx context.Context, payload json.RawMessage) error {
	completedBooking := domain.CompletedBooking{}
	if err := json.Unmarshal(payload, &completedBooking); err != nil {
		return err
	}

	var userId int64
	err := r.QueryRowContext(ctx, `SELECT id FROM users WHERE user_ref = $1`, completedBooking.Booking.UserRef).Scan(&userId)
	if err != nil {
		return err
	}

	q := `INSERT INTO booking_ended_events (fk_user, booking_ref, start_date, end_date, distance_meters) VALUES ($1, $2, $3, $4, $5)`
	_, err = r.ExecContext(ctx, q, userId, completedBooking.Booking.BookingReference, completedBooking.Booking.StartDate, completedBooking.Booking.EndDate, completedBooking.Distance.Distance())
	return err
}

type FinishedBookingFilter struct {
	Year    *int
	Month   *int
	UserRef *uuid.UUID
}

func (r EventRepository) GetFinishedBookings(ctx context.Context, f FinishedBookingFilter) ([]FinishedBooking, error) {
	const query = `
SELECT b.booking_ref, u.user_ref, b.start_date, b.end_date, b.distance_meters
FROM booking_ended_events b
INNER JOIN users u ON b.fk_user = u.id
WHERE ($1::int IS NULL OR EXTRACT(YEAR  FROM b.start_date) = $1)
  AND ($2::int IS NULL OR EXTRACT(MONTH FROM b.start_date) = $2)
  AND ($3::uuid IS NULL OR u.user_ref = $3)
ORDER BY b.start_date DESC`

	rows, err := r.QueryContext(ctx, query, f.Year, f.Month, f.UserRef)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]FinishedBooking, 0)
	for rows.Next() {
		var fb FinishedBooking
		if err := rows.Scan(&fb.BookingRef, &fb.UserRef, &fb.StartDate, &fb.EndDate, &fb.DistanceMeters); err != nil {
			return nil, err
		}
		bookings = append(bookings, fb)
	}
	return bookings, rows.Err()
}

func (bdb EventRepository) createTransaction(ctx context.Context) (EventRepository, *sql.Tx, error) {
	tx, err := bdb.BeginTx(ctx, nil)
	if err != nil {
		return EventRepository{}, nil, err
	}
	return EventRepository{DbActions: tx, Transactioner: bdb.Transactioner}, tx, nil
}
