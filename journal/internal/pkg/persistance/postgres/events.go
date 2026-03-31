package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

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
		e.MessageId,
		e.Type,
		e.Message.CorrelationId,
		e.Message.Producer,
		e.Message.EmittedAt,
	)
	if err != nil {
		return err
	}

	completedBooking := domain.CompletedBooking{}
	err = json.Unmarshal(e.Message.Payload, &completedBooking)
	if err != nil {
		return err
	}

	var userId int64
	err = txr.QueryRowContext(ctx, `SELECT id FROM users WHERE user_ref = $1`, completedBooking.Booking.UserRef).Scan(&userId)
	if err != nil {
		return err
	}

	q = `INSERT INTO booking_ended_events (fk_user, booking_ref, start_date, end_date, distance_meters) VALUES ($1, $2, $3, $4, $5)`
	_, err = txr.ExecContext(ctx, q, userId, completedBooking.Booking.BookingReference, completedBooking.Booking.StartDate, completedBooking.Booking.EndDate, completedBooking.Distance.Distance())
	if err != nil {
		return err
	}
	tx.Commit()
	return nil
}

func (bdb EventRepository) createTransaction(ctx context.Context) (EventRepository, *sql.Tx, error) {
	tx, err := bdb.BeginTx(ctx, nil)
	if err != nil {
		return EventRepository{}, nil, err
	}
	return EventRepository{DbActions: tx, Transactioner: bdb.Transactioner}, tx, nil
}
