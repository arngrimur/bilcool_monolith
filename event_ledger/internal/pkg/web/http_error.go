package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HTTPError struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"status bad request"`
}

func NewError(c *gin.Context, status int, message string) {
	c.JSON(status, HTTPError{Code: status, Message: message})
}

func NewHttpError(err error) HTTPError {
	return HTTPError{Message: "internal server error", Code: http.StatusInternalServerError}
}
