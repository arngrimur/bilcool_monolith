package domain

import (
	"encoding/json"
	"time"
)

type EventItem struct {
	EventId       string `dynamodbav:"event_id" json:"event_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EventType     string `dynamodbav:"event_type" json:"event_type" example:"booking_ended"`
	CorrelationId string `dynamodbav:"correlation_id" json:"correlation_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Producer      string `dynamodbav:"producer" json:"producer" example:"bookings"`
	EmittedAt     string `dynamodbav:"emitted_at" json:"emitted_at" example:"2026-01-01T12:00:00Z"`
	Payload       string `dynamodbav:"payload" json:"payload"`
	ReceivedAt    string `dynamodbav:"received_at" json:"received_at" example:"2026-01-01T12:00:01Z"`
}

type EventResponse struct {
	EventId       string          `json:"event_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	EventType     string          `json:"event_type" example:"booking_ended"`
	CorrelationId string          `json:"correlation_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Producer      string          `json:"producer" example:"bookings"`
	EmittedAt     string          `json:"emitted_at" example:"2026-01-01T12:00:00Z"`
	Payload       json.RawMessage `json:"payload"`
	ReceivedAt    string          `json:"received_at" example:"2026-01-01T12:00:01Z"`
}

type QueryParams struct {
	EventId        *string    `form:"event_id"`
	Producer       *string    `form:"producer"`
	EventType      *string    `form:"event_type"`
	EmittedAt      *time.Time `form:"emitted_at" time_format:"2006-01-02T15:04:05Z07:00"`
	EmittedAtGte   *time.Time `form:"emitted_at_gte" time_format:"2006-01-02T15:04:05Z07:00"`
	EmittedAtLte   *time.Time `form:"emitted_at_lte" time_format:"2006-01-02T15:04:05Z07:00"`
	Limit          int        `form:"limit,default=50"`
	Offset         int        `form:"offset,default=0"`
	OrderBy        string     `form:"order_by,default=emitted_at"`
	OrderDirection string     `form:"order_direction,default=asc"`
}

func (e EventItem) ToResponse() EventResponse {
	return EventResponse{
		EventId:       e.EventId,
		EventType:     e.EventType,
		CorrelationId: e.CorrelationId,
		Producer:      e.Producer,
		EmittedAt:     e.EmittedAt,
		Payload:       json.RawMessage(e.Payload),
		ReceivedAt:    e.ReceivedAt,
	}
}
