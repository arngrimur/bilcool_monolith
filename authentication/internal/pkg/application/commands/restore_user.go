package commands

import (
	"context"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

type RestoreUserHandler struct {
	*domain.Users
}

func NewRestoreUserHandler(users *domain.Users) RestoreUserHandler {
	return RestoreUserHandler{Users: users}
}

func (h RestoreUserHandler) RestoreUser(ctx context.Context, userRef uuid.UUID) (extdomain.UserResponse, error) {
	return h.Users.RestoreUser(ctx, userRef)
}
