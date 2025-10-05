package processing

import (
	"context"
	"encoding/json"
	"fmt"

	//	"io"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/jjckrbbt/chimera/backend/internal/config"
	"github.com/jjckrbbt/chimera/backend/internal/ingestion"
	"github.com/jjckrbbt/chimera/backend/internal/interfaces"
	"github.com/jjckrbbt/chimera/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool" // Import the pgxpool package
)

// Service orchestrates the processing of an ingestion job.
type Service struct {
	ingestionService *ingestion.Service
	configLoader     *ConfigLoader
	queries          *repository.Queries
	gcsClient        *storage.Client
	gcsBucket        string
	logger           *slog.Logger
	cfg              *config.Config
	// CORRECTED: Use a connection pool
	dbpool *pgxpool.Pool
}

// NewService creates and initializes a new processing service.
func NewService(
	ingestionService *ingestion.Service,
	configLoader *ConfigLoader,
	queries *repository.Queries,
	gcsClient *storage.Client,
	logger *slog.Logger,
	cfg *config.Config,
	dbpool *pgxpool.Pool, // CORRECTED: Expect a pool
) *Service {
	return &Service{
		ingestionService: ingestionService,
		configLoader:     configLoader,
		queries:          queries,
		gcsClient:        gcsClient,
		gcsBucket:        cfg.GCSBucketName,
		logger:           logger,
		cfg:              cfg,
		dbpool:           dbpool,
	}
}

// RunJob is the main entry point for processing a file. It's designed to be run in a goroutine.
func (s *Service) RunJob(ctx context.Context, jobID uuid.UUID, reportType, gcsURI string, embedder interfaces.EmbedderFunc) {
	// ... (The beginning of this function is unchanged)
	jobCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	procLogger := s.logger.With("job_id", jobID.String(), "report_type", reportType)
	procLogger.InfoContext(jobCtx, "Starting asynchronous processing job")

	err := s.ingestionService.UpdateJobStatus(jobCtx, jobID, "PROCESSING", "", 0, 0)
	if err != nil {
		procLogger.ErrorContext(jobCtx, "Failed to update job status to PROCESSING, aborting", "error", err)
		return
	}

	storageKey := strings.TrimPrefix(gcsURI, fmt.Sprintf("gs://%s/", s.gcsBucket))
	reader, err := s.gcsClient.Bucket(s.gcsBucket).Object(storageKey).NewReader(jobCtx)
	if err != nil {
		procLogger.ErrorContext(jobCtx, "Failed to create GCS reader for file", "storage_key", storageKey, "error", err)
		_ = s.ingestionService.UpdateJobStatus(jobCtx, jobID, "FAILED", fmt.Sprintf("Failed to read file from storage: %v", err), 0, 0)
		return
	}
	defer reader.Close()

	ingestionConfig, found := s.configLoader.GetConfig(reportType)
	if !found {
		errorMsg := fmt.Sprintf("No processor configuration found for report type: %s", reportType)
		procLogger.ErrorContext(jobCtx, errorMsg)
		_ = s.ingestionService.UpdateJobStatus(jobCtx, jobID, "FAILED", errorMsg, 0, 0)
		return
	}

	processor := NewGenericProcessor(ingestionConfig)
	result, err := processor.Process(jobCtx, reader, s.queries, embedder)

	if result != nil && len(result.TriageRows) > 0 {
		s.logTriageItems(jobCtx, jobID, result.TriageRows)
	}

	if err != nil {
		errorMsg := err.Error()
		rowsTriaged := int64(0)
		if result != nil {
			rowsTriaged = int64(len(result.TriageRows))
		}
		procLogger.ErrorContext(jobCtx, "Processing job finished with critical error", "error", err)
		_ = s.ingestionService.UpdateJobStatus(jobCtx, jobID, "FAILED", errorMsg, 0, rowsTriaged)
		return
	}

	var rowsUpserted int64 = 0
	if result != nil && len(result.SuccessfulItems) > 0 {
		upsertedCount, err := s.saveSuccessfulItems(jobCtx, result.SuccessfulItems)
		if err != nil {
			procLogger.ErrorContext(jobCtx, "Failed to save successful items to database", "error", err)
			_ = s.ingestionService.UpdateJobStatus(jobCtx, jobID, "FAILED", "Error saving processed data to database", 0, int64(len(result.TriageRows)))
			return
		}
		rowsUpserted = upsertedCount
	}

	rowsTriaged := int64(len(result.TriageRows))
	finalStatus := "COMPLETE"
	finalMessage := fmt.Sprintf("Processed %d items successfully. %d rows sent for triage. %d blank rows discarded.", rowsUpserted, rowsTriaged, result.BlankRowsDiscarded)
	if rowsTriaged > 0 {
		finalStatus = "COMPLETE_WITH_ISSUES"
	}
	procLogger.InfoContext(jobCtx, "Processing job completed", "status", finalStatus, "rows_upserted", rowsUpserted, "rows_for_triage", rowsTriaged)
	_ = s.ingestionService.UpdateJobStatus(jobCtx, jobID, finalStatus, finalMessage, rowsUpserted, rowsTriaged)
}

