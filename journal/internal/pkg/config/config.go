package config

import (
	"fmt"
	"os"
)

var (
	databaseUrl string
	apiPort     string
	ginMode     string
)

func Init() error {
	ok := false
	databaseUrl, ok = os.LookupEnv("DATABASE_URL")
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
