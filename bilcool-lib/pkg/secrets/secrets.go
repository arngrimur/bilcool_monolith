// Package secrets reads secrets through a provider chosen by the SECRETS_PROVIDER environment variable, so the
// services can switch between secret stores (AWS Secrets Manager, Frostmoln Secrets, plain environment variables)
// without code changes. Every store uses the same keys: bilcool-<service>-<name>.
package secrets

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	// EnvVar names the environment variable that selects the provider.
	EnvVar = "SECRETS_PROVIDER"

	// Env reads each secret from a process environment variable. It is the default.
	Env = "env"
)

// ErrNotFound is returned (wrapped) when a provider has no value for a key.
var ErrNotFound = errors.New("secret not found")

// Key identifies a secret. Its name in a secret store is bilcool-<service>-<name>, identical in every environment.
type Key struct {
	// Service is the owning service, e.g. "bookings" or "event_ledger".
	Service string
	// Name is the secret's name within the service in lowercase kebab case, e.g. "database-url".
	Name string
}

// String returns the store key, e.g. "bilcool-event-ledger-database-url".
func (k Key) String() string {
	return "bilcool-" + strings.ReplaceAll(k.Service, "_", "-") + "-" + k.Name
}

// EnvVar returns the environment variable the env provider reads, e.g. "DATABASE_URL" for name "database-url".
// These are the variable names the services read before the secrets package existed.
func (k Key) EnvVar() string {
	return strings.ToUpper(strings.ReplaceAll(k.Name, "-", "_"))
}

// Provider returns the value of a secret.
type Provider interface {
	// Get returns the secret's value, or an error wrapping ErrNotFound when the provider has no value for it.
	Get(ctx context.Context, key Key) (string, error)
}

// Name returns the selected provider: the value of SECRETS_PROVIDER, or Env when it is unset or empty.
func Name() (string, error) {
	switch name := os.Getenv(EnvVar); name {
	case "", Env:
		return Env, nil
	default:
		return "", fmt.Errorf("%s=%q is not supported (supported: %q)", EnvVar, name, Env)
	}
}

// New creates the selected provider.
func New(ctx context.Context) (Provider, error) {
	name, err := Name()
	if err != nil {
		return nil, err
	}
	switch name {
	case Env:
		return NewEnvProvider(), nil
	default:
		return nil, fmt.Errorf("no secrets provider for %s=%q", EnvVar, name)
	}
}

// EnvProvider reads secrets from process environment variables named by Key.EnvVar.
type EnvProvider struct {
	lookup func(string) (string, bool)
}

// NewEnvProvider returns a provider that reads the process environment.
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{lookup: os.LookupEnv}
}

// Get returns the variable's value. A variable that is set, even to the empty string, counts as present.
func (p *EnvProvider) Get(_ context.Context, key Key) (string, error) {
	v, ok := p.lookup(key.EnvVar())
	if !ok {
		return "", fmt.Errorf("%s (environment variable %s): %w", key, key.EnvVar(), ErrNotFound)
	}
	return v, nil
}
