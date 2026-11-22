package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/arngrimur/bilcool-lib/pkg/secrets"
)

// service is this service's name in secret keys (bilcool-journal-<name>).
const service = "journal"

var (
	databaseUrl string
	apiPort     string
	ginMode     string
)

func Init() error {
	store, ctx, cancel, err := newSecretStore()
	if err != nil {
		return err
	}
	defer cancel()
	ok := false
	databaseUrl, ok, err = readSecret(ctx, store, "database-url")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("DATABASE_URL not set")
	}

	apiPort, ok = os.LookupEnv("API_PORT")
	if !ok {
		apiPort = ":8080"
	}
	if _, set := os.LookupEnv("RELEASE"); set {
		ginMode = "release"
	} else {
		ginMode = "debug"
	}
	return nil
}

func GinMode() string {
	return ginMode
}

func DatabaseUrl() string {
	return databaseUrl
}

func APIPort() string {
	return apiPort
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
