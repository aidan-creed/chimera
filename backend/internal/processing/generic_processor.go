package processing

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jjckrbbt/chimera/backend/internal/interfaces"
	"github.com/jjckrbbt/chimera/backend/internal/repository"
	"github.com/pgvector/pgvector-go"
)

// Processor defines the standard interface for any processor.
type Processor interface {
	Process(
		ctx context.Context,
		file io.Reader,
		queries repository.Querier,
	) (*ProcessingResult, error)
}

// ProcessingResult holds the outcome of a file processing operation
type ProcessingResult struct {
	SuccessfulItems    []repository.Item
	TriageRows         []TriageRow
	BlankRowsDiscarded int
}

// TriageRow represents a row that failed processing and needs human review
type TriageRow struct {
	OriginalRecord map[string]string `json:"original_record"`
	FailureReason  string            `json:"failure_reason"`
}

// GenericProcessor uses an IngestionConfig to process a CSV file
type GenericProcessor struct {
	config IngestionConfig
}

// NewGenericProcessor creates a new processor with a specific configuration
func NewGenericProcessor(config IngestionConfig) *GenericProcessor {
	return &GenericProcessor{config: config}
}

// Process is the main entry point that executes the entire ingestion logic
func (p *GenericProcessor) Process(
	ctx context.Context,
	file io.Reader,
	queries repository.Querier,
	embedder interfaces.EmbedderFunc,
) (*ProcessingResult, error) {
	result := &ProcessingResult{}
	csvReader := csv.NewReader(file)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1 // prevents reader from crashing

	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading header row: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

	// ADD THIS BLOCK: Fail-fast on configuration errors.
	for _, mapping := range p.config.ColumnMappings {
		if _, ok := headerMap[mapping.CSVHeader]; !ok {
			return nil, fmt.Errorf("configuration error: CSV file is missing required header '%s'", mapping.CSVHeader)
		}
	}

	numHeaders := len(headers)

	mergeColumnIndex := -1
	for _, mapping := range p.config.ColumnMappings {
		if mapping.MergeExcessFields {
			if idx, ok := headerMap[mapping.CSVHeader]; ok {
				mergeColumnIndex = idx
				break // assume only one column can be merge target
			}
		}
	}

	allRecords, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read all CSV records: %w", err)
	}

	var scopeJSONField string
	for _, mapping := range p.config.ColumnMappings {
		if mapping.CSVHeader == p.config.ScopeField {
			scopeJSONField = mapping.JSONField
			break
		}
	}
	if scopeJSONField == "" {
		return nil, fmt.Errorf("config validation error: could not find a column mapping for the specified scope_field '%s'", p.config.ScopeField)
	}

RecordLoop:
	for i, record := range allRecords {
		if len(record) > numHeaders && mergeColumnIndex != -1 {
			numExtraFields := len(record) - numHeaders

			endOfMergeIndex := mergeColumnIndex + numExtraFields
			fieldsToMerge := record[mergeColumnIndex : endOfMergeIndex+1]
			rejoinedValue := strings.Join(fieldsToMerge, ",")

			correctedRecord := make([]string, 0, numHeaders)
			correctedRecord = append(correctedRecord, record[:mergeColumnIndex]...)
			correctedRecord = append(correctedRecord, rejoinedValue)
			correctedRecord = append(correctedRecord, record[endOfMergeIndex+1:]...)

			record = correctedRecord
		}

		if len(record) != numHeaders {
			result.TriageRows = append(result.TriageRows, TriageRow{
				OriginalRecord: createOriginalRecordMap(record, headers),
				FailureReason:  fmt.Sprintf("Row as %d fields, but header has %d. Triage required.", len(record), numHeaders),
			})
			continue RecordLoop // skip to next record
		}

		if isRowBlank(record) {
			result.BlankRowsDiscarded++
			continue
		}

		processedData, err := p.processRow(ctx, record, headerMap, queries)
		if err != nil {
			result.TriageRows = append(result.TriageRows, TriageRow{
				OriginalRecord: createOriginalRecordMap(record, headers),
				FailureReason:  err.Error(),
			})
			continue
		}

		var embedding pgvector.Vector
		if p.config.EmbedContent != nil && embedder != nil {

			var textToEmbedBuilder strings.Builder
			for _, colName := range p.config.EmbedContent.SourceColumns {
				if val, ok := processedData[colName]; ok {
					textToEmbedBuilder.WriteString(fmt.Sprintf("%v ", val))
				}
			}
			textToEmbed := strings.TrimSpace(textToEmbedBuilder.String())

			if textToEmbed != "" {
				slog.Debug("Generating embedding for text", "text", textToEmbed)
				embeddingVector, err := embedder(ctx, textToEmbed)
				if err != nil {
					triageRow := TriageRow{
						OriginalRecord: createOriginalRecordMap(record, headers),
						FailureReason:  fmt.Sprintf("Row %d: failed to generate embedding: %s", i+2, err.Error()),
					}
					result.TriageRows = append(result.TriageRows, triageRow)
					continue
				}
				embedding = pgvector.NewVector(embeddingVector)

			}
		}

		customPropsJSON, err := json.Marshal(processedData)
		if err != nil {
			result.TriageRows = append(result.TriageRows, TriageRow{
				OriginalRecord: createOriginalRecordMap(record, headers),
				FailureReason:  fmt.Sprintf("Row %d: failed to marshal processed data to JSON: %s", i+2, err.Error()),
			})
			continue
		}

		scopeVal, ok := processedData[scopeJSONField]
		if !ok || scopeVal == nil {
			result.TriageRows = append(result.TriageRows, TriageRow{
				OriginalRecord: createOriginalRecordMap(record, headers),
				FailureReason:  fmt.Sprintf("scope field '%s' is missing or nil", scopeJSONField),
			})
			continue
		}

		scopeString, ok := scopeVal.(string)
		if !ok {
			result.TriageRows = append(result.TriageRows, TriageRow{
				OriginalRecord: createOriginalRecordMap(record, headers),
				FailureReason:  fmt.Sprintf("scope field '%s' is not a string", scopeJSONField),
			})
			continue
		}

		// Build the business key, and if any part is missing, triage the row ONCE and move to the next record.
		var businessKeyParts []string
		for _, field := range p.config.BusinessKey {
			val, ok := processedData[field]
			if !ok || val == nil {
				result.TriageRows = append(result.TriageRows, TriageRow{
					OriginalRecord: createOriginalRecordMap(record, headers),
					FailureReason:  fmt.Sprintf("business key field '%s' is missing or nil", field),
				})
				continue RecordLoop // This is the key change to prevent multiple errors for one row
			}
			businessKeyParts = append(businessKeyParts, fmt.Sprintf("%v", val))
		}

		item := repository.Item{
			ItemType:         repository.ItemType(p.config.ItemType),
			Scope:            pgtype.Text{String: scopeString, Valid: true},
			BusinessKey:      pgtype.Text{String: strings.Join(businessKeyParts, "-"), Valid: true},
			Status:           "active",
			CustomProperties: customPropsJSON,
			Embedding:        embedding,
		}
		result.SuccessfulItems = append(result.SuccessfulItems, item)
	}

	slog.InfoContext(ctx, "Processing complete",
		"successful_items", len(result.SuccessfulItems),
		"triage_rows", len(result.TriageRows),
		"blank_rows_discarded", result.BlankRowsDiscarded,
	)
	return result, nil
}

