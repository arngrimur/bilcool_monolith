package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
)

type LoginBeginHandler struct {
	users    *domain.Users
	webAuthn domain.WebAuthnProvider
}

func NewLoginBeginHandler(users *domain.Users, webAuthn domain.WebAuthnProvider) LoginBeginHandler {
	return LoginBeginHandler{users: users, webAuthn: webAuthn}
}

func (h LoginBeginHandler) LoginBegin(ctx context.Context, req domain.LoginBeginRequest) (domain.LoginBeginResponse, error) {
	user, err := h.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return domain.LoginBeginResponse{}, err
	}

	passkeys, err := h.users.GetPasskeys(ctx, user.UserRef)
	if err != nil {
		return domain.LoginBeginResponse{}, err
	}

	if len(passkeys) == 0 {
		token, err := domain.GenerateSecurityToken()
		if err != nil {
			return domain.LoginBeginResponse{}, err
		}
		expiresAt := time.Now().Add(10 * time.Minute)
		if err = h.users.CreateSecurityToken(ctx, user.UserRef, token, expiresAt); err != nil {
			return domain.LoginBeginResponse{}, err
		}
		if err = h.users.SendToken(ctx, user.Email, token); err != nil {
			return domain.LoginBeginResponse{}, err
		}
		return domain.LoginBeginResponse{NextStep: "verify_token"}, nil
	}

	webAuthnUser := domain.WebAuthnUser{
		UserRef:  user.UserRef,
		Username: user.Username,
		Passkeys: passkeys,
	}
	options, sessionData, err := h.webAuthn.BeginLogin(webAuthnUser)
	if err != nil {
		return domain.LoginBeginResponse{}, err
	}

	sessionBytes, err := json.Marshal(sessionData)
	if err != nil {
		return domain.LoginBeginResponse{}, err
	}
	sessionID := uuid.New()
	err = h.users.StoreWebAuthnSession(ctx, domain.WebAuthnSession{
		SessionID:   sessionID,
		UserRef:     user.UserRef,
		SessionType: "assertion",
		Data:        sessionBytes,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})
	if err != nil {
		return domain.LoginBeginResponse{}, err
	}

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return domain.LoginBeginResponse{}, err
	}

	return domain.LoginBeginResponse{
		NextStep:  "passkey_assertion",
		SessionID: &sessionID,
		Options:   optionsJSON,
	}, nil
}

type VerifyTokenHandler struct {
	users    *domain.Users
	webAuthn domain.WebAuthnProvider
}

func NewVerifyTokenHandler(users *domain.Users, webAuthn domain.WebAuthnProvider) VerifyTokenHandler {
	return VerifyTokenHandler{users: users, webAuthn: webAuthn}
}

func (h VerifyTokenHandler) VerifyToken(ctx context.Context, req domain.VerifyTokenRequest) (domain.VerifyTokenResponse, error) {
	user, err := h.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return domain.VerifyTokenResponse{}, err
	}

	if err = h.users.VerifyAndConsumeToken(ctx, user.UserRef, req.Token); err != nil {
		return domain.VerifyTokenResponse{}, err
	}

	webAuthnUser := domain.WebAuthnUser{
		UserRef:  user.UserRef,
		Username: user.Username,
	}
	options, sessionData, err := h.webAuthn.BeginRegistration(webAuthnUser)
	if err != nil {
		return domain.VerifyTokenResponse{}, err
	}

	sessionBytes, err := json.Marshal(sessionData)
	if err != nil {
		return domain.VerifyTokenResponse{}, err
	}
	sessionID := uuid.New()
	err = h.users.StoreWebAuthnSession(ctx, domain.WebAuthnSession{
		SessionID:   sessionID,
		UserRef:     user.UserRef,
		SessionType: "registration",
		Data:        sessionBytes,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	})
	if err != nil {
		return domain.VerifyTokenResponse{}, err
	}

	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return domain.VerifyTokenResponse{}, err
	}

	return domain.VerifyTokenResponse{
		SessionID: sessionID,
		Options:   optionsJSON,
	}, nil
}

type LoginCompleteHandler struct {
	users     *domain.Users
	webAuthn  domain.WebAuthnProvider
	jwtSecret []byte
}

func NewLoginCompleteHandler(users *domain.Users, webAuthn domain.WebAuthnProvider, jwtSecret string) LoginCompleteHandler {
	return LoginCompleteHandler{
		users:     users,
		webAuthn:  webAuthn,
		jwtSecret: []byte(jwtSecret),
	}
}

func (h LoginCompleteHandler) LoginComplete(ctx context.Context, req domain.LoginCompleteRequest) (domain.LoginCompleteResponse, error) {
	session, err := h.users.GetWebAuthnSession(ctx, req.SessionID)
	if err != nil {
		return domain.LoginCompleteResponse{}, err
	}

	if err = h.users.DeleteWebAuthnSession(ctx, req.SessionID); err != nil {
		return domain.LoginCompleteResponse{}, err
	}

	user, err := h.users.FindByRef(ctx, session.UserRef)
	if err != nil {
		return domain.LoginCompleteResponse{}, err
	}

	passkeys, err := h.users.GetPasskeys(ctx, user.UserRef)
	if err != nil {
		return domain.LoginCompleteResponse{}, err
	}

	webAuthnUser := domain.WebAuthnUser{
		UserRef:  user.UserRef,
		Username: user.Username,
		Passkeys: passkeys,
	}

	var sessionData webauthn.SessionData
	if err = json.Unmarshal(session.Data, &sessionData); err != nil {
		return domain.LoginCompleteResponse{}, err
	}

	credJSON := req.Credential

	if session.SessionType == "registration" {
		parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(credJSON))
		if err != nil {
			return domain.LoginCompleteResponse{}, domain.ErrInvalidCredential
		}
		credential, err := h.webAuthn.CreateCredential(webAuthnUser, sessionData, parsed)
		if err != nil {
			return domain.LoginCompleteResponse{}, domain.ErrInvalidCredential
		}
		credData, err := json.Marshal(credential)
		if err != nil {
			return domain.LoginCompleteResponse{}, err
		}
		if err = h.users.StorePasskey(ctx, user.UserRef, domain.Passkey{
			CredentialID: credential.ID,
			Data:         credData,
		}); err != nil {
			return domain.LoginCompleteResponse{}, err
		}
	} else {
		parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(credJSON))
		if err != nil {
			return domain.LoginCompleteResponse{}, domain.ErrInvalidCredential
		}
		_, err = h.webAuthn.ValidateLogin(webAuthnUser, sessionData, parsed)
		if err != nil {
			return domain.LoginCompleteResponse{}, domain.ErrInvalidCredential
		}
	}

	token, err := h.generateJWT(user.UserRef)
	if err != nil {
		return domain.LoginCompleteResponse{}, err
	}

	return domain.LoginCompleteResponse{Token: token}, nil
}

func (h LoginCompleteHandler) generateJWT(userRef uuid.UUID) (string, error) {
	claims := jwtlib.MapClaims{
		"user_ref": userRef.String(),
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}
