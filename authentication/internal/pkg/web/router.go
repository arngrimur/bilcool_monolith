package web

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
)

type HttpRouter struct {
	router    *gin.Engine
	commands  application.Commands
	queries   application.Queries
	jwtSecret []byte
}

func NewRouter(commands application.Commands, queries application.Queries, jwtSecret string) *HttpRouter {
	engine := gin.Default()
	h := &HttpRouter{
		commands:  commands,
		queries:   queries,
		router:    engine,
		jwtSecret: []byte(jwtSecret),
	}
	h.router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})
	h.router.POST("/api/v1/users", h.createUser)
	h.router.GET("/api/v1/users", h.listUsers)
	h.router.DELETE("/api/v1/users/:id", h.deleteUser)
	h.router.GET("/api/v1/users/:id", h.getUser)
	h.router.POST("/api/v1/users/login", h.loginBegin)
	h.router.POST("/api/v1/users/login/token", h.verifyToken)
	h.router.POST("/api/v1/users/login/complete", h.loginComplete)

	admin := h.router.Group("/api/v1", h.jwtMiddleware(), h.requireAdmin())
	admin.PATCH("/users/:id/role", h.changeUserRole)

	return h
}

func (h *HttpRouter) jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			NewError(c, http.StatusUnauthorized, "missing or invalid authorization header")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwtlib.Parse(tokenStr, func(t *jwtlib.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
				return nil, domain.ErrInvalidCredential
			}
			return h.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			NewError(c, http.StatusUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwtlib.MapClaims)
		if !ok {
			NewError(c, http.StatusUnauthorized, "invalid token claims")
			c.Abort()
			return
		}
		userRef, _ := claims["user_ref"].(string)
		role, _ := claims["role"].(string)
		c.Set("caller_ref", userRef)
		c.Set("caller_role", role)
		c.Next()
	}
}

func (h *HttpRouter) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if role, _ := c.Get("caller_role"); role != "admin" {
			NewError(c, http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
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

func (h *HttpRouter) createUser(c *gin.Context) {
	var req domain.CreateUserRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		NewError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.commands.CreateUser(c.Request.Context(), req)
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *HttpRouter) deleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		NewError(c, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.commands.DeleteUser(c.Request.Context(), id); err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *HttpRouter) listUsers(c *gin.Context) {
	resp, err := h.queries.ListUsers(c.Request.Context())
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HttpRouter) getUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		NewError(c, http.StatusBadRequest, "invalid user id")
		return
	}
	resp, err := h.queries.GetUserByRef(c.Request.Context(), id)
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HttpRouter) changeUserRole(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		NewError(c, http.StatusBadRequest, "invalid user id")
		return
	}
	callerRefStr, _ := c.Get("caller_ref")
	callerRef, err := uuid.Parse(callerRefStr.(string))
	if err != nil {
		NewError(c, http.StatusUnauthorized, "invalid caller identity")
		return
	}
	var req domain.ChangeUserRoleRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		NewError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		NewError(c, http.StatusBadRequest, "role must be admin or user")
		return
	}
	if err := h.commands.ChangeUserRole(c.Request.Context(), callerRef, targetID, req.Role); err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *HttpRouter) loginBegin(c *gin.Context) {
	var req domain.LoginBeginRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		NewError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.commands.LoginBegin(c.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("login begin failed")
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HttpRouter) verifyToken(c *gin.Context) {
	var req domain.VerifyTokenRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		log.Error().Err(err).Msg("invalid request body")
		NewError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.commands.VerifyToken(c.Request.Context(), req)
	if err != nil {
		log.Error().Err(err).Msg("verify token failed")
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *HttpRouter) loginComplete(c *gin.Context) {
	var req domain.LoginCompleteRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		NewError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.commands.LoginComplete(c.Request.Context(), req)
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}
	c.JSON(http.StatusOK, resp)
}
