package domain

import "github.com/google/uuid"

type UserResponse struct {
	UserRef  uuid.UUID `json:"user_ref" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username string    `json:"username" example:"johndoe"`
	Email    string    `json:"email" example:"john@example.com"`
}
