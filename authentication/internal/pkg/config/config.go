package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/arngrimur/bilcool-lib/pkg/secrets"
)

// service is this service's name in secret keys (bilcool-authentication-<name>).
const service = "authentication"

const (
	MailSenderBrevo = "brevo"
	MailSenderLog   = "log"
)

var (
	databaseUrl         string
	jwtSecret           string
	webAuthnRPID        string
	webAuthnRPOrigins   []string
	webAuthnDisplayName string
	fromEmail           string
	brevoAPIKey         string
	mailSender          string
	outboxMode          string
	ginMode             string
)

func DatabaseUrl() string {
	return databaseUrl
}

func JWTSecret() string {
	return jwtSecret
}

func WebAuthnRPID() string {
	return webAuthnRPID
}

func WebAuthnRPOrigins() []string {
	return webAuthnRPOrigins
}

func WebAuthnDisplayName() string {
	return webAuthnDisplayName
}

func Init() error {
	store, ctx, cancel, err := newSecretStore()
	if err != nil {
		return err
	}
	defer cancel()
	var missing []string

	if v, ok, err := readSecret(ctx, store, "database-url"); err != nil {
		return err
	} else if ok {
		databaseUrl = v
	} else {
		missing = append(missing, "DATABASE_URL")
	}
	if v, ok, err := readSecret(ctx, store, "jwt-secret"); err != nil {
		return err
	} else if ok {
		jwtSecret = v
	} else {
		missing = append(missing, "JWT_SECRET")
	}
	if v, ok := os.LookupEnv("WEBAUTHN_RP_ID"); ok {
		webAuthnRPID = v
	} else {
		missing = append(missing, "WEBAUTHN_RP_ID")
	}
	if v, ok := os.LookupEnv("WEBAUTHN_RP_ORIGINS"); ok {
		webAuthnRPOrigins = strings.Split(v, ",")
	} else {
		missing = append(missing, "WEBAUTHN_RP_ORIGINS")
	}
	if v, ok := os.LookupEnv("WEBAUTHN_DISPLAY_NAME"); ok {
		webAuthnDisplayName = v
	} else {
		webAuthnDisplayName = "BilCool"
	}
	if v, ok := os.LookupEnv("FROM_EMAIL"); ok {
		fromEmail = v
	} else {
		missing = append(missing, "FROM_EMAIL")
	}
	mailSender = MailSenderBrevo
	if v, ok := os.LookupEnv("MAIL_SENDER"); ok && v != "" {
		mailSender = v
	}
	switch mailSender {
	case MailSenderBrevo:
		if v, ok, err := readSecret(ctx, store, "brevo-api-key"); err != nil {
			return err
		} else if ok {
			brevoAPIKey = v
		} else {
			missing = append(missing, "BREVO_API_KEY")
		}
	case MailSenderLog:
		// Development only: the security token is logged instead of mailed, no Brevo key needed.
	default:
		return fmt.Errorf("unknown MAIL_SENDER %q (use %q or %q)", mailSender, MailSenderBrevo, MailSenderLog)
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
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
	return nil
}

func GinMode() string {
	return ginMode
}

func OutboxMode() string {
	return outboxMode
}

func FromSenderEmail() string {
	return fromEmail
}

// MailSender is "brevo" (default) or "log" (development only: the token is written to the log).
func MailSender() string {
	return mailSender
}

func BrevoAPIKey() string {
	return brevoAPIKey
}

func AwsSecret() string {
	panic("not implemented")
}

func AwsKey() string {
	panic("not implemented")
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