func (s *Service) saveSuccessfulItems(ctx context.Context, items []repository.Item) (int64, error) {
	// Start a new database transaction. This is crucial for data integrity.
	tx, err := s.dbpool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer a rollback. If we commit successfully, this does nothing. If we error out, it cleans up the mess.
	defer tx.Rollback(ctx)

	// Create a new querier that is bound to our transaction.
	qtx := s.queries.WithTx(tx)

	// --- Step 1: Create the temp table using our new sqlc function ---
	if err := qtx.CreateTempItemsStagingTable(ctx); err != nil {
		return 0, fmt.Errorf("failed to create temp staging table: %w", err)
	}

	// --- Step 2: Use pgx.CopyFrom to bulk-insert data into the temp table ---

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"temp_items_staging"},
		[]string{"item_type", "scope", "business_key", "status", "custom_properties", "embedding"},
		pgx.CopyFromSlice(len(items), func(i int) ([]interface{}, error) {
			var embeddingValue interface{}

			// Assuming items[i].Embedding is a type with a Slice() method returning []float32
			embeddingSlice := items[i].Embedding.Slice()

			// Check if the slice is not nil AND actually contains data
			if len(embeddingSlice) > 0 {

				// DEFENSIVE CHECK: Add a hard limit to prevent the DB error.
				// The root cause is likely upstream data corruption, but this protects the database.
				const maxEmbeddingDims = 384
				if len(embeddingSlice) > maxEmbeddingDims {
					s.logger.WarnContext(ctx, "Embedding exceeds maximum allowed dimensions, nullifying", "business_key", items[i].BusinessKey, "dims", len(embeddingSlice))
					// This should not happen. If it does, it's a sign of the upstream bug.
					embeddingValue = nil
				} else {
					// The embedding looks valid, use it.
					embeddingValue = items[i].Embedding
				}
			} else {
				// The embedding is nil or empty, so we'll insert NULL.
				embeddingValue = nil
			}

			return []interface{}{
				items[i].ItemType,
				items[i].Scope,
				items[i].BusinessKey,
				items[i].Status,
				items[i].CustomProperties,
				embeddingValue,
			}, nil
		}),
	)

	if err != nil {
		return 0, fmt.Errorf("failed to copy data to staging table: %w", err)
	}

	// --- Step 3: Upsert from the staging table using our existing sqlc function ---
	rowsAffected, err := qtx.UpsertItems(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert items from staging table: %w", err)
	}

	// --- Step 4: If all steps succeeded, commit the transaction ---
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return rowsAffected, nil
}

func (s *Service) logTriageItems(ctx context.Context, jobID uuid.UUID, triageRows []TriageRow) {
	procLogger := s.logger.With("job_id", jobID.String())
	procLogger.Info("Logging triage items to database", "count", len(triageRows))

	pgJobID := pgtype.UUID{Bytes: jobID, Valid: true}

	for _, row := range triageRows {
		rowDataJSON, err := json.Marshal(row.OriginalRecord)
		if err != nil {
			procLogger.Error("Failed to marshal original row data for triage", "error", err, "row", row.OriginalRecord)
			continue
		}

		params := repository.CreateIngestionErrorParams{
			ID:               pgtype.UUID{Bytes: uuid.New(), Valid: true},
			JobID:            pgJobID,
			OriginalRowData:  rowDataJSON,
			ReasonForFailure: row.FailureReason,
		}

		_, err = s.queries.CreateIngestionError(ctx, params)
		if err != nil {
			procLogger.Error("Failed to insert ingestion error record into database", "error", err)
		}
	}
}
