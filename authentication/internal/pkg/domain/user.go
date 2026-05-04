package domain

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

type ChangeUserRoleRequest struct {
	Role string `json:"role" validate:"required" binding:"required" example:"admin"`
}

type CreateUserRequest struct {
	Username string `json:"username" validate:"required" binding:"required" example:"johndoe"`
	Email    string `json:"email" validate:"required,email" binding:"required" example:"john@example.com"`
}

type UpdateUserRequest struct {
	Username *string `json:"username" example:"johndoe"`
	Email    *string `json:"email" example:"john@example.com"`
}

type LoginBeginRequest struct {
	Email  string `json:"email"  validate:"required,email" binding:"required" example:"john@example.com"`
	Locale string `json:"locale" example:"sv"`
}

type ResetLoginRequest struct {
	Email  string `json:"email"  validate:"required,email" binding:"required" example:"john@example.com"`
	Locale string `json:"locale" example:"sv"`
}

type LoginBeginResponse struct {
	NextStep  string          `json:"next_step"`
	SessionID *uuid.UUID      `json:"session_id,omitempty"`
	Options   json.RawMessage `json:"options,omitempty"`
}

type VerifyTokenRequest struct {
	Email string `json:"email" validate:"required,email" binding:"required" example:"john@example.com"`
	Token string `json:"token" validate:"required" binding:"required" example:"123456"`
}

type VerifyTokenResponse struct {
	SessionID uuid.UUID       `json:"session_id"`
	Options   json.RawMessage `json:"options"`
}

type LoginCompleteRequest struct {
	SessionID  uuid.UUID       `json:"session_id" validate:"required" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Credential json.RawMessage `json:"credential" validate:"required" binding:"required"`
}

type LoginCompleteResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"`
}

type WebAuthnSession struct {
	SessionID   uuid.UUID
	UserRef     uuid.UUID
	SessionType string
	Data        []byte
	ExpiresAt   time.Time
}

type Passkey struct {
	CredentialID []byte
	Data         []byte
}

type WebAuthnUser struct {
	UserRef  uuid.UUID
	Username string
	Passkeys []Passkey
}

func (u WebAuthnUser) WebAuthnID() []byte          { return u.UserRef[:] }
func (u WebAuthnUser) WebAuthnName() string        { return u.Username }
func (u WebAuthnUser) WebAuthnDisplayName() string { return u.Username }
func (u WebAuthnUser) WebAuthnIcon() string        { return "" }
func (u WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, 0, len(u.Passkeys))
	for _, p := range u.Passkeys {
		var c webauthn.Credential
		if err := json.Unmarshal(p.Data, &c); err != nil {
			continue
		}
		creds = append(creds, c)
	}
	return creds
}

type WebAuthnProvider interface {
	BeginRegistration(user webauthn.User, opts ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error)
	BeginLogin(user webauthn.User, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	CreateCredential(user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error)
	ValidateLogin(user webauthn.User, session webauthn.SessionData, parsedResponse *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error)
}

type Users struct {
	r UsersRepository
	m mail.MailSender
}

func NewUsers(r UsersRepository, m mail.MailSender) *Users {
	return &Users{r: r, m: m}
}

func (u *Users) CreateUser(ctx context.Context, req CreateUserRequest) (extdomain.UserResponse, error) {
	return u.r.CreateUser(ctx, req)
}

func (u *Users) DeleteUser(ctx context.Context, userRef uuid.UUID) error {
	return u.r.DeleteUser(ctx, userRef)
}

func (u *Users) FindAll(ctx context.Context) ([]extdomain.UserResponse, error) {
	return u.r.FindAll(ctx)
}

func (u *Users) FindByEmail(ctx context.Context, email string) (extdomain.UserResponse, error) {
	return u.r.FindByEmail(ctx, email)
}

func (u *Users) FindByRef(ctx context.Context, userRef uuid.UUID) (extdomain.UserResponse, error) {
	return u.r.FindByRef(ctx, userRef)
}

func (u *Users) GetPasskeys(ctx context.Context, userRef uuid.UUID) ([]Passkey, error) {
	return u.r.GetPasskeys(ctx, userRef)
}

func (u *Users) CreateSecurityToken(ctx context.Context, userRef uuid.UUID, token string, expiresAt time.Time) error {
	return u.r.CreateSecurityToken(ctx, userRef, token, expiresAt)
}

func (u *Users) VerifyAndConsumeToken(ctx context.Context, userRef uuid.UUID, token string) error {
	return u.r.VerifyAndConsumeToken(ctx, userRef, token)
}

func (u *Users) SendToken(ctx context.Context, email, token, locale string) error {
	return u.m.SendSecurityToken(ctx, email, token, locale)
}

func (u *Users) StoreWebAuthnSession(ctx context.Context, session WebAuthnSession) error {
	return u.r.StoreWebAuthnSession(ctx, session)
}

func (u *Users) GetWebAuthnSession(ctx context.Context, sessionID uuid.UUID) (WebAuthnSession, error) {
	return u.r.GetWebAuthnSession(ctx, sessionID)
}

func (u *Users) DeleteWebAuthnSession(ctx context.Context, sessionID uuid.UUID) error {
	return u.r.DeleteWebAuthnSession(ctx, sessionID)
}

func (u *Users) StorePasskey(ctx context.Context, userRef uuid.UUID, passkey Passkey) error {
	return u.r.StorePasskey(ctx, userRef, passkey)
}

func (u *Users) DeletePasskeys(ctx context.Context, userRef uuid.UUID) error {
	return u.r.DeletePasskeys(ctx, userRef)
}

func (u *Users) UpdateUser(ctx context.Context, userRef uuid.UUID, req UpdateUserRequest) (extdomain.UserResponse, error) {
	return u.r.UpdateUser(ctx, userRef, req)
}

func (u *Users) ChangeUserRole(ctx context.Context, callerRef, targetRef uuid.UUID, newRole string) error {
	if callerRef == targetRef {
		return ErrSelfRoleChange
	}
	return u.r.ChangeUserRole(ctx, targetRef, newRole)
}

func GenerateSecurityToken() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
