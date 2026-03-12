package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

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

// @host localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.basic  None

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
type httpRouter struct {
	router   *gin.Engine
	queries  application.Queries
	commands application.Commands
}

func NewRouter(q application.Queries, c application.Commands) *httpRouter {
	engine := gin.Default()
	h := &httpRouter{
		queries:  q,
		commands: c,
		router:   engine,
	}

	internalRoutes(h)
	queryRoutes(h)
	commandRoutes(h)
	return h
}

func internalRoutes(h *httpRouter) {
	h.router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	// use ginSwagger middleware to serve the API docs
	h.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func queryRoutes(h *httpRouter) {
	h.router.GET("/api/v1/bookings", h.getAllBookings)
	h.router.GET("/api/v1/bookings/:id", h.getBooking)
}

func commandRoutes(h *httpRouter) {
	h.router.PUT("/api/v1/bookings", h.updateBooking)
	h.router.DELETE("/api/v1/bookings/:id", h.deleteBooking)
	h.router.POST("/api/v1/bookings/:id/end", h.endBooking)
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
			log.Fatal().Err(err).Msg("failed to start server")
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

// GetBooking godoc
// @Summary Show a booking
// @Description  get booking by booking reference
// @Tags         bookings
// @Accept       json
// @Produce json
// @Param id path uuid.UUID true "Booking reference"
// @Success 200  {object}  domain.BookingResponse
// @Failure 400  {object}  HTTPError
// @Failure 404  {object}  HTTPError
// @Failure 500  {object}  HTTPError
// @Router       /bookings/{id} [get]
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

// GetAllBookings godoc
// @Summary List all bookings
// @Description  get all booking
// @Tags         bookings
// @Accept       json
// @Produce json
// @Success 200  {array}   domain.BookingResponse
// @Failure 400  {object}  HTTPError
// @Failure 404  {object}  HTTPError
// @Failure 500  {object}  HTTPError
// @Router       /bookings [get]
func (h *httpRouter) getAllBookings(c *gin.Context) {
	bookings, err := h.queries.GetAllBooking(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get all bookings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ooops! Something went wrong"})
		return
	}
	c.JSON(http.StatusOK, bookings)

}

// updateBooking godoc
// @Description  Update or add a new booking
// @Tags         bookings
// @Accept       json
// @Produce json
// @BParam   body   domain.UpdateBookingRequest  true  "Booking to update"
// @Success 202
// @Failure 400  {object}  HTTPError
// @Failure 404  {object}  HTTPError
// @Failure 500  {object}  HTTPError
// @Router       /bookings [post]
func (h *httpRouter) updateBooking(c *gin.Context) {
	request := domain.UpdateBookingRequest{}
	err := c.ShouldBindBodyWithJSON(&request)
	if err != nil {
		NewError(c, http.StatusBadRequest, fmt.Errorf("failed to bind body"))
		return
	}
	err = h.commands.UpdateBooking(c.Request.Context(), request)
	if err != nil {
		NewError(c, http.StatusBadRequest, fmt.Errorf("failed to update booking"))
		return
	}
	c.Status(http.StatusAccepted)
}

// deleteooking godoc
// @Summary Delete a booking
// @Description  delete booking by booking reference
// @Tags         bookings
// @Accept       json
// @Produce json
// @Param id path uuid.UUID true "Booking reference"
// @Success 202
// @Failure 400  {object}  HTTPError
// @Failure 404  {object}  HTTPError
// @Failure 500  {object}  HTTPError
// @Router       /bookings/{id} [delete]

func (h *httpRouter) deleteBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}

	err = h.commands.DeleteBooking(c.Request.Context(), domain.BookingRequest{BookingReference: id})
	if err != nil {
		NewError(c, http.StatusBadRequest, fmt.Errorf("failed to delete booking"))
		return
	}
	c.Status(http.StatusAccepted)
}

func (h httpRouter) endBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}
	distance := domain.Distance{}
	err = c.ShouldBindBodyWithJSON(&distance)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	err = h.commands.EndBooking(c.Request.Context(), domain.EndBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: id},
		Distance:       distance,
	})
	if err != nil {
		NewError(c, http.StatusBadRequest, fmt.Errorf("failed to end booking"))
		return
	}
	c.Status(http.StatusAccepted)
}
