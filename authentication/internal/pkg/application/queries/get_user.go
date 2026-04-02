package queries

import (
	"context"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

type GetUserHandler struct {
	*domain.Users
}

func NewGetUserHandler(users *domain.Users) GetUserHandler {
	return GetUserHandler{Users: users}
}

func (h GetUserHandler) GetUserByRef(ctx context.Context, userRef uuid.UUID) (extdomain.UserResponse, error) {
	return h.Users.FindByRef(ctx, userRef)
}
