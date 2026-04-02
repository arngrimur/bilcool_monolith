package web

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
)

type HTTPError struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"status bad request"`
}

func NewError(ctx *gin.Context, status int, message string) {
	ctx.JSON(status, HTTPError{Code: status, Message: message})
}

func NewHttpError(err error) HTTPError {
	switch err {
	case sql.ErrNoRows, domain.ErrUserNotFound:
		return HTTPError{Message: "not found", Code: http.StatusNotFound}
	case domain.ErrUserAlreadyExists:
		return HTTPError{Message: "user already exists", Code: http.StatusConflict}
	case domain.ErrInvalidToken:
		return HTTPError{Message: "invalid or expired token", Code: http.StatusUnauthorized}
	case domain.ErrSessionNotFound:
		return HTTPError{Message: "session not found or expired", Code: http.StatusUnauthorized}
	case domain.ErrInvalidCredential:
		return HTTPError{Message: "invalid credential", Code: http.StatusUnauthorized}
	default:
		return HTTPError{Message: "internal server error", Code: http.StatusInternalServerError}
	}
}
