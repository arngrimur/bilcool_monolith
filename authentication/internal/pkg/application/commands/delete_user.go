package commands

import (
	"context"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
)

type DeleteUserHandler struct {
	*domain.Users
}

func NewDeleteUserHandler(users *domain.Users) DeleteUserHandler {
	return DeleteUserHandler{Users: users}
}

func (h DeleteUserHandler) DeleteUser(ctx context.Context, userRef uuid.UUID) error {
	return h.Users.DeleteUser(ctx, userRef)
}
