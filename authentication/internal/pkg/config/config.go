package config

import (
	"fmt"
	"os"
	"strings"
)

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
	var missing []string

	if v, ok := os.LookupEnv("DATABASE_URL"); ok {
		databaseUrl = v
	} else {
		missing = append(missing, "DATABASE_URL")
	}
	if v, ok := os.LookupEnv("JWT_SECRET"); ok {
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
		if v, ok := os.LookupEnv("BREVO_API_KEY"); ok {
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
