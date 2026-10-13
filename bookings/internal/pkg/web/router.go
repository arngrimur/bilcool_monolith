package web

import (
	"context"
	"errors"
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

	"github.com/arngrimur/bilcool-lib/pkg/middleware"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	_ "github.com/arngrimur/bilcool_monolith/docs"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

// @title BilCool REST API
// @version 1.0
// @description The REST API for BilCool to book and view bookings and other stuff.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.basic None

// HttpRouter
// @externalDocs.description OpenAPI
// @externalDocs.url https://swagger.io/resources/open-api/
type HttpRouter struct {
	router   *gin.Engine
	queries  application.Queries
	commands application.Commands
}

func NewRouter(q application.Queries, c application.Commands) *HttpRouter {
	gin.SetMode(config.GinMode())
	engine := gin.Default()
	engine.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	engine.Use(middleware.LogWithCorrelationID())
	h := &HttpRouter{
		queries:  q,
		commands: c,
		router:   engine,
	}

	internalRoutes(h)
	queryRoutes(h)
	commandRoutes(h)
	return h
}

func internalRoutes(h *HttpRouter) {
	h.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func queryRoutes(h *HttpRouter) {
	h.router.GET("/api/v1/bookings", h.getAllBookings)
	h.router.GET("/api/v1/bookings/:id", h.getBooking)
	h.router.GET("/api/v1/bookings/:id/track", h.getTrackPoints)
	h.router.GET("/api/v1/bookings/:id/route", h.getRoute)
}

func commandRoutes(h *HttpRouter) {
	h.router.PUT("/api/v1/bookings", h.updateBooking)
	h.router.DELETE("/api/v1/bookings/:id", h.deleteBooking)
	h.router.POST("/api/v1/bookings/:id/end", h.endBooking)
	h.router.POST("/api/v1/bookings/:id/pause", h.pauseBooking)
	h.router.POST("/api/v1/bookings/:id/resume", h.resumeBooking)
	h.router.POST("/api/v1/bookings/:id/track", h.addTrackPoints)
}

func (h *HttpRouter) Engine() *gin.Engine { return h.router }

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

	// Wait for the interrupt signal for graceful shutdown
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
// @Description get booking by booking reference
// @Tags bookings
// @Accept json
// @Produce json
// @Param id path uuid.UUID true "Booking reference"
// @Success 200 {object} domain.BookingResponse
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /bookings/{id} [get]
func (h *HttpRouter) getBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}

	booking, err := h.queries.GetBooking(c.Request.Context(), domain.BookingRequest{BookingReference: id})
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to find booking %s", e.Message))
		return
	}
	c.JSON(http.StatusOK, booking)
}

// GetTrackPoints godoc
// @Summary Show the route of a booking
// @Description get the recorded GPS track points of a booking, ordered by recording time
// @Tags bookings
// @Produce json
// @Param id path uuid.UUID true "Booking reference"
// @Success 200 {array} domain.TrackPoint
// @Failure 400 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /bookings/{id}/track [get]
func (h *HttpRouter) getTrackPoints(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}

	points, err := h.queries.GetTrackPoints(c.Request.Context(), domain.BookingRequest{BookingReference: id})
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to get track points %s", e.Message))
		return
	}
	c.JSON(http.StatusOK, points)
}

// GetRoute godoc
// @Summary Show the route of a booking snapped to the road network
// @Description get the recorded GPS track points of a booking and the same track snapped to roads. The route is empty when snapping is not possible, in which case the points should be drawn.
// @Tags bookings
// @Produce json
// @Param id path uuid.UUID true "Booking reference"
// @Success 200 {object} domain.RouteResponse
// @Failure 400 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /bookings/{id}/route [get]
func (h *HttpRouter) getRoute(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}

	route, err := h.queries.GetRoute(c.Request.Context(), domain.BookingRequest{BookingReference: id})
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to get route %s", e.Message))
		return
	}
	// The recorded track of a finished booking does not change, so the snapped route (which
	// costs a Mapbox request) may be reused by the browser for a few minutes.
	c.Header("Cache-Control", "private, max-age=300")
	c.JSON(http.StatusOK, route)
}

