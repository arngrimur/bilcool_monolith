package web

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

func NewError(ctx *gin.Context, status int, err error) {
	er := HTTPError{
		Code:    status,
		Message: err.Error(),
	}
	ctx.JSON(status, er)
}

// HTTPError example
type HTTPError struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"status bad request"`
}

func NewHttpError(err error) HTTPError {
	switch err {
	case sql.ErrNoRows:
		return HTTPError{
			Message: "not found",
			Code:    http.StatusNotFound,
		}
	case domain.ErrBookingAlreadyStarted:
		return HTTPError{
			Message: "booking already started",
			Code:    http.StatusUnprocessableEntity,
		}
	case domain.ErrUserNotFound:
		return HTTPError{
			Message: "user not found",
			Code:    http.StatusNotFound,
		}
	default:
		return HTTPError{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		}
	}
}
