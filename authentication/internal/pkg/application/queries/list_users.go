package queries

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

type ListUsersHandler struct {
	*domain.Users
}

func NewListUsersHandler(users *domain.Users) ListUsersHandler {
	return ListUsersHandler{Users: users}
}

func (h ListUsersHandler) ListUsers(ctx context.Context) ([]extdomain.UserResponse, error) {
	return h.Users.FindAll(ctx)
}