// processRow handles the 'attempts' logic for a single, non-blank row.
func (p *GenericProcessor) processRow(ctx context.Context, record []string, headerMap map[string]int, queries repository.Querier) (map[string]interface{}, error) {
	processedData := make(map[string]interface{})

	for _, mapping := range p.config.ColumnMappings {
		// The check for header existence is now done in the main Process loop.
		// We can safely assume the key exists here.
		colIdx := headerMap[mapping.CSVHeader]

		var rawValue string
		if colIdx < len(record) {
			rawValue = record[colIdx]
		}

		var transformedValue interface{} = rawValue
		var transformError error
		var transformSuccessful bool = false

		if len(mapping.Attempts) > 0 {
			for _, attempt := range mapping.Attempts {
				val, err := applyTransforms(rawValue, attempt.Transforms)
				if err == nil {
					transformedValue = val
					transformSuccessful = true
					break
				}
				transformError = err
			}

			if !transformSuccessful {
				return nil, fmt.Errorf("all transform attempts failed for column '%s' with value '%s': %w", mapping.CSVHeader, rawValue, transformError)
			}
		} else {
			transformSuccessful = true
		}

		if err := applyValidation(ctx, queries, transformedValue, mapping.Validation); err != nil {
			return nil, fmt.Errorf("validation failed for column '%s' with value '%v': %w", mapping.CSVHeader, transformedValue, err)
		}

		// Add detailed logging to trace the final value and type for each field.
		slog.Debug("processRow: field processed",
			"csv_header", mapping.CSVHeader,
			"json_field", mapping.JSONField,
			"final_value", transformedValue,
			"final_type", fmt.Sprintf("%T", transformedValue))

		processedData[mapping.JSONField] = transformedValue
	}

	return processedData, nil
}

// --- Helper functions ---

func isRowBlank(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}

func createOriginalRecordMap(record []string, headers []string) map[string]string {
	rowMap := make(map[string]string)
	for i, header := range headers {
		if i < len(record) {
			rowMap[header] = record[i]
		} else {
			rowMap[header] = ""
		}
	}
	return rowMap
}

func applyTransforms(value string, transforms []string) (interface{}, error) {
	var currentValue interface{} = value
	for _, transformCall := range transforms {
		parts := strings.SplitN(transformCall, ":", 2)
		transformName := parts[0]
		var arg string
		if len(parts) > 1 {
			arg = parts[1]
		}
		transformer, ok := transformRegistry[transformName]
		if !ok {
			return nil, fmt.Errorf("unknown transform function: %s", transformName)
		}
		newValue, err := transformer(currentValue, arg)
		if err != nil {
			return nil, fmt.Errorf("transform '%s' failed: %w", transformName, err)
		}
		currentValue = newValue
	}
	return currentValue, nil
}

func applyValidation(ctx context.Context, queries repository.Querier, value interface{}, rules ValidationRule) error {
	if str, ok := value.(string); ok && str == "" && !rules.Required {
		return nil
	}
	for name, validationFunc := range validationRegistry {
		err := validationFunc(ctx, queries, value, rules)
		if err != nil {
			return fmt.Errorf("validation rule '%s' failed: %w", name, err)
		}
	}
	return nil
}
