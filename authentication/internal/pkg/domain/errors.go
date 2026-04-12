package domain

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidToken      = errors.New("invalid or expired security token")
	ErrSessionNotFound   = errors.New("session not found or expired")
	ErrInvalidCredential = errors.New("invalid credential")
	ErrForbidden         = errors.New("forbidden")
	ErrSelfRoleChange    = errors.New("cannot change your own role")
	ErrLastAdmin         = errors.New("cannot remove the last admin")
)
