package commands

import (
	"context"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
)

type ChangeUserRoleHandler struct {
	users *domain.Users
}

func NewChangeUserRoleHandler(users *domain.Users) ChangeUserRoleHandler {
	return ChangeUserRoleHandler{users: users}
}

func (h ChangeUserRoleHandler) ChangeUserRole(ctx context.Context, callerRef, targetRef uuid.UUID, newRole string) error {
	return h.users.ChangeUserRole(ctx, callerRef, targetRef, newRole)
}
