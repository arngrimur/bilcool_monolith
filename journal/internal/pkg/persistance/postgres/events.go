package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	authdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

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

func (r EventRepository) SaveMessage(ctx context.Context, e postgres.Message) error {
	txr, tx, err := r.createTransaction(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := `INSERT INTO inbox (event_id, type, correlation_id, producer, emitted_at) VALUES ($1, $2, $3, $4, $5)`
	_, err = txr.ExecContext(
		ctx,
		q,
		e.Message.EventId.String(),
		e.Type,
		e.Message.CorrelationId,
		e.Message.Producer,
		e.Message.EmittedAt,
	)
	if err != nil {
		log.Ctx(ctx).Debug().Err(err).Msg("failed to save message")
		return err
	}

	switch e.Message.Type {
	case authdomain.EventUserCreated:
		err = r.handleUserCreated(ctx, txr, e.Message.Payload)
	case authdomain.EventUserDeleted:
		err = r.handleUserDeleted(ctx, txr, e.Message.Payload)
	case "booking_ended":
		err = r.handleBookingEnded(ctx, txr, e.Message.Payload)
	default:
		log.Ctx(ctx).Debug().Str("type", e.Message.Type).Msg("unhandled event type, skipping")
	}

	if err != nil {
		return err
	}

	tx.Commit()
	return nil
}

func (r EventRepository) handleUserCreated(ctx context.Context, txr EventRepository, payload json.RawMessage) error {
	user := authdomain.UserResponse{}
	if err := json.Unmarshal(payload, &user); err != nil {
		return err
	}
	_, err := txr.ExecContext(ctx,
		`INSERT INTO users (user_ref, created_at) VALUES ($1, NOW()) ON CONFLICT (user_ref) DO NOTHING`,
		user.UserRef,
	)
	return err
}

func (r EventRepository) handleUserDeleted(ctx context.Context, txr EventRepository, payload json.RawMessage) error {
	user := authdomain.UserResponse{}
	if err := json.Unmarshal(payload, &user); err != nil {
		return err
	}
	_, err := txr.ExecContext(ctx,
		`UPDATE users SET deleted_at = NOW() WHERE user_ref = $1 AND deleted_at IS NULL`,
		user.UserRef,
	)
	return err
}

func (r EventRepository) handleBookingEnded(ctx context.Context, txr EventRepository, payload json.RawMessage) error {
	completedBooking := domain.CompletedBooking{}
	if err := json.Unmarshal(payload, &completedBooking); err != nil {
		return err
	}

	var userId int64
	err := txr.QueryRowContext(ctx, `SELECT id FROM users WHERE user_ref = $1`, completedBooking.Booking.UserRef).Scan(&userId)
	if err != nil {
		return err
	}

	q := `INSERT INTO booking_ended_events (fk_user, booking_ref, start_date, end_date, distance_meters) VALUES ($1, $2, $3, $4, $5)`
	_, err = txr.ExecContext(ctx, q, userId, completedBooking.Booking.BookingReference, completedBooking.Booking.StartDate, completedBooking.Booking.EndDate, completedBooking.Distance.Distance())
	return err
}

func (r EventRepository) GetFinishedBookings(ctx context.Context) ([]FinishedBooking, error) {
	const query = `
SELECT b.booking_ref, u.user_ref, b.start_date, b.end_date, b.distance_meters
FROM booking_ended_events b
INNER JOIN users u ON b.fk_user = u.id
ORDER BY b.start_date DESC`

	rows, err := r.QueryContext(ctx, query)
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
