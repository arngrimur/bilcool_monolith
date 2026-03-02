package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	_ "github.com/arngrimur/bilcool_monolith/docs"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

// @title          BilCool REST API
// @version         1.0
// @description     The REST API for BilCool to book and view bookings and other stuff.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.basic  None

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
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
	// use ginSwagger middleware to serve the API docs
	h.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	h.router.GET("/api/v1/bookings", h.getAllBookings)
	h.router.GET("/api/v1/bookings/:id", h.getBooking)
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

// GetBookingAccount godoc
// @Summary      Show a booking
// @Description  get booking by booking reference
// @Tags         bookings
// @Accept       json
// @Produce      json
// @Param        id   path      uuid  true  "Booking reference"
// @Success      200  {object}  domain.BookingRequest
// @Failure      400  {object}  HTTPError
// @Failure      404  {object}  HTTPError
// @Failure      500  {object}  HTTPError
// @Router       /booking/{id} [get]
func (h *httpRouter) getBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}

	booking, err := h.queries.GetBooking(c.Request.Context(), domain.BookingRequest{BookingReference: id})
	if err != nil {
		NewError(c, http.StatusBadRequest, fmt.Errorf("failed to get booking"))
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
