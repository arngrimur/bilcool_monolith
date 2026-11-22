package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/arngrimur/bilcool-lib/pkg/secrets"
)

func setRequired(t *testing.T) {
	t.Setenv(secrets.EnvVar, "")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("WEBAUTHN_RP_ID", "localhost")
	t.Setenv("WEBAUTHN_RP_ORIGINS", "http://localhost:3000")
	t.Setenv("FROM_EMAIL", "a@example.com")
	t.Setenv("MAIL_SENDER", "brevo")
	t.Setenv("BREVO_API_KEY", "brevo")
}

// unset removes variables for the rest of the test; t.Setenv restores them afterwards.
func unset(t *testing.T, names ...string) {
	for _, n := range names {
		t.Setenv(n, "")
		require.NoError(t, os.Unsetenv(n))
	}
}

func TestInitReadsSecretsThroughTheEnvProvider(t *testing.T) {
	setRequired(t)
	require.NoError(t, Init())
	require.Equal(t, "postgres://example", DatabaseUrl())
	require.Equal(t, "jwt", JWTSecret())
}

func TestInitAcceptsSecretsSetToEmpty(t *testing.T) {
	setRequired(t)
	t.Setenv("JWT_SECRET", "")
	require.NoError(t, Init(), "a variable set to the empty string counts as present, as before")
}

func TestInitListsMissingSecretsByVariableName(t *testing.T) {
	setRequired(t)
	unset(t, "JWT_SECRET", "BREVO_API_KEY")
	require.ErrorContains(t, Init(), "missing required env vars: JWT_SECRET, BREVO_API_KEY")
}

func TestInitLogSenderNeedsNoBrevoKey(t *testing.T) {
	setRequired(t)
	t.Setenv("MAIL_SENDER", "log")
	unset(t, "BREVO_API_KEY")
	require.NoError(t, Init())
}

func TestInitRejectsUnknownSecretsProvider(t *testing.T) {
	setRequired(t)
	t.Setenv(secrets.EnvVar, "vault")
	require.ErrorContains(t, Init(), secrets.EnvVar)
}
