package ingestion

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"cloud.google.com/go/storage"
	//	"github.com/jackc/pgx/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	//	"github.com/jjckrbbt/chimera/backend/internal/logger"
	"github.com/jjckrbbt/chimera/backend/internal/config"
	"github.com/jjckrbbt/chimera/backend/internal/repository"
)

type Service struct {
	queries   repository.Querier
	gcsClient *storage.Client
	gcsBucket string
	logger    *slog.Logger
	cfg       *config.Config
}

func NewService(queries repository.Querier, gcsClient *storage.Client, cfg *config.Config, logger *slog.Logger) (*Service, error) {
	return &Service{
		queries:   queries,
		gcsClient: gcsClient,
		gcsBucket: cfg.GCSBucketName,
		logger:    logger.With("component", "ingestion_service"),
		cfg:       cfg,
	}, nil
}

func (s *Service) StartJob(ctx context.Context, file io.Reader, originalFilename, itemType string, userID int64) (*repository.IngestionJob, error) {
	jobID := uuid.New()
	gcsObjectKey := fmt.Sprintf("raw-reports/%s/%s-/%s", itemType, jobID.String(), originalFilename)

	s.logger.InfoContext(ctx, "Starting ingestion job", "job_id", jobID, "item_type", itemType, "user_id", userID)

	// --- Upload file to GCS ---
	wc := s.gcsClient.Bucket(s.gcsBucket).Object(gcsObjectKey).NewWriter(ctx)

	if _, err := io.Copy(wc, file); err != nil {
		s.logger.ErrorContext(ctx, "Failed to upload file to GCS", slog.Any("error", err))
		return nil, fmt.Errorf("failed to upload file to GCS: %w", err)
	}
	// Close the writer to finalize the upload
	if err := wc.Close(); err != nil {
		s.logger.ErrorContext(ctx, "Failed to close GCS writer", slog.Any("error", err))
		return nil, fmt.Errorf("failed to close GCS writer: %w", err)
	}
	s.logger.InfoContext(ctx, "File successfully uploaded to GCS", "job_id", jobID, "gcs_object_key", gcsObjectKey)

	// --- Create ingestion job record ---
	params := repository.CreateIngestionJobParams{
		ID:            pgtype.UUID{Bytes: jobID, Valid: true},
		SourceType:    "FILE_UPLOAD",
		ItemType:      itemType,
		Status:        "UPLOADED",
		UserID:        pgtype.Int8{Int64: userID, Valid: true},
		SourceDetails: []byte(fmt.Sprintf(`{"filename": "%s"}`, originalFilename)),
		SourceUri:     pgtype.Text{String: gcsObjectKey, Valid: true},
	}
	createdJob, err := s.queries.CreateIngestionJob(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to create ingestion job record", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create ingestion job record: %w", err)
	}
	s.logger.InfoContext(ctx, "Ingestion job record created", "job_id", jobID)

	return &createdJob, nil
}

// UpdateJobStatus updates the status of an ingestion job
func (s *Service) UpdateJobStatus(ctx context.Context, jobID uuid.UUID, status string, errorDetails string, rowsUpserted int64, rowsTriaged int64) error {
	params := repository.UpdateIngestionJobStatusParams{
		ID:                pgtype.UUID{Bytes: jobID, Valid: true},
		Status:            status,
		ErrorDetails:      pgtype.Text{String: errorDetails, Valid: errorDetails != ""},
		ProcessedRows:     pgtype.Int4{Int32: int32(rowsUpserted), Valid: rowsUpserted > 0},
		InitialErrorCount: pgtype.Int4{Int32: int32(rowsTriaged), Valid: rowsTriaged > 0},
	}

	err := s.queries.UpdateIngestionJobStatus(ctx, params)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update ingestion job status",
			"error", err,
			"job_id", jobID,
			"new_status", status,
		)
		return err
	}
	s.logger.InfoContext(ctx, "Ingestion job status updated", "job_id", jobID, "status", status)
	return nil
}