// GetAllBookings godoc
// @Summary List all bookings
// @Description get all booking
// @Tags bookings
// @Accept json
// @Produce json
// @Success 200 {array} domain.BookingResponse
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /bookings [get]
func (h *HttpRouter) getAllBookings(c *gin.Context) {
	bookings, err := h.queries.GetAllBooking(c.Request.Context())
	if err != nil {
		log.Ctx(c.Request.Context()).Error().Err(err).Msg("failed to get all bookings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ooops! Something went wrong"})
		return
	}
	c.JSON(http.StatusOK, bookings)

}

// updateBooking godoc
// @Description Update or add a new booking
// @Tags bookings
// @Accept json
// @Produce json
// @BParam body domain.UpdateBookingRequest true "Booking to update"
// @Success 202
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /bookings [post]
func (h *HttpRouter) updateBooking(c *gin.Context) {
	request := domain.UpdateBookingRequest{}
	err := c.ShouldBindBodyWithJSON(&request)
	if err != nil {
		NewError(c, http.StatusBadRequest, fmt.Errorf("failed to bind body"))
		return
	}
	err = h.commands.UpdateBooking(c.Request.Context(), request)
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to update booking %s", e.Message))
		return
	}
	c.Status(http.StatusAccepted)
}

// deleteooking godoc
// @Summary Delete a booking
// @Description delete booking by booking reference
// @Tags bookings
// @Accept json
// @Produce json
// @Param id path uuid.UUID true "Booking reference"
// @Success 202
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /bookings/{id} [delete]

func (h *HttpRouter) deleteBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}

	err = h.commands.DeleteBooking(c.Request.Context(), domain.BookingRequest{BookingReference: id})
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to delete booking %s", e.Message))
		return
	}
	c.Status(http.StatusAccepted)
}

// pauseBooking godoc
// @Summary Pause a booking
// @Description Pause an active booking. Requires speed below 7 km/h. Saves the current GPS position.
// @Tags bookings
// @Accept json
// @Produce json
// @Param id path string true "Booking reference"
// @Param body body extdomain.Position true "Current GPS position"
// @Success 202
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 422 {object} HTTPError
// @Router /bookings/{id}/pause [post]
func (h *HttpRouter) pauseBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}
	var pos extdomain.Position
	if err = c.ShouldBindBodyWithJSON(&pos); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	err = h.commands.PauseBooking(c.Request.Context(), domain.PauseBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: id},
		Position:       pos,
	})
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to pause booking %s", e.Message))
		return
	}
	c.Status(http.StatusAccepted)
}

// resumeBooking godoc
// @Summary Resume a paused booking
// @Description Resume a previously paused booking. Returns the saved pause position to resume GPS tracking from.
// @Tags bookings
// @Produce json
// @Param id path string true "Booking reference"
// @Success 200 {object} domain.PauseBookingResponse
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 422 {object} HTTPError
// @Router /bookings/{id}/resume [post]
func (h *HttpRouter) resumeBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}
	resp, err := h.commands.ResumeBooking(c.Request.Context(), domain.BookingRequest{BookingReference: id})
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to resume booking %s", e.Message))
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HttpRouter) endBooking(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}
	var body struct {
		OdometerStart *int                `json:"odometer_start,omitempty"`
		OdometerEnd   *int                `json:"odometer_end,omitempty"`
		Position      *extdomain.Position `json:"position,omitempty"`
	}
	err = c.ShouldBindBodyWithJSON(&body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	err = h.commands.EndBooking(c.Request.Context(), domain.EndBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: id},
		OdometerStart:  body.OdometerStart,
		OdometerEnd:    body.OdometerEnd,
		Position:       body.Position,
	})
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to end booking %s", e.Message))
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *HttpRouter) addTrackPoints(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format or missing id"})
		return
	}
	var body struct {
		Points []extdomain.TrackPoint `json:"points"`
	}
	if err = c.ShouldBindBodyWithJSON(&body); err != nil || len(body.Points) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	err = h.commands.AddTrackPoints(c.Request.Context(), domain.AddTrackPointsRequest{
		BookingRequest: domain.BookingRequest{BookingReference: id},
		Points:         body.Points,
	})
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, fmt.Errorf("failed to add track points %s", e.Message))
		return
	}
	c.Status(http.StatusAccepted)
}
