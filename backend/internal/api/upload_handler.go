package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jjckrbbt/chimera/backend/internal/ingestion"
	"github.com/jjckrbbt/chimera/backend/internal/interfaces"
	"github.com/jjckrbbt/chimera/backend/internal/processing"
	"github.com/jjckrbbt/chimera/backend/internal/rag"
	"github.com/labstack/echo/v4"
)

// UploadHandler is responsible for orchestrating the file upload and processing.
type UploadHandler struct {
	ingestionService  *ingestion.Service
	processingService *processing.Service
	ragService        *rag.RAGService
	configLoader      *processing.ConfigLoader
	logger            *slog.Logger
}

// NewUploadHandler creates a new instance of the UploadHandler.
func NewUploadHandler(is *ingestion.Service, ps *processing.Service, ragSvc *rag.RAGService, cl *processing.ConfigLoader, logger *slog.Logger) *UploadHandler {
	return &UploadHandler{
		ingestionService:  is,
		processingService: ps,
		ragService:       ragSvc,
		configLoader:      cl,
		logger:            logger,
	}
}

// HandleUpload receives a file, starts an ingestion job, and triggers async processing.
func (h *UploadHandler) HandleUpload(c echo.Context) error {
	ctx := c.Request().Context()
	// NOTE: In a real app, you would get the user ID from the JWT in the context.
	// For this demo, we can hardcode it or leave it as 0.
	var userID int64 = 1
	reportType := c.Param("reportType")

	file, err := c.FormFile("report_file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "report_file is required")
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to open uploaded file")
	}
	defer src.Close()

	// 1. Start the ingestion job (uploads to GCS, creates DB record)
	job, err := h.ingestionService.StartJob(ctx, src, file.Filename, reportType, userID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to start ingestion job", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not start file processing.")
	}
	h.logger.InfoContext(ctx, "Successfully started ingestion job, queueing for processing", "job_id", job.ID)

	// 2. Determine which embedding function (if any) to use for this job
	var embedder interfaces.EmbedderFunc
	config, found := h.configLoader.GetConfig(reportType)
	if !found {
		h.logger.WarnContext(ctx, "No ingestion config found for reportType, processing will likely fail", "reportType", reportType)
	} else {
		if config.EmbedContent != nil {
			embedder = h.getEmbedding
		}
	}

	// 3. Trigger the processing service in a background goroutine
	go h.processingService.RunJob(
		context.Background(),
		uuid.UUID(job.ID.Bytes),
		reportType,
		job.SourceUri.String,
		embedder,
	)

	// 4. Return an immediate success response
	return c.JSON(http.StatusAccepted, job)
}

func (h *UploadHandler) getEmbedding(ctx context.Context, text string) ([]float32, error) {
	return h.ragService.GetEmbedding(ctx, text)
}