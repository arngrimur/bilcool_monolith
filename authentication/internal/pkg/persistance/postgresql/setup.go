package postgresql

import (
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
)

func SetupPostgresDatabase() *sql.DB {
	psqlDb, err := sql.Open("postgres", config.DatabaseUrl())
	if err != nil {
		log.Fatal().Err(err).Msg("error opening database connection")
	}
	maxTries := 10
	for i := 1; i <= maxTries; i++ {
		if err := psqlDb.Ping(); err != nil {
			time.Sleep(1 * time.Second)
			log.Err(err).Msgf("error pinging database, attempt: %d", i)
		} else {
			log.Info().Msg("database connection successful")
			return psqlDb
		}
	}
	log.Fatal().Msgf("error pinging database, gave up after %d attempts", maxTries)
	return nil
}
