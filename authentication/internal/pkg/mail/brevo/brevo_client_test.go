package brevo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSendingOkMessage(t *testing.T) {
	t.Skipf("We don't want ting to send emails in tests and create spam")
	client := NewBrevoClient()
	err := client.SendSecurityToken(context.Background(), "arngrimurbjarnason@gmail.com", "123456")
	require.NoError(t, err)
}
