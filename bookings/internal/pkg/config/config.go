package config

import (
	"fmt"
	"os"
)

var (
	databaseUrl string
	outboxMode  string
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
	return nil
}
