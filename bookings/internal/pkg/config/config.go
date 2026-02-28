package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseUrl string
}

func Init() (Config, error) {
	ok := true
	c := Config{}
	c.DatabaseUrl, ok = os.LookupEnv("DATABASE_URL")
	if !ok {
		return c, fmt.Errorf("DATABASE_URL not set")
	}
	return c, nil
}
