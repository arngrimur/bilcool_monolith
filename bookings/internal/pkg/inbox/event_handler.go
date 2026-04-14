package inbox

//go:generate mockgen -source=event_handler.go -destination=sqs_client_mock.go -package=inbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	authdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type sqsClient interface {
	VisibilityTimeout() int
	DeleteMessages(ctx context.Context, messages []brokerpostgres.Message) (int, error)
	RetrieveMessages(ctx context.Context) ([]brokerpostgres.Message, error)
}

type EventHandler struct {
	sqsClient sqsClient
	repo      domain.BookingsRepository
}

func NewEventHandler(client sqsClient, repo domain.BookingsRepository) *EventHandler {
	return &EventHandler{
		sqsClient: client,
		repo:      repo,
	}
}

func (e EventHandler) ProcessMessages(ctx context.Context, messages []brokerpostgres.Message) {
	processedCtx, cancel := context.WithTimeout(ctx, time.Duration(e.sqsClient.VisibilityTimeout()-5)*time.Second)
	defer cancel()
	markedForDeletion := make([]brokerpostgres.Message, 0)
	for _, m := range messages {
		switch m.Message.Type {
		case authdomain.EventUserCreated:
			user, err := e.userRef(m)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to parse user event")
				continue
			}
			err = e.repo.AddUser(ctx, user, m.MessageId)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to add user")
				continue
			}
			markedForDeletion = append(markedForDeletion, m)
		case authdomain.EventUserDeleted:
		default:
			continue
		}

	}

	n, err := e.sqsClient.DeleteMessages(processedCtx, markedForDeletion)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to delete messages")
	}
	log.Info().Int("deleted_messages", n).Send()
}
func (e EventHandler) userRef(m brokerpostgres.Message) (uuid.UUID, error) {
	user := authdomain.UserResponse{}
	err := json.Unmarshal(m.Message.Payload, &user)
	if err != nil {
		return uuid.UUID{}, err
	}
	return user.UserRef, nil
}

func (e EventHandler) RetrieveMessages(ctx context.Context) ([]brokerpostgres.Message, error) {
	return e.sqsClient.RetrieveMessages(ctx)
}
