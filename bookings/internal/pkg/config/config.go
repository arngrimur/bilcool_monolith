package config

import (
	"fmt"
	"os"
)

type Config struct {
	databaseUrl string
}

func (c Config) DatabaseUrl() string {
	return c.databaseUrl
}

func Init() (Config, error) {
	ok := true
	c := Config{}
	c.databaseUrl, ok = os.LookupEnv("DATABASE_URL")
	if !ok {
		return c, fmt.Errorf("DATABASE_URL not set")
	}
	return c, nil
}
