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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/arngrimur/bilcool-lib/pkg/middleware"
	_ "github.com/arngrimur/bilcool_monolith/event_ledger/docs"
)

// @title Event Ledger API
// @version 1.0
// @description REST API for querying the event ledger.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

type HttpRouter struct {
	router  *gin.Engine
	querier EventQuerier
}

func NewRouter(querier EventQuerier) *HttpRouter {
	engine := gin.Default()
	engine.Use(middleware.LogWithCorrelationID())
	h := &HttpRouter{
		router:  engine,
		querier: querier,
	}

	h.router.GET("/health", h.health)
	h.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	h.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	h.router.GET("/api/v1/events", h.getEvents)

	return h
}

func (h *HttpRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *HttpRouter) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
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
