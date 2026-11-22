package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/arngrimur/bilcool-lib/pkg/secrets"
)

// service is this service's name in secret keys (bilcool-bookings-<name>).
const service = "bookings"

var (
	databaseUrl       string
	outboxMode        string
	ginMode           string
	mapboxAccessToken string
)

func DatabaseUrl() string {
	return databaseUrl
}

func OutboxMode() string {
	return outboxMode
}

func Init() error {
	store, ctx, cancel, err := newSecretStore()
	if err != nil {
		return err
	}
	defer cancel()
	var ok bool
	databaseUrl, ok, err = readSecret(ctx, store, "database-url")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("DATABASE_URL not set")
	}
	if mode, set := os.LookupEnv("OUTBOX_MODE"); set {
		outboxMode = mode
	} else {
		outboxMode = "replication"
	}
	if _, set := os.LookupEnv("RELEASE"); set {
		ginMode = "release"
	} else {
		ginMode = "debug"
	}
	// Optional: without it bookings returns only the raw GPS track.
	if mapboxAccessToken, _, err = readSecret(ctx, store, "mapbox-access-token"); err != nil {
		return err
	}
	return nil
}

// readSecret returns the secret, or ok=false when the provider has no value for it. Any other error is returned.
func readSecret(ctx context.Context, store secrets.Provider, name string) (value string, ok bool, err error) {
	value, err = store.Get(ctx, secrets.Key{Service: service, Name: name})
	if errors.Is(err, secrets.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func newSecretStore() (secrets.Provider, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	store, err := secrets.New(ctx)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return store, ctx, cancel, nil
}

func GinMode() string {
	return ginMode
}

func MapboxAccessToken() string {
	return mapboxAccessToken
}
