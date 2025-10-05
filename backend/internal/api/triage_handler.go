package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jjckrbbt/chimera/backend/internal/repository" // Use your project's import path
	"github.com/labstack/echo/v4"
)

type TriageHandler struct {
	db      *pgxpool.Pool
	queries *repository.Queries
	logger  *slog.Logger
}

// NewTriageHandler creates a new instance of the TriageHandler.
func NewTriageHandler(db *pgxpool.Pool, queries *repository.Queries, logger *slog.Logger) *TriageHandler {
	return &TriageHandler{
		db:      db,
		queries: queries,
		logger:  logger.With("component", "triage_handler"),
	}
}

func (h *TriageHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/ingestion-jobs", h.listIngestionJobs)
	g.GET("/ingestion-jobs/:jobId/errors", h.getIngestionErrors)
	g.PATCH("/ingestion-errors/:errorId", h.updateIngestionError)
}

func (h *TriageHandler) listIngestionJobs(c echo.Context) error {
	ctx := c.Request().Context()

	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	params := repository.ListIngestionJobsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	jobs, err := h.queries.ListIngestionJobs(ctx, params)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list ingestion jobs", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get ingestion jobs").SetInternal(err)
	}

	h.logger.InfoContext(ctx, "successfully retrieved ingestion jobs", "count", len(jobs), "limit", limit, "offset", offset)
	return c.JSON(http.StatusOK, jobs)
}

func (h *TriageHandler) getIngestionErrors(c echo.Context) error {
	ctx := c.Request().Context()
	jobIDStr := c.Param("jobId")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		h.logger.WarnContext(ctx, "invalid job ID format provided", "error", err, "job_id_param", jobIDStr)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job ID format")
	}

	pgJobID := pgtype.UUID{
		Bytes: jobID,
		Valid: true,
	}

	rows, err := h.queries.GetIngestionErrorsByJobID(ctx, pgJobID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get ingestion errors for job", "error", err, "job_id", jobID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get errored rows").SetInternal(err)
	}

	h.logger.InfoContext(ctx, "successfully retrieved ingestion errors", "job_id", jobID, "count", len(rows))
	return c.JSON(http.StatusOK, rows)
}

func (h *TriageHandler) updateIngestionError(c echo.Context) error {
	ctx := c.Request().Context()
	errorIDStr := c.Param("errorId")
	errorID, err := uuid.Parse(errorIDStr)
	if err != nil {
		h.logger.WarnContext(ctx, "invalid error ID format provided", "error", err, "error_id_param", errorIDStr)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid error ID format")
	}

	pgErrorID := pgtype.UUID{
		Bytes: errorID,
		Valid: true,
	}

	var requestBody map[string]json.RawMessage
	if err := c.Bind(&requestBody); err != nil {
		h.logger.WarnContext(ctx, "failed to bind request body for updating error", "error", err, "error_id", errorID)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body").SetInternal(err)
	}

	correctedData, ok := requestBody["corrected_data"]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "missing corrected_data in request body")
	}

	// In a real app, you would get this from the JWT token in the context.
	placeholderUserID := int64(1)
	pgResolvedBy := pgtype.Int8{
		Int64: placeholderUserID,
		Valid: true,
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		h.logger.ErrorContext(ctx, "could not start db transaction", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "could not start transaction").SetInternal(err)
	}
	defer tx.Rollback(ctx)

	qtx := h.queries.WithTx(tx)

	updateParams := repository.UpdateIngestionErrorWithCorrectionParams{
		ID:            pgErrorID,
		CorrectedData: correctedData,
		ResolvedBy:    pgResolvedBy,
	}

	updatedError, err := qtx.UpdateIngestionErrorWithCorrection(c.Request().Context(), updateParams)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to update ingestion error record", "error", err, "error_id", errorID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update ingestion error").SetInternal(err)
	}

	err = qtx.IncrementIngestionJobResolvedRows(c.Request().Context(), pgErrorID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to increment resolved rows count", "error", err, "error_id", errorID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update job counters").SetInternal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		h.logger.ErrorContext(ctx, "could not commit db transaction", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "could not commit transaction").SetInternal(err)
	}

	h.logger.InfoContext(ctx, "successfully triaged ingestion error", "error_id", errorID, "resolved_by", placeholderUserID)
	return c.JSON(http.StatusOK, updatedError)
}
