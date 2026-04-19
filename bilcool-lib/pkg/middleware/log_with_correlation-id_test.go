package middleware

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
)

func TestAddCorrelationIdToLogger(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 500))
	log.Logger = zerolog.New(buf)

	ctx := context.Background()
	logging.NewDefaultLogger(ctx)
	r := gin.New()
	r.Use(LogWithCorrelationID())

	correlationID := uuid.NewString()
	r.GET("/test", func(c *gin.Context) {
		log.Ctx(c.Request.Context()).Info().Msg("test")
		require.Contains(t, buf.String(), "correlation-id")
		require.Contains(t, buf.String(), correlationID)
	})

	// Listen and serve on 0.0.0.0:8080
	go func() { _ = r.Run(":8080") }()

	request, err := http.NewRequest("GET", "http://localhost:8080/test", nil)
	require.NoError(t, err)
	request.Header.Add("Correlation-Id", correlationID)
	client := &http.Client{}
	_, err = client.Do(request)
	require.NoError(t, err)
}

func TestNoCorrelationIdFails(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 500))
	log.Logger = zerolog.New(buf)

	ctx := context.Background()
	logging.NewDefaultLogger(ctx)
	r := gin.New()
	r.Use(LogWithCorrelationID())

	noCalls := 0
	r.GET("/test", func(c *gin.Context) {
		noCalls++
	})

	// Listen and serve on 0.0.0.0:8080
	go func() { _ = r.Run(":8080") }()

	request, err := http.NewRequest("GET", "http://localhost:8080/test", nil)
	require.NoError(t, err)
	client := &http.Client{}
	_, err = client.Do(request)
	require.NoError(t, err)
	require.Equal(t, 0, noCalls)

}
