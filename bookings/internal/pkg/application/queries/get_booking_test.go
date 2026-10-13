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

	handler := NewGetBookingsHandler(domain.NewBookings(repo), nil)
	got, err := handler.GetTrackPoints(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestGetTrackPoints_NoPointsReturnsEmptySlice(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(nil, nil)

	handler := NewGetBookingsHandler(domain.NewBookings(repo), nil)
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

	handler := NewGetBookingsHandler(domain.NewBookings(repo), nil)
	_, err := handler.GetTrackPoints(context.Background(), request)

	require.ErrorIs(t, err, boom)
}

type stubRouteMatcher struct {
	lines [][]extdomain.Position
	err   error
	calls int
}

func (s *stubRouteMatcher) MatchRoute(_ context.Context, _ []extdomain.TrackPoint) ([][]extdomain.Position, error) {
	s.calls++
	return s.lines, s.err
}

func twoPoints() []extdomain.TrackPoint {
	now := time.Now().UTC()
	return []extdomain.TrackPoint{
		{Lat: 59.33, Lon: 18.06, RecordedAt: now},
		{Lat: 59.34, Lon: 18.07, RecordedAt: now.Add(2 * time.Minute)},
	}
}

func TestGetRoute_ReturnsPointsAndSnappedRoute(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	points := twoPoints()
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(points, nil)
	snapped := [][]extdomain.Position{{{Lat: 59.331, Lon: 18.061}, {Lat: 59.339, Lon: 18.069}}}
	matcher := &stubRouteMatcher{lines: snapped}

	handler := NewGetBookingsHandler(domain.NewBookings(repo), matcher)
	got, err := handler.GetRoute(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, points, got.Points)
	require.Equal(t, snapped, got.Route)
}

func TestGetRoute_MatcherErrorFallsBackToPointsOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	points := twoPoints()
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(points, nil)
	matcher := &stubRouteMatcher{err: errors.New("mapbox down")}

	handler := NewGetBookingsHandler(domain.NewBookings(repo), matcher)
	got, err := handler.GetRoute(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, points, got.Points)
	require.NotNil(t, got.Route)
	require.Empty(t, got.Route)
}

func TestGetRoute_TooFewPointsSkipsMatching(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(twoPoints()[:1], nil)
	matcher := &stubRouteMatcher{}

	handler := NewGetBookingsHandler(domain.NewBookings(repo), matcher)
	got, err := handler.GetRoute(context.Background(), request)

	require.NoError(t, err)
	require.Len(t, got.Points, 1)
	require.Empty(t, got.Route)
	require.Zero(t, matcher.calls)
}

func TestGetRoute_NoPointsReturnsEmptySlices(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(nil, nil)

	handler := NewGetBookingsHandler(domain.NewBookings(repo), &stubRouteMatcher{})
	got, err := handler.GetRoute(context.Background(), request)

	require.NoError(t, err)
	require.NotNil(t, got.Points)
	require.NotNil(t, got.Route)
}

func TestGetRoute_WithoutMatcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(twoPoints(), nil)

	handler := NewGetBookingsHandler(domain.NewBookings(repo), nil)
	got, err := handler.GetRoute(context.Background(), request)

	require.NoError(t, err)
	require.Len(t, got.Points, 2)
	require.Empty(t, got.Route)
}

func TestGetRoute_TrackPointsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockBookingsRepository(ctrl)
	request := domain.BookingRequest{BookingReference: uuid.New()}
	boom := errors.New("boom")
	repo.EXPECT().GetTrackPoints(gomock.Any(), request).Return(nil, boom)
	matcher := &stubRouteMatcher{}

	handler := NewGetBookingsHandler(domain.NewBookings(repo), matcher)
	_, err := handler.GetRoute(context.Background(), request)

	require.ErrorIs(t, err, boom)
	require.Zero(t, matcher.calls)
}
