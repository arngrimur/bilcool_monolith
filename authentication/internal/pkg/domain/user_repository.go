package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

//go:generate mockgen -source=user_repository.go -destination=user_repository_mock.go -package=domain
type UsersRepository interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (extdomain.UserResponse, error)
	DeleteUser(ctx context.Context, userRef uuid.UUID) error
	FindAll(ctx context.Context) ([]extdomain.UserResponse, error)
	FindByEmail(ctx context.Context, email string) (extdomain.UserResponse, error)
	FindByRef(ctx context.Context, userRef uuid.UUID) (extdomain.UserResponse, error)
	CreateSecurityToken(ctx context.Context, userRef uuid.UUID, token string, expiresAt time.Time) error
	VerifyAndConsumeToken(ctx context.Context, userRef uuid.UUID, token string) error
	StoreWebAuthnSession(ctx context.Context, session WebAuthnSession) error
	GetWebAuthnSession(ctx context.Context, sessionID uuid.UUID) (WebAuthnSession, error)
	DeleteWebAuthnSession(ctx context.Context, sessionID uuid.UUID) error
	GetPasskeys(ctx context.Context, userRef uuid.UUID) ([]Passkey, error)
	StorePasskey(ctx context.Context, userRef uuid.UUID, passkey Passkey) error
	DeletePasskeys(ctx context.Context, userRef uuid.UUID) error
	ChangeUserRole(ctx context.Context, targetRef uuid.UUID, newRole string) error
}
