package logging

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type WithOpts func(c zerolog.Context) zerolog.Context

func WithService(value string) WithOpts {
	return func(c zerolog.Context) zerolog.Context {
		return c.Str("service", value)
	}
}

// NewDefaultLogger Attach logger to context for zerolog logger
// Default logger is attached to context with caller and timestamp
func NewDefaultLogger(ctx context.Context, opts ...WithOpts) context.Context {
	c := log.With().Caller().Timestamp()
	for _, o := range opts {
		c = o(c)
	}
	ctx = c.Logger().WithContext(ctx)
	return ctx
}
