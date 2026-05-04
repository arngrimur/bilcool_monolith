package domain

import (
	"github.com/google/uuid"
)

type UserResponse struct {
	UserRef  uuid.UUID `json:"user_ref" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username string    `json:"username" example:"johndoe"`
	Email    string    `json:"email" example:"john@example.com"`
	Role     string    `json:"role" example:"user"`
}

type DeletedUserResponse struct {
	UserRef   uuid.UUID `json:"user_ref" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string    `json:"username" example:"johndoe"`
	Email     string    `json:"email" example:"john@example.com"`
	Role      string    `json:"role" example:"user"`
	DeletedAt string    `json:"deleted_at" example:"2026-05-04T12:00:00Z"`
}

const EventProducer string = "authentication"
const EventUserCreated string = "user.created"
const EventUserDeleted string = "user.deleted"
const EventUserRestored string = "user.restored"
