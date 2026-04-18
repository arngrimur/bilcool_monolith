package web

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/persistance/postgres"
)

type BookingQuerier interface {
	GetFinishedBookings(ctx context.Context) ([]postgres.FinishedBooking, error)
}

type HttpRouter struct {
	router  *gin.Engine
	querier BookingQuerier
}

func NewRouter(querier BookingQuerier) *HttpRouter {
	engine := gin.Default()
	h := &HttpRouter{
		router:  engine,
		querier: querier,
	}

	h.router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	h.router.GET("/api/v1/bookings", h.getFinishedBookings)

	return h
}

func (h *HttpRouter) getFinishedBookings(c *gin.Context) {
	bookings, err := h.querier.GetFinishedBookings(c.Request.Context())
	if err != nil {
		log.Ctx(c.Request.Context()).Error().Err(err).Msg("failed to get finished bookings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve finished bookings"})
		return
	}
	c.JSON(http.StatusOK, bookings)
}

func (h *HttpRouter) StartRouter(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      h.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
