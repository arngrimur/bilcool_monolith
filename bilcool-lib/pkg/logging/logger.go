package logging

import (
	"context"

	"github.com/rs/zerolog/log"
)

// NewDefaultLogger Attach logger to context for zerolog logger
// Default logger is attached to context with caller and timestamp
func NewDefaultLogger(ctx context.Context) context.Context {
	c := log.With().Caller().Timestamp()
	ctx = c.Logger().WithContext(ctx)
	return ctx
}
