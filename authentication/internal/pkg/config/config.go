package config

import (
	"fmt"
	"os"
	"strings"
)

var (
	databaseUrl         string
	jwtSecret           string
	webAuthnRPID        string
	webAuthnRPOrigins   []string
	webAuthnDisplayName string
	sesFromEmail        string
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

func SESFromEmail() string {
	return sesFromEmail
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
	if v, ok := os.LookupEnv("SES_FROM_EMAIL"); ok {
		sesFromEmail = v
	} else {
		missing = append(missing, "SES_FROM_EMAIL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return nil
}
