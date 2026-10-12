package logsender

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestSendSecurityTokenLogsTokenAndSucceeds(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	ctx := logger.WithContext(context.Background())

	err := NewSender().SendSecurityToken(ctx, "user@example.com", "123456", "en")

	require.NoError(t, err)
	require.Contains(t, buf.String(), "123456")
	require.Contains(t, buf.String(), "user@example.com")
}
