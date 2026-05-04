package queries

import (
	"context"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

type ListDeletedUsersHandler struct {
	*domain.Users
}

func NewListDeletedUsersHandler(users *domain.Users) ListDeletedUsersHandler {
	return ListDeletedUsersHandler{Users: users}
}

func (h ListDeletedUsersHandler) ListDeletedUsers(ctx context.Context) ([]extdomain.DeletedUserResponse, error) {
	return h.Users.FindAllDeleted(ctx)
}
