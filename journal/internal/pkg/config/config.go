package config

import (
	"fmt"
	"os"
)

var databaseUrl string

func Init() error {
	ok := false
	databaseUrl, ok = os.LookupEnv("DATABASE_URL")

	if !ok {
		return fmt.Errorf("DATABASE_URL not set")
	}
	return nil
}

func DatabaseUrl() string {
	return databaseUrl
}
