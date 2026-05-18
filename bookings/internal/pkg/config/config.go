package config

import (
	"fmt"
	"os"
)

var (
	databaseUrl        string
	outboxMode         string
	ginMode            string
	mapboxAccessToken  string
)

func DatabaseUrl() string {
	return databaseUrl
}

func OutboxMode() string {
	return outboxMode
}

func Init() error {
	ok := true
	databaseUrl, ok = os.LookupEnv("DATABASE_URL")
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
	mapboxAccessToken, _ = os.LookupEnv("MAPBOX_ACCESS_TOKEN")
	return nil
}

func GinMode() string {
	return ginMode
}

func MapboxAccessToken() string {
	return mapboxAccessToken
}
