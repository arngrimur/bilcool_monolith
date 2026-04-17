package inbox

//go:generate mockgen -source=sqs_subscriber.go -destination=sqs_client_mock.go -package=inbox

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	authdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	bookingsdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/persistance/postgres"
	broker "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type sqsClient interface {
	VisibilityTimeout() int
	DeleteMessages(ctx context.Context, messages []broker.Message) (int, error)
	RetrieveMessages(ctx context.Context) ([]broker.Message, error)
}

type EventHandler struct {
	sqsClient sqsClient
	repo      *postgres.EventRepository
}

func NewEventHandler(client sqsClient, repo *postgres.EventRepository) *EventHandler {
	return &EventHandler{
		sqsClient: client,
		repo:      repo,
	}
}

func (e EventHandler) ProcessMessages(ctx context.Context, messages []broker.Message) {
	processedCtx, cancel := context.WithTimeout(ctx, time.Duration(e.sqsClient.VisibilityTimeout()-5)*time.Second)
	defer cancel()
	markedForDeletion := make([]broker.Message, 0)
	log.Ctx(ctx).Info().Int("messages_to_process", len(messages)).Msg("journal processing messages")
	for _, m := range messages {
		var err error
		switch m.Message.Type {
		case authdomain.EventUserCreated:
			err = e.repo.SaveUserCreated(processedCtx, m)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to add user")
				continue
			}
		case authdomain.EventUserDeleted:
			err = e.repo.SaveUserDeleted(processedCtx, m)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to delete user")
				continue
			}
		case bookingsdomain.EventBookingEnded:
			err = e.repo.SaveBookingEnded(processedCtx, m)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to save booking ended event")
				continue
			}
		default:
			continue
		}

		markedForDeletion = append(markedForDeletion, m)
	}

	n, err := e.sqsClient.DeleteMessages(processedCtx, markedForDeletion)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to delete messages")
	}
	log.Info().Int("deleted_messages", n).Send()
}

func (e EventHandler) RetrieveMessages(ctx context.Context) ([]broker.Message, error) {
	return e.sqsClient.RetrieveMessages(ctx)
}
