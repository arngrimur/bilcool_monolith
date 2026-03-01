package web

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

type httpRouter struct {
	router  *gin.Engine
	queries application.Queries
}

func NewRouter(q application.Queries) *httpRouter {
	h := &httpRouter{
		queries: q,
		router:  gin.Default(),
	}

	h.router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	h.router.GET("/api/bookings", h.getAllBookings)
	h.router.GET("/api/bookings/:id", h.getBooking)
	return h
}

func (h *httpRouter) StartRouter(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      h.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}

func (h *httpRouter) getBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}

	booking, err := h.queries.GetBooking(c.Request.Context(), domain.BookingRequest{BookingReference: id})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no booking found"})
		return
	}
	c.JSON(http.StatusOK, booking)

}

func (h *httpRouter) getAllBookings(c *gin.Context) {
	bookings, err := h.queries.GetAllBooking(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, bookings)

}
