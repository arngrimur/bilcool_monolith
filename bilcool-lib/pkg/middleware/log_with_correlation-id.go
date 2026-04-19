package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Middleware for adding correlation id to logs using Gin router
func LogWithCorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		v := c.GetHeader("Correlation-Id")
		parse, err := uuid.Parse(v)
		if err != nil {
			c.Abort()
		}
		logger := log.With().Str("correlation-id", parse.String())
		ctx := logger.Logger().WithContext(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
