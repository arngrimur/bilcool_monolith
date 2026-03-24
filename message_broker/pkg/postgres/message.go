package postgres

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	EventId       uuid.UUID  `json:"event_id"`
	Type          string     `json:"type"`
	CorrelationId uuid.UUID  `json:"correlation_id"`
	Producer      string     `json:"producer"`
	EmittedAt     *time.Time `json:"emitted_at"`
	Payload       json.RawMessage `json:"payload"`
}

type MessageBody struct {
	Type             string    `json:"Type"`
	MessageId        string    `json:"MessageId"`
	TopicArn         string    `json:"TopicArn"`
	Message          Event     `json:"Message"`
	Timestamp        time.Time `json:"Timestamp"`
	UnsubscribeURL   string    `json:"UnsubscribeURL"`
	SignatureVersion string    `json:"SignatureVersion"`
	Signature        string    `json:"Signature"`
	SigningCertURL   string    `json:"SigningCertURL"`
}

func (m *MessageBody) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Type             string    `json:"Type"`
		MessageId        string    `json:"MessageId"`
		TopicArn         string    `json:"TopicArn"`
		Message          string    `json:"Message"`
		Timestamp        time.Time `json:"Timestamp"`
		UnsubscribeURL   string    `json:"UnsubscribeURL"`
		SignatureVersion string    `json:"SignatureVersion"`
		Signature        string    `json:"Signature"`
		SigningCertURL   string    `json:"SigningCertURL"`
	}
	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	m.Type = aux.Type
	m.MessageId = aux.MessageId
	m.TopicArn = aux.TopicArn
	m.Timestamp = aux.Timestamp
	m.UnsubscribeURL = aux.UnsubscribeURL
	m.SignatureVersion = aux.SignatureVersion
	m.Signature = aux.Signature
	m.SigningCertURL = aux.SigningCertURL
	return json.Unmarshal([]byte(aux.Message), &m.Message)
}
