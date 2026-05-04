package commands

import (
	"context"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

type UpdateUserHandler struct {
	users *domain.Users
}

func NewUpdateUserHandler(users *domain.Users) UpdateUserHandler {
	return UpdateUserHandler{users: users}
}

func (h UpdateUserHandler) UpdateUser(ctx context.Context, userRef uuid.UUID, req domain.UpdateUserRequest) (extdomain.UserResponse, error) {
	return h.users.UpdateUser(ctx, userRef, req)
}
