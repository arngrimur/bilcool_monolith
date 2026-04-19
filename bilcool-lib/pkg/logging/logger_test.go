package logging

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultLogger(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 500))
	log.Logger = zerolog.New(buf)

	ctx := context.Background()
	ctx = NewDefaultLogger(ctx)
	log.Ctx(ctx).Info().Msg("test")
	require.Contains(t, buf.String(), "caller")
	require.Contains(t, buf.String(), "time")
}

func TestNewDefaultLoggerWithService(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 500))
	log.Logger = zerolog.New(buf)

	ctx := context.Background()
	ctx = NewDefaultLogger(ctx, WithService("tester"))
	log.Ctx(ctx).Info().Msg("test")
	require.Contains(t, buf.String(), "caller")
	require.Contains(t, buf.String(), "time")
	require.Contains(t, buf.String(), "service")
	require.Contains(t, buf.String(), "tester")
}
