package postgres

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	EventId       uuid.UUID
	Type          string
	CorrelationId uuid.UUID
	Producer      string
	EmittedAt     time.Time
	Payload       []byte
}
