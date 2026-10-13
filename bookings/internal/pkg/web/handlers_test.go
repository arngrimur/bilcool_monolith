package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// correlationID is a fixed UUID sent as the Correlation-Id header required by the middleware.
const correlationID = "11111111-1111-1111-1111-111111111111"

func addCorrelationID(req *http.Request) *http.Request {
	req.Header.Set("Correlation-Id", correlationID)
	return req
}

// stubCommands implements application.Commands with configurable return values.
type stubCommands struct {
	pauseErr   error
	resumeResp domain.PauseBookingResponse
	resumeErr  error
}

func (s *stubCommands) UpdateBooking(_ context.Context, _ domain.UpdateBookingRequest) error {
	return nil
}
func (s *stubCommands) DeleteBooking(_ context.Context, _ domain.BookingRequest) error {
	return nil
}
func (s *stubCommands) EndBooking(_ context.Context, _ domain.EndBookingRequest) error {
	return nil
}
func (s *stubCommands) PauseBooking(_ context.Context, _ domain.PauseBookingRequest) error {
	return s.pauseErr
}
func (s *stubCommands) ResumeBooking(_ context.Context, _ domain.BookingRequest) (domain.PauseBookingResponse, error) {
	return s.resumeResp, s.resumeErr
}
func (s *stubCommands) AddTrackPoints(_ context.Context, _ domain.AddTrackPointsRequest) error {
	return nil
}

// stubQueries implements application.Queries returning configurable track points.
type stubQueries struct {
	points    []extdomain.TrackPoint
	pointsErr error
	route     extdomain.RouteResponse
	routeErr  error
}

func (s *stubQueries) GetBooking(_ context.Context, _ domain.BookingRequest) (extdomain.BookingResponse, error) {
	return extdomain.BookingResponse{}, nil
}
func (s *stubQueries) GetAllBooking(_ context.Context) ([]extdomain.BookingResponse, error) {
	return nil, nil
}

func (s *stubQueries) GetTrackPoints(_ context.Context, _ domain.BookingRequest) ([]extdomain.TrackPoint, error) {
	return s.points, s.pointsErr
}

func (s *stubQueries) GetRoute(_ context.Context, _ domain.BookingRequest) (extdomain.RouteResponse, error) {
	return s.route, s.routeErr
}

func newTestRouter(cmds *stubCommands) *HttpRouter {
	return NewRouter(&stubQueries{}, cmds)
}

// region pauseBooking

