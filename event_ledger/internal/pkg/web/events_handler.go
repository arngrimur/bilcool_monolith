package web

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/domain"
)

type EventQuerier interface {
	QueryEvents(ctx context.Context, p domain.QueryParams) ([]domain.EventItem, error)
}

// GetEvents godoc
// @Summary List events
// @Description Query events from the ledger with optional filters
// @Tags events
// @Accept json
// @Produce json
// @Param event_id query string false "Filter by event ID"
// @Param producer query string false "Filter by producer"
// @Param event_type query string false "Filter by event type"
// @Param emitted_at query string false "Filter by exact emitted_at (RFC3339)"
// @Param emitted_at_gte query string false "Filter events emitted at or after this time (RFC3339)"
// @Param emitted_at_lte query string false "Filter events emitted at or before this time (RFC3339)"
// @Param limit query int false "Maximum number of results" default(50)
// @Param offset query int false "Number of results to skip" default(0)
// @Param order_by query string false "Field to order by" default(emitted_at)
// @Param order_direction query string false "Order direction: asc or desc" default(asc)
// @Success 200 {array} domain.EventResponse
// @Failure 400 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /api/v1/events [get]
func (h *HttpRouter) getEvents(c *gin.Context) {
	var p domain.QueryParams
	if err := c.ShouldBindQuery(&p); err != nil {
		NewError(c, http.StatusBadRequest, "invalid query parameters")
		return
	}

	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 50
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	events, err := h.querier.QueryEvents(c.Request.Context(), p)
	if err != nil {
		e := NewHttpError(err)
		NewError(c, e.Code, e.Message)
		return
	}

	resp := make([]domain.EventResponse, 0, len(events))
	for _, e := range events {
		resp = append(resp, e.ToResponse())
	}
	c.JSON(http.StatusOK, resp)
}
