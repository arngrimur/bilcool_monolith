package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func LogWithCorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("Correlation-Id")
		if _, err := uuid.Parse(id); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Correlation-Id header is required"})
			return
		}
		logger := log.Logger.With().Str("correlation_id", id).Logger()
		c.Request = c.Request.WithContext(logger.WithContext(c.Request.Context()))
		c.Next()
	}
}
