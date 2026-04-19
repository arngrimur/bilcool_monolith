package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
)

func TestAddCorrelationIdToLogger(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 0, 500))
	log.Logger = zerolog.New(buf)

	ctx := context.Background()
	logging.NewDefaultLogger(ctx)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LogWithCorrelationID())

	correlationID := uuid.NewString()
	r.GET("/test", func(c *gin.Context) {
		log.Ctx(c.Request.Context()).Info().Msg("test")
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Correlation-Id", correlationID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, buf.String(), "correlation_id")
	require.Contains(t, buf.String(), correlationID)
}

func TestMissingCorrelationIdReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(LogWithCorrelationID())

	handlerCalled := false
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
	})

	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Correlation-Id header is required")
	require.False(t, handlerCalled)
}
