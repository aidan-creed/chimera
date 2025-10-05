package processing

import "fmt"

// ValidationRule defines the validation rules for a single column
// yaml tags tell our parser how to map the YAML fields to our struct
type ValidationRule struct {
	Required      bool     `yaml:"required"`
	AllowZero     *bool    `yaml:"allow_zero,omitempty"`
	Enum          []string `yaml:"enum"`
	Regex         string   `yaml:"regex"`
	ExistsInItems string   `yaml:"exists_in_items,omitempty"`
}

// ProcessingAttempt defines an attempt to process an item
type ProcessingAttempt struct {
	Transforms []string `yaml:"transforms,omitempty"`
}

// ColumnMapping defines how to map and transform a single CSV column
type ColumnMapping struct {
	CSVHeader         string              `yaml:"csv_header"`
	JSONField         string              `yaml:"json_field"`
	MergeExcessFields bool                `yaml:"merge_excess_fields,omitempty"`
	Attempts          []ProcessingAttempt `yaml:"attempts"`
	Validation        ValidationRule      `yaml:"validation"`
}

// EmbedContent defines the configuration for generating embeddings during ingestion
type EmbedContent struct {
	SourceColumns []string `yaml:"source_columns"`
}

// IngestionConfig is the top-level struct that represents a full ingestion configuration fields
type IngestionConfig struct {
	ReportType     string          `yaml:"report_type"`
	ItemType       string          `yaml:"item_type"`
	ScopeField     string          `yaml:"scope_field"`
	BusinessKey    []string        `yaml:"business_key"`
	EmbedContent   *EmbedContent   `yaml:"embed_content,omitempty"`
	ColumnMappings []ColumnMapping `yaml:"column_mappings"`
}

// Validate checks if the IngestionConfig is valid
func (c *IngestionConfig) Validate() error {
	if c.ReportType == "" {
		return fmt.Errorf("config validation failed: report_type is required")
	}
	if c.ItemType == "" {
		return fmt.Errorf("config validation failed: item_type is required")
	}
	if c.ScopeField == "" {
		return fmt.Errorf("config validation failed: scope_field is required")
	}
	if len(c.BusinessKey) == 0 {
		return fmt.Errorf("config validation failed: business_key must contain at least one field")
	}
	if len(c.ColumnMappings) == 0 {
		return fmt.Errorf("config validation failed: have at least one column mapping")
	}

	// Create a quick lookup map of all defined CSV headers
	definedHeaders := make(map[string]bool)
	for _, mapping := range c.ColumnMappings {
		definedHeaders[mapping.CSVHeader] = true
	}

	// Check if the scopeFields value exists in the defined headers
	if _, exists := definedHeaders[c.ScopeField]; !exists {
		return fmt.Errorf("config validation failed: scope_field '%s' does not match any defined CSV headers", c.ScopeField)
	}
	return nil
}