func TestPauseBooking_Success(t *testing.T) {
	id := uuid.New()
	router := newTestRouter(&stubCommands{})

	body := `{"lat": 59.334591, "lon": 18.063240}`
	req := addCorrelationID(httptest.NewRequest(http.MethodPost, "/api/v1/bookings/"+id.String()+"/pause", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
}

func TestPauseBooking_AlreadyPaused(t *testing.T) {
	id := uuid.New()
	router := newTestRouter(&stubCommands{pauseErr: domain.ErrBookingAlreadyPaused})

	body := `{"lat": 59.334591, "lon": 18.063240}`
	req := addCorrelationID(httptest.NewRequest(http.MethodPost, "/api/v1/bookings/"+id.String()+"/pause", strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPauseBooking_InvalidID(t *testing.T) {
	router := newTestRouter(&stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodPost, "/api/v1/bookings/not-a-uuid/pause", strings.NewReader(`{"lat":1,"lon":1}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPauseBooking_InvalidBody(t *testing.T) {
	id := uuid.New()
	router := newTestRouter(&stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodPost, "/api/v1/bookings/"+id.String()+"/pause", strings.NewReader("not-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// endregion pauseBooking

// region resumeBooking

func TestResumeBooking_Success(t *testing.T) {
	id := uuid.New()
	expected := domain.PauseBookingResponse{
		Position: extdomain.Position{Lat: 59.334591, Lon: 18.063240},
	}
	router := newTestRouter(&stubCommands{resumeResp: expected})

	req := addCorrelationID(httptest.NewRequest(http.MethodPost, "/api/v1/bookings/"+id.String()+"/resume", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var got domain.PauseBookingResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.InDelta(t, expected.Position.Lat, got.Position.Lat, 0.000001)
	require.InDelta(t, expected.Position.Lon, got.Position.Lon, 0.000001)
}

func TestResumeBooking_NotPaused(t *testing.T) {
	id := uuid.New()
	router := newTestRouter(&stubCommands{resumeErr: domain.ErrBookingNotPaused})

	req := addCorrelationID(httptest.NewRequest(http.MethodPost, "/api/v1/bookings/"+id.String()+"/resume", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResumeBooking_InvalidID(t *testing.T) {
	router := newTestRouter(&stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodPost, "/api/v1/bookings/not-a-uuid/resume", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

// endregion resumeBooking

// region getTrackPoints

func TestGetTrackPoints_Success(t *testing.T) {
	id := uuid.New()
	recorded := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	expected := []extdomain.TrackPoint{
		{Lat: 59.33, Lon: 18.06, RecordedAt: recorded},
		{Lat: 59.34, Lon: 18.07, RecordedAt: recorded.Add(2 * time.Minute)},
	}
	router := NewRouter(&stubQueries{points: expected}, &stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+id.String()+"/track", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var got []extdomain.TrackPoint
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got, 2)
	require.InDelta(t, expected[0].Lat, got[0].Lat, 0.000001)
	require.InDelta(t, expected[1].Lon, got[1].Lon, 0.000001)
	require.True(t, expected[1].RecordedAt.Equal(got[1].RecordedAt))
}

func TestGetTrackPoints_Empty(t *testing.T) {
	id := uuid.New()
	router := NewRouter(&stubQueries{points: []extdomain.TrackPoint{}}, &stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+id.String()+"/track", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, "[]", w.Body.String())
}

func TestGetTrackPoints_InvalidID(t *testing.T) {
	router := newTestRouter(&stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodGet, "/api/v1/bookings/not-a-uuid/track", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetTrackPoints_Error(t *testing.T) {
	id := uuid.New()
	router := NewRouter(&stubQueries{pointsErr: sql.ErrNoRows}, &stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+id.String()+"/track", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

// endregion getTrackPoints

// region getRoute

func TestGetRoute_Success(t *testing.T) {
	id := uuid.New()
	recorded := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	expected := extdomain.RouteResponse{
		Points: []extdomain.TrackPoint{
			{Lat: 59.33, Lon: 18.06, RecordedAt: recorded},
			{Lat: 59.34, Lon: 18.07, RecordedAt: recorded.Add(2 * time.Minute)},
		},
		Route: [][]extdomain.Position{{{Lat: 59.331, Lon: 18.061}, {Lat: 59.339, Lon: 18.069}}},
	}
	router := NewRouter(&stubQueries{route: expected}, &stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+id.String()+"/route", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "private, max-age=300", w.Header().Get("Cache-Control"))

	var got extdomain.RouteResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Len(t, got.Points, 2)
	require.Len(t, got.Route, 1)
	require.InDelta(t, 59.339, got.Route[0][1].Lat, 0.000001)
}

func TestGetRoute_EmptyRouteSerialisesAsArray(t *testing.T) {
	id := uuid.New()
	router := NewRouter(&stubQueries{route: extdomain.RouteResponse{
		Points: []extdomain.TrackPoint{},
		Route:  [][]extdomain.Position{},
	}}, &stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+id.String()+"/route", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"points":[],"route":[]}`, w.Body.String())
}

func TestGetRoute_InvalidID(t *testing.T) {
	router := newTestRouter(&stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodGet, "/api/v1/bookings/not-a-uuid/route", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Empty(t, w.Header().Get("Cache-Control"))
}

func TestGetRoute_Error(t *testing.T) {
	id := uuid.New()
	router := NewRouter(&stubQueries{routeErr: sql.ErrNoRows}, &stubCommands{})

	req := addCorrelationID(httptest.NewRequest(http.MethodGet, "/api/v1/bookings/"+id.String()+"/route", nil))
	w := httptest.NewRecorder()
	router.Engine().ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Empty(t, w.Header().Get("Cache-Control"))
}

// endregion getRoute
