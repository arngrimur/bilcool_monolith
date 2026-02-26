package persistance

import (
	"context"

	"bilcool_monolith/internal/pkg/booking/internal/pkg/domain"
)

type BookingsRepository interface {
	Find(ctx context.Context, request domain.BookingRequest) (domain.BookingResponse, error)
}
