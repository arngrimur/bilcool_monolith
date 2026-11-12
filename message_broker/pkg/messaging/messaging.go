// Package messaging defines the provider-neutral contract between the outbox relay and a message broker.
// Concrete implementations live next to their transport (pkg/outbox/sns today); pkg/messaging/provider
// selects one from the MESSAGING_PROVIDER environment variable.
package messaging

import (
	"context"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

//go:generate mockgen -source=messaging.go -destination=messaging_mock.go -package=messaging

// Publisher publishes outbox events to a topic.
type Publisher interface {
	// PublishBatch publishes events to topic. The result lists, per event, what the broker accepted.
	// A non-nil error means the batch as a whole could not be completed; the caller must then treat
	// no event as published, whatever the result contains.
	PublishBatch(ctx context.Context, topic string, events []postgres.Event) (PublishResult, error)
}

// PublishResult is the per-event outcome of a PublishBatch call.
type PublishResult struct {
	// Successful holds the ids of the events the broker accepted.
	Successful []uuid.UUID
	Failed     []PublishFailure
}

// PublishFailure describes one event the broker did not accept.
type PublishFailure struct {
	// EventId is uuid.Nil when the failure cannot be tied to a single event.
	EventId uuid.UUID
	Reason  string
}
