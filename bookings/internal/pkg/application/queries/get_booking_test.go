package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

func TestGetTrackPoints_ReturnsPoints(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	expected := []extdomain.TrackPoint{{Lat: 59.33, Lon: 18.06, RecordedAt: time.Now().UTC()}}
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(expected, nil)

	handler := NewGetBookingsHandler(domain.NewBookings(repo))
	got, err := handler.GetTrackPoints(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestGetTrackPoints_NoPointsReturnsEmptySlice(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(nil, nil)

	handler := NewGetBookingsHandler(domain.NewBookings(repo))
	got, err := handler.GetTrackPoints(context.Background(), request)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestGetTrackPoints_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	boom := errors.New("boom")
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(nil, boom)

	handler := NewGetBookingsHandler(domain.NewBookings(repo))
	_, err := handler.GetTrackPoints(context.Background(), request)

	require.ErrorIs(t, err, boom)
}
