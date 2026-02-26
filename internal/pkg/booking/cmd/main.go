package main

import (
	"database/sql"

	"github.com/rs/zerolog/log"

	"bilcool_monolith/internal/pkg/booking/internal/pkg/application"
	"bilcool_monolith/internal/pkg/booking/internal/pkg/config"
	"bilcool_monolith/internal/pkg/booking/internal/pkg/persistance/postgresql"
)

func main() {
	// TODO: setup context for the application
	// Read Config
	c, err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading config")
	}
	// Create Db Connection
	psqlDb, err := sql.Open("postgres", c.DatabaseUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Error opening database connection")
	}
	// Create Application
	app := application.New(postgresql.NewBookingsDb(psqlDb))
	_ = app
}
