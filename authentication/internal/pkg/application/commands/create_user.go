package commands

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

type CreateUserHandler struct {
	*domain.Users
}

func NewCreateUserHandler(users *domain.Users) CreateUserHandler {
	return CreateUserHandler{Users: users}
}

func (h CreateUserHandler) CreateUser(ctx context.Context, req domain.CreateUserRequest) (extdomain.UserResponse, error) {
	return h.Users.CreateUser(ctx, req)
}
