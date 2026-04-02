package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	databaseUrl         string
	jwtSecret           string
	webAuthnRPID        string
	webAuthnRPOrigins   []string
	webAuthnDisplayName string
	sesFromEmail        string
}

func (c Config) DatabaseUrl() string {
	return c.databaseUrl
}

func (c Config) JWTSecret() string {
	return c.jwtSecret
}

func (c Config) WebAuthnRPID() string {
	return c.webAuthnRPID
}

func (c Config) WebAuthnRPOrigins() []string {
	return c.webAuthnRPOrigins
}

func (c Config) WebAuthnDisplayName() string {
	return c.webAuthnDisplayName
}

func (c Config) SESFromEmail() string {
	return c.sesFromEmail
}

func Init() (Config, error) {
	c := Config{}
	var missing []string

	if v, ok := os.LookupEnv("DATABASE_URL"); ok {
		c.databaseUrl = v
	} else {
		missing = append(missing, "DATABASE_URL")
	}
	if v, ok := os.LookupEnv("JWT_SECRET"); ok {
		c.jwtSecret = v
	} else {
		missing = append(missing, "JWT_SECRET")
	}
	if v, ok := os.LookupEnv("WEBAUTHN_RP_ID"); ok {
		c.webAuthnRPID = v
	} else {
		missing = append(missing, "WEBAUTHN_RP_ID")
	}
	if v, ok := os.LookupEnv("WEBAUTHN_RP_ORIGINS"); ok {
		c.webAuthnRPOrigins = strings.Split(v, ",")
	} else {
		missing = append(missing, "WEBAUTHN_RP_ORIGINS")
	}
	if v, ok := os.LookupEnv("WEBAUTHN_DISPLAY_NAME"); ok {
		c.webAuthnDisplayName = v
	} else {
		c.webAuthnDisplayName = "BilCool"
	}
	if v, ok := os.LookupEnv("SES_FROM_EMAIL"); ok {
		c.sesFromEmail = v
	} else {
		missing = append(missing, "SES_FROM_EMAIL")
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return c, nil
}
