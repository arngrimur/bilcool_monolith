package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	outbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type UsersRepository struct {
	DbActions
	Transactioner
}

func NewUsersRepository(db *sql.DB) UsersRepository {
	return UsersRepository{DbActions: db, Transactioner: db}
}

func (r UsersRepository) createTransaction(ctx context.Context) (UsersRepository, *sql.Tx, error) {
	tx, err := r.BeginTx(ctx, nil)
	if err != nil {
		return UsersRepository{}, nil, err
	}
	return UsersRepository{DbActions: tx, Transactioner: r.Transactioner}, tx, nil
}

func (r UsersRepository) CreateUser(ctx context.Context, req domain.CreateUserRequest) (extdomain.UserResponse, error) {
	local_r, tx, err := r.createTransaction(ctx)
	if err != nil {
		return extdomain.UserResponse{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var resp extdomain.UserResponse
	err = local_r.QueryRowContext(ctx,
		`INSERT INTO users (username, email)
		 VALUES ($1, $2)
		 RETURNING user_ref, username, email, (SELECT name FROM roles WHERE id = role_id)`,
		req.Username, req.Email,
	).Scan(&resp.UserRef, &resp.Username, &resp.Email, &resp.Role)
	if err != nil {
		return extdomain.UserResponse{}, err
	}

	payload, err := json.Marshal(resp)
	if err != nil {
		return extdomain.UserResponse{}, err
	}
	err = outbox.Insert(ctx, tx, outbox.Event{
		EventId:       uuid.New(),
		Type:          extdomain.EventUserCreated,
		CorrelationId: uuid.New(),
		Producer:      extdomain.EventProducer,
		Payload:       payload,
	})
	if err != nil {
		return extdomain.UserResponse{}, err
	}

	return resp, tx.Commit()
}

func (r UsersRepository) DeleteUser(ctx context.Context, userRef uuid.UUID) error {
	local_r, tx, err := r.createTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var isAdmin bool
	err = local_r.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM users u JOIN roles r ON r.id = u.role_id
			WHERE u.user_ref = $1 AND r.name = 'admin' AND u.deleted_at IS NULL
		)`,
		userRef,
	).Scan(&isAdmin)
	if err != nil {
		return err
	}
	if isAdmin {
		var adminCount int
		err = local_r.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM users u JOIN roles r ON r.id = u.role_id WHERE r.name = 'admin' AND u.deleted_at IS NULL`,
		).Scan(&adminCount)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return domain.ErrLastAdmin
		}
	}

	var ref uuid.UUID
	err = local_r.QueryRowContext(ctx,
		`UPDATE users SET deleted_at = NOW() WHERE user_ref = $1 AND deleted_at IS NULL RETURNING user_ref`,
		userRef,
	).Scan(&ref)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(extdomain.UserResponse{UserRef: ref})
	if err != nil {
		return err
	}
	err = outbox.Insert(ctx, tx, outbox.Event{
		EventId:       uuid.New(),
		Type:          extdomain.EventUserDeleted,
		CorrelationId: uuid.New(),
		Producer:      extdomain.EventProducer,
		Payload:       payload,
	})
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r UsersRepository) FindAll(ctx context.Context) ([]extdomain.UserResponse, error) {
	rows, err := r.QueryContext(ctx,
		`SELECT u.user_ref, u.username, u.email, r.name
		 FROM users u JOIN roles r ON r.id = u.role_id
		 WHERE u.deleted_at IS NULL`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	users := make([]extdomain.UserResponse, 0)
	for rows.Next() {
		var u extdomain.UserResponse
		if err := rows.Scan(&u.UserRef, &u.Username, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r UsersRepository) FindByEmail(ctx context.Context, email string) (extdomain.UserResponse, error) {
	var resp extdomain.UserResponse
	err := r.QueryRowContext(ctx,
		`SELECT u.user_ref, u.username, u.email, r.name
		 FROM users u JOIN roles r ON r.id = u.role_id
		 WHERE u.email = $1 AND u.deleted_at IS NULL`,
		email,
	).Scan(&resp.UserRef, &resp.Username, &resp.Email, &resp.Role)
	if err != nil {
		return extdomain.UserResponse{}, err
	}
	return resp, nil
}

func (r UsersRepository) FindByRef(ctx context.Context, userRef uuid.UUID) (extdomain.UserResponse, error) {
	var resp extdomain.UserResponse
	err := r.QueryRowContext(ctx,
		`SELECT u.user_ref, u.username, u.email, r.name
		 FROM users u JOIN roles r ON r.id = u.role_id
		 WHERE u.user_ref = $1 AND u.deleted_at IS NULL`,
		userRef,
	).Scan(&resp.UserRef, &resp.Username, &resp.Email, &resp.Role)
	if err != nil {
		return extdomain.UserResponse{}, err
	}
	return resp, nil
}

func (r UsersRepository) CreateSecurityToken(ctx context.Context, userRef uuid.UUID, token string, expiresAt time.Time) error {
	_, err := r.ExecContext(ctx,
		`INSERT INTO security_tokens (user_ref, token, expires_at) VALUES ($1, $2, $3)`,
		userRef, token, expiresAt,
	)
	return err
}

func (r UsersRepository) VerifyAndConsumeToken(ctx context.Context, userRef uuid.UUID, token string) error {
	result, err := r.ExecContext(ctx,
		`UPDATE security_tokens SET used_at = NOW()
		 WHERE user_ref = $1 AND token = $2 AND expires_at > NOW() AND used_at IS NULL`,
		userRef, token,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrInvalidToken
	}
	return nil
}

func (r UsersRepository) StoreWebAuthnSession(ctx context.Context, session domain.WebAuthnSession) error {
	_, err := r.ExecContext(ctx,
		`INSERT INTO webauthn_sessions (session_id, user_ref, session_type, data, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		session.SessionID, session.UserRef, session.SessionType, session.Data, session.ExpiresAt,
	)
	return err
}

func (r UsersRepository) GetWebAuthnSession(ctx context.Context, sessionID uuid.UUID) (domain.WebAuthnSession, error) {
	var s domain.WebAuthnSession
	var data []byte
	err := r.QueryRowContext(ctx,
		`SELECT session_id, user_ref, session_type, data, expires_at FROM webauthn_sessions WHERE session_id = $1 AND expires_at > NOW()`,
		sessionID,
	).Scan(&s.SessionID, &s.UserRef, &s.SessionType, &data, &s.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.WebAuthnSession{}, domain.ErrSessionNotFound
		}
		return domain.WebAuthnSession{}, err
	}
	s.Data = data
	return s, nil
}

func (r UsersRepository) DeleteWebAuthnSession(ctx context.Context, sessionID uuid.UUID) error {
	_, err := r.ExecContext(ctx,
		`DELETE FROM webauthn_sessions WHERE session_id = $1`,
		sessionID,
	)
	return err
}

func (r UsersRepository) GetPasskeys(ctx context.Context, userRef uuid.UUID) ([]domain.Passkey, error) {
	rows, err := r.QueryContext(ctx,
		`SELECT credential_id, credential FROM passkeys WHERE user_ref = $1`,
		userRef,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var passkeys []domain.Passkey
	for rows.Next() {
		var p domain.Passkey
		if err := rows.Scan(&p.CredentialID, &p.Data); err != nil {
			return nil, err
		}
		passkeys = append(passkeys, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return passkeys, nil
}

func (r UsersRepository) ChangeUserRole(ctx context.Context, targetRef uuid.UUID, newRole string) error {
	local_r, tx, err := r.createTransaction(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if newRole == "user" {
		var adminCount int
		err = local_r.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM users u JOIN roles r ON r.id = u.role_id WHERE r.name = 'admin' AND u.deleted_at IS NULL`,
		).Scan(&adminCount)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return domain.ErrLastAdmin
		}
	}

	var ref uuid.UUID
	err = local_r.QueryRowContext(ctx,
		`UPDATE users SET role_id = (SELECT id FROM roles WHERE name = $2) WHERE user_ref = $1 AND deleted_at IS NULL RETURNING user_ref`,
		targetRef, newRole,
	).Scan(&ref)
	if err == sql.ErrNoRows {
		return domain.ErrUserNotFound
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r UsersRepository) StorePasskey(ctx context.Context, userRef uuid.UUID, passkey domain.Passkey) error {
	_, err := r.ExecContext(ctx,
		`INSERT INTO passkeys (user_ref, credential_id, credential) VALUES ($1, $2, $3)`,
		userRef, passkey.CredentialID, passkey.Data,
	)
	return err
}
