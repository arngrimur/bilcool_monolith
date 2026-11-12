package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/persistance/postgresql"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/messaging/provider"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/poller"
	snspublisher "github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/sns"
)

func main() {
	ctx := logging.NewDefaultLogger(context.Background(), logging.WithService("bookings-outbox"))
	log.Ctx(ctx).Info().Msg("starting bookings outbox lambda")

	if err := config.Init(); err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	publisher, err := provider.NewPublisher(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating publisher")
	}

	db := postgresql.SetupPostgresDatabase()
	p := poller.New(db, publisher, snspublisher.TopicBookings)

	lambda.Start(func(ctx context.Context) error {
		return p.Poll(ctx)
	})
}
