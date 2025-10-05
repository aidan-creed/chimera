package api

import (
	"log/slog"
	"net/http"

	"github.com/jjckrbbt/chimera/backend/internal/repository"
	"github.com/labstack/echo/v4"
)

type DashboardHandler struct {
	queries repository.Querier
	logger  *slog.Logger
}

func NewDashboardHandler(q repository.Querier, logger *slog.Logger) *DashboardHandler {
	return &DashboardHandler{
		queries: q,
		logger:  logger.With("component", "dashboard_handler"),
	}
}

func (h *DashboardHandler) HandleGetDashboardStats(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "Not Implemented")
}
