package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

// stubQueries implements application.Queries returning zero values.
type stubQueries struct{}

func (s *stubQueries) GetBooking(_ context.Context, _ domain.BookingRequest) (extdomain.BookingResponse, error) {
	return extdomain.BookingResponse{}, nil
}
func (s *stubQueries) GetAllBooking(_ context.Context) ([]extdomain.BookingResponse, error) {
	return nil, nil
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
