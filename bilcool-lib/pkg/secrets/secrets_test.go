package secrets

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKey(t *testing.T) {
	tests := []struct {
		key        Key
		wantString string
		wantEnv    string
	}{
		{Key{Service: "bookings", Name: "database-url"}, "bilcool-bookings-database-url", "DATABASE_URL"},
		{Key{Service: "event_ledger", Name: "database-url"}, "bilcool-event-ledger-database-url", "DATABASE_URL"},
		{Key{Service: "authentication", Name: "jwt-secret"}, "bilcool-authentication-jwt-secret", "JWT_SECRET"},
		{Key{Service: "bookings", Name: "mapbox-access-token"}, "bilcool-bookings-mapbox-access-token", "MAPBOX_ACCESS_TOKEN"},
	}
	for _, tt := range tests {
		t.Run(tt.wantString, func(t *testing.T) {
			require.Equal(t, tt.wantString, tt.key.String())
			require.Equal(t, tt.wantEnv, tt.key.EnvVar())
		})
	}
}

func TestName(t *testing.T) {
	for _, value := range []string{"", "env"} {
		t.Setenv(EnvVar, value)
		name, err := Name()
		require.NoError(t, err)
		require.Equal(t, Env, name)
	}

	t.Setenv(EnvVar, "vault")
	_, err := Name()
	require.ErrorContains(t, err, EnvVar)
	provider, err := New(t.Context())
	require.Error(t, err)
	require.Nil(t, provider)
}

func TestEnvProvider(t *testing.T) {
	key := Key{Service: "bookings", Name: "database-url"}

	t.Setenv("DATABASE_URL", "postgres://example")
	v, err := NewEnvProvider().Get(t.Context(), key)
	require.NoError(t, err)
	require.Equal(t, "postgres://example", v)

	t.Setenv("DATABASE_URL", "")
	v, err = NewEnvProvider().Get(t.Context(), key)
	require.NoError(t, err, "a variable set to the empty string counts as present, as before")
	require.Empty(t, v)
}

func TestEnvProviderNotFound(t *testing.T) {
	p := &EnvProvider{lookup: func(string) (string, bool) { return "", false }}
	_, err := p.Get(t.Context(), Key{Service: "bookings", Name: "database-url"})
	require.True(t, errors.Is(err, ErrNotFound))
	require.ErrorContains(t, err, "bilcool-bookings-database-url")
	require.ErrorContains(t, err, "DATABASE_URL")
	require.NotContains(t, err.Error(), "postgres", "errors must never carry a secret value")
}

func TestNewDefaultsToEnv(t *testing.T) {
	t.Setenv(EnvVar, "")
	p, err := New(t.Context())
	require.NoError(t, err)
	require.IsType(t, &EnvProvider{}, p)
}
