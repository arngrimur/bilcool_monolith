package postgresql

import (
	"context"
	"database/sql"
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
	query := `SELECT  start_date, end_date, user_ref 
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
